package asr

import "time"

// T026: Unified ASR interface and types. FR-013

// Segment represents a single transcribed segment with timing.
type Segment struct {
	Start    time.Duration
	End      time.Duration
	Text     string
	Language string
	Score    float64 // confidence 0.0–1.0
}

// Transcript represents a complete transcription result.
type Transcript struct {
	Segments []Segment
	Language string
	Duration time.Duration
	Model    string
	Backend  string
}

// TranscribeOptions configures a transcription request.
type TranscribeOptions struct {
	Language   string // "" = auto-detect
	Model      string // backend-specific model name
	Timestamps bool
	MaxSegLen  int // max segment length in seconds
}

// HealthStatus reports backend health.
type HealthStatus struct {
	OK      bool
	Backend string
	Message string
	Latency time.Duration
}

// Backend is the interface that ASR backends must implement.
type Backend interface {
	Name() string
	TranscribeFile(filePath string, opts TranscribeOptions) (*Transcript, error)
	HealthCheck() (*HealthStatus, error)
}
