package detector

import (
	"time"

	"github.com/tiroq/memofy/internal/config"
)

// GoogleMeetDetector detects active Google Meet meetings via browser
type GoogleMeetDetector struct {
	processDetector *ProcessDetection
	rules           []config.DetectionRule
}

// NewGoogleMeetDetector creates a Google Meet meeting detector
func NewGoogleMeetDetector(rules []config.DetectionRule) *GoogleMeetDetector {
	return &GoogleMeetDetector{
		processDetector: NewProcessDetection(),
		rules:           rules,
	}
}

// Name returns the detector identifier
func (gd *GoogleMeetDetector) Name() string {
	return "google_meet"
}

// Detect checks if a Google Meet meeting is active
func (gd *GoogleMeetDetector) Detect() (*DetectionState, error) {
	state := &DetectionState{
		MeetingDetected: false,
		DetectedApp:     AppNone,
		RawDetections: RawDetection{
			GoogleMeetRunning:     false,
			GoogleMeetWindowMatch: false,
		},
		ConfidenceLevel: ConfidenceNone,
		EvaluatedAt:     time.Now(),
	}

	// Find Google Meet rule
	var googleMeetRule *config.DetectionRule
	for i := range gd.rules {
		if gd.rules[i].Application == "google_meet" && gd.rules[i].Enabled {
			googleMeetRule = &gd.rules[i]
			break
		}
	}

	if googleMeetRule == nil {
		return state, nil
	}

	// Check if browser is running (Chrome, Safari, Edge, Firefox, Brave)
	browserRunning, _ := gd.processDetector.IsProcessRunning(googleMeetRule.ProcessNames)
	state.RawDetections.GoogleMeetRunning = browserRunning

	if !browserRunning {
		return state, nil
	}

	// Browser process found - at least low confidence
	state.ConfidenceLevel = ConfidenceLow

	// Check window title for Google Meet indicators
	windowMatch, windowTitle := gd.processDetector.WindowMatches(googleMeetRule.WindowHints)
	state.RawDetections.GoogleMeetWindowMatch = windowMatch
	if windowMatch {
		state.WindowTitle = windowTitle
	}

	// Determine confidence and meeting detection
	// Google Meet window title matching is the primary signal
	if windowMatch {
		// Browser + Google Meet window title = medium confidence
		state.ConfidenceLevel = ConfidenceMedium
		state.MeetingDetected = true
		state.DetectedApp = AppGoogleMeet
	}

	return state, nil
}
