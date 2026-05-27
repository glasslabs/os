// Package api implements the HTTP management API for glass-agent.
package api

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/glasslabs/os/agent/proc"
	"github.com/hamba/logger/v2"
)

// Supervisor is the interface the Server uses to control the glass process.
type Supervisor interface {
	Restart()
	Info() proc.Info
	Lines() []string
	Follow(ctx context.Context) <-chan string
}

// Server serves the glass-agent HTTP management API.
type Server struct {
	addr       string
	supervisor Supervisor
	glassBin   string
	dataDir    string
	h          http.Handler
	log        *logger.Logger
}

// NewServer returns a new Server.
func NewServer(addr string, supervisor Supervisor, glassBin, dataDir string, log *logger.Logger) *Server {
	s := &Server{
		addr:       addr,
		supervisor: supervisor,
		glassBin:   glassBin,
		dataDir:    dataDir,
		log:        log,
	}

	s.h = s.routes()

	return s
}

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /status", s.handleStatus())
	mux.HandleFunc("GET /logs", s.handleLogs())
	mux.HandleFunc("POST /ota", s.handleOTA())
	mux.HandleFunc("POST /config", s.handleConfig())
	mux.HandleFunc("POST /secrets", s.handleSecrets())
	mux.HandleFunc("POST /assets/{name}", s.handleUploadAsset())
	mux.HandleFunc("DELETE /assets/{name}", s.handleDeleteAsset())
	mux.HandleFunc("POST /os-update", s.handleOSUpdate())
	mux.HandleFunc("GET /os-status", s.handleOSStatus())
	mux.HandleFunc("POST /reboot", s.handleReboot())

	return mux
}

// ServeHTTP serves an HTTP request.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.h.ServeHTTP(w, r)
}

func download(ctx context.Context, w io.Writer, h io.Writer, url string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetching url: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	_, err = io.Copy(io.MultiWriter(w, h), resp.Body)
	return err
}
