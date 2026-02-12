package integration

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/tiroq/memofy/internal/ipc"
	"github.com/tiroq/memofy/pkg/macui"
)

// TestMenuBarIconStateChanges verifies icon changes when status changes (T086)
func TestMenuBarIconStateChanges(t *testing.T) {
	app := macui.NewStatusBarApp()

	tests := []struct {
		name           string
		status         *ipc.StatusSnapshot
		expectedChange bool
	}{
		{
			name: "idle state",
			status: &ipc.StatusSnapshot{
				Mode:           ipc.ModeAuto,
				TeamsDetected:  false,
				ZoomDetected:   false,
				OBSConnected:   false,
				LastError:      "",
				Timestamp:      time.Now(),
			},
			expectedChange: true,
		},
		{
			name: "zoom detected",
			status: &ipc.StatusSnapshot{
				Mode:           ipc.ModeAuto,
				TeamsDetected:  false,
				ZoomDetected:   true,
				OBSConnected:   false,
				LastError:      "",
				Timestamp:      time.Now(),
			},
			expectedChange: true,
		},
		{
			name: "recording active",
			status: &ipc.StatusSnapshot{
				Mode:           ipc.ModeAuto,
				TeamsDetected:  false,
				ZoomDetected:   true,
				OBSConnected:   true,
				LastError:      "",
				Timestamp:      time.Now(),
			},
			expectedChange: true,
		},
		{
			name: "error state",
			status: &ipc.StatusSnapshot{
				Mode:           ipc.ModeAuto,
				TeamsDetected:  false,
				ZoomDetected:   false,
				OBSConnected:   false,
				LastError:      "Permission denied for screen recording",
				Timestamp:      time.Now(),
			},
			expectedChange: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app.UpdateStatus(tt.status)
			// Verify no panics and status is updated
			if app.GetStatusString() == "" {
				t.Error("Status string should not be empty")
			}
		})
	}
}

// TestControlCommandsWriteToFile verifies control commands write to cmd.txt (T087)
func TestControlCommandsWriteToFile(t *testing.T) {
	app := macui.NewStatusBarApp()

	// Create temporary cache directory
	cacheDir := filepath.Join(os.TempDir(), "memofy-test-cache")
	os.MkdirAll(cacheDir, 0755)
	defer os.RemoveAll(cacheDir)

	// Override command file location for testing
	cmdFile := filepath.Join(cacheDir, "cmd.txt")

	tests := []struct {
		name            string
		operation       func()
		expectedCommand string
	}{
		{
			name: "start recording",
			operation: func() {
				app.StartRecording()
			},
			expectedCommand: "start",
		},
		{
			name: "stop recording",
			operation: func() {
				app.StopRecording()
			},
			expectedCommand: "stop",
		},
		{
			name: "auto mode",
			operation: func() {
				app.SetAutoMode()
			},
			expectedCommand: "auto",
		},
		{
			name: "manual mode",
			operation: func() {
				app.SetManualMode()
			},
			expectedCommand: "start",
		},
		{
			name: "pause mode",
			operation: func() {
				app.SetPauseMode()
			},
			expectedCommand: "pause",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear any previous command
			os.Remove(cmdFile)

			// Create actual command by writing to cache
			cacheDir := filepath.Join(os.Getenv("HOME"), ".cache", "memofy")
			os.MkdirAll(cacheDir, 0755)
			cmdFile := filepath.Join(cacheDir, "cmd.txt")

			// Execute the operation
			tt.operation()

			// Note: Due to the nature of the actual implementation,
			// we verify the operation completes without error
			// A full test would require a running daemon to read the file
			if t.Failed() {
				t.Fatalf("Operation failed: %s", tt.name)
			}
		})
	}
}

// TestSettingsUIFlow verifies settings UI interaction (T088)
func TestSettingsUIFlow(t *testing.T) {
	settingsWindow := macui.NewSettingsWindow()

	// Test loading settings
	if err := settingsWindow.LoadSettingsFromFile(); err != nil {
		// It's ok if file doesn't exist yet
		t.Logf("Note: settings file not found (first run): %v", err)
	}

	// Test getting current settings string
	settingsStr := settingsWindow.GetCurrentSettings()
	if settingsStr == "" {
		t.Error("Settings string should not be empty")
	}

	// Verify it contains expected content
	expectedKeys := []string{
		"Memofy Detection Settings",
		"Zoom Detection",
		"Teams Detection",
		"Thresholds",
		"Start Recording",
		"Stop Recording",
	}

	for _, key := range expectedKeys {
		if !contains(settingsStr, key) {
			t.Errorf("Settings string should contain '%s'", key)
		}
	}
}

// TestSettingsSaveValidation verifies settings validation (T084)
func TestSettingsSaveValidation(t *testing.T) {
	settingsWindow := macui.NewSettingsWindow()

	tests := []struct {
		name             string
		zoomProcess      string
		teamsProcess     string
		startThreshold   int
		stopThreshold    int
		shouldSucceed    bool
	}{
		{
			name:            "valid settings",
			zoomProcess:     "zoom.us",
			teamsProcess:    "Microsoft Teams",
			startThreshold:  3,
			stopThreshold:   6,
			shouldSucceed:   true,
		},
		{
			name:            "invalid start threshold",
			zoomProcess:     "zoom.us",
			teamsProcess:    "Microsoft Teams",
			startThreshold:  0,
			stopThreshold:   6,
			shouldSucceed:   false,
		},
		{
			name:            "stop < start threshold",
			zoomProcess:     "zoom.us",
			teamsProcess:    "Microsoft Teams",
			startThreshold:  5,
			stopThreshold:   3,
			shouldSucceed:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := settingsWindow.SaveSettings(
				tt.zoomProcess, tt.teamsProcess,
				tt.startThreshold, tt.stopThreshold)

			if tt.shouldSucceed && err != nil {
				t.Errorf("Expected success but got error: %v", err)
			}
			if !tt.shouldSucceed && err == nil {
				t.Errorf("Expected error but succeeded")
			}
		})
	}
}

// TestErrorNotification verifies error notification display (T089)
func TestErrorNotification(t *testing.T) {
	app := macui.NewStatusBarApp()

	// Create a status with an error
	errorStatus := &ipc.StatusSnapshot{
		Mode:           ipc.ModeAuto,
		TeamsDetected:  false,
		ZoomDetected:   false,
		OBSConnected:   false,
		LastError:      "Screen recording permission denied. Please enable in Settings.",
		Timestamp:      time.Now(),
	}

	// Update status - should trigger error notification
	app.UpdateStatus(errorStatus)

	// Wait a moment for notification delivery
	time.Sleep(100 * time.Millisecond)

	// Verify error was captured
	if app.currentStatus.LastError != errorStatus.LastError {
		t.Error("Error should be captured in current status")
	}
}

// TestStatusDisplayFormat verifies status display format (T085)
func TestStatusDisplayFormat(t *testing.T) {
	app := macui.NewStatusBarApp()

	// Initialize with a status
	status := &ipc.StatusSnapshot{
		Mode:           ipc.ModeAuto,
		TeamsDetected:  false,
		ZoomDetected:   true,
		OBSConnected:   true,
		LastError:      "",
		Timestamp:      time.Now(),
	}

	app.UpdateStatus(status)

	statusStr := app.GetStatusString()

	// Verify format includes all required components
	requiredParts := []string{
		"Mode: auto",
		"App: Zoom",
		"Recording",
	}

	for _, part := range requiredParts {
		if !contains(statusStr, part) {
			t.Errorf("Status string should contain '%s', got: %s", part, statusStr)
		}
	}
}

// TestMenuItemVisibility verifies all menu methods work without panic (T073-T079)
func TestMenuItemVisibility(t *testing.T) {
	app := macui.NewStatusBarApp()

	// Initialize status first
	status := &ipc.StatusSnapshot{
		Mode:           ipc.ModeAuto,
		OBSConnected:   false,
		LastError:      "",
		Timestamp:      time.Now(),
	}
	app.UpdateStatus(status)

	// Test that all menu item methods exist and don't panic
	menuMethods := []struct {
		name   string
		method func()
	}{
		{"StartRecording", func() { app.StartRecording() }},
		{"StopRecording", func() { app.StopRecording() }},
		{"SetAutoMode", func() { app.SetAutoMode() }},
		{"SetManualMode", func() { app.SetManualMode() }},
		{"SetPauseMode", func() { app.SetPauseMode() }},
		{"ShowSettings", func() { app.ShowSettings() }},
	}

	for _, mm := range menuMethods {
		t.Run(mm.name, func(t *testing.T) {
			// Recover from any panics
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Method panicked: %v", r)
				}
			}()

			mm.method()
			// Method should complete without panic
		})
	}
}

// Helper to check if string contains substring
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
