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
	defer l.Close()

	entries := []LogEntry{
		{Component: ComponentOBSClient, Event: EventWSConnect},
		{Component: ComponentStateMachine, Event: EventRecordingStart, Reason: "manual", SessionID: "abc123"},
		{Component: ComponentAutoDetector, Event: EventRecordingStop},
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
	defer f.Close()

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
	if lines[0]["component"] != ComponentOBSClient {
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
	defer rw.close()

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

func TestNoOpWhenDisabled(t *testing.T) {
	os.Unsetenv("MEMOFY_DEBUG_RECORDING")

	tmp := t.TempDir() + "/noop.ndjson"
	l, err := New(tmp)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	l.Log(LogEntry{Component: ComponentOBSClient, Event: EventWSConnect})
	_ = l.Close()

	if _, err := os.Stat(tmp); !os.IsNotExist(err) {
		t.Error("log file should not exist when debug disabled")
	}
}
