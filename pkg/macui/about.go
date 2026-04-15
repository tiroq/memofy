package macui

import (
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/tiroq/memofy/internal/autoupdate"
)

// AboutWindow manages the About dialog.
type AboutWindow struct {
	version       string
	updateChecker *autoupdate.UpdateChecker
}

// NewAboutWindow creates a new About window.
func NewAboutWindow(version string, checker *autoupdate.UpdateChecker) *AboutWindow {
	return &AboutWindow{
		version:       version,
		updateChecker: checker,
	}
}

// Show displays the About dialog with app information.
func (aw *AboutWindow) Show() error {
	aboutText := fmt.Sprintf(`Memofy - Lightweight Automatic Audio Recorder

Version: %s

Automatically captures system audio when sound activity is detected. Uses silence-based splitting to create separate recording files.

Supported Platforms:
• macOS (via BlackHole)
• Linux (via PulseAudio / PipeWire)

Repository: github.com/tiroq/memofy
License: MIT`, aw.version)

	aboutText = strings.ReplaceAll(aboutText, `"`, `\"`)
	aboutText = strings.ReplaceAll(aboutText, "\n", "\\n")

	script := fmt.Sprintf(`tell application "System Events"
	activate
	return button returned of (display alert "About Memofy" message "%s" buttons {"Check for Updates", "Close"} default button "Close" as informational)
end tell`, aboutText)

	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.CombinedOutput()
	result := strings.TrimSpace(string(output))
	if err != nil {
		log.Printf("About dialog error: %v (output: %s)", err, output)
		return nil
	}

	if result == "Check for Updates" {
		aw.checkForUpdates()
	}

	return nil
}

// RunUpdateCheck performs a manual update check (callable from menu).
func (aw *AboutWindow) RunUpdateCheck() {
	aw.checkForUpdates()
}

func (aw *AboutWindow) checkForUpdates() {
	_ = SendNotification("Memofy", "Checking for Updates", "Please wait...")

	available, release, err := aw.updateChecker.IsUpdateAvailable()
	if err != nil {
		log.Printf("Update check failed: %v", err)
		_ = SendErrorNotification("Update Check Failed", fmt.Sprintf("Failed to check: %v", err))
		return
	}

	if !available || release == nil {
		msg := fmt.Sprintf("You are running the latest version (%s).", aw.version)
		_ = SendNotification("Memofy", "Up to Date", msg)
		return
	}

	log.Printf("Update available: %s (current: %s)", release.TagName, aw.version)

	msg := fmt.Sprintf("Memofy %s is available (you have %s).\\n\\nVisit the release page to download.", release.TagName, aw.version)
	script := fmt.Sprintf(`tell application "System Events"
	activate
	return button returned of (display alert "Update Available" message "%s" buttons {"Later", "Open Release Page"} default button "Open Release Page" as informational)
end tell`, msg)

	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.CombinedOutput()
	result := strings.TrimSpace(string(output))
	if err != nil {
		log.Printf("Update dialog error: %v", err)
		return
	}

	if result == "Open Release Page" {
		url := fmt.Sprintf("https://github.com/tiroq/memofy/releases/tag/%s", release.TagName)
		openCmd := exec.Command("open", url)
		if err := openCmd.Run(); err != nil {
			log.Printf("Failed to open release page: %v", err)
		}
	}
}
