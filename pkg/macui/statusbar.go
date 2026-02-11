package macui

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/progrium/darwinkit/macos/appkit"
	"github.com/progrium/darwinkit/macos/foundation"
	"github.com/tiroq/memofy/internal/ipc"
)

// StatusBarApp represents the menu bar application
type StatusBarApp struct {
	statusItem    appkit.StatusItem
	menu          appkit.Menu
	currentStatus *ipc.StatusSnapshot
	cacheDir      string
}

// IconState represents the current state of the status bar icon
type IconState int

const (
	IconIDLE  IconState = iota // Gray - not recording, no meeting detected
	IconWAIT                   // Yellow - meeting detected, waiting for threshold
	IconREC                    // Red - actively recording
	IconERROR                  // Orange - error condition
)

// NewStatusBarApp creates and initializes the menu bar application
func NewStatusBarApp() *StatusBarApp {
	app := &StatusBarApp{
		cacheDir: filepath.Join(os.Getenv("HOME"), ".cache", "memofy"),
	}

	// Create status item in menu bar
	statusBar := appkit.StatusBar_SystemStatusBar()
	app.statusItem = statusBar.StatusItemWithLength(appkit.VariableStatusItemLength)

	// Set initial icon
	app.updateIcon(IconIDLE)

	// Create menu
	app.menu = appkit.NewMenu()
	app.buildMenu()

	// Attach menu to status item
	app.statusItem.SetMenu(app.menu)

	return app
}

// buildMenu constructs the menu structure
func (app *StatusBarApp) buildMenu() {
	app.menu.RemoveAllItems()

	// Status Section
	statusItem := appkit.NewMenuItemWithSelector("Status: IDLE", nil, "")
	statusItem.SetEnabled(false)
	app.menu.AddItem(statusItem)

	app.menu.AddItem(appkit.SeparatorMenuItem())

	// Controls Section
	startItem := appkit.NewMenuItemWithSelector("Start Recording", app.handleStart, "s")
	app.menu.AddItem(startItem)

	stopItem := appkit.NewMenuItemWithSelector("Stop Recording", app.handleStop, "x")
	app.menu.AddItem(stopItem)

	app.menu.AddItem(appkit.SeparatorMenuItem())

	// Mode Section
	autoItem := appkit.NewMenuItemWithSelector("Auto Mode", app.handleAuto, "a")
	autoItem.SetState(appkit.OnState) // Default mode
	app.menu.AddItem(autoItem)

	pauseItem := appkit.NewMenuItemWithSelector("Pause", app.handlePause, "p")
	app.menu.AddItem(pauseItem)

	app.menu.AddItem(appkit.SeparatorMenuItem())

	// Actions Section
	recordingsItem := appkit.NewMenuItemWithSelector("Open Recordings Folder", app.handleOpenRecordings, "r")
	app.menu.AddItem(recordingsItem)

	logsItem := appkit.NewMenuItemWithSelector("Open Logs", app.handleOpenLogs, "l")
	app.menu.AddItem(logsItem)

	app.menu.AddItem(appkit.SeparatorMenuItem())

	// Settings
	settingsItem := appkit.NewMenuItemWithSelector("Settings...", app.handleSettings, ",")
	app.menu.AddItem(settingsItem)

	app.menu.AddItem(appkit.SeparatorMenuItem())

	// Quit
	quitItem := appkit.NewMenuItemWithSelector("Quit", app.handleQuit, "q")
	app.menu.AddItem(quitItem)
}

// updateIcon changes the menu bar icon based on state
func (app *StatusBarApp) updateIcon(state IconState) {
	button := app.statusItem.Button()
	if button.Ptr() == nil {
		return
	}

	var iconText string
	switch state {
	case IconIDLE:
		iconText = "◯" // Gray circle (will use template image)
	case IconWAIT:
		iconText = "◐" // Half-filled circle
	case IconREC:
		iconText = "●" // Filled circle (recording)
	case IconERROR:
		iconText = "⚠" // Warning symbol
	}

	button.SetTitle(iconText)

	// TODO: In production, use actual icon images instead of text
	// image := appkit.Image_AllocImage().InitWithContentsOfFile(iconPath)
	// button.SetImage(image)
}

// UpdateStatus refreshes the menu based on current status
func (app *StatusBarApp) UpdateStatus(status *ipc.StatusSnapshot) {
	app.currentStatus = status

	// Update icon
	icon := app.determineIconState(status)
	app.updateIcon(icon)

	// Update menu
	app.buildMenu()

	// Update status text in menu
	statusText := app.buildStatusText(status)
	if app.menu.NumberOfItems() > 0 {
		statusItem := app.menu.ItemAtIndex(0)
		statusItem.SetTitle(statusText)
	}
}

// determineIconState calculates the appropriate icon based on status
func (app *StatusBarApp) determineIconState(status *ipc.StatusSnapshot) IconState {
	if status.RecordingState.OBSStatus != "connected" {
		return IconERROR
	}

	if status.RecordingState.Recording {
		return IconREC
	}

	if status.DetectionState.MeetingDetected {
		if status.DetectionStreak >= 1 {
			return IconWAIT
		}
	}

	return IconIDLE
}

// buildStatusText creates the status line for the menu
func (app *StatusBarApp) buildStatusText(status *ipc.StatusSnapshot) string {
	if status.RecordingState.Recording {
		duration := status.RecordingState.Duration
		return fmt.Sprintf("Status: RECORDING (%ds)", duration)
	}

	if status.DetectionState.MeetingDetected {
		return fmt.Sprintf("Status: DETECTED (%s - %d/%d)",
			status.DetectionState.DetectedApp,
			status.DetectionStreak,
			3) // Start threshold
	}

	return fmt.Sprintf("Status: IDLE (Mode: %s)", status.Mode)
}

// Command handlers
func (app *StatusBarApp) handleStart(_ foundation.Object, selector foundation.Selector) {
	app.sendCommand(ipc.CmdStart)
}

func (app *StatusBarApp) handleStop(_ foundation.Object, selector foundation.Selector) {
	app.sendCommand(ipc.CmdStop)
}

func (app *StatusBarApp) handleAuto(_ foundation.Object, selector foundation.Selector) {
	app.sendCommand(ipc.CmdAuto)
}

func (app *StatusBarApp) handlePause(_ foundation.Object, selector foundation.Selector) {
	app.sendCommand(ipc.CmdPaused)
}

func (app *StatusBarApp) handleOpenRecordings(_ foundation.Object, selector foundation.Selector) {
	// Get OBS recording directory from config (default: ~/Movies)
	recordingsDir := filepath.Join(os.Getenv("HOME"), "Movies")
	app.openInFinder(recordingsDir)
}

func (app *StatusBarApp) handleOpenLogs(_ foundation.Object, selector foundation.Selector) {
	app.openInFinder("/tmp")
}

func (app *StatusBarApp) handleSettings(_ foundation.Object, selector foundation.Selector) {
	// TODO: Open settings window
	fmt.Println("Settings not yet implemented")
}

func (app *StatusBarApp) handleQuit(_ foundation.Object, selector foundation.Selector) {
	// Send quit command to daemon
	app.sendCommand(ipc.CmdQuit)

	// Quit UI
	appkit.App().Terminate(nil)
}

// sendCommand writes a command to the command file
func (app *StatusBarApp) sendCommand(cmd string) {
	if err := ipc.WriteCommand(cmd); err != nil {
		fmt.Printf("Error sending command %s: %v\n", cmd, err)
	}
}

// openInFinder opens a path in Finder
func (app *StatusBarApp) openInFinder(path string) {
	workspace := appkit.Workspace_SharedWorkspace()
	url := foundation.URL_FileURLWithPath(path)
	workspace.OpenURL(url)
}
