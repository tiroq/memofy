package engine_test

import (
	"encoding/binary"
	"os"
	"path/filepath"
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

// --- WAV validation tests ---

func TestValidateWAVFile_ValidWAV(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "valid.wav")

	// Create minimal valid WAV: RIFF header + some data.
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	header := make([]byte, 44)
	copy(header[0:4], "RIFF")
	binary.LittleEndian.PutUint32(header[4:8], 36+100) // file size - 8
	copy(header[8:12], "WAVE")
	copy(header[12:16], "fmt ")
	binary.LittleEndian.PutUint32(header[16:20], 16)
	binary.LittleEndian.PutUint16(header[20:22], 1) // PCM
	binary.LittleEndian.PutUint16(header[22:24], 1) // mono
	binary.LittleEndian.PutUint32(header[24:28], 44100)
	binary.LittleEndian.PutUint32(header[28:32], 88200)
	binary.LittleEndian.PutUint16(header[32:34], 2)
	binary.LittleEndian.PutUint16(header[34:36], 16)
	copy(header[36:40], "data")
	binary.LittleEndian.PutUint32(header[40:44], 100)
	f.Write(header)
	f.Write(make([]byte, 100)) // audio data
	f.Close()

	if !engine.ValidateWAVFile(path) {
		t.Error("expected valid WAV to pass validation")
	}
}

func TestValidateWAVFile_HeaderOnly(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "headeronly.wav")

	header := make([]byte, 44)
	copy(header[0:4], "RIFF")
	copy(header[8:12], "WAVE")
	os.WriteFile(path, header, 0644)

	if engine.ValidateWAVFile(path) {
		t.Error("expected header-only WAV to fail validation")
	}
}

func TestValidateWAVFile_NotRIFF(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "notriff.wav")

	data := make([]byte, 100)
	copy(data[0:4], "XXXX")
	copy(data[8:12], "WAVE")
	os.WriteFile(path, data, 0644)

	if engine.ValidateWAVFile(path) {
		t.Error("expected non-RIFF file to fail validation")
	}
}

func TestValidateWAVFile_NotWAVE(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "notwave.wav")

	data := make([]byte, 100)
	copy(data[0:4], "RIFF")
	copy(data[8:12], "XXXX")
	os.WriteFile(path, data, 0644)

	if engine.ValidateWAVFile(path) {
		t.Error("expected non-WAVE file to fail validation")
	}
}

func TestValidateWAVFile_TooSmall(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tiny.wav")
	os.WriteFile(path, []byte("tiny"), 0644)

	if engine.ValidateWAVFile(path) {
		t.Error("expected tiny file to fail validation")
	}
}

func TestValidateWAVFile_Nonexistent(t *testing.T) {
	if engine.ValidateWAVFile("/nonexistent/path.wav") {
		t.Error("expected nonexistent file to fail validation")
	}
}

// --- Config defaults tests ---

func TestDefaultConfig_ExitThreshold(t *testing.T) {
	cfg := config.Default()
	if cfg.Audio.ExitThreshold <= 0 {
		t.Errorf("ExitThreshold default should be > 0, got %f", cfg.Audio.ExitThreshold)
	}
	if cfg.Audio.ExitThreshold >= cfg.Audio.Threshold {
		t.Errorf("ExitThreshold (%f) should be < Threshold (%f)",
			cfg.Audio.ExitThreshold, cfg.Audio.Threshold)
	}
}

func TestDefaultConfig_MinSessionSeconds(t *testing.T) {
	cfg := config.Default()
	if cfg.Session.MinSessionSeconds <= 0 {
		t.Errorf("MinSessionSeconds default should be > 0, got %d", cfg.Session.MinSessionSeconds)
	}
}

func TestDefaultConfig_DiscardShortSessions(t *testing.T) {
	cfg := config.Default()
	if !cfg.Session.DiscardShortSessions {
		t.Error("DiscardShortSessions should default to true")
	}
}

func TestNew_HysteresisConfigured(t *testing.T) {
	cfg := config.Default()
	cfg.Output.Dir = t.TempDir()
	// Defaults have ExitThreshold < Threshold, so hysteresis should be configured.
	eng := engine.New(cfg, nil)
	// Engine should create without panic — hysteresis is wired internally.
	status := eng.GetStatus()
	if status.State != "idle" {
		t.Errorf("state: got %q, want idle", status.State)
	}
}
