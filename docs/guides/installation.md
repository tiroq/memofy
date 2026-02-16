# Installation

## Quick Install

```bash
# Latest stable
curl -fsSL https://raw.githubusercontent.com/tiroq/memofy/main/scripts/quick-install.sh | bash

# Pre-release
curl -fsSL https://raw.githubusercontent.com/tiroq/memofy/main/scripts/quick-install.sh | bash -s -- --pre-release

# Specific version
curl -fsSL https://raw.githubusercontent.com/tiroq/memofy/main/scripts/quick-install.sh | bash -s -- --release 0.1.0
```

**Requirements**: macOS 11.0+, OBS 28.0+, 50MB disk

## OBS Setup

```bash
brew install --cask obs
```

**Enable WebSocket**: OBS → Tools → WebSocket Server Settings → Enable (port 4455, no password)

Sources auto-created on first run.

## Usage

1. Grant permissions (Screen Recording, Accessibility)
2. Click menu bar icon
3. Join meeting (Zoom/Teams/Meet) → auto-records

**Icons**: ⚫ Idle | 🔴 Recording | ⏸️ Paused

## Locations

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
