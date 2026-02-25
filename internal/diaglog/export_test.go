package diaglog

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"
)

func seedLogFile(t *testing.T, n int) string {
	t.Helper()
	tmp := t.TempDir() + "/seed.ndjson"
	f, err := os.Create(tmp)
	if err != nil {
		t.Fatalf("create seed: %v", err)
	}
	defer func() { _ = f.Close() }()
	for i := 0; i < n; i++ {
		_, _ = fmt.Fprintf(f, "{\"ts\":\"2026-01-01T00:00:00Z\",\"component\":\"test\",\"event\":\"e%d\"}\n", i)
	}
	return tmp
}

func TestExportWritesBundleHeader(t *testing.T) {
	src := seedLogFile(t, 10)
	dest := t.TempDir()

	path, lines, err := Export(src, dest)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if lines != 10 {
		t.Errorf("lines: want 10, got %d", lines)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open output: %v", err)
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		t.Fatal("no first line in output")
	}
	var bundle DiagBundle
	if err := json.Unmarshal(scanner.Bytes(), &bundle); err != nil {
		t.Fatalf("unmarshal bundle header: %v", err)
	}
	if bundle.EntryCount != 10 {
		t.Errorf("entry_count: want 10, got %d", bundle.EntryCount)
	}
	if bundle.GoVersion == "" {
		t.Error("go_version missing")
	}
	if bundle.OS == "" {
		t.Error("os missing")
	}
}

func TestExportContainsAllLines(t *testing.T) {
	src := seedLogFile(t, 5)
	dest := t.TempDir()

	outPath, _, err := Export(src, dest)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}

	readLines := func(p string, skip int) []string {
		f, _ := os.Open(p)
		defer f.Close()
		var ls []string
		s := bufio.NewScanner(f)
		for s.Scan() {
			if skip > 0 {
				skip--
				continue
			}
			ls = append(ls, s.Text())
		}
		return ls
	}

	srcLines := readLines(src, 0)
	outLines := readLines(outPath, 1)

	if len(outLines) != len(srcLines) {
		t.Fatalf("want %d lines, got %d", len(srcLines), len(outLines))
	}
	for i := range srcLines {
		if outLines[i] != srcLines[i] {
			t.Errorf("line %d mismatch", i)
		}
	}
}

func TestExportMissingFile(t *testing.T) {
	_, _, err := Export("/nonexistent/path/memofy-debug.log", t.TempDir())
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("want os.ErrNotExist, got %v", err)
	}
}

func TestExportCompletesUnder10s(t *testing.T) {
	tmp := t.TempDir() + "/large.ndjson"
	f, err := os.Create(tmp)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	for i := 0; i < 10000; i++ {
		_, _ = fmt.Fprintf(f, "{\"ts\":\"2026-01-01T00:00:00Z\",\"component\":\"test\",\"event\":\"e%d\"}\n", i)
	}
	_ = f.Close()

	start := time.Now()
	_, _, err = Export(tmp, t.TempDir())
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 10*time.Second {
		t.Errorf("Export took %v, want < 10s", elapsed)
	}
}
