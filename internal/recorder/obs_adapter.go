package recorder

import (
	"fmt"
	"time"

	"github.com/tiroq/memofy/internal/diaglog"
	"github.com/tiroq/memofy/internal/obsws"
)

// T025: OBSAdapter wraps obsws.Client to implement the Recorder interface.

// OBSAdapter delegates recording operations to an obsws.Client.
type OBSAdapter struct {
	client *obsws.Client
}

// NewOBSAdapter creates a new OBSAdapter wrapping the given obsws.Client.
func NewOBSAdapter(client *obsws.Client) *OBSAdapter {
	return &OBSAdapter{client: client}
}

// Connect establishes WebSocket connection to OBS.
func (a *OBSAdapter) Connect() error {
	return a.client.Connect()
}

// Disconnect gracefully closes the OBS WebSocket connection.
func (a *OBSAdapter) Disconnect() {
	a.client.Disconnect()
}

// StartRecording initiates recording with the specified filename.
func (a *OBSAdapter) StartRecording(filename string) error {
	return a.client.StartRecord(filename)
}

// StopRecording stops the current recording and returns the result.
func (a *OBSAdapter) StopRecording(reason string) (RecordingResult, error) {
	// Snapshot start time before stopping so we can compute duration.
	state := a.client.GetRecordingState()
	startedAt := state.StartTime

	outputPath, err := a.client.StopRecord(reason)
	if err != nil {
		return RecordingResult{}, fmt.Errorf("stop recording: %w", err)
	}

	var duration time.Duration
	if !startedAt.IsZero() {
		duration = time.Since(startedAt)
	}

	return RecordingResult{
		OutputPath: outputPath,
		Duration:   duration,
		StartedAt:  startedAt,
	}, nil
}

// GetState returns the current recorder state mapped from obsws.RecordingState.
func (a *OBSAdapter) GetState() RecorderState {
	s := a.client.GetRecordingState()
	return RecorderState{
		Recording:   s.Recording,
		Connected:   a.client.IsConnected(),
		BackendName: "obs",
		OutputPath:  s.OutputPath,
		StartTime:   s.StartTime,
		Duration:    s.Duration,
	}
}

// IsConnected returns whether the OBS WebSocket is connected and identified.
func (a *OBSAdapter) IsConnected() bool {
	return a.client.IsConnected()
}

// HealthCheck queries OBS for current record status to verify connectivity.
func (a *OBSAdapter) HealthCheck() error {
	_, err := a.client.GetRecordStatus()
	if err != nil {
		return fmt.Errorf("health check: %w", err)
	}
	return nil
}

// SetLogger injects a diaglog.Logger into the underlying obsws.Client.
func (a *OBSAdapter) SetLogger(l *diaglog.Logger) {
	a.client.SetLogger(l)
}

// OnStateChanged registers a callback for recording state changes.
func (a *OBSAdapter) OnStateChanged(fn func(recording bool)) {
	a.client.OnRecordStateChanged(fn)
}

// OnDisconnected registers a callback for disconnection events.
func (a *OBSAdapter) OnDisconnected(fn func()) {
	a.client.OnDisconnected(fn)
}
