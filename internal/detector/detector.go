package detector

import "time"

// DetectedApp represents which application triggered detection
type DetectedApp string

const (
	AppNone  DetectedApp = ""
	AppZoom  DetectedApp = "zoom"
	AppTeams DetectedApp = "teams"
)

// ConfidenceLevel represents detection confidence
type ConfidenceLevel string

const (
	ConfidenceNone   ConfidenceLevel = "none"   // No meeting detected
	ConfidenceLow    ConfidenceLevel = "low"    // Process only
	ConfidenceMedium ConfidenceLevel = "medium" // Process + window OR host
	ConfidenceHigh   ConfidenceLevel = "high"   // Process + window + host
)

// RawDetection contains individual signal checks
type RawDetection struct {
	ZoomProcessRunning  bool `json:"zoom_process_running"`
	ZoomHostRunning     bool `json:"zoom_host_running"`      // CptHost process
	ZoomWindowMatch     bool `json:"zoom_window_match"`      // Window title hint match
	TeamsProcessRunning bool `json:"teams_process_running"`
	TeamsWindowMatch    bool `json:"teams_window_match"`     // Window title hint match
}

// DetectionState represents current meeting detection evaluation
type DetectionState struct {
	MeetingDetected bool            `json:"meeting_detected"`    // Stable detection result
	DetectedApp     DetectedApp     `json:"detected_app"`        // Which app triggered
	WindowTitle     string          `json:"window_title"`        // Active window title for filename
	RawDetections   RawDetection    `json:"raw_detections"`      // Individual signal checks
	ConfidenceLevel ConfidenceLevel `json:"confidence"`          // Detection confidence
	EvaluatedAt     time.Time       `json:"evaluated_at"`        // When evaluated
}

// Detector interface for meeting detection
type Detector interface {
	Detect() (*DetectionState, error)
	Name() string
}
