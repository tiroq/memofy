package transcript

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tiroq/memofy/internal/asr"
)

// T027: Transcript file writers for batch ASR output. FR-013

// WriteText writes a plain text transcript with one segment per line, each
// prefixed by its timestamp in [HH:MM:SS] format. The file is written
// atomically (temp file + rename) to avoid partial writes.
func WriteText(path string, t *asr.Transcript) error {
	var b strings.Builder
	for _, seg := range t.Segments {
		ts := formatTextTimestamp(seg.Start)
		fmt.Fprintf(&b, "[%s] %s\n", ts, seg.Text)
	}
	return atomicWrite(path, []byte(b.String()))
}

// WriteSRT writes a SubRip (.srt) subtitle file. Each segment is numbered
// sequentially with start/end timestamps in HH:MM:SS,mmm format.
func WriteSRT(path string, t *asr.Transcript) error {
	var b strings.Builder
	for i, seg := range t.Segments {
		if i > 0 {
			b.WriteByte('\n')
		}
		fmt.Fprintf(&b, "%d\n", i+1)
		fmt.Fprintf(&b, "%s --> %s\n", formatSRTTimestamp(seg.Start), formatSRTTimestamp(seg.End))
		fmt.Fprintf(&b, "%s\n", seg.Text)
	}
	return atomicWrite(path, []byte(b.String()))
}

// WriteVTT writes a WebVTT (.vtt) subtitle file. Each segment has
// start/end timestamps in HH:MM:SS.mmm format, preceded by the WEBVTT header.
func WriteVTT(path string, t *asr.Transcript) error {
	var b strings.Builder
	b.WriteString("WEBVTT\n")
	for _, seg := range t.Segments {
		b.WriteByte('\n')
		fmt.Fprintf(&b, "%s --> %s\n", formatVTTTimestamp(seg.Start), formatVTTTimestamp(seg.End))
		fmt.Fprintf(&b, "%s\n", seg.Text)
	}
	return atomicWrite(path, []byte(b.String()))
}

// WriteAll writes the transcript in every requested format. basePath is the
// file path without extension (e.g. "/recordings/2024-01-15_meeting").
// Supported formats: "txt", "srt", "vtt". If formats is nil or empty,
// defaults to ["txt"]. Returns a combined error listing all failures.
func WriteAll(basePath string, t *asr.Transcript, formats []string) error {
	if len(formats) == 0 {
		formats = []string{"txt"}
	}
	var errs []string
	for _, f := range formats {
		var err error
		switch f {
		case "txt":
			err = WriteText(basePath+".txt", t)
		case "srt":
			err = WriteSRT(basePath+".srt", t)
		case "vtt":
			err = WriteVTT(basePath+".vtt", t)
		default:
			errs = append(errs, fmt.Sprintf("unknown format %q", f))
			continue
		}
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", f, err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("transcript write errors: %s", strings.Join(errs, "; "))
	}
	return nil
}

// formatTextTimestamp formats a duration as HH:MM:SS for plain text output.
func formatTextTimestamp(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

// formatSRTTimestamp formats a duration as HH:MM:SS,mmm (SRT subtitle format).
func formatSRTTimestamp(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	ms := int(d.Milliseconds()) % 1000
	return fmt.Sprintf("%02d:%02d:%02d,%03d", h, m, s, ms)
}

// formatVTTTimestamp formats a duration as HH:MM:SS.mmm (WebVTT format).
func formatVTTTimestamp(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	ms := int(d.Milliseconds()) % 1000
	return fmt.Sprintf("%02d:%02d:%02d.%03d", h, m, s, ms)
}

// atomicWrite writes data to path atomically using a temp file + rename.
func atomicWrite(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}

	tmpFile, err := os.CreateTemp(dir, "transcript-*.tmp")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	// Ensure cleanup on error.
	defer func() {
		if tmpFile != nil {
			tmpFile.Close()
			os.Remove(tmpPath)
		}
	}()

	if _, err := tmpFile.Write(data); err != nil {
		return fmt.Errorf("writing transcript: %w", err)
	}
	if err := tmpFile.Sync(); err != nil {
		return fmt.Errorf("syncing transcript: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("closing transcript: %w", err)
	}
	tmpFile = nil // prevent defer cleanup

	if err := os.Rename(tmpPath, path); err != nil {
		// Clean up temp file on rename failure.
		os.Remove(tmpPath)
		return fmt.Errorf("renaming transcript: %w", err)
	}
	return nil
}
