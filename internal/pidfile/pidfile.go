package pidfile

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// PIDFile manages a PID file for preventing duplicate instances
type PIDFile struct {
	path string
	pid  int
}

// New creates a new PID file at the specified path
// Returns an error if a PID file already exists with a running process
func New(path string) (*PIDFile, error) {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create PID directory: %w", err)
	}

	// Check if PID file already exists
	if data, err := os.ReadFile(path); err == nil {
		// PID file exists, check if process is running
		pidStr := strings.TrimSpace(string(data))
		if existingPID, err := strconv.Atoi(pidStr); err == nil {
			if isProcessRunning(existingPID) {
				return nil, fmt.Errorf("another instance is already running (PID %d)", existingPID)
			}
			// Process not running, remove stale PID file
			if err := os.Remove(path); err != nil {
				return nil, fmt.Errorf("failed to remove stale PID file: %w", err)
			}
		}
	}

	// Write current process PID
	currentPID := os.Getpid()
	if err := os.WriteFile(path, []byte(fmt.Sprintf("%d\n", currentPID)), 0644); err != nil {
		return nil, fmt.Errorf("failed to write PID file: %w", err)
	}

	return &PIDFile{
		path: path,
		pid:  currentPID,
	}, nil
}

// Remove deletes the PID file
func (p *PIDFile) Remove() error {
	if p == nil {
		return nil
	}

	// Only remove if it contains our PID
	if data, err := os.ReadFile(p.path); err == nil {
		pidStr := strings.TrimSpace(string(data))
		if pid, err := strconv.Atoi(pidStr); err == nil && pid == p.pid {
			return os.Remove(p.path)
		}
	}

	return nil
}

// isProcessRunning checks if a process with the given PID is running
func isProcessRunning(pid int) bool {
	// Send signal 0 to check if process exists
	// This doesn't actually send a signal, just checks if we can
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// On Unix systems, FindProcess always succeeds, so we need to actually check
	err = process.Signal(syscall.Signal(0))
	if err == nil {
		return true
	}

	// Check for specific error types
	if err == syscall.ESRCH {
		// No such process
		return false
	}

	if err == syscall.EPERM {
		// Process exists but we don't have permission to signal it
		return true
	}

	return false
}

// GetPIDFilePath returns the standard PID file path for a given application name
func GetPIDFilePath(appName string) string {
	homeDir := os.Getenv("HOME")
	return filepath.Join(homeDir, ".cache", "memofy", appName+".pid")
}
