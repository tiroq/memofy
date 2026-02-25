package recorder

import (
	"testing"
	"time"

	"github.com/tiroq/memofy/internal/obsws"
)

// T025: Compile-time interface compliance check.
var _ Recorder = (*OBSAdapter)(nil)

func TestNewOBSAdapter(t *testing.T) {
	client := obsws.NewClient("ws://localhost:4455", "")
	adapter := NewOBSAdapter(client)
	if adapter == nil {
		t.Fatal("NewOBSAdapter returned nil")
	}
}

func TestGetState_MapsFields(t *testing.T) {
	// Create a client with default state — no connection needed for
	// GetRecordingState (cached value) and GetState mapping.
	client := obsws.NewClient("ws://localhost:4455", "")
	adapter := NewOBSAdapter(client)

	state := adapter.GetState()

	if state.BackendName != "obs" {
		t.Errorf("BackendName = %q, want %q", state.BackendName, "obs")
	}
	if state.Connected {
		t.Error("Connected = true, want false for unconnected client")
	}
	if state.Recording {
		t.Error("Recording = true, want false for fresh client")
	}
}

func TestRecordingResult_Fields(t *testing.T) {
	now := time.Now()
	result := RecordingResult{
		OutputPath: "/tmp/recording.mkv",
		Duration:   5 * time.Minute,
		StartedAt:  now,
	}

	if result.OutputPath != "/tmp/recording.mkv" {
		t.Errorf("OutputPath = %q, want %q", result.OutputPath, "/tmp/recording.mkv")
	}
	if result.Duration != 5*time.Minute {
		t.Errorf("Duration = %v, want %v", result.Duration, 5*time.Minute)
	}
	if !result.StartedAt.Equal(now) {
		t.Errorf("StartedAt = %v, want %v", result.StartedAt, now)
	}
}

func TestRecorderState_Fields(t *testing.T) {
	now := time.Now()
	state := RecorderState{
		Recording:   true,
		Connected:   true,
		BackendName: "obs",
		OutputPath:  "/tmp/out.mkv",
		StartTime:   now,
		Duration:    120,
	}

	if !state.Recording {
		t.Error("Recording = false, want true")
	}
	if !state.Connected {
		t.Error("Connected = false, want true")
	}
	if state.BackendName != "obs" {
		t.Errorf("BackendName = %q, want %q", state.BackendName, "obs")
	}
	if state.Duration != 120 {
		t.Errorf("Duration = %d, want 120", state.Duration)
	}
}
