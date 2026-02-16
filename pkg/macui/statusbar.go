package macui

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/progrium/darwinkit/helper/action"
	"github.com/progrium/darwinkit/macos/appkit"
	"github.com/progrium/darwinkit/objc"
	"github.com/tiroq/memofy/internal/autoupdate"
	"github.com/tiroq/memofy/internal/config"
	"github.com/tiroq/memofy/internal/ipc"
)

// StatusBarApp represents the menu bar application
// Implements status monitoring, menu display, and command interface
type StatusBarApp struct {
	statusItem         appkit.StatusItem
	menu               appkit.Menu
	currentStatus      *ipc.StatusSnapshot
	lastErrorShown     string
	lastErrorTime      time.Time
	settingsWindow     *SettingsWindow
	aboutWindow        *AboutWindow
	previousRecording  bool
	recordingStartTime time.Time
	updateChecker      *autoupdate.UpdateChecker
}

// GetCurrentStatus returns the current status snapshot (for testing)
func (app *StatusBarApp) GetCurrentStatus() *ipc.StatusSnapshot {
	return app.currentStatus
}

// NewStatusBarApp creates and initializes the menu bar application
func NewStatusBarApp(version string) *StatusBarApp {
	log.Println("✓ StatusBarApp initialized")
	log.Println("  Control commands:")
	log.Println("    echo 'start' > ~/.cache/memofy/cmd.txt   (force start)")
	log.Println("    echo 'stop' > ~/.cache/memofy/cmd.txt    (force stop)")
	log.Println("    echo 'auto' > ~/.cache/memofy/cmd.txt    (auto mode)")
	log.Println("    echo 'pause' > ~/.cache/memofy/cmd.txt   (pause)")
	log.Println("  Status: cat ~/.cache/memofy/status.json")

	installDir := filepath.Join(os.Getenv("HOME"), ".local", "bin")
	log.Printf("  Current version: %s", version)
	checker := autoupdate.NewUpdateChecker("tiroq", "memofy", version, installDir)

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

	app := &StatusBarApp{
		settingsWindow: NewSettingsWindow(),
		updateChecker:  checker,
	}

	// Create About window
	app.aboutWindow = NewAboutWindow(version, checker)

	// Create status bar item
	app.createStatusBar()

	return app
}

// createStatusBar initializes the menu bar icon and menu
func (app *StatusBarApp) createStatusBar() {
	// Create status item in system status bar
	statusBar := appkit.StatusBar_SystemStatusBar()
	app.statusItem = statusBar.StatusItemWithLength(appkit.VariableStatusItemLength)

	// Set initial button state
	button := app.statusItem.Button()
	button.SetTitle("⚫") // Initial idle state

	// Create menu
	app.menu = appkit.NewMenu()
	app.menu.SetAutoenablesItems(false)

	// Build menu items
	app.rebuildMenu()

	// Attach menu to status item
	app.statusItem.SetMenu(app.menu)

	log.Println("✓ Menu bar icon created")
}

// UpdateStatus refreshes the UI based on current status
func (app *StatusBarApp) UpdateStatus(status *ipc.StatusSnapshot) {
	// CRITICAL: All GUI updates must happen on main thread for macOS AppKit
	// Schedule the actual update on the main dispatch queue
	if status == nil {
		return
	}

	// Store status for main thread to access
	app.currentStatus = status

	// Perform the update on the main thread
	app.performUpdateOnMainThread(status)
}

// performUpdateOnMainThread handles the actual UI update logic
func (app *StatusBarApp) performUpdateOnMainThread(status *ipc.StatusSnapshot) {
	if app.currentStatus == nil {
		// First update - show initial notification
		app.currentStatus = status
		app.updateMenuBarIcon()
		app.rebuildMenu()
		if err := SendNotification("Memofy", "Monitoring Active", "Automatic meeting detector started"); err != nil {
			log.Printf("Warning: failed to send notification: %v", err)
		}
		return
	}

	app.currentStatus = status

	// Update menu bar icon
	app.updateMenuBarIcon()

	// Rebuild menu to reflect current state
	app.rebuildMenu()

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
		if err := SendNotification("Memofy", "Recording Started", getDetectedAppString(status)); err != nil {
			log.Printf("Warning: failed to send notification: %v", err)
		}
	} else if !isRecording && app.previousRecording {
		// Stopped recording
		duration := time.Since(app.recordingStartTime)
		if err := SendNotification("Memofy", "Recording Stopped", fmt.Sprintf("Duration: %s", formatDuration(duration))); err != nil {
			log.Printf("Warning: failed to send notification: %v", err)
		}
	}
	app.previousRecording = isRecording

	// Handle error notifications (T081: ERROR state notification)
	if status.LastError != "" && status.LastError != app.lastErrorShown {
		app.lastErrorShown = status.LastError
		app.lastErrorTime = time.Now()
		if err := SendErrorNotification("Memofy Error", status.LastError); err != nil {
			log.Printf("Warning: failed to send error notification: %v", err)
		}
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

// updateMenuBarIcon updates the menu bar button icon based on status
func (app *StatusBarApp) updateMenuBarIcon() {
	if app.currentStatus == nil {
		return
	}

	button := app.statusItem.Button()
	icon := getStatusIcon(app.currentStatus)
	button.SetTitle(icon)
}

// rebuildMenu reconstructs the menu based on current status
func (app *StatusBarApp) rebuildMenu() {
	app.menu.RemoveAllItems()

	status := app.currentStatus
	if status == nil {
		item := appkit.NewMenuItem()
		item.SetTitle("Loading...")
		app.menu.AddItem(item)
		return
	}

	// Status header
	isRecording := false
	if recordingState, ok := status.RecordingState.(map[string]interface{}); ok {
		if recording, exists := recordingState["recording"]; exists {
			if recordingBool, ok := recording.(bool); ok {
				isRecording = recordingBool
			}
		}
	}

	statusText := fmt.Sprintf("Status: %s", getStatusIcon(status))
	if isRecording {
		duration := time.Since(app.recordingStartTime)
		statusText += fmt.Sprintf(" (Recording %.0fs)", duration.Seconds())
	}
	statusItem := appkit.NewMenuItem()
	statusItem.SetTitle(statusText)
	statusItem.SetEnabled(false)
	app.menu.AddItem(statusItem)

	app.menu.AddItem(appkit.MenuItem_SeparatorItem())

	// Recording controls
	if isRecording {
		stopItem := appkit.NewMenuItem()
		stopItem.SetTitle("⏹ Stop Recording")
		action.Set(stopItem, app.StopRecording)
		app.menu.AddItem(stopItem)
	} else {
		startItem := appkit.NewMenuItem()
		startItem.SetTitle("▶️ Start Recording")
		action.Set(startItem, app.StartRecording)
		app.menu.AddItem(startItem)
	}

	// Mode selection
	app.menu.AddItem(appkit.MenuItem_SeparatorItem())

	autoItem := appkit.NewMenuItem()
	autoItem.SetTitle("🔄 Auto Mode")
	if status.Mode == ipc.ModeAuto {
		autoItem.SetState(appkit.OnState)
	} else {
		autoItem.SetState(appkit.OffState)
	}
	action.Set(autoItem, app.SetAutoMode)
	app.menu.AddItem(autoItem)

	pauseItem := appkit.NewMenuItem()
	pauseItem.SetTitle("⏸ Pause Detection")
	if status.Mode == ipc.ModePaused {
		pauseItem.SetState(appkit.OnState)
	} else {
		pauseItem.SetState(appkit.OffState)
	}
	action.Set(pauseItem, app.SetPauseMode)
	app.menu.AddItem(pauseItem)

	// Utilities
	app.menu.AddItem(appkit.MenuItem_SeparatorItem())

	settingsItem := appkit.NewMenuItem()
	settingsItem.SetTitle("⚙️ Settings...")
	action.Set(settingsItem, app.ShowSettings)
	app.menu.AddItem(settingsItem)

	logsItem := appkit.NewMenuItem()
	logsItem.SetTitle("📄 Open Logs")
	action.Set(logsItem, app.OpenLogs)
	app.menu.AddItem(logsItem)

	aboutItem := appkit.NewMenuItem()
	aboutItem.SetTitle("ℹ️ About Memofy")
	action.Set(aboutItem, app.ShowAbout)
	app.menu.AddItem(aboutItem)

	// Quit
	app.menu.AddItem(appkit.MenuItem_SeparatorItem())
	quitItem := appkit.NewMenuItem()
	quitItem.SetTitle("Quit")
	quitItem.SetKeyEquivalent("q")
	quitFunc := func(sender objc.Object) {
		appkit.Application_SharedApplication().Terminate(nil)
	}
	action.Set(quitItem, quitFunc)
	app.menu.AddItem(quitItem)
}

// ShowAbout displays the About dialog
func (app *StatusBarApp) ShowAbout(sender objc.Object) {
	if err := app.aboutWindow.Show(); err != nil {
		log.Printf("Failed to show About dialog: %v", err)
		if notifErr := SendNotification("Memofy", "Error", "Could not open About window"); notifErr != nil {
			log.Printf("Warning: failed to send notification: %v", notifErr)
		}
	}
}

// sendCommand writes a command to the daemon
func (app *StatusBarApp) sendCommand(cmd ipc.Command) {
	if err := ipc.WriteCommand(cmd); err != nil {
		log.Printf("Failed to write command: %v", err)
		if notifErr := SendNotification("Memofy", "Error", "Could not send command"); notifErr != nil {
			log.Printf("Warning: failed to send notification: %v", notifErr)
		}
	}
}

// StartRecording sends start command (T075)
func (app *StatusBarApp) StartRecording(sender objc.Object) {
	app.sendCommand(ipc.CmdStart)
	if err := SendNotification("Memofy", "Command Sent", "Starting recording"); err != nil {
		log.Printf("Warning: failed to send notification: %v", err)
	}
}

// StopRecording sends stop command (T076)
func (app *StatusBarApp) StopRecording(sender objc.Object) {
	app.sendCommand(ipc.CmdStop)
	if err := SendNotification("Memofy", "Command Sent", "Stopping recording"); err != nil {
		log.Printf("Warning: failed to send notification: %v", err)
	}
}

// SetAutoMode sends auto mode command (T077)
func (app *StatusBarApp) SetAutoMode(sender objc.Object) {
	app.sendCommand(ipc.CmdAuto)
	if err := SendNotification("Memofy", "Mode Changed", "Automatic detection enabled"); err != nil {
		log.Printf("Warning: failed to send notification: %v", err)
	}
}

// SetPauseMode sends pause command (T077)
func (app *StatusBarApp) SetPauseMode(sender objc.Object) {
	app.sendCommand(ipc.CmdPause)
	if err := SendNotification("Memofy", "Mode Changed", "Monitoring paused"); err != nil {
		log.Printf("Warning: failed to send notification: %v", err)
	}
}

// OpenRecordingsFolder opens the OBS recordings directory in Finder (T078)
// Assumes recordings are saved to ~/Movies/Memofy (OBS default recording path)
// TODO: Read actual recording path from OBS configuration if different
func (app *StatusBarApp) OpenRecordingsFolder() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Printf("Failed to get home directory: %v", err)
		if notifErr := SendNotification("Memofy", "Error", "Could not determine recordings folder location"); notifErr != nil {
			log.Printf("Warning: failed to send notification: %v", notifErr)
		}
		return
	}

	recordingsPath := filepath.Join(homeDir, "Movies", "Memofy")
	cmd := exec.Command("open", recordingsPath, "-a", "Finder")
	if err := cmd.Run(); err != nil {
		log.Printf("Failed to open recordings folder: %v", err)
		if notifErr := SendNotification("Memofy", "Error", "Could not open recordings folder"); notifErr != nil {
			log.Printf("Warning: failed to send notification: %v", notifErr)
		}
	}
}

// OpenLogs opens the /tmp directory in Finder showing logs (T079)
func (app *StatusBarApp) OpenLogs(sender objc.Object) {
	cmd := exec.Command("open", "/tmp", "-a", "Finder")
	if err := cmd.Run(); err != nil {
		log.Printf("Failed to open logs: %v", err)
		if notifErr := SendNotification("Memofy", "Error", "Could not open logs folder"); notifErr != nil {
			log.Printf("Warning: failed to send notification: %v", notifErr)
		}
	}
}

// ShowSettings opens the settings window (T082-T084)
func (app *StatusBarApp) ShowSettings(sender objc.Object) {
	if err := app.settingsWindow.showSimpleSettingsDialog(); err != nil {
		log.Printf("Failed to show settings: %v", err)
		if notifErr := SendNotification("Memofy", "Error", "Could not open settings"); notifErr != nil {
			log.Printf("Warning: failed to send notification: %v", notifErr)
		}
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

// Objective-C compatible menu action handlers
