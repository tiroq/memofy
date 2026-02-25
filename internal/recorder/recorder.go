// Package recorder abstracts recording backends behind a common interface.
// T025: FR-013 — enables future native macOS recorder alongside OBS.
package recorder

import (
	"time"

	"github.com/tiroq/memofy/internal/diaglog"
)

// RecordingResult contains the outcome of a completed recording.
type RecordingResult struct {
	OutputPath string
	Duration   time.Duration
	StartedAt  time.Time
}

// RecorderState represents the current state of a recording backend.
type RecorderState struct {
	Recording   bool
	Connected   bool
	BackendName string // "obs" | "native" (future)
	OutputPath  string
	StartTime   time.Time
	Duration    int // seconds
}

// Recorder is the interface that recording backends must implement.
type Recorder interface {
	Connect() error
	Disconnect()
	StartRecording(filename string) error
	StopRecording(reason string) (RecordingResult, error)
	GetState() RecorderState
	IsConnected() bool
	HealthCheck() error
	SetLogger(l *diaglog.Logger)
	OnStateChanged(fn func(recording bool))
	OnDisconnected(fn func())
}
