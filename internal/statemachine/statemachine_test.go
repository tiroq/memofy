package statemachine

import (
	"testing"
	"time"

	"github.com/tiroq/memofy/internal/config"
	"github.com/tiroq/memofy/internal/detector"
	"github.com/tiroq/memofy/internal/ipc"
)

func TestProcessDetection_StartThreshold(t *testing.T) {
	tests := []struct {
		name              string
		startThreshold    int
		detectionSequence []bool // sequence of detection states
		wantStartAt       int    // index where recording should start (-1 if never)
	}{
		{
			name:              "starts at threshold 3",
			startThreshold:    3,
			detectionSequence: []bool{false, true, true, true, false},
			wantStartAt:       3, // 0-indexed: 4th item (3 consecutive detections)
		},
		{
			name:              "starts at threshold 1",
			startThreshold:    1,
			detectionSequence: []bool{false, true, false},
			wantStartAt:       1,
		},
		{
			name:              "interrupted streak resets",
			startThreshold:    3,
			detectionSequence: []bool{true, true, false, true, true, true},
			wantStartAt:       5, // Streak resets at index 2
		},
		{
			name:              "never reaches threshold",
			startThreshold:    5,
			detectionSequence: []bool{true, true, false, true, true},
			wantStartAt:       -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.DetectionConfig{
				StartThreshold: tt.startThreshold,
				StopThreshold:  6,
				PollInterval:   2,
			}
			sm := NewStateMachine(cfg)

			for i, detected := range tt.detectionSequence {
				state := detector.DetectionState{
					MeetingDetected: detected,
					DetectedApp:     detector.AppZoom,
				}

				shouldStart, shouldStop, _ := sm.ProcessDetection(state)

				if shouldStart {
					if tt.wantStartAt == -1 {
						t.Errorf("unexpected start at index %d", i)
					} else if i != tt.wantStartAt {
						t.Errorf("started at index %d, want %d", i, tt.wantStartAt)
					}
					sm.StartRecording(detector.AppZoom)
				}

				if shouldStop {
					t.Errorf("unexpected stop at index %d", i)
				}
			}

			if tt.wantStartAt != -1 && !sm.IsRecording() {
				t.Error("expected recording to have started, but it didn't")
			}
		})
	}
}

func TestProcessDetection_StopThreshold(t *testing.T) {
	tests := []struct {
		name              string
		stopThreshold     int
		detectionSequence []bool
		wantStopAt        int // index where recording should stop (-1 if never)
	}{
		{
			name:              "stops at threshold 6",
			stopThreshold:     6,
			detectionSequence: []bool{false, false, false, false, false, false},
			wantStopAt:        5, // 6 consecutive absences (indices 0-5)
		},
		{
			name:              "interrupted absence resets",
			stopThreshold:     6,
			detectionSequence: []bool{false, false, false, true, false, false, false, false, false, false},
			wantStopAt:        9,
		},
		{
			name:              "never reaches threshold",
			stopThreshold:     10,
			detectionSequence: []bool{false, false, false, true},
			wantStopAt:        -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.DetectionConfig{
				StartThreshold: 1,
				StopThreshold:  tt.stopThreshold,
				PollInterval:   2,
			}
			sm := NewStateMachine(cfg)

			// Start recording first
			sm.StartRecording(detector.AppZoom)

			for i, detected := range tt.detectionSequence {
				state := detector.DetectionState{
					MeetingDetected: detected,
					DetectedApp:     detector.AppZoom,
				}

				shouldStart, shouldStop, _ := sm.ProcessDetection(state)

				if shouldStart {
					t.Errorf("unexpected start at index %d", i)
				}

				if shouldStop {
					if tt.wantStopAt == -1 {
						t.Errorf("unexpected stop at index %d", i)
					} else if i != tt.wantStopAt {
						t.Errorf("stopped at index %d, want %d", i, tt.wantStopAt)
					}
					sm.StopRecording(StopRequest{RequestOrigin: OriginManual, Reason: "test_stop", Component: "test"})
				}
			}

			if tt.wantStopAt != -1 && sm.IsRecording() {
				t.Error("expected recording to have stopped, but it didn't")
			}
		})
	}
}

func TestProcessDetection_PausedMode(t *testing.T) {
	cfg := &config.DetectionConfig{
		StartThreshold: 3,
		StopThreshold:  6,
		PollInterval:   2,
	}
	sm := NewStateMachine(cfg)
	sm.SetMode(ipc.ModePaused)

	// Detection sequence that would normally trigger start
	for i := 0; i < 5; i++ {
		state := detector.DetectionState{
			MeetingDetected: true,
			DetectedApp:     detector.AppZoom,
		}

		shouldStart, shouldStop, _ := sm.ProcessDetection(state)

		if shouldStart || shouldStop {
			t.Errorf("paused mode should ignore detections, got start=%v stop=%v at iteration %d", shouldStart, shouldStop, i)
		}
	}

	if sm.IsRecording() {
		t.Error("recording should not start in paused mode")
	}
}

func TestForceStart(t *testing.T) {
	cfg := &config.DetectionConfig{
		StartThreshold: 3,
		StopThreshold:  6,
		PollInterval:   2,
	}
	sm := NewStateMachine(cfg)

	// Force start should work
	err := sm.ForceStart(detector.AppTeams)
	if err != nil {
		t.Errorf("ForceStart failed: %v", err)
	}

	if !sm.IsRecording() {
		t.Error("recording should be active after ForceStart")
	}

	if sm.RecordingApp() != detector.AppTeams {
		t.Errorf("recording app = %v, want %v", sm.RecordingApp(), detector.AppTeams)
	}

	if sm.CurrentMode() != ipc.ModePaused {
		t.Errorf("mode = %v, want %v (ForceStart should switch to paused)", sm.CurrentMode(), ipc.ModePaused)
	}

	// Second ForceStart should fail
	err = sm.ForceStart(detector.AppZoom)
	if err == nil {
		t.Error("ForceStart should fail when already recording")
	}
}

func TestForceStop(t *testing.T) {
	cfg := &config.DetectionConfig{
		StartThreshold: 3,
		StopThreshold:  6,
		PollInterval:   2,
	}
	sm := NewStateMachine(cfg)

	// ForceStop should fail when not recording
	err := sm.ForceStop()
	if err == nil {
		t.Error("ForceStop should fail when not recording")
	}

	// Start recording
	sm.StartRecording(detector.AppZoom)

	// ForceStop should work
	err = sm.ForceStop()
	if err != nil {
		t.Errorf("ForceStop failed: %v", err)
	}

	if sm.IsRecording() {
		t.Error("recording should be stopped after ForceStop")
	}
}

func TestToggleMode(t *testing.T) {
	cfg := &config.DetectionConfig{
		StartThreshold: 3,
		StopThreshold:  6,
		PollInterval:   2,
	}
	sm := NewStateMachine(cfg)

	// Initial mode is auto
	if sm.CurrentMode() != ipc.ModeAuto {
		t.Errorf("initial mode = %v, want %v", sm.CurrentMode(), ipc.ModeAuto)
	}

	// Toggle to paused
	sm.ToggleMode()
	if sm.CurrentMode() != ipc.ModePaused {
		t.Errorf("after toggle mode = %v, want %v", sm.CurrentMode(), ipc.ModePaused)
	}

	// Toggle back to auto
	sm.ToggleMode()
	if sm.CurrentMode() != ipc.ModeAuto {
		t.Errorf("after second toggle mode = %v, want %v", sm.CurrentMode(), ipc.ModeAuto)
	}
}

func TestRecordingDuration(t *testing.T) {
	cfg := &config.DetectionConfig{
		StartThreshold: 3,
		StopThreshold:  6,
		PollInterval:   2,
	}
	sm := NewStateMachine(cfg)

	// Duration should be 0 when not recording
	if d := sm.RecordingDuration(); d != 0 {
		t.Errorf("duration when not recording = %v, want 0", d)
	}

	// Start recording
	sm.StartRecording(detector.AppZoom)
	time.Sleep(100 * time.Millisecond)

	duration := sm.RecordingDuration()
	if duration < 100*time.Millisecond {
		t.Errorf("duration = %v, want >= 100ms", duration)
	}

	// Stop recording
	sm.StopRecording(StopRequest{RequestOrigin: OriginManual, Reason: "test_stop", Component: "test"})
	if d := sm.RecordingDuration(); d != 0 {
		t.Errorf("duration after stop = %v, want 0", d)
	}
}

func TestStreakTracking(t *testing.T) {
	cfg := &config.DetectionConfig{
		StartThreshold: 3,
		StopThreshold:  6,
		PollInterval:   2,
	}
	sm := NewStateMachine(cfg)

	// Detection streak should increment
	for i := 1; i <= 5; i++ {
		state := detector.DetectionState{
			MeetingDetected: true,
			DetectedApp:     detector.AppZoom,
		}
		sm.ProcessDetection(state)

		if sm.GetDetectionStreak() != i {
			t.Errorf("detection streak = %d, want %d", sm.GetDetectionStreak(), i)
		}
		if sm.GetAbsenceStreak() != 0 {
			t.Errorf("absence streak = %d, want 0", sm.GetAbsenceStreak())
		}
	}

	// Start recording to reset streaks
	sm.StartRecording(detector.AppZoom)

	if sm.GetDetectionStreak() != 0 {
		t.Errorf("detection streak after start = %d, want 0", sm.GetDetectionStreak())
	}

	// Absence streak should increment
	for i := 1; i <= 5; i++ {
		state := detector.DetectionState{
			MeetingDetected: false,
			DetectedApp:     detector.AppNone,
		}
		sm.ProcessDetection(state)

		if sm.GetAbsenceStreak() != i {
			t.Errorf("absence streak = %d, want %d", sm.GetAbsenceStreak(), i)
		}
		if sm.GetDetectionStreak() != 0 {
			t.Errorf("detection streak = %d, want 0", sm.GetDetectionStreak())
		}
	}
}

// ── T018: Session authority & debounce tests ──────────────────────────────────

// TestManualSessionBlocksAutoStop verifies FR-007: a Manual-origin session
// cannot be stopped by an Auto-origin request.
func TestManualSessionBlocksAutoStop(t *testing.T) {
	cfg := &config.DetectionConfig{StartThreshold: 1, StopThreshold: 1}
	sm := NewStateMachine(cfg)
	sm.SetDebounceDuration(0) // disable debounce for this test

	if err := sm.ForceStart(detector.AppZoom); err != nil {
		t.Fatalf("ForceStart failed: %v", err)
	}

	stopped := sm.StopRecording(StopRequest{
		RequestOrigin: OriginAuto,
		Reason:        "auto_detection_stop",
		Component:     "auto-detector",
	})

	if stopped {
		t.Error("expected auto stop to be rejected for manual session, but it succeeded")
	}
	if !sm.IsRecording() {
		t.Error("expected recording to still be active after rejected auto stop")
	}
}

// TestManualSessionAllowsUserStop verifies that a Manual-origin stop request
// succeeds even on a Manual-origin session.
func TestManualSessionAllowsUserStop(t *testing.T) {
	cfg := &config.DetectionConfig{StartThreshold: 1, StopThreshold: 1}
	sm := NewStateMachine(cfg)
	sm.SetDebounceDuration(0) // disable debounce

	if err := sm.ForceStart(detector.AppZoom); err != nil {
		t.Fatalf("ForceStart failed: %v", err)
	}

	stopped := sm.StopRecording(StopRequest{
		RequestOrigin: OriginManual,
		Reason:        "user_stop",
		Component:     "memofy-core",
	})

	if !stopped {
		t.Error("expected manual stop to succeed on manual session")
	}
	if sm.IsRecording() {
		t.Error("expected recording to have stopped")
	}
}

// TestDebounceRejectsAutoStopEarly verifies FR-008: auto-origin stops within
// the debounce window are rejected.
func TestDebounceRejectsAutoStopEarly(t *testing.T) {
	cfg := &config.DetectionConfig{StartThreshold: 1, StopThreshold: 1}
	sm := NewStateMachine(cfg)
	sm.SetDebounceDuration(30 * time.Second) // very long debounce

	sm.StartRecording(detector.AppZoom)

	stopped := sm.StopRecording(StopRequest{
		RequestOrigin: OriginAuto,
		Reason:        "auto_detection_stop",
		Component:     "auto-detector",
	})

	if stopped {
		t.Error("expected auto stop within debounce window to be rejected")
	}
	if !sm.IsRecording() {
		t.Error("expected recording to remain active after debounce rejection")
	}
}

// TestDebounceAllowsUserStopEarly verifies that Manual-origin stops bypass
// the debounce guard (FR-008: "does NOT block manual-origin stops").
func TestDebounceAllowsUserStopEarly(t *testing.T) {
	cfg := &config.DetectionConfig{StartThreshold: 1, StopThreshold: 1}
	sm := NewStateMachine(cfg)
	sm.SetDebounceDuration(30 * time.Second) // very long debounce

	sm.StartRecording(detector.AppZoom)

	stopped := sm.StopRecording(StopRequest{
		RequestOrigin: OriginManual,
		Reason:        "user_stop",
		Component:     "memofy-core",
	})

	if !stopped {
		t.Error("expected manual stop to bypass debounce guard")
	}
	if sm.IsRecording() {
		t.Error("expected recording to have stopped after manual stop")
	}
}

// TestAutoSessionAllowsAutoStop verifies that Auto-origin stops succeed on an
// Auto-origin session once the debounce window has elapsed.
func TestAutoSessionAllowsAutoStop(t *testing.T) {
	cfg := &config.DetectionConfig{StartThreshold: 1, StopThreshold: 1}
	sm := NewStateMachine(cfg)
	sm.SetDebounceDuration(0) // no debounce

	sm.StartRecording(detector.AppZoom)

	stopped := sm.StopRecording(StopRequest{
		RequestOrigin: OriginAuto,
		Reason:        "auto_detection_stop",
		Component:     "auto-detector",
	})

	if !stopped {
		t.Error("expected auto stop to succeed on auto session with zero debounce")
	}
	if sm.IsRecording() {
		t.Error("expected recording to have stopped")
	}
}

// TestSessionIDGeneratedOnStart verifies that a non-empty session ID is
// assigned when ForceStart is called.
func TestSessionIDGeneratedOnStart(t *testing.T) {
	cfg := &config.DetectionConfig{StartThreshold: 1, StopThreshold: 1}
	sm := NewStateMachine(cfg)

	if err := sm.ForceStart(detector.AppZoom); err != nil {
		t.Fatalf("ForceStart failed: %v", err)
	}

	sid := sm.SessionID()
	if sid == "" {
		t.Error("expected non-empty session ID after ForceStart, got empty string")
	}
	if len(sid) != 16 {
		t.Errorf("expected 16-character hex session ID, got %q (len %d)", sid, len(sid))
	}

	origin := sm.SessionOrigin()
	if origin != OriginManual {
		t.Errorf("expected session origin %q, got %q", OriginManual, origin)
	}
}
