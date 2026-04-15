package metadata

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

func TestWrite(t *testing.T) {
	dir := t.TempDir()
	wavPath := dir + "/2026-02-12_143015_audio_mic.wav"
	os.WriteFile(wavPath, []byte("fake"), 0644)

	meta := Recording{
		StartedAt:    time.Date(2026, 2, 12, 14, 30, 15, 0, time.UTC),
		EndedAt:      time.Date(2026, 2, 12, 15, 0, 15, 0, time.UTC),
		MicActive:    true,
		ZoomRunning:  true,
		TeamsRunning: false,
		Platform:     "darwin",
	}

	if err := Write(wavPath, meta); err != nil {
		t.Fatalf("Write: %v", err)
	}

	jsonPath := dir + "/2026-02-12_143015_audio_mic.json"
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("read json: %v", err)
	}

	var got Recording
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.DurationSecs != 1800 {
		t.Errorf("duration: got %f, want 1800", got.DurationSecs)
	}
	if !got.MicActive {
		t.Error("mic_active should be true")
	}
	if !got.ZoomRunning {
		t.Error("zoom_running should be true")
	}
	if got.Platform != "darwin" {
		t.Errorf("platform: got %s, want darwin", got.Platform)
	}
}
