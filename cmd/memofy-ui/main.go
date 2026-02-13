package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/progrium/darwinkit/macos/appkit"
	"github.com/tiroq/memofy/internal/ipc"
	"github.com/tiroq/memofy/internal/pidfile"
	"github.com/tiroq/memofy/pkg/macui"
)

var (
	// Version is set at build time via -ldflags "-X main.Version=..."
	Version = "dev"

	statusBarApp *macui.StatusBarApp
)

func main() {
	// Panic recovery - prevents hanging if UI framework crashes
	defer func() {
		if r := recover(); r != nil {
			log.Printf("PANIC: memofy-ui crashed: %v", r)
			fmt.Fprintf(os.Stderr, "FATAL: memofy-ui panicked: %v\n", r)
			os.Exit(1)
		}
	}()

	log.Println("===========================================")
	log.Println("Memofy UI starting (version " + Version + ")...")
	log.Printf("PID: %d", os.Getpid())
	log.Printf("Timestamp: %s", time.Now().Format(time.RFC3339))
	log.Println("===========================================")

	// Check for duplicate instances
	pidFilePath := pidfile.GetPIDFilePath("memofy-ui")
	pf, err := pidfile.New(pidFilePath)
	if err != nil {
		log.Printf("Failed to create PID file: %v", err)
		log.Println("Another instance of memofy-ui may already be running.")
		log.Printf("If you're sure no other instance is running, remove: %s", pidFilePath)
		os.Exit(1)
	}
	defer func() {
		log.Println("[SHUTDOWN] Removing PID file...")
		if err := pf.Remove(); err != nil {
			log.Printf("Warning: failed to remove PID file: %v", err)
		}
	}()
	log.Printf("[STARTUP] PID file created: %s (PID %d)", pidFilePath, os.Getpid())

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		log.Printf("[SHUTDOWN] Received signal %v, cleaning up...", sig)
		if err := pf.Remove(); err != nil {
			log.Printf("Warning: failed to remove PID file: %v", err)
		}
		os.Exit(0)
	}()

	// Initialize macOS application with timeout protection
	log.Println("[STARTUP] Initializing macOS application...")
	
	// Create a timeout context for UI initialization (5 seconds)
	// If UI framework hangs, this will fail fast instead of deadlocking
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Use a channel to track initialization status
	initDone := make(chan bool, 1)
	var initErr error
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				initErr = fmt.Errorf("panic during UI init: %v", r)
				log.Printf("[STARTUP] ERROR: %v", initErr)
			}
		}()

		// Initialize the application
		app := appkit.Application_SharedApplication()
		app.SetActivationPolicy(appkit.ApplicationActivationPolicyAccessory)
		log.Println("[STARTUP] macOS Application initialized")

		// Create status bar app with panic protection
		statusBarApp, err = macui.NewStatusBarApp()
		if err != nil {
			initErr = fmt.Errorf("failed to create status bar app: %w", err)
			log.Printf("[STARTUP] ERROR: %v", initErr)
			return
		}
		log.Println("[STARTUP] Status bar app created successfully")

		initDone <- true
	}()

	// Wait for either initialization to complete or timeout
	select {
	case <-initDone:
		log.Println("[STARTUP] UI initialization completed")
	case <-ctx.Done():
		log.Println("[STARTUP] ERROR: UI initialization timeout (5s) - macOS event loop may be hung")
		wg.Wait()
		if initErr != nil {
			log.Printf("[STARTUP] Error during init: %v", initErr)
		}
		log.Println("[STARTUP] Exiting due to UI initialization failure")
		os.Exit(1)
	}

	if initErr != nil {
		log.Printf("[STARTUP] ERROR: %v", initErr)
		os.Exit(1)
	}

	log.Println("[STARTUP] UI initialization verified")

	// Load initial status
	log.Println("[STARTUP] Loading initial status...")
	if err := updateStatus(); err != nil {
		log.Printf("Failed to load initial status: %v", err)
	}

	// Start watching status file in background
	log.Println("[STARTUP] Starting status file watcher...")
	go watchStatusFile()

	log.Println("===========================================")
	log.Println("[RUNNING] Memofy UI is running")
	log.Println("===========================================")

	// Run the application event loop
	app.Run()
}

// updateStatus reads status.json and updates UI
func updateStatus() error {
	status, err := ipc.ReadStatus()
	if err != nil {
		// If status file doesn't exist yet, use default
		if os.IsNotExist(err) {
			defaultStatus := &ipc.StatusSnapshot{
				Mode:             ipc.ModeAuto,
				TeamsDetected:    false,
				ZoomDetected:     false,
				GoogleMeetActive: false,
				StartStreak:      0,
				StopStreak:       0,
				LastAction:       "initialized",
				LastError:        "",
				Timestamp:        time.Now(),
				OBSConnected:     false,
			}
			statusBarApp.UpdateStatus(defaultStatus)
			return nil
		}
		return err
	}

	statusBarApp.UpdateStatus(status)
	return nil
}

// watchStatusFile monitors status.json for changes
func watchStatusFile() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := watcher.Close(); err != nil {
			log.Printf("Failed to close watcher: %v", err)
		}
	}()

	statusDir := filepath.Join(os.Getenv("HOME"), ".cache", "memofy")
	statusPath := filepath.Join(statusDir, "status.json")

	// Ensure directory exists
	if err := os.MkdirAll(statusDir, 0755); err != nil {
		log.Printf("Warning: failed to create status directory: %v", err)
	}

	// Watch the directory (not the file, as it may be recreated)
	if err := watcher.Add(statusDir); err != nil {
		log.Fatal(err)
	}

	log.Println("Watching status file for changes...")

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			if event.Name == statusPath && (event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create) {
				// Small delay to ensure write is complete
				time.Sleep(50 * time.Millisecond)

				if err := updateStatus(); err != nil {
					log.Printf("Failed to update status: %v", err)
				}
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Printf("Watcher error: %v", err)
		}
	}
}
