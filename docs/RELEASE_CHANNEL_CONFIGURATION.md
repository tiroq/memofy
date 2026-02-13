# Release Channel Configuration

This document explains how Memofy's release channel system works and how to configure it.

## Overview

Memofy supports three release channels to accommodate different user preferences:

- **Stable** (default): Only stable releases (e.g., v0.1.0, v0.2.0)
- **Prerelease**: Stable releases + beta/rc versions (e.g., v0.2.0-rc1, v0.2.0-beta1)
- **Dev**: All releases including development versions

## Configuration

The release channel preference is controlled via the `allow_dev_updates` flag in `~/.config/memofy/detection-rules.json`:

```json
{
  "rules": [...],
  "poll_interval_seconds": 2,
  "start_threshold": 3,
  "stop_threshold": 6,
  "allow_dev_updates": false
}
```

### Configuration Options

| Setting | Value | Behavior |
|---------|-------|----------|
| `allow_dev_updates` | `false` (default) | Only notify about stable releases (v0.2.0) |
| `allow_dev_updates` | `true` | Notify about stable AND pre-releases (v0.2.0-rc1, v0.2.0-beta1) |

## How It Works

### At Startup

When Memofy starts, it:

1. Loads `detection-rules.json` from `~/.config/memofy/`
2. Reads the `allow_dev_updates` flag
3. Configures the UpdateChecker with the appropriate release channel:
   - `allow_dev_updates: false` → `ChannelStable`
   - `allow_dev_updates: true` → `ChannelPrerelease`

### When Checking for Updates

The UpdateChecker automatically:

1. Queries GitHub API for latest releases
2. Filters releases based on the configured channel:
   - **ChannelStable**: Only non-prerelease releases
   - **ChannelPrerelease**: All releases except drafts
   - **ChannelDev**: All releases including dev builds

3. Notifies the user if an update is available

## GitHub Workflow Automation

### Release Process

1. **Create a version tag** (local):
   ```bash
   git tag v0.2.0          # Stable release
   git tag v0.2.0-rc1      # Pre-release
   git push origin v0.2.0
   ```

2. **GitHub Actions automatically**:
   - Detects the tag push
   - Builds for all platforms (macOS Intel/Apple Silicon, Linux amd64/arm64, Windows)
   - Creates a GitHub Release
   - Marks as prerelease if tag contains `rc`, `beta`, or `alpha`

3. **Memofy users are notified**:
   - Users with `allow_dev_updates: false` see only stable releases
   - Users with `allow_dev_updates: true` see both stable and pre-releases

## Version Detection

The workflow automatically detects if a release is a pre-release by checking the git tag:

- `v0.2.0` → **Stable** (marked as "Latest Release")
- `v0.2.0-rc1` → **Pre-release** (marked as "Pre-release")
- `v0.2.0-beta1` → **Pre-release** (marked as "Pre-release")
- `v0.2.0-alpha1` → **Pre-release** (marked as "Pre-release")

Any tag containing `rc`, `beta`, or `alpha` is automatically marked as a pre-release.

## Changing Release Channel

To enable pre-release notifications:

1. Edit `~/.config/memofy/detection-rules.json`
2. Change `allow_dev_updates` from `false` to `true`
3. Save the file
4. Memofy will pick up the change on startup

Example:
```json
{
  "allow_dev_updates": true
}
```

## Code Implementation

### Files Modified

1. **internal/autoupdate/checker.go**
   - Added `ReleaseChannel` type with three values: `ChannelStable`, `ChannelPrerelease`, `ChannelDev`
   - Updated `UpdateChecker` struct to include `channel` field
   - Added `SetChannel()` method to change the release channel
   - Enhanced `GetLatestRelease()` to filter based on channel:
     - Uses GitHub `/releases/latest` endpoint for stable (fast)
     - Uses `/releases` endpoint with filtering for pre-release/dev (comprehensive)
   - Added helper methods:
     - `getLatestStableRelease()`: Uses the fast `/latest` endpoint
     - `getLatestReleaseInChannel()`: Fetches all releases and filters
     - `matchesChannel()`: Determines if a release matches the selected channel

2. **pkg/macui/statusbar.go**
   - Updated imports to include `config` package
   - Modified `NewStatusBarApp()` to:
     - Load the detection configuration
     - Read the `allow_dev_updates` flag
     - Set the appropriate release channel
     - Log which channel is active

3. **internal/config/detection_rules.go**
   - Added `AllowDevUpdates` field to `DetectionConfig` struct

4. **configs/default-detection-rules.json**
   - Added `"allow_dev_updates": false` field with default value

5. **.github/workflows/release.yml**
   - Automated release workflow (already created)
   - Builds all platform combinations
   - Auto-detects pre-release based on tag content
   - Creates GitHub Release with all artifacts

6. **.github/workflows/ci.yml**
   - Continuous integration pipeline (already created)
   - Runs tests on every PR and push

## Examples

### Scenario 1: User wants stable releases only (default)

1. `detect-rules.json` has `"allow_dev_updates": false`
2. UpdateChecker is set to `ChannelStable`
3. User is notified about v0.2.0 (stable)
4. User is NOT notified about v0.2.0-rc1 (pre-release)

### Scenario 2: User wants to test pre-releases

1. User edits `detect-rules.json` to `"allow_dev_updates": true`
2. UpdateChecker is set to `ChannelPrerelease`
3. User is notified about v0.2.0-rc1 (pre-release)
4. User can install and test before stable release

### Scenario 3: Release workflow publishes new version

1. Maintainer: `git tag v0.2.0 && git push origin v0.2.0`
2. GitHub Actions:
   - Detects tag push
   - Builds for all platforms
   - Creates GitHub Release marked as "Latest"
3. All users are notified (regardless of their channel setting)

## Technical Details

### Release Channel Enum Values

```go
const (
    ChannelStable     ReleaseChannel = "stable"     // Only stable releases
    ChannelPrerelease ReleaseChannel = "prerelease" // Stable + pre-releases
    ChannelDev        ReleaseChannel = "dev"        // All releases
)
```

### GitHub API Queries

**For Stable Channel** (fast):
```
GET /repos/tiroq/memofy/releases/latest
```
Returns the latest stable release (GitHub's definition).

**For Prerelease/Dev Channels** (comprehensive):
```
GET /repos/tiroq/memofy/releases?per_page=30
```
Fetches up to 30 releases and filters based on channel.

## Future Enhancements

Possible future improvements:

1. **Menu item to change channel** - Allow users to switch channels from Settings menu
2. **Release notes display** - Show what's new in each release before updating
3. **Automatic restart** - Restart Memofy after successful update
4. **GPG signature verification** - Verify release authenticity
5. **Download progress** - Show update download progress
6. **Schedule updates** - Allow automatic updates at specified times

## Troubleshooting

### Not receiving update notifications

1. Check `allow_dev_updates` setting matches expected behavior
2. Verify `~/.config/memofy/detection-rules.json` exists
3. Restart Memofy to reload configuration
4. Check logs: `cat ~/.cache/memofy/*.log`

### Getting too many notifications

1. Set `allow_dev_updates: false` to only get stable releases
2. Pre-releases are usually temporary; wait for stable release

### Update checker not working

1. Verify GitHub API access (no firewall blocking api.github.com)
2. Check network connectivity
3. Ensure memofy-ui is running (status bar app)
