package googlestt

import (
	"os"
	"testing"

	"github.com/tiroq/memofy/internal/asr"
)

// T030: Tests for Google STT stub backend. FR-013

func TestBackendImplementsInterface(t *testing.T) {
	var _ asr.Backend = (*Backend)(nil)
}

func TestName(t *testing.T) {
	b := NewBackend(Config{})
	if b.Name() != "google_stt" {
		t.Errorf("Name() = %q, want %q", b.Name(), "google_stt")
	}
}

func TestTranscribeFileReturnsNotImplemented(t *testing.T) {
	b := NewBackend(Config{})
	result, err := b.TranscribeFile("/some/file.wav", asr.TranscribeOptions{})
	if result != nil {
		t.Errorf("TranscribeFile result should be nil, got %v", result)
	}
	if err == nil {
		t.Fatal("TranscribeFile should return error")
	}
	if err.Error() != "google_stt: not yet implemented" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHealthCheckNoCredentials(t *testing.T) {
	b := NewBackend(Config{})
	status, err := b.HealthCheck()
	if err != nil {
		t.Fatalf("HealthCheck returned error: %v", err)
	}
	if status.OK {
		t.Error("HealthCheck should be unhealthy without credentials")
	}
	if status.Backend != "google_stt" {
		t.Errorf("Backend = %q, want %q", status.Backend, "google_stt")
	}
	if status.Message != "no credentials file configured" {
		t.Errorf("unexpected message: %s", status.Message)
	}
}

func TestHealthCheckNonExistentFile(t *testing.T) {
	b := NewBackend(Config{CredentialsFile: "/nonexistent/creds.json"})
	status, err := b.HealthCheck()
	if err != nil {
		t.Fatalf("HealthCheck returned error: %v", err)
	}
	if status.OK {
		t.Error("HealthCheck should be unhealthy with missing file")
	}
}

func TestHealthCheckExistingFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "google-creds-*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	b := NewBackend(Config{CredentialsFile: tmpFile.Name()})
	status, err := b.HealthCheck()
	if err != nil {
		t.Fatalf("HealthCheck returned error: %v", err)
	}
	if !status.OK {
		t.Errorf("HealthCheck should be healthy with existing file, got message: %s", status.Message)
	}
	if status.Backend != "google_stt" {
		t.Errorf("Backend = %q, want %q", status.Backend, "google_stt")
	}
}
