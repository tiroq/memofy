package engine_test

import (
	"testing"

	"github.com/tiroq/memofy/internal/config"
	"github.com/tiroq/memofy/internal/engine"
)

// newTestEngine creates an Engine from default config without starting audio.
func newTestEngine(t *testing.T) *engine.Engine {
	t.Helper()
	cfg := config.Default()
	cfg.Output.Dir = t.TempDir()
	return engine.New(cfg, nil)
}

// TestNew_InitialState verifies that a newly created engine reports idle state
// and that the mic is not active before Start() is called.
func TestNew_InitialState(t *testing.T) {
	eng := newTestEngine(t)

	status := eng.GetStatus()
	if status.State != "idle" {
		t.Errorf("initial State: got %q, want %q", status.State, "idle")
	}
	if status.MicActive {
		t.Error("MicActive should be false before Start()")
	}
	if status.ZoomRunning || status.TeamsRunning || status.MeetRunning {
		t.Error("no meeting apps should be reported before Start()")
	}
}

// TestNew_DeviceSwitchChannelCapacity asserts the device-switch channel has
// capacity exactly 1. A capacity of 0 would cause pollMonitor to block; a
// capacity > 1 would queue stale requests. One slot is the correct invariant.
func TestNew_DeviceSwitchChannelCapacity(t *testing.T) {
	eng := newTestEngine(t)
	if got := eng.DeviceSwitchChCap(); got != 1 {
		t.Errorf("deviceSwitchCh capacity: got %d, want 1", got)
	}
}

// TestNew_IsMicActiveInitiallyFalse asserts that isMicActive() returns false
// for a freshly constructed engine (no monitor has run yet).
func TestNew_IsMicActiveInitiallyFalse(t *testing.T) {
	eng := newTestEngine(t)
	if eng.IsMicActive() {
		t.Error("IsMicActive() should return false before any monitor poll")
	}
}
