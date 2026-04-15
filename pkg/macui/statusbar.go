//go:build darwin

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
	"github.com/progrium/darwinkit/macos/foundation"
	"github.com/progrium/darwinkit/objc"
	"github.com/tiroq/memofy/internal/autoupdate"
	"github.com/tiroq/memofy/internal/config"
	"github.com/tiroq/memofy/internal/engine"
)

// StatusBarApp represents the macOS menu bar application.
type StatusBarApp struct {
	statusItem     appkit.StatusItem
	menu           appkit.Menu
	eng            *engine.Engine
	cfg            config.Config
	version        string
	settingsWindow *SettingsWindow
	aboutWindow    *AboutWindow
	updateChecker  *autoupdate.UpdateChecker
	lastStatus     engine.StatusSnapshot
	wasRecording   bool
	recordStart    time.Time
}

// NewStatusBarApp creates and initializes the menu bar application.
func NewStatusBarApp(version string, eng *engine.Engine, cfg config.Config) *StatusBarApp {
	checker := autoupdate.NewUpdateChecker("tiroq", "memofy", version, "")
	checker.SetChannel(autoupdate.ChannelStable)

	app := &StatusBarApp{
		eng:           eng,
		cfg:           cfg,
		version:       version,
		updateChecker: checker,
	}

	app.settingsWindow = NewSettingsWindow(cfg)
	app.aboutWindow = NewAboutWindow(version, checker)
	app.createStatusBar()

	log.Printf("Menu bar app initialized (version %s)", version)
	return app
}

// createStatusBar initializes the menu bar icon and menu.
func (app *StatusBarApp) createStatusBar() {
	statusBar := appkit.StatusBar_SystemStatusBar()
	app.statusItem = statusBar.StatusItemWithLength(appkit.VariableStatusItemLength)
	app.statusItem.Retain() // prevent deallocation when autorelease pool drains

	button := app.statusItem.Button()
	button.SetTitle("")
	button.SetImage(loadMenubarIcon())

	app.menu = appkit.NewMenu()
	app.menu.Retain() // prevent deallocation when autorelease pool drains
	app.menu.SetAutoenablesItems(false)
	app.rebuildMenu()
	app.statusItem.SetMenu(app.menu)
}

// StartUpdateTimer schedules a repeating timer that polls engine status
// and updates the menu bar UI. Must be called from the main thread.
func (app *StatusBarApp) StartUpdateTimer() {
	foundation.Timer_ScheduledTimerWithTimeIntervalRepeatsBlock(0.5, true, func(_ foundation.Timer) {
		app.pollAndUpdate()
	})
	log.Println("UI update timer started (0.5s)")
}

// pollAndUpdate reads current engine status and updates the UI.
func (app *StatusBarApp) pollAndUpdate() {
	if app.eng == nil {
		return
	}

	status := app.eng.GetStatus()
	isRecording := status.State == "recording" || status.State == "silence_wait"

	// Detect state changes
	stateChanged := status.State != app.lastStatus.State
	recordingChanged := isRecording != app.wasRecording

	if stateChanged {
		app.updateMenuBarIcon(status)
		app.rebuildMenu()
	}

	if recordingChanged {
		if isRecording {
			app.recordStart = time.Now()
			_ = SendNotification("Memofy", "Recording Started", fmt.Sprintf("Device: %s", status.DeviceName))
		} else if app.wasRecording {
			dur := time.Since(app.recordStart)
			_ = SendNotification("Memofy", "Recording Stopped", fmt.Sprintf("Duration: %s", FormatDuration(dur.Seconds())))
		}
		app.wasRecording = isRecording
	}

	// Handle errors
	if status.LastError != "" && status.LastError != app.lastStatus.LastError {
		_ = SendErrorNotification("Memofy Error", status.LastError)
	}

	app.lastStatus = status
}

// updateMenuBarIcon sets a state indicator in the button title.
// The template icon handles its own tinting; we prefix a symbol for active states.
func (app *StatusBarApp) updateMenuBarIcon(status engine.StatusSnapshot) {
	button := app.statusItem.Button()
	switch status.State {
	case "recording":
		button.SetTitle("⏺")
	case "silence_wait":
		button.SetTitle("⏸")
	case "arming":
		button.SetTitle("…")
	case "error":
		button.SetTitle("!")
	default:
		button.SetTitle("")
	}
}

// rebuildMenu constructs the menu based on current status.
func (app *StatusBarApp) rebuildMenu() {
	app.menu.RemoveAllItems()

	status := app.lastStatus

	// Status header
	stateLabel := stateDisplayName(status.State)
	statusText := fmt.Sprintf("Status: %s", stateLabel)
	if status.State == "recording" || status.State == "silence_wait" {
		if !status.RecordingStart.IsZero() {
			dur := time.Since(status.RecordingStart)
			statusText += fmt.Sprintf(" (%.0fs)", dur.Seconds())
		}
	}
	statusItem := appkit.NewMenuItem()
	statusItem.SetTitle(statusText)
	statusItem.SetEnabled(false)
	app.menu.AddItem(statusItem)

	// Device info
	if status.DeviceName != "" {
		devItem := appkit.NewMenuItem()
		devItem.SetTitle(fmt.Sprintf("Device: %s", status.DeviceName))
		devItem.SetEnabled(false)
		app.menu.AddItem(devItem)
	}

	// Format profile
	profileLabel := status.FormatProfile
	if profileLabel == "" {
		profileLabel = "high"
	}
	formatItem := appkit.NewMenuItem()
	formatItem.SetTitle(fmt.Sprintf("Format: %s", profileLabel))
	formatItem.SetEnabled(false)
	app.menu.AddItem(formatItem)

	// Current file
	if status.CurrentFile != "" {
		fileItem := appkit.NewMenuItem()
		fileItem.SetTitle(fmt.Sprintf("File: %s", filepath.Base(status.CurrentFile)))
		fileItem.SetEnabled(false)
		app.menu.AddItem(fileItem)
	}

	app.menu.AddItem(appkit.MenuItem_SeparatorItem())

	// Change Format submenu
	formatMenu := appkit.NewMenu()
	formatMenu.SetTitle("Change Format")
	for _, profile := range []string{"high", "balanced", "lightweight", "wav"} {
		p := profile // capture for closure
		label := formatDisplayName(p)
		if p == profileLabel {
			label = "✓ " + label
		}
		item := appkit.NewMenuItem()
		item.SetTitle(label)
		action.Set(item, func(_ objc.Object) {
			app.eng.SetFormatProfile(p)
			app.rebuildMenu()
			log.Printf("Format changed to: %s", p)
		})
		formatMenu.AddItem(item)
	}
	changeFormatItem := appkit.NewMenuItem()
	changeFormatItem.SetTitle("Change Format")
	changeFormatItem.SetSubmenu(formatMenu)
	app.menu.AddItem(changeFormatItem)

	// Open Recordings Folder
	recordingsItem := appkit.NewMenuItem()
	recordingsItem.SetTitle("Open Recordings Folder")
	action.Set(recordingsItem, func(_ objc.Object) {
		dir := config.ResolvePath(app.cfg.Output.Dir)
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Printf("Failed to create recordings folder: %v", err)
			return
		}
		cmd := exec.Command("open", dir)
		if err := cmd.Run(); err != nil {
			log.Printf("Failed to open recordings folder: %v", err)
		}
	})
	app.menu.AddItem(recordingsItem)

	// Settings
	settingsItem := appkit.NewMenuItem()
	settingsItem.SetTitle("Settings...")
	action.Set(settingsItem, func(_ objc.Object) {
		if err := app.settingsWindow.Show(); err != nil {
			log.Printf("Failed to show settings: %v", err)
		}
	})
	app.menu.AddItem(settingsItem)

	app.menu.AddItem(appkit.MenuItem_SeparatorItem())

	// Check for Updates
	updateItem := appkit.NewMenuItem()
	updateItem.SetTitle("Check for Updates...")
	action.Set(updateItem, func(_ objc.Object) {
		app.aboutWindow.RunUpdateCheck()
	})
	app.menu.AddItem(updateItem)

	// About
	aboutItem := appkit.NewMenuItem()
	aboutItem.SetTitle("About Memofy")
	action.Set(aboutItem, func(_ objc.Object) {
		if err := app.aboutWindow.Show(); err != nil {
			log.Printf("Failed to show About: %v", err)
		}
	})
	app.menu.AddItem(aboutItem)

	// Quit
	app.menu.AddItem(appkit.MenuItem_SeparatorItem())
	quitItem := appkit.NewMenuItem()
	quitItem.SetTitle("Quit")
	quitItem.SetKeyEquivalent("q")
	action.Set(quitItem, func(_ objc.Object) {
		if app.eng != nil {
			app.eng.Stop()
		}
		appkit.Application_SharedApplication().Terminate(nil)
	})
	app.menu.AddItem(quitItem)
}

// stateDisplayName returns a human-readable name for a state.
func stateDisplayName(state string) string {
	switch state {
	case "idle":
		return "Idle"
	case "arming":
		return "Listening"
	case "recording":
		return "Recording"
	case "silence_wait":
		return "Recording (silence)"
	case "finalizing":
		return "Finalizing"
	case "error":
		return "Error"
	default:
		return state
	}
}

// formatDisplayName returns a human-readable name for a format profile.
func formatDisplayName(profile string) string {
	switch profile {
	case "high":
		return "High Quality"
	case "balanced":
		return "Balanced"
	case "lightweight":
		return "Lightweight"
	case "wav":
		return "WAV (Raw)"
	default:
		return profile
	}
}
