# Auto-Update System Documentation

This document explains how Memofy's automatic update checking and installation system works.

## Overview

The auto-update system:

1. **Checks for updates hourly** - Queries GitHub for new releases
2. **Notifies users** - Desktop notification when update is available
3. **Respects release channels** - Only offers updates based on user preference
4. **One-click install** - Users can install updates from the menu bar

## How It Works

### Update Check Cycle

```
Startup
  ↓
Load config (allow_dev_updates flag)
  ↓
Set release channel (Stable or Prerelease)
  ↓
Every 60 minutes
  ↓
Query GitHub API
  ↓
Compare versions
  ↓
If newer: Show notification
```

### Version Checking

The system compares versions using semantic versioning:

```
Current: v0.1.0
Latest:  v0.2.0
Result:  Update available ✓

Current: v0.2.0
Latest:  v0.2.0-rc1 (with allow_dev_updates: false)
Result:  No update (pre-release hidden)

Current: v0.2.0
Latest:  v0.2.0-rc1 (with allow_dev_updates: true)
Result:  Update available ✓
```

### Update Installation Flow

```
User clicks "Update Now"
  ↓
Show progress notification
  ↓
Download binary from GitHub Release
  ↓
Extract archive
  ↓
Back up current binary
  ↓
Install new binary to ~/.local/bin/
  ↓
Show success notification
  ↓
Suggest restart Memofy
```

## Configuration

### Via detection-rules.json

```json
{
  "allow_dev_updates": false
}
```

- **false** (default): Only stable releases (v0.2.0)
- **true**: Stable + pre-releases (v0.2.0-rc1, v0.2.0-beta1)

## Code Integration

### Files Involved

1. **internal/autoupdate/checker.go** - Core update logic
   - `UpdateChecker` struct - Main update checker
   - `GetLatestRelease()` - Fetch latest release info
   - `IsUpdateAvailable()` - Check if update needed
   - `DownloadAndInstall()` - Download and install binary

2. **pkg/macui/statusbar.go** - Menu bar UI
   - `CheckForUpdates()` - Hourly update check
   - `UpdateNow()` - Immediate update + install
   - Menu items: "Check for Updates", "Update Now"

3. **internal/config/detection_rules.go** - Configuration
   - `AllowDevUpdates` field in `DetectionConfig`
   - Controls which releases are considered for updates

## Update Checker API

### Creating an UpdateChecker

```go
// Create checker with GitHub repo info
checker := autoupdate.NewUpdateChecker(
    "tiroq",          // GitHub owner
    "memofy",         // GitHub repository
    "0.1.0",          // Current version
    installDir,       // Where to install binaries
)

// Set release channel
checker.SetChannel(autoupdate.ChannelStable)  // or ChannelPrerelease, ChannelDev
```

### Checking for Updates

```go
// Check if update is available
isAvailable, release, err := checker.IsUpdateAvailable()
if err != nil {
    log.Printf("Update check failed: %v", err)
    return
}

if isAvailable {
    log.Printf("Update available: %s -> %s", 
        checker.currentVersion, 
        release.TagName)
    
    // Optional: Show notification
    // Optional: Auto-install or ask user
}
```

### Downloading and Installing

```go
// Get latest release info
release, err := checker.GetLatestRelease()
if err != nil {
    log.Fatal(err)
}

// Download and install
if err := checker.DownloadAndInstall(release); err != nil {
    log.Printf("Update failed: %v", err)
    return
}

log.Println("Update successful!")
```

## Release Channels

### Channel Types

```go
const (
    // Only stable releases (e.g., v0.2.0)
    ChannelStable ReleaseChannel = "stable"
    
    // Stable + pre-releases (e.g., v0.2.0-rc1)
    ChannelPrerelease ReleaseChannel = "prerelease"
    
    // All releases (for development)
    ChannelDev ReleaseChannel = "dev"
)
```

### Channel Filtering

**Stable Channel**:
- Uses GitHub's `/releases/latest` endpoint (fast)
- Returns latest non-prerelease release
- Example: Returns v0.2.0, skips v0.2.0-rc1

**Prerelease Channel**:
- Fetches all releases via `/releases` endpoint
- Filters out drafts, includes pre-releases
- Example: Returns v0.2.0-rc1 if available, else v0.2.0

**Dev Channel**:
- Fetches all releases
- Returns latest regardless of type
- For development/testing only

## Platform Detection

Auto-update automatically detects your platform:

| OS | Architecture | Archive Type |
|---|---|---|
| macOS (Intel) | amd64 | .tar.gz |
| macOS (Apple Silicon) | arm64 | .tar.gz |
| Linux | amd64 | .tar.gz |
| Linux (ARM) | arm64 | .tar.gz |
| Windows | amd64 | .zip |

The system downloads the correct binary for your system.

## Installation Details

### Binary Locations

- **Before Update**: `~/.local/bin/memofy-core`, `~/.local/bin/memofy-ui`
- **Downloaded**: Extracted from GitHub Release assets
- **Backup**: Old binaries backed up (in case of issues)
- **Installed**: New binaries placed in `~/.local/bin/`

### Backup Strategy

Before installing new binaries:
1. Current binaries are moved to `.backup-{timestamp}`
2. New binaries are installed
3. If installation fails, original binaries can be restored

### Permissions

- Binary files: `755` (executable by owner, readable by all)
- Makes binaries runnable as normal commands

## User Notifications

### Notification Types

**Update Available**
```
Title: "Update Available"
Message: "New version: v0.2.0"
Actions: "Update Now", "Remind Later"
```

**Update in Progress**
```
Title: "Updating Memofy"
Message: "Downloading and installing v0.2.0..."
```

**Update Successful**
```
Title: "Update Successful"
Message: "v0.2.0 installed. Please restart Memofy to apply changes."
Actions: "Restart", "Later"
```

**Update Failed**
```
Title: "Update Failed"
Message: "Could not download update: [error details]"
```

## Menu Bar Integration

### Menu Items

**Check for Updates**
- Manual check for new releases
- Shows dialog with result
- Available in macOS menu bar

**Update Now**
- Only available when update is detected
- Downloads and installs in background
- Shows progress notification

### Status Indicators

Menu bar can display:
- ✓ Running normally
- ⚕️ Update available (dot/badge)
- ⟳ Downloading update

## Error Handling

### Network Issues

If GitHub API is unreachable:
- Log error (not fatal)
- Skip update check
- Retry next hour

### Invalid Release

If release info is corrupted:
- Log error
- Do not attempt download
- Notify user

### Download Failure

If binary download fails:
- Log error
- Do not attempt installation
- Retry next hour

### Installation Failure

If extraction or install fails:
- Restore old binaries from backup
- Log detailed error
- Notify user with troubleshooting steps

## Troubleshooting

### Updates not detected

1. **Check configuration**:
   ```bash
   cat ~/.config/memofy/detection-rules.json
   ```
   Verify `allow_dev_updates` matches your preference

2. **Restart Memofy**:
   Update checks run at startup and every hour

3. **Manual check**:
   Use "Check for Updates" menu item

4. **Check logs**:
   ```bash
   cat ~/.cache/memofy/memofy-ui.log
   ```

### Download fails

1. **Network connectivity**:
   Test: `curl -I https://api.github.com`

2. **GitHub rate limit**:
   GitHub API has 60 requests/hour for unauthenticated requests
   Wait an hour or authenticate with GitHub token

3. **File permissions**:
   Check `~/.local/bin/` is writable:
   ```bash
   ls -la ~/.local/bin/
   ```

### Installation fails after download

1. **Disk space**:
   Check available space: `df -h`

2. **File permissions**:
   Ensure `~/.local/bin/` is writable

3. **Binary corruption**:
   Check file size matches release page

### Update installed but old version runs

1. **Restart required**:
   Restart Memofy to load new binary

2. **Wrong binary location**:
   Check which binary is running:
   ```bash
   which memofy-core
   ls -la ~/.local/bin/memofy-core
   ```

3. **PATH issue**:
   Ensure `~/.local/bin` is in PATH:
   ```bash
   echo $PATH | grep local/bin
   ```

## Advanced: Manual Update

If auto-update fails, manually update:

### Via Quick Install Script

```bash
./scripts/quick-install.sh
```

### Via Downloaded Release

1. Visit: https://github.com/tiroq/memofy/releases
2. Download version matching your platform
3. Extract archive
4. Replace files in `~/.local/bin/`

## API Examples

### Check for specific release

```go
// Get latest release info without comparing versions
release, err := checker.GetLatestRelease()
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Latest release: %s\n", release.TagName)
fmt.Printf("Released: %s\n", release.Published)
fmt.Printf("Prerelease: %v\n", release.Prerelease)
```

### Download without installing

```go
// Just download, don't install
asset := checker.findBinaryAsset(release)
tempFile, err := checker.downloadAsset(asset)
// Use tempFile...
```

### Custom version comparison

```go
// Override version comparison logic if needed
// (Advanced use case)
```

## Future Enhancements

Possible improvements to the auto-update system:

1. **Scheduled updates** - Update at specific times
2. **Update frequency** - Make check interval configurable
3. **Release notes** - Show changelog before updating
4. **Automatic restart** - Restart Memofy after update
5. **Rollback** - Easy way to revert to previous version
6. **GPG verification** - Verify binary signatures
7. **Delta updates** - Download only changes (smaller downloads)
8. **HTTP/2 push** - Push update availability to clients

## Performance Considerations

### GitHub API Rate Limiting

- **Unauthenticated**: 60 requests/hour
- **Authenticated**: 5000 requests/hour
- Memofy checks once per hour (safe)

### Bandwidth Usage

- GitHub release assets are ~50-100MB per binary
- Memofy only downloads when user clicks "Update Now"
- Archives are compressed (.tar.gz, .zip)

### Disk Usage

- Update download is temporary (deleted after install)
- Old binary backup is kept (can be manually deleted)
- ~100-200MB disk space needed

## Related Documentation

- [Release Channel Configuration](RELEASE_CHANNEL_CONFIGURATION.md)
- [Release Process Guide](RELEASE_PROCESS_GUIDE.md)
- [Installation Guide](../INSTALLATION_GUIDE.md)
- [GitHub Workflows](.github/workflows/release.yml)
