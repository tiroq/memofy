// Package wav provides a simple WAV file writer for PCM float32 audio.
package wav

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"sync"
)

// Writer writes PCM audio data to a WAV file.
// It writes a placeholder header on creation and updates it on Close()
// with the actual data size, making it crash-safe for partial writes.
type Writer struct {
	f          *os.File
	sampleRate int
	channels   int
	dataBytes  int64
	mu         sync.Mutex
	closed     bool
}

// Create opens a new WAV file for writing.
func Create(path string, sampleRate, channels int) (*Writer, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("create wav: %w", err)
	}

	w := &Writer{
		f:          f,
		sampleRate: sampleRate,
		channels:   channels,
	}

	// Write placeholder header (44 bytes), will be updated on Close().
	if err := w.writeHeader(0); err != nil {
		f.Close()
		os.Remove(path)
		return nil, err
	}

	return w, nil
}

// Write appends interleaved float32 samples to the WAV file as 16-bit PCM.
func (w *Writer) Write(samples []float32) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return fmt.Errorf("wav writer is closed")
	}

	buf := make([]byte, len(samples)*2) // 16-bit = 2 bytes per sample
	for i, s := range samples {
		// Clamp to [-1, 1]
		if s > 1.0 {
			s = 1.0
		} else if s < -1.0 {
			s = -1.0
		}
		v := int16(s * math.MaxInt16)
		binary.LittleEndian.PutUint16(buf[i*2:], uint16(v))
	}

	n, err := w.f.Write(buf)
	w.dataBytes += int64(n)
	if err != nil {
		return fmt.Errorf("write wav data: %w", err)
	}
	return nil
}

// Close finalizes the WAV file by updating the header with actual sizes.
func (w *Writer) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return nil
	}
	w.closed = true

	// Seek to beginning and rewrite header with actual size.
	if _, err := w.f.Seek(0, io.SeekStart); err != nil {
		w.f.Close()
		return fmt.Errorf("seek for header update: %w", err)
	}
	if err := w.writeHeader(w.dataBytes); err != nil {
		w.f.Close()
		return fmt.Errorf("update header: %w", err)
	}

	return w.f.Close()
}

// Path returns the file path.
func (w *Writer) Path() string {
	return w.f.Name()
}

// DataBytes returns the number of audio data bytes written so far.
func (w *Writer) DataBytes() int64 {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.dataBytes
}

// DurationSeconds returns the duration of written audio in seconds.
func (w *Writer) DurationSeconds() float64 {
	w.mu.Lock()
	defer w.mu.Unlock()
	bytesPerSecond := int64(w.sampleRate) * int64(w.channels) * 2 // 16-bit
	if bytesPerSecond == 0 {
		return 0
	}
	return float64(w.dataBytes) / float64(bytesPerSecond)
}

// writeHeader writes a 44-byte WAV header.
func (w *Writer) writeHeader(dataSize int64) error {
	bitsPerSample := 16
	byteRate := w.sampleRate * w.channels * bitsPerSample / 8
	blockAlign := w.channels * bitsPerSample / 8

	h := make([]byte, 44)

	// RIFF header
	copy(h[0:4], "RIFF")
	binary.LittleEndian.PutUint32(h[4:8], uint32(36+dataSize))
	copy(h[8:12], "WAVE")

	// fmt chunk
	copy(h[12:16], "fmt ")
	binary.LittleEndian.PutUint32(h[16:20], 16) // chunk size
	binary.LittleEndian.PutUint16(h[20:22], 1)  // PCM format
	binary.LittleEndian.PutUint16(h[22:24], uint16(w.channels))
	binary.LittleEndian.PutUint32(h[24:28], uint32(w.sampleRate))
	binary.LittleEndian.PutUint32(h[28:32], uint32(byteRate))
	binary.LittleEndian.PutUint16(h[32:34], uint16(blockAlign))
	binary.LittleEndian.PutUint16(h[34:36], uint16(bitsPerSample))

	// data chunk
	copy(h[36:40], "data")
	binary.LittleEndian.PutUint32(h[40:44], uint32(dataSize))

	_, err := w.f.Write(h)
	return err
}
