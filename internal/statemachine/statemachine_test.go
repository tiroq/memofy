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
