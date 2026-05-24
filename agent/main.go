// Package main is the GlassOS management agent.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/glasslabs/os/agent/handlers"
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

	super := newSupervisor(*glassBin, *dataDir, log)

	srv := newServer(*addr, &handlers.Config{
		Supervisor: super,
		DataDir:    *dataDir,
		Log:        log,
	})

	if err = super.Start(ctx); err != nil {
		log.Error("Could not start supervisor", lctx.Err(err))
		os.Exit(1)
	}

	if err = srv.Run(ctx); err != nil {
		log.Error("Could not run server", lctx.Err(err))
		os.Exit(1)
	}
}
