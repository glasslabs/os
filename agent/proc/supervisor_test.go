package proc_test

import (
	"context"
	"flag"
	"io"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/glasslabs/os/agent/internal/exec"
	"github.com/glasslabs/os/agent/internal/exectest"
	"github.com/glasslabs/os/agent/proc"
	"github.com/hamba/logger/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	flag.Parse()

	if os.Getenv("EXEC_TEST_PID") == "" {
		_ = os.Setenv("EXEC_TEST_PID", strconv.Itoa(os.Getpid()))
		os.Exit(m.Run())
	}

	os.Exit(exectest.Run(flag.Args(), exectest.DefaultCommands))
}

func TestSupervisor_Info(t *testing.T) {
	t.Parallel()

	exe := exectest.New(t, "wait", time.Minute.String())
	sup := newSupervisor(exe)

	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)

	require.NoError(t, sup.Start(ctx))

	require.Eventually(t, func() bool {
		return sup.Info().PID > 0
	}, 5*time.Second, 10*time.Millisecond)

	info := sup.Info()
	assert.Greater(t, info.PID, 0)
	assert.False(t, info.Started.IsZero())
	assert.Equal(t, int32(0), info.Restarts)
}

func TestSupervisor_Lines(t *testing.T) {
	t.Parallel()

	exe := exectest.New(t, "echo", "hello world")
	sup := newSupervisor(exe)

	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)

	require.NoError(t, sup.Start(ctx))

	require.Eventually(t, func() bool {
		lines := sup.Lines()
		if len(lines) == 0 {
			return false
		}
		return lines[0] == "hello world"
	}, 5*time.Second, 10*time.Millisecond)

	assert.Equal(t, []string{"hello world"}, sup.Lines())
}

func TestSupervisor_Follow(t *testing.T) {
	t.Parallel()

	exe := exectest.New(t, "echo", "streamed line")
	sup := newSupervisor(exe)

	require.NoError(t, sup.Start(t.Context()))

	ch := sup.Follow(t.Context())

	var got []string
	require.Eventually(t, func() bool {
		for {
			select {
			case line, ok := <-ch:
				if !ok {
					return false
				}
				got = append(got, line)
				if len(got) >= 1 {
					return true
				}
			default:
				return false
			}
		}
	}, 5*time.Second, 10*time.Millisecond)

	assert.Contains(t, got, "streamed line")
}

func TestSupervisor_Restart(t *testing.T) {
	t.Parallel()

	exe := exectest.New(t, "wait", time.Minute.String())
	sup := newSupervisor(exe)

	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)

	require.NoError(t, sup.Start(ctx))

	require.Eventually(t, func() bool {
		return sup.Info().PID > 0
	}, 5*time.Second, 10*time.Millisecond)

	firstPID := sup.Info().PID

	sup.Restart()

	require.Eventually(t, func() bool {
		info := sup.Info()
		return info.PID > 0 && info.PID != firstPID
	}, 10*time.Second, 10*time.Millisecond)
}

func TestSupervisor_RestartIncrementsCount(t *testing.T) {
	t.Parallel()

	exe := exectest.New(t, "exit", "0")
	sup := newSupervisor(exe)

	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)

	require.NoError(t, sup.Start(ctx))

	require.Eventually(t, func() bool {
		return sup.Info().Restarts > 0
	}, 10*time.Second, 10*time.Millisecond)
}

func TestSupervisor_StopsOnContextCancel(t *testing.T) {
	t.Parallel()

	exe := exectest.New(t, "wait", time.Minute.String())
	sup := newSupervisor(exe)

	ctx, cancel := context.WithCancel(t.Context())

	require.NoError(t, sup.Start(ctx))

	require.Eventually(t, func() bool {
		return sup.Info().PID > 0
	}, 5*time.Second, 10*time.Millisecond, "Process never started")

	cancel()

	require.Eventually(t, func() bool {
		return sup.Info().PID == 0
	}, 10*time.Second, 10*time.Millisecond)
}

func newSupervisor(exe *exec.Executable) *proc.Supervisor {
	log := logger.New(io.Discard, logger.LogfmtFormat(), logger.Info)

	return proc.New(exe, log)
}
