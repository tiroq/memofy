package diaglog

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"runtime"
	"time"
)

// Version is injected at link time from the main package; defaults to "dev".
var Version = "dev"

// DiagBundle is the first line written to the export file (valid NDJSON).
type DiagBundle struct {
	ExportedAt    string `json:"exported_at"`
	MemofyVersion string `json:"memofy_version"`
	GoVersion     string `json:"go_version"`
	OS            string `json:"os"`
	Arch          string `json:"arch"`
	LogFile       string `json:"log_file"`
	EntryCount    int    `json:"entry_count"`
}

// Export reads logPath, counts its NDJSON entries, prepends a DiagBundle
// metadata line, and writes the result to dest/memofy-diag-<ts>.ndjson.
// Returns the written file path and number of log lines included (FR-006).
func Export(logPath, dest string) (path string, lines int, err error) {
	src, err := os.Open(logPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", 0, fmt.Errorf("log file not found at %s: %w", logPath, os.ErrNotExist)
		}
		return "", 0, fmt.Errorf("log file unreadable: %w", err)
	}
	defer func() { _ = src.Close() }()

	// Buffer all lines (log is capped at 10 MB so this is safe).
	var rawLines [][]byte
	scanner := bufio.NewScanner(src)
	scanner.Buffer(make([]byte, 10*1024*1024), 10*1024*1024)
	for scanner.Scan() {
		line := make([]byte, len(scanner.Bytes()))
		copy(line, scanner.Bytes())
		rawLines = append(rawLines, line)
	}
	if serr := scanner.Err(); serr != nil {
		return "", 0, fmt.Errorf("log file unreadable: %w", serr)
	}

	tstamp := time.Now().UTC().Format("20060102T150405")
	outPath := dest + "/memofy-diag-" + tstamp + ".ndjson"

	out, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return "", 0, fmt.Errorf("output file could not be created: %w", err)
	}
	defer func() { _ = out.Close() }()

	// Write bundle header.
	bundle := DiagBundle{
		ExportedAt:    time.Now().UTC().Format(time.RFC3339),
		MemofyVersion: Version,
		GoVersion:     runtime.Version(),
		OS:            runtime.GOOS,
		Arch:          runtime.GOARCH,
		LogFile:       logPath,
		EntryCount:    len(rawLines),
	}
	header, merr := json.Marshal(bundle)
	if merr != nil {
		return "", 0, merr
	}
	if _, err := out.Write(append(header, '\n')); err != nil {
		return "", 0, err
	}

	// Write source log lines verbatim.
	w := bufio.NewWriter(out)
	for _, line := range rawLines {
		if _, err := w.Write(append(line, '\n')); err != nil {
			return "", 0, err
		}
	}
	if err := w.Flush(); err != nil {
		return "", 0, err
	}

	return outPath, len(rawLines), nil
}
