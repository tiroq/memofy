package diaglog

import (
	"bufio"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestLogWritesNDJSON(t *testing.T) {
	t.Setenv("MEMOFY_DEBUG_RECORDING", "true")

	tmp := t.TempDir() + "/test.ndjson"
	l, err := New(tmp)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = l.Close() }()

	entries := []LogEntry{
		{Component: ComponentEngine, Event: EventSoundDetected},
		{Component: ComponentStateMachine, Event: EventRecordingStart, Reason: "auto", SessionID: "abc123"},
		{Component: ComponentAudioCapture, Event: EventRecordingStop},
	}
	for _, e := range entries {
		l.Log(e)
	}
	if err := l.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	f, err := os.Open(tmp)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	var lines []map[string]interface{}
	for scanner.Scan() {
		var m map[string]interface{}
		if err := json.Unmarshal(scanner.Bytes(), &m); err != nil {
			t.Fatalf("invalid JSON line: %v -> %s", err, scanner.Text())
		}
		lines = append(lines, m)
	}
	if len(lines) != len(entries) {
		t.Fatalf("want %d lines, got %d", len(entries), len(lines))
	}
	if lines[0]["component"] != ComponentEngine {
		t.Errorf("component mismatch: %v", lines[0]["component"])
	}
	if lines[1]["session_id"] != "abc123" {
		t.Errorf("session_id mismatch: %v", lines[1]["session_id"])
	}
	if lines[0]["ts"] == nil {
		t.Error("ts field missing")
	}
}

func TestRollingTruncatesAt10MB(t *testing.T) {
	t.Setenv("MEMOFY_DEBUG_RECORDING", "true")

	tmp := t.TempDir() + "/roll.ndjson"
	const maxSize = 1024
	rw, err := newRollingWriter(tmp, maxSize)
	if err != nil {
		t.Fatalf("newRollingWriter: %v", err)
	}
	defer func() { _ = rw.close() }()

	chunk := []byte(strings.Repeat("x", 512) + "\n")
	for i := 0; i < 3; i++ {
		if _, err := rw.Write(chunk); err != nil {
			t.Fatalf("write %d: %v", i, err)
		}
	}

	info, err := os.Stat(tmp)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Size() > maxSize {
		t.Errorf("file size %d exceeds maxSize %d", info.Size(), maxSize)
	}
}

func TestRedactSensitiveFields(t *testing.T) {
	input := map[string]interface{}{
		"authentication": "secret-token",
		"challenge":      "xyz",
		"salt":           "abc",
		"auth":           "tok",
		"password":       "hunter2",
		"secret":         "s3cr3t",
		"safe_field":     "keep-me",
		"nested": map[string]interface{}{
			"password": "nested-pass",
			"ok":       "value",
		},
	}

	out := Redact(input).(map[string]interface{})
	for _, k := range []string{"authentication", "challenge", "salt", "auth", "password", "secret"} {
		if out[k] != "[REDACTED]" {
			t.Errorf("key %q: want [REDACTED], got %v", k, out[k])
		}
	}
	if out["safe_field"] != "keep-me" {
		t.Errorf("safe_field should be preserved")
	}
	nested := out["nested"].(map[string]interface{})
	if nested["password"] != "[REDACTED]" {
		t.Error("nested password not redacted")
	}
	if nested["ok"] != "value" {
		t.Error("nested ok field should be preserved")
	}
}

func TestNewNoOp(t *testing.T) {
	l := NewNoOp()
	if l == nil {
		t.Fatal("NewNoOp returned nil")
	}
	// Should not panic
	l.Log(LogEntry{Component: ComponentEngine, Event: EventSoundDetected})
	if err := l.Close(); err != nil {
		t.Errorf("Close on no-op: %v", err)
	}
}

func TestLogNilLogger(t *testing.T) {
	var l *Logger
	// Should not panic
	l.Log(LogEntry{Component: ComponentEngine, Event: EventSoundDetected})
	if err := l.Close(); err != nil {
		t.Errorf("Close on nil: %v", err)
	}
}

func TestLogDisabledLogger(t *testing.T) {
	t.Setenv("MEMOFY_DEBUG_RECORDING", "false")

	tmp := t.TempDir() + "/disabled.ndjson"
	l, err := New(tmp)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	l.Log(LogEntry{Component: ComponentEngine, Event: EventSoundDetected})
	if err := l.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
	// File should not exist since debug is off
	if _, err := os.Stat(tmp); err == nil {
		t.Error("file should not exist when debug is disabled")
	}
}

func TestRedactSlice(t *testing.T) {
	input := []interface{}{
		map[string]interface{}{"password": "secret", "data": "ok"},
		"plain",
		42,
	}
	out := Redact(input).([]interface{})
	m := out[0].(map[string]interface{})
	if m["password"] != "[REDACTED]" {
		t.Error("password in slice element not redacted")
	}
	if m["data"] != "ok" {
		t.Error("safe field in slice element should be preserved")
	}
	if out[1] != "plain" {
		t.Error("plain string in slice should be preserved")
	}
}

func TestRedactNonMap(t *testing.T) {
	if got := Redact("hello"); got != "hello" {
		t.Errorf("Redact string: got %v, want hello", got)
	}
	if got := Redact(42); got != 42 {
		t.Errorf("Redact int: got %v, want 42", got)
	}
}

func TestLogWithPayloadRedaction(t *testing.T) {
	t.Setenv("MEMOFY_DEBUG_RECORDING", "true")

	tmp := t.TempDir() + "/redact.ndjson"
	l, err := New(tmp)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer l.Close()

	l.Log(LogEntry{
		Component: ComponentEngine,
		Event:     EventSoundDetected,
		Payload: map[string]interface{}{
			"password": "secret123",
			"level":    0.05,
		},
	})
	l.Close()

	data, err := os.ReadFile(tmp)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	payload := m["payload"].(map[string]interface{})
	if payload["password"] != "[REDACTED]" {
		t.Error("password should be redacted in log output")
	}
}

func TestIsDebugEnabled(t *testing.T) {
	t.Setenv("MEMOFY_DEBUG_RECORDING", "true")
	if !IsDebugEnabled() {
		t.Error("should be enabled when MEMOFY_DEBUG_RECORDING=true")
	}
	t.Setenv("MEMOFY_DEBUG_RECORDING", "false")
	if IsDebugEnabled() {
		t.Error("should be disabled when MEMOFY_DEBUG_RECORDING=false")
	}
}

func TestNoOpWhenDisabled(t *testing.T) {
	_ = os.Unsetenv("MEMOFY_DEBUG_RECORDING")

	tmp := t.TempDir() + "/noop.ndjson"
	l, err := New(tmp)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	l.Log(LogEntry{Component: ComponentEngine, Event: EventSoundDetected})
	_ = l.Close()

	if _, err := os.Stat(tmp); !os.IsNotExist(err) {
		t.Error("log file should not exist when debug disabled")
	}
}
