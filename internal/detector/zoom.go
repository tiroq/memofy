package detector

import (
	"time"

	"github.com/tiroq/memofy/internal/config"
)

// ZoomDetector detects active Zoom meetings
type ZoomDetector struct {
	processDetector *ProcessDetection
	rules           []config.DetectionRule
}

// NewZoomDetector creates a Zoom meeting detector
func NewZoomDetector(rules []config.DetectionRule) *ZoomDetector {
	return &ZoomDetector{
		processDetector: NewProcessDetection(),
		rules:           rules,
	}
}

// Name returns the detector identifier
func (zd *ZoomDetector) Name() string {
	return "zoom"
}

// Detect checks if a Zoom meeting is active
func (zd *ZoomDetector) Detect() (*DetectionState, error) {
	state := &DetectionState{
		MeetingDetected: false,
		DetectedApp:     AppNone,
		RawDetections: RawDetection{
			ZoomProcessRunning: false,
			ZoomHostRunning:    false,
			ZoomWindowMatch:    false,
		},
		ConfidenceLevel: ConfidenceNone,
		EvaluatedAt:     time.Now(),
	}

	// Find Zoom rule
	var zoomRule *config.DetectionRule
	for i := range zd.rules {
		if zd.rules[i].Application == "zoom" && zd.rules[i].Enabled {
			zoomRule = &zd.rules[i]
			break
		}
	}

	if zoomRule == nil {
		return state, nil
	}

	// Check Zoom main process
	zoomRunning, _ := zd.processDetector.IsProcessRunning(zoomRule.ProcessNames)
	state.RawDetections.ZoomProcessRunning = zoomRunning

	if !zoomRunning {
		return state, nil
	}

	// Zoom process found - at least low confidence
	state.ConfidenceLevel = ConfidenceLow

	// Check for CptHost process (indicator of active meeting)
	hostRunning, _ := zd.processDetector.IsProcessRunning([]string{"CptHost"})
	state.RawDetections.ZoomHostRunning = hostRunning

	// Check window title
	windowMatch, _ := zd.processDetector.WindowMatches(zoomRule.WindowHints)
	state.RawDetections.ZoomWindowMatch = windowMatch

	// Determine confidence and meeting detection
	if hostRunning && windowMatch {
		// Process + host + window title = high confidence
		state.ConfidenceLevel = ConfidenceHigh
		state.MeetingDetected = true
		state.DetectedApp = AppZoom
	} else if hostRunning || windowMatch {
		// Process + (host OR window) = medium confidence
		state.ConfidenceLevel = ConfidenceMedium
		state.MeetingDetected = true
		state.DetectedApp = AppZoom
	}

	return state, nil
}
