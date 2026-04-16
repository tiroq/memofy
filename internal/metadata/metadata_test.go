package metadata

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

func TestWrite(t *testing.T) {
	dir := t.TempDir()
	wavPath := dir + "/2026-02-12_143015_audio_high.wav"
	os.WriteFile(wavPath, []byte("fake"), 0644)

	meta := Recording{
		StartedAt:           time.Date(2026, 2, 12, 14, 30, 15, 0, time.UTC),
		EndedAt:             time.Date(2026, 2, 12, 15, 0, 15, 0, time.UTC),
		MicActive:           true,
		ZoomRunning:         true,
		TeamsRunning:        false,
		Platform:            "darwin",
		DeviceName:          "BlackHole 2ch",
		FormatProfile:       "high",
		Container:           "m4a",
		Codec:               "aac",
		SampleRate:          32000,
		Channels:            1,
		BitrateKbps:         64,
		Threshold:           0.02,
		SilenceSplitSeconds: 60,
		SplitReason:         "silence_threshold",
		AppVersion:          "0.2.0",
	}

	if err := Write(wavPath, meta); err != nil {
		t.Fatalf("Write: %v", err)
	}

	jsonPath := dir + "/2026-02-12_143015_audio_high.json"
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("read json: %v", err)
	}

	var got Recording
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.DurationSecs != 1800 {
		t.Errorf("duration: got %f, want 1800", got.DurationSecs)
	}
	if !got.MicActive {
		t.Error("mic_active should be true")
	}
	if !got.ZoomRunning {
		t.Error("zoom_running should be true")
	}
	if got.Platform != "darwin" {
		t.Errorf("platform: got %s, want darwin", got.Platform)
	}
	if got.DeviceName != "BlackHole 2ch" {
		t.Errorf("device_name: got %s, want BlackHole 2ch", got.DeviceName)
	}
	if got.FormatProfile != "high" {
		t.Errorf("format_profile: got %s, want high", got.FormatProfile)
	}
	if got.Container != "m4a" {
		t.Errorf("container: got %s, want m4a", got.Container)
	}
	if got.Codec != "aac" {
		t.Errorf("codec: got %s, want aac", got.Codec)
	}
	if got.SampleRate != 32000 {
		t.Errorf("sample_rate: got %d, want 32000", got.SampleRate)
	}
	if got.BitrateKbps != 64 {
		t.Errorf("bitrate_kbps: got %d, want 64", got.BitrateKbps)
	}
	if got.SplitReason != "silence_threshold" {
		t.Errorf("split_reason: got %s, want silence_threshold", got.SplitReason)
	}
	if got.SessionID == "" {
		t.Error("session_id should not be empty")
	}
	if got.AppVersion != "0.2.0" {
		t.Errorf("version: got %s, want 0.2.0", got.AppVersion)
	}
}

func TestWriteAutocomputedFields(t *testing.T) {
	dir := t.TempDir()
	wavPath := dir + "/test_auto.wav"
	os.WriteFile(wavPath, []byte("fake"), 0644)

	start := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	end := time.Date(2026, 1, 1, 10, 5, 0, 0, time.UTC)

	// No duration, no platform, no session_id, no threshold — should be autocomputed
	meta := Recording{
		StartedAt: start,
		EndedAt:   end,
	}

	if err := Write(wavPath, meta); err != nil {
		t.Fatalf("Write: %v", err)
	}

	jsonPath := dir + "/test_auto.json"
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("read json: %v", err)
	}

	var got Recording
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.DurationSecs != 300 {
		t.Errorf("duration: got %f, want 300", got.DurationSecs)
	}
	if got.Platform == "" {
		t.Error("platform should be autocomputed")
	}
	if got.SessionID == "" {
		t.Error("session_id should be autocomputed")
	}
	if got.Threshold == 0 {
		t.Error("threshold should be autocomputed from default")
	}
}

func TestWriteM4AExtension(t *testing.T) {
	dir := t.TempDir()
	m4aPath := dir + "/recording.m4a"
	os.WriteFile(m4aPath, []byte("fake"), 0644)

	meta := Recording{
		StartedAt: time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC),
		EndedAt:   time.Date(2026, 1, 1, 10, 1, 0, 0, time.UTC),
		Container: "m4a",
	}

	if err := Write(m4aPath, meta); err != nil {
		t.Fatalf("Write: %v", err)
	}

	// Should create .json alongside .m4a
	jsonPath := dir + "/recording.json"
	if _, err := os.Stat(jsonPath); os.IsNotExist(err) {
		t.Error("expected .json file alongside .m4a")
	}
}

func TestWriteErrorOnInvalidDir(t *testing.T) {
	meta := Recording{
		StartedAt: time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC),
		EndedAt:   time.Date(2026, 1, 1, 10, 1, 0, 0, time.UTC),
	}

	err := Write("/nonexistent/dir/recording.wav", meta)
	if err == nil {
		t.Error("expected error writing to nonexistent directory")
	}
}

func TestWritePreexistingSessionID(t *testing.T) {
	dir := t.TempDir()
	wavPath := dir + "/test_sid.wav"
	os.WriteFile(wavPath, []byte("fake"), 0644)

	meta := Recording{
		SessionID: "custom-session-123",
		StartedAt: time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC),
		EndedAt:   time.Date(2026, 1, 1, 10, 1, 0, 0, time.UTC),
	}
	if err := Write(wavPath, meta); err != nil {
		t.Fatalf("Write: %v", err)
	}

	data, _ := os.ReadFile(dir + "/test_sid.json")
	var got Recording
	json.Unmarshal(data, &got)
	if got.SessionID != "custom-session-123" {
		t.Errorf("session_id: got %s, want custom-session-123", got.SessionID)
	}
}

// --- SessionDiagnostics tests ---

func TestSessionDiagnostics_RecordRMS(t *testing.T) {
	var d SessionDiagnostics

	d.RecordRMS(0.01)
	d.RecordRMS(0.05)
	d.RecordRMS(0.03)

	if d.RMSPeak != 0.05 {
		t.Errorf("RMSPeak: got %f, want 0.05", d.RMSPeak)
	}
	if d.RMSCount != 3 {
		t.Errorf("RMSCount: got %d, want 3", d.RMSCount)
	}
	if d.RMSSum != 0.09 {
		t.Errorf("RMSSum: got %f, want 0.09", d.RMSSum)
	}
}

func TestSessionDiagnostics_Finalize_HasMeaningfulAudio(t *testing.T) {
	tests := []struct {
		name           string
		framesWritten  int64
		bytesWritten   int64
		rmsPeak        float64
		minRMS         float64
		wantMeaningful bool
	}{
		{
			name:           "all valid",
			framesWritten:  1000,
			bytesWritten:   100,
			rmsPeak:        0.05,
			minRMS:         0.01,
			wantMeaningful: true,
		},
		{
			name:           "no frames written",
			framesWritten:  0,
			bytesWritten:   100,
			rmsPeak:        0.05,
			minRMS:         0.01,
			wantMeaningful: false,
		},
		{
			name:           "only WAV header bytes",
			framesWritten:  1,
			bytesWritten:   44,
			rmsPeak:        0.05,
			minRMS:         0.01,
			wantMeaningful: false,
		},
		{
			name:           "RMS below min",
			framesWritten:  1000,
			bytesWritten:   100,
			rmsPeak:        0.005,
			minRMS:         0.01,
			wantMeaningful: false,
		},
		{
			name:           "exactly at boundary bytes",
			framesWritten:  1,
			bytesWritten:   45,
			rmsPeak:        0.01,
			minRMS:         0.01,
			wantMeaningful: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := SessionDiagnostics{
				FramesWritten: tt.framesWritten,
				BytesWritten:  tt.bytesWritten,
				RMSPeak:       tt.rmsPeak,
			}
			d.Finalize(tt.minRMS)
			if d.HasMeaningfulAudio != tt.wantMeaningful {
				t.Errorf("HasMeaningfulAudio: got %v, want %v", d.HasMeaningfulAudio, tt.wantMeaningful)
			}
		})
	}
}

func TestSessionDiagnostics_Finalize_Average(t *testing.T) {
	var d SessionDiagnostics
	d.RecordRMS(0.02)
	d.RecordRMS(0.04)
	d.RecordRMS(0.06)
	d.FramesWritten = 100
	d.BytesWritten = 200

	d.Finalize(0.01)

	want := 0.04 // (0.02 + 0.04 + 0.06) / 3
	if d.RMSAverage < want-0.001 || d.RMSAverage > want+0.001 {
		t.Errorf("RMSAverage: got %f, want ~%f", d.RMSAverage, want)
	}
}

func TestSessionDiagnostics_Finalize_ZeroCount(t *testing.T) {
	var d SessionDiagnostics
	d.Finalize(0.01)
	if d.RMSAverage != 0 {
		t.Errorf("RMSAverage with zero count: got %f, want 0", d.RMSAverage)
	}
}

func TestFinalizationReasonConstants(t *testing.T) {
	// Verify all reason constants are distinct non-empty strings.
	reasons := []FinalizationReason{
		ReasonSilenceTimeout,
		ReasonManualStop,
		ReasonShutdown,
		ReasonDeviceLost,
		ReasonError,
		ReasonDiscardedShort,
		ReasonDiscardedEmpty,
	}
	seen := make(map[FinalizationReason]bool)
	for _, r := range reasons {
		if r == "" {
			t.Errorf("empty finalization reason")
		}
		if seen[r] {
			t.Errorf("duplicate finalization reason: %s", r)
		}
		seen[r] = true
	}
}

func TestWriteDiagnosticsFields(t *testing.T) {
	dir := t.TempDir()
	wavPath := dir + "/diag_test.wav"
	os.WriteFile(wavPath, []byte("fake"), 0644)

	meta := Recording{
		StartedAt:          time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC),
		EndedAt:            time.Date(2026, 1, 1, 10, 5, 0, 0, time.UTC),
		FinalizationReason: ReasonSilenceTimeout,
		FramesReceived:     44100,
		FramesWritten:      40000,
		BytesWritten:       80000,
		RMSPeak:            0.15,
		RMSAverage:         0.03,
		HasMeaningfulAudio: true,
	}

	if err := Write(wavPath, meta); err != nil {
		t.Fatalf("Write: %v", err)
	}

	data, _ := os.ReadFile(dir + "/diag_test.json")
	var got Recording
	json.Unmarshal(data, &got)

	if got.FinalizationReason != ReasonSilenceTimeout {
		t.Errorf("finalization_reason: got %s, want %s", got.FinalizationReason, ReasonSilenceTimeout)
	}
	if got.FramesReceived != 44100 {
		t.Errorf("frames_received: got %d, want 44100", got.FramesReceived)
	}
	if got.FramesWritten != 40000 {
		t.Errorf("frames_written: got %d, want 40000", got.FramesWritten)
	}
	if got.BytesWritten != 80000 {
		t.Errorf("bytes_written: got %d, want 80000", got.BytesWritten)
	}
	if got.RMSPeak != 0.15 {
		t.Errorf("rms_peak: got %f, want 0.15", got.RMSPeak)
	}
	if got.RMSAverage != 0.03 {
		t.Errorf("rms_average: got %f, want 0.03", got.RMSAverage)
	}
	if !got.HasMeaningfulAudio {
		t.Error("has_meaningful_audio: got false, want true")
	}
}
