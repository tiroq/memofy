package testutil

import (
	"bytes"
	"io"
	"log"
	"os"
	"regexp"
	"strings"
	"sync"
)

// LogCapture captures log output for testing
type LogCapture struct {
	buf      bytes.Buffer
	mu       sync.Mutex
	original io.Writer
}

// NewLogCapture creates a new log capture instance
func NewLogCapture() *LogCapture {
	lc := &LogCapture{
		original: log.Writer(),
	}
	return lc
}

// Start begins capturing log output
func (lc *LogCapture) Start() {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	// Redirect log output to our buffer
	log.SetOutput(&lc.buf)
}

// Stop restores original log output
func (lc *LogCapture) Stop() {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	log.SetOutput(lc.original)
}

// String returns all captured log output
func (lc *LogCapture) String() string {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	return lc.buf.String()
}

// Reset clears the capture buffer
func (lc *LogCapture) Reset() {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	lc.buf.Reset()
}

// Contains checks if the log output contains the given substring
func (lc *LogCapture) Contains(substr string) bool {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	return strings.Contains(lc.buf.String(), substr)
}

// ContainsAll checks if the log output contains all given substrings
func (lc *LogCapture) ContainsAll(substrs ...string) bool {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	content := lc.buf.String()
	for _, substr := range substrs {
		if !strings.Contains(content, substr) {
			return false
		}
	}
	return true
}

// NotContains checks if the log output does NOT contain the given substring
func (lc *LogCapture) NotContains(substr string) bool {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	return !strings.Contains(lc.buf.String(), substr)
}

// MatchesPattern checks if the log output matches the given regex pattern
func (lc *LogCapture) MatchesPattern(pattern string) bool {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	re, err := regexp.Compile(pattern)
	if err != nil {
		return false
	}

	return re.MatchString(lc.buf.String())
}

// Count returns the number of times a substring appears in the log
func (lc *LogCapture) Count(substr string) int {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	return strings.Count(lc.buf.String(), substr)
}

// Lines returns all captured log lines
func (lc *LogCapture) Lines() []string {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	content := lc.buf.String()
	if content == "" {
		return []string{}
	}

	return strings.Split(strings.TrimSpace(content), "\n")
}

// LastLine returns the last line of captured log output
func (lc *LogCapture) LastLine() string {
	lines := lc.Lines()
	if len(lines) == 0 {
		return ""
	}
	return lines[len(lines)-1]
}

// CaptureStdout temporarily redirects stdout to capture it
type StdoutCapture struct {
	buf      bytes.Buffer
	mu       sync.Mutex
	original *os.File
	r        *os.File
	w        *os.File
}

// NewStdoutCapture creates a new stdout capture instance
func NewStdoutCapture() *StdoutCapture {
	return &StdoutCapture{
		original: os.Stdout,
	}
}

// Start begins capturing stdout
func (sc *StdoutCapture) Start() error {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	var err error
	sc.r, sc.w, err = os.Pipe()
	if err != nil {
		return err
	}

	os.Stdout = sc.w

	go func() {
		_, _ = io.Copy(&sc.buf, sc.r)
	}()

	return nil
}

// Stop restores original stdout and returns captured content
func (sc *StdoutCapture) Stop() string {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	if sc.w != nil {
		_ = sc.w.Close()
	}

	os.Stdout = sc.original

	if sc.r != nil {
		_ = sc.r.Close()
	}

	return sc.buf.String()
}

// String returns all captured stdout
func (sc *StdoutCapture) String() string {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	return sc.buf.String()
}
