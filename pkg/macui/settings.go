package macui

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
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
	// Create settings window with proper UI
	return sw.showNativeSettingsWindow()
}

// showNativeSettingsWindow creates a comprehensive settings window using AppleScript for simplicity
// This allows for a scrollable form with all settings
func (sw *SettingsWindow) showNativeSettingsWindow() error {
	configPath := filepath.Join(os.Getenv("HOME"), ".config", "memofy", "detection-rules.json")

	// Ensure config exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := config.SaveDetectionRules(sw.detectionRules); err != nil {
			log.Printf("Failed to create config: %v", err)
			return err
		}
	}

	// Get current values
	var zoomProcesses, zoomHints, teamsProcesses, teamsHints, meetProcesses, meetHints string
	zoomEnabled, teamsEnabled, meetEnabled := "true", "true", "true"

	for _, rule := range sw.detectionRules.Rules {
		switch rule.Application {
		case "zoom":
			zoomProcesses = strings.Join(rule.ProcessNames, ", ")
			zoomHints = strings.Join(rule.WindowHints, ", ")
			zoomEnabled = fmt.Sprintf("%t", rule.Enabled)
		case "teams":
			teamsProcesses = strings.Join(rule.ProcessNames, ", ")
			teamsHints = strings.Join(rule.WindowHints, ", ")
			teamsEnabled = fmt.Sprintf("%t", rule.Enabled)
		case "meet":
			meetProcesses = strings.Join(rule.ProcessNames, ", ")
			meetHints = strings.Join(rule.WindowHints, ", ")
			meetEnabled = fmt.Sprintf("%t", rule.Enabled)
		}
	}

	// Build the form using AppleScript
	script := fmt.Sprintf(`
tell application "System Events"
	activate
	
	-- Settings form
	set settingsDialog to display dialog "MEMOFY SETTINGS
	
Configure meeting detection rules and thresholds.

ZOOM DETECTION
Process names: %s
Window hints: %s
Enabled: %s

TEAMS DETECTION  
Process names: %s
Window hints: %s
Enabled: %s

GOOGLE MEET DETECTION
Process names: %s  
Window hints: %s
Enabled: %s

THRESHOLDS
Poll interval: %d seconds
Start threshold: %d detections
Stop threshold: %d non-detections  
Allow dev updates: %t

Choose an option:" buttons {"Edit Config", "Advanced", "Close"} default button "Close" with title "Memofy Settings" giving up after 3600
	
	set buttonChoice to button returned of settingsDialog
	
	if buttonChoice is "Edit Config" then
		return "edit"
	else if buttonChoice is "Advanced" then
		return "advanced"
	else
		return "close"
	end if
end tell
`, escapeAppleScript(zoomProcesses), escapeAppleScript(zoomHints), zoomEnabled,
		escapeAppleScript(teamsProcesses), escapeAppleScript(teamsHints), teamsEnabled,
		escapeAppleScript(meetProcesses), escapeAppleScript(meetHints), meetEnabled,
		sw.detectionRules.PollInterval, sw.detectionRules.StartThreshold,
		sw.detectionRules.StopThreshold, sw.detectionRules.AllowDevUpdates)

	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Settings dialog dismissed: %v", err)
		return nil
	}

	choice := strings.TrimSpace(string(output))
	switch choice {
	case "edit":
		// Open config file in editor
		if err := exec.Command("open", "-e", configPath).Run(); err != nil {
			log.Printf("Failed to open config in editor: %v", err)
		}
	case "advanced":
		// Show advanced settings editor
		return sw.showAdvancedSettings()
	}

	return nil
}

// showAdvancedSettings shows a form to edit individual settings
func (sw *SettingsWindow) showAdvancedSettings() error {
	// Get current values
	var zoomProcesses, teamsProcesses, meetProcesses string
	for _, rule := range sw.detectionRules.Rules {
		switch rule.Application {
		case "zoom":
			zoomProcesses = strings.Join(rule.ProcessNames, ", ")
		case "teams":
			teamsProcesses = strings.Join(rule.ProcessNames, ", ")
		case "meet":
			meetProcesses = strings.Join(rule.ProcessNames, ", ")
		}
	}

	script := fmt.Sprintf(`
tell application "System Events"
	activate
	
	-- Get Poll Interval
	set pollInterval to text returned of (display dialog "Poll Interval (1-10 seconds):" default answer "%d" with title "Memofy Settings")
	
	-- Get Start Threshold
	set startThresh to text returned of (display dialog "Start Threshold (1-10 detections):" default answer "%d" with title "Memofy Settings")
	
	-- Get Stop Threshold  
	set stopThresh to text returned of (display dialog "Stop Threshold (>= start, detections):" default answer "%d" with title "Memofy Settings")
	
	-- Get Zoom process
	set zoomProc to text returned of (display dialog "Zoom Process Name:" default answer "%s" with title "Memofy Settings")
	
	-- Get Teams process
	set teamsProc to text returned of (display dialog "Teams Process Name:" default answer "%s" with title "Memofy Settings")
	
	-- Get Meet browser processes
	set meetProc to text returned of (display dialog "Google Meet Browsers (comma-separated):" default answer "%s" with title "Memofy Settings")
	
	-- Get dev updates preference
	set devUpdates to button returned of (display dialog "Allow development/pre-release updates?" buttons {"No", "Yes"} default button "No" with title "Memofy Settings")
	
	set allowDev to "false"
	if devUpdates is "Yes" then
		set allowDev to "true"
	end if
	
	-- Return all values separated by |
	return pollInterval & "|" & startThresh & "|" & stopThresh & "|" & zoomProc & "|" & teamsProc & "|" & meetProc & "|" & allowDev
end tell
`, sw.detectionRules.PollInterval, sw.detectionRules.StartThreshold, sw.detectionRules.StopThreshold,
		escapeAppleScript(zoomProcesses), escapeAppleScript(teamsProcesses), escapeAppleScript(meetProcesses))

	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Advanced settings cancelled: %v", err)
		return nil
	}

	// Parse the response
	result := strings.TrimSpace(string(output))
	parts := strings.Split(result, "|")
	if len(parts) != 7 {
		log.Printf("Invalid response from settings dialog: %s", result)
		return nil
	}

	// Convert and validate
	pollInterval, err := strconv.Atoi(parts[0])
	if err != nil || pollInterval < 1 || pollInterval > 10 {
		return fmt.Errorf("invalid poll interval: %s (must be 1-10)", parts[0])
	}

	startThresh, err := strconv.Atoi(parts[1])
	if err != nil || startThresh < 1 || startThresh > 10 {
		return fmt.Errorf("invalid start threshold: %s (must be 1-10)", parts[1])
	}

	stopThresh, err := strconv.Atoi(parts[2])
	if err != nil || stopThresh < startThresh {
		return fmt.Errorf("invalid stop threshold: %s (must be >= %d)", parts[2], startThresh)
	}

	zoomProc := strings.TrimSpace(parts[3])
	teamsProc := strings.TrimSpace(parts[4])
	meetProc := strings.TrimSpace(parts[5])
	allowDev := parts[6] == "true"

	// Update config
	sw.detectionRules.PollInterval = pollInterval
	sw.detectionRules.StartThreshold = startThresh
	sw.detectionRules.StopThreshold = stopThresh
	sw.detectionRules.AllowDevUpdates = allowDev

	// Update rules
	for i, rule := range sw.detectionRules.Rules {
		switch rule.Application {
		case "zoom":
			if zoomProc != "" {
				sw.detectionRules.Rules[i].ProcessNames = strings.Split(zoomProc, ",")
				for j := range sw.detectionRules.Rules[i].ProcessNames {
					sw.detectionRules.Rules[i].ProcessNames[j] = strings.TrimSpace(sw.detectionRules.Rules[i].ProcessNames[j])
				}
			}
		case "teams":
			if teamsProc != "" {
				sw.detectionRules.Rules[i].ProcessNames = strings.Split(teamsProc, ",")
				for j := range sw.detectionRules.Rules[i].ProcessNames {
					sw.detectionRules.Rules[i].ProcessNames[j] = strings.TrimSpace(sw.detectionRules.Rules[i].ProcessNames[j])
				}
			}
		case "meet":
			if meetProc != "" {
				sw.detectionRules.Rules[i].ProcessNames = strings.Split(meetProc, ",")
				for j := range sw.detectionRules.Rules[i].ProcessNames {
					sw.detectionRules.Rules[i].ProcessNames[j] = strings.TrimSpace(sw.detectionRules.Rules[i].ProcessNames[j])
				}
			}
		}
	}

	// Save changes
	if err := config.SaveDetectionRules(sw.detectionRules); err != nil {
		return fmt.Errorf("failed to save settings: %v", err)
	}

	log.Printf("✓ Settings saved: poll=%ds, thresholds=%d/%d, dev_updates=%t",
		pollInterval, startThresh, stopThresh, allowDev)

	if err := SendNotification("Memofy", "Settings Updated", "Detection rules saved successfully"); err != nil {
		log.Printf("Warning: failed to send notification: %v", err)
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
