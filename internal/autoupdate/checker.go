package autoupdate

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ReleaseChannel defines which releases to check for
type ReleaseChannel string

const (
	ChannelStable     ReleaseChannel = "stable"     // Only stable releases
	ChannelPrerelease ReleaseChannel = "prerelease" // Stable + pre-releases (beta, rc)
	ChannelDev        ReleaseChannel = "dev"        // All releases including dev builds
)

// Release represents a GitHub release
type Release struct {
	TagName    string    `json:"tag_name"`
	Name       string    `json:"name"`
	Body       string    `json:"body"`
	Published  time.Time `json:"published_at"`
	Assets     []Asset   `json:"assets"`
	Prerelease bool      `json:"prerelease"`
	Draft      bool      `json:"draft"`
}

// Asset represents a release asset (binary)
type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

// UpdateChecker handles version checking and updates
type UpdateChecker struct {
	repoOwner      string
	repoName       string
	currentVersion string
	apiURL         string
	installDir     string
	channel        ReleaseChannel
}

// NewUpdateChecker creates a new update checker
func NewUpdateChecker(owner, repo, currentVersion, installDir string) *UpdateChecker {
	return &UpdateChecker{
		repoOwner:      owner,
		repoName:       repo,
		currentVersion: currentVersion,
		apiURL:         fmt.Sprintf("https://api.github.com/repos/%s/%s", owner, repo),
		installDir:     installDir,
		channel:        ChannelStable, // Default to stable releases
	}
}

// SetChannel sets the release channel for this checker
func (uc *UpdateChecker) SetChannel(channel ReleaseChannel) {
	uc.channel = channel
}

// GetLatestRelease fetches the latest release from GitHub matching the current channel
func (uc *UpdateChecker) GetLatestRelease() (*Release, error) {
	// For stable channel, use the GitHub "latest" endpoint which returns the latest stable release
	// For other channels, fetch all releases and filter
	if uc.channel == ChannelStable {
		return uc.getLatestStableRelease()
	}

	return uc.getLatestReleaseInChannel()
}

// getLatestStableRelease fetches the latest stable (non-prerelease, non-draft) release
func (uc *UpdateChecker) getLatestStableRelease() (*Release, error) {
	url := fmt.Sprintf("%s/releases/latest", uc.apiURL)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch release: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("Warning: failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github API returned status %d", resp.StatusCode)
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to parse release: %w", err)
	}

	return &release, nil
}

// getLatestReleaseInChannel fetches all releases and returns the latest matching the current channel
func (uc *UpdateChecker) getLatestReleaseInChannel() (*Release, error) {
	url := fmt.Sprintf("%s/releases?per_page=30", uc.apiURL)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch releases: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("Warning: failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github API returned status %d", resp.StatusCode)
	}

	var releases []Release
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("failed to parse releases: %w", err)
	}

	// Find the first (latest) release matching our channel filter
	for _, release := range releases {
		if uc.matchesChannel(&release) {
			return &release, nil
		}
	}

	return nil, fmt.Errorf("no releases found matching channel %s", uc.channel)
}

// matchesChannel checks if a release matches the current channel setting
func (uc *UpdateChecker) matchesChannel(release *Release) bool {
	// Skip drafts in all channels
	if release.Draft {
		return false
	}

	switch uc.channel {
	case ChannelStable:
		// Only non-prerelease stable releases
		return !release.Prerelease
	case ChannelPrerelease:
		// Both stable and prerelease versions
		return true
	case ChannelDev:
		// All releases
		return true
	default:
		return false
	}
}

// IsUpdateAvailable checks if a newer version is available
func (uc *UpdateChecker) IsUpdateAvailable() (bool, *Release, error) {
	release, err := uc.GetLatestRelease()
	if err != nil {
		return false, nil, err
	}

	// Compare versions (simple semantic version comparison)
	// Convert "v0.2.0" to "0.2.0" for comparison
	latestVer := strings.TrimPrefix(release.TagName, "v")
	currentVer := strings.TrimPrefix(uc.currentVersion, "v")

	// Normalize current version by removing git metadata (e.g., "0.3.0-2-g5ea24ba-dirty" -> "0.3.0")
	currentVer = normalizeVersion(currentVer)

	if isNewer(latestVer, currentVer) {
		return true, release, nil
	}

	return false, nil, nil
}

// DownloadAndInstall downloads and installs the latest release
func (uc *UpdateChecker) DownloadAndInstall(release *Release) error {
	// Find the appropriate binary for this platform
	asset := uc.findBinaryAsset(release)
	if asset == nil {
		return fmt.Errorf("no compatible binary found for this platform")
	}

	// Download the binary
	tempFile, err := uc.downloadAsset(asset)
	if err != nil {
		return err
	}
	defer func() {
		if err := os.Remove(tempFile); err != nil && !os.IsNotExist(err) {
			log.Printf("Warning: failed to remove temp file: %v", err)
		}
	}()

	// Extract and install binaries
	if err := uc.installFromArchive(tempFile, asset.Name); err != nil {
		return err
	}

	return nil
}

// findBinaryAsset finds the appropriate binary for the current platform.
// Release assets are named like: memofy-0.8.2-darwin-arm64.tar.gz
func (uc *UpdateChecker) findBinaryAsset(release *Release) *Asset {
	osName := "darwin"
	arch := "arm64" // Apple Silicon default
	if isIntelMac() {
		arch = "amd64"
	}

	// Match versioned names: memofy-VERSION-darwin-arm64.tar.gz
	for i, asset := range release.Assets {
		if strings.Contains(asset.Name, osName+"-"+arch) {
			return &release.Assets[i]
		}
	}

	// Fallback: any darwin asset
	for i, asset := range release.Assets {
		if strings.Contains(asset.Name, "darwin") {
			return &release.Assets[i]
		}
	}

	return nil
}

// downloadAsset downloads a release asset
func (uc *UpdateChecker) downloadAsset(asset *Asset) (string, error) {
	tempFile := filepath.Join(os.TempDir(), asset.Name)

	resp, err := http.Get(asset.BrowserDownloadURL)
	if err != nil {
		return "", fmt.Errorf("failed to download: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("Warning: failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	file, err := os.Create(tempFile)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("Warning: failed to close file: %v", err)
		}
	}()

	// Download with progress tracking (optional)
	if _, err := io.Copy(file, resp.Body); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return tempFile, nil
}

// installFromArchive extracts and installs binaries from archive
func (uc *UpdateChecker) installFromArchive(archivePath, archiveName string) error {
	switch {
	case strings.HasSuffix(archiveName, ".tar.gz") || strings.HasSuffix(archiveName, ".tgz"):
		return uc.installFromTarGz(archivePath)
	case strings.HasSuffix(archiveName, ".zip"):
		return uc.installFromZip(archivePath)
	default:
		return fmt.Errorf("unsupported archive format: %s", archiveName)
	}
}

// installFromTarGz extracts and installs from a .tar.gz file
func (uc *UpdateChecker) installFromTarGz(archivePath string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Printf("Warning: failed to close archive: %v", err)
		}
	}()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer func() {
		if err := gzr.Close(); err != nil {
			log.Printf("Warning: failed to close gzip reader: %v", err)
		}
	}()

	tempDir := filepath.Join(os.TempDir(), "memofy-update")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			log.Printf("Warning: failed to remove temp dir: %v", err)
		}
	}()

	tr := tar.NewReader(gzr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar: %w", err)
		}

		// Security: prevent path traversal
		target := filepath.Join(tempDir, filepath.Clean("/"+hdr.Name))

		if hdr.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0755); err != nil {
				return fmt.Errorf("failed to create dir: %w", err)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return fmt.Errorf("failed to create parent dir: %w", err)
		}

		out, err := os.Create(target)
		if err != nil {
			return fmt.Errorf("failed to create file: %w", err)
		}
		_, copyErr := io.Copy(out, tr)
		if closeErr := out.Close(); closeErr != nil {
			log.Printf("Warning: failed to close extracted file: %v", closeErr)
		}
		if copyErr != nil {
			return fmt.Errorf("failed to extract file: %w", copyErr)
		}
	}

	return uc.installBinaries(tempDir)
}

// installFromZip extracts and installs from a zip file
func (uc *UpdateChecker) installFromZip(zipPath string) error {
	file, err := os.Open(zipPath)
	if err != nil {
		return fmt.Errorf("failed to open zip: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("Warning: failed to close file: %v", err)
		}
	}()

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat zip: %w", err)
	}

	reader, err := zip.NewReader(file, info.Size())
	if err != nil {
		return fmt.Errorf("failed to create zip reader: %w", err)
	}

	tempDir := filepath.Join(os.TempDir(), "memofy-update")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			log.Printf("Warning: failed to remove temp dir: %v", err)
		}
	}()

	// Extract all files
	for _, f := range reader.File {
		fpath := filepath.Join(tempDir, f.Name)

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(fpath, os.ModePerm); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
			continue
		}

		// Create parent directory
		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return fmt.Errorf("failed to create parent directory: %w", err)
		}

		// Extract file
		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("failed to open file in zip: %w", err)
		}

		outFile, err := os.Create(fpath)
		if err != nil {
			if closeErr := rc.Close(); closeErr != nil {
				log.Printf("Warning: failed to close reader: %v", closeErr)
			}
			return fmt.Errorf("failed to create extracted file: %w", err)
		}

		_, err = io.Copy(outFile, rc)
		if closeErr := outFile.Close(); closeErr != nil {
			log.Printf("Warning: failed to close output file: %v", closeErr)
		}
		if closeErr := rc.Close(); closeErr != nil {
			log.Printf("Warning: failed to close reader: %v", closeErr)
		}

		if err != nil {
			return fmt.Errorf("failed to extract file: %w", err)
		}
	}

	// Install binaries
	if err := uc.installBinaries(tempDir); err != nil {
		return err
	}

	return nil
}

// installBinaries copies binaries to install directory
func (uc *UpdateChecker) installBinaries(sourceDir string) error {
	if err := os.MkdirAll(uc.installDir, 0755); err != nil {
		return fmt.Errorf("failed to create install dir %s: %w", uc.installDir, err)
	}

	binaries := []string{"memofy-core", "memofy-ui"}

	for _, binary := range binaries {
		// Walk the entire extracted tree; use the last match so nested dirs work.
		var sourcePath string
		_ = filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			// Exact name match or name == "memofy-core-darwin-arm64" style
			base := info.Name()
			if base == binary || strings.HasPrefix(base, binary) {
				sourcePath = path
			}
			return nil
		})

		if sourcePath == "" {
			// Non-fatal: archive may only contain one binary
			log.Printf("Warning: binary %s not found in archive, skipping", binary)
			continue
		}

		destPath := filepath.Join(uc.installDir, binary)
		if err := copyFile(sourcePath, destPath); err != nil {
			return fmt.Errorf("failed to install %s: %w", binary, err)
		}
		if err := os.Chmod(destPath, 0755); err != nil {
			log.Printf("Warning: failed to chmod %s: %v", destPath, err)
		}
		log.Printf("✓ Installed %s → %s", binary, destPath)
	}

	return nil
}

// copyFile copies a file
func copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		if err := source.Close(); err != nil {
			log.Printf("Warning: failed to close source file: %v", err)
		}
	}()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		if err := destination.Close(); err != nil {
			log.Printf("Warning: failed to close destination file: %v", err)
		}
	}()

	_, err = io.Copy(destination, source)
	return err
}

// isNewer checks if version1 > version2 (simple comparison)
func isNewer(version1, version2 string) bool {
	parts1 := strings.Split(version1, ".")
	parts2 := strings.Split(version2, ".")

	for i := 0; i < len(parts1) && i < len(parts2); i++ {
		var v1, v2 int
		if _, err := fmt.Sscanf(parts1[i], "%d", &v1); err != nil {
			v1 = 0
		}
		if _, err := fmt.Sscanf(parts2[i], "%d", &v2); err != nil {
			v2 = 0
		}

		if v1 > v2 {
			return true
		}
		if v1 < v2 {
			return false
		}
	}

	return len(parts1) > len(parts2)
}

// normalizeVersion removes git metadata from version strings
// Converts "0.3.0-2-g5ea24ba-dirty" to "0.3.0"
// Handles formats: X.Y.Z, X.Y.Z-rc1, X.Y.Z-N-gHASH-dirty
func normalizeVersion(version string) string {
	// Look for pattern: X.Y.Z or X.Y.Z-suffix
	// Strip git describe metadata (everything after digit-gHASH pattern)

	// First remove "-dirty" suffix if present
	version = strings.TrimSuffix(version, "-dirty")

	// Check if it matches git describe format (e.g., "0.3.0-2-g5ea24ba")
	// Format: TAG-COMMITS-gHASH
	parts := strings.Split(version, "-")
	if len(parts) >= 3 {
		// Check if any element matches "gHASH" pattern
		for i := 1; i < len(parts); i++ {
			if strings.HasPrefix(parts[i], "g") && len(parts[i]) > 1 {
				// This looks like git hash, return everything before the commit count
				return parts[0]
			}
		}
	}

	// If no git metadata, return as-is
	return version
}

// isIntelMac checks if running on Intel Mac
func isIntelMac() bool {
	// This is a simplified check - in production, you'd use more robust method
	// For now, assume Apple Silicon (arm64) as default
	return false
}
