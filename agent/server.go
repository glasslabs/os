package main

import (
	"context"
	"errors"
	"net/http"

	"github.com/glasslabs/os/agent/handlers"
	"github.com/hamba/logger/v2"
	lctx "github.com/hamba/logger/v2/ctx"
)

type server struct {
	addr string

	mux *http.ServeMux

	log *logger.Logger
}

func newServer(addr string, cfg *handlers.Config) *server {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /status", handlers.Status(cfg))
	mux.HandleFunc("GET /logs", handlers.Logs(cfg))
	mux.HandleFunc("POST /ota", handlers.OTA(cfg))
	mux.HandleFunc("POST /config", handlers.Config_(cfg))
	mux.HandleFunc("POST /secrets", handlers.Secrets(cfg))
	mux.HandleFunc("POST /assets/{name}", handlers.UploadAsset(cfg))
	mux.HandleFunc("DELETE /assets/{name}", handlers.DeleteAsset(cfg))
	mux.HandleFunc("POST /modules/{name}", handlers.UploadModule(cfg))
	mux.HandleFunc("DELETE /modules/{name}", handlers.DeleteModule(cfg))
	mux.HandleFunc("POST /os-update", handlers.OSUpdate(cfg))
	mux.HandleFunc("GET /os-status", handlers.OSStatus(cfg))
	mux.HandleFunc("POST /reboot", handlers.Reboot(cfg))

	return &server{
		addr: addr,
		mux:  mux,
		log:  cfg.Log,
	}
}

// Run starts the HTTP server and blocks until ctx is cancelled.
func (s *server) Run(ctx context.Context) error {
	srv := &http.Server{
		Addr:    s.addr,
		Handler: s.mux,
	}

	go func() {
		<-ctx.Done()
		_ = srv.Shutdown(context.WithoutCancel(ctx))
	}()

	s.log.Info("HTTP server listening", lctx.Str("addr", s.addr))

	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
