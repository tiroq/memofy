package macui

// settings_logic_test.go tests the pure (no-AppKit) logic helpers exposed by
// the settings window: ParseCSVField, BuildConfigFromFields, and their helpers.
//
// AppKit-dependent code (Show, ReadFields, buildWindow, etc.) is excluded from
// unit tests because it requires a macOS display and run loop.

import (
	"strings"
	"testing"

	"github.com/tiroq/memofy/internal/config"
)

// ─────────────────────────────────────────────────────────────────────────────
// ParseCSVField
// ─────────────────────────────────────────────────────────────────────────────

func TestParseCSVField_single(t *testing.T) {
	got := ParseCSVField("zoom.us")
	want := []string{"zoom.us"}
	assertStringSlice(t, got, want)
}

func TestParseCSVField_multiple(t *testing.T) {
	got := ParseCSVField("zoom.us, CptHost, zoomusApp")
	want := []string{"zoom.us", "CptHost", "zoomusApp"}
	assertStringSlice(t, got, want)
}

func TestParseCSVField_trimWhitespace(t *testing.T) {
	got := ParseCSVField("  Microsoft Teams  ,  Microsoft Teams Helper  ")
	want := []string{"Microsoft Teams", "Microsoft Teams Helper"}
	assertStringSlice(t, got, want)
}

func TestParseCSVField_empty(t *testing.T) {
	got := ParseCSVField("")
	if len(got) != 0 {
		t.Errorf("expected empty slice, got %v", got)
	}
}

func TestParseCSVField_whitespaceOnly(t *testing.T) {
	got := ParseCSVField("   ,  ,  ")
	if len(got) != 0 {
		t.Errorf("expected empty slice, got %v", got)
	}
}

func TestParseCSVField_mixedEmptyEntries(t *testing.T) {
	got := ParseCSVField("zoom.us,,, CptHost ,")
	want := []string{"zoom.us", "CptHost"}
	assertStringSlice(t, got, want)
}

// ─────────────────────────────────────────────────────────────────────────────
// BuildConfigFromFields – valid inputs
// ─────────────────────────────────────────────────────────────────────────────

// validFields returns a fully populated SettingsFields with valid values.
func validFields() SettingsFields {
	return SettingsFields{
		ZoomEnabled:     true,
		ZoomProcesses:   "zoom.us, CptHost",
		ZoomHints:       "Zoom Meeting, Zoom Webinar",
		TeamsEnabled:    true,
		TeamsProcesses:  "Microsoft Teams",
		TeamsHints:      "Meeting, Call",
		MeetEnabled:     true,
		MeetProcesses:   "Google Chrome, Safari",
		MeetHints:       "Google Meet, meet.google.com",
		PollInterval:    "2",
		StartThreshold:  "3",
		StopThreshold:   "6",
		AllowDevUpdates: false,
	}
}

func TestBuildConfigFromFields_valid(t *testing.T) {
	f := validFields()
	cfg, err := BuildConfigFromFields(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.PollInterval != 2 {
		t.Errorf("PollInterval: got %d, want 2", cfg.PollInterval)
	}
	if cfg.StartThreshold != 3 {
		t.Errorf("StartThreshold: got %d, want 3", cfg.StartThreshold)
	}
	if cfg.StopThreshold != 6 {
		t.Errorf("StopThreshold: got %d, want 6", cfg.StopThreshold)
	}
	if cfg.AllowDevUpdates != false {
		t.Errorf("AllowDevUpdates: got %v, want false", cfg.AllowDevUpdates)
	}
}

func TestBuildConfigFromFields_allDetectionFields_zoom(t *testing.T) {
	f := validFields()
	f.ZoomProcesses = "zoom.us, CptHost"
	f.ZoomHints = "Zoom Meeting, Zoom Webinar"
	f.ZoomEnabled = true

	cfg, err := BuildConfigFromFields(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	zoom := cfg.RuleByApp("zoom")
	if zoom == nil {
		t.Fatal("zoom rule not found")
	}
	if !zoom.Enabled {
		t.Error("zoom should be enabled")
	}
	assertStringSlice(t, zoom.ProcessNames, []string{"zoom.us", "CptHost"})
	assertStringSlice(t, zoom.WindowHints, []string{"Zoom Meeting", "Zoom Webinar"})
}

func TestBuildConfigFromFields_allDetectionFields_teams(t *testing.T) {
	f := validFields()
	f.TeamsProcesses = "Microsoft Teams, msteams"
	f.TeamsHints = "Meeting, Call, Presentation"
	f.TeamsEnabled = true

	cfg, err := BuildConfigFromFields(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	teams := cfg.RuleByApp("teams")
	if teams == nil {
		t.Fatal("teams rule not found")
	}
	if !teams.Enabled {
		t.Error("teams should be enabled")
	}
	assertStringSlice(t, teams.ProcessNames, []string{"Microsoft Teams", "msteams"})
	assertStringSlice(t, teams.WindowHints, []string{"Meeting", "Call", "Presentation"})
}

func TestBuildConfigFromFields_allDetectionFields_googleMeet(t *testing.T) {
	f := validFields()
	f.MeetProcesses = "Google Chrome, Firefox, Brave Browser"
	f.MeetHints = "Google Meet, meet.google.com"
	f.MeetEnabled = true

	cfg, err := BuildConfigFromFields(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	meet := cfg.RuleByApp("google_meet")
	if meet == nil {
		t.Fatal("google_meet rule not found")
	}
	if !meet.Enabled {
		t.Error("google_meet should be enabled")
	}
	assertStringSlice(t, meet.ProcessNames, []string{"Google Chrome", "Firefox", "Brave Browser"})
	assertStringSlice(t, meet.WindowHints, []string{"Google Meet", "meet.google.com"})
}

func TestBuildConfigFromFields_allowDevUpdates_true(t *testing.T) {
	f := validFields()
	f.AllowDevUpdates = true

	cfg, err := BuildConfigFromFields(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.AllowDevUpdates {
		t.Error("AllowDevUpdates should be true")
	}
}

func TestBuildConfigFromFields_allowDevUpdates_false(t *testing.T) {
	f := validFields()
	f.AllowDevUpdates = false

	cfg, err := BuildConfigFromFields(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.AllowDevUpdates {
		t.Error("AllowDevUpdates should be false")
	}
}

func TestBuildConfigFromFields_disableApp(t *testing.T) {
	f := validFields()
	f.ZoomEnabled = false // disable zoom but keep teams and meet enabled

	cfg, err := BuildConfigFromFields(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	zoom := cfg.RuleByApp("zoom")
	if zoom == nil {
		t.Fatal("zoom rule should still be present")
	}
	if zoom.Enabled {
		t.Error("zoom should be disabled")
	}
}

func TestBuildConfigFromFields_pollInterval_boundaries(t *testing.T) {
	for _, poll := range []string{"1", "5", "10"} {
		f := validFields()
		f.PollInterval = poll
		if _, err := BuildConfigFromFields(f); err != nil {
			t.Errorf("poll_interval=%s should be valid, got error: %v", poll, err)
		}
	}
}

func TestBuildConfigFromFields_startThreshold_boundaries(t *testing.T) {
	for _, s := range []string{"1", "5", "10"} {
		f := validFields()
		f.StartThreshold = s
		f.StopThreshold = s // keep stop >= start
		if _, err := BuildConfigFromFields(f); err != nil {
			t.Errorf("start_threshold=%s should be valid, got error: %v", s, err)
		}
	}
}

func TestBuildConfigFromFields_stopThreshold_equalsStart(t *testing.T) {
	f := validFields()
	f.StartThreshold = "4"
	f.StopThreshold = "4" // equal is allowed
	if _, err := BuildConfigFromFields(f); err != nil {
		t.Errorf("stop_threshold == start_threshold should be valid, got: %v", err)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// BuildConfigFromFields – invalid inputs
// ─────────────────────────────────────────────────────────────────────────────

func TestBuildConfigFromFields_invalidPollInterval_zero(t *testing.T) {
	f := validFields()
	f.PollInterval = "0"
	_, err := BuildConfigFromFields(f)
	if err == nil {
		t.Error("expected error for poll_interval=0")
	}
}

func TestBuildConfigFromFields_invalidPollInterval_eleven(t *testing.T) {
	f := validFields()
	f.PollInterval = "11"
	_, err := BuildConfigFromFields(f)
	if err == nil {
		t.Error("expected error for poll_interval=11")
	}
}

func TestBuildConfigFromFields_invalidPollInterval_nonNumeric(t *testing.T) {
	f := validFields()
	f.PollInterval = "two"
	_, err := BuildConfigFromFields(f)
	if err == nil {
		t.Error("expected error for non-numeric poll_interval")
	}
}

func TestBuildConfigFromFields_invalidStartThreshold_zero(t *testing.T) {
	f := validFields()
	f.StartThreshold = "0"
	_, err := BuildConfigFromFields(f)
	if err == nil {
		t.Error("expected error for start_threshold=0")
	}
}

func TestBuildConfigFromFields_invalidStartThreshold_eleven(t *testing.T) {
	f := validFields()
	f.StartThreshold = "11"
	_, err := BuildConfigFromFields(f)
	if err == nil {
		t.Error("expected error for start_threshold=11")
	}
}

func TestBuildConfigFromFields_invalidStartThreshold_nonNumeric(t *testing.T) {
	f := validFields()
	f.StartThreshold = "three"
	_, err := BuildConfigFromFields(f)
	if err == nil {
		t.Error("expected error for non-numeric start_threshold")
	}
}

func TestBuildConfigFromFields_invalidStopThreshold_lessThanStart(t *testing.T) {
	f := validFields()
	f.StartThreshold = "5"
	f.StopThreshold = "4" // stop < start: invalid
	_, err := BuildConfigFromFields(f)
	if err == nil {
		t.Error("expected error when stop_threshold < start_threshold")
	}
}

func TestBuildConfigFromFields_invalidStopThreshold_nonNumeric(t *testing.T) {
	f := validFields()
	f.StopThreshold = "six"
	_, err := BuildConfigFromFields(f)
	if err == nil {
		t.Error("expected error for non-numeric stop_threshold")
	}
}

func TestBuildConfigFromFields_allAppsDisabled(t *testing.T) {
	f := validFields()
	f.ZoomEnabled = false
	f.TeamsEnabled = false
	f.MeetEnabled = false
	_, err := BuildConfigFromFields(f)
	if err == nil {
		t.Error("expected error when all detection rules are disabled")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Rule ordering: all three apps must always be present in the output
// ─────────────────────────────────────────────────────────────────────────────

func TestBuildConfigFromFields_alwaysThreeRules(t *testing.T) {
	cfg, err := BuildConfigFromFields(validFields())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Rules) != 3 {
		t.Errorf("expected 3 rules, got %d", len(cfg.Rules))
	}
	apps := []string{"zoom", "teams", "google_meet"}
	for _, app := range apps {
		if cfg.RuleByApp(app) == nil {
			t.Errorf("rule for %q not found in config", app)
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Config round-trip: BuildConfigFromFields → SaveDetectionRules → LoadDetectionRules
// ─────────────────────────────────────────────────────────────────────────────

func TestBuildConfigFromFields_roundTrip(t *testing.T) {
	f := SettingsFields{
		ZoomEnabled:     true,
		ZoomProcesses:   "zoom.us",
		ZoomHints:       "Zoom Meeting",
		TeamsEnabled:    false,
		TeamsProcesses:  "Microsoft Teams",
		TeamsHints:      "Meeting",
		MeetEnabled:     true,
		MeetProcesses:   "Google Chrome",
		MeetHints:       "Google Meet",
		PollInterval:    "3",
		StartThreshold:  "2",
		StopThreshold:   "5",
		AllowDevUpdates: true,
	}

	cfg, err := BuildConfigFromFields(f)
	if err != nil {
		t.Fatalf("BuildConfigFromFields: %v", err)
	}

	// Save to the user config location.
	if err := config.SaveDetectionRules(cfg); err != nil {
		t.Fatalf("SaveDetectionRules: %v", err)
	}

	// Reload and verify all fields.
	loaded, err := config.LoadDetectionRules()
	if err != nil {
		t.Fatalf("LoadDetectionRules: %v", err)
	}

	if loaded.PollInterval != 3 {
		t.Errorf("PollInterval: got %d, want 3", loaded.PollInterval)
	}
	if loaded.StartThreshold != 2 {
		t.Errorf("StartThreshold: got %d, want 2", loaded.StartThreshold)
	}
	if loaded.StopThreshold != 5 {
		t.Errorf("StopThreshold: got %d, want 5", loaded.StopThreshold)
	}
	if !loaded.AllowDevUpdates {
		t.Error("AllowDevUpdates should be true after reload")
	}

	zoom := loaded.RuleByApp("zoom")
	if zoom == nil || !zoom.Enabled {
		t.Error("zoom should be enabled after reload")
	}
	assertStringSlice(t, zoom.ProcessNames, []string{"zoom.us"})
	assertStringSlice(t, zoom.WindowHints, []string{"Zoom Meeting"})

	teams := loaded.RuleByApp("teams")
	if teams == nil || teams.Enabled {
		t.Error("teams should be disabled after reload")
	}

	meet := loaded.RuleByApp("google_meet")
	if meet == nil || !meet.Enabled {
		t.Error("google_meet should be enabled after reload")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// defaultDetectionConfig
// ─────────────────────────────────────────────────────────────────────────────

func TestDefaultDetectionConfig_isValid(t *testing.T) {
	cfg := defaultDetectionConfig()
	if err := cfg.Validate(); err != nil {
		t.Errorf("defaultDetectionConfig is invalid: %v", err)
	}
}

func TestDefaultDetectionConfig_hasAllApps(t *testing.T) {
	cfg := defaultDetectionConfig()
	for _, app := range []string{"zoom", "teams", "google_meet"} {
		rule := cfg.RuleByApp(app)
		if rule == nil {
			t.Errorf("defaultDetectionConfig missing rule for %q", app)
			continue
		}
		if !rule.Enabled {
			t.Errorf("defaultDetectionConfig: rule for %q should be enabled", app)
		}
		if len(rule.ProcessNames) == 0 {
			t.Errorf("defaultDetectionConfig: rule for %q has no process names", app)
		}
		if len(rule.WindowHints) == 0 {
			t.Errorf("defaultDetectionConfig: rule for %q has no window hints", app)
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Helper assertions
// ─────────────────────────────────────────────────────────────────────────────

func assertStringSlice(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("slice length mismatch: got %v (len %d), want %v (len %d)",
			got, len(got), want, len(want))
		return
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("index %d: got %q, want %q", i, got[i], want[i])
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Whitespace trimming in field values
// ─────────────────────────────────────────────────────────────────────────────

func TestBuildConfigFromFields_trimsPollIntervalWhitespace(t *testing.T) {
	f := validFields()
	f.PollInterval = "  2  "
	cfg, err := BuildConfigFromFields(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.PollInterval != 2 {
		t.Errorf("PollInterval: got %d, want 2", cfg.PollInterval)
	}
}

func TestBuildConfigFromFields_trimsThresholdWhitespace(t *testing.T) {
	f := validFields()
	f.StartThreshold = " 3 "
	f.StopThreshold = " 6 "
	cfg, err := BuildConfigFromFields(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.StartThreshold != 3 {
		t.Errorf("StartThreshold: got %d, want 3", cfg.StartThreshold)
	}
	if cfg.StopThreshold != 6 {
		t.Errorf("StopThreshold: got %d, want 6", cfg.StopThreshold)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// processNames / windowHints nil-safety
// ─────────────────────────────────────────────────────────────────────────────

func TestProcessNames_nilRule(t *testing.T) {
	if got := processNames(nil); got != nil {
		t.Errorf("processNames(nil) should return nil, got %v", got)
	}
}

func TestWindowHints_nilRule(t *testing.T) {
	if got := windowHints(nil); got != nil {
		t.Errorf("windowHints(nil) should return nil, got %v", got)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Error message quality checks
// ─────────────────────────────────────────────────────────────────────────────

func TestBuildConfigFromFields_errorMessages(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*SettingsFields)
		wantMsg string
	}{
		{
			name:    "non-numeric poll interval",
			mutate:  func(f *SettingsFields) { f.PollInterval = "abc" },
			wantMsg: "poll interval",
		},
		{
			name:    "non-numeric start threshold",
			mutate:  func(f *SettingsFields) { f.StartThreshold = "abc" },
			wantMsg: "start threshold",
		},
		{
			name:    "non-numeric stop threshold",
			mutate:  func(f *SettingsFields) { f.StopThreshold = "abc" },
			wantMsg: "stop threshold",
		},
		{
			name:    "stop < start",
			mutate:  func(f *SettingsFields) { f.StartThreshold = "5"; f.StopThreshold = "3" },
			wantMsg: "stop_threshold",
		},
		{
			name:    "all disabled",
			mutate:  func(f *SettingsFields) { f.ZoomEnabled = false; f.TeamsEnabled = false; f.MeetEnabled = false },
			wantMsg: "at least one",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := validFields()
			tc.mutate(&f)
			_, err := BuildConfigFromFields(f)
			if err == nil {
				t.Fatal("expected an error")
			}
			if !strings.Contains(err.Error(), tc.wantMsg) {
				t.Errorf("error %q does not mention %q", err.Error(), tc.wantMsg)
			}
		})
	}
}
