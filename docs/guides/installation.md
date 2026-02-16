# Installation Guide

## Quick Install

### One-Line Install (Recommended)

**Latest stable release**:
```bash
curl -fsSL https://raw.githubusercontent.com/tiroq/memofy/main/scripts/quick-install.sh | bash
```

**Latest pre-release (alpha/beta)**:
```bash
curl -fsSL https://raw.githubusercontent.com/tiroq/memofy/main/scripts/quick-install.sh | bash -s -- --pre-release
```

**Specific version**:
```bash
curl -fsSL https://raw.githubusercontent.com/tiroq/memofy/main/scripts/quick-install.sh | bash -s -- --release 0.1.0
```

The install script automatically:
- ✅ Downloads pre-built binaries (or builds from source)
- ✅ Installs to `~/.local/bin/`
- ✅ Sets up auto-start (LaunchAgent)
- ✅ Creates config directory

---

## System Requirements

- macOS 11.0+ (Big Sur or later)
- OBS Studio 28.0+ with WebSocket enabled
- 50MB disk space

---

## OBS Setup

### 1. Install OBS Studio

```bash
brew install --cask obs
```

Or download from [obsproject.com](https://obsproject.com/)

### 2. Enable WebSocket

1. Open OBS
2. Go to **Tools → WebSocket Server Settings**
3. Enable **WebSocket Server**
4. Set port: **4455** (default)
5. Password: leave blank

### 3. Auto-Configuration (Optional)

Memofy automatically creates missing sources on first run:
- **Audio capture** (system audio)
- **Display capture** (screen recording)

No manual source configuration needed!

---

## Usage

### First Run

1. **Grant macOS permissions** when prompted:
   - Screen Recording
   - Accessibility

2. **Start menu bar app**:
   - Click the Memofy icon in menu bar
   - It will auto-start the daemon

3. **Join a meeting**:
   - Open Zoom/Teams/Google Meet
   - Memofy detects and starts recording automatically

### Controls

**Menu bar icon**:
- ⚫ Idle (no meeting)
- 🔴 Recording
- ⏸️ Paused

**Menu options**:
- Force Start/Stop Recording
- Settings
- Check for Updates
- Quit

---

## File Locations

```
~/.local/bin/
  ├── memofy-core       # Daemon
  └── memofy-ui         # Menu bar app

~/.config/memofy/
  └── detection-rules.json  # Settings

~/.cache/memofy/
  ├── status.json       # Status file
  └── cmd.txt          # Command file
  
~/Library/LaunchAgents/
  └── com.memofy.core.plist  # Auto-start
```

---

## Uninstall

```bash
# Stop and remove daemon
launchctl unload ~/Library/LaunchAgents/com.memofy.core.plist
rm ~/Library/LaunchAgents/com.memofy.core.plist

# Remove binaries
rm ~/.local/bin/memofy-*

# Remove config (optional)
rm -rf ~/.config/memofy
rm -rf ~/.cache/memofy
```

---

## Troubleshooting

### Memofy doesn't detect meetings

1. Check **Screen Recording** permission granted
2. Verify OBS WebSocket enabled (port 4455)
3. Check logs: `/tmp/memofy-core.out.log`

### OBS won't connect

1. Verify OBS is running
2. Check WebSocket port: 4455
3. Ensure no password set
4. Restart both Memofy and OBS

### Auto-start not working

```bash
# Reload LaunchAgent
launchctl unload ~/Library/LaunchAgents/com.memofy.core.plist
launchctl load ~/Library/LaunchAgents/com.memofy.core.plist
```

---

**Next**: See [Development Guide](development.md) to build from source
