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

	"github.com/tiroq/memofy/internal/config"
)

// Recording holds metadata about a single recording session.
type Recording struct {
	StartedAt           time.Time `json:"started_at"`
	EndedAt             time.Time `json:"ended_at"`
	DurationSecs        float64   `json:"duration_seconds"`
	Platform            string    `json:"platform"`
	DeviceName          string    `json:"device_name"`
	Threshold           float64   `json:"threshold"`
	SilenceSplitSeconds int       `json:"silence_split_seconds"`
	MicActive           bool      `json:"mic_active"`
	ZoomRunning         bool      `json:"zoom_running"`
	TeamsRunning        bool      `json:"teams_running"`
	SessionID           string    `json:"session_id"`
	AppVersion          string    `json:"app_version"`
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
	if meta.SessionID == "" {
		meta.SessionID = meta.StartedAt.Format("20060102T150405")
	}
	if meta.Threshold == 0 {
		meta.Threshold = config.Default().Audio.Threshold
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
