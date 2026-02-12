# Memofy your meetings

![Memofy Logo](./docs/memofy.png)

Memofy is a macOS menu bar application that automatically detects and records Zoom/Teams meetings via OBS using intelligent detection and stable state control.

## Features

- **Automatic Detection**: Detects Zoom and Microsoft Teams meetings in real-time
- **Intelligent Recording**: Anti-flap debounce logic prevents recording fragmentation
- **Menu Bar Control**: Simple status display and manual controls
- **OBS Integration**: Uses OBS WebSocket v5 for reliable recording
- **File-based IPC**: Daemon and UI communicate via status/command files

## System Requirements

- macOS 11.0 (Big Sur) or later
- OBS Studio 28.0+ with WebSocket server enabled
- Go 1.21+ (for building from source)

## Quick Start

### 1. Prerequisites

**OBS Setup**:
1. Install [OBS Studio](https://obsproject.com/)
2. Enable WebSocket server: `Tools > obs-websocket Settings`
3. Set port to `4455` (default) with no password
4. Configure recording output path (e.g., `~/Movies`)

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

> **Note**: The menu bar UI is currently a stub implementation. Full macOS menu bar integration with darwinkit is deferred pending macOS GUI expertise. The daemon (memofy-core) is fully functional and can be controlled via command-line by writing to `~/.cache/memofy/cmd.txt`.
>
> **CLI Control**:
> ```bash
> # Start recording manually
> echo 'start' > ~/.cache/memofy/cmd.txt
> 
> # Stop recording
> echo 'stop' > ~/.cache/memofy/cmd.txt
> 
> # Toggle recording
> echo 'toggle' > ~/.cache/memofy/cmd.txt
> 
> # Switch to auto mode
> echo 'auto' > ~/.cache/memofy/cmd.txt
> 
> # Pause detection
> echo 'pause' > ~/.cache/memofy/cmd.txt
> 
> # Check status
> cat ~/.cache/memofy/status.json | jq
> ```

**Menu Bar States** (when full UI is implemented):
- ◯ **IDLE** (gray): Not recording, no meeting detected
- ◐ **WAIT** (half-filled): Meeting detected, waiting for threshold (3 detections)
- ● **REC** (filled): Actively recording
- ⚠ **ERROR** (warning): Connection or permission error

**Controls**:
- **Start Recording**: Manually start recording immediately
- **Stop Recording**: Manually stop current recording
- **Auto Mode**: Automatic detection-based control (default)
- **Pause**: Suspend all detection and recording
- **Open Recordings Folder**: Opens Finder to OBS output directory
- **Open Logs**: Opens `/tmp` to view daemon logs

**Configuration**:
- Config file: `~/.config/memofy/detection-rules.json`
- Status file: `~/.cache/memofy/status.json`
- Command file: `~/.cache/memofy/cmd.txt`
- Log files: `/tmp/memofy-core.{out,err}.log`

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
