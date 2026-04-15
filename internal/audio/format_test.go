package audio

import "testing"

func TestGetFormatSpec(t *testing.T) {
	tests := []struct {
		profile   string
		wantCodec string
		wantRate  int
		wantBps   int
		wantExt   string
	}{
		{"high", "aac", 32000, 64, ".m4a"},
		{"balanced", "aac", 24000, 48, ".m4a"},
		{"lightweight", "aac", 16000, 32, ".m4a"},
		{"wav", "pcm_s16le", 44100, 0, ".wav"},
		{"unknown", "aac", 32000, 64, ".m4a"}, // fallback to high
	}
	for _, tt := range tests {
		t.Run(tt.profile, func(t *testing.T) {
			spec := GetFormatSpec(tt.profile)
			if spec.Codec != tt.wantCodec {
				t.Errorf("codec = %q, want %q", spec.Codec, tt.wantCodec)
			}
			if spec.SampleRate != tt.wantRate {
				t.Errorf("sample rate = %d, want %d", spec.SampleRate, tt.wantRate)
			}
			if spec.BitrateKbps != tt.wantBps {
				t.Errorf("bitrate = %d, want %d", spec.BitrateKbps, tt.wantBps)
			}
			if ext := spec.FileExtension(); ext != tt.wantExt {
				t.Errorf("extension = %q, want %q", ext, tt.wantExt)
			}
		})
	}
}

func TestIsValidProfile(t *testing.T) {
	if !IsValidProfile("high") {
		t.Error("expected 'high' to be valid")
	}
	if !IsValidProfile("balanced") {
		t.Error("expected 'balanced' to be valid")
	}
	if IsValidProfile("ultra") {
		t.Error("expected 'ultra' to be invalid")
	}
}

func TestValidProfiles(t *testing.T) {
	profiles := ValidProfiles()
	if len(profiles) != 4 {
		t.Errorf("expected 4 profiles, got %d", len(profiles))
	}
}
