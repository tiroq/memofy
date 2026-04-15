# Memofy

Memofy is a lightweight automatic audio recorder for macOS and Linux that captures system sound when activity is detected.

It runs in the background, starts recording when sound appears, and splits recordings automatically when silence exceeds a configurable threshold. On macOS, it includes a simple menu bar UI for status and configuration.

## Features

- **Automatic recording** — starts when system audio exceeds threshold
- **Silence-based splitting** — creates separate WAV files per audio session
- **Activation window** — configurable time of continuous sound before recording starts (avoids false triggers)
- **Cross-platform** — macOS (via BlackHole) and Linux (via PulseAudio/PipeWire)
- **Menu bar UI** — macOS native status bar icon with settings and status display
- **WAV output** — standard 16-bit PCM WAV files
- **Metadata sidecars** — JSON files with timestamps, duration, device, and process info
- **Process detection** — optional Zoom/Teams detection enriches metadata
- **Update checker** — checks GitHub releases for new versions
- **Simple CLI** — `run`, `status`, `doctor`, `test-audio`, `check-updates`
- **Low resource usage** — single process, no server, no database

## How It Works

Memofy captures audio from a virtual loopback device (BlackHole on macOS, PulseAudio monitor on Linux). It continuously measures the RMS level of incoming audio:

1. When audio exceeds the threshold, the system enters an **arming** state
2. If sound persists for the activation window (default 400ms), recording **starts**
3. Recording continues through brief silence
4. When silence lasts longer than the configured duration (default 60s), the file is **finalized**
5. A new session begins on next sound

```
Audio In → RMS Detection → State Machine → WAV Writer → File + Metadata
                                ↑
                          Silence Timer
```

### State Machine

```
idle → arming → recording → silence_wait → finalizing → idle
                     ↑            |
                     └── sound ───┘
```

## Prerequisites

### macOS

1. Install [BlackHole](https://existential.audio/blackhole/) (virtual audio driver):
   ```
   brew install blackhole-2ch
   ```
2. Set up a Multi-Output Device in Audio MIDI Setup:
   - Open **Audio MIDI Setup** (Applications → Utilities)
   - Click **+** → Create Multi-Output Device
   - Check both your speakers/headphones AND **BlackHole 2ch**
   - Set this as your system output device

3. Install PortAudio:
   ```
   brew install portaudio
   ```

### Linux

1. Ensure PulseAudio or PipeWire is running (default on most distributions)
2. The system's default monitor source is used automatically
3. Install PortAudio development files:
   ```
   # Debian/Ubuntu
   sudo apt install libportaudio2 portaudio19-dev

   # Fedora
   sudo dnf install portaudio portaudio-devel

   # Arch
   sudo pacman -S portaudio
   ```

## Installation

```bash
# Build from source
go build -o memofy ./cmd/memofy/

# Build with version tag
go build -ldflags "-X main.Version=1.0.0" -o memofy ./cmd/memofy/

# Using task runner
task build
```

## Usage

### Start recording

```bash
memofy run
```

On macOS, a menu bar icon appears showing current status. Use the menu to access settings, check for updates, or quit. Recording starts automatically when system audio is detected.

Use Ctrl+C or the Quit menu item to stop.

### Check system setup

```bash
memofy doctor
```

Lists audio devices, verifies BlackHole/PulseAudio setup, and checks output directory.

### Test audio capture

```bash
memofy test-audio
```

Captures audio for 5 seconds and displays real-time RMS levels, showing whether sound is detected.

### Check for updates

```bash
memofy check-updates
```

Compares the current version against the latest GitHub release.

### Show version

```bash
memofy version
```

## Configuration

Create `~/.config/memofy/config.yaml` or use the Settings window on macOS:

```yaml
audio:
  device: auto              # "auto" or device name substring (e.g. "BlackHole 2ch")
  threshold: 0.02           # RMS level for sound detection (0.0 - 1.0)
  activation_ms: 400        # milliseconds of continuous sound before recording starts
  silence_seconds: 60       # seconds of silence before splitting into a new file

output:
  dir: ~/Recordings/Memofy  # where recordings are saved

monitoring:
  detect_zoom: true         # detect Zoom process (metadata only)
  detect_teams: true        # detect Teams process (metadata only)
  detect_mic_usage: true    # detect microphone activity (best-effort)
  keep_single_session_while_mic_active: true

ui:
  auto_check_updates: true  # check for updates on startup

logging:
  level: info               # debug, info, warn, error
```

All settings have sensible defaults. The config file is optional.

### Settings Window (macOS)

Open **Settings...** from the menu bar to edit configuration through a native GUI. Changes are saved to the config file and take effect on restart.

## Output

### File naming

```
YYYY-MM-DD_HHMMSS_audio_[mic|nomic].wav
```

Examples:
- `2026-02-12_143015_audio_nomic.wav`
- `2026-02-12_153422_audio_mic.wav`

### Metadata sidecar

Each WAV file gets a companion `.json` file:

```json
{
  "started_at": "2026-02-12T14:30:15Z",
  "ended_at": "2026-02-12T15:00:15Z",
  "duration_seconds": 1800,
  "platform": "darwin",
  "device_name": "BlackHole 2ch",
  "threshold": 0.02,
  "silence_split_seconds": 60,
  "mic_active": false,
  "zoom_running": true,
  "teams_running": false,
  "session_id": "20260212T143015",
  "app_version": "0.1.0"
}
```

## Menu Bar (macOS)

The menu bar icon shows current state:

| Color | State |
|-------|-------|
| Gray | Idle — no audio detected |
| Yellow | Listening — sound detected, arming |
| Red | Recording — actively writing audio |
| Orange | Recording (silence) — in silence wait |

Menu items:
- **Status** — current state, device, and file info
- **Open Recordings Folder** — opens output directory in Finder
- **Settings...** — edit configuration
- **Check for Updates...** — compare with latest GitHub release
- **About Memofy** — version and info
- **Quit** — stop recording and exit

## Architecture

```
cmd/memofy/         CLI entry point (run, status, doctor, test-audio, check-updates)
internal/
  audio/            PortAudio capture + device detection (macOS/Linux)
  config/           YAML configuration loading + saving
  engine/           Main recording loop + status reporting
  statemachine/     Recording lifecycle FSM (idle/arming/recording/silence_wait/finalizing)
  metadata/         JSON sidecar writer
  monitor/          Process detection (Zoom/Teams, best-effort)
  wav/              WAV file writer (16-bit PCM)
  autoupdate/       GitHub release version checker
  diaglog/          Structured NDJSON debug logging
  pidfile/          Single-instance enforcement
pkg/
  macui/            macOS menu bar UI (darwinkit/AppKit)
```

## Limitations

- **System audio only** — requires a virtual audio device (BlackHole on macOS)
- **Audio only** — no video recording
- **macOS and Linux only** — Windows is not supported
- **Process detection is best-effort** — Zoom/Teams detection enriches metadata but does not control recording
- **Microphone detection** — best-effort, may not be accurate on all systems
- **PortAudio dependency** — requires libportaudio installed on the system
- **Settings require restart** — audio setting changes take effect after restarting the app

## Debug Logging

Set `MEMOFY_DEBUG_RECORDING=true` to enable NDJSON debug logs.

## License

See [LICENSE](LICENSE).
