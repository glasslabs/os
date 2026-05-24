package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/glasslabs/os/agent/handlers"
	"github.com/hamba/logger/v2"
	lctx "github.com/hamba/logger/v2/ctx"
)

const ringSize = 2000

// supervisor starts and supervises the glass process, capturing its output.
type supervisor struct {
	glassBin string
	dataDir  string

	mu       sync.Mutex
	cmd      *exec.Cmd
	started  time.Time
	restarts atomic.Int32

	ring *ringBuffer

	stopCh chan struct{}

	log *logger.Logger
}

func newSupervisor(glassBin, dataDir string, log *logger.Logger) *supervisor {
	return &supervisor{
		glassBin: glassBin,
		dataDir:  dataDir,
		ring:     newRingBuffer(ringSize),
		stopCh:   make(chan struct{}),
		log:      log,
	}
}

// Start begins the supervision loop in a background goroutine.
func (s *supervisor) Start(ctx context.Context) error {
	go s.loop(ctx)
	return nil
}

// Restart sends SIGTERM to the running process. The supervision loop restarts it.
func (s *supervisor) Restart() {
	s.mu.Lock()
	cmd := s.cmd
	s.mu.Unlock()

	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Signal(syscall.SIGTERM)
	}
}

// Info returns a snapshot of current supervisor state.
func (s *supervisor) Info() handlers.SupervisorInfo {
	s.mu.Lock()
	pid := 0
	if s.cmd != nil && s.cmd.Process != nil {
		pid = s.cmd.Process.Pid
	}
	started := s.started
	s.mu.Unlock()

	return handlers.SupervisorInfo{
		PID:      pid,
		Restarts: s.restarts.Load(),
		Started:  started,
	}
}

// Lines returns the current ring buffer contents in chronological order.
func (s *supervisor) Lines() []string {
	return s.ring.lines()
}

// Follow returns a channel that receives new log lines until ctx is cancelled.
func (s *supervisor) Follow(ctx context.Context) <-chan string {
	return s.ring.follow(ctx)
}

func (s *supervisor) loop(ctx context.Context) {
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

func (s *supervisor) run(ctx context.Context) error {
	args := []string{
		"/usr/bin/cage", "--",
		s.glassBin, "run",
		"--config", filepath.Join(s.dataDir, "config", "config.yaml"),
		"--secrets", filepath.Join(s.dataDir, "config", "secrets.yaml"),
		"--assets", filepath.Join(s.dataDir, "assets"),
		"--modules", filepath.Join(s.dataDir, "modules"),
		"--log.format=logfmt",
	}

	cmd := exec.CommandContext(ctx, args[0], args[1:]...) //nolint:gosec // paths are from trusted config
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	pr, pw, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("creating output pipe: %w", err)
	}
	cmd.Stdout = pw
	cmd.Stderr = pw

	if err = cmd.Start(); err != nil {
		_ = pw.Close()
		_ = pr.Close()
		return fmt.Errorf("starting process: %w", err)
	}
	_ = pw.Close() // close write end in parent so reader gets EOF when process exits

	s.mu.Lock()
	s.cmd = cmd
	s.started = time.Now()
	s.mu.Unlock()

	scanner := bufio.NewScanner(pr)
	for scanner.Scan() {
		s.ring.write(scanner.Text())
	}

	err = cmd.Wait()

	s.mu.Lock()
	s.cmd = nil
	s.mu.Unlock()

	return err
}

// ringBuffer is a fixed-capacity circular log buffer safe for concurrent use.
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
