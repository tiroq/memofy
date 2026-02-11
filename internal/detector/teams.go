package detector

import (
	"time"

	"github.com/tiroq/memofy/internal/config"
)

// TeamsDetector detects active Microsoft Teams meetings
type TeamsDetector struct {
	processDetector *ProcessDetection
	rules           []config.DetectionRule
}

// NewTeamsDetector creates a Teams meeting detector
func NewTeamsDetector(rules []config.DetectionRule) *TeamsDetector {
	return &TeamsDetector{
		processDetector: NewProcessDetection(),
		rules:           rules,
	}
}

// Name returns the detector identifier
func (td *TeamsDetector) Name() string {
	return "teams"
}

// Detect checks if a Microsoft Teams meeting is active
func (td *TeamsDetector) Detect() (*DetectionState, error) {
	state := &DetectionState{
		MeetingDetected: false,
		DetectedApp:     AppNone,
		RawDetections: RawDetection{
			TeamsProcessRunning: false,
			TeamsWindowMatch:    false,
		},
		ConfidenceLevel: ConfidenceNone,
		EvaluatedAt:     time.Now(),
	}

	// Find Teams rule
	var teamsRule *config.DetectionRule
	for i := range td.rules {
		if td.rules[i].Application == "teams" && td.rules[i].Enabled {
			teamsRule = &td.rules[i]
			break
		}
	}

	if teamsRule == nil {
		return state, nil
	}

	// Check Teams process
	teamsRunning, _ := td.processDetector.IsProcessRunning(teamsRule.ProcessNames)
	state.RawDetections.TeamsProcessRunning = teamsRunning

	if !teamsRunning {
		return state, nil
	}

	// Teams process found - at least low confidence
	state.ConfidenceLevel = ConfidenceLow

	// Check window title for meeting indicators
	windowMatch, _ := td.processDetector.WindowMatches(teamsRule.WindowHints)
	state.RawDetections.TeamsWindowMatch = windowMatch

	// Determine confidence and meeting detection
	if windowMatch {
		// Process + window title = medium-high confidence for Teams
		// (Teams doesn't have a separate host process like Zoom's CptHost)
		state.ConfidenceLevel = ConfidenceMedium
		state.MeetingDetected = true
		state.DetectedApp = AppTeams
	}

	return state, nil
}
