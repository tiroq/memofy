package ipc

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// OperatingMode represents user control mode for recording behavior
type OperatingMode string

const (
	ModeAuto   OperatingMode = "auto"   // Automatic detection-based recording
	ModeManual OperatingMode = "manual" // User-controlled recording only
	ModePaused OperatingMode = "paused" // All detection suspended
)

// StatusSnapshot represents the complete system state at a point in time
type StatusSnapshot struct {
	Mode           OperatingMode `json:"mode"`            // Current operating mode
	DetectionState interface{}   `json:"detection_state"` // Raw detection state (defined in detector package)
	RecordingState interface{}   `json:"recording_state"` // Actual recording state (defined in obsws package)
	TeamsDetected  bool          `json:"teams_detected"`  // Teams meeting active
	ZoomDetected   bool          `json:"zoom_detected"`   // Zoom meeting active
	StartStreak    int           `json:"start_streak"`    // Consecutive detections
	StopStreak     int           `json:"stop_streak"`     // Consecutive non-detections
	LastAction     string        `json:"last_action"`     // Last action taken
	LastError      string        `json:"last_error"`      // Last error message
	Timestamp      time.Time     `json:"timestamp"`       // Snapshot time
	OBSConnected   bool          `json:"obs_connected"`   // OBS connection status
}

// WriteStatus persists StatusSnapshot to ~/.cache/memofy/status.json using atomic write
func WriteStatus(status *StatusSnapshot) error {
	cacheDir := filepath.Join(os.Getenv("HOME"), ".cache", "memofy")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return err
	}

	statusPath := filepath.Join(cacheDir, "status.json")
	return atomicWriteJSON(statusPath, status)
}

// ReadStatus loads StatusSnapshot from ~/.cache/memofy/status.json
func ReadStatus() (*StatusSnapshot, error) {
	statusPath := filepath.Join(os.Getenv("HOME"), ".cache", "memofy", "status.json")

	data, err := os.ReadFile(statusPath)
	if err != nil {
		return nil, err
	}

	var status StatusSnapshot
	if err := json.Unmarshal(data, &status); err != nil {
		return nil, err
	}

	return &status, nil
}

// atomicWriteJSON writes data to a file atomically using temp file + rename
func atomicWriteJSON(path string, data interface{}) error {
	// Create temp file in same directory
	dir := filepath.Dir(path)
	tmpFile, err := os.CreateTemp(dir, "status-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()

	// Ensure cleanup on error
	defer func() {
		if tmpFile != nil {
			tmpFile.Close()
			os.Remove(tmpPath)
		}
	}()

	// Write JSON with indentation for readability
	encoder := json.NewEncoder(tmpFile)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		return err
	}

	// Sync to disk before rename
	if err := tmpFile.Sync(); err != nil {
		return err
	}

	// Close file before rename
	if err := tmpFile.Close(); err != nil {
		return err
	}
	tmpFile = nil // Prevent defer cleanup

	// Atomic rename
	return os.Rename(tmpPath, path)
}
