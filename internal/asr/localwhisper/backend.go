// T029: FR-013 — local whisper CLI backend for ASR transcription.
package localwhisper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/tiroq/memofy/internal/asr"
)

// Config configures the local whisper CLI backend.
type Config struct {
	BinaryPath     string // path to whisper-cpp or faster-whisper CLI
	ModelPath      string // path to .bin model file
	Model          string // model name (e.g., "small", "base")
	Threads        int    // CPU threads (0 = auto)
	TimeoutSeconds int    // default 300 (5 minutes for long recordings)
}

// Backend shells out to a whisper CLI binary for local transcription.
type Backend struct {
	cfg Config
}

// NewBackend creates a new local whisper backend with the given config.
func NewBackend(cfg Config) *Backend {
	if cfg.TimeoutSeconds <= 0 {
		cfg.TimeoutSeconds = 300
	}
	return &Backend{cfg: cfg}
}

// Name returns the backend identifier.
func (b *Backend) Name() string {
	return "local_whisper"
}

// whisperSegment represents a single segment in whisper CLI JSON output.
type whisperSegment struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Text  string  `json:"text"`
	Score float64 `json:"score"`
}

// whisperOutput represents the JSON output from whisper CLI.
type whisperOutput struct {
	Segments []whisperSegment `json:"segments"`
	Language string           `json:"language"`
}

// TranscribeFile invokes the whisper CLI subprocess to transcribe an audio file.
func (b *Backend) TranscribeFile(filePath string, opts asr.TranscribeOptions) (*asr.Transcript, error) {
	// Validate binary exists
	if _, err := os.Stat(b.cfg.BinaryPath); err != nil {
		return nil, fmt.Errorf("localwhisper: binary not found at %q: %w", b.cfg.BinaryPath, err)
	}

	args := b.buildArgs(filePath, opts)
	cmd := exec.Command(b.cfg.BinaryPath, args...)

	// Use process group so we can kill the entire tree on timeout
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("localwhisper: failed to start subprocess: %w", err)
	}

	// Set up timeout kill using time.AfterFunc + Process.Kill
	var mu sync.Mutex
	var killed bool
	timer := time.AfterFunc(time.Duration(b.cfg.TimeoutSeconds)*time.Second, func() {
		mu.Lock()
		killed = true
		mu.Unlock()
		// Kill the entire process group
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	})

	err := cmd.Wait()
	timer.Stop()

	if err != nil {
		mu.Lock()
		wasKilled := killed
		mu.Unlock()
		if wasKilled {
			return nil, fmt.Errorf("localwhisper: transcription timed out after %d seconds", b.cfg.TimeoutSeconds)
		}
		return nil, fmt.Errorf("localwhisper: subprocess failed: %w", err)
	}

	// Parse JSON output
	var output whisperOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		return nil, fmt.Errorf("localwhisper: failed to parse JSON output: %w", err)
	}

	// Convert to asr.Transcript
	transcript := &asr.Transcript{
		Language: output.Language,
		Model:    b.resolveModel(opts),
		Backend:  b.Name(),
	}

	for _, seg := range output.Segments {
		transcript.Segments = append(transcript.Segments, asr.Segment{
			Start: floatToDuration(seg.Start),
			End:   floatToDuration(seg.End),
			Text:  seg.Text,
			Score: seg.Score,
		})
	}

	// Compute total duration from last segment end
	if len(transcript.Segments) > 0 {
		transcript.Duration = transcript.Segments[len(transcript.Segments)-1].End
	}

	return transcript, nil
}

// HealthCheck verifies the whisper binary exists, is executable, and responds.
func (b *Backend) HealthCheck() (*asr.HealthStatus, error) {
	status := &asr.HealthStatus{
		Backend: b.Name(),
	}

	// Check binary exists and is executable
	info, err := os.Stat(b.cfg.BinaryPath)
	if err != nil {
		status.Message = fmt.Sprintf("binary not found at %q: %v", b.cfg.BinaryPath, err)
		return status, nil
	}
	if info.Mode()&0111 == 0 {
		status.Message = fmt.Sprintf("binary at %q is not executable", b.cfg.BinaryPath)
		return status, nil
	}

	// Check model path exists if configured
	if b.cfg.ModelPath != "" {
		if _, err := os.Stat(b.cfg.ModelPath); err != nil {
			status.Message = fmt.Sprintf("model not found at %q: %v", b.cfg.ModelPath, err)
			return status, nil
		}
	}

	// Run binary with --help to verify it works
	start := time.Now()
	cmd := exec.Command(b.cfg.BinaryPath, "--help")
	err = cmd.Run()
	status.Latency = time.Since(start)

	// --help may exit non-zero on some binaries; we just need it to execute
	if err != nil {
		// Check if it's an exec error (binary can't run at all)
		if _, ok := err.(*exec.ExitError); !ok {
			status.Message = fmt.Sprintf("binary failed to execute: %v", err)
			return status, nil
		}
	}

	status.OK = true
	status.Message = "binary is available and executable"
	return status, nil
}

// buildArgs constructs the CLI arguments for the whisper binary.
func (b *Backend) buildArgs(filePath string, opts asr.TranscribeOptions) []string {
	var args []string

	if b.cfg.ModelPath != "" {
		args = append(args, "--model", b.cfg.ModelPath)
	}

	args = append(args, "--output-json")

	if opts.Language != "" {
		args = append(args, "--language", opts.Language)
	}

	if b.cfg.Threads > 0 {
		args = append(args, "--threads", strconv.Itoa(b.cfg.Threads))
	}

	args = append(args, filePath)
	return args
}

// resolveModel returns the model name, preferring opts over config.
func (b *Backend) resolveModel(opts asr.TranscribeOptions) string {
	if opts.Model != "" {
		return opts.Model
	}
	return b.cfg.Model
}

// floatToDuration converts seconds (float64) to time.Duration.
func floatToDuration(seconds float64) time.Duration {
	return time.Duration(seconds * float64(time.Second))
}
