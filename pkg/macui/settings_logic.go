// Package macui provides macOS menu bar UI for Memofy.
package macui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/tiroq/memofy/internal/config"
)

// SettingsFields holds raw form values captured from the UI.
// Decoupled from AppKit so it can be used in pure unit tests.
type SettingsFields struct {
	Device         string
	Threshold      string
	ActivationMs   string
	SilenceSeconds string
	OutputDir      string

	DetectZoom                      bool
	DetectTeams                     bool
	DetectMicUsage                  bool
	KeepSingleSessionWhileMicActive bool

	AutoCheckUpdates bool
	LogLevel         string
}

// FieldsFromConfig extracts UI form field values from a Config.
func FieldsFromConfig(cfg config.Config) SettingsFields {
	return SettingsFields{
		Device:                          cfg.Audio.Device,
		Threshold:                       fmt.Sprintf("%.4f", cfg.Audio.Threshold),
		ActivationMs:                    strconv.Itoa(cfg.Audio.ActivationMs),
		SilenceSeconds:                  strconv.Itoa(cfg.Audio.SilenceSeconds),
		OutputDir:                       cfg.Output.Dir,
		DetectZoom:                      cfg.Monitoring.DetectZoom,
		DetectTeams:                     cfg.Monitoring.DetectTeams,
		DetectMicUsage:                  cfg.Monitoring.DetectMicUsage,
		KeepSingleSessionWhileMicActive: cfg.Monitoring.KeepSingleSessionWhileMicActive,
		AutoCheckUpdates:                cfg.UI.AutoCheckUpdates,
		LogLevel:                        cfg.Logging.Level,
	}
}

// BuildConfigFromFields validates and converts UI form fields into a Config.
// Only the fields exposed in the settings window are modified; other values
// are taken from the provided base config.
func BuildConfigFromFields(f SettingsFields, base config.Config) (config.Config, error) {
	cfg := base

	// Audio device
	device := strings.TrimSpace(f.Device)
	if device == "" {
		device = "auto"
	}
	cfg.Audio.Device = device

	// Threshold
	threshold, err := strconv.ParseFloat(f.Threshold, 64)
	if err != nil {
		return cfg, fmt.Errorf("threshold must be a number (got %q)", f.Threshold)
	}
	if threshold <= 0 || threshold >= 1.0 {
		return cfg, fmt.Errorf("threshold must be between 0 and 1 (got %f)", threshold)
	}
	cfg.Audio.Threshold = threshold

	// Activation window
	activationMs, err := strconv.Atoi(f.ActivationMs)
	if err != nil {
		return cfg, fmt.Errorf("activation_ms must be a number (got %q)", f.ActivationMs)
	}
	if activationMs < 0 || activationMs > 5000 {
		return cfg, fmt.Errorf("activation_ms must be between 0 and 5000 (got %d)", activationMs)
	}
	cfg.Audio.ActivationMs = activationMs

	// Silence split
	silenceSec, err := strconv.Atoi(f.SilenceSeconds)
	if err != nil {
		return cfg, fmt.Errorf("silence_seconds must be a number (got %q)", f.SilenceSeconds)
	}
	if silenceSec < 1 {
		return cfg, fmt.Errorf("silence_seconds must be >= 1 (got %d)", silenceSec)
	}
	cfg.Audio.SilenceSeconds = silenceSec

	// Output
	dir := strings.TrimSpace(f.OutputDir)
	if dir == "" {
		return cfg, fmt.Errorf("output directory must not be empty")
	}
	cfg.Output.Dir = dir

	// Monitoring
	cfg.Monitoring.DetectZoom = f.DetectZoom
	cfg.Monitoring.DetectTeams = f.DetectTeams
	cfg.Monitoring.DetectMicUsage = f.DetectMicUsage
	cfg.Monitoring.KeepSingleSessionWhileMicActive = f.KeepSingleSessionWhileMicActive

	// UI
	cfg.UI.AutoCheckUpdates = f.AutoCheckUpdates

	// Logging
	level := strings.TrimSpace(strings.ToLower(f.LogLevel))
	if level == "" {
		level = "info"
	}
	cfg.Logging.Level = level

	return cfg, nil
}

// ParseCSVField splits a comma-separated string into trimmed, non-empty values.
func ParseCSVField(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// FormatDuration formats a duration for display.
func FormatDuration(seconds float64) string {
	if seconds >= 3600 {
		return fmt.Sprintf("%.1fh", seconds/3600)
	}
	if seconds >= 60 {
		return fmt.Sprintf("%.1fm", seconds/60)
	}
	return fmt.Sprintf("%.0fs", seconds)
}
