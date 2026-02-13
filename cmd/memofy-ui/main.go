package main

import (
	"log"
	"os"
	"path/filepath"
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
	log.Println("Memofy UI starting (version " + Version + ")...")

	// Check for duplicate instances
	pidFilePath := pidfile.GetPIDFilePath("memofy-ui")
	pf, err := pidfile.New(pidFilePath)
	if err != nil {
		log.Printf("Failed to create PID file: %v", err)
		log.Println("Another instance of memofy-ui may already be running.")
		log.Printf("If you're sure no other instance is running, remove: %s", pidFilePath)
		os.Exit(1)
	}
	defer pf.Remove()
	log.Printf("PID file created: %s (PID %d)", pidFilePath, os.Getpid())

	// Initialize macOS application
	app := appkit.Application_SharedApplication()
	app.SetActivationPolicy(appkit.ApplicationActivationPolicyAccessory) // Run as menu bar only (no dock icon)

	// Create status bar app
	statusBarApp = macui.NewStatusBarApp()

	// Load initial status
	if err := updateStatus(); err != nil {
		log.Printf("Failed to load initial status: %v", err)
	}

	// Start watching status file in background
	go watchStatusFile()

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
