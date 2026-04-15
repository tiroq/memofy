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
