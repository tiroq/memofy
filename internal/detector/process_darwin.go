//go:build darwin

package detector

import (
	"strings"

	"github.com/progrium/darwinkit/macos/appkit"
)

// ProcessDetection provides utility functions for macOS process detection
type ProcessDetection struct {
	workspace appkit.Workspace
}

// NewProcessDetection creates a new process detector
func NewProcessDetection() *ProcessDetection {
	return &ProcessDetection{
		workspace: appkit.Workspace_SharedWorkspace(),
	}
}

// IsProcessRunning checks if any running app matches the given process patterns
// Matches against both bundle IDs and localized names
func (pd *ProcessDetection) IsProcessRunning(processPatterns []string) (bool, string) {
	apps := pd.workspace.RunningApplications()

	// RunningApplications returns a slice in darwinkit
	for _, app := range apps {
		if app.Ptr() == nil {
			continue
		}

		bundleID := app.BundleIdentifier()
		localizedName := app.LocalizedName()

		// Check if bundle ID or localized name matches any pattern
		for _, pattern := range processPatterns {
			patternLower := strings.ToLower(pattern)

			// Check bundle ID
			if bundleID != "" && strings.Contains(strings.ToLower(bundleID), patternLower) {
				return true, bundleID
			}

			// Check localized name
			if localizedName != "" && strings.Contains(strings.ToLower(localizedName), patternLower) {
				return true, localizedName
			}
		}
	}

	return false, ""
}

// GetActiveWindowTitle returns the localized name of the frontmost application
// Note: This returns the application name, not the actual window title.
// macOS accessibility APIs would be needed for true window titles.
func (pd *ProcessDetection) GetActiveWindowTitle() (string, error) {
	// Get frontmost app
	frontApp := pd.workspace.FrontmostApplication()
	if frontApp.Ptr() == nil {
		return "", nil
	}

	localizedName := frontApp.LocalizedName()
	if localizedName == "" {
		return "", nil
	}

	return localizedName, nil
}

// WindowMatches checks if the frontmost application name contains any of the given hints
// Note: This checks the application name, not the actual window title.
// Returns true if any hint is found in the application name
func (pd *ProcessDetection) WindowMatches(windowHints []string) (bool, string) {
	appName, err := pd.GetActiveWindowTitle()
	if err != nil || appName == "" {
		return false, ""
	}

	appNameLower := strings.ToLower(appName)

	for _, hint := range windowHints {
		if strings.Contains(appNameLower, strings.ToLower(hint)) {
			return true, appName
		}
	}

	return false, appName
}
