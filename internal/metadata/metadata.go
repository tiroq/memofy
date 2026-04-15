// Package metadata writes JSON sidecar files alongside recordings.
package metadata

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Recording holds metadata about a single recording session.
type Recording struct {
	StartedAt    time.Time `json:"started_at"`
	EndedAt      time.Time `json:"ended_at"`
	DurationSecs float64   `json:"duration"`
	MicActive    bool      `json:"mic_active"`
	ZoomRunning  bool      `json:"zoom_running"`
	TeamsRunning bool      `json:"teams_running"`
	Platform     string    `json:"platform"`
}

// Write creates a JSON sidecar file next to the recording.
// Given "/path/to/recording.wav", it writes "/path/to/recording.json".
func Write(wavPath string, meta Recording) error {
	if meta.Platform == "" {
		meta.Platform = runtime.GOOS
	}
	if meta.DurationSecs == 0 && !meta.EndedAt.IsZero() && !meta.StartedAt.IsZero() {
		meta.DurationSecs = meta.EndedAt.Sub(meta.StartedAt).Seconds()
	}

	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	jsonPath := strings.TrimSuffix(wavPath, filepath.Ext(wavPath)) + ".json"

	// Atomic write via temp file
	tmp := jsonPath + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return fmt.Errorf("write metadata: %w", err)
	}
	if err := os.Rename(tmp, jsonPath); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("rename metadata: %w", err)
	}

	return nil
}
