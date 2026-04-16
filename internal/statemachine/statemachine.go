// Package statemachine implements the audio recording lifecycle FSM.
//
// States:
//
//	idle         → no audio activity detected
//	arming       → sound detected, waiting for activation window before recording
//	recording    → actively recording audio to file
//	silence_wait → silence detected during recording, waiting for threshold
//	finalizing   → finishing the current recording file
//	error        → a fatal error occurred
package statemachine

import (
	"fmt"
	"sync"
	"time"
)

// State represents the current phase of the recording lifecycle.
type State string

const (
	StateIdle        State = "idle"
	StateArming      State = "arming"
	StateRecording   State = "recording"
	StateSilenceWait State = "silence_wait"
	StateFinalizing  State = "finalizing"
	StateError       State = "error"
)

// Event represents an input signal to the state machine.
type Event int

const (
	EventSoundDetected Event = iota
	EventSilenceDetected
	EventSilenceThresholdExceeded
	EventRecordingStarted
	EventRecordingFinalized
)

// String returns a human-readable name for the event.
func (e Event) String() string {
	switch e {
	case EventSoundDetected:
		return "sound_detected"
	case EventSilenceDetected:
		return "silence_detected"
	case EventSilenceThresholdExceeded:
		return "silence_threshold_exceeded"
	case EventRecordingStarted:
		return "recording_started"
	case EventRecordingFinalized:
		return "recording_finalized"
	default:
		return fmt.Sprintf("unknown(%d)", int(e))
	}
}

// StateMachine manages the recording lifecycle based on audio activity.
type StateMachine struct {
	mu    sync.RWMutex
	state State

	// Timing
	silenceStart       time.Time
	recordingStart     time.Time
	armingStart        time.Time
	silenceDuration    time.Duration // configurable silence threshold
	activationDuration time.Duration // consecutive sound required to start recording

	// Callbacks
	onStateChange func(from, to State)
}

// New creates a state machine with the given silence threshold and activation window.
func New(silenceThreshold, activationDuration time.Duration) *StateMachine {
	return &StateMachine{
		state:              StateIdle,
		silenceDuration:    silenceThreshold,
		activationDuration: activationDuration,
	}
}

// SetOnStateChange sets a callback invoked on every state transition.
func (sm *StateMachine) SetOnStateChange(fn func(from, to State)) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.onStateChange = fn
}

// State returns the current state.
func (sm *StateMachine) CurrentState() State {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.state
}

// RecordingStart returns when the current recording started (zero if not recording).
func (sm *StateMachine) RecordingStart() time.Time {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.recordingStart
}

// SilenceElapsed returns how long silence has been detected during silence_wait.
func (sm *StateMachine) SilenceElapsed() time.Duration {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	if sm.state != StateSilenceWait || sm.silenceStart.IsZero() {
		return 0
	}
	return time.Since(sm.silenceStart)
}

// ProcessAudio is the main entry point. Call it with each audio buffer's RMS level.
// Returns the action the caller should take.
type Action int

const (
	ActionNone           Action = iota
	ActionStartRecording        // begin a new WAV file and start writing
	ActionStopRecording         // finalize current WAV file
	ActionContinue              // keep writing to current file
)

// String returns a human-readable name for the action.
func (a Action) String() string {
	switch a {
	case ActionNone:
		return "none"
	case ActionStartRecording:
		return "start_recording"
	case ActionStopRecording:
		return "stop_recording"
	case ActionContinue:
		return "continue"
	default:
		return fmt.Sprintf("unknown(%d)", int(a))
	}
}

// ProcessAudio evaluates the current RMS level and returns an action.
// threshold is the RMS level above which audio is considered "sound".
func (sm *StateMachine) ProcessAudio(rmsLevel float64, threshold float64) Action {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	hasSound := rmsLevel >= threshold

	switch sm.state {
	case StateIdle:
		if hasSound {
			sm.armingStart = time.Now()
			sm.transition(StateArming)
			return ActionNone
		}
		return ActionNone

	case StateArming:
		if !hasSound {
			sm.armingStart = time.Time{}
			sm.transition(StateIdle)
			return ActionNone
		}
		if time.Since(sm.armingStart) >= sm.activationDuration {
			sm.recordingStart = time.Now()
			sm.transition(StateRecording)
			return ActionStartRecording
		}
		return ActionNone

	case StateRecording:
		if hasSound {
			return ActionContinue
		}
		// Silence detected — enter silence_wait
		sm.silenceStart = time.Now()
		sm.transition(StateSilenceWait)
		return ActionContinue // keep recording during silence_wait

	case StateSilenceWait:
		if hasSound {
			// Sound resumed — back to recording
			sm.silenceStart = time.Time{}
			sm.transition(StateRecording)
			return ActionContinue
		}
		// Still silent — check threshold
		if time.Since(sm.silenceStart) >= sm.silenceDuration {
			sm.transition(StateFinalizing)
			return ActionStopRecording
		}
		return ActionContinue // keep recording during brief silence

	case StateFinalizing:
		// Recording is being finalized. Once done, caller should call Reset().
		return ActionNone

	case StateError:
		return ActionNone

	default:
		return ActionNone
	}
}

// Reset returns the state machine to idle. Call after finalizing a recording.
func (sm *StateMachine) Reset() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.transition(StateIdle)
	sm.recordingStart = time.Time{}
	sm.silenceStart = time.Time{}
	sm.armingStart = time.Time{}
}

// EnterError transitions to the error state.
func (sm *StateMachine) EnterError() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.transition(StateError)
}

// ForceStartRecording immediately transitions from idle or arming to recording.
// Returns ActionStartRecording if the transition was made, ActionNone otherwise.
// Used when an external signal (e.g. mic usage detected) should start recording
// regardless of the current audio RMS level.
func (sm *StateMachine) ForceStartRecording() Action {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if sm.state != StateIdle && sm.state != StateArming {
		return ActionNone
	}
	sm.recordingStart = time.Now()
	sm.transition(StateRecording)
	return ActionStartRecording
}

// transition changes state and fires the callback.
func (sm *StateMachine) transition(to State) {
	from := sm.state
	if from == to {
		return
	}
	sm.state = to
	if sm.onStateChange != nil {
		sm.onStateChange(from, to)
	}
}
