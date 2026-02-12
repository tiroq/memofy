package detector

import (
	"time"

	"github.com/tiroq/memofy/internal/config"
)

// MultiDetector runs multiple detectors and aggregates results
type MultiDetector struct {
	detectors []Detector
}

// NewMultiDetector creates a detector that runs all configured detectors
func NewMultiDetector(cfg *config.DetectionConfig) *MultiDetector {
	md := &MultiDetector{
		detectors: make([]Detector, 0),
	}

	// Add Zoom detector
	md.detectors = append(md.detectors, NewZoomDetector(cfg.Rules))

	// Add Teams detector
	md.detectors = append(md.detectors, NewTeamsDetector(cfg.Rules))

	// Add Google Meet detector
	md.detectors = append(md.detectors, NewGoogleMeetDetector(cfg.Rules))

	return md
}

// Name returns the detector identifier
func (md *MultiDetector) Name() string {
	return "multi"
}

// Detect runs all detectors and returns aggregated result
// Returns the first positive detection with highest confidence
func (md *MultiDetector) Detect() (*DetectionState, error) {
	// Default state: no meeting detected
	result := &DetectionState{
		MeetingDetected: false,
		DetectedApp:     AppNone,
		RawDetections:   RawDetection{},
		ConfidenceLevel: ConfidenceNone,
		EvaluatedAt:     time.Now(),
	}

	var bestState *DetectionState
	var bestConfidence int

	confidenceLevels := map[ConfidenceLevel]int{
		ConfidenceNone:   0,
		ConfidenceLow:    1,
		ConfidenceMedium: 2,
		ConfidenceHigh:   3,
	}

	// Run all detectors
	for _, detector := range md.detectors {
		state, err := detector.Detect()
		if err != nil {
			// Log error but continue with other detectors
			continue
		}

		// Merge raw detections (OR logic - any detection counts)
		result.RawDetections = mergeRawDetections(result.RawDetections, state.RawDetections)

		// Track highest confidence detection
		if state.MeetingDetected {
			confidenceValue := confidenceLevels[state.ConfidenceLevel]
			if bestState == nil || confidenceValue > bestConfidence {
				bestState = state
				bestConfidence = confidenceValue
			}
		}
	}

	// Use best detection if found
	if bestState != nil {
		result.MeetingDetected = true
		result.DetectedApp = bestState.DetectedApp
		result.ConfidenceLevel = bestState.ConfidenceLevel
		result.WindowTitle = bestState.WindowTitle
	}

	return result, nil
}

// mergeRawDetections combines detection signals with OR logic
func mergeRawDetections(a, b RawDetection) RawDetection {
	return RawDetection{
		ZoomProcessRunning:     a.ZoomProcessRunning || b.ZoomProcessRunning,
		ZoomHostRunning:        a.ZoomHostRunning || b.ZoomHostRunning,
		ZoomWindowMatch:        a.ZoomWindowMatch || b.ZoomWindowMatch,
		TeamsProcessRunning:    a.TeamsProcessRunning || b.TeamsProcessRunning,
		TeamsWindowMatch:       a.TeamsWindowMatch || b.TeamsWindowMatch,
		GoogleMeetRunning:      a.GoogleMeetRunning || b.GoogleMeetRunning,
		GoogleMeetWindowMatch:  a.GoogleMeetWindowMatch || b.GoogleMeetWindowMatch,
	}
}

// DetectMeeting is a convenience function that creates a multi-detector and runs detection
func DetectMeeting(cfg *config.DetectionConfig) (*DetectionState, error) {
	detector := NewMultiDetector(cfg)
	return detector.Detect()
}
