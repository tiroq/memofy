//go:build !darwin

package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/tiroq/memofy/internal/config"
	"github.com/tiroq/memofy/internal/engine"
)

// platformRunLoop waits for a shutdown signal. On non-macOS platforms there is no GUI.
func platformRunLoop(eng *engine.Engine, cfg config.Config, version string, logger *log.Logger) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh

	logger.Printf("Received %s, shutting down...", sig)
	eng.Stop()
}
