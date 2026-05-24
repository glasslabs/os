// Package handlers implements the HTTP management API handlers for glass-agent.
package handlers

import (
	"context"
	"time"

	"github.com/hamba/logger/v2"
)

// Supervisor is the interface the handlers use to control the glass process.
type Supervisor interface {
	Restart()
	Info() SupervisorInfo
	Lines() []string
	Follow(ctx context.Context) <-chan string
}

// SupervisorInfo holds a snapshot of supervisor state.
type SupervisorInfo struct {
	PID      int
	Restarts int32
	Started  time.Time
}

// Config holds dependencies shared across all handlers.
type Config struct {
	Supervisor Supervisor
	DataDir    string
	Log        *logger.Logger
}


