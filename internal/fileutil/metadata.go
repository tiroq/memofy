// Package fileutil provides recording file utilities.
// T034: FR-013 — sidecar metadata JSON for recordings.
package fileutil

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// RecordingMetadata is the sidecar metadata written alongside each recording.
type RecordingMetadata struct {
	Version         string    `json:"version"`
	SessionID       string    `json:"session_id"`
	StartedAt       time.Time `json:"started_at"`
	StoppedAt       time.Time `json:"stopped_at"`
	Duration        string    `json:"duration"`
	DurationMs      int64     `json:"duration_ms"`
	App             string    `json:"app"`
	WindowTitle     string    `json:"window_title"`
	Origin          string    `json:"recording_origin"`
	RecorderBackend string    `json:"recorder_backend"`
	OutputFile      string    `json:"output_file"`
	ASR             *ASRMeta  `json:"asr,omitempty"`
}

// ASRMeta captures transcription details for the sidecar.
type ASRMeta struct {
	Backend       string    `json:"backend"`
	Model         string    `json:"model"`
	Language      string    `json:"language"`
	Formats       []string  `json:"formats"`
	Success       bool      `json:"success"`
	Error         string    `json:"error,omitempty"`
	TranscribedAt time.Time `json:"transcribed_at,omitempty"`
}

// WriteMetadata writes a <basepath>.meta.json sidecar file alongside the
// recording. Uses atomic write (temp + rename) consistent with ipc patterns.
// T034: FR-013.
func WriteMetadata(recordingPath string, meta *RecordingMetadata) error {
	metaPath := metadataPath(recordingPath)
	dir := filepath.Dir(metaPath)

	tmpFile, err := os.CreateTemp(dir, "meta-*.tmp")
	if err != nil {
		return fmt.Errorf("create metadata temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	// Ensure cleanup on error.
	success := false
	defer func() {
		if !success {
			tmpFile.Close()
			os.Remove(tmpPath)
		}
	}()

	encoder := json.NewEncoder(tmpFile)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(meta); err != nil {
		return fmt.Errorf("encode metadata: %w", err)
	}

	if err := tmpFile.Sync(); err != nil {
		return fmt.Errorf("sync metadata: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close metadata temp: %w", err)
	}
	success = true // prevent defer cleanup

	if err := os.Rename(tmpPath, metaPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename metadata: %w", err)
	}
	return nil
}

// metadataPath returns <basepath>.meta.json for a given recording file path.
func metadataPath(recordingPath string) string {
	ext := filepath.Ext(recordingPath)
	base := recordingPath[:len(recordingPath)-len(ext)]
	return base + ".meta.json"
}
