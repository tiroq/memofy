// T029: FR-013 — tests for local whisper CLI backend.
package localwhisper

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/tiroq/memofy/internal/asr"
)

// writeFakeScript creates a shell script in the temp dir that outputs the given
// content to stdout and exits with the given code.
func writeFakeScript(t *testing.T, dir, name, script string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write fake script: %v", err)
	}
	return path
}

func TestName(t *testing.T) {
	b := NewBackend(Config{})
	if b.Name() != "local_whisper" {
		t.Errorf("expected name %q, got %q", "local_whisper", b.Name())
	}
}

func TestTranscribeFile_Success(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell scripts not supported on windows")
	}

	dir := t.TempDir()

	// Fake whisper binary that outputs valid JSON
	jsonOutput := `{"segments": [{"start": 0.0, "end": 5.2, "text": "Hello world", "score": 0.95}, {"start": 5.2, "end": 10.0, "text": "Second segment", "score": 0.88}], "language": "en"}`
	script := "#!/bin/sh\necho '" + jsonOutput + "'\n"
	binPath := writeFakeScript(t, dir, "whisper", script)

	// Create a fake input file
	inputFile := filepath.Join(dir, "test.wav")
	if err := os.WriteFile(inputFile, []byte("fake audio"), 0644); err != nil {
		t.Fatal(err)
	}

	b := NewBackend(Config{
		BinaryPath:     binPath,
		Model:          "small",
		TimeoutSeconds: 10,
	})

	transcript, err := b.TranscribeFile(inputFile, asr.TranscribeOptions{
		Language: "en",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if transcript.Backend != "local_whisper" {
		t.Errorf("expected backend %q, got %q", "local_whisper", transcript.Backend)
	}
	if transcript.Language != "en" {
		t.Errorf("expected language %q, got %q", "en", transcript.Language)
	}
	if transcript.Model != "small" {
		t.Errorf("expected model %q, got %q", "small", transcript.Model)
	}
	if len(transcript.Segments) != 2 {
		t.Fatalf("expected 2 segments, got %d", len(transcript.Segments))
	}

	seg := transcript.Segments[0]
	if seg.Text != "Hello world" {
		t.Errorf("expected text %q, got %q", "Hello world", seg.Text)
	}
	if seg.Score != 0.95 {
		t.Errorf("expected score 0.95, got %f", seg.Score)
	}
	expectedStart := time.Duration(0)
	if seg.Start != expectedStart {
		t.Errorf("expected start %v, got %v", expectedStart, seg.Start)
	}
	expectedEnd := time.Duration(5.2 * float64(time.Second))
	if seg.End != expectedEnd {
		t.Errorf("expected end %v, got %v", expectedEnd, seg.End)
	}

	// Duration should be last segment's end
	expectedDuration := time.Duration(10.0 * float64(time.Second))
	if transcript.Duration != expectedDuration {
		t.Errorf("expected duration %v, got %v", expectedDuration, transcript.Duration)
	}
}

func TestTranscribeFile_BinaryNotFound(t *testing.T) {
	b := NewBackend(Config{
		BinaryPath:     "/nonexistent/whisper-binary",
		TimeoutSeconds: 5,
	})

	_, err := b.TranscribeFile("/some/file.wav", asr.TranscribeOptions{})
	if err == nil {
		t.Fatal("expected error for missing binary")
	}
	if !strings.Contains(err.Error(), "binary not found") {
		t.Errorf("expected 'binary not found' in error, got: %v", err)
	}
}

func TestTranscribeFile_Timeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell scripts not supported on windows")
	}

	dir := t.TempDir()

	// Fake binary that sleeps longer than the timeout
	script := "#!/bin/sh\nsleep 30\n"
	binPath := writeFakeScript(t, dir, "whisper-slow", script)

	inputFile := filepath.Join(dir, "test.wav")
	if err := os.WriteFile(inputFile, []byte("fake audio"), 0644); err != nil {
		t.Fatal(err)
	}

	b := NewBackend(Config{
		BinaryPath:     binPath,
		TimeoutSeconds: 1, // 1 second timeout
	})

	start := time.Now()
	_, err := b.TranscribeFile(inputFile, asr.TranscribeOptions{})
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected error for timed out subprocess")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Errorf("expected 'timed out' in error, got: %v", err)
	}
	// Should have been killed well before 30 seconds
	if elapsed > 5*time.Second {
		t.Errorf("timeout took too long: %v (expected ~1s)", elapsed)
	}
}

func TestTranscribeFile_OptsModelOverridesConfig(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell scripts not supported on windows")
	}

	dir := t.TempDir()

	jsonOutput := `{"segments": [], "language": "en"}`
	script := "#!/bin/sh\necho '" + jsonOutput + "'\n"
	binPath := writeFakeScript(t, dir, "whisper", script)

	inputFile := filepath.Join(dir, "test.wav")
	if err := os.WriteFile(inputFile, []byte("fake"), 0644); err != nil {
		t.Fatal(err)
	}

	b := NewBackend(Config{
		BinaryPath:     binPath,
		Model:          "base",
		TimeoutSeconds: 10,
	})

	transcript, err := b.TranscribeFile(inputFile, asr.TranscribeOptions{
		Model: "large",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if transcript.Model != "large" {
		t.Errorf("expected opts model %q to override config, got %q", "large", transcript.Model)
	}
}

func TestHealthCheck_BinaryExists(t *testing.T) {
	// Use /bin/echo as a known-good binary
	status, err := NewBackend(Config{
		BinaryPath: "/bin/echo",
	}).HealthCheck()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !status.OK {
		t.Errorf("expected OK=true, got OK=false, message: %s", status.Message)
	}
	if status.Backend != "local_whisper" {
		t.Errorf("expected backend %q, got %q", "local_whisper", status.Backend)
	}
	if status.Latency <= 0 {
		t.Errorf("expected positive latency, got %v", status.Latency)
	}
}

func TestHealthCheck_MissingBinary(t *testing.T) {
	status, err := NewBackend(Config{
		BinaryPath: "/nonexistent/whisper",
	}).HealthCheck()
	if err != nil {
		t.Fatalf("unexpected error from HealthCheck: %v", err)
	}
	if status.OK {
		t.Error("expected OK=false for missing binary")
	}
	if !strings.Contains(status.Message, "binary not found") {
		t.Errorf("expected 'binary not found' in message, got: %s", status.Message)
	}
}

func TestHealthCheck_MissingModel(t *testing.T) {
	status, err := NewBackend(Config{
		BinaryPath: "/bin/echo",
		ModelPath:  "/nonexistent/model.bin",
	}).HealthCheck()
	if err != nil {
		t.Fatalf("unexpected error from HealthCheck: %v", err)
	}
	if status.OK {
		t.Error("expected OK=false for missing model")
	}
	if !strings.Contains(status.Message, "model not found") {
		t.Errorf("expected 'model not found' in message, got: %s", status.Message)
	}
}

func TestHealthCheck_NotExecutable(t *testing.T) {
	dir := t.TempDir()
	// Create a file that is NOT executable
	path := filepath.Join(dir, "not-exec")
	if err := os.WriteFile(path, []byte("not a binary"), 0644); err != nil {
		t.Fatal(err)
	}

	status, err := NewBackend(Config{
		BinaryPath: path,
	}).HealthCheck()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.OK {
		t.Error("expected OK=false for non-executable file")
	}
	if !strings.Contains(status.Message, "not executable") {
		t.Errorf("expected 'not executable' in message, got: %s", status.Message)
	}
}

func TestDefaultTimeout(t *testing.T) {
	b := NewBackend(Config{})
	if b.cfg.TimeoutSeconds != 300 {
		t.Errorf("expected default timeout 300, got %d", b.cfg.TimeoutSeconds)
	}
}

func TestBuildArgs(t *testing.T) {
	b := NewBackend(Config{
		BinaryPath: "/usr/bin/whisper",
		ModelPath:  "/models/small.bin",
		Threads:    4,
	})

	args := b.buildArgs("/tmp/audio.wav", asr.TranscribeOptions{
		Language: "de",
	})

	expected := []string{
		"--model", "/models/small.bin",
		"--output-json",
		"--language", "de",
		"--threads", "4",
		"/tmp/audio.wav",
	}

	if len(args) != len(expected) {
		t.Fatalf("expected %d args, got %d: %v", len(expected), len(args), args)
	}
	for i, a := range args {
		if a != expected[i] {
			t.Errorf("arg[%d]: expected %q, got %q", i, expected[i], a)
		}
	}
}

func TestBuildArgs_Minimal(t *testing.T) {
	b := NewBackend(Config{})
	args := b.buildArgs("/tmp/audio.wav", asr.TranscribeOptions{})

	// Should only have --output-json and the file path
	expected := []string{"--output-json", "/tmp/audio.wav"}
	if len(args) != len(expected) {
		t.Fatalf("expected %d args, got %d: %v", len(expected), len(args), args)
	}
	for i, a := range args {
		if a != expected[i] {
			t.Errorf("arg[%d]: expected %q, got %q", i, expected[i], a)
		}
	}
}

// Verify Backend implements asr.Backend at compile time.
var _ asr.Backend = (*Backend)(nil)
