package wav

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func TestCreate_WritesHeader(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.wav")

	w, err := Create(path, 48000, 2)
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if err := w.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}

	data, _ := os.ReadFile(path)
	if len(data) < 44 {
		t.Fatalf("file too small: %d bytes", len(data))
	}

	// Check RIFF header
	if string(data[0:4]) != "RIFF" {
		t.Errorf("missing RIFF header")
	}
	if string(data[8:12]) != "WAVE" {
		t.Errorf("missing WAVE marker")
	}
	if string(data[12:16]) != "fmt " {
		t.Errorf("missing fmt chunk")
	}

	// PCM format = 1
	format := binary.LittleEndian.Uint16(data[20:22])
	if format != 1 {
		t.Errorf("format = %d, want 1 (PCM)", format)
	}

	// Channels
	ch := binary.LittleEndian.Uint16(data[22:24])
	if ch != 2 {
		t.Errorf("channels = %d, want 2", ch)
	}

	// Sample rate
	sr := binary.LittleEndian.Uint32(data[24:28])
	if sr != 48000 {
		t.Errorf("sample rate = %d, want 48000", sr)
	}
}

func TestWrite_AudioData(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.wav")

	w, err := Create(path, 48000, 2)
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	// Write 1 second of silence (48000 * 2 channels)
	samples := make([]float32, 48000*2)
	if err := w.Write(samples); err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	dur := w.DurationSeconds()
	if dur < 0.99 || dur > 1.01 {
		t.Errorf("duration = %f, want ~1.0", dur)
	}

	if err := w.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}

	// Check file size: 44 header + 48000*2*2 bytes = 44 + 192000
	info, _ := os.Stat(path)
	expected := int64(44 + 48000*2*2)
	if info.Size() != expected {
		t.Errorf("file size = %d, want %d", info.Size(), expected)
	}

	// Verify header data size matches
	data, _ := os.ReadFile(path)
	dataSize := binary.LittleEndian.Uint32(data[40:44])
	if dataSize != uint32(48000*2*2) {
		t.Errorf("data chunk size = %d, want %d", dataSize, 48000*2*2)
	}
}

func TestWrite_ClampingValues(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.wav")

	w, err := Create(path, 8000, 1)
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	// Values beyond [-1, 1] should be clamped
	samples := []float32{-2.0, -1.0, 0.0, 1.0, 2.0}
	if err := w.Write(samples); err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	if err := w.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}

	// File should be valid (no panic, no error)
	info, _ := os.Stat(path)
	if info.Size() != 44+10 { // 5 samples * 2 bytes
		t.Errorf("file size = %d, want %d", info.Size(), 44+10)
	}
}

func TestCreate_DoubleClose(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.wav")

	w, _ := Create(path, 8000, 1)
	w.Close()

	// Second close should be a no-op
	if err := w.Close(); err != nil {
		t.Errorf("double close should be ok: %v", err)
	}
}

func TestWrite_AfterClose(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.wav")

	w, _ := Create(path, 8000, 1)
	w.Close()

	err := w.Write([]float32{0.5})
	if err == nil {
		t.Error("should error on write after close")
	}
}

func TestPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.wav")

	w, _ := Create(path, 8000, 1)
	defer w.Close()

	if w.Path() != path {
		t.Errorf("Path() = %q, want %q", w.Path(), path)
	}
}
