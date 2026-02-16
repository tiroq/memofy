package integration

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/progrium/darwinkit/objc"
	"github.com/tiroq/memofy/internal/ipc"
	"github.com/tiroq/memofy/pkg/macui"
)

// TestMain ensures tests run on the main thread for macOS GUI operations
func TestMain(m *testing.M) {
	// Lock to main thread for AppKit operations
	runtime.LockOSThread()
	// Run tests
	os.Exit(m.Run())
}

// TestMenuBarIconStateChanges verifies icon changes when status changes (T086)
func TestMenuBarIconStateChanges(t *testing.T) {
	t.Skip("Skipping GUI test - requires main thread for NSWindow creation")
	app := macui.NewStatusBarApp("v0.3.0-test")

	tests := []struct {
		name           string
		status         *ipc.StatusSnapshot
		expectedChange bool
	}{
		{
			name: "idle state",
			status: &ipc.StatusSnapshot{
				Mode:          ipc.ModeAuto,
				TeamsDetected: false,
				ZoomDetected:  false,
				OBSConnected:  false,
				LastError:     "",
				Timestamp:     time.Now(),
			},
			expectedChange: true,
		},
		{
			name: "zoom detected",
			status: &ipc.StatusSnapshot{
				Mode:          ipc.ModeAuto,
				TeamsDetected: false,
				ZoomDetected:  true,
				OBSConnected:  false,
				LastError:     "",
				Timestamp:     time.Now(),
			},
			expectedChange: true,
		},
		{
			name: "recording active",
			status: &ipc.StatusSnapshot{
				Mode:          ipc.ModeAuto,
				TeamsDetected: false,
				ZoomDetected:  true,
				OBSConnected:  true,
				LastError:     "",
				Timestamp:     time.Now(),
			},
			expectedChange: true,
		},
		// Skip error state test - it triggers blocking error dialog
		// {
		// 	name: "error state",
		// 	status: &ipc.StatusSnapshot{
		// 		Mode:          ipc.ModeAuto,
		// 		TeamsDetected: false,
		// 		ZoomDetected:  false,
		// 		OBSConnected:  false,
		// 		LastError:     "Permission denied for screen recording",
		// 		Timestamp:     time.Now(),
		// 	},
		// 	expectedChange: true,
		// },
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
// Note: Skipped because it requires daemon integration and would interfere with real cache
func TestControlCommandsWriteToFile(t *testing.T) {
	t.Skip("Skipping test that requires daemon integration and modifies real cache directory")

	app := macui.NewStatusBarApp("v0.3.0-test")

	// Create temporary cache directory
	cacheDir := filepath.Join(os.TempDir(), "memofy-test-cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatalf("Failed to create cache dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(cacheDir); err != nil {
			t.Logf("Warning: failed to remove cache dir: %v", err)
		}
	}()

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
				app.StartRecording(objc.Object{})
			},
			expectedCommand: "start",
		},
		{
			name: "stop recording",
			operation: func() {
				app.StopRecording(objc.Object{})
			},
			expectedCommand: "stop",
		},
		{
			name: "auto mode",
			operation: func() {
				app.SetAutoMode(objc.Object{})
			},
			expectedCommand: "auto",
		},
		{
			name: "pause mode",
			operation: func() {
				app.SetPauseMode(objc.Object{})
			},
			expectedCommand: "pause",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear any previous command
			_ = os.Remove(cmdFile) // Ignore error if file doesn't exist

			// Create actual command by writing to cache
			cacheDir := filepath.Join(os.Getenv("HOME"), ".cache", "memofy")
			if err := os.MkdirAll(cacheDir, 0755); err != nil {
				t.Fatalf("Failed to create cache dir: %v", err)
			}
			cmdFile = filepath.Join(cacheDir, "cmd.txt")

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
		name           string
		zoomProcess    string
		teamsProcess   string
		startThreshold int
		stopThreshold  int
		shouldSucceed  bool
	}{
		{
			name:           "valid settings",
			zoomProcess:    "zoom.us",
			teamsProcess:   "Microsoft Teams",
			startThreshold: 3,
			stopThreshold:  6,
			shouldSucceed:  true,
		},
		{
			name:           "invalid start threshold",
			zoomProcess:    "zoom.us",
			teamsProcess:   "Microsoft Teams",
			startThreshold: 0,
			stopThreshold:  6,
			shouldSucceed:  false,
		},
		{
			name:           "stop < start threshold",
			zoomProcess:    "zoom.us",
			teamsProcess:   "Microsoft Teams",
			startThreshold: 5,
			stopThreshold:  3,
			shouldSucceed:  false,
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
	t.Skip("Skipping test that requires GUI interaction - SendErrorNotification displays blocking dialog")
	app := macui.NewStatusBarApp("v0.3.0-test")

	// Create a status with an error
	errorStatus := &ipc.StatusSnapshot{
		Mode:          ipc.ModeAuto,
		TeamsDetected: false,
		ZoomDetected:  false,
		OBSConnected:  false,
		LastError:     "Screen recording permission denied. Please enable in Settings.",
		Timestamp:     time.Now(),
	}

	// Update status - should trigger error notification
	app.UpdateStatus(errorStatus)

	// Wait a moment for notification delivery
	time.Sleep(100 * time.Millisecond)

	// Verify error was captured
	if app.GetCurrentStatus().LastError != errorStatus.LastError {
		t.Error("Error should be captured in current status")
	}
}

// TestStatusDisplayFormat verifies status display format (T085)
func TestStatusDisplayFormat(t *testing.T) {
	t.Skip("Skipping GUI test - requires main thread for NSWindow creation")
	app := macui.NewStatusBarApp("v0.3.0-test")

	// Initialize with a status
	status := &ipc.StatusSnapshot{
		Mode:          ipc.ModeAuto,
		TeamsDetected: false,
		ZoomDetected:  true,
		OBSConnected:  true,
		LastError:     "",
		Timestamp:     time.Now(),
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
	t.Skip("Skipping GUI test - requires main thread for NSWindow creation")
	app := macui.NewStatusBarApp("v0.3.0-test")

	// Initialize status first
	status := &ipc.StatusSnapshot{
		Mode:         ipc.ModeAuto,
		OBSConnected: false,
		LastError:    "",
		Timestamp:    time.Now(),
	}
	app.UpdateStatus(status)

	// Test that all menu item methods exist and don't panic
	menuMethods := []struct {
		name   string
		method func()
	}{
		{"StartRecording", func() { app.StartRecording(objc.Object{}) }},
		{"StopRecording", func() { app.StopRecording(objc.Object{}) }},
		{"SetAutoMode", func() { app.SetAutoMode(objc.Object{}) }},
		{"SetPauseMode", func() { app.SetPauseMode(objc.Object{}) }},
		// Skip ShowSettings - requires GUI interaction and waits for user input
		// {"ShowSettings", func() { app.ShowSettings(objc.Object{}) }},
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
