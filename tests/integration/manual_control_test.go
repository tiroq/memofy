package integration

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/tiroq/memofy/internal/config"
	"github.com/tiroq/memofy/internal/detector"
	"github.com/tiroq/memofy/internal/ipc"
	"github.com/tiroq/memofy/internal/statemachine"
)

// TestManualStartNoMeeting tests starting recording when no meeting is detected
func TestManualStartNoMeeting(t *testing.T) {
	cfg := createTestConfig()
	sm := statemachine.NewStateMachine(cfg)

	// Verify not recording initially
	if sm.IsRecording() {
		t.Error("Expected not recording initially")
	}

	// Force start recording
	err := sm.ForceStart(detector.AppNone)
	if err != nil {
		t.Fatalf("ForceStart failed: %v", err)
	}

	// Verify recording started
	if !sm.IsRecording() {
		t.Error("Expected recording after ForceStart")
	}

	// Verify mode switched to paused (prevent auto-stop)
	if sm.CurrentMode() != ipc.ModePaused {
		t.Errorf("Expected mode to be paused after ForceStart, got %s", sm.CurrentMode())
	}
}

// TestManualStopDuringRecording tests stopping an active recording
func TestManualStopDuringRecording(t *testing.T) {
	cfg := createTestConfig()
	sm := statemachine.NewStateMachine(cfg)

	// Start recording
	err := sm.ForceStart(detector.AppNone)
	if err != nil {
		t.Fatalf("ForceStart failed: %v", err)
	}

	if !sm.IsRecording() {
		t.Fatal("Expected recording after ForceStart")
	}

	// Now stop
	err = sm.ForceStop()
	if err != nil {
		t.Fatalf("ForceStop failed: %v", err)
	}

	// Verify stopped
	if sm.IsRecording() {
		t.Error("Expected not recording after ForceStop")
	}
}

// TestModeSwitchingAutoManualAuto tests switching modes preserves behavior
func TestModeSwitchingAutoManualAuto(t *testing.T) {
	cfg := createTestConfig()
	sm := statemachine.NewStateMachine(cfg)

	// Start in auto mode
	if sm.CurrentMode() != ipc.ModeAuto {
		t.Errorf("Expected mode to start as auto, got %s", sm.CurrentMode())
	}

	// Simulate meeting detection does NOT start recording in paused mode
	sm.SetMode(ipc.ModePaused)
	detectionState := &detector.DetectionState{
		MeetingDetected: true,
		DetectedApp:     detector.AppZoom,
	}

	shouldStart, _, _ := sm.ProcessDetection(*detectionState)
	if shouldStart {
		t.Error("Expected no start action in paused mode")
	}

	// Switch back to auto mode
	sm.SetMode(ipc.ModeAuto)

	// Now with 3 detections, should start
	for i := 0; i < 3; i++ {
		shouldStart, shouldStop, _ := sm.ProcessDetection(*detectionState)
		if shouldStop {
			t.Errorf("Iteration %d: expected no stop action", i)
		}
		if i < 2 && shouldStart {
			t.Errorf("Iteration %d: expected no start before threshold", i)
		}
		if i == 2 && !shouldStart {
			t.Errorf("Iteration %d: expected start action at threshold", i)
		}
	}
}

// TestPauseModeDetectionDoesNotRun tests that detection is skipped in pause mode
func TestPauseModeDetectionDoesNotRun(t *testing.T) {
	cfg := createTestConfig()
	sm := statemachine.NewStateMachine(cfg)

	// Set to pause mode
	sm.SetMode(ipc.ModePaused)

	// Simulate 10 meeting detections
	detectionState := &detector.DetectionState{
		MeetingDetected: true,
		DetectedApp:     detector.AppZoom,
	}

	for i := 0; i < 10; i++ {
		shouldStart, shouldStop, _ := sm.ProcessDetection(*detectionState)
		if shouldStart || shouldStop {
			t.Errorf("Iteration %d: expected no actions in paused mode", i)
		}
	}

	// Verify recording was never triggered
	if sm.IsRecording() {
		t.Error("Expected not recording after paused detection")
	}

	// Now switch to auto mode and verify detection works
	sm.SetMode(ipc.ModeAuto)
	for i := 0; i < 3; i++ {
		shouldStart, _, _ := sm.ProcessDetection(*detectionState)
		if i == 2 && !shouldStart {
			t.Error("Expected start action after switching back to auto")
		}
	}
}

// TestCommandInterface tests the command file read/write interface (T063-T066)
func TestCommandInterface(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()
	cmdPath := filepath.Join(tmpDir, "cmd.txt")

	t.Run("WriteAndReadCommand", func(t *testing.T) {
		// Simulate writing a command
		testCmd := ipc.CmdStart
		data := []byte(string(testCmd))
		err := os.WriteFile(cmdPath, data, 0644)
		if err != nil {
			t.Fatalf("Failed to write command: %v", err)
		}

		// Read it back
		content, err := os.ReadFile(cmdPath)
		if err != nil {
			t.Fatalf("Failed to read command: %v", err)
		}

		if string(content) != string(testCmd) {
			t.Errorf("Expected %s, got %s", testCmd, string(content))
		}
	})

	t.Run("CommandModification", func(t *testing.T) {
		// Simulate command file being updated
		cmd1Path := filepath.Join(tmpDir, "cmd2.txt")
		os.WriteFile(cmd1Path, []byte("start"), 0644)

		// Check modification time
		info1, _ := os.Stat(cmd1Path)
		time.Sleep(100 * time.Millisecond)

		// Update command
		os.WriteFile(cmd1Path, []byte("stop"), 0644)
		info2, _ := os.Stat(cmd1Path)

		// Verify modification time changed
		if !info2.ModTime().After(info1.ModTime()) {
			t.Error("Expected modification time to increase")
		}
	})
}

// createTestConfig returns a minimal test configuration
func createTestConfig() *config.DetectionConfig {
	return &config.DetectionConfig{
		Rules: []config.DetectionRule{
			{
				Application:  "zoom",
				ProcessNames: []string{"zoom.us"},
				WindowHints:  []string{"Zoom Meeting"},
				Enabled:      true,
			},
			{
				Application:  "teams",
				ProcessNames: []string{"Microsoft Teams"},
				WindowHints:  []string{"Meeting"},
				Enabled:      true,
			},
		},
		PollInterval:   2,
		StartThreshold: 3,
		StopThreshold:  6,
	}
}
