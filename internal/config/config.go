// Package config provides YAML configuration loading for Memofy.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config is the top-level Memofy configuration.
type Config struct {
	Audio      AudioConfig      `yaml:"audio"`
	Session    SessionConfig    `yaml:"session"`
	Output     OutputConfig     `yaml:"output"`
	Monitoring MonitoringConfig `yaml:"monitoring"`
	Logging    LoggingConfig    `yaml:"logging"`
	Platform   PlatformConfig   `yaml:"platform"`
	UI         UIConfig         `yaml:"ui"`
}

// AudioConfig controls audio capture and silence detection.
type AudioConfig struct {
	Device              string  `yaml:"device"`                // "auto" or device name substring
	InputDeviceName     string  `yaml:"input_device_name"`     // alias for device
	Threshold           float64 `yaml:"threshold"`             // RMS level for sound detection
	LevelThreshold      float64 `yaml:"level_threshold"`       // alias for threshold
	SilenceSeconds      int     `yaml:"silence_seconds"`       // seconds of silence before splitting
	SilenceSplitSeconds int     `yaml:"silence_split_seconds"` // alias for silence_seconds
	SilenceHysteresis   float64 `yaml:"silence_hysteresis"`    // hysteresis value
	HysteresisRatio     float64 `yaml:"hysteresis_ratio"`      // ratio for hysteresis band
	ActivationMs        int     `yaml:"activation_ms"`         // consecutive active-signal ms to start
	SampleRate          int     `yaml:"sample_rate"`           // capture sample rate (default 44100)
	Channels            int     `yaml:"channels"`              // capture channels (default 2)
	FormatProfile       string  `yaml:"format_profile"`        // high, balanced, lightweight, wav
}

// SessionConfig controls recording session behavior.
type SessionConfig struct {
	MinSessionSeconds               int  `yaml:"min_session_seconds"`
	KeepSingleSessionWhileMicActive bool `yaml:"keep_single_session_while_mic_active"`
}

// OutputConfig controls where recordings are saved.
type OutputConfig struct {
	Dir               string `yaml:"dir"`       // output directory, supports ~ expansion
	Directory         string `yaml:"directory"` // alias for dir
	WriteMetadataJSON bool   `yaml:"write_metadata_json"`
}

// MonitoringConfig controls meeting app detection.
type MonitoringConfig struct {
	DetectZoom                      bool `yaml:"detect_zoom"`
	DetectTeams                     bool `yaml:"detect_teams"`
	DetectMicUsage                  bool `yaml:"detect_mic_usage"`
	KeepSingleSessionWhileMicActive bool `yaml:"keep_single_session_while_mic_active"`
	PollIntervalMs                  int  `yaml:"poll_interval_ms"`
}

// LoggingConfig controls log file output.
type LoggingConfig struct {
	File  string `yaml:"file"`
	Level string `yaml:"level"`
}

// PlatformConfig holds platform-specific device hints.
type PlatformConfig struct {
	MacOSDevice string `yaml:"macos_device"` // e.g. "BlackHole"
	LinuxDevice string `yaml:"linux_device"` // e.g. "default" or "monitor"
}

// UIConfig controls UI behavior.
type UIConfig struct {
	AutoCheckUpdates bool `yaml:"auto_check_updates"`
}

// Default returns a Config with sensible defaults.
func Default() Config {
	return Config{
		Audio: AudioConfig{
			Device:              "auto",
			InputDeviceName:     "BlackHole",
			Threshold:           0.02,
			LevelThreshold:      0.02,
			SilenceSeconds:      60,
			SilenceSplitSeconds: 60,
			SilenceHysteresis:   0.005,
			HysteresisRatio:     0.6,
			ActivationMs:        500,
			SampleRate:          44100,
			Channels:            2,
			FormatProfile:       "high",
		},
		Session: SessionConfig{
			MinSessionSeconds:               5,
			KeepSingleSessionWhileMicActive: false,
		},
		Output: OutputConfig{
			Dir:               "~/Recordings/Memofy",
			Directory:         "~/Recordings/Memofy",
			WriteMetadataJSON: true,
		},
		Monitoring: MonitoringConfig{
			DetectZoom:                      true,
			DetectTeams:                     true,
			DetectMicUsage:                  true,
			KeepSingleSessionWhileMicActive: true,
			PollIntervalMs:                  5000,
		},
		Logging: LoggingConfig{
			File:  "~/.local/share/memofy/memofy.log",
			Level: "info",
		},
		Platform: PlatformConfig{
			MacOSDevice: "BlackHole",
			LinuxDevice: "default",
		},
		UI: UIConfig{
			AutoCheckUpdates: true,
		},
	}
}

// DefaultConfigPath returns the standard config file location.
func DefaultConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "memofy", "config.yaml")
}

// LoadOrDefault tries to load the config from the standard path.
// Returns defaults if the file doesn't exist.
func LoadOrDefault() Config {
	path := DefaultConfigPath()
	cfg, err := Load(path)
	if err != nil {
		d := Default()
		d.Output.Dir = ResolvePath(d.Output.Dir)
		d.Output.Directory = d.Output.Dir
		return d
	}
	return cfg
}

// Load reads a YAML config file. Returns defaults with the file's overrides applied.
func Load(path string) (Config, error) {
	cfg := Default()
	path = ResolvePath(path)

	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("read config: %w", err)
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parse config: %w", err)
	}

	// Resolve aliases — if the alias field was set, use it for the primary
	if cfg.Audio.InputDeviceName != "" && cfg.Audio.Device == "auto" {
		cfg.Audio.Device = cfg.Audio.InputDeviceName
	}
	if cfg.Audio.Device != "auto" && cfg.Audio.InputDeviceName == "" {
		cfg.Audio.InputDeviceName = cfg.Audio.Device
	}
	if cfg.Audio.LevelThreshold > 0 && cfg.Audio.Threshold == 0.02 {
		cfg.Audio.Threshold = cfg.Audio.LevelThreshold
	}
	if cfg.Audio.Threshold != 0.02 && cfg.Audio.LevelThreshold == 0.02 {
		cfg.Audio.LevelThreshold = cfg.Audio.Threshold
	}
	if cfg.Audio.SilenceSplitSeconds > 0 && cfg.Audio.SilenceSeconds == 60 {
		cfg.Audio.SilenceSeconds = cfg.Audio.SilenceSplitSeconds
	}
	if cfg.Audio.SilenceSeconds != 60 && cfg.Audio.SilenceSplitSeconds == 60 {
		cfg.Audio.SilenceSplitSeconds = cfg.Audio.SilenceSeconds
	}
	if cfg.Output.Directory != "" && cfg.Output.Dir == "~/Recordings/Memofy" {
		cfg.Output.Dir = cfg.Output.Directory
	}
	if cfg.Output.Dir != "" && cfg.Output.Directory == "" {
		cfg.Output.Directory = cfg.Output.Dir
	}

	cfg.Output.Dir = ResolvePath(cfg.Output.Dir)
	cfg.Output.Directory = cfg.Output.Dir

	if err := cfg.Validate(); err != nil {
		return cfg, err
	}

	return cfg, nil
}

// LoadConfig loads config from the given path, or defaults if path is empty.
func LoadConfig(path string) (*Config, error) {
	if path == "" {
		path = DefaultConfigPath()
	}
	cfg, err := Load(path)
	if err != nil {
		// If file not found, return defaults
		if os.IsNotExist(err) || strings.Contains(err.Error(), "read config") {
			d := Default()
			d.Output.Dir = ResolvePath(d.Output.Dir)
			d.Output.Directory = d.Output.Dir
			return &d, nil
		}
		return nil, err
	}
	return &cfg, nil
}

// Validate checks config values for validity.
func (c *Config) Validate() error {
	if c.Audio.Threshold <= 0 || c.Audio.Threshold >= 1.0 {
		return fmt.Errorf("audio.threshold must be between 0 and 1 (got %f)", c.Audio.Threshold)
	}
	if c.Audio.SilenceSeconds < 1 {
		return fmt.Errorf("audio.silence_seconds must be >= 1 (got %d)", c.Audio.SilenceSeconds)
	}
	if c.Audio.SampleRate <= 0 {
		c.Audio.SampleRate = 44100
	}
	if c.Audio.Channels <= 0 {
		c.Audio.Channels = 2
	}
	return nil
}

// Save writes the config to the given path as YAML.
func (c *Config) Save(path string) error {
	path = ResolvePath(path)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

// ResolvePath expands ~ to the user's home directory and environment variables.
func ResolvePath(path string) string {
	if path == "" {
		return path
	}
	// Expand environment variables
	path = os.ExpandEnv(path)
	// Expand ~
	if len(path) > 0 && path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[1:])
	}
	return path
}
