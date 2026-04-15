package config

import (
	"os"
	"testing"
)

func TestDefault(t *testing.T) {
	cfg := Default()
	if cfg.Audio.Threshold != 0.02 {
		t.Errorf("threshold: got %f, want 0.02", cfg.Audio.Threshold)
	}
	if cfg.Audio.SilenceSeconds != 60 {
		t.Errorf("silence_seconds: got %d, want 60", cfg.Audio.SilenceSeconds)
	}
	if cfg.Audio.Device != "auto" {
		t.Errorf("device: got %s, want auto", cfg.Audio.Device)
	}
	if cfg.Audio.FormatProfile != "high" {
		t.Errorf("format_profile: got %s, want high", cfg.Audio.FormatProfile)
	}
}

func TestLoadValid(t *testing.T) {
	content := `
audio:
  device: "BlackHole 2ch"
  threshold: 0.05
  silence_seconds: 30
output:
  dir: /tmp/test-recordings
platform:
  macos_device: "BlackHole"
`
	tmp := t.TempDir() + "/config.yaml"
	if err := os.WriteFile(tmp, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(tmp)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Audio.Device != "BlackHole 2ch" {
		t.Errorf("device: got %s, want BlackHole 2ch", cfg.Audio.Device)
	}
	if cfg.Audio.Threshold != 0.05 {
		t.Errorf("threshold: got %f, want 0.05", cfg.Audio.Threshold)
	}
	if cfg.Audio.SilenceSeconds != 30 {
		t.Errorf("silence_seconds: got %d, want 30", cfg.Audio.SilenceSeconds)
	}
	if cfg.Output.Dir != "/tmp/test-recordings" {
		t.Errorf("dir: got %s, want /tmp/test-recordings", cfg.Output.Dir)
	}
}

func TestLoadInvalidThreshold(t *testing.T) {
	content := `
audio:
  threshold: 1.5
  silence_seconds: 60
`
	tmp := t.TempDir() + "/config.yaml"
	os.WriteFile(tmp, []byte(content), 0644)

	_, err := Load(tmp)
	if err == nil {
		t.Fatal("expected error for invalid threshold")
	}
}

func TestLoadMissing(t *testing.T) {
	_, err := Load("/nonexistent/config.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadOrDefaultMissing(t *testing.T) {
	cfg := LoadOrDefault()
	if cfg.Audio.Threshold != 0.02 {
		t.Errorf("threshold: got %f, want 0.02", cfg.Audio.Threshold)
	}
}

func TestResolvePath(t *testing.T) {
	home, _ := os.UserHomeDir()
	result := ResolvePath("~/test")
	if result != home+"/test" {
		t.Errorf("ResolvePath: got %s, want %s/test", result, home)
	}

	result = ResolvePath("/absolute/path")
	if result != "/absolute/path" {
		t.Errorf("ResolvePath absolute: got %s", result)
	}
}

func TestResolvePathEmpty(t *testing.T) {
	if got := ResolvePath(""); got != "" {
		t.Errorf("ResolvePath empty: got %q, want empty", got)
	}
}

func TestValidateDefaults(t *testing.T) {
	cfg := Default()
	if err := cfg.Validate(); err != nil {
		t.Errorf("default config should be valid: %v", err)
	}
}

func TestValidateThresholdZero(t *testing.T) {
	cfg := Default()
	cfg.Audio.Threshold = 0
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for threshold=0")
	}
}

func TestValidateThresholdOne(t *testing.T) {
	cfg := Default()
	cfg.Audio.Threshold = 1.0
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for threshold=1.0")
	}
}

func TestValidateSilenceZero(t *testing.T) {
	cfg := Default()
	cfg.Audio.SilenceSeconds = 0
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for silence_seconds=0")
	}
}

func TestValidateFixesSampleRate(t *testing.T) {
	cfg := Default()
	cfg.Audio.SampleRate = 0
	if err := cfg.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
	if cfg.Audio.SampleRate != 44100 {
		t.Errorf("sample_rate: got %d, want 44100", cfg.Audio.SampleRate)
	}
}

func TestValidateFixesChannels(t *testing.T) {
	cfg := Default()
	cfg.Audio.Channels = 0
	if err := cfg.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
	if cfg.Audio.Channels != 2 {
		t.Errorf("channels: got %d, want 2", cfg.Audio.Channels)
	}
}

func TestSaveAndLoad(t *testing.T) {
	cfg := Default()
	cfg.Audio.Threshold = 0.05
	cfg.Audio.SilenceSeconds = 30
	cfg.Output.Dir = "/tmp/test-save"
	cfg.Output.Directory = "/tmp/test-save"

	path := t.TempDir() + "/subdir/config.yaml"
	if err := cfg.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.Audio.Threshold != 0.05 {
		t.Errorf("threshold: got %f, want 0.05", loaded.Audio.Threshold)
	}
	if loaded.Audio.SilenceSeconds != 30 {
		t.Errorf("silence_seconds: got %d, want 30", loaded.Audio.SilenceSeconds)
	}
}

func TestLoadConfigEmpty(t *testing.T) {
	cfg, err := LoadConfig("")
	if err != nil {
		// File might not exist, but should return defaults
		if cfg == nil {
			t.Fatal("LoadConfig with empty path should return non-nil config")
		}
	}
	if cfg != nil && cfg.Audio.Threshold != 0.02 {
		t.Errorf("threshold: got %f, want 0.02", cfg.Audio.Threshold)
	}
}

func TestLoadConfigValid(t *testing.T) {
	content := `
audio:
  threshold: 0.03
  silence_seconds: 45
output:
  dir: /tmp/test-loadconfig
`
	path := t.TempDir() + "/config.yaml"
	os.WriteFile(path, []byte(content), 0644)

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Audio.Threshold != 0.03 {
		t.Errorf("threshold: got %f, want 0.03", cfg.Audio.Threshold)
	}
}

func TestLoadConfigInvalid(t *testing.T) {
	content := `audio: [not valid yaml`
	path := t.TempDir() + "/bad.yaml"
	os.WriteFile(path, []byte(content), 0644)

	_, err := LoadConfig(path)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestLoadAliasResolution(t *testing.T) {
	content := `
audio:
  level_threshold: 0.08
  silence_split_seconds: 90
output:
  directory: /tmp/alias-test
`
	path := t.TempDir() + "/config.yaml"
	os.WriteFile(path, []byte(content), 0644)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Audio.Threshold != 0.08 {
		t.Errorf("threshold (from level_threshold): got %f, want 0.08", cfg.Audio.Threshold)
	}
	if cfg.Audio.SilenceSeconds != 90 {
		t.Errorf("silence_seconds (from silence_split_seconds): got %d, want 90", cfg.Audio.SilenceSeconds)
	}
}

func TestDefaultConfigPath(t *testing.T) {
	path := DefaultConfigPath()
	if path == "" {
		t.Error("DefaultConfigPath should not be empty")
	}
}
