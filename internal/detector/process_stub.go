//go:build !darwin

package detector

import "fmt"

// ProcessDetection provides utility functions for process detection
// This is a stub implementation for non-darwin platforms
type ProcessDetection struct{}

// NewProcessDetection creates a new process detector
func NewProcessDetection() *ProcessDetection {
	return &ProcessDetection{}
}

// IsProcessRunning checks if any running app matches the given process patterns
func (pd *ProcessDetection) IsProcessRunning(processPatterns []string) (bool, string) {
	return false, ""
}

// GetActiveWindowTitle returns the title of the frontmost application's window
func (pd *ProcessDetection) GetActiveWindowTitle() (string, error) {
	return "", fmt.Errorf("process detection not supported on this platform")
}

// WindowMatches checks if any window title contains the given hints
func (pd *ProcessDetection) WindowMatches(windowHints []string) (bool, string) {
	return false, ""
}
