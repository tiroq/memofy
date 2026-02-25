package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// DetectionRule represents configurable meeting detection criteria
type DetectionRule struct {
	Application  string   `json:"application"`   // "zoom" or "teams"
	ProcessNames []string `json:"process_names"` // Process name patterns
	WindowHints  []string `json:"window_hints"`  // Window title substrings
	Enabled      bool     `json:"enabled"`       // Rule active
}

// DetectionConfig holds all detection configuration
type DetectionConfig struct {
	Rules           []DetectionRule `json:"rules"`
	PollInterval    int             `json:"poll_interval_seconds"`       // Detection polling interval
	StartThreshold  int             `json:"start_threshold"`             // Consecutive detections to start
	StopThreshold   int             `json:"stop_threshold"`              // Consecutive non-detections to stop
	AllowDevUpdates bool            `json:"allow_dev_updates,omitempty"` // Allow pre-release and dev versions (optional, defaults to false)
	ASR             *ASRConfig      `json:"asr,omitempty"`              // T032: ASR configuration (nil = disabled). FR-013
}

// LoadDetectionRules reads configuration from ~/.config/memofy/detection-rules.json
// Falls back to configs/default-detection-rules.json if user config doesn't exist
func LoadDetectionRules() (*DetectionConfig, error) {
	// Try user config first
	configDir := filepath.Join(os.Getenv("HOME"), ".config", "memofy")
	userConfigPath := filepath.Join(configDir, "detection-rules.json")

	data, err := os.ReadFile(userConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Fall back to default config
			defaultPath := "configs/default-detection-rules.json"
			data, err = os.ReadFile(defaultPath)
			if err != nil {
				return nil, fmt.Errorf("failed to load config: %w", err)
			}

			// Create user config directory for future saves
			if err := os.MkdirAll(configDir, 0755); err != nil {
				return nil, fmt.Errorf("failed to create config directory: %w", err)
			}
		} else {
			return nil, err
		}
	}

	var config DetectionConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Validate config
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &config, nil
}

// SaveDetectionRules writes configuration to ~/.config/memofy/detection-rules.json
func SaveDetectionRules(config *DetectionConfig) error {
	// Validate before saving
	if err := config.Validate(); err != nil {
		return err
	}

	configDir := filepath.Join(os.Getenv("HOME"), ".config", "memofy")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	configPath := filepath.Join(configDir, "detection-rules.json")

	// Write with indentation for readability
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

// RuleByApp returns the first DetectionRule whose Application field matches
// appName, or nil if no such rule exists.
func (c *DetectionConfig) RuleByApp(appName string) *DetectionRule {
	for i := range c.Rules {
		if c.Rules[i].Application == appName {
			return &c.Rules[i]
		}
	}
	return nil
}

// Validate checks DetectionConfig for validity
func (c *DetectionConfig) Validate() error {
	// PollInterval must be between 1 and 10 seconds
	if c.PollInterval < 1 || c.PollInterval > 10 {
		return fmt.Errorf("poll_interval_seconds must be between 1 and 10, got %d", c.PollInterval)
	}

	// StartThreshold must be at least 1
	if c.StartThreshold < 1 || c.StartThreshold > 10 {
		return fmt.Errorf("start_threshold must be between 1 and 10, got %d", c.StartThreshold)
	}

	// StopThreshold must be >= StartThreshold
	if c.StopThreshold < c.StartThreshold {
		return fmt.Errorf("stop_threshold (%d) must be >= start_threshold (%d)", c.StopThreshold, c.StartThreshold)
	}

	// At least one rule must be enabled
	hasEnabled := false
	for _, rule := range c.Rules {
		if rule.Enabled {
			hasEnabled = true
			break
		}
	}
	if !hasEnabled {
		return fmt.Errorf("at least one detection rule must be enabled")
	}

	// T032: validate ASR config (FR-013)
	if err := c.validateASR(); err != nil {
		return err
	}

	return nil
}

// T032: ASR configuration types. FR-013

// ASRConfig holds Automatic Speech Recognition settings.
type ASRConfig struct {
	Enabled         bool     `json:"enabled"`                     // false = ASR disabled entirely
	Mode            string   `json:"mode"`                        // "batch" | "live" | "hybrid" (default "batch")
	Backend         string   `json:"backend"`                     // "remote_whisper_api" | "local_whisper" | "google_stt"
	FallbackBackend string   `json:"fallback_backend,omitempty"`  // optional fallback
	DraftModel      string   `json:"draft_model,omitempty"`       // future: live/hybrid
	RecoveryModel   string   `json:"recovery_model,omitempty"`    // future: two-pass
	OutputFormats   []string `json:"output_formats,omitempty"`    // ["txt", "srt", "vtt"] default ["txt"]

	Remote RemoteWhisperConfig `json:"remote,omitempty"`
	Local  LocalWhisperConfig  `json:"local,omitempty"`
	Google GoogleSTTConfig     `json:"google,omitempty"`
}

// RemoteWhisperConfig holds remote Whisper API backend settings.
type RemoteWhisperConfig struct {
	BaseURL        string `json:"base_url"`
	Token          string `json:"token,omitempty"`
	TimeoutSeconds int    `json:"timeout_seconds"` // default 120
	Retries        int    `json:"retries"`          // default 3
	Model          string `json:"model"`            // default "small"
}

// LocalWhisperConfig holds local whisper CLI backend settings.
type LocalWhisperConfig struct {
	BinaryPath string `json:"binary_path"`
	ModelPath  string `json:"model_path"`
	Model      string `json:"model"`   // default "small"
	Threads    int    `json:"threads"` // 0 = auto
}

// GoogleSTTConfig holds Google Cloud Speech-to-Text settings.
type GoogleSTTConfig struct {
	CredentialsFile string `json:"credentials_file,omitempty"`
	LanguageCode    string `json:"language_code,omitempty"` // default "en-US"
}

// validASRModes lists accepted ASR modes.
var validASRModes = map[string]bool{
	"batch":  true,
	"live":   true,
	"hybrid": true,
}

// validASRBackends lists accepted ASR backend names.
var validASRBackends = map[string]bool{
	"remote_whisper_api": true,
	"local_whisper":      true,
	"google_stt":         true,
}

// validOutputFormats lists accepted transcript output formats.
var validOutputFormats = map[string]bool{
	"txt": true,
	"srt": true,
	"vtt": true,
}

// applyDefaults fills in zero-value fields with sensible defaults.
func (a *ASRConfig) applyDefaults() {
	if a.Mode == "" {
		a.Mode = "batch"
	}
	if len(a.OutputFormats) == 0 {
		a.OutputFormats = []string{"txt"}
	}
}

// validateASR validates ASR configuration if present and enabled.
func (c *DetectionConfig) validateASR() error {
	if c.ASR == nil || !c.ASR.Enabled {
		return nil
	}

	c.ASR.applyDefaults()

	if !validASRModes[c.ASR.Mode] {
		return fmt.Errorf("asr.mode must be \"batch\", \"live\", or \"hybrid\", got %q", c.ASR.Mode)
	}
	if !validASRBackends[c.ASR.Backend] {
		return fmt.Errorf("asr.backend must be \"remote_whisper_api\", \"local_whisper\", or \"google_stt\", got %q", c.ASR.Backend)
	}
	if c.ASR.FallbackBackend != "" {
		if !validASRBackends[c.ASR.FallbackBackend] {
			return fmt.Errorf("asr.fallback_backend must be a valid backend name, got %q", c.ASR.FallbackBackend)
		}
		if c.ASR.FallbackBackend == c.ASR.Backend {
			return fmt.Errorf("asr.fallback_backend must differ from asr.backend")
		}
	}
	for _, f := range c.ASR.OutputFormats {
		if !validOutputFormats[f] {
			return fmt.Errorf("asr.output_formats: unknown format %q", f)
		}
	}
	return nil
}
