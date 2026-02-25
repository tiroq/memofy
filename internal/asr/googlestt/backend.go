// Package googlestt provides a stub Google Cloud Speech-to-Text backend.
// T030: FR-013 — stub backend implementing asr.Backend interface.
package googlestt

import (
	"fmt"
	"os"

	"github.com/tiroq/memofy/internal/asr"
)

// Compile-time interface check.
var _ asr.Backend = (*Backend)(nil)

// Config holds Google Cloud STT settings.
type Config struct {
	CredentialsFile string // path to service account JSON
	LanguageCode    string // e.g., "en-US"
}

// Backend is a stub Google Cloud STT implementation.
type Backend struct {
	cfg Config
}

// NewBackend creates a new Google STT stub backend.
func NewBackend(cfg Config) *Backend {
	return &Backend{cfg: cfg}
}

// Name returns the backend identifier.
func (b *Backend) Name() string { return "google_stt" }

// TranscribeFile is not yet implemented — returns an error.
func (b *Backend) TranscribeFile(filePath string, opts asr.TranscribeOptions) (*asr.Transcript, error) {
	return nil, fmt.Errorf("google_stt: not yet implemented")
}

// HealthCheck verifies whether the credentials file is accessible.
func (b *Backend) HealthCheck() (*asr.HealthStatus, error) {
	status := &asr.HealthStatus{
		Backend: b.Name(),
	}

	if b.cfg.CredentialsFile == "" {
		status.OK = false
		status.Message = "no credentials file configured"
		return status, nil
	}

	if _, err := os.Stat(b.cfg.CredentialsFile); err != nil {
		status.OK = false
		status.Message = fmt.Sprintf("credentials file not accessible: %v", err)
		return status, nil
	}

	status.OK = true
	status.Message = "credentials file present (stub backend — transcription not implemented)"
	return status, nil
}
