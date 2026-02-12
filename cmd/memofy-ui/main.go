package main

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/tiroq/memofy/internal/ipc"
	"github.com/tiroq/memofy/pkg/macui"
)

const appName = "Memofy"

var (
	// Version is set at build time via -ldflags "-X main.Version=..."
	Version = "dev"

	statusBarApp *macui.StatusBarApp
)

func main() {
	log.Println("Memofy UI starting (version " + Version + ")...")

	// Create status bar app
	statusBarApp = macui.NewStatusBarApp()

	// Load initial status
	if err := updateStatus(); err != nil {
		log.Printf("Failed to load initial status: %v", err)
	}

	// Start watching status file
	watchStatusFile()
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
	defer watcher.Close()

	statusDir := filepath.Join(os.Getenv("HOME"), ".cache", "memofy")
	statusPath := filepath.Join(statusDir, "status.json")

	// Ensure directory exists
	os.MkdirAll(statusDir, 0755)

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
