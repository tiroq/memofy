package asr

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// mockBackend is a test double for the Backend interface.
type mockBackend struct {
	name       string
	transcript *Transcript
	err        error
	health     *HealthStatus
	healthErr  error
}

func (m *mockBackend) Name() string { return m.name }
func (m *mockBackend) TranscribeFile(filePath string, opts TranscribeOptions) (*Transcript, error) {
	return m.transcript, m.err
}
func (m *mockBackend) HealthCheck() (*HealthStatus, error) {
	return m.health, m.healthErr
}

func TestRegistryRegisterAndGet(t *testing.T) {
	r := NewRegistry()
	b := &mockBackend{name: "test"}

	r.Register("test", b)

	got, ok := r.Get("test")
	if !ok {
		t.Fatal("expected Get to return true for registered backend")
	}
	if got.Name() != "test" {
		t.Errorf("expected name %q, got %q", "test", got.Name())
	}

	_, ok = r.Get("missing")
	if ok {
		t.Fatal("expected Get to return false for unregistered backend")
	}
}

func TestRegistryPrimary(t *testing.T) {
	r := NewRegistry()
	first := &mockBackend{name: "first"}
	second := &mockBackend{name: "second"}

	r.Register("first", first)
	r.Register("second", second)

	primary := r.Primary()
	if primary == nil {
		t.Fatal("expected primary to be set")
	}
	if primary.Name() != "first" {
		t.Errorf("expected first registered backend as primary, got %q", primary.Name())
	}
}

func TestRegistrySetPrimary(t *testing.T) {
	r := NewRegistry()
	first := &mockBackend{name: "first"}
	second := &mockBackend{name: "second"}

	r.Register("first", first)
	r.Register("second", second)
	r.SetPrimary("second")

	primary := r.Primary()
	if primary == nil {
		t.Fatal("expected primary to be set")
	}
	if primary.Name() != "second" {
		t.Errorf("expected primary %q, got %q", "second", primary.Name())
	}
}

func TestTranscribeWithFallback_PrimarySucceeds(t *testing.T) {
	r := NewRegistry()
	expected := &Transcript{
		Segments: []Segment{{Text: "hello", Start: 0, End: time.Second}},
		Backend:  "primary",
	}
	primary := &mockBackend{name: "primary", transcript: expected}
	fallback := &mockBackend{name: "fallback", transcript: &Transcript{Backend: "fallback"}}

	r.Register("primary", primary)
	r.Register("fallback", fallback)
	r.SetFallback("fallback")

	result, err := r.TranscribeWithFallback("test.wav", TranscribeOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Backend != "primary" {
		t.Errorf("expected primary backend result, got %q", result.Backend)
	}
}

func TestTranscribeWithFallback_PrimaryFailsFallbackSucceeds(t *testing.T) {
	r := NewRegistry()
	expected := &Transcript{
		Segments: []Segment{{Text: "hello"}},
		Backend:  "fallback",
	}
	primary := &mockBackend{name: "primary", err: fmt.Errorf("primary down")}
	fallback := &mockBackend{name: "fallback", transcript: expected}

	r.Register("primary", primary)
	r.Register("fallback", fallback)
	r.SetFallback("fallback")

	result, err := r.TranscribeWithFallback("test.wav", TranscribeOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Backend != "fallback" {
		t.Errorf("expected fallback backend result, got %q", result.Backend)
	}
}

func TestTranscribeWithFallback_BothFail(t *testing.T) {
	r := NewRegistry()
	primary := &mockBackend{name: "primary", err: fmt.Errorf("primary down")}
	fallback := &mockBackend{name: "fallback", err: fmt.Errorf("fallback down")}

	r.Register("primary", primary)
	r.Register("fallback", fallback)
	r.SetFallback("fallback")

	_, err := r.TranscribeWithFallback("test.wav", TranscribeOptions{})
	if err == nil {
		t.Fatal("expected error when both backends fail")
	}
	if !strings.Contains(err.Error(), "primary") || !strings.Contains(err.Error(), "fallback") {
		t.Errorf("expected error to mention both backends, got: %v", err)
	}
}

func TestTranscribeWithFallback_NoPrimary(t *testing.T) {
	r := NewRegistry()

	_, err := r.TranscribeWithFallback("test.wav", TranscribeOptions{})
	if err == nil {
		t.Fatal("expected error with no primary backend")
	}
	if !strings.Contains(err.Error(), "no primary backend") {
		t.Errorf("expected 'no primary backend' error, got: %v", err)
	}
}

func TestTranscribeWithFallback_NoFallback(t *testing.T) {
	r := NewRegistry()
	primary := &mockBackend{name: "primary", err: fmt.Errorf("primary down")}

	r.Register("primary", primary)

	_, err := r.TranscribeWithFallback("test.wav", TranscribeOptions{})
	if err == nil {
		t.Fatal("expected error when primary fails and no fallback configured")
	}
	if !strings.Contains(err.Error(), "primary") {
		t.Errorf("expected error to mention primary backend, got: %v", err)
	}
	if strings.Contains(err.Error(), "fallback") && strings.Contains(err.Error(), "also failed") {
		t.Errorf("error should not mention fallback failure when no fallback configured, got: %v", err)
	}
}
