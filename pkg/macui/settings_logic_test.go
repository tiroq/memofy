package macui

import (
	"testing"

	"github.com/tiroq/memofy/internal/config"
)

func validFields() SettingsFields {
	return SettingsFields{
		Device:                          "auto",
		Threshold:                       "0.02",
		ActivationMs:                    "400",
		SilenceSeconds:                  "60",
		OutputDir:                       "~/Recordings/Memofy",
		DetectZoom:                      true,
		DetectTeams:                     true,
		DetectMicUsage:                  true,
		KeepSingleSessionWhileMicActive: true,
		AutoCheckUpdates:                true,
		LogLevel:                        "info",
	}
}

func TestParseCSVField_single(t *testing.T) {
	got := ParseCSVField("zoom.us")
	if len(got) != 1 || got[0] != "zoom.us" {
		t.Errorf("got %v, want [zoom.us]", got)
	}
}

func TestParseCSVField_multiple(t *testing.T) {
	got := ParseCSVField("zoom.us, CptHost, zoomusApp")
	want := []string{"zoom.us", "CptHost", "zoomusApp"}
	if len(got) != len(want) {
		t.Fatalf("len: got %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("[%d]: got %q, want %q", i, got[i], want[i])
		}
	}
}

func TestParseCSVField_empty(t *testing.T) {
	if got := ParseCSVField(""); got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestParseCSVField_whitespaceOnly(t *testing.T) {
	if got := ParseCSVField("   ,  ,  "); got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestBuildConfigFromFields_valid(t *testing.T) {
	f := validFields()
	base := config.Default()
	cfg, err := BuildConfigFromFields(f, base)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Audio.Device != "auto" {
		t.Errorf("device: got %s, want auto", cfg.Audio.Device)
	}
	if cfg.Audio.Threshold != 0.02 {
		t.Errorf("threshold: got %f, want 0.02", cfg.Audio.Threshold)
	}
	if cfg.Audio.ActivationMs != 400 {
		t.Errorf("activation_ms: got %d, want 400", cfg.Audio.ActivationMs)
	}
	if cfg.Audio.SilenceSeconds != 60 {
		t.Errorf("silence_seconds: got %d, want 60", cfg.Audio.SilenceSeconds)
	}
	if !cfg.UI.AutoCheckUpdates {
		t.Error("auto_check_updates should be true")
	}
}

func TestBuildConfigFromFields_invalidThreshold(t *testing.T) {
	f := validFields()
	f.Threshold = "1.5"
	_, err := BuildConfigFromFields(f, config.Default())
	if err == nil {
		t.Fatal("expected error for invalid threshold")
	}
}

func TestBuildConfigFromFields_nonNumericThreshold(t *testing.T) {
	f := validFields()
	f.Threshold = "abc"
	_, err := BuildConfigFromFields(f, config.Default())
	if err == nil {
		t.Fatal("expected error for non-numeric threshold")
	}
}

func TestBuildConfigFromFields_invalidSilence(t *testing.T) {
	f := validFields()
	f.SilenceSeconds = "0"
	_, err := BuildConfigFromFields(f, config.Default())
	if err == nil {
		t.Fatal("expected error for silence_seconds = 0")
	}
}

func TestBuildConfigFromFields_emptyOutputDir(t *testing.T) {
	f := validFields()
	f.OutputDir = ""
	_, err := BuildConfigFromFields(f, config.Default())
	if err == nil {
		t.Fatal("expected error for empty output dir")
	}
}

func TestFieldsFromConfig_roundtrip(t *testing.T) {
	cfg := config.Default()
	fields := FieldsFromConfig(cfg)
	rebuilt, err := BuildConfigFromFields(fields, cfg)
	if err != nil {
		t.Fatalf("roundtrip error: %v", err)
	}
	if rebuilt.Audio.Device != cfg.Audio.Device {
		t.Errorf("device roundtrip: got %s, want %s", rebuilt.Audio.Device, cfg.Audio.Device)
	}
	if rebuilt.Audio.Threshold != cfg.Audio.Threshold {
		t.Errorf("threshold roundtrip: got %f, want %f", rebuilt.Audio.Threshold, cfg.Audio.Threshold)
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		input float64
		want  string
	}{
		{30, "30s"},
		{90, "1.5m"},
		{3700, "1.0h"},
	}
	for _, tc := range tests {
		got := FormatDuration(tc.input)
		if got != tc.want {
			t.Errorf("FormatDuration(%f): got %s, want %s", tc.input, got, tc.want)
		}
	}
}
