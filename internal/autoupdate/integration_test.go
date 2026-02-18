package autoupdate

import (
	"os"
	"path/filepath"
	"testing"
)

// TestUpdateE2E performs a real end-to-end test of the update mechanism.
// It simulates running an old version (0.1.0) and verifies that:
//   - IsUpdateAvailable correctly detects a newer GitHub release
//   - DownloadAndInstall actually downloads, extracts, and installs the binaries
//
// This test makes real network calls to GitHub. Skip with -short flag.
func TestUpdateE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test in short mode")
	}

	// Simulate an old installed version
	oldVersion := "0.1.0"
	tmpDir := t.TempDir()

	t.Logf("Simulated old version: %s", oldVersion)
	t.Logf("Install dir: %s", tmpDir)

	checker := NewUpdateChecker("tiroq", "memofy", oldVersion, tmpDir)
	checker.SetChannel(ChannelStable)

	// ── Step 1: version check ────────────────────────────────────────────────
	available, release, err := checker.IsUpdateAvailable()
	if err != nil {
		t.Fatalf("IsUpdateAvailable error: %v", err)
	}
	if !available || release == nil {
		t.Fatalf("expected update to be available from %s, got available=%v release=%v",
			oldVersion, available, release)
	}
	t.Logf("✓ Update detected: %s is available (current=%s)", release.TagName, oldVersion)
	t.Logf("  Assets:")
	for _, a := range release.Assets {
		t.Logf("    %s (%d bytes)", a.Name, a.Size)
	}

	// ── Step 2: asset matching ───────────────────────────────────────────────
	asset := checker.findBinaryAsset(release)
	if asset == nil {
		t.Fatalf("findBinaryAsset returned nil — no matching asset for this platform in release %s", release.TagName)
	}
	t.Logf("✓ Asset found: %s", asset.Name)

	// ── Step 3: download + install ───────────────────────────────────────────
	t.Logf("Downloading and installing to %s ...", tmpDir)
	if err := checker.DownloadAndInstall(release); err != nil {
		t.Fatalf("DownloadAndInstall error: %v", err)
	}

	// ── Step 4: verify installed binaries ────────────────────────────────────
	for _, bin := range []string{"memofy-core", "memofy-ui"} {
		path := filepath.Join(tmpDir, bin)
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("binary %s NOT found at %s: %v", bin, path, err)
			continue
		}
		if info.Size() == 0 {
			t.Errorf("binary %s has zero size", bin)
			continue
		}
		// Verify it's executable
		if info.Mode()&0111 == 0 {
			t.Errorf("binary %s is not executable (mode=%s)", bin, info.Mode())
			continue
		}
		t.Logf("✓ %s: %d bytes, mode=%s", bin, info.Size(), info.Mode())
	}
}

// TestIsUpdateAvailableWithOldVersion verifies the version-check logic using
// real GitHub data (no installation, just API call).
func TestIsUpdateAvailableWithOldVersion(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test in short mode")
	}

	cases := []struct {
		version   string
		wantAvail bool
	}{
		{"0.0.1", true},  // very old — update must be available
		{"99.0.0", false}, // future version — no update available
	}

	for _, tc := range cases {
		t.Run("v"+tc.version, func(t *testing.T) {
			checker := NewUpdateChecker("tiroq", "memofy", tc.version, t.TempDir())
			checker.SetChannel(ChannelStable)

			available, release, err := checker.IsUpdateAvailable()
			if err != nil {
				t.Fatalf("IsUpdateAvailable error: %v", err)
			}
			if available != tc.wantAvail {
				releaseTag := "<nil>"
				if release != nil {
					releaseTag = release.TagName
				}
				t.Errorf("IsUpdateAvailable(%s) = %v (release=%s), want %v",
					tc.version, available, releaseTag, tc.wantAvail)
			} else {
				if release != nil {
					t.Logf("✓ v%s → update available: %s", tc.version, release.TagName)
				} else {
					t.Logf("✓ v%s → no update (already latest)", tc.version)
				}
			}
		})
	}
}
