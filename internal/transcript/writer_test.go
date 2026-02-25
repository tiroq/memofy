package transcript

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tiroq/memofy/internal/asr"
)

// T027: Tests for transcript file writers. FR-013

func sampleTranscript() *asr.Transcript {
	return &asr.Transcript{
		Segments: []asr.Segment{
			{Start: 0, End: 5*time.Second + 230*time.Millisecond, Text: "Hello, welcome to the meeting."},
			{Start: 5*time.Second + 500*time.Millisecond, End: 10*time.Second + 100*time.Millisecond, Text: "Let's discuss the agenda."},
		},
		Language: "en",
		Duration: 10*time.Second + 100*time.Millisecond,
		Model:    "small",
		Backend:  "remote_whisper_api",
	}
}

func tmpPath(t *testing.T, ext string) string {
	t.Helper()
	dir := t.TempDir()
	return filepath.Join(dir, "transcript"+ext)
}

func TestWriteText(t *testing.T) {
	path := tmpPath(t, ".txt")
	tr := sampleTranscript()

	if err := WriteText(path, tr); err != nil {
		t.Fatalf("WriteText: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	got := string(data)

	// Check each segment line.
	if !strings.Contains(got, "[00:00:00] Hello, welcome to the meeting.") {
		t.Errorf("missing first segment; got:\n%s", got)
	}
	if !strings.Contains(got, "[00:00:05] Let's discuss the agenda.") {
		t.Errorf("missing second segment; got:\n%s", got)
	}

	// Two lines total.
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 lines, got %d", len(lines))
	}
}

func TestWriteSRT(t *testing.T) {
	path := tmpPath(t, ".srt")
	tr := sampleTranscript()

	if err := WriteSRT(path, tr); err != nil {
		t.Fatalf("WriteSRT: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	got := string(data)

	// Verify segment numbering starts at 1.
	if !strings.HasPrefix(got, "1\n") {
		t.Errorf("SRT should start with segment number 1; got:\n%s", got)
	}

	// Verify SRT timestamp format (comma separator).
	if !strings.Contains(got, "00:00:00,000 --> 00:00:05,230") {
		t.Errorf("missing first SRT timestamp; got:\n%s", got)
	}
	if !strings.Contains(got, "00:00:05,500 --> 00:00:10,100") {
		t.Errorf("missing second SRT timestamp; got:\n%s", got)
	}

	// Verify both texts present.
	if !strings.Contains(got, "Hello, welcome to the meeting.") {
		t.Errorf("missing first segment text")
	}
	if !strings.Contains(got, "Let's discuss the agenda.") {
		t.Errorf("missing second segment text")
	}

	// Verify second segment number.
	if !strings.Contains(got, "\n2\n") {
		t.Errorf("missing segment number 2; got:\n%s", got)
	}
}

func TestWriteVTT(t *testing.T) {
	path := tmpPath(t, ".vtt")
	tr := sampleTranscript()

	if err := WriteVTT(path, tr); err != nil {
		t.Fatalf("WriteVTT: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	got := string(data)

	// Must start with WEBVTT header.
	if !strings.HasPrefix(got, "WEBVTT\n") {
		t.Errorf("VTT should start with WEBVTT header; got:\n%s", got)
	}

	// Verify VTT timestamp format (period separator).
	if !strings.Contains(got, "00:00:00.000 --> 00:00:05.230") {
		t.Errorf("missing first VTT timestamp; got:\n%s", got)
	}
	if !strings.Contains(got, "00:00:05.500 --> 00:00:10.100") {
		t.Errorf("missing second VTT timestamp; got:\n%s", got)
	}
}

func TestWriteAll(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "meeting")
	tr := sampleTranscript()

	err := WriteAll(base, tr, []string{"txt", "srt", "vtt"})
	if err != nil {
		t.Fatalf("WriteAll: %v", err)
	}

	// All three files should exist.
	for _, ext := range []string{".txt", ".srt", ".vtt"} {
		path := base + ext
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected file %s to exist: %v", path, err)
		}
	}
}

func TestWriteAll_DefaultTxt(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "meeting")
	tr := sampleTranscript()

	// nil formats should default to txt.
	if err := WriteAll(base, tr, nil); err != nil {
		t.Fatalf("WriteAll with nil formats: %v", err)
	}
	if _, err := os.Stat(base + ".txt"); err != nil {
		t.Errorf("expected .txt file: %v", err)
	}

	// Empty slice should also default to txt.
	dir2 := t.TempDir()
	base2 := filepath.Join(dir2, "meeting2")
	if err := WriteAll(base2, tr, []string{}); err != nil {
		t.Fatalf("WriteAll with empty formats: %v", err)
	}
	if _, err := os.Stat(base2 + ".txt"); err != nil {
		t.Errorf("expected .txt file: %v", err)
	}
}

func TestWriteAll_UnknownFormat(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "meeting")
	tr := sampleTranscript()

	err := WriteAll(base, tr, []string{"txt", "json"})
	if err == nil {
		t.Fatal("expected error for unknown format")
	}
	if !strings.Contains(err.Error(), `unknown format "json"`) {
		t.Errorf("error should mention unknown format; got: %v", err)
	}
	// txt should still have been written.
	if _, err := os.Stat(base + ".txt"); err != nil {
		t.Errorf("expected .txt file despite json error: %v", err)
	}
}

func TestWriteText_EmptyTranscript(t *testing.T) {
	path := tmpPath(t, ".txt")
	tr := &asr.Transcript{}

	if err := WriteText(path, tr); err != nil {
		t.Fatalf("WriteText empty: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if len(data) != 0 {
		t.Errorf("expected empty file for empty transcript, got %d bytes: %q", len(data), string(data))
	}
}

func TestWriteSRT_EmptyTranscript(t *testing.T) {
	path := tmpPath(t, ".srt")
	tr := &asr.Transcript{}

	if err := WriteSRT(path, tr); err != nil {
		t.Fatalf("WriteSRT empty: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if len(data) != 0 {
		t.Errorf("expected empty file for empty SRT, got %d bytes", len(data))
	}
}

func TestWriteVTT_EmptyTranscript(t *testing.T) {
	path := tmpPath(t, ".vtt")
	tr := &asr.Transcript{}

	if err := WriteVTT(path, tr); err != nil {
		t.Fatalf("WriteVTT empty: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	// VTT should still have the header even with no segments.
	got := string(data)
	if got != "WEBVTT\n" {
		t.Errorf("expected only WEBVTT header for empty transcript, got: %q", got)
	}
}

func TestWriteText_SingleSegment(t *testing.T) {
	path := tmpPath(t, ".txt")
	tr := &asr.Transcript{
		Segments: []asr.Segment{
			{Start: 0, End: 3 * time.Second, Text: "Single line."},
		},
	}

	if err := WriteText(path, tr); err != nil {
		t.Fatalf("WriteText: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	expected := "[00:00:00] Single line.\n"
	if string(data) != expected {
		t.Errorf("got %q, want %q", string(data), expected)
	}
}

func TestWriteSRT_Unicode(t *testing.T) {
	path := tmpPath(t, ".srt")
	tr := &asr.Transcript{
		Segments: []asr.Segment{
			{Start: 0, End: 2 * time.Second, Text: "こんにちは、会議へようこそ。"},
			{Start: 2 * time.Second, End: 4 * time.Second, Text: "Ñoño café résumé naïve"},
			{Start: 4 * time.Second, End: 6 * time.Second, Text: "Привет мир 🌍"},
		},
	}

	if err := WriteSRT(path, tr); err != nil {
		t.Fatalf("WriteSRT: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	got := string(data)

	// Verify all Unicode text is preserved.
	for _, seg := range tr.Segments {
		if !strings.Contains(got, seg.Text) {
			t.Errorf("missing Unicode text %q in output", seg.Text)
		}
	}
}

func TestFormatSRTTimestamp(t *testing.T) {
	tests := []struct {
		name string
		d    time.Duration
		want string
	}{
		{"zero", 0, "00:00:00,000"},
		{"one_second", time.Second, "00:00:01,000"},
		{"one_minute", time.Minute, "00:01:00,000"},
		{"one_hour", time.Hour, "01:00:00,000"},
		{"mixed", 1*time.Hour + 23*time.Minute + 45*time.Second + 678*time.Millisecond, "01:23:45,678"},
		{"millis_only", 999 * time.Millisecond, "00:00:00,999"},
		{"large_hours", 99*time.Hour + 59*time.Minute + 59*time.Second + 999*time.Millisecond, "99:59:59,999"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatSRTTimestamp(tt.d)
			if got != tt.want {
				t.Errorf("formatSRTTimestamp(%v) = %q, want %q", tt.d, got, tt.want)
			}
		})
	}
}

func TestFormatVTTTimestamp(t *testing.T) {
	tests := []struct {
		name string
		d    time.Duration
		want string
	}{
		{"zero", 0, "00:00:00.000"},
		{"one_second", time.Second, "00:00:01.000"},
		{"one_minute", time.Minute, "00:01:00.000"},
		{"one_hour", time.Hour, "01:00:00.000"},
		{"mixed", 1*time.Hour + 23*time.Minute + 45*time.Second + 678*time.Millisecond, "01:23:45.678"},
		{"millis_only", 999 * time.Millisecond, "00:00:00.999"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatVTTTimestamp(tt.d)
			if got != tt.want {
				t.Errorf("formatVTTTimestamp(%v) = %q, want %q", tt.d, got, tt.want)
			}
		})
	}
}

func TestWriteText_LongDuration(t *testing.T) {
	path := tmpPath(t, ".txt")
	tr := &asr.Transcript{
		Segments: []asr.Segment{
			{Start: 2*time.Hour + 30*time.Minute, End: 2*time.Hour + 30*time.Minute + 5*time.Second, Text: "Late segment."},
		},
	}

	if err := WriteText(path, tr); err != nil {
		t.Fatalf("WriteText: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	expected := "[02:30:00] Late segment.\n"
	if string(data) != expected {
		t.Errorf("got %q, want %q", string(data), expected)
	}
}

func TestAtomicWrite_CreatesParentDir(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "sub", "dir", "transcript.txt")
	tr := &asr.Transcript{
		Segments: []asr.Segment{
			{Start: 0, End: time.Second, Text: "Test."},
		},
	}

	if err := WriteText(nested, tr); err != nil {
		t.Fatalf("WriteText to nested path: %v", err)
	}
	if _, err := os.Stat(nested); err != nil {
		t.Errorf("expected nested file to exist: %v", err)
	}
}
