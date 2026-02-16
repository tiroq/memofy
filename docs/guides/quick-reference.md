# Quick Reference

## Installation

```bash
# Latest stable
curl -fsSL https://raw.githubusercontent.com/tiroq/memofy/main/scripts/quick-install.sh | bash

# Latest pre-release (alpha/beta)
curl -fsSL https://raw.githubusercontent.com/tiroq/memofy/main/scripts/quick-install.sh | bash -s -- --pre-release

# Specific version
curl -fsSL https://raw.githubusercontent.com/tiroq/memofy/main/scripts/quick-install.sh | bash -s -- --release 0.1.0
```

## File Locations

```
~/.local/bin/memofy-{core,ui}          # Binaries
~/.config/memofy/detection-rules.json  # Config
~/.cache/memofy/{status.json,cmd.txt}  # IPC files
~/Library/LaunchAgents/com.memofy.core.plist  # Auto-start
/tmp/memofy-core.{out,err}.log         # Logs
```

## Build Commands

```bash
# Task (recommended)
task build          # Build both binaries
task test           # Run tests
task lint           # Run linter
task install        # Install to ~/.local/bin

# Make
make build
make test
make install
```

## Release Commands

```bash
# Auto-bump
task release-major    # 0.1.0 → 1.0.0
task release-minor    # 0.1.0 → 0.2.0
task release-patch    # 0.1.0 → 0.1.1

# Pre-releases
task release-alpha-auto    # Auto alpha from latest stable
task release-beta-auto     # Auto beta from latest stable

# Utilities
task release-list          # List all tags
task release-verify VERSION=v0.1.0  # Verify CI passed
```

## Control Daemon

```bash
# Via file commands
echo 'start' > ~/.cache/memofy/cmd.txt
echo 'stop' > ~/.cache/memofy/cmd.txt
echo 'auto' > ~/.cache/memofy/cmd.txt

# Check status
cat ~/.cache/memofy/status.json | jq

# View logs
tail -f /tmp/memofy-core.out.log
```

## OBS Setup

1. Install: `brew install --cask obs`
2. Enable WebSocket: Tools → WebSocket Server Settings
3. Port: 4455 (no password)
4. Sources auto-created on first run

## Troubleshooting

```bash
# Check processes
pgrep -fl memofy
pgrep -fl OBS

# Test OBS connection
nc -zv localhost 4455

# Restart daemon
launchctl unload ~/Library/LaunchAgents/com.memofy.core.plist
launchctl load ~/Library/LaunchAgents/com.memofy.core.plist

# Check permissions
tccutil reset ScreenCapture
tccutil reset Accessibility
```

## Detection Thresholds

- **Start recording**: 3 consecutive detections (6-9 seconds)
- **Stop recording**: 6 consecutive non-detections (12-18 seconds)
- **Poll interval**: 2 seconds

## Supported Platforms

- Zoom (process: `zoom.us` + window hints)
- Microsoft Teams (process: `Microsoft Teams` + window hints)
- Google Meet (browser window titles)

---

For detailed guides, see [docs/](../docs/)
