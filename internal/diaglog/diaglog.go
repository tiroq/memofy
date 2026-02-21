// Package diaglog provides structured NDJSON diagnostic logging for Memofy.
// Activated by MEMOFY_DEBUG_RECORDING=true. When the env var is absent, all
// Log calls are no-ops and no file is created.
package diaglog

import (
	"encoding/json"
	"os"
	"sync"
	"time"
)

// ── Component labels (FR-002) ────────────────────────────────────────────────

const (
	ComponentOBSClient    = "obs-ws-client"
	ComponentStateMachine = "state-machine"
	ComponentAutoDetector = "auto-detector"
	ComponentReconnect    = "reconnect-handler"
	ComponentDiagExport   = "diag-export"
	ComponentMemofyCore   = "memofy-core"
)

// ── Event names (FR-001 / FR-003 / FR-011 / FR-012) ─────────────────────────

const (
	EventWSSend                = "ws_send"
	EventWSRecv                = "ws_recv"
	EventWSConnect             = "ws_connect"
	EventWSDisconnect          = "ws_disconnect"
	EventWSReconnectAttempt    = "ws_reconnect_attempt"
	EventWSReconnectSuccess    = "ws_reconnect_success"
	EventWSReconnectFailed     = "ws_reconnect_failed"
	EventMultiClientWarning    = "multi_client_warning"
	EventRecordingStart        = "recording_start"
	EventRecordingStop         = "recording_stop"
	EventRecordingStopRejected = "recording_stop_rejected"
)

// ── LogEntry ─────────────────────────────────────────────────────────────────

// LogEntry is one structured event record written as a single JSON line.
type LogEntry struct {
	Timestamp string      `json:"ts"`                   // RFC3339Nano
	Component string      `json:"component"`            // see Component* constants
	Event     string      `json:"event"`                // see Event* constants
	SessionID string      `json:"session_id,omitempty"` // FR-012
	Reason    string      `json:"reason,omitempty"`     // FR-003
	Payload   interface{} `json:"payload,omitempty"`    // redacted before write (FR-013)
}

// ── Logger ───────────────────────────────────────────────────────────────────

// Logger writes LogEntry values to a rolling NDJSON file. When debug mode is
// disabled every Log call is a no-op.
type Logger struct {
	rw      *rollingWriter
	mu      sync.Mutex
	enabled bool
}

// New opens (or creates) the NDJSON log file at path. If debug mode is
// disabled, path is ignored and a no-op logger is returned.
func New(path string) (*Logger, error) {
	if !IsDebugEnabled() {
		return &Logger{enabled: false}, nil
	}
	rw, err := newRollingWriter(path, 10*1024*1024)
	if err != nil {
		return nil, err
	}
	return &Logger{rw: rw, enabled: true}, nil
}

// Log serialises entry to JSON, appends a newline, and writes to the rolling
// file. Sensitive payload fields are redacted before serialisation (FR-013).
func (l *Logger) Log(entry LogEntry) {
	if l == nil || !l.enabled {
		return
	}
	if entry.Timestamp == "" {
		entry.Timestamp = time.Now().UTC().Format(time.RFC3339Nano)
	}
	if entry.Payload != nil {
		entry.Payload = Redact(entry.Payload)
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return
	}
	data = append(data, '\n')

	l.mu.Lock()
	defer l.mu.Unlock()
	_, _ = l.rw.Write(data)
}

// Close flushes and closes the underlying file. Safe on nil/disabled logger.
func (l *Logger) Close() error {
	if l == nil || !l.enabled || l.rw == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.rw.close()
}

// IsDebugEnabled reports whether MEMOFY_DEBUG_RECORDING is set to "true".
func IsDebugEnabled() bool {
	return os.Getenv("MEMOFY_DEBUG_RECORDING") == "true"
}

// NewNoOp returns a logger where every Log call is a no-op. Use as a safe
// fallback when New fails (e.g., disk full, permissions error).
func NewNoOp() *Logger {
	return &Logger{enabled: false}
}
