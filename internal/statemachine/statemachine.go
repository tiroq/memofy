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

	// Threshold hysteresis: enterThreshold is used when idle/arming (higher),
	// exitThreshold is used when recording/silence_wait (lower). This prevents
	// flapping between recording and idle on borderline signal levels.
	enterThreshold float64
	exitThreshold  float64

	// Mic session lock: keeps the current recording alive while mic is in use.
	micSessionLock  bool          // feature enabled flag
	micLockActive   bool          // lock is currently in effect
	micReleaseSince time.Time     // non-zero: release debounce timer is running
	micReleaseDur   time.Duration // how long to hold after mic goes inactive

	// Callbacks
	onStateChange func(from, to State)
	logFn         func(string, ...any)
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

// SetThresholds configures the hysteresis thresholds. enterThreshold is used
// to detect sound when idle/arming, exitThreshold is used to detect silence
// when recording/silence_wait. exitThreshold must be <= enterThreshold.
func (sm *StateMachine) SetThresholds(enter, exit float64) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.enterThreshold = enter
	sm.exitThreshold = exit
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
// When hysteresis thresholds are configured via SetThresholds, the threshold
// parameter is ignored and the enter/exit pair is used instead.
func (sm *StateMachine) ProcessAudio(rmsLevel float64, threshold float64) Action {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.resolveMicRelease()

	// Determine effective thresholds. If hysteresis is configured, use
	// enter/exit thresholds based on current state. Otherwise, use the
	// single threshold for both entry and exit.
	enterTh := threshold
	exitTh := threshold
	if sm.enterThreshold > 0 {
		enterTh = sm.enterThreshold
		exitTh = sm.exitThreshold
		if exitTh <= 0 {
			exitTh = enterTh
		}
	}

	// Use enter threshold when deciding to start recording (idle/arming),
	// exit threshold when deciding to stop (recording/silence_wait).
	var hasSound bool
	switch sm.state {
	case StateIdle, StateArming:
		hasSound = rmsLevel >= enterTh
	default:
		hasSound = rmsLevel >= exitTh
	}

	switch sm.state {
	case StateIdle:
		if hasSound {
			sm.armingStart = time.Now()
			sm.transition(StateArming)
			sm.logf("state=arming reason=blackhole_active")
			return ActionNone
		}
		return ActionNone

	case StateArming:
		if !hasSound {
			sm.armingStart = time.Time{}
			sm.transition(StateIdle)
			sm.logf("state=idle reason=arming_cancelled_blackhole_silent")
			return ActionNone
		}
		if time.Since(sm.armingStart) >= sm.activationDuration {
			sm.recordingStart = time.Now()
			sm.transition(StateRecording)
			sm.logf("state=recording action=start_recording reason=activation_window_met")
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
		sm.logf("state=silence_wait reason=blackhole_inactive")
		return ActionContinue // keep recording during silence_wait

	case StateSilenceWait:
		if hasSound {
			// Sound resumed — back to recording
			sm.silenceStart = time.Time{}
			sm.transition(StateRecording)
			sm.logf("state=recording reason=blackhole_active_resumed")
			return ActionContinue
		}
		// Mic lock holds the session open regardless of silence duration.
		if sm.micLockActive {
			sm.logf("state=silence_wait mic_lock=true silence=%s", time.Since(sm.silenceStart).Truncate(time.Second))
			return ActionContinue
		}
		// Still silent — check threshold
		if time.Since(sm.silenceStart) >= sm.silenceDuration {
			sm.logf("action=stop_recording reason=silence_timeout_no_mic_lock silence=%s", time.Since(sm.silenceStart).Truncate(time.Second))
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
	sm.micLockActive = false
	sm.micReleaseSince = time.Time{}
}

// EnterError transitions to the error state.
func (sm *StateMachine) EnterError() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.transition(StateError)
}

// SetMicSessionLock enables or disables the microphone session lock feature.
// When enabled, an active microphone prevents the recording from being split
// even if BlackHole has been silent for longer than SilenceSplitSeconds.
// releaseDur is the debounce period: the lock stays active for this long after
// mic goes inactive, preventing split jitter from brief mic toggles.
func (sm *StateMachine) SetMicSessionLock(enabled bool, releaseDur time.Duration) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.micSessionLock = enabled
	sm.micReleaseDur = releaseDur
}

// SetLogger sets a printf-style logging function for state machine transitions.
// The engine wires this to its own logger so all SM events appear in the log.
func (sm *StateMachine) SetLogger(fn func(string, ...any)) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.logFn = fn
}

// SetMicActive updates microphone activity and manages the session lock.
// Call this whenever the monitor detects a mic-usage change.
//
//   - active=true  → immediately activates (or re-activates) the session lock.
//   - active=false → starts the release debounce timer; the lock stays active
//     until MicReleaseSeconds expires or mic becomes active again.
//
// Microphone activity alone never starts recording — it only keeps an already
// running session alive through BlackHole-silent pauses.
func (sm *StateMachine) SetMicActive(active bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if !sm.micSessionLock {
		return
	}
	if active {
		sm.micReleaseSince = time.Time{} // cancel any pending release
		if !sm.micLockActive {
			sm.micLockActive = true
			sm.logf("mic_lock=true reason=mic_active")
		}
	} else {
		if sm.micLockActive && sm.micReleaseSince.IsZero() {
			sm.micReleaseSince = time.Now()
			sm.logf("mic_lock=pending reason=mic_inactive release_debounce=%s", sm.micReleaseDur)
		}
	}
}

// MicLockActive returns whether the microphone session lock is currently
// holding the recording session open.
func (sm *StateMachine) MicLockActive() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.micLockActive
}

// resolveMicRelease checks whether the mic release debounce timer has expired
// and clears the session lock if so. Must be called with sm.mu write-lock held.
func (sm *StateMachine) resolveMicRelease() {
	if !sm.micLockActive || sm.micReleaseSince.IsZero() {
		return
	}
	if time.Since(sm.micReleaseSince) >= sm.micReleaseDur {
		sm.micLockActive = false
		sm.micReleaseSince = time.Time{}
		sm.logf("mic_lock=false reason=release_debounce_expired")
	}
}

// logf emits a message via the configured log function, if set.
func (sm *StateMachine) logf(format string, args ...any) {
	if sm.logFn != nil {
		sm.logFn("[sm] "+format, args...)
	}
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
