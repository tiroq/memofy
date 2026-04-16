package statemachine

import (
	"testing"
	"time"
)

func TestNewStartsIdle(t *testing.T) {
	sm := New(60*time.Second, 0)
	if sm.CurrentState() != StateIdle {
		t.Errorf("initial state: got %s, want %s", sm.CurrentState(), StateIdle)
	}
}

func TestSoundTriggersRecording(t *testing.T) {
	sm := New(60*time.Second, 0)

	// Sound detected → arming (no recording yet)
	action := sm.ProcessAudio(0.05, 0.02)
	if action != ActionNone {
		t.Errorf("first sound: got %s, want %s", action, ActionNone)
	}
	if sm.CurrentState() != StateArming {
		t.Errorf("state after first sound: got %s, want %s", sm.CurrentState(), StateArming)
	}

	// Continue with sound → activation window passed (0ms) → start recording
	action = sm.ProcessAudio(0.05, 0.02)
	if action != ActionStartRecording {
		t.Errorf("second sound: got %s, want %s", action, ActionStartRecording)
	}
	if sm.CurrentState() != StateRecording {
		t.Errorf("state during recording: got %s, want %s", sm.CurrentState(), StateRecording)
	}

	// Continue with sound → continue recording
	action = sm.ProcessAudio(0.05, 0.02)
	if action != ActionContinue {
		t.Errorf("continued sound: got %s, want %s", action, ActionContinue)
	}
}

func TestSilenceEntersSilenceWait(t *testing.T) {
	sm := New(60*time.Second, 0)

	// Start recording (arming + start + continue)
	sm.ProcessAudio(0.05, 0.02)
	sm.ProcessAudio(0.05, 0.02)
	sm.ProcessAudio(0.05, 0.02)

	// Silence detected
	action := sm.ProcessAudio(0.001, 0.02)
	if action != ActionContinue {
		t.Errorf("silence during recording: got %s, want %s", action, ActionContinue)
	}
	if sm.CurrentState() != StateSilenceWait {
		t.Errorf("state after silence: got %s, want %s", sm.CurrentState(), StateSilenceWait)
	}
}

func TestSoundResumesDuringContinue(t *testing.T) {
	sm := New(60*time.Second, 0)

	// Start recording
	sm.ProcessAudio(0.05, 0.02)
	sm.ProcessAudio(0.05, 0.02)
	sm.ProcessAudio(0.05, 0.02)

	// Brief silence
	sm.ProcessAudio(0.001, 0.02)
	if sm.CurrentState() != StateSilenceWait {
		t.Fatalf("expected silence_wait, got %s", sm.CurrentState())
	}

	// Sound resumes
	action := sm.ProcessAudio(0.05, 0.02)
	if action != ActionContinue {
		t.Errorf("sound resume: got %s, want %s", action, ActionContinue)
	}
	if sm.CurrentState() != StateRecording {
		t.Errorf("state after resume: got %s, want %s", sm.CurrentState(), StateRecording)
	}
}

func TestSilenceThresholdStopsRecording(t *testing.T) {
	sm := New(10*time.Millisecond, 0) // short threshold for testing

	// Start recording
	sm.ProcessAudio(0.05, 0.02)
	sm.ProcessAudio(0.05, 0.02)
	sm.ProcessAudio(0.05, 0.02)

	// Enter silence_wait
	sm.ProcessAudio(0.001, 0.02)

	// Wait for threshold
	time.Sleep(15 * time.Millisecond)

	action := sm.ProcessAudio(0.001, 0.02)
	if action != ActionStopRecording {
		t.Errorf("after threshold: got %s, want %s", action, ActionStopRecording)
	}
	if sm.CurrentState() != StateFinalizing {
		t.Errorf("state after stop: got %s, want %s", sm.CurrentState(), StateFinalizing)
	}
}

func TestReset(t *testing.T) {
	sm := New(10*time.Millisecond, 0)

	// Go through full cycle
	sm.ProcessAudio(0.05, 0.02)
	sm.ProcessAudio(0.05, 0.02)
	sm.ProcessAudio(0.05, 0.02)
	sm.ProcessAudio(0.001, 0.02)
	time.Sleep(15 * time.Millisecond)
	sm.ProcessAudio(0.001, 0.02)

	sm.Reset()
	if sm.CurrentState() != StateIdle {
		t.Errorf("after reset: got %s, want %s", sm.CurrentState(), StateIdle)
	}
}

func TestStateChangeCallback(t *testing.T) {
	sm := New(60*time.Second, 0)

	var transitions []string
	sm.SetOnStateChange(func(from, to State) {
		transitions = append(transitions, string(from)+"→"+string(to))
	})

	sm.ProcessAudio(0.05, 0.02)  // idle → arming
	sm.ProcessAudio(0.05, 0.02)  // arming → recording
	sm.ProcessAudio(0.05, 0.02)  // recording (continue)
	sm.ProcessAudio(0.001, 0.02) // recording → silence_wait

	if len(transitions) != 3 {
		t.Fatalf("expected 3 transitions, got %d: %v", len(transitions), transitions)
	}
}

func TestIdleSilenceNoAction(t *testing.T) {
	sm := New(60*time.Second, 0)
	action := sm.ProcessAudio(0.001, 0.02)
	if action != ActionNone {
		t.Errorf("silence in idle: got %s, want %s", action, ActionNone)
	}
	if sm.CurrentState() != StateIdle {
		t.Errorf("state: got %s, want %s", sm.CurrentState(), StateIdle)
	}
}

func TestActivationWindow(t *testing.T) {
	sm := New(60*time.Second, 50*time.Millisecond)

	// Sound detected → arming
	action := sm.ProcessAudio(0.05, 0.02)
	if action != ActionNone || sm.CurrentState() != StateArming {
		t.Fatalf("expected arming, got state=%s action=%s", sm.CurrentState(), action)
	}

	// Sound continues but activation window not elapsed
	action = sm.ProcessAudio(0.05, 0.02)
	if action != ActionNone || sm.CurrentState() != StateArming {
		t.Fatalf("expected still arming, got state=%s action=%s", sm.CurrentState(), action)
	}

	// Wait for activation window
	time.Sleep(55 * time.Millisecond)

	// Now sound should trigger recording
	action = sm.ProcessAudio(0.05, 0.02)
	if action != ActionStartRecording {
		t.Errorf("after activation window: got %s, want %s", action, ActionStartRecording)
	}
	if sm.CurrentState() != StateRecording {
		t.Errorf("state: got %s, want %s", sm.CurrentState(), StateRecording)
	}
}

func TestArmingCancelledBySilence(t *testing.T) {
	sm := New(60*time.Second, 100*time.Millisecond)

	// Sound detected → arming
	sm.ProcessAudio(0.05, 0.02)
	if sm.CurrentState() != StateArming {
		t.Fatalf("expected arming, got %s", sm.CurrentState())
	}

	// Silence during arming → back to idle
	action := sm.ProcessAudio(0.001, 0.02)
	if action != ActionNone {
		t.Errorf("arming cancelled: got %s, want %s", action, ActionNone)
	}
	if sm.CurrentState() != StateIdle {
		t.Errorf("state: got %s, want %s", sm.CurrentState(), StateIdle)
	}
}

// --- Threshold hysteresis tests ---

func TestHysteresis_EnterThresholdRequiredToStart(t *testing.T) {
	sm := New(60*time.Second, 0)
	sm.SetThresholds(0.05, 0.01)

	// RMS between exit and enter thresholds — should NOT trigger arming.
	action := sm.ProcessAudio(0.03, 0.02)
	if action != ActionNone {
		t.Errorf("below enter threshold: got %s, want %s", action, ActionNone)
	}
	if sm.CurrentState() != StateIdle {
		t.Errorf("state: got %s, want %s", sm.CurrentState(), StateIdle)
	}

	// RMS at enter threshold — should trigger arming.
	action = sm.ProcessAudio(0.05, 0.02)
	if sm.CurrentState() != StateArming {
		t.Errorf("at enter threshold: got %s, want %s", sm.CurrentState(), StateArming)
	}

	// Continue above enter → start recording.
	action = sm.ProcessAudio(0.05, 0.02)
	if action != ActionStartRecording {
		t.Errorf("second buffer: got %s, want %s", action, ActionStartRecording)
	}
}

func TestHysteresis_ExitThresholdUsedDuringRecording(t *testing.T) {
	sm := New(60*time.Second, 0)
	sm.SetThresholds(0.05, 0.01)

	// Start recording.
	sm.ProcessAudio(0.05, 0.02)
	sm.ProcessAudio(0.05, 0.02)
	if sm.CurrentState() != StateRecording {
		t.Fatalf("expected recording, got %s", sm.CurrentState())
	}

	// RMS between exit (0.01) and enter (0.05) — should CONTINUE recording
	// because exit threshold is the one used during recording.
	action := sm.ProcessAudio(0.03, 0.02)
	if action != ActionContinue {
		t.Errorf("between thresholds: got %s, want %s", action, ActionContinue)
	}
	if sm.CurrentState() != StateRecording {
		t.Errorf("state: got %s, want %s", sm.CurrentState(), StateRecording)
	}

	// RMS below exit threshold → silence_wait.
	action = sm.ProcessAudio(0.005, 0.02)
	if action != ActionContinue {
		t.Errorf("below exit: got %s, want %s", action, ActionContinue)
	}
	if sm.CurrentState() != StateSilenceWait {
		t.Errorf("state: got %s, want %s", sm.CurrentState(), StateSilenceWait)
	}
}

func TestHysteresis_SilenceWaitUsesExitThreshold(t *testing.T) {
	sm := New(60*time.Second, 0)
	sm.SetThresholds(0.05, 0.01)

	// Start recording and enter silence_wait.
	sm.ProcessAudio(0.05, 0.02)
	sm.ProcessAudio(0.05, 0.02)
	sm.ProcessAudio(0.005, 0.02) // → silence_wait
	if sm.CurrentState() != StateSilenceWait {
		t.Fatalf("expected silence_wait, got %s", sm.CurrentState())
	}

	// RMS between exit and enter — should resume recording because
	// silence_wait uses exit threshold (0.01), and 0.015 >= 0.01.
	action := sm.ProcessAudio(0.015, 0.02)
	if action != ActionContinue {
		t.Errorf("resume from silence: got %s, want %s", action, ActionContinue)
	}
	if sm.CurrentState() != StateRecording {
		t.Errorf("state: got %s, want %s", sm.CurrentState(), StateRecording)
	}
}

func TestHysteresis_FallbackToSingleThreshold(t *testing.T) {
	sm := New(60*time.Second, 0)
	// No SetThresholds call — should use threshold param as both enter and exit.

	// 0.015 < 0.02 threshold → no arming.
	action := sm.ProcessAudio(0.015, 0.02)
	if action != ActionNone {
		t.Errorf("below threshold: got %s, want %s", action, ActionNone)
	}
	if sm.CurrentState() != StateIdle {
		t.Errorf("state: got %s, want %s", sm.CurrentState(), StateIdle)
	}

	// 0.02 >= 0.02 → arming.
	sm.ProcessAudio(0.02, 0.02)
	if sm.CurrentState() != StateArming {
		t.Errorf("at threshold: got %s, want %s", sm.CurrentState(), StateArming)
	}
}

// --- ForceStartRecording tests ---

func TestForceStartRecording_FromIdle(t *testing.T) {
	sm := New(60*time.Second, 500*time.Millisecond)

	action := sm.ForceStartRecording()
	if action != ActionStartRecording {
		t.Errorf("from idle: got %s, want %s", action, ActionStartRecording)
	}
	if sm.CurrentState() != StateRecording {
		t.Errorf("state after force: got %s, want %s", sm.CurrentState(), StateRecording)
	}
}

func TestForceStartRecording_FromArming(t *testing.T) {
	// activation window is long so the machine stays in arming
	sm := New(60*time.Second, 10*time.Second)
	sm.ProcessAudio(0.05, 0.02) // idle → arming
	if sm.CurrentState() != StateArming {
		t.Fatalf("expected arming, got %s", sm.CurrentState())
	}

	action := sm.ForceStartRecording()
	if action != ActionStartRecording {
		t.Errorf("from arming: got %s, want %s", action, ActionStartRecording)
	}
	if sm.CurrentState() != StateRecording {
		t.Errorf("state after force from arming: got %s, want %s", sm.CurrentState(), StateRecording)
	}
}

func TestForceStartRecording_AlreadyRecording(t *testing.T) {
	sm := New(60*time.Second, 0)
	sm.ProcessAudio(0.05, 0.02) // idle → arming
	sm.ProcessAudio(0.05, 0.02) // arming → recording

	action := sm.ForceStartRecording()
	if action != ActionNone {
		t.Errorf("already recording: got %s, want %s", action, ActionNone)
	}
	if sm.CurrentState() != StateRecording {
		t.Errorf("state unchanged: got %s, want %s", sm.CurrentState(), StateRecording)
	}
}

func TestForceStartRecording_DuringSilenceWait(t *testing.T) {
	sm := New(60*time.Second, 0)
	sm.ProcessAudio(0.05, 0.02)  // idle → arming
	sm.ProcessAudio(0.05, 0.02)  // arming → recording
	sm.ProcessAudio(0.001, 0.02) // recording → silence_wait

	action := sm.ForceStartRecording()
	if action != ActionNone {
		t.Errorf("silence_wait (still recording): got %s, want %s", action, ActionNone)
	}
	if sm.CurrentState() != StateSilenceWait {
		t.Errorf("state unchanged: got %s, want %s", sm.CurrentState(), StateSilenceWait)
	}
}

func TestEventString(t *testing.T) {
	tests := []struct {
		event Event
		want  string
	}{
		{EventSoundDetected, "sound_detected"},
		{EventSilenceDetected, "silence_detected"},
		{EventSilenceThresholdExceeded, "silence_threshold_exceeded"},
		{EventRecordingStarted, "recording_started"},
		{EventRecordingFinalized, "recording_finalized"},
		{Event(99), "unknown(99)"},
	}
	for _, tc := range tests {
		if got := tc.event.String(); got != tc.want {
			t.Errorf("Event(%d).String() = %q, want %q", int(tc.event), got, tc.want)
		}
	}
}

func TestActionString(t *testing.T) {
	tests := []struct {
		action Action
		want   string
	}{
		{ActionNone, "none"},
		{ActionStartRecording, "start_recording"},
		{ActionStopRecording, "stop_recording"},
		{ActionContinue, "continue"},
		{Action(99), "unknown(99)"},
	}
	for _, tc := range tests {
		if got := tc.action.String(); got != tc.want {
			t.Errorf("Action(%d).String() = %q, want %q", int(tc.action), got, tc.want)
		}
	}
}

func TestRecordingStart(t *testing.T) {
	sm := New(60*time.Second, 0)

	// Before recording, RecordingStart should be zero
	if !sm.RecordingStart().IsZero() {
		t.Error("RecordingStart should be zero before recording")
	}

	// Start recording
	sm.ProcessAudio(0.05, 0.02) // arming
	sm.ProcessAudio(0.05, 0.02) // recording

	rs := sm.RecordingStart()
	if rs.IsZero() {
		t.Error("RecordingStart should not be zero during recording")
	}

	// After reset, RecordingStart should be zero again
	sm.Reset()
	if !sm.RecordingStart().IsZero() {
		t.Error("RecordingStart should be zero after reset")
	}
}

func TestSilenceElapsed(t *testing.T) {
	sm := New(60*time.Second, 0)

	// Not in silence_wait: should return 0
	if sm.SilenceElapsed() != 0 {
		t.Error("SilenceElapsed should be 0 in idle state")
	}

	// Get to silence_wait
	sm.ProcessAudio(0.05, 0.02)  // arming
	sm.ProcessAudio(0.05, 0.02)  // recording
	sm.ProcessAudio(0.001, 0.02) // silence_wait

	time.Sleep(10 * time.Millisecond)
	elapsed := sm.SilenceElapsed()
	if elapsed < 10*time.Millisecond {
		t.Errorf("SilenceElapsed = %v, want >= 10ms", elapsed)
	}
}

func TestEnterError(t *testing.T) {
	sm := New(60*time.Second, 0)
	sm.EnterError()
	if sm.CurrentState() != StateError {
		t.Errorf("state = %s, want error", sm.CurrentState())
	}

	// Error state should return ActionNone
	action := sm.ProcessAudio(0.05, 0.02)
	if action != ActionNone {
		t.Errorf("action in error state = %s, want none", action)
	}
}

func TestFinalizingReturnsNone(t *testing.T) {
	sm := New(10*time.Millisecond, 0)

	// Get to finalizing
	sm.ProcessAudio(0.05, 0.02)  // arming
	sm.ProcessAudio(0.05, 0.02)  // recording
	sm.ProcessAudio(0.001, 0.02) // silence_wait
	time.Sleep(15 * time.Millisecond)
	sm.ProcessAudio(0.001, 0.02) // finalizing

	if sm.CurrentState() != StateFinalizing {
		t.Fatalf("expected finalizing, got %s", sm.CurrentState())
	}
	action := sm.ProcessAudio(0.05, 0.02)
	if action != ActionNone {
		t.Errorf("action in finalizing = %s, want none", action)
	}
}

func TestErrorState(t *testing.T) {
	sm := New(60*time.Second, 0)

	sm.EnterError()
	if sm.CurrentState() != StateError {
		t.Errorf("state: got %s, want %s", sm.CurrentState(), StateError)
	}

	// Error state does nothing
	action := sm.ProcessAudio(0.05, 0.02)
	if action != ActionNone {
		t.Errorf("error state: got %s, want %s", action, ActionNone)
	}

	// Reset recovers from error
	sm.Reset()
	if sm.CurrentState() != StateIdle {
		t.Errorf("after reset: got %s, want %s", sm.CurrentState(), StateIdle)
	}
}

// ============================================================================
// Scenario tests A–J (recording session policy)
// ============================================================================

// helper: arms and starts recording (activation window = 0).
func startRecordingSM(sm *StateMachine) {
	sm.ProcessAudio(0.05, 0.02) // idle → arming
	sm.ProcessAudio(0.05, 0.02) // arming → recording
}

// helper: reaches silence_wait from idle (activation window = 0).
func enterSilenceWait(sm *StateMachine) {
	startRecordingSM(sm)
	sm.ProcessAudio(0.001, 0.02) // recording → silence_wait
}

// A. Mic activity alone must never start recording.
func TestScenario_A_MicAloneDoesNotStartRecording(t *testing.T) {
	sm := New(60*time.Second, 0)
	sm.SetMicSessionLock(true, 20*time.Second)

	sm.SetMicActive(true) // mic is active, but BlackHole is silent

	for i := 0; i < 5; i++ {
		action := sm.ProcessAudio(0.001, 0.02) // below threshold
		if action != ActionNone {
			t.Errorf("iter %d: mic alone triggered recording, got %s", i, action)
		}
	}
	if sm.CurrentState() != StateIdle {
		t.Errorf("state: got %s, want idle", sm.CurrentState())
	}
}

// B. BlackHole activity starts recording (activation window satisfied).
func TestScenario_B_BlackHoleStartsRecording(t *testing.T) {
	sm := New(60*time.Second, 0)

	action1 := sm.ProcessAudio(0.05, 0.02) // idle → arming
	if action1 != ActionNone {
		t.Errorf("first buffer: got %s, want none", action1)
	}
	if sm.CurrentState() != StateArming {
		t.Errorf("after first sound: got %s, want arming", sm.CurrentState())
	}

	action2 := sm.ProcessAudio(0.05, 0.02) // arming → recording (0 ms window)
	if action2 != ActionStartRecording {
		t.Errorf("second buffer: got %s, want start_recording", action2)
	}
	if sm.CurrentState() != StateRecording {
		t.Errorf("state: got %s, want recording", sm.CurrentState())
	}
}

// C. A brief BlackHole burst shorter than ActivationMs does not start recording.
func TestScenario_C_ShortBurstDoesNotStart(t *testing.T) {
	sm := New(60*time.Second, 100*time.Millisecond)

	sm.ProcessAudio(0.05, 0.02) // idle → arming

	// Silence arrives before activation window expires.
	action := sm.ProcessAudio(0.001, 0.02)
	if action != ActionNone {
		t.Errorf("arming cancelled: got %s, want none", action)
	}
	if sm.CurrentState() != StateIdle {
		t.Errorf("state: got %s, want idle", sm.CurrentState())
	}
}

// D. A short BlackHole pause during recording does not split the session.
func TestScenario_D_ShortPauseDoesNotSplit(t *testing.T) {
	sm := New(60*time.Second, 0) // 60 s silence threshold

	startRecordingSM(sm)

	// First silent buffer — enters silence_wait.
	action := sm.ProcessAudio(0.001, 0.02)
	if action != ActionContinue {
		t.Errorf("silence detected: got %s, want continue", action)
	}
	if sm.CurrentState() != StateSilenceWait {
		t.Errorf("state: got %s, want silence_wait", sm.CurrentState())
	}

	// More silent buffers — still well inside the 60 s threshold.
	for i := 0; i < 3; i++ {
		a := sm.ProcessAudio(0.001, 0.02)
		if a != ActionContinue {
			t.Errorf("iter %d: got %s, want continue", i, a)
		}
	}
	if sm.CurrentState() != StateSilenceWait {
		t.Errorf("state: got %s, want silence_wait (no split yet)", sm.CurrentState())
	}
}

// E. Long silence without mic lock finalizes the recording.
func TestScenario_E_LongSilenceSplitsWithoutMic(t *testing.T) {
	sm := New(10*time.Millisecond, 0)

	startRecordingSM(sm)
	sm.ProcessAudio(0.001, 0.02) // → silence_wait

	time.Sleep(15 * time.Millisecond)

	action := sm.ProcessAudio(0.001, 0.02)
	if action != ActionStopRecording {
		t.Errorf("got %s, want stop_recording", action)
	}
	if sm.CurrentState() != StateFinalizing {
		t.Errorf("state: got %s, want finalizing", sm.CurrentState())
	}
}

// F. Mic lock prevents splitting even when silence threshold is exceeded.
func TestScenario_F_MicLockPreventsSplit(t *testing.T) {
	sm := New(10*time.Millisecond, 0)
	sm.SetMicSessionLock(true, 50*time.Millisecond)

	enterSilenceWait(sm)
	sm.SetMicActive(true) // lock engaged

	time.Sleep(15 * time.Millisecond) // silence threshold exceeded

	action := sm.ProcessAudio(0.001, 0.02)
	if action != ActionContinue {
		t.Errorf("mic lock should prevent split: got %s, want continue", action)
	}
	if sm.CurrentState() != StateSilenceWait {
		t.Errorf("state: got %s, want silence_wait", sm.CurrentState())
	}
}

// G. Mic release debounce prevents split immediately after mic goes inactive.
func TestScenario_G_MicReleaseDebouncePreventsSplit(t *testing.T) {
	sm := New(10*time.Millisecond, 0)
	sm.SetMicSessionLock(true, 50*time.Millisecond) // 50 ms debounce

	enterSilenceWait(sm)
	sm.SetMicActive(true)  // lock on
	sm.SetMicActive(false) // debounce starts

	// Silence threshold (10 ms) has passed, but debounce (50 ms) has not.
	time.Sleep(15 * time.Millisecond)

	action := sm.ProcessAudio(0.001, 0.02)
	if action != ActionContinue {
		t.Errorf("debounce should still hold: got %s, want continue", action)
	}
	if sm.CurrentState() != StateSilenceWait {
		t.Errorf("state: got %s, want silence_wait", sm.CurrentState())
	}
}

// H. Split happens after both silence and release debounce have expired.
func TestScenario_H_SplitAfterReleaseDebounceExpires(t *testing.T) {
	sm := New(10*time.Millisecond, 0)
	sm.SetMicSessionLock(true, 20*time.Millisecond)

	enterSilenceWait(sm)
	sm.SetMicActive(true)  // lock on
	sm.SetMicActive(false) // debounce starts

	// Both silence (10 ms) and debounce (20 ms) have expired.
	time.Sleep(30 * time.Millisecond)

	action := sm.ProcessAudio(0.001, 0.02)
	if action != ActionStopRecording {
		t.Errorf("both expired: got %s, want stop_recording", action)
	}
	if sm.CurrentState() != StateFinalizing {
		t.Errorf("state: got %s, want finalizing", sm.CurrentState())
	}
}

// I. BlackHole returning during silence_wait resumes the same session.
func TestScenario_I_BlackHoleReturnsInSilenceWait(t *testing.T) {
	sm := New(60*time.Second, 0)

	enterSilenceWait(sm)

	// BlackHole becomes active again.
	action := sm.ProcessAudio(0.05, 0.02)
	if action != ActionContinue {
		t.Errorf("sound resume: got %s, want continue", action)
	}
	if sm.CurrentState() != StateRecording {
		t.Errorf("state: got %s, want recording", sm.CurrentState())
	}
}

// J. Multiple BlackHole/mic toggles do not corrupt state or produce
// duplicate start/stop calls.
func TestScenario_J_MultipleTogglesDoNotCorruptState(t *testing.T) {
	sm := New(10*time.Millisecond, 0)
	sm.SetMicSessionLock(true, 20*time.Millisecond)

	var starts, stops int

	// Start recording.
	sm.ProcessAudio(0.05, 0.02)
	if a := sm.ProcessAudio(0.05, 0.02); a == ActionStartRecording {
		starts++
	}

	// BlackHole silent → silence_wait; mic holds it open.
	sm.ProcessAudio(0.001, 0.02)
	sm.SetMicActive(true)

	// BlackHole returns → back to recording.
	sm.ProcessAudio(0.05, 0.02)

	// BlackHole silent again; mic deactivates.
	sm.ProcessAudio(0.001, 0.02)
	sm.SetMicActive(false) // debounce starts (20 ms)

	// 5 ms into debounce: still held.
	time.Sleep(5 * time.Millisecond)
	a := sm.ProcessAudio(0.001, 0.02)
	if a == ActionStopRecording {
		t.Error("debounce should still hold at 5ms, got stop_recording")
	}

	// Both silence (10 ms) and debounce (20 ms) expired.
	time.Sleep(25 * time.Millisecond)
	if a := sm.ProcessAudio(0.001, 0.02); a == ActionStopRecording {
		stops++
	}

	if starts != 1 {
		t.Errorf("expected 1 start, got %d", starts)
	}
	if stops != 1 {
		t.Errorf("expected 1 stop, got %d", stops)
	}
	if sm.CurrentState() != StateFinalizing {
		t.Errorf("final state: got %s, want finalizing", sm.CurrentState())
	}
}
