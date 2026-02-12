package macui

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/tiroq/memofy/internal/autoupdate"
	"github.com/tiroq/memofy/internal/config"
	"github.com/tiroq/memofy/internal/ipc"
)

// StatusBarApp represents the menu bar application
// Implements status monitoring, menu display, and command interface
type StatusBarApp struct {
	currentStatus       *ipc.StatusSnapshot
	lastErrorShown      string
	lastErrorTime       time.Time
	settingsWindow      *SettingsWindow
	previousRecording   bool
	recordingStartTime  time.Time
	updateChecker       *autoupdate.UpdateChecker
	lastUpdateCheckTime time.Time
}

// GetCurrentStatus returns the current status snapshot (for testing)
func (app *StatusBarApp) GetCurrentStatus() *ipc.StatusSnapshot {
	return app.currentStatus
}

// NewStatusBarApp creates and initializes the menu bar application
func NewStatusBarApp() *StatusBarApp {
	log.Println("✓ StatusBarApp initialized")
	log.Println("  Control commands:")
	log.Println("    echo 'start' > ~/.cache/memofy/cmd.txt   (force start)")
	log.Println("    echo 'stop' > ~/.cache/memofy/cmd.txt    (force stop)")
	log.Println("    echo 'auto' > ~/.cache/memofy/cmd.txt    (auto mode)")
	log.Println("    echo 'pause' > ~/.cache/memofy/cmd.txt   (pause)")
	log.Println("  Status: cat ~/.cache/memofy/status.json")

	installDir := filepath.Join(os.Getenv("HOME"), ".local", "bin")
	checker := autoupdate.NewUpdateChecker("tiroq", "memofy", "0.1.0", installDir)

	// Set release channel based on config
	cfg, err := config.LoadDetectionRules()
	if err != nil {
		log.Printf("Warning: Could not load config for release channel setting: %v", err)
		checker.SetChannel(autoupdate.ChannelStable) // Default to stable
	} else if cfg.AllowDevUpdates {
		checker.SetChannel(autoupdate.ChannelPrerelease) // Allow pre-releases
		log.Println("✓ Release channel set to: prerelease (dev updates enabled)")
	} else {
		checker.SetChannel(autoupdate.ChannelStable) // Default to stable
		log.Println("✓ Release channel set to: stable")
	}

	return &StatusBarApp{
		settingsWindow: NewSettingsWindow(),
		updateChecker:  checker,
	}
}

// UpdateStatus refreshes the UI based on current status
func (app *StatusBarApp) UpdateStatus(status *ipc.StatusSnapshot) {
	if app.currentStatus == nil {
		// First update - show initial notification
		app.currentStatus = status
		SendNotification("Memofy", "Monitoring Active", "Automatic meeting detector started")
		return
	}

	app.currentStatus = status

	// Detect recording state change (T085: Display recording duration)
	// Check actual recording state from OBS, not just connection status
	isRecording := false
	if recordingState, ok := status.RecordingState.(map[string]interface{}); ok {
		if recording, exists := recordingState["recording"]; exists {
			if recordingBool, ok := recording.(bool); ok {
				isRecording = recordingBool
			}
		}
	}
	
	if isRecording && !app.previousRecording {
		// Started recording
		app.recordingStartTime = time.Now()
		SendNotification("Memofy", "Recording Started", getDetectedAppString(status))
	} else if !isRecording && app.previousRecording {
		// Stopped recording
		duration := time.Since(app.recordingStartTime)
		SendNotification("Memofy", "Recording Stopped", fmt.Sprintf("Duration: %s", formatDuration(duration)))
	}
	app.previousRecording = isRecording

	// Handle error notifications (T081: ERROR state notification)
	if status.LastError != "" && status.LastError != app.lastErrorShown {
		app.lastErrorShown = status.LastError
		app.lastErrorTime = time.Now()
		SendErrorNotification("Memofy Error", status.LastError)
	}

	// Log detailed status (T085: Status display with all information)
	icon := getStatusIcon(status)
	appDetected := getDetectedAppString(status)
	duration := ""
	if isRecording {
		duration = fmt.Sprintf(" (%.0fs)", time.Since(app.recordingStartTime).Seconds())
	}

	log.Printf("%s Status: Mode=%s, App=%s, OBS=%v, Recording=%v%s, Error=%q",
		icon,
		status.Mode,
		appDetected,
		status.OBSConnected,
		isRecording,
		duration,
		status.LastError)
}

// sendCommand writes a command to the command file
func (app *StatusBarApp) sendCommand(cmd ipc.Command) {
	if err := ipc.WriteCommand(cmd); err != nil {
		log.Printf("❌ Error sending command %s: %v", cmd, err)
	}
}

// StartRecording sends start command (T073)
func (app *StatusBarApp) StartRecording() {
	app.sendCommand(ipc.CmdStart)
	SendNotification("Memofy", "Command Sent", "Manual recording started")
}

// StopRecording sends stop command (T074)
func (app *StatusBarApp) StopRecording() {
	app.sendCommand(ipc.CmdStop)
	SendNotification("Memofy", "Command Sent", "Recording stopped")
}

// SetAutoMode sends auto mode command (T075)
func (app *StatusBarApp) SetAutoMode() {
	app.sendCommand(ipc.CmdAuto)
	SendNotification("Memofy", "Mode Changed", "Switched to Auto mode")
}

// SetManualMode sends start command then switches tracking (T076)
func (app *StatusBarApp) SetManualMode() {
	app.sendCommand(ipc.CmdStart)
	SendNotification("Memofy", "Recording Started", "Manual recording started - auto-detection paused")
}

// SetPauseMode sends pause command (T077)
func (app *StatusBarApp) SetPauseMode() {
	app.sendCommand(ipc.CmdPause)
	SendNotification("Memofy", "Mode Changed", "Monitoring paused")
}

// OpenRecordingsFolder opens the OBS recordings directory in Finder (T078)
func (app *StatusBarApp) OpenRecordingsFolder() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Printf("Failed to get home directory: %v", err)
		SendNotification("Memofy", "Error", "Could not determine recordings folder location")
		return
	}
	
	recordingsPath := filepath.Join(homeDir, "Movies", "Memofy")
	cmd := exec.Command("open", recordingsPath, "-a", "Finder")
	if err := cmd.Run(); err != nil {
		log.Printf("Failed to open recordings folder: %v", err)
		SendNotification("Memofy", "Error", "Could not open recordings folder")
	}
}

// OpenLogs opens the /tmp directory in Finder showing logs (T079)
func (app *StatusBarApp) OpenLogs() {
	cmd := exec.Command("open", "/tmp", "-a", "Finder")
	if err := cmd.Run(); err != nil {
		log.Printf("Failed to open logs: %v", err)
		SendNotification("Memofy", "Error", "Could not open logs folder")
	}
}

// ShowSettings opens the settings window (T082-T084)
func (app *StatusBarApp) ShowSettings() {
	if err := app.settingsWindow.showSimpleSettingsDialog(); err != nil {
		log.Printf("Failed to show settings: %v", err)
		SendNotification("Memofy", "Error", "Could not open settings")
	}
}

// GetStatusString returns a formatted status string for display (T085)
func (app *StatusBarApp) GetStatusString() string {
	if app.currentStatus == nil {
		return "Status: Initializing..."
	}

	status := app.currentStatus
	icon := getStatusIcon(status)
	appDetected := getDetectedAppString(status)
	recordingStatus := "Not Recording"

	if status.OBSConnected {
		recordingStatus = "Recording"
		if status.Mode == ipc.ModePaused {
			recordingStatus = "Paused"
		}
	}

	return fmt.Sprintf("%s | Mode: %s | App: %s | %s",
		icon, status.Mode, appDetected, recordingStatus)
}

// Helper functions

// getStatusIcon returns an icon string based on the status
func getStatusIcon(status *ipc.StatusSnapshot) string {
	if status.LastError != "" {
		return "⚠️"
	}
	if status.OBSConnected {
		if status.Mode == ipc.ModePaused {
			return "⏸"
		}
		return "🔴"
	}
	if status.TeamsDetected || status.ZoomDetected {
		return "🟡"
	}
	return "⚪"
}

// getDetectedAppString returns the detected meeting app name
func getDetectedAppString(status *ipc.StatusSnapshot) string {
	if status.ZoomDetected {
		return "Zoom"
	}
	if status.TeamsDetected {
		return "Teams"
	}
	if status.GoogleMeetActive {
		return "Google Meet"
	}
	return "None"
}

// formatDuration formats a duration nicely
func formatDuration(d time.Duration) string {
	if d.Hours() > 0 {
		return fmt.Sprintf("%.1fh", d.Hours())
	}
	if d.Minutes() > 0 {
		return fmt.Sprintf("%.1fm", d.Minutes())
	}
	return fmt.Sprintf("%.0fs", d.Seconds())
}

// CheckForUpdates checks if a new version is available
// Only checks once per hour
func (app *StatusBarApp) CheckForUpdates() (bool, string, error) {
	// Throttle updates to once per hour
	if time.Since(app.lastUpdateCheckTime) < time.Hour {
		return false, "", nil
	}

	app.lastUpdateCheckTime = time.Now()

	available, release, err := app.updateChecker.IsUpdateAvailable()
	if err != nil {
		log.Printf("Update check failed: %v", err)
		return false, "", err
	}

	if available && release != nil {
		return true, release.TagName, nil
	}

	return false, "", nil
}

// UpdateNow downloads and installs the latest version
func (app *StatusBarApp) UpdateNow() {
	SendNotification("Memofy", "Updating...", "Downloading latest version")
	log.Println("Starting update...")

	go func() {
		release, err := app.updateChecker.GetLatestRelease()
		if err != nil {
			log.Printf("Failed to get latest release: %v", err)
			SendErrorNotification("Update Failed", fmt.Sprintf("Could not fetch update: %v", err))
			return
		}

		if err := app.updateChecker.DownloadAndInstall(release); err != nil {
			log.Printf("Update failed: %v", err)
			SendErrorNotification("Update Failed", fmt.Sprintf("Could not install update: %v", err))
			return
		}

		SendNotification("Memofy", "Update Complete", "Version "+release.TagName+" installed. Please restart the app.")
		log.Println("Update completed successfully. Restart required.")

		// Optionally restart the app
		// exec.Command("open", "-a", "Memofy").Run()
	}()
}
