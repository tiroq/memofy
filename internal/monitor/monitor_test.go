package monitor

import "testing"

func TestIsMicNoiseBundle(t *testing.T) {
	tests := []struct {
		id   string
		want bool
	}{
		// Known noise bundles — should be filtered.
		{"com.apple.CoreSpeech", true},
		{"com.apple.SpeechRecognitionCore", true},
		{"com.apple.accessibility.heard", true},
		// Case-insensitive match.
		{"com.apple.corespeech", true},
		{"COM.APPLE.CORESPEECH", true},
		// Prefix match — sub-components should also match.
		{"com.apple.CoreSpeech.speechservicesd", true},
		// Real meeting apps — must NOT be filtered.
		{"com.microsoft.teams2", false},
		{"com.microsoft.teams", false},
		{"us.zoom.xos", false},
		{"com.google.Chrome", false},
		{"org.mozilla.firefox", false},
		{"com.apple.Safari", false},
		// Empty / unknown.
		{"", false},
		{"com.example.unrelated", false},
	}
	for _, tc := range tests {
		if got := isMicNoiseBundle(tc.id); got != tc.want {
			t.Errorf("isMicNoiseBundle(%q) = %v, want %v", tc.id, got, tc.want)
		}
	}
}

func TestContainsAny(t *testing.T) {
	procs := []string{"Google Chrome", "Safari", "Finder", "zoom.us"}
	tests := []struct {
		hints []string
		want  bool
	}{
		{[]string{"zoom.us"}, true},
		{[]string{"zoom"}, true},
		{[]string{"Google Meet", "meet"}, false}, // Meet runs in browser, no "meet" process
		{[]string{"Chrome"}, true},
		{[]string{"Microsoft Teams"}, false},
		{[]string{"nothing"}, false},
	}
	for _, tc := range tests {
		if got := containsAny(procs, tc.hints...); got != tc.want {
			t.Errorf("containsAny(procs, %v) = %v, want %v", tc.hints, got, tc.want)
		}
	}
}

func TestContainsAny_CaseInsensitive(t *testing.T) {
	procs := []string{"Microsoft Teams"}
	if !containsAny(procs, "microsoft teams") {
		t.Error("containsAny should be case-insensitive")
	}
	if !containsAny(procs, "MICROSOFT TEAMS") {
		t.Error("containsAny should be case-insensitive (upper)")
	}
}

func TestContainsAny_EmptyInputs(t *testing.T) {
	if containsAny(nil, "zoom") {
		t.Error("nil procs should return false")
	}
	if containsAny([]string{}, "zoom") {
		t.Error("empty procs should return false")
	}
	if containsAny([]string{"zoom"}) {
		t.Error("no hints should return false")
	}
}

func TestSnapshotInCall(t *testing.T) {
	tests := []struct {
		name string
		snap Snapshot
		want bool
	}{
		{"no meeting", Snapshot{}, false},
		{"zoom in call", Snapshot{ZoomInCall: true}, true},
		{"mic active", Snapshot{MicActive: true}, true},
		{"zoom open but not in call", Snapshot{ZoomRunning: true}, false},
		{"teams open but no mic", Snapshot{TeamsRunning: true}, false},
	}
	for _, tc := range tests {
		if got := tc.snap.InCall(); got != tc.want {
			t.Errorf("%s: InCall() = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestNewMonitor_InitialSnapshot(t *testing.T) {
	m := New()
	snap := m.Current()
	if snap.ZoomRunning || snap.TeamsRunning || snap.MeetRunning || snap.MicActive {
		t.Error("initial snapshot should have all fields false")
	}
	if snap.MicBundleIDs != nil {
		t.Errorf("initial MicBundleIDs should be nil, got %v", snap.MicBundleIDs)
	}
}

// TestMicActiveFiltersNoiseBundles verifies the filtering logic that should
// be applied in Poll(). We test the filtering predicate directly since Poll()
// depends on live system state.
func TestMicActiveFiltersNoiseBundles(t *testing.T) {
	// Simulate the filtering loop from Poll().
	filterMicActive := func(ids []string) bool {
		for _, id := range ids {
			if !isMicNoiseBundle(id) {
				return true
			}
		}
		return false
	}

	tests := []struct {
		name string
		ids  []string
		want bool
	}{
		{"no ids", nil, false},
		{"only noise", []string{"com.apple.CoreSpeech"}, false},
		{"noise + meeting app", []string{"com.apple.CoreSpeech", "com.google.Chrome"}, true},
		{"only meeting app", []string{"com.google.Chrome"}, true},
		{"multiple noise bundles", []string{"com.apple.CoreSpeech", "com.apple.SpeechRecognitionCore"}, false},
		{"noise + teams", []string{"com.apple.CoreSpeech", "com.microsoft.teams2"}, true},
		{"noise + zoom", []string{"com.apple.CoreSpeech", "us.zoom.xos"}, true},
	}
	for _, tc := range tests {
		if got := filterMicActive(tc.ids); got != tc.want {
			t.Errorf("%s: micActive = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestGetPIDs(t *testing.T) {
	pids := map[string]string{
		"100": "Google Chrome",
		"101": "Google Chrome Helper",
		"200": "Microsoft Teams",
		"300": "Finder",
	}

	got := getPIDs(pids, "chrome")
	if len(got) != 2 {
		t.Errorf("expected 2 Chrome PIDs, got %d: %v", len(got), got)
	}

	got = getPIDs(pids, "teams")
	if len(got) != 1 {
		t.Errorf("expected 1 Teams PID, got %d: %v", len(got), got)
	}

	got = getPIDs(pids, "zoom")
	if len(got) != 0 {
		t.Errorf("expected 0 Zoom PIDs, got %d: %v", len(got), got)
	}
}
