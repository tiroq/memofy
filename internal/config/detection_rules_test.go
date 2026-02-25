package config

import (
	"testing"
)

// ─────────────────────────────────────────────────────────────────────────────
// RuleByApp
// ─────────────────────────────────────────────────────────────────────────────

func TestRuleByApp_found(t *testing.T) {
	cfg := &DetectionConfig{
		Rules: []DetectionRule{
			{Application: "zoom", Enabled: true, ProcessNames: []string{"zoom.us"}},
			{Application: "teams", Enabled: true, ProcessNames: []string{"Microsoft Teams"}},
		},
	}
	rule := cfg.RuleByApp("zoom")
	if rule == nil {
		t.Fatal("expected zoom rule, got nil")
	}
	if rule.Application != "zoom" {
		t.Errorf("got application %q, want %q", rule.Application, "zoom")
	}
}

func TestRuleByApp_notFound(t *testing.T) {
	cfg := &DetectionConfig{
		Rules: []DetectionRule{
			{Application: "zoom", Enabled: true},
		},
	}
	if got := cfg.RuleByApp("nonexistent"); got != nil {
		t.Errorf("expected nil for unknown app, got %+v", got)
	}
}

func TestRuleByApp_emptyRules(t *testing.T) {
	cfg := &DetectionConfig{}
	if got := cfg.RuleByApp("zoom"); got != nil {
		t.Errorf("expected nil for empty rules, got %+v", got)
	}
}

func TestRuleByApp_returnsPointerToSliceElement(t *testing.T) {
	cfg := &DetectionConfig{
		Rules: []DetectionRule{
			{Application: "teams", Enabled: true, ProcessNames: []string{"Microsoft Teams"}},
		},
	}
	rule := cfg.RuleByApp("teams")
	if rule == nil {
		t.Fatal("rule should not be nil")
	}
	// Mutate through the pointer – the change must be visible in the original slice.
	rule.Enabled = false
	if cfg.Rules[0].Enabled {
		t.Error("mutation through RuleByApp pointer should affect original slice")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Validate
// ─────────────────────────────────────────────────────────────────────────────

func validTestConfig() *DetectionConfig {
	return &DetectionConfig{
		PollInterval:   2,
		StartThreshold: 3,
		StopThreshold:  6,
		Rules: []DetectionRule{
			{Application: "zoom", Enabled: true, ProcessNames: []string{"zoom.us"},
				WindowHints: []string{"Zoom Meeting"}},
			{Application: "teams", Enabled: true, ProcessNames: []string{"Microsoft Teams"},
				WindowHints: []string{"Meeting"}},
		},
	}
}

func TestValidate_valid(t *testing.T) {
	if err := validTestConfig().Validate(); err != nil {
		t.Errorf("expected nil error for valid config, got: %v", err)
	}
}

func TestValidate_pollInterval_zero(t *testing.T) {
	cfg := validTestConfig()
	cfg.PollInterval = 0
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for poll_interval=0")
	}
}

func TestValidate_pollInterval_eleven(t *testing.T) {
	cfg := validTestConfig()
	cfg.PollInterval = 11
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for poll_interval=11")
	}
}

func TestValidate_pollInterval_one(t *testing.T) {
	cfg := validTestConfig()
	cfg.PollInterval = 1
	if err := cfg.Validate(); err != nil {
		t.Errorf("poll_interval=1 should be valid, got: %v", err)
	}
}

func TestValidate_pollInterval_ten(t *testing.T) {
	cfg := validTestConfig()
	cfg.PollInterval = 10
	if err := cfg.Validate(); err != nil {
		t.Errorf("poll_interval=10 should be valid, got: %v", err)
	}
}

func TestValidate_startThreshold_zero(t *testing.T) {
	cfg := validTestConfig()
	cfg.StartThreshold = 0
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for start_threshold=0")
	}
}

func TestValidate_startThreshold_eleven(t *testing.T) {
	cfg := validTestConfig()
	cfg.StartThreshold = 11
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for start_threshold=11")
	}
}

func TestValidate_stopThreshold_lessThanStart(t *testing.T) {
	cfg := validTestConfig()
	cfg.StartThreshold = 5
	cfg.StopThreshold = 4
	if err := cfg.Validate(); err == nil {
		t.Error("expected error when stop_threshold < start_threshold")
	}
}

func TestValidate_stopThreshold_equalsStart(t *testing.T) {
	cfg := validTestConfig()
	cfg.StartThreshold = 5
	cfg.StopThreshold = 5
	if err := cfg.Validate(); err != nil {
		t.Errorf("stop_threshold == start_threshold should be valid, got: %v", err)
	}
}

func TestValidate_noEnabledRules(t *testing.T) {
	cfg := validTestConfig()
	for i := range cfg.Rules {
		cfg.Rules[i].Enabled = false
	}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error when no rules are enabled")
	}
}

func TestValidate_oneEnabledRule_isEnough(t *testing.T) {
	cfg := validTestConfig()
	cfg.Rules[0].Enabled = false
	cfg.Rules[1].Enabled = true
	if err := cfg.Validate(); err != nil {
		t.Errorf("one enabled rule should be valid, got: %v", err)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// AllowDevUpdates field round-trip
// ─────────────────────────────────────────────────────────────────────────────

func TestAllowDevUpdates_defaultFalse(t *testing.T) {
	cfg := validTestConfig()
	if cfg.AllowDevUpdates {
		t.Error("AllowDevUpdates should default to false")
	}
}

func TestAllowDevUpdates_saveAndLoad(t *testing.T) {
	cfg := validTestConfig()
	cfg.AllowDevUpdates = true
	if err := SaveDetectionRules(cfg); err != nil {
		t.Fatalf("SaveDetectionRules: %v", err)
	}
	loaded, err := LoadDetectionRules()
	if err != nil {
		t.Fatalf("LoadDetectionRules: %v", err)
	}
	if !loaded.AllowDevUpdates {
		t.Error("AllowDevUpdates should be true after round-trip")
	}
	// Restore for other tests.
	cfg.AllowDevUpdates = false
	if err := SaveDetectionRules(cfg); err != nil {
		t.Fatalf("cleanup SaveDetectionRules: %v", err)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// DetectionRule fields
// ─────────────────────────────────────────────────────────────────────────────

func TestDetectionRule_allFields(t *testing.T) {
	rule := DetectionRule{
		Application:  "zoom",
		ProcessNames: []string{"zoom.us", "CptHost"},
		WindowHints:  []string{"Zoom Meeting", "Zoom Webinar"},
		Enabled:      true,
	}
	if rule.Application != "zoom" {
		t.Errorf("Application: got %q, want %q", rule.Application, "zoom")
	}
	if len(rule.ProcessNames) != 2 {
		t.Errorf("ProcessNames length: got %d, want 2", len(rule.ProcessNames))
	}
	if len(rule.WindowHints) != 2 {
		t.Errorf("WindowHints length: got %d, want 2", len(rule.WindowHints))
	}
	if !rule.Enabled {
		t.Error("Enabled should be true")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// DetectionConfig fields
// ─────────────────────────────────────────────────────────────────────────────

func TestDetectionConfig_allFields(t *testing.T) {
	cfg := &DetectionConfig{
		PollInterval:    5,
		StartThreshold:  2,
		StopThreshold:   4,
		AllowDevUpdates: true,
		Rules: []DetectionRule{
			{Application: "zoom", Enabled: true, ProcessNames: []string{"zoom.us"},
				WindowHints: []string{"Zoom Meeting"}},
		},
	}
	if cfg.PollInterval != 5 {
		t.Errorf("PollInterval: got %d, want 5", cfg.PollInterval)
	}
	if cfg.StartThreshold != 2 {
		t.Errorf("StartThreshold: got %d, want 2", cfg.StartThreshold)
	}
	if cfg.StopThreshold != 4 {
		t.Errorf("StopThreshold: got %d, want 4", cfg.StopThreshold)
	}
	if !cfg.AllowDevUpdates {
		t.Error("AllowDevUpdates: got false, want true")
	}
	if len(cfg.Rules) != 1 {
		t.Errorf("Rules length: got %d, want 1", len(cfg.Rules))
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// T032: ASR Config Validation (FR-013)
// ─────────────────────────────────────────────────────────────────────────────

func TestValidate_noASR_backwardCompat(t *testing.T) {
	cfg := validTestConfig()
	cfg.ASR = nil
	if err := cfg.Validate(); err != nil {
		t.Errorf("config without ASR should be valid, got: %v", err)
	}
}

func TestValidate_asrDisabled_noValidation(t *testing.T) {
	cfg := validTestConfig()
	cfg.ASR = &ASRConfig{Enabled: false, Backend: "invalid"}
	if err := cfg.Validate(); err != nil {
		t.Errorf("disabled ASR should skip validation, got: %v", err)
	}
}

func TestValidate_asrEnabled_valid(t *testing.T) {
	cfg := validTestConfig()
	cfg.ASR = &ASRConfig{
		Enabled:       true,
		Mode:          "batch",
		Backend:       "remote_whisper_api",
		OutputFormats: []string{"txt", "srt"},
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("valid ASR config should pass, got: %v", err)
	}
}

func TestValidate_asrEnabled_invalidMode(t *testing.T) {
	cfg := validTestConfig()
	cfg.ASR = &ASRConfig{Enabled: true, Mode: "realtime", Backend: "local_whisper"}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for invalid ASR mode")
	}
}

func TestValidate_asrEnabled_invalidBackend(t *testing.T) {
	cfg := validTestConfig()
	cfg.ASR = &ASRConfig{Enabled: true, Backend: "openai"}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for invalid ASR backend")
	}
}

func TestValidate_asrEnabled_invalidFormat(t *testing.T) {
	cfg := validTestConfig()
	cfg.ASR = &ASRConfig{Enabled: true, Backend: "local_whisper", OutputFormats: []string{"txt", "mp3"}}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for invalid output format")
	}
}

func TestValidate_asrEnabled_fallbackSameAsBackend(t *testing.T) {
	cfg := validTestConfig()
	cfg.ASR = &ASRConfig{Enabled: true, Backend: "local_whisper", FallbackBackend: "local_whisper"}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error when fallback equals primary backend")
	}
}

func TestValidate_asrEnabled_fallbackValid(t *testing.T) {
	cfg := validTestConfig()
	cfg.ASR = &ASRConfig{Enabled: true, Backend: "local_whisper", FallbackBackend: "remote_whisper_api"}
	if err := cfg.Validate(); err != nil {
		t.Errorf("valid fallback should pass, got: %v", err)
	}
}

func TestASRConfig_applyDefaults_emptyMode(t *testing.T) {
	cfg := validTestConfig()
	cfg.ASR = &ASRConfig{Enabled: true, Backend: "local_whisper"}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ASR.Mode != "batch" {
		t.Errorf("Mode default: got %q, want %q", cfg.ASR.Mode, "batch")
	}
	if len(cfg.ASR.OutputFormats) != 1 || cfg.ASR.OutputFormats[0] != "txt" {
		t.Errorf("OutputFormats default: got %v, want [txt]", cfg.ASR.OutputFormats)
	}
}

func TestValidate_asrEnabled_invalidFallbackBackend(t *testing.T) {
	cfg := validTestConfig()
	cfg.ASR = &ASRConfig{Enabled: true, Backend: "local_whisper", FallbackBackend: "invalid"}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for invalid fallback backend name")
	}
}
