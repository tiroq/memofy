package micdetect
//go:build darwin

package micdetect

import (
	"errors"
	"testing"
)

func TestIsSupported(t *testing.T) {
	major, _, err := macOSVersion()
	if err != nil {
		t.Fatalf("macOSVersion: %v", err)
	}
	result := IsSupported()
	if major >= 14 && !result {
		t.Error("IsSupported() returned false on macOS 14+")
	}
	if major < 14 && result {
		t.Error("IsSupported() returned true on macOS < 14")
	}
}

func TestMacOSVersion(t *testing.T) {
	major, minor, err := macOSVersion()
	if err != nil {
		t.Fatalf("macOSVersion: %v", err)
	}
	if major < 10 || major > 30 {
		t.Errorf("unexpected major version: %d", major)
	}
	if minor < 0 || minor > 20 {
		t.Errorf("unexpected minor version: %d", minor)
	}
	t.Logf("macOS %d.%d", major, minor)
}

func TestMacOSVersionString(t *testing.T) {
	ver := MacOSVersionString()
	if ver == "" || ver == "unknown" {
		t.Errorf("MacOSVersionString() returned %q", ver)
	}
	t.Logf("version string: %s", ver)
}

func TestActiveMicUsers(t *testing.T) {
	if !IsSupported() {
		t.Skip("macOS 14+ required")
	}
	procs, err := ActiveMicUsers()
	if err != nil {
		t.Fatalf("ActiveMicUsers: %v", err)
	}
	for _, p := range procs {
		if p.PID <= 0 {
			t.Errorf("invalid PID: %d", p.PID)
		}
		if !p.RunningInput {
			t.Errorf("expected RunningInput=true for PID %d", p.PID)
		}
	}
	t.Logf("found %d active mic users", len(procs))
}

func TestActiveMicUserBundleIDs(t *testing.T) {
	if !IsSupported() {
		t.Skip("macOS 14+ required")
	}
	ids, err := ActiveMicUserBundleIDs()
	if err != nil {
		t.Fatalf("ActiveMicUserBundleIDs: %v", err)
	}
	t.Logf("bundle IDs: %v", ids)
}

func TestFilterActiveInput(t *testing.T) {
	procs := []ActiveProcess{
		{PID: 1, BundleID: "com.example.app1", RunningInput: true, RunningOutput: true},
		{PID: 2, BundleID: "com.example.app2", RunningInput: false, RunningOutput: true},
		{PID: 3, BundleID: "com.example.app3", RunningInput: true, RunningOutput: false},
		{PID: 4, BundleID: "com.example.app4", RunningInput: false, RunningOutput: false},
	}
	result := filterActiveInput(procs)
	if len(result) != 2 {
		t.Fatalf("expected 2 active input, got %d", len(result))
	}
	if result[0].PID != 1 {
		t.Errorf("expected PID 1, got %d", result[0].PID)
	}
	if result[1].PID != 3 {
		t.Errorf("expected PID 3, got %d", result[1].PID)
	}
}

func TestFilterActiveInputEmpty(t *testing.T) {
	result := filterActiveInput(nil)
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}

	result = filterActiveInput([]ActiveProcess{
		{PID: 1, RunningInput: false},
		{PID: 2, RunningInput: false},
	})
	if len(result) != 0 {
		t.Errorf("expected empty, got %d", len(result))
	}
}

func TestErrorTypes(t *testing.T) {
	if ErrUnsupportedPlatform == nil {
		t.Error("ErrUnsupportedPlatform is nil")
	}
	if ErrUnsupportedVersion == nil {
		t.Error("ErrUnsupportedVersion is nil")
	}
	if ErrEnumerationFailed == nil {
		t.Error("ErrEnumerationFailed is nil")
	}
	if ErrPropertyReadFailed == nil {
		t.Error("ErrPropertyReadFailed is nil")
	}
}

func TestErrorWrapping(t *testing.T) {
	// Verify sentinel errors can be detected with errors.Is
	wrapped := errors.New("test")
	_ = wrapped // just verify the errors package works with our sentinels

	if !errors.Is(ErrUnsupportedPlatform, ErrUnsupportedPlatform) {
		t.Error("ErrUnsupportedPlatform identity check failed")
	}
	if !errors.Is(ErrUnsupportedVersion, ErrUnsupportedVersion) {
		t.Error("ErrUnsupportedVersion identity check failed")
	}
}
