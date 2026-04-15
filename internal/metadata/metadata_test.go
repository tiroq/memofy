package metadata

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

func TestWrite(t *testing.T) {
	dir := t.TempDir()
	wavPath := dir + "/2026-02-12_143015_audio_high.wav"
	os.WriteFile(wavPath, []byte("fake"), 0644)

	meta := Recording{
		StartedAt:           time.Date(2026, 2, 12, 14, 30, 15, 0, time.UTC),
		EndedAt:             time.Date(2026, 2, 12, 15, 0, 15, 0, time.UTC),
		MicActive:           true,
		ZoomRunning:         true,
		TeamsRunning:        false,
		Platform:            "darwin",
		DeviceName:          "BlackHole 2ch",
		FormatProfile:       "high",
		Container:           "m4a",
		Codec:               "aac",
		SampleRate:          32000,
		Channels:            1,
		BitrateKbps:         64,
		Threshold:           0.02,
		SilenceSplitSeconds: 60,
		SplitReason:         "silence_threshold",
		AppVersion:          "0.2.0",
	}

	if err := Write(wavPath, meta); err != nil {
		t.Fatalf("Write: %v", err)
	}

	jsonPath := dir + "/2026-02-12_143015_audio_high.json"
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
	if got.DeviceName != "BlackHole 2ch" {
		t.Errorf("device_name: got %s, want BlackHole 2ch", got.DeviceName)
	}
	if got.FormatProfile != "high" {
		t.Errorf("format_profile: got %s, want high", got.FormatProfile)
	}
	if got.Container != "m4a" {
		t.Errorf("container: got %s, want m4a", got.Container)
	}
	if got.Codec != "aac" {
		t.Errorf("codec: got %s, want aac", got.Codec)
	}
	if got.SampleRate != 32000 {
		t.Errorf("sample_rate: got %d, want 32000", got.SampleRate)
	}
	if got.BitrateKbps != 64 {
		t.Errorf("bitrate_kbps: got %d, want 64", got.BitrateKbps)
	}
	if got.SplitReason != "silence_threshold" {
		t.Errorf("split_reason: got %s, want silence_threshold", got.SplitReason)
	}
	if got.SessionID == "" {
		t.Error("session_id should not be empty")
	}
	if got.AppVersion != "0.2.0" {
		t.Errorf("version: got %s, want 0.2.0", got.AppVersion)
	}
}

func TestWriteAutocomputedFields(t *testing.T) {
	dir := t.TempDir()
	wavPath := dir + "/test_auto.wav"
	os.WriteFile(wavPath, []byte("fake"), 0644)

	start := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	end := time.Date(2026, 1, 1, 10, 5, 0, 0, time.UTC)

	// No duration, no platform, no session_id, no threshold — should be autocomputed
	meta := Recording{
		StartedAt: start,
		EndedAt:   end,
	}

	if err := Write(wavPath, meta); err != nil {
		t.Fatalf("Write: %v", err)
	}

	jsonPath := dir + "/test_auto.json"
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("read json: %v", err)
	}

	var got Recording
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.DurationSecs != 300 {
		t.Errorf("duration: got %f, want 300", got.DurationSecs)
	}
	if got.Platform == "" {
		t.Error("platform should be autocomputed")
	}
	if got.SessionID == "" {
		t.Error("session_id should be autocomputed")
	}
	if got.Threshold == 0 {
		t.Error("threshold should be autocomputed from default")
	}
}

func TestWriteM4AExtension(t *testing.T) {
	dir := t.TempDir()
	m4aPath := dir + "/recording.m4a"
	os.WriteFile(m4aPath, []byte("fake"), 0644)

	meta := Recording{
		StartedAt: time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC),
		EndedAt:   time.Date(2026, 1, 1, 10, 1, 0, 0, time.UTC),
		Container: "m4a",
	}

	if err := Write(m4aPath, meta); err != nil {
		t.Fatalf("Write: %v", err)
	}

	// Should create .json alongside .m4a
	jsonPath := dir + "/recording.json"
	if _, err := os.Stat(jsonPath); os.IsNotExist(err) {
		t.Error("expected .json file alongside .m4a")
	}
}
