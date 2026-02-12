# Memofy your meetings

![Memofy Logo](./docs/memofy.png)

Memofy is a macOS menu bar application that automatically detects and records Zoom, Microsoft Teams, and Google Meet meetings via OBS using intelligent detection and stable state control.

## Features

- **Automatic Detection**: Detects Zoom, Microsoft Teams, and Google Meet meetings in real-time with smart debounce (3/6)
- **Intelligent Recording**: Anti-flap logic prevents short interruptions from fragmenting recordings
- **Menu Bar Control**: Native macOS status display with quick-access controls
- **Manual Override**: Force start/stop recording regardless of meeting detection
- **OBS Integration**: Uses OBS WebSocket v5 for stable, reliable recording control
- **Native Notifications**: macOS notifications for recording start/stop and errors
- **Settings UI**: Adjust detection rules and thresholds from menu
- **Smart Filenames**: Automatic renaming to `YYYY-MM-DD_HHMM_Application_Title.mp4`
- **File-based IPC**: Daemon and UI communicate via status/command files
- **Auto-start**: LaunchAgent ensures daemon runs at login
- **Comprehensive Logging**: Detailed logs with 10MB rotation for troubleshooting

## System Requirements

- macOS 11.0 (Big Sur) or later
- OBS Studio 28.0+ with WebSocket server enabled
- Go 1.21+ (for building from source)

## Quick Start

### 1. Prerequisites

**OBS Setup**:
1. Install [OBS Studio](https://obsproject.com/)
2. Enable WebSocket server:
   - Open OBS
   - Go to `Tools > obs-websocket Settings`
   - Enable "Enable WebSocket server" checkbox
   - Set port to `4455` (default) with no password
3. Optional: Pre-configure recording output path (e.g., `~/Movies`)

**Automatic Setup**:
Memofy will **automatically** on first run:
- ✅ Start OBS if not running
- ✅ Create audio capture source if missing
- ✅ Create display capture source if missing

See [OBS_AUTO_INITIALIZATION.md](OBS_AUTO_INITIALIZATION.md) for details.

**macOS Permissions**:
- Grant **Screen Recording** permission to Terminal (for daemon)
- Grant **Accessibility** permission to Terminal (for process detection)

### 2. Installation

```bash
# Clone repository
git clone https://github.com/tiroq/memofy.git
cd memofy

# Build binaries
make build

# Install (creates LaunchAgent, starts daemon)
./scripts/install-launchagent.sh

# Start menu bar UI
~/.local/bin/memofy-ui
```

### 3. Usage

**Menu Bar UI**:
The menu bar application displays meeting status and provides quick controls:

- Status Icons:
  - ⚪ **IDLE** (white): Not recording, no meeting detected
  - 🟡 **WAIT** (yellow): Meeting detected, waiting for threshold
  - 🔴 **REC** (red): Actively recording
  - ⏸ **PAUSED** (pause symbol): Detection paused
  - ⚠️ **ERROR** (warning): Connection or permission error

- Menu Controls:
  - **Start Recording**: Manually start recording (switches to manual mode)
  - **Stop Recording**: Manually stop recording
  - **Auto Mode**: Automatic detection-based control (default)
  - **Manual Mode**: Continuous recording, requires manual stop
  - **Pause**: Suspend all detection and recording
  - **Open Recordings Folder**: Opens OBS output directory
  - **Open Logs**: View daemon logs for debugging
  - **Settings**: Configure detection rules and thresholds

**Settings UI**:
Click "Settings" in the menu to:
- Modify process names for Zoom and Teams detection
- Adjust window title hints for meeting identification
- Configure start/stop detection thresholds
- Validate and save configuration

**Notifications**:
The app sends native macOS notifications for:
- Recording started/stopped with duration
- Mode changes (Auto/Manual/Pause)
- Detection of meetings
- Errors with actionable guidance

**Command-Line Control** (alternative):
If menu bar is unavailable, control daemon via:
```bash
# Start recording manually
echo 'start' > ~/.cache/memofy/cmd.txt

# Stop recording
echo 'stop' > ~/.cache/memofy/cmd.txt

# Switch to auto mode
echo 'auto' > ~/.cache/memofy/cmd.txt

# Pause detection
echo 'pause' > ~/.cache/memofy/cmd.txt

# Check current status
cat ~/.cache/memofy/status.json | jq
```

**Files and Paths**:
- **Config file**: `~/.config/memofy/detection-rules.json` (editable via Settings menu)
- **Status file**: `~/.cache/memofy/status.json` (read-only, updated by daemon)
- **Command file**: `~/.cache/memofy/cmd.txt` (write to send commands)
- **Log files**: `/tmp/memofy-core.out.log` and `/tmp/memofy-core.err.log`
- **LaunchAgent**: `~/Library/LaunchAgents/com.memofy.core.plist`

## Architecture

### Components

**memofy-core** (Daemon):
- Runs as LaunchAgent (auto-starts at login)
- Polls every 2 seconds for meeting detection
- Controls OBS recording via WebSocket
- Writes status updates to `status.json`
- Reads commands from `cmd.txt`

**memofy-ui** (Menu Bar App):
- Displays status icon and menu
- Monitors `status.json` for updates (fsnotify)
- Writes user commands to `cmd.txt`
- Opens Finder windows and logs

### Detection Logic

**Zoom Meeting**:
- Process: `zoom.us` running
- AND (Host process: `CptHost` OR Window title hint)

**Teams Meeting**:
- Process: `Microsoft Teams` running
- Window title hints: configurable patterns

**Google Meet**:
- Browser running: Chrome, Safari, Firefox, Edge, Brave
- Window title hints: "Google Meet" or "meet.google.com"

**Debounce Thresholds**:
- Start: 3 consecutive detections (6-9 seconds)
- Stop: 6 consecutive non-detections (12-18 seconds)

### Recording Filename Format

```
YYYY-MM-DD_HHMM_Application_Title.mp4
```

Example: `2024-02-12_1430_Zoom_Meeting.mp4`

## Configuration

Edit `~/.config/memofy/detection-rules.json`:

```json
{
  "rules": [
    {
      "application": "zoom",
      "process_names": ["zoom.us"],
      "window_hints": ["Zoom Meeting"],
      "enabled": true
    },
    {
      "application": "teams",
      "process_names": ["Microsoft Teams"],
      "window_hints": ["Meeting", "Call"],
      "enabled": true
    },
    {
      "application": "google_meet",
      "process_names": ["Google Chrome", "Safari", "Firefox"],
      "window_hints": ["Google Meet", "meet.google.com"],
      "enabled": true
    }
  ],
  "poll_interval_seconds": 2,
  "start_threshold": 3,
  "stop_threshold": 6
}
```

**Tuning**:
- `start_threshold`: Lower = faster start (more false positives)
- `stop_threshold`: Higher = prevents fragmentation (slower stop)
- `window_hints`: Add app-specific keywords from window titles

## Troubleshooting

### Daemon Not Starting

```bash
# Check LaunchAgent status
launchctl list | grep memofy

# View error logs
tail -f /tmp/memofy-core.err.log

# Manually start daemon for debugging
~/.local/bin/memofy-core
```

### No Recording Starts

1. **Check OBS Connection**:
   ```bash
   # Verify WebSocket settings in OBS
   # Tools > obs-websocket Settings > Enable
   ```

2. **Check Detection**:
   ```bash
   # View detection logs
   tail -f /tmp/memofy-core.out.log | grep Detection
   ```

3. **Check Permissions**:
   - System Preferences > Security & Privacy > Screen Recording
   - System Preferences > Security & Privacy > Accessibility

### Recording Fragments (Multiple Files)

- Increase `stop_threshold` in detection rules
- Check for network interruptions during Zoom/Teams calls
- Verify CptHost process stays running during Zoom meetings

### Black Screen in Recordings

- Grant Screen Recording permission to OBS
- Ensure OBS window capture is configured correctly
- Check that auto-created display capture source is enabled

### OBS Auto-Initialization Issues

**OBS Won't Auto-Start**:
- Ensure OBS is installed: `ls -d /Applications/OBS.app`
- Manually start OBS: `open -a OBS`
- Check daemon logs for errors: `tail -f /tmp/memofy-core.err.log`

**WebSocket Server Error**:
- Verify WebSocket is enabled: `Tools > obs-websocket Settings`
- Restart OBS after enabling WebSocket
- Check port 4455 is not in use: `lsof -i :4455`

**Sources Not Auto-Created**:
- Check daemon logs: `grep -i "sources\|source" /tmp/memofy-core.out.log`
- Manually create sources in OBS:
  1. Click "+" in Sources panel
  2. Add "Audio Input Capture"
  3. Add "Display Capture"
- See [OBS_AUTO_INITIALIZATION.md](OBS_AUTO_INITIALIZATION.md) for details

## Development

### Building

```bash
# Build both binaries
make build

# Build individual components
make build-core
make build-ui

# Run tests
make test

# Clean build artifacts
make clean
```

### Project Structure

```
memofy/
├── cmd/
│   ├── memofy-core/    # Daemon main
│   └── memofy-ui/      # Menu bar UI main
├── internal/
│   ├── detector/       # Meeting detection logic
│   ├── statemachine/   # Debounce state machine
│   ├── obsws/          # OBS WebSocket client
│   ├── ipc/            # Status/command file handlers
│   └── config/         # Configuration loading
├── pkg/
│   └── macui/          # macOS menu bar UI components
├── scripts/
│   ├── com.memofy.core.plist   # LaunchAgent plist
│   ├── install-launchagent.sh  # Installation script
│   └── uninstall.sh            # Uninstallation script
├── configs/
│   └── default-detection-rules.json
├── Makefile
└── README.md
```

### Dependencies

- `github.com/gorilla/websocket` - OBS WebSocket v5 client
- `github.com/progrium/darwinkit` - macOS native APIs (NSStatusBar, NSWorkspace)
- `github.com/fsnotify/fsnotify` - File system notifications

## Uninstallation

```bash
./scripts/uninstall.sh
```

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Acknowledgments

- [OBS Studio](https://obsproject.com/) - Recording software
- [obs-websocket](https://github.com/obsproject/obs-websocket) - OBS remote control
- [progrium/darwinkit](https://github.com/progrium/darwinkit) - macOS Go bindings

## Future Enhancements

- Audio activity detection for more reliable meeting detection
- Automatic transcription integration
- Calendar-based recording triggers
- AI-powered meeting summaries
- Indexed meeting archive with search
