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
