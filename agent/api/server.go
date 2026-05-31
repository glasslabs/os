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

// NetworkManager is the interface the Server uses to configure WiFi.
type NetworkManager interface {
	SetWiFi(ctx context.Context, ssid, password string) error
}

// Server serves the glass-agent HTTP management API.
type Server struct {
	addr       string
	supervisor Supervisor
	network    NetworkManager
	glassBin   string
	dataDir    string
	h          http.Handler
	log        *logger.Logger
}

// NewServer returns a new Server.
func NewServer(addr string, supervisor Supervisor, network NetworkManager, glassBin, dataDir string, log *logger.Logger) *Server {
	s := &Server{
		addr:       addr,
		supervisor: supervisor,
		network:    network,
		glassBin:   glassBin,
		dataDir:    dataDir,
		log:        log,
	}

	s.h = s.routes()

	return s
}

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /glass/status", s.handleStatus())
	mux.HandleFunc("GET /glass/logs", s.handleLogs())
	mux.HandleFunc("POST /glass/restart", s.handleRestart())
	mux.HandleFunc("POST /glass/update", s.handleUpdate())
	mux.HandleFunc("GET /glass/config", s.handleGetConfig())
	mux.HandleFunc("POST /glass/config", s.handleConfig())
	mux.HandleFunc("POST /glass/secrets", s.handleSecrets())
	mux.HandleFunc("GET /glass/assets", s.handleListAssets())
	mux.HandleFunc("GET /glass/assets/{name}", s.handleGetAsset())
	mux.HandleFunc("POST /glass/assets/{name}", s.handleUploadAsset())
	mux.HandleFunc("DELETE /glass/assets/{name}", s.handleDeleteAsset())
	mux.HandleFunc("POST /network/wifi", s.handleSetWifi())
	mux.HandleFunc("POST /os/update", s.handleOSUpdate())
	mux.HandleFunc("GET /os/status", s.handleOSStatus())
	mux.HandleFunc("POST /os/reboot", s.handleOSReboot())

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
