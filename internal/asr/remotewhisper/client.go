// T028: FR-013 — Remote Whisper API backend for ASR transcription.
package remotewhisper

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/tiroq/memofy/internal/asr"
	"github.com/tiroq/memofy/internal/diaglog"
)

// Config configures the remote Whisper API client.
type Config struct {
	BaseURL        string
	Token          string // optional auth token, sent as Bearer
	TimeoutSeconds int    // default 120
	Retries        int    // default 3
	Model          string // default "small"
}

// Client is an asr.Backend that calls a remote Whisper HTTP API.
type Client struct {
	cfg         Config
	client      *http.Client
	backoffBase time.Duration // default time.Second; tests override to 1ms

	logger   *diaglog.Logger
	loggerMu sync.RWMutex
}

// NewClient creates a new remote Whisper API client.
func NewClient(cfg Config) *Client {
	if cfg.TimeoutSeconds <= 0 {
		cfg.TimeoutSeconds = 120
	}
	if cfg.Retries <= 0 {
		cfg.Retries = 3
	}
	if cfg.Model == "" {
		cfg.Model = "small"
	}
	return &Client{
		cfg:         cfg,
		backoffBase: time.Second,
		client: &http.Client{
			Timeout: time.Duration(cfg.TimeoutSeconds) * time.Second,
		},
	}
}

// SetLogger injects a diaglog.Logger for debug logging.
func (c *Client) SetLogger(l *diaglog.Logger) {
	c.loggerMu.Lock()
	c.logger = l
	c.loggerMu.Unlock()
}

func (c *Client) log(entry diaglog.LogEntry) {
	c.loggerMu.RLock()
	l := c.logger
	c.loggerMu.RUnlock()
	if l == nil {
		return
	}
	if entry.Component == "" {
		entry.Component = "remote-whisper"
	}
	l.Log(entry)
}

// Name returns the backend identifier.
func (c *Client) Name() string {
	return "remote_whisper_api"
}

// transcribeResponse mirrors the JSON shape returned by the remote API.
type transcribeResponse struct {
	Segments []struct {
		Start    float64 `json:"start"`
		End      float64 `json:"end"`
		Text     string  `json:"text"`
		Language string  `json:"language"`
		Score    float64 `json:"score"`
	} `json:"segments"`
	Language string  `json:"language"`
	Duration float64 `json:"duration"`
	Model    string  `json:"model"`
}

// TranscribeFile sends the audio file to the remote Whisper API and returns
// a parsed Transcript. Retries on transient errors (5xx, network).
func (c *Client) TranscribeFile(filePath string, opts asr.TranscribeOptions) (*asr.Transcript, error) {
	model := opts.Model
	if model == "" {
		model = c.cfg.Model
	}

	timestamps := "false"
	if opts.Timestamps {
		timestamps = "true"
	}

	var lastErr error
	for attempt := 0; attempt <= c.cfg.Retries; attempt++ {
		if attempt > 0 {
			backoff := c.backoff(attempt)
			c.log(diaglog.LogEntry{
				Event:   "transcribe_retry",
				Payload: map[string]interface{}{"attempt": attempt, "backoff_ms": backoff.Milliseconds()},
			})
			time.Sleep(backoff)
		}

		result, err := c.doTranscribe(filePath, model, opts.Language, timestamps)
		if err == nil {
			return result, nil
		}

		if !isRetryable(err) {
			return nil, fmt.Errorf("transcribe %s: %w", filepath.Base(filePath), err)
		}
		lastErr = err
	}

	return nil, fmt.Errorf("transcribe %s: all %d retries exhausted: %w", filepath.Base(filePath), c.cfg.Retries, lastErr)
}

// doTranscribe performs a single multipart POST to the transcription endpoint.
func (c *Client) doTranscribe(filePath, model, language, timestamps string) (*asr.Transcript, error) {
	// Open audio file.
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open audio file: %w", err)
	}
	defer f.Close()

	// Build multipart body.
	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	// Write multipart in a goroutine so the pipe feeds the request body.
	errCh := make(chan error, 1)
	go func() {
		defer pw.Close()

		part, err := writer.CreateFormFile("file", filepath.Base(filePath))
		if err != nil {
			errCh <- fmt.Errorf("create form file: %w", err)
			return
		}
		if _, err := io.Copy(part, f); err != nil {
			errCh <- fmt.Errorf("copy audio data: %w", err)
			return
		}
		_ = writer.WriteField("model", model)
		_ = writer.WriteField("language", language)
		_ = writer.WriteField("timestamps", timestamps)

		errCh <- writer.Close()
	}()

	url := c.cfg.BaseURL + "/v1/transcribe"
	req, err := http.NewRequest(http.MethodPost, url, pr)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if c.cfg.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.cfg.Token)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, &retryableError{err: fmt.Errorf("http request: %w", err)}
	}
	defer resp.Body.Close()

	// Drain the multipart writer goroutine.
	if writeErr := <-errCh; writeErr != nil {
		return nil, fmt.Errorf("multipart write: %w", writeErr)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &retryableError{err: fmt.Errorf("read response body: %w", err)}
	}

	if resp.StatusCode >= 500 {
		return nil, &retryableError{err: fmt.Errorf("server error %d: %s", resp.StatusCode, truncate(body, 200))}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("http %d: %s", resp.StatusCode, truncate(body, 200))
	}

	var parsed transcribeResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	segments := make([]asr.Segment, len(parsed.Segments))
	for i, s := range parsed.Segments {
		segments[i] = asr.Segment{
			Start:    floatSecToDuration(s.Start),
			End:      floatSecToDuration(s.End),
			Text:     s.Text,
			Language: s.Language,
			Score:    s.Score,
		}
	}

	return &asr.Transcript{
		Segments: segments,
		Language: parsed.Language,
		Duration: floatSecToDuration(parsed.Duration),
		Model:    parsed.Model,
		Backend:  c.Name(),
	}, nil
}

// HealthCheck queries the remote API health endpoint.
func (c *Client) HealthCheck() (*asr.HealthStatus, error) {
	start := time.Now()
	url := c.cfg.BaseURL + "/v1/health"

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create health request: %w", err)
	}
	if c.cfg.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.cfg.Token)
	}

	resp, err := c.client.Do(req)
	latency := time.Since(start)
	if err != nil {
		return &asr.HealthStatus{
			OK:      false,
			Backend: c.Name(),
			Message: fmt.Sprintf("health check failed: %v", err),
			Latency: latency,
		}, nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &asr.HealthStatus{
			OK:      false,
			Backend: c.Name(),
			Message: fmt.Sprintf("unhealthy: http %d: %s", resp.StatusCode, truncate(body, 200)),
			Latency: latency,
		}, nil
	}

	var parsed struct {
		OK bool `json:"ok"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return &asr.HealthStatus{
			OK:      false,
			Backend: c.Name(),
			Message: fmt.Sprintf("invalid health response: %v", err),
			Latency: latency,
		}, nil
	}

	msg := "healthy"
	if !parsed.OK {
		msg = "service reports not ok"
	}

	return &asr.HealthStatus{
		OK:      parsed.OK,
		Backend: c.Name(),
		Message: msg,
		Latency: latency,
	}, nil
}

// ── helpers ──────────────────────────────────────────────────────────────────

// retryableError wraps errors that should trigger a retry.
type retryableError struct {
	err error
}

func (e *retryableError) Error() string { return e.err.Error() }
func (e *retryableError) Unwrap() error { return e.err }

// isRetryable returns true for retryableError instances.
func isRetryable(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(*retryableError)
	return ok
}

// backoff returns exponential backoff duration: base * 2^(attempt-1) + jitter.
func (c *Client) backoff(attempt int) time.Duration {
	base := c.backoffBase
	if base <= 0 {
		base = time.Second
	}
	delay := base
	for i := 1; i < attempt; i++ {
		delay *= 2
	}
	// Add jitter: 0–25% of delay.
	jitter := time.Duration(rand.Int63n(int64(delay/4) + 1))
	return delay + jitter
}

// floatSecToDuration converts fractional seconds to time.Duration.
func floatSecToDuration(sec float64) time.Duration {
	return time.Duration(sec * float64(time.Second))
}

// truncate returns the first n bytes of body as a string.
func truncate(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "..."
}
