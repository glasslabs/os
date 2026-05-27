// Package proc manages the supervised glass child process.
package proc

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/glasslabs/os/agent/internal/exec"
	"github.com/hamba/logger/v2"
	lctx "github.com/hamba/logger/v2/ctx"
)

const ringSize = 2000

// Info holds a snapshot of supervisor state.
type Info struct {
	PID      int
	Restarts int32
	Started  time.Time
}

// Supervisor starts and supervises a process, capturing its output.
type Supervisor struct {
	exe *exec.Executable

	mu       sync.Mutex
	cancelFn func()
	started  time.Time
	restarts atomic.Int32

	ring *ringBuffer

	log *logger.Logger
}

// New returns a new Supervisor.
func New(exe *exec.Executable, log *logger.Logger) *Supervisor {
	s := &Supervisor{
		exe:  exe,
		ring: newRingBuffer(ringSize),
		log:  log,
	}
	return s
}

// Start begins the supervision loop in a background goroutine.
func (s *Supervisor) Start(ctx context.Context) error {
	go s.loop(ctx)
	return nil
}

// Restart sends SIGTERM to the running process. The supervision loop restarts it.
func (s *Supervisor) Restart() {
	s.mu.Lock()
	cancel := s.cancelFn
	s.mu.Unlock()

	if cancel != nil {
		cancel()
	}
}

// Info returns a snapshot of current supervisor state.
func (s *Supervisor) Info() Info {
	s.mu.Lock()
	pid := s.exe.PID()
	started := s.started
	s.mu.Unlock()

	return Info{
		PID:      pid,
		Restarts: s.restarts.Load(),
		Started:  started,
	}
}

// Lines returns the current ring buffer contents in chronological order.
func (s *Supervisor) Lines() []string {
	return s.ring.lines()
}

// Follow returns a channel that receives new log lines until ctx is cancelled.
func (s *Supervisor) Follow(ctx context.Context) <-chan string {
	return s.ring.follow(ctx)
}

func (s *Supervisor) loop(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			return
		}

		s.log.Info("Starting glass")

		if err := s.run(ctx); err != nil && ctx.Err() == nil {
			s.log.Error("Glass exited with error", lctx.Err(err))
		} else {
			s.log.Info("Glass exited")
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(2 * time.Second):
			s.restarts.Add(1)
		}
	}
}

func (s *Supervisor) run(ctx context.Context) error {
	pr, pw, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("creating output pipe: %w", err)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errCh, err := s.exe.Run(ctx, pw, pw)
	if err != nil {
		return fmt.Errorf("starting process: %w", err)
	}

	s.mu.Lock()
	s.cancelFn = cancel
	s.started = time.Now()
	s.mu.Unlock()

	doneCh := make(chan struct{}, 0)
	go func() {
		defer close(doneCh)

		scanner := bufio.NewScanner(pr)
		for scanner.Scan() {
			s.ring.write(scanner.Text())
		}
	}()

	err = <-errCh

	s.mu.Lock()
	s.cancelFn = nil
	s.mu.Unlock()

	_ = pw.Close()
	_ = pr.Close()

	<-doneCh

	return err
}

type ringBuffer struct {
	mu    sync.Mutex
	buf   []string
	pos   int // next write position
	total int // total lines ever written
	cond  *sync.Cond
}

func newRingBuffer(size int) *ringBuffer {
	rb := &ringBuffer{buf: make([]string, size)}
	rb.cond = sync.NewCond(&rb.mu)
	return rb
}

func (rb *ringBuffer) write(line string) {
	rb.mu.Lock()
	rb.buf[rb.pos] = line
	rb.pos = (rb.pos + 1) % len(rb.buf)
	rb.total++
	rb.cond.Broadcast()
	rb.mu.Unlock()
}

// lines returns all buffered lines in chronological order.
func (rb *ringBuffer) lines() []string {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	n := len(rb.buf)
	count := rb.total
	if count > n {
		count = n
	}
	if count == 0 {
		return nil
	}

	result := make([]string, count)
	start := (rb.pos - count + n) % n
	for i := range count {
		result[i] = rb.buf[(start+i)%n]
	}
	return result
}

// linesSince returns lines written after the given total count.
// Called with rb.mu held.
func (rb *ringBuffer) linesSince(known int) []string {
	available := rb.total - known
	if available <= 0 {
		return nil
	}
	n := len(rb.buf)
	if available > n {
		available = n
	}
	result := make([]string, available)
	start := (rb.pos - available + n) % n
	for i := range available {
		result[i] = rb.buf[(start+i)%n]
	}
	return result
}

// follow returns a channel of new lines. It closes when ctx is cancelled.
func (rb *ringBuffer) follow(ctx context.Context) <-chan string {
	ch := make(chan string, 64)

	// Wake the blocked cond.Wait when ctx is cancelled.
	go func() {
		<-ctx.Done()
		rb.cond.Broadcast()
	}()

	go func() {
		defer close(ch)

		rb.mu.Lock()
		known := rb.total
		rb.mu.Unlock()

		for {
			rb.mu.Lock()
			for rb.total == known && ctx.Err() == nil {
				rb.cond.Wait()
			}
			if ctx.Err() != nil {
				rb.mu.Unlock()
				return
			}
			lines := rb.linesSince(known)
			known = rb.total
			rb.mu.Unlock()

			for _, line := range lines {
				select {
				case ch <- line:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return ch
}
