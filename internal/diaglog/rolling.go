package diaglog

import (
	"os"
	"sync"
)

// rollingWriter is a mutex-guarded append-only writer that truncates the file
// to zero when the next write would exceed maxSize. This preserves the most
// recent entries (the overflowing write is written fresh after truncation).
type rollingWriter struct {
	path    string
	maxSize int64
	f       *os.File
	size    int64
	mu      sync.Mutex
}

// newRollingWriter opens path (creating it if needed) and returns a writer
// capped at maxSize bytes.
func newRollingWriter(path string, maxSize int64) (*rollingWriter, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	info, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, err
	}
	return &rollingWriter{path: path, maxSize: maxSize, f: f, size: info.Size()}, nil
}

// Write appends p to the file. If size+len(p) would exceed maxSize, the file
// is truncated to zero first. A Sync is called after every write for durability.
func (rw *rollingWriter) Write(p []byte) (int, error) {
	rw.mu.Lock()
	defer rw.mu.Unlock()

	if rw.size+int64(len(p)) > rw.maxSize {
		if err := rw.f.Truncate(0); err != nil {
			return 0, err
		}
		if _, err := rw.f.Seek(0, 0); err != nil {
			return 0, err
		}
		rw.size = 0
	}

	n, err := rw.f.Write(p)
	if err != nil {
		return n, err
	}
	rw.size += int64(n)
	_ = rw.f.Sync()
	return n, nil
}

// close flushes and closes the file.
func (rw *rollingWriter) close() error {
	_ = rw.f.Sync()
	return rw.f.Close()
}
