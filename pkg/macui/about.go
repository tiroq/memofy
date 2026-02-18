package macui

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/tiroq/memofy/internal/autoupdate"
)

// AboutWindow manages the About dialog
type AboutWindow struct {
	version       string
	updateChecker *autoupdate.UpdateChecker
}

// NewAboutWindow creates a new About window
func NewAboutWindow(version string, checker *autoupdate.UpdateChecker) *AboutWindow {
	return &AboutWindow{
		version:       version,
		updateChecker: checker,
	}
}

// Show displays the About dialog with app information and update checker.
// Must be called from a goroutine (NOT the main/UI thread) — it blocks while
// the dialog is open.
func (aw *AboutWindow) Show() error {
	// Create About dialog with app information
	aboutText := fmt.Sprintf(`Memofy - Automatic Meeting Recorder

Version: %s

Automatically detects and records video meetings using OBS Studio.

Supports:
• Zoom meetings
• Microsoft Teams
• Google Meet

Repository: github.com/tiroq/memofy`, aw.version)

	// Escape quotes and newlines for AppleScript
	aboutText = strings.ReplaceAll(aboutText, `"`, `\"`)
	aboutText = strings.ReplaceAll(aboutText, "\n", "\\n")

	script := fmt.Sprintf(`tell application "System Events"
	activate
	return button returned of (display alert "About Memofy" message "%s" buttons {"Check for Updates", "Close"} default button "Close" as informational)
end tell`, aboutText)

	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.CombinedOutput()
	result := strings.TrimSpace(string(output))
	log.Printf("[About] dialog result=%q err=%v", result, err)
	if err != nil {
		log.Printf("About dialog error: %v (output: %s)", err, output)
		return nil
	}

	// Check if user clicked "Check for Updates"
	if result == "Check for Updates" {
		aw.checkForUpdates()
	}

	return nil
}

// checkForUpdates performs the update check.
// Must be called from a goroutine (NOT the main/UI thread).
func (aw *AboutWindow) checkForUpdates() {
	// Show checking notification
	if err := SendNotification("Memofy", "Checking for Updates", "Please wait..."); err != nil {
		log.Printf("Failed to send notification: %v", err)
	}

	available, release, err := aw.updateChecker.IsUpdateAvailable()
	if err != nil {
		errMsg := fmt.Sprintf("Failed to check for updates: %v", err)
		log.Printf("Update check failed: %v", err)
		if notifErr := SendErrorNotification("Update Check Failed", errMsg); notifErr != nil {
			log.Printf("Failed to send error notification: %v", notifErr)
		}
		return
	}

	if available && release != nil {
		log.Printf("Update available: %s (current: %s)", release.TagName, aw.version)
		// Use display alert — it does NOT raise error -128 on non-default button clicks,
		// so osascript always exits 0 regardless of which button the user presses.
		// This avoids the persistent "exit status 1" / Cancel-button gotcha with display dialog.
		msg := fmt.Sprintf("Memofy %s is now available (you have %s).\\n\\nInstall and restart now?", release.TagName, aw.version)
		script := fmt.Sprintf(`tell application "System Events"
	activate
	return button returned of (display alert "Update Available" message "%s" buttons {"Not Now", "Install"} default button "Install" as informational)
end tell`, msg)

		cmd := exec.Command("osascript", "-e", script)
		output, err := cmd.CombinedOutput()
		result := strings.TrimSpace(string(output))
		log.Printf("[Update] dialog result=%q err=%v", result, err)
		if err != nil {
			log.Printf("Update alert error (output: %s): %v", output, err)
			return
		}
		if result == "Install" {
			aw.installUpdate(release)
		}
	} else {
		msg := fmt.Sprintf("You are running the latest version (%s).", aw.version)
		log.Printf("No update: %s", msg)
		if notifErr := SendNotification("Memofy", "Up to Date", msg); notifErr != nil {
			log.Printf("Failed to send notification: %v", notifErr)
		}
	}
}

// RunUpdateCheck is the entry-point for a background-goroutine-triggered update
// check (e.g. from a menu item). It is identical to checkForUpdates but is
// exported so statusbar can call it without going through the About dialog.
func (aw *AboutWindow) RunUpdateCheck() {
	aw.checkForUpdates()
}

// installUpdate downloads and installs the update
func (aw *AboutWindow) installUpdate(release *autoupdate.Release) {
	if err := SendNotification("Memofy", "Downloading Update", fmt.Sprintf("Installing version %s...", release.TagName)); err != nil {
		log.Printf("Warning: failed to send notification: %v", err)
	}
	log.Printf("Starting update to %s...", release.TagName)

	// Run update in background
	go func() {
		if err := aw.updateChecker.DownloadAndInstall(release); err != nil {
			log.Printf("Update failed: %v", err)
			errMsg := fmt.Sprintf("Failed to install update: %v", err)
			if notifErr := SendErrorNotification("Update Failed", errMsg); notifErr != nil {
				log.Printf("Warning: failed to send error notification: %v", notifErr)
			}
			return
		}

		// Show success message and restart
		msg := fmt.Sprintf("Successfully installed version %s!\\n\\nMemofy will now restart to apply the update.", release.TagName)
		script := fmt.Sprintf(`
tell application "System Events"
	activate
	display dialog "%s" buttons {"Restart Now"} default button "Restart Now" with title "Update Complete" with icon note
end tell
`, msg)

		cmd := exec.Command("osascript", "-e", script)
		if err := cmd.Run(); err != nil {
			log.Printf("Failed to show update success dialog: %v", err)
		}

		log.Printf("Update completed successfully to version %s. Restarting...", release.TagName)
		// Restart memofy-core via launchctl so it also picks up the new binary.
		restartCore := exec.Command("launchctl", "kickstart", "-k", "gui/"+fmt.Sprintf("%d", os.Getuid())+"/com.memofy.core")
		if err := restartCore.Run(); err != nil {
			log.Printf("Warning: could not restart memofy-core via launchctl: %v", err)
		}
		// Exit with non-zero so launchd (KeepAlive) restarts the process with the new binary.
		os.Exit(42)
	}()
}
