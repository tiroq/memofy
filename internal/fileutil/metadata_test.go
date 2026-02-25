package fileutil

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWriteMetadata_Basic(t *testing.T) {
	dir := t.TempDir()
	recPath := filepath.Join(dir, "2025-01-15_1430_Zoom_Standup.mp4")
	// Create a dummy recording file so the dir exists.
	if err := os.WriteFile(recPath, []byte("fake"), 0644); err != nil {
		t.Fatal(err)
	}

	meta := &RecordingMetadata{
		Version:         "1.2.3",
		SessionID:       "abc123",
		StartedAt:       time.Date(2025, 1, 15, 14, 30, 0, 0, time.UTC),
		StoppedAt:       time.Date(2025, 1, 15, 15, 0, 0, 0, time.UTC),
		Duration:        "30m0s",
		DurationMs:      1800000,
		App:             "zoom",
		WindowTitle:     "Daily Standup",
		Origin:          "auto",
		RecorderBackend: "obs",
		OutputFile:      recPath,
	}

	if err := WriteMetadata(recPath, meta); err != nil {
		t.Fatalf("WriteMetadata: %v", err)
	}

	// Verify file exists at expected path.
	metaPath := filepath.Join(dir, "2025-01-15_1430_Zoom_Standup.meta.json")
	data, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatalf("read meta file: %v", err)
	}

	var got RecordingMetadata
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.Version != "1.2.3" {
		t.Errorf("version = %q, want %q", got.Version, "1.2.3")
	}
	if got.SessionID != "abc123" {
		t.Errorf("session_id = %q, want %q", got.SessionID, "abc123")
	}
	if got.App != "zoom" {
		t.Errorf("app = %q, want %q", got.App, "zoom")
	}
	if got.DurationMs != 1800000 {
		t.Errorf("duration_ms = %d, want %d", got.DurationMs, 1800000)
	}
	if got.Origin != "auto" {
		t.Errorf("origin = %q, want %q", got.Origin, "auto")
	}
}

func TestWriteMetadata_WithASR(t *testing.T) {
	dir := t.TempDir()
	recPath := filepath.Join(dir, "recording.mp4")
	if err := os.WriteFile(recPath, []byte("fake"), 0644); err != nil {
		t.Fatal(err)
	}

	meta := &RecordingMetadata{
		Version:    "dev",
		OutputFile: recPath,
		ASR: &ASRMeta{
			Backend:       "remote_whisper_api",
			Model:         "small",
			Language:      "en",
			Formats:       []string{"txt", "srt"},
			Success:       true,
			TranscribedAt: time.Date(2025, 1, 15, 15, 1, 0, 0, time.UTC),
		},
	}

	if err := WriteMetadata(recPath, meta); err != nil {
		t.Fatalf("WriteMetadata: %v", err)
	}

	metaPath := filepath.Join(dir, "recording.meta.json")
	data, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	var got RecordingMetadata
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.ASR == nil {
		t.Fatal("ASR is nil, expected non-nil")
	}
	if got.ASR.Backend != "remote_whisper_api" {
		t.Errorf("asr.backend = %q, want %q", got.ASR.Backend, "remote_whisper_api")
	}
	if !got.ASR.Success {
		t.Error("asr.success = false, want true")
	}
	if len(got.ASR.Formats) != 2 {
		t.Errorf("asr.formats len = %d, want 2", len(got.ASR.Formats))
	}
}

func TestWriteMetadata_NilASR(t *testing.T) {
	dir := t.TempDir()
	recPath := filepath.Join(dir, "recording.mp4")
	if err := os.WriteFile(recPath, []byte("fake"), 0644); err != nil {
		t.Fatal(err)
	}

	meta := &RecordingMetadata{
		Version:    "dev",
		OutputFile: recPath,
	}

	if err := WriteMetadata(recPath, meta); err != nil {
		t.Fatalf("WriteMetadata: %v", err)
	}

	metaPath := filepath.Join(dir, "recording.meta.json")
	data, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	// ASR should be omitted from JSON.
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := raw["asr"]; ok {
		t.Error("expected no 'asr' field in JSON when ASR is nil")
	}
}

func TestMetadataPath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"recording.mp4", "recording.meta.json"},
		{"/path/to/file.mkv", "/path/to/file.meta.json"},
		{"no-ext", "no-ext.meta.json"},
	}
	for _, tt := range tests {
		got := metadataPath(tt.input)
		if got != tt.want {
			t.Errorf("metadataPath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestWriteMetadata_AtomicNoPartialFile(t *testing.T) {
	// Write to a non-existent directory should fail cleanly.
	badPath := filepath.Join(t.TempDir(), "nonexistent", "sub", "recording.mp4")
	meta := &RecordingMetadata{Version: "dev"}
	err := WriteMetadata(badPath, meta)
	if err == nil {
		t.Fatal("expected error for non-existent directory")
	}
}
