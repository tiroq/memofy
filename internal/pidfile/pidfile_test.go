package pidfile

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestNewPIDFile(t *testing.T) {
	tmpDir := t.TempDir()
	pidPath := filepath.Join(tmpDir, "test.pid")

	// Create first PID file
	pf, err := New(pidPath)
	if err != nil {
		t.Fatalf("Failed to create PID file: %v", err)
	}
	defer func() {
		if err := pf.Remove(); err != nil {
			t.Logf("Warning: failed to remove PID file: %v", err)
		}
	}()

	// Verify PID file exists
	if _, err := os.Stat(pidPath); os.IsNotExist(err) {
		t.Error("PID file was not created")
	}

	// Verify PID file contains current process PID
	data, err := os.ReadFile(pidPath)
	if err != nil {
		t.Fatalf("Failed to read PID file: %v", err)
	}

	pidStr := strings.TrimSpace(string(data))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		t.Fatalf("Invalid PID in file: %s", pidStr)
	}

	if pid != os.Getpid() {
		t.Errorf("PID mismatch: got %d, want %d", pid, os.Getpid())
	}
}

func TestDuplicateInstance(t *testing.T) {
	tmpDir := t.TempDir()
	pidPath := filepath.Join(tmpDir, "test.pid")

	// Create first PID file
	pf1, err := New(pidPath)
	if err != nil {
		t.Fatalf("Failed to create first PID file: %v", err)
	}
	defer func() {
		if err := pf1.Remove(); err != nil {
			t.Logf("Warning: failed to remove PID file: %v", err)
		}
	}()

	// Try to create second PID file (should fail)
	_, err = New(pidPath)
	if err == nil {
		t.Error("Expected error when creating duplicate PID file, got nil")
	}

	if !strings.Contains(err.Error(), "already running") {
		t.Errorf("Expected 'already running' error, got: %v", err)
	}
}

func TestStalePIDFile(t *testing.T) {
	tmpDir := t.TempDir()
	pidPath := filepath.Join(tmpDir, "test.pid")

	// Create a stale PID file with a non-existent PID
	stalePID := 99999
	err := os.WriteFile(pidPath, []byte(strconv.Itoa(stalePID)+"\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to create stale PID file: %v", err)
	}

	// Should successfully create new PID file (stale one should be removed)
	pf, err := New(pidPath)
	if err != nil {
		t.Fatalf("Failed to create PID file after removing stale one: %v", err)
	}
	defer pf.Remove()

	// Verify PID file contains current process PID
	data, err := os.ReadFile(pidPath)
	if err != nil {
		t.Fatalf("Failed to read PID file: %v", err)
	}

	pidStr := strings.TrimSpace(string(data))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		t.Fatalf("Invalid PID in file: %s", pidStr)
	}

	if pid != os.Getpid() {
		t.Errorf("PID mismatch after stale removal: got %d, want %d", pid, os.Getpid())
	}
}

func TestRemovePIDFile(t *testing.T) {
	tmpDir := t.TempDir()
	pidPath := filepath.Join(tmpDir, "test.pid")

	// Create PID file
	pf, err := New(pidPath)
	if err != nil {
		t.Fatalf("Failed to create PID file: %v", err)
	}

	// Remove PID file
	if err := pf.Remove(); err != nil {
		t.Errorf("Failed to remove PID file: %v", err)
	}

	// Verify PID file is gone
	if _, err := os.Stat(pidPath); !os.IsNotExist(err) {
		t.Error("PID file still exists after removal")
	}
}

func TestRemoveOnlyOwnPID(t *testing.T) {
	tmpDir := t.TempDir()
	pidPath := filepath.Join(tmpDir, "test.pid")

	// Create PID file
	pf, err := New(pidPath)
	if err != nil {
		t.Fatalf("Failed to create PID file: %v", err)
	}

	// Manually change PID file to different PID
	differentPID := os.Getpid() + 1
	err = os.WriteFile(pidPath, []byte(strconv.Itoa(differentPID)+"\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to write different PID: %v", err)
	}

	// Try to remove - should not remove since it's not our PID
	pf.Remove()

	// Verify PID file still exists
	if _, err := os.Stat(pidPath); os.IsNotExist(err) {
		t.Error("PID file was removed even though it contained different PID")
	}

	// Verify it contains the different PID
	data, err := os.ReadFile(pidPath)
	if err != nil {
		t.Fatalf("Failed to read PID file: %v", err)
	}

	pidStr := strings.TrimSpace(string(data))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		t.Fatalf("Invalid PID in file: %s", pidStr)
	}

	if pid != differentPID {
		t.Errorf("PID changed unexpectedly: got %d, want %d", pid, differentPID)
	}
}

func TestGetPIDFilePath(t *testing.T) {
	path := GetPIDFilePath("test-app")

	homeDir := os.Getenv("HOME")
	expectedPath := filepath.Join(homeDir, ".cache", "memofy", "test-app.pid")

	if path != expectedPath {
		t.Errorf("GetPIDFilePath returned wrong path: got %s, want %s", path, expectedPath)
	}
}

func TestIsProcessRunning(t *testing.T) {
	// Test with current process (should be running)
	if !isProcessRunning(os.Getpid()) {
		t.Error("Current process should be detected as running")
	}

	// Test with non-existent PID (should not be running)
	if isProcessRunning(99999) {
		t.Error("Non-existent process should not be detected as running")
	}
}
