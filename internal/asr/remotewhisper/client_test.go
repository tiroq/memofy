// T028: FR-013 — Tests for remote Whisper API backend.
package remotewhisper

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tiroq/memofy/internal/asr"
)

// newTestClient creates a Client pointing at the given test server with fast
// retry settings suitable for tests (no hardcoded sleeps).
func newTestClient(ts *httptest.Server) *Client {
	c := NewClient(Config{
		BaseURL:        ts.URL,
		TimeoutSeconds: 5,
		Retries:        3,
		Model:          "small",
	})
	c.backoffBase = time.Millisecond // fast retries in tests
	return c
}

// createTempAudio creates a temporary file with dummy audio data for testing.
func createTempAudio(t *testing.T) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "test-audio-*.wav")
	if err != nil {
		t.Fatalf("create temp audio: %v", err)
	}
	_, _ = f.WriteString("fake-audio-data")
	f.Close()
	return f.Name()
}

// validTranscribeResponse returns a valid JSON response body.
func validTranscribeResponse() string {
	return `{
		"segments": [
			{"start": 0.0, "end": 5.2, "text": "Hello world", "language": "en", "score": 0.95},
			{"start": 5.2, "end": 10.0, "text": "How are you", "language": "en", "score": 0.88}
		],
		"language": "en",
		"duration": 120.5,
		"model": "small"
	}`
}

func TestTranscribeFile_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request shape.
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/transcribe" {
			t.Errorf("expected /v1/transcribe, got %s", r.URL.Path)
		}
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
			t.Errorf("expected multipart content-type, got %s", r.Header.Get("Content-Type"))
		}

		// Parse multipart form.
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Fatalf("parse multipart: %v", err)
		}

		// Verify fields.
		if got := r.FormValue("model"); got != "small" {
			t.Errorf("expected model=small, got %q", got)
		}
		if got := r.FormValue("language"); got != "en" {
			t.Errorf("expected language=en, got %q", got)
		}
		if got := r.FormValue("timestamps"); got != "true" {
			t.Errorf("expected timestamps=true, got %q", got)
		}

		// Verify file upload.
		file, header, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("expected file field: %v", err)
		}
		defer file.Close()
		if header.Filename == "" {
			t.Error("expected non-empty filename")
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, validTranscribeResponse())
	}))
	defer ts.Close()

	c := newTestClient(ts)
	audioPath := createTempAudio(t)

	result, err := c.TranscribeFile(audioPath, asr.TranscribeOptions{
		Language:   "en",
		Model:      "small",
		Timestamps: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Backend != "remote_whisper_api" {
		t.Errorf("expected backend %q, got %q", "remote_whisper_api", result.Backend)
	}
	if result.Language != "en" {
		t.Errorf("expected language %q, got %q", "en", result.Language)
	}
	if result.Model != "small" {
		t.Errorf("expected model %q, got %q", "small", result.Model)
	}
	if len(result.Segments) != 2 {
		t.Fatalf("expected 2 segments, got %d", len(result.Segments))
	}

	seg := result.Segments[0]
	if seg.Text != "Hello world" {
		t.Errorf("expected text %q, got %q", "Hello world", seg.Text)
	}
	if seg.Start != 0 {
		t.Errorf("expected start 0, got %v", seg.Start)
	}
	expectedEnd := time.Duration(5.2 * float64(time.Second))
	if seg.End != expectedEnd {
		t.Errorf("expected end %v, got %v", expectedEnd, seg.End)
	}
	if seg.Score != 0.95 {
		t.Errorf("expected score 0.95, got %f", seg.Score)
	}

	expectedDur := time.Duration(120.5 * float64(time.Second))
	if result.Duration != expectedDur {
		t.Errorf("expected duration %v, got %v", expectedDur, result.Duration)
	}
}

func TestTranscribeFile_RetryOn500(t *testing.T) {
	var calls int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		if n <= 2 {
			// Drain the request body to avoid broken pipe.
			_, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, `{"error": "temporary failure"}`)
			return
		}
		// Drain body before responding.
		_, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, validTranscribeResponse())
	}))
	defer ts.Close()

	c := NewClient(Config{
		BaseURL:        ts.URL,
		TimeoutSeconds: 30,
		Retries:        3,
		Model:          "small",
	})
	c.backoffBase = time.Millisecond // fast retries in tests

	audioPath := createTempAudio(t)
	result, err := c.TranscribeFile(audioPath, asr.TranscribeOptions{})
	if err != nil {
		t.Fatalf("unexpected error after retries: %v", err)
	}
	if result.Backend != "remote_whisper_api" {
		t.Errorf("expected backend %q, got %q", "remote_whisper_api", result.Backend)
	}
	totalCalls := atomic.LoadInt32(&calls)
	if totalCalls != 3 {
		t.Errorf("expected 3 calls (2 failures + 1 success), got %d", totalCalls)
	}
}

func TestTranscribeFile_Timeout(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Drain body to avoid broken pipe.
		_, _ = io.ReadAll(r.Body)
		// Block longer than client timeout.
		time.Sleep(500 * time.Millisecond)
		fmt.Fprint(w, validTranscribeResponse())
	}))
	defer ts.Close()

	c := NewClient(Config{
		BaseURL:        ts.URL,
		TimeoutSeconds: 1, // 1s timeout; Retries defaults to 3 → all attempts time out
		Model:          "small",
	})
	c.backoffBase = time.Millisecond
	// Override to shorter timeout via the http.Client directly.
	c.client.Timeout = 100 * time.Millisecond

	audioPath := createTempAudio(t)
	_, err := c.TranscribeFile(audioPath, asr.TranscribeOptions{})
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestHealthCheck_Healthy(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/v1/health" {
			t.Errorf("expected /v1/health, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok": true}`)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	status, err := c.HealthCheck()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !status.OK {
		t.Error("expected OK=true")
	}
	if status.Backend != "remote_whisper_api" {
		t.Errorf("expected backend %q, got %q", "remote_whisper_api", status.Backend)
	}
	if status.Message != "healthy" {
		t.Errorf("expected message %q, got %q", "healthy", status.Message)
	}
	if status.Latency <= 0 {
		t.Error("expected positive latency")
	}
}

func TestHealthCheck_Unhealthy(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"error": "service down"}`)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	status, err := c.HealthCheck()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.OK {
		t.Error("expected OK=false for 500 response")
	}
	if status.Backend != "remote_whisper_api" {
		t.Errorf("expected backend %q, got %q", "remote_whisper_api", status.Backend)
	}
	if !strings.Contains(status.Message, "500") {
		t.Errorf("expected message to contain status code, got %q", status.Message)
	}
}

func TestTranscribeFile_BearerToken(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-token-123" {
			t.Errorf("expected Bearer auth header, got %q", auth)
		}
		// Drain body.
		_, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, validTranscribeResponse())
	}))
	defer ts.Close()

	c := NewClient(Config{
		BaseURL:        ts.URL,
		Token:          "test-token-123",
		TimeoutSeconds: 5,
		Retries:        0,
		Model:          "small",
	})

	audioPath := createTempAudio(t)
	_, err := c.TranscribeFile(audioPath, asr.TranscribeOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTranscribeFile_Non5xxError_NoRetry(t *testing.T) {
	var calls int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		_, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"error": "bad request"}`)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	audioPath := createTempAudio(t)

	_, err := c.TranscribeFile(audioPath, asr.TranscribeOptions{})
	if err == nil {
		t.Fatal("expected error for 400 response")
	}
	totalCalls := atomic.LoadInt32(&calls)
	if totalCalls != 1 {
		t.Errorf("expected 1 call (no retry on 400), got %d", totalCalls)
	}
}

func TestName(t *testing.T) {
	c := NewClient(Config{BaseURL: "http://localhost"})
	if c.Name() != "remote_whisper_api" {
		t.Errorf("expected name %q, got %q", "remote_whisper_api", c.Name())
	}
}

func TestNewClient_Defaults(t *testing.T) {
	c := NewClient(Config{BaseURL: "http://localhost"})
	if c.cfg.TimeoutSeconds != 120 {
		t.Errorf("expected default timeout 120, got %d", c.cfg.TimeoutSeconds)
	}
	if c.cfg.Retries != 3 {
		t.Errorf("expected default retries 3, got %d", c.cfg.Retries)
	}
	if c.cfg.Model != "small" {
		t.Errorf("expected default model %q, got %q", "small", c.cfg.Model)
	}
}

// TestBackendInterface verifies at compile time that *Client satisfies asr.Backend.
var _ asr.Backend = (*Client)(nil)

func TestTranscribeFile_OptsModelOverride(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseMultipartForm(10 << 20)
		if got := r.FormValue("model"); got != "large-v2" {
			t.Errorf("expected model override %q, got %q", "large-v2", got)
		}
		w.Header().Set("Content-Type", "application/json")
		resp := transcribeResponse{
			Segments: nil,
			Language: "en",
			Duration: 10.0,
			Model:    "large-v2",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	audioPath := createTempAudio(t)

	result, err := c.TranscribeFile(audioPath, asr.TranscribeOptions{Model: "large-v2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Model != "large-v2" {
		t.Errorf("expected model %q, got %q", "large-v2", result.Model)
	}
}

func TestHealthCheck_BearerToken(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer my-secret" {
			t.Errorf("expected Bearer auth, got %q", auth)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok": true}`)
	}))
	defer ts.Close()

	c := NewClient(Config{
		BaseURL:        ts.URL,
		Token:          "my-secret",
		TimeoutSeconds: 5,
	})
	status, err := c.HealthCheck()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !status.OK {
		t.Error("expected OK=true")
	}
}

func TestTranscribeFile_FileNotFound(t *testing.T) {
	c := NewClient(Config{BaseURL: "http://localhost", Retries: 0})
	_, err := c.TranscribeFile(filepath.Join(t.TempDir(), "nonexistent.wav"), asr.TranscribeOptions{})
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}
