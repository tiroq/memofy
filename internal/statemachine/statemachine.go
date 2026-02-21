package statemachine

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/tiroq/memofy/internal/config"
	"github.com/tiroq/memofy/internal/detector"
	"github.com/tiroq/memofy/internal/diaglog"
	"github.com/tiroq/memofy/internal/ipc"
)

// ── RecordingOrigin (T013) ────────────────────────────────────────────────────

// RecordingOrigin encodes who initiated (or requested to stop) a recording.
// Priority order (highest first): manual > auto > forced.
type RecordingOrigin string

const (
	OriginUnknown RecordingOrigin = ""
	OriginManual  RecordingOrigin = "manual" // user action via UI
	OriginAuto    RecordingOrigin = "auto"   // detection threshold crossed
	OriginForced  RecordingOrigin = "forced" // programmatic / test path
)

// priorityOf returns a numeric priority for origin comparisons.
func priorityOf(o RecordingOrigin) int {
	switch o {
	case OriginManual:
		return 2
	case OriginAuto:
		return 1
	case OriginForced:
		return 1 // same tier as auto (test & programmatic path)
	default:
		return 1
	}
}

// ── RecordingSession (T013) ───────────────────────────────────────────────────

// RecordingSession holds metadata about the active recording interval.
type RecordingSession struct {
	SessionID string          // 16-char hex, crypto/rand
	Origin    RecordingOrigin // who started it
	App       detector.DetectedApp
	StartedAt time.Time
}

// ── StopRequest (T013) ────────────────────────────────────────────────────────

// StopRequest carries full attribution for a recording stop signal (FR-003).
type StopRequest struct {
	RequestOrigin RecordingOrigin // who is requesting the stop
	Reason        string          // machine-readable reason code
	Component     string          // source component label
}

// ── StateMachine ─────────────────────────────────────────────────────────────

// StateMachine manages recording state with debounced detection.
type StateMachine struct {
	config          *config.DetectionConfig
	currentMode     ipc.OperatingMode
	recording       bool
	recordingApp    detector.DetectedApp
	recordingStart  time.Time
	detectionStreak int
	absenceStreak   int

	// Session authority fields (T014)
	session     *RecordingSession
	debounceDur time.Duration
	logger      *diaglog.Logger
}

// NewStateMachine creates a state machine with given config.
func NewStateMachine(cfg *config.DetectionConfig) *StateMachine {
	return &StateMachine{
		config:      cfg,
		currentMode: ipc.ModeAuto,
		recording:   false,
		debounceDur: 5 * time.Second,
	}
}

// SetLogger injects a diaglog.Logger for structured event tracing (T014).
func (sm *StateMachine) SetLogger(l *diaglog.Logger) {
	sm.logger = l
}

// SetDebounceDuration overrides the default 5-second session-start race guard (T014).
func (sm *StateMachine) SetDebounceDuration(d time.Duration) {
	sm.debounceDur = d
}

// newSessionID generates a 16-character hex session ID using crypto/rand.
func newSessionID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "0000000000000000"
	}
	return hex.EncodeToString(b)
}

// ── ProcessDetection ─────────────────────────────────────────────────────────

// ProcessDetection evaluates detection state and returns action to take.
// Returns: shouldStartRecording, shouldStopRecording, newApp
func (sm *StateMachine) ProcessDetection(state detector.DetectionState) (bool, bool, detector.DetectedApp) {
	if sm.currentMode == ipc.ModePaused {
		return false, false, detector.AppNone
	}

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
		sm.absenceStreak = 0
		sm.detectionStreak++

		if !sm.recording && sm.detectionStreak >= sm.config.StartThreshold {
			return true, false, state.DetectedApp
		}
		return false, false, detector.AppNone
	}

	sm.detectionStreak = 0
	sm.absenceStreak++

	if sm.recording && sm.absenceStreak >= sm.config.StopThreshold {
		return false, true, detector.AppNone
	}

	return false, false, detector.AppNone
}

// ── StartRecording / StopRecording / ForceStart / ForceStop ──────────────────

// StartRecording updates state for an auto-detected recording start (T014).
func (sm *StateMachine) StartRecording(app detector.DetectedApp) {
	sm.recording = true
	sm.recordingApp = app
	sm.recordingStart = time.Now()
	sm.detectionStreak = 0
	sm.absenceStreak = 0

	sid := newSessionID()
	sm.session = &RecordingSession{
		SessionID: sid,
		Origin:    OriginAuto,
		App:       app,
		StartedAt: sm.recordingStart,
	}
	if sm.logger != nil {
		sm.logger.Log(diaglog.LogEntry{
			Component: diaglog.ComponentStateMachine,
			Event:     diaglog.EventRecordingStart,
			SessionID: sid,
			Payload:   map[string]interface{}{"origin": "auto", "app": string(app)},
		})
	}
}

// StopRecording implements the authority hierarchy (T015 / FR-007 / FR-008).
// Returns true if the stop was executed, false if it was rejected.
func (sm *StateMachine) StopRecording(req StopRequest) bool {
	if sm.session != nil && sm.session.Origin == OriginManual {
		// Reject any stop from a lower-priority origin (FR-007).
		if priorityOf(req.RequestOrigin) < priorityOf(OriginManual) {
			if sm.logger != nil {
				sm.logger.Log(diaglog.LogEntry{
					Component: diaglog.ComponentStateMachine,
					Event:     diaglog.EventRecordingStopRejected,
					SessionID: sm.session.SessionID,
					Reason:    "manual_mode_override",
					Payload: map[string]interface{}{
						"requested_by": string(req.RequestOrigin),
						"component":    req.Component,
						"reason":       req.Reason,
					},
				})
			}
			return false
		}
	}

	// Debounce guard: reject auto-origin stops in the race window (FR-008).
	// Applies to all session origins; only manual-request stops bypass the guard.
	// NOTE: explicit user stops always succeed immediately.
	if sm.session != nil && req.RequestOrigin != OriginManual && time.Since(sm.session.StartedAt) < sm.debounceDur {
		if sm.logger != nil {
			sm.logger.Log(diaglog.LogEntry{
				Component: diaglog.ComponentStateMachine,
				Event:     diaglog.EventRecordingStopRejected,
				SessionID: sm.session.SessionID,
				Reason:    "debounce_guard",
				Payload: map[string]interface{}{
					"requested_by": string(req.RequestOrigin),
					"component":    req.Component,
					"reason":       "debounce_guard",
				},
			})
		}
		return false
	}

	// Stop allowed.
	sid := ""
	if sm.session != nil {
		sid = sm.session.SessionID
	}
	sm.recording = false
	sm.recordingApp = detector.AppNone
	sm.recordingStart = time.Time{}
	sm.detectionStreak = 0
	sm.absenceStreak = 0
	sm.session = nil

	if sm.logger != nil {
		sm.logger.Log(diaglog.LogEntry{
			Component: diaglog.ComponentStateMachine,
			Event:     diaglog.EventRecordingStop,
			SessionID: sid,
			Reason:    req.Reason,
			Payload:   map[string]interface{}{"requested_by": string(req.RequestOrigin), "component": req.Component},
		})
	}
	return true
}

// ForceStart manually starts recording (from command interface, T014).
func (sm *StateMachine) ForceStart(app detector.DetectedApp) error {
	if sm.recording {
		return fmt.Errorf("already recording")
	}
	now := time.Now()
	sid := newSessionID()
	sm.recording = true
	sm.recordingApp = app
	sm.recordingStart = now
	sm.detectionStreak = 0
	sm.absenceStreak = 0
	sm.session = &RecordingSession{
		SessionID: sid,
		Origin:    OriginManual,
		App:       app,
		StartedAt: now,
	}
	if sm.logger != nil {
		sm.logger.Log(diaglog.LogEntry{
			Component: diaglog.ComponentStateMachine,
			Event:     diaglog.EventRecordingStart,
			SessionID: sid,
			Payload:   map[string]interface{}{"origin": "manual", "app": string(app)},
		})
	}
	// In manual mode keep manual mode; otherwise switch to paused to prevent auto-stop.
	if sm.currentMode != ipc.ModeManual {
		sm.currentMode = ipc.ModePaused
	}
	return nil
}

// ForceStop manually stops recording (from command interface).
func (sm *StateMachine) ForceStop() error {
	if !sm.recording {
		return fmt.Errorf("not recording")
	}
	sm.StopRecording(StopRequest{
		RequestOrigin: OriginManual,
		Reason:        "user_stop",
		Component:     diaglog.ComponentMemofyCore,
	})
	return nil
}

// ── Mode management ───────────────────────────────────────────────────────────

// ToggleMode switches between auto and paused modes.
func (sm *StateMachine) ToggleMode() {
	if sm.currentMode == ipc.ModeAuto {
		sm.currentMode = ipc.ModePaused
	} else {
		sm.currentMode = ipc.ModeAuto
	}
}

// SetMode explicitly sets the operating mode.
func (sm *StateMachine) SetMode(mode ipc.OperatingMode) {
	sm.currentMode = mode
}

// ── Read-only accessors ───────────────────────────────────────────────────────

// IsRecording returns current recording status.
func (sm *StateMachine) IsRecording() bool {
	return sm.recording
}

// CurrentMode returns current operating mode.
func (sm *StateMachine) CurrentMode() ipc.OperatingMode {
	return sm.currentMode
}

// GetDetectionStreak returns current detection streak count.
func (sm *StateMachine) GetDetectionStreak() int {
	return sm.detectionStreak
}

// GetAbsenceStreak returns current absence streak count.
func (sm *StateMachine) GetAbsenceStreak() int {
	return sm.absenceStreak
}

// RecordingDuration returns how long current recording has been active.
func (sm *StateMachine) RecordingDuration() time.Duration {
	if !sm.recording {
		return 0
	}
	return time.Since(sm.recordingStart)
}

// RecordingApp returns the app being recorded.
func (sm *StateMachine) RecordingApp() detector.DetectedApp {
	return sm.recordingApp
}

// SessionOrigin returns the origin of the active recording session, or
// OriginUnknown if no session is active (T017).
func (sm *StateMachine) SessionOrigin() RecordingOrigin {
	if sm.session == nil {
		return OriginUnknown
	}
	return sm.session.Origin
}

// SessionID returns the session ID of the active recording, or empty string
// if no session is active (T017).
func (sm *StateMachine) SessionID() string {
	if sm.session == nil {
		return ""
	}
	return sm.session.SessionID
}
