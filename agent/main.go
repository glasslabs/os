// Package main is the GlassOS management agent.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/glasslabs/os/agent/api"
	"github.com/glasslabs/os/agent/internal/exec"
	"github.com/glasslabs/os/agent/proc"
	"github.com/hamba/logger/v2"
	lctx "github.com/hamba/logger/v2/ctx"
)

var version = "dev"

func main() {
	addr := flag.String("addr", ":8080", "HTTP server listen address")
	glassBin := flag.String("glass-bin", "/usr/bin/glass", "Path to the glass binary")
	dataDir := flag.String("data-dir", "/data", "Path to the data directory")
	logLevel := flag.String("log.level", "info", "Log level (trace, debug, info, warn, error)")
	flag.Parse()

	lvl, err := logger.LevelFromString(*logLevel)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "invalid log level %q: %v\n", *logLevel, err)
		os.Exit(1)
	}

	log := logger.New(os.Stderr, logger.LogfmtFormat(), lvl)
	log.Info("Starting glass-agent", lctx.Str("version", version), lctx.Str("addr", *addr))

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	exe := &exec.Executable{
		Path: "/usr/bin/cage",
		Args: []string{"--",
			*glassBin, "run",
			"--config", filepath.Join(*dataDir, "config", "config.yaml"),
			"--secrets", filepath.Join(*dataDir, "config", "secrets.yaml"),
			"--assets", filepath.Join(*dataDir, "assets"),
			"--modules", filepath.Join(*dataDir, "modules")},
		SysProcAttr: &syscall.SysProcAttr{Setpgid: true},
	}
	super := proc.New(exe, log)

	apiSrv := api.NewServer(*addr, super, *glassBin, *dataDir, log)

	if err = super.Start(ctx); err != nil {
		log.Error("Could not start supervisor", lctx.Err(err))
		os.Exit(1)
	}

	srv := &http.Server{
		Addr:    *addr,
		Handler: apiSrv,
	}
	go func() {
		<-ctx.Done()
		_ = srv.Shutdown(context.WithoutCancel(ctx))
	}()

	log.Info("Starting server", lctx.Str("addr", *addr))

	if err = srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		log.Error("Could not run server", lctx.Err(err))
		os.Exit(1)
	}
}
