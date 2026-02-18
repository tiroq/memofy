package statemachine

import (
	"fmt"
	"time"

	"github.com/tiroq/memofy/internal/config"
	"github.com/tiroq/memofy/internal/detector"
	"github.com/tiroq/memofy/internal/ipc"
)

// StateMachine manages recording state with debounced detection
type StateMachine struct {
	config          *config.DetectionConfig
	currentMode     ipc.OperatingMode
	recording       bool
	recordingApp    detector.DetectedApp
	recordingStart  time.Time
	detectionStreak int // Consecutive detections
	absenceStreak   int // Consecutive non-detections
}

// NewStateMachine creates a state machine with given config
func NewStateMachine(cfg *config.DetectionConfig) *StateMachine {
	return &StateMachine{
		config:      cfg,
		currentMode: ipc.ModeAuto, // Start in auto mode
		recording:   false,
	}
}

// ProcessDetection evaluates detection state and returns action to take
// Returns: shouldStartRecording, shouldStopRecording, newApp
func (sm *StateMachine) ProcessDetection(state detector.DetectionState) (bool, bool, detector.DetectedApp) {
	// Paused mode: bypass detection entirely
	if sm.currentMode == ipc.ModePaused {
		return false, false, detector.AppNone
	}

	// Manual mode: update streaks for UI display but never auto-control OBS
	if sm.currentMode == ipc.ModeManual {
		if state.MeetingDetected {
			sm.absenceStreak = 0
			sm.detectionStreak++
		} else {
			sm.detectionStreak = 0
			sm.absenceStreak++
		}
		return false, false, detector.AppNone
	}

	if state.MeetingDetected {
		// Reset absence streak, increment detection streak
		sm.absenceStreak = 0
		sm.detectionStreak++

		// Check for start threshold
		if !sm.recording && sm.detectionStreak >= sm.config.StartThreshold {
			// Start recording
			return true, false, state.DetectedApp
		}

		// Already recording - continue
		return false, false, detector.AppNone
	}

	// No meeting detected
	sm.detectionStreak = 0
	sm.absenceStreak++

	// Check for stop threshold
	if sm.recording && sm.absenceStreak >= sm.config.StopThreshold {
		// Stop recording
		return false, true, detector.AppNone
	}

	return false, false, detector.AppNone
}

// StartRecording updates state to reflect recording started
func (sm *StateMachine) StartRecording(app detector.DetectedApp) {
	sm.recording = true
	sm.recordingApp = app
	sm.recordingStart = time.Now()
	sm.detectionStreak = 0 // Reset streaks
	sm.absenceStreak = 0
}

// StopRecording updates state to reflect recording stopped
func (sm *StateMachine) StopRecording() {
	sm.recording = false
	sm.recordingApp = detector.AppNone
	sm.recordingStart = time.Time{}
	sm.detectionStreak = 0
	sm.absenceStreak = 0
}

// ForceStart manually starts recording (from command interface)
func (sm *StateMachine) ForceStart(app detector.DetectedApp) error {
	if sm.recording {
		return fmt.Errorf("already recording")
	}
	sm.StartRecording(app)
	// In manual mode keep manual mode, otherwise switch to paused to prevent auto-stop
	if sm.currentMode != ipc.ModeManual {
		sm.currentMode = ipc.ModePaused
	}
	return nil
}

// ForceStop manually stops recording (from command interface)
func (sm *StateMachine) ForceStop() error {
	if !sm.recording {
		return fmt.Errorf("not recording")
	}
	sm.StopRecording()
	return nil
}

// ToggleMode switches between auto and paused modes
func (sm *StateMachine) ToggleMode() {
	if sm.currentMode == ipc.ModeAuto {
		sm.currentMode = ipc.ModePaused
	} else {
		sm.currentMode = ipc.ModeAuto
	}
}

// SetMode explicitly sets the operating mode
func (sm *StateMachine) SetMode(mode ipc.OperatingMode) {
	sm.currentMode = mode
}

// IsRecording returns current recording status
func (sm *StateMachine) IsRecording() bool {
	return sm.recording
}

// CurrentMode returns current operating mode
func (sm *StateMachine) CurrentMode() ipc.OperatingMode {
	return sm.currentMode
}

// GetDetectionStreak returns current detection streak count
func (sm *StateMachine) GetDetectionStreak() int {
	return sm.detectionStreak
}

// GetAbsenceStreak returns current absence streak count
func (sm *StateMachine) GetAbsenceStreak() int {
	return sm.absenceStreak
}

// RecordingDuration returns how long current recording has been active
func (sm *StateMachine) RecordingDuration() time.Duration {
	if !sm.recording {
		return 0
	}
	return time.Since(sm.recordingStart)
}

// RecordingApp returns the app being recorded
func (sm *StateMachine) RecordingApp() detector.DetectedApp {
	return sm.recordingApp
}
