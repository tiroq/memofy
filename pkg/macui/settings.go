package macui

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/tiroq/memofy/internal/config"
)

// SettingsWindow manages the detection rules configuration UI
type SettingsWindow struct {
	detectionRules *config.DetectionConfig
}

// NewSettingsWindow creates a new settings window
func NewSettingsWindow() *SettingsWindow {
	rules, err := config.LoadDetectionRules()
	if err != nil {
		log.Printf("Failed to load detection rules: %v, using defaults", err)
		rules = &config.DetectionConfig{
			PollInterval: 2, // Default to 2 seconds
			Rules: []config.DetectionRule{
				{
					Application:  "zoom",
					ProcessNames: []string{"zoom.us"},
					WindowHints:  []string{"Zoom Meeting", "Zoom Webinar"},
					Enabled:      true,
				},
				{
					Application:  "teams",
					ProcessNames: []string{"Microsoft Teams"},
					WindowHints:  []string{"Presentation in Teams"},
					Enabled:      true,
				},
			},
			StartThreshold: 3,
			StopThreshold:  6,
		}
	}

	return &SettingsWindow{
		detectionRules: rules,
	}
}

// Show displays the settings UI using AppleScript UI
func (sw *SettingsWindow) Show() error {
	// Create a more detailed settings form
	script := `
tell application "System Events"
	activate
	display dialog "Memofy Detection Settings" buttons {"Save", "Cancel"} default button "Cancel" with title "Settings"
	
	-- Window to collect input
	set response to (display dialog "Configure Meeting Detection" buttons {"Cancel", "OK"} default button "OK" with title "Memofy" with icon note giving up after 3600)
	
	if button returned of response is "OK" then
		-- Show simple confirmation
		display notification "Settings saved" with title "Memofy" subtitle "Detection rules updated"
	end if
end tell
`

	cmd := exec.Command("osascript", "-e", script)
	err := cmd.Run()
	if err != nil {
		log.Printf("Settings dialog error (may be expected if cancelled): %v", err)
		// Don't treat dialog dismissal as error
		return nil
	}

	return nil
}

// ShowSettingsForm displays an interactive form for editing detection rules
func (sw *SettingsWindow) ShowSettingsForm() error {
	// Create a temporary text file with current settings
	defaultConfigPath := filepath.Join(os.Getenv("HOME"), ".config", "memofy", "detection-rules.json")
	if err := os.MkdirAll(filepath.Dir(defaultConfigPath), 0755); err != nil {
		log.Printf("Warning: failed to create config directory: %v", err)
	}

	// Display in system editor
	cmd := exec.Command("open", "-t", defaultConfigPath)
	if err := cmd.Run(); err != nil {
		log.Printf("Failed to open settings file: %v", err)
		// Try alternative
		return sw.showSimpleSettingsDialog()
	}

	return nil
}

// showSimpleSettingsDialog shows a basic settings dialog
func (sw *SettingsWindow) showSimpleSettingsDialog() error {
	// Create AppleScript to show settings options
	script := `
tell application "System Events"
	activate
	set settingsChoice to choose from list {"Edit Detection Rules", "View Current Settings", "Reset to Defaults"} with title "Memofy Settings" with prompt "Select an option:"
	
	if settingsChoice is not false then
		display notification (item 1 of settingsChoice) with title "Memofy" subtitle "Coming soon"
	end if
end tell
`

	cmd := exec.Command("osascript", "-e", script)
	err := cmd.Run()
	if err != nil {
		log.Printf("Settings dialog error: %v", err)
		return nil // Don't treat cancellation as error
	}

	return nil
}

// SaveSettings validates and saves the detection rules
func (sw *SettingsWindow) SaveSettings(zoomProcess, teamsProcess string, startThreshold, stopThreshold int) error {
	// Validate thresholds
	if startThreshold < 1 {
		return fmt.Errorf("start threshold must be >= 1, got %d", startThreshold)
	}
	if stopThreshold < startThreshold {
		return fmt.Errorf("stop threshold (%d) must be >= start threshold (%d)", stopThreshold, startThreshold)
	}

	// Update configuration
	for i, rule := range sw.detectionRules.Rules {
		switch rule.Application {
		case "zoom":
			sw.detectionRules.Rules[i].ProcessNames = []string{zoomProcess}
		case "teams":
			sw.detectionRules.Rules[i].ProcessNames = []string{teamsProcess}
		}
	}
	sw.detectionRules.StartThreshold = startThreshold
	sw.detectionRules.StopThreshold = stopThreshold

	// Save to file
	if err := config.SaveDetectionRules(sw.detectionRules); err != nil {
		return fmt.Errorf("failed to save detection rules: %v", err)
	}

	log.Printf("✓ Settings saved: Zoom=%s, Teams=%s, thresholds=%d/%d",
		zoomProcess, teamsProcess, startThreshold, stopThreshold)

	if err := SendNotification("Memofy", "Settings Updated", "Detection rules saved successfully"); err != nil {
		log.Printf("Warning: failed to send notification: %v", err)
	}

	return nil
}

// LoadSettingsFromFile reads settings from the JSON file
func (sw *SettingsWindow) LoadSettingsFromFile() error {
	configPath := filepath.Join(os.Getenv("HOME"), ".config", "memofy", "detection-rules.json")

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Will use defaults
		}
		return err
	}

	var rules config.DetectionConfig
	if err := json.Unmarshal(data, &rules); err != nil {
		return fmt.Errorf("invalid JSON in detection rules: %v", err)
	}

	sw.detectionRules = &rules
	return nil
}

// GetCurrentSettings returns the current settings as a formatted string
func (sw *SettingsWindow) GetCurrentSettings() string {
	zoomProcess := "unknown"
	zoomWindowHints := []string{}
	teamsProcess := "unknown"
	teamsWindowHints := []string{}

	for _, rule := range sw.detectionRules.Rules {
		switch rule.Application {
		case "zoom":
			if len(rule.ProcessNames) > 0 {
				zoomProcess = rule.ProcessNames[0]
			}
			zoomWindowHints = rule.WindowHints
		case "teams":
			if len(rule.ProcessNames) > 0 {
				teamsProcess = rule.ProcessNames[0]
			}
			teamsWindowHints = rule.WindowHints
		}
	}

	return fmt.Sprintf(`
Memofy Detection Settings:
=======================

Zoom Detection:
  Process: %s
  Window Hints: %s

Teams Detection:
  Process: %s
  Window Hints: %s

Thresholds:
  Start Recording: %d detections
  Stop Recording: %d non-detections

Settings File: %s
`,
		zoomProcess,
		strings.Join(zoomWindowHints, ", "),
		teamsProcess,
		strings.Join(teamsWindowHints, ", "),
		sw.detectionRules.StartThreshold,
		sw.detectionRules.StopThreshold,
		filepath.Join(os.Getenv("HOME"), ".config", "memofy", "detection-rules.json"),
	)
}
