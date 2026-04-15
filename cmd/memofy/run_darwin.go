//go:build darwin

package main

import (
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/progrium/darwinkit/macos"
	"github.com/progrium/darwinkit/macos/appkit"
	"github.com/tiroq/memofy/internal/config"
	"github.com/tiroq/memofy/internal/engine"
	"github.com/tiroq/memofy/pkg/macui"
)

func init() {
	runtime.LockOSThread()
}

// prevent GC of StatusBarApp while AppKit run loop is active
var statusBarKeepAlive *macui.StatusBarApp

// platformRunLoop starts the macOS menu bar UI and runs the AppKit event loop.
// It blocks until the application terminates.
func platformRunLoop(eng *engine.Engine, cfg config.Config, version string, logger *log.Logger) {
	macos.RunApp(func(app appkit.Application, delegate *appkit.ApplicationDelegate) {
		app.SetActivationPolicy(appkit.ApplicationActivationPolicyAccessory)

		statusBarApp := macui.NewStatusBarApp(version, eng, cfg)
		statusBarKeepAlive = statusBarApp
		statusBarApp.StartUpdateTimer()

		// Handle shutdown signals in a goroutine
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			sig := <-sigCh
			logger.Printf("Received %s, shutting down...", sig)
			eng.Stop()
			app.Terminate(nil)
		}()

		delegate.SetApplicationShouldTerminateAfterLastWindowClosed(func(appkit.Application) bool {
			return false
		})
	})
}
