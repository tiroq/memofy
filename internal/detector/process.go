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
func (pd *ProcessDetection) IsProcessRunning(processPatterns []string) (bool, string) {
	apps := pd.workspace.RunningApplications()

	for i := uint(0); i < apps.Count(); i++ {
		app := appkit.RunningApplication_fromRef(apps.ObjectAtIndex(i).Ptr())
		if app.Ptr() == nil {
			continue
		}

		bundleID := app.BundleIdentifier()
		if bundleID == "" {
			continue
		}

		bundleIDStr := bundleID.String()

		// Check if bundle ID matches any pattern
		for _, pattern := range processPatterns {
			if strings.Contains(strings.ToLower(bundleIDStr), strings.ToLower(pattern)) {
				return true, bundleIDStr
			}
		}
	}

	return false, ""
}

// GetActiveWindowTitle returns the title of the frontmost application's window
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

	return localizedName.String(), nil
}

// WindowMatches checks if any window title contains the given hints
// Returns true if any hint is found in the window title
func (pd *ProcessDetection) WindowMatches(windowHints []string) (bool, string) {
	windowTitle, err := pd.GetActiveWindowTitle()
	if err != nil || windowTitle == "" {
		return false, ""
	}

	windowTitleLower := strings.ToLower(windowTitle)

	for _, hint := range windowHints {
		if strings.Contains(windowTitleLower, strings.ToLower(hint)) {
			return true, windowTitle
		}
	}

	return false, windowTitle
}
