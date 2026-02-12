# Installation & Auto-Update Guide

## Quick Install (One Command)

### Fastest Way to Get Started

```bash
git clone https://github.com/tiroq/memofy.git
cd memofy
bash scripts/quick-install.sh
```

**That's it!** The script handles everything:
- ✅ Checks for OBS installation
- ✅ Installs Go if needed
- ✅ Builds or downloads pre-compiled binaries
- ✅ Installs daemon and menu bar app
- ✅ Configures LaunchAgent auto-start
- ✅ Starts the menu bar UI
- ✅ Opens OBS for WebSocket configuration

### What Gets Installed

```
~/.local/bin/
  ├── memofy-core       # Daemon (auto-starts at login)
  └── memofy-ui         # Menu bar app

~/.config/memofy/
  └── detection-rules.json  # Configuration

~/.cache/memofy/
  ├── status.json       # Status file (updated by daemon)
  └── cmd.txt          # Command file (for control)

~/Library/LaunchAgents/
  └── com.memofy.core.plist  # Auto-start configuration
```

---

## Installation Options

### Option 1: One-Command Install (Recommended)
```bash
bash <(curl -fsSL https://raw.githubusercontent.com/tiroq/memofy/main/scripts/quick-install.sh)
```

### Option 2: Clone & Install
```bash
git clone https://github.com/tiroq/memofy.git
cd memofy
make quick-install
```

### Option 3: Build from Source
```bash
git clone https://github.com/tiroq/memofy.git
cd memofy
make build
make install
~/.local/bin/memofy-ui
```

### Option 4: From Pre-Compiled Release
```bash
# Download from GitHub releases
# https://github.com/tiroq/memofy/releases

# Extract and run
unzip memofy-*.zip
cd memofy-*
./memofy-ui
```

---

## Auto-Update Feature

### How It Works

1. **Menu bar checks for updates** - Once per hour automatically
2. **User is notified** - If a newer version is available
3. **One-click update** - Click "Update Now" to download and install
4. **Binary replacement** - Existing binaries are replaced with new ones
5. **Restart to complete** - App guides you to restart

### Accessing Updates

#### From Menu Bar
1. Click the Memofy menu bar icon
2. Look for "Check for Updates" or "Update Now"
3. Click to check for or install updates

#### Manual Check
```bash
# The UI checks automatically, but you can trigger manually:
~/.local/bin/memofy-ui --check-updates
```

### What Gets Updated

- ✅ `memofy-core` daemon
- ✅ `memofy-ui` menu bar app
- ✅ Configuration files (preserved)
- ✅ Detection rules (preserved)

### What Stays the Same

- ✅ Your settings and configuration
- ✅ Detection rules customizations
- ✅ Recordings and logs
- ✅ LaunchAgent auto-start setup

---

## Release Management

### For Developers: Creating Releases

#### Build Release Artifacts
```bash
# Build binaries for all platforms
make release VERSION=0.2.0

# Or with the build script directly
bash scripts/build-release.sh 0.2.0
```

#### What Gets Built
- `memofy-0.2.0-darwin-arm64.zip` - macOS Apple Silicon
- `memofy-0.2.0-darwin-amd64.zip` - macOS Intel
- `memofy-0.2.0-linux-amd64.tar.gz` - Linux x86_64
- `memofy-0.2.0-linux-arm64.tar.gz` - Linux ARM64
- `memofy-0.2.0-windows-amd64.zip` - Windows x86_64
- `memofy-0.2.0-windows-arm64.zip` - Windows ARM64

#### Create GitHub Release
```bash
# Tag the release
git tag v0.2.0
git push origin v0.2.0

# Go to GitHub releases page
# https://github.com/tiroq/memofy/releases

# Create new release
# - Upload artifacts from dist/ folder
# - Add release notes
# - Publish
```

---

## Troubleshooting Installation

### Issue: "Go not installed"
**Solution**: The script will auto-install via Homebrew
```bash
# Or manually install:
brew install go
```

### Issue: "OBS not installed"
**Solution**: The script will auto-install via Homebrew
```bash
# Or manually install:
brew install --cask obs
```

### Issue: "Permission denied" when running scripts
**Solution**: Make scripts executable
```bash
chmod +x scripts/quick-install.sh
chmod +x scripts/build-release.sh
```

### Issue: "Command not found: memofy-ui"
**Solution**: Make sure ~/.local/bin is in PATH
```bash
# Add to ~/.zshrc or ~/.bash_profile:
export PATH="$HOME/.local/bin:$PATH"

# Then:
source ~/.zshrc
memofy-ui
```

### Issue: WebSocket not working after install
**Solution**: Manual OBS configuration
```
1. Open OBS
2. Go to: Tools > obs-websocket Settings
3. Enable "Enable WebSocket server"
4. Set port to 4455
5. Click Apply and OK
```

---

## Uninstall

### Remove Everything
```bash
# Uninstall daemon
bash scripts/uninstall.sh

# Remove binaries
rm -rf ~/.local/bin/memofy-*

# Remove configuration
rm -rf ~/.config/memofy
rm -rf ~/.cache/memofy

# Remove LaunchAgent
rm ~/Library/LaunchAgents/com.memofy.core.plist
```

### Keep Config, Remove Binaries
```bash
# Remove will update binaries on next install
rm ~/.local/bin/memofy-core
rm ~/.local/bin/memofy-ui
```

---

## Update Process Details

### Automatic Background Update

1. **Initialize**
   ```
   UpdateChecker created with repo info
   ↓
   ```

2. **Check for Updates** (hourly)
   ```
   GitHub API query → GetLatestRelease()
   ↓
   Compare versions → IsUpdateAvailable()
   ↓
   Return: (available bool, release *Release, error)
   ```

3. **Download** (when user clicks "Update Now")
   ```
   Find platform-specific asset
   ↓
   Download .zip file → downloadAsset()
   ↓
   Extract archive → installFromZip()
   ```

4. **Install**
   ```
   Extract to temp directory
   ↓
   Copy memofy-core to ~/.local/bin/
   ↓
   Copy memofy-ui to ~/.local/bin/
   ↓
   Make executable (chmod +x)
   ↓
   Send notification: "Update Complete"
   ```

5. **Complete**
   ```
   Notification: "Restart app to use new version"
   ↓
   User kills app
   ↓
   Kill old process, start new binary
   ↓
   New version running
   ```

### API Flow

```
User clicks "Update Now"
        ↓
UpdateChecker.DownloadAndInstall()
        ├─ GetLatestRelease() → GitHub API
        ├─ findBinaryAsset() → Platform detection
        ├─ downloadAsset() → HTTP download
        ├─ installFromZip() → Extraction & install
        ├─ installBinaries() → Copy to ~/.local/bin/
        └─ Notification sent
```

---

## Advanced Configuration

### Custom Installation Directory

Edit `scripts/quick-install.sh` and change:
```bash
INSTALL_DIR="$HOME/.local/bin"
```

### Disable Auto-Update Checks

In menu bar app code, comment out auto-update initialization:
```go
// updateChecker: autoupdate.NewUpdateChecker(...)
```

### Pre-release Updates

Edit `internal/autoupdate/checker.go`:
```go
// Change GetLatestRelease() to include pre-releases
// if release.Prerelease {
//     return release  // Include pre-releases
// }
```

---

## Version Compatibility

| Component | Minimum | Recommended |
|-----------|---------|-------------|
| macOS | 11.0 (Big Sur) | 13.0+ |
| Go | 1.21 | 1.21+ |
| OBS | 28.0 | 30.0+ |
| Terminal | Any | Zsh/Bash |

---

## File Locations Summary

| File | Location | Purpose |
|------|----------|---------|
| Daemon | `~/.local/bin/memofy-core` | Automatic recorder |
| UI | `~/.local/bin/memofy-ui` | Menu bar app |
| Config | `~/.config/memofy/detection-rules.json` | Settings |
| Status | `~/.cache/memofy/status.json` | Current state |
| Commands | `~/.cache/memofy/cmd.txt` | Control file |
| LaunchAgent | `~/Library/LaunchAgents/com.memofy.core.plist` | Auto-start |
| Logs | `/tmp/memofy-core.*.log` | Debug output |

---

## Getting Help

**Installation issues?**
```bash
# Run with debug logging
DEBUG=1 bash scripts/quick-install.sh

# Check logs
tail -f /tmp/memofy-core.err.log
```

**Update problems?**
```bash
# Check update checker logs
tail -f /tmp/memofy-ui.log

# Manually download release
# https://github.com/tiroq/memofy/releases
```

**Command-line installation**
```bash
# If GUI doesn't work, use CLI:
echo 'start' > ~/.cache/memofy/cmd.txt
echo 'stop' > ~/.cache/memofy/cmd.txt
echo 'auto' > ~/.cache/memofy/cmd.txt
```

---

All installation and update options covered! Choose the one that works best for you.
