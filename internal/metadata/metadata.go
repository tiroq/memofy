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

// FinalizationReason describes why a recording session was finalized.
type FinalizationReason string

const (
	ReasonSilenceTimeout FinalizationReason = "silence_timeout_no_mic_lock"
	ReasonManualStop     FinalizationReason = "manual_stop"
	ReasonShutdown       FinalizationReason = "shutdown"
	ReasonDeviceLost     FinalizationReason = "device_lost"
	ReasonError          FinalizationReason = "error"
	ReasonDiscardedShort FinalizationReason = "discarded_short_session"
	ReasonDiscardedEmpty FinalizationReason = "discarded_empty_audio"
)

// SessionDiagnostics holds per-session audio capture statistics.
type SessionDiagnostics struct {
	FramesReceived      int64     `json:"frames_received"`
	FramesWritten       int64     `json:"frames_written"`
	BytesWritten        int64     `json:"bytes_written"`
	RMSPeak             float64   `json:"rms_peak"`
	RMSSum              float64   `json:"-"` // internal accumulator
	RMSCount            int64     `json:"-"` // internal counter
	RMSAverage          float64   `json:"rms_average"`
	HasMeaningfulAudio  bool      `json:"has_meaningful_audio"`
	FirstAudioTimestamp time.Time `json:"first_audio_timestamp,omitempty"`
	LastAudioTimestamp  time.Time `json:"last_audio_timestamp,omitempty"`
}

// RecordRMS updates the diagnostics with an RMS reading from a buffer.
func (d *SessionDiagnostics) RecordRMS(rms float64) {
	if rms > d.RMSPeak {
		d.RMSPeak = rms
	}
	d.RMSSum += rms
	d.RMSCount++
}

// Finalize computes derived fields. Call before writing metadata.
func (d *SessionDiagnostics) Finalize(minRMS float64) {
	if d.RMSCount > 0 {
		d.RMSAverage = d.RMSSum / float64(d.RMSCount)
	}
	d.HasMeaningfulAudio = d.FramesWritten > 0 &&
		d.BytesWritten > 44 && // more than WAV header
		d.RMSPeak >= minRMS
}

// Recording holds metadata about a single recording session.
type Recording struct {
	SessionID           string             `json:"session_id"`
	StartedAt           time.Time          `json:"started_at"`
	EndedAt             time.Time          `json:"ended_at"`
	DurationSecs        float64            `json:"duration_seconds"`
	Platform            string             `json:"platform"`
	DeviceName          string             `json:"device_name"`
	FormatProfile       string             `json:"format_profile"`
	Container           string             `json:"container"`
	Codec               string             `json:"codec"`
	SampleRate          int                `json:"sample_rate"`
	Channels            int                `json:"channels"`
	BitrateKbps         int                `json:"bitrate_kbps,omitempty"`
	Threshold           float64            `json:"threshold"`
	SilenceSplitSeconds int                `json:"silence_split_seconds"`
	SplitReason         string             `json:"split_reason"`
	FinalizationReason  FinalizationReason `json:"finalization_reason"`
	MicActive           bool               `json:"mic_active,omitempty"`
	MicBundleIDs        []string           `json:"mic_bundle_ids,omitempty"`
	ZoomRunning         bool               `json:"zoom_running,omitempty"`
	TeamsRunning        bool               `json:"teams_running,omitempty"`
	MeetRunning         bool               `json:"meet_running,omitempty"`
	AppVersion          string             `json:"version"`

	// Session diagnostics
	FramesReceived     int64   `json:"frames_received"`
	FramesWritten      int64   `json:"frames_written"`
	BytesWritten       int64   `json:"bytes_written"`
	RMSPeak            float64 `json:"rms_peak"`
	RMSAverage         float64 `json:"rms_average"`
	HasMeaningfulAudio bool    `json:"has_meaningful_audio"`
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
