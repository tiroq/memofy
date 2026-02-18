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

// Show displays the About dialog with app information and update checker
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

	script := fmt.Sprintf(`
tell application "System Events"
	activate
	set response to display dialog "%s" buttons {"Check for Updates", "Close"} default button "Close" with title "About Memofy" with icon note
	
	if button returned of response is "Check for Updates" then
		return "check_updates"
	else
		return "close"
	end if
end tell
`, aboutText)

	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.Output()
	if err != nil {
		log.Printf("About dialog error (may be expected if cancelled): %v", err)
		return nil
	}

	// Check if user clicked "Check for Updates"
	result := strings.TrimSpace(string(output))
	if result == "check_updates" {
		aw.checkForUpdates()
	}

	return nil
}

// checkForUpdates performs the update check
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
		// Ask user if they want to update
		msg := fmt.Sprintf("Version %s is available.\\n\\nCurrent version: %s\\n\\nWould you like to download and install it?", release.TagName, aw.version)
		script := fmt.Sprintf(`
tell application "System Events"
	activate
	display dialog "%s" buttons {"Cancel", "Install Update"} default button "Install Update" with title "Update Available" with icon note
	return button returned
end tell
`, msg)

		cmd := exec.Command("osascript", "-e", script)
		output, err := cmd.Output()
		if err != nil {
			log.Printf("Update dialog cancelled or error: %v", err)
			return
		}

		result := strings.TrimSpace(string(output))
		if result == "Install Update" {
			aw.installUpdate(release)
		}
	} else {
		msg := fmt.Sprintf("You are running the latest version (%s).", aw.version)
		if notifErr := SendNotification("Memofy", "Up to Date", msg); notifErr != nil {
			log.Printf("Failed to send notification: %v", notifErr)
		}
	}
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
