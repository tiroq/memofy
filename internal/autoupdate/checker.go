package autoupdate

import (
	"archive/zip"
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

// findBinaryAsset finds the appropriate binary for the current platform
func (uc *UpdateChecker) findBinaryAsset(release *Release) *Asset {
	// Determine platform and architecture
	osName := "darwin" // macOS
	arch := "arm64"    // Apple Silicon (most common)

	// Check if running on Intel Mac
	if isIntelMac() {
		arch = "amd64"
	}

	// Look for matching binary
	pattern := fmt.Sprintf("memofy-%s-%s.zip", osName, arch)

	for _, asset := range release.Assets {
		if strings.Contains(asset.Name, pattern) {
			return &asset
		}
	}

	// Fallback to generic darwin binary
	for _, asset := range release.Assets {
		if strings.Contains(asset.Name, "darwin") {
			return &asset
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
	// Determine if it's a zip or tar.gz
	if strings.HasSuffix(archiveName, ".zip") {
		return uc.installFromZip(archivePath)
	}

	// Could add tar.gz support here for Linux/other platforms
	return fmt.Errorf("unsupported archive format: %s", archiveName)
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
	binaries := []string{"memofy-core", "memofy-ui"}

	for _, binary := range binaries {
		// Find the binary in the extracted directory
		var sourcePath string
		err := filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && strings.Contains(info.Name(), binary) {
				sourcePath = path
				return filepath.SkipDir
			}
			return nil
		})

		if err != nil || sourcePath == "" {
			return fmt.Errorf("binary %s not found in archive", binary)
		}

		// Copy to install directory
		destPath := filepath.Join(uc.installDir, binary)
		if err := copyFile(sourcePath, destPath); err != nil {
			return fmt.Errorf("failed to install %s: %w", binary, err)
		}

		// Make executable
		if err := os.Chmod(destPath, 0755); err != nil {
			log.Printf("Warning: failed to chmod %s: %v", destPath, err)
		}
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

// isIntelMac checks if running on Intel Mac
func isIntelMac() bool {
	// This is a simplified check - in production, you'd use more robust method
	// For now, assume Apple Silicon (arm64) as default
	return false
}
