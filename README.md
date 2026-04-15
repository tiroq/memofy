# Memofy

Memofy is a lightweight automatic audio recorder for macOS and Linux. It captures system audio in the background, starts recording when sound is detected, and splits recordings automatically when silence exceeds a configurable threshold.

## Features

- **Automatic recording** — starts when system audio exceeds threshold
- **Silence-based splitting** — creates separate files per audio session
- **Format profiles** — High Quality (M4A/AAC 32kHz 64kbps), Balanced, Lightweight, and WAV
- **Cross-platform** — macOS (native CoreAudio + BlackHole) and Linux (PortAudio + PulseAudio/PipeWire)
- **Menu bar UI** — macOS native status bar icon with format switching, settings, and status
- **Settings window** — native macOS settings with audio, recording, monitoring, and general tabs
- **Update checker** — checks GitHub releases for new versions
- **Metadata sidecars** — JSON files with full recording metadata
- **Process detection** — optional Zoom/Teams detection enriches metadata
- **Simple CLI** — `run`, `status`, `doctor`, `test-audio`, `check-updates`

## How It Works

Memofy captures audio from a virtual loopback device (BlackHole on macOS, PulseAudio monitor on Linux). It continuously measures the RMS level of incoming audio:

1. When audio exceeds the threshold, the system enters an **arming** state
2. If sound persists for the activation window (default 400ms), recording **starts**
3. Recording continues through brief silence
4. When silence lasts longer than the configured duration (default 60s), the file is **finalized**
5. The WAV recording is converted to M4A/AAC (unless WAV format is selected)
6. A JSON metadata sidecar is created alongside the recording

```
Audio In → RMS Detection → State Machine → WAV Writer → [M4A Conversion] → File + Metadata
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

No other dependencies are required on macOS. Audio capture uses native CoreAudio and M4A conversion uses the built-in `afconvert` tool.

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
4. Install ffmpeg for M4A/AAC conversion:
   ```
   # Debian/Ubuntu
   sudo apt install ffmpeg

   # Fedora
   sudo dnf install ffmpeg

   # Arch
   sudo pacman -S ffmpeg
   ```

## Installation

```bash
# Build from source
task build

# Or manually
go build -ldflags "-X main.Version=$(git describe --tags --always)" -o build/memofy ./cmd/memofy/
```

## Usage

### Start recording

```bash
memofy run
```

On macOS, a menu bar icon appears showing current status. Recording starts automatically when system audio is detected. Use the menu to change format, access settings, or quit.

### Check system setup

```bash
memofy doctor
```

Verifies audio devices, BlackHole/PulseAudio setup, conversion tools, and output directory.

### Show status

```bash
memofy status
```

Shows platform, format profile, output directory, and current recording state.

### Test audio capture

```bash
memofy test-audio
```

Captures audio for 5 seconds and displays real-time RMS levels.

### Check for updates

```bash
memofy check-updates
```

## Format Profiles

Change format from the menu bar or settings window. Default is **High Quality**.

| Profile | Container | Codec | Sample Rate | Bitrate | Use Case |
|---------|-----------|-------|-------------|---------|----------|
| **High Quality** (default) | M4A | AAC | 32 kHz | 64 kbps | Best audio quality |
| **Balanced** | M4A | AAC | 24 kHz | 48 kbps | Good quality, smaller files |
| **Lightweight** | M4A | AAC | 16 kHz | 32 kbps | Minimal storage |
| **WAV** | WAV | PCM 16-bit | 44.1 kHz | — | Raw/debug |

All profiles record in mono.

## Configuration

Create `~/.config/memofy/config.yaml` or use the Settings window on macOS:

```yaml
audio:
  device: auto              # "auto" or device name substring (e.g. "BlackHole 2ch")
  threshold: 0.02           # RMS level for sound detection (0.0 - 1.0)
  activation_ms: 400        # milliseconds of continuous sound before recording starts
  silence_seconds: 60       # seconds of silence before splitting into a new file
  format_profile: high      # high, balanced, lightweight, wav

output:
  dir: ~/Recordings/Memofy  # where recordings are saved

monitoring:
  detect_zoom: true         # detect Zoom process (metadata only)
  detect_teams: true        # detect Teams process (metadata only)
  detect_mic_usage: true    # detect microphone activity (best-effort)

ui:
  auto_check_updates: true  # check for updates on startup

logging:
  level: info               # debug, info, warn, error
```

All settings have sensible defaults. The config file is optional.

## Output

### File naming

```
YYYY-MM-DD_HHMMSS_audio_<quality>.<ext>
```

Examples:
- `2026-02-12_143015_audio_high.m4a`
- `2026-02-12_153422_audio_balanced.m4a`
- `2026-02-12_160000_audio_wav.wav`

### Metadata sidecar

Each recording gets a companion `.json` file:

```json
{
  "session_id": "20260212T143015",
  "started_at": "2026-02-12T14:30:15Z",
  "ended_at": "2026-02-12T15:00:15Z",
  "duration_seconds": 1800,
  "platform": "darwin",
  "device_name": "BlackHole 2ch",
  "format_profile": "high",
  "container": "m4a",
  "codec": "aac",
  "sample_rate": 32000,
  "channels": 1,
  "bitrate_kbps": 64,
  "threshold": 0.02,
  "silence_split_seconds": 60,
  "split_reason": "silence_threshold",
  "version": "0.2.0"
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
- **Status** — current state and device
- **Format** — current format profile
- **Change Format** — switch between High Quality, Balanced, Lightweight, WAV
- **Open Recordings Folder** — opens output directory in Finder
- **Settings...** — edit configuration
- **Check for Updates...** — compare with latest GitHub release
- **About Memofy** — version and platform info
- **Quit** — stop recording and exit

## Settings Window (macOS)

Open **Settings...** from the menu bar to edit configuration:

### Audio Tab
- Input device
- Threshold
- Activation window (ms)
- Silence split (seconds)

### Recording Tab
- Format profile (high/balanced/lightweight/wav)
- Output directory

### Monitoring Tab
- Zoom detection
- Teams detection
- Microphone activity detection

### General Tab
- Auto check for updates
- Log level

## Limitations

- **System audio only** — requires a virtual audio device (BlackHole on macOS)
- **Audio only** — no video recording
- **macOS and Linux only** — Windows is not supported
- **M4A conversion requires tools** — `afconvert` (macOS, built-in) or `ffmpeg` (Linux)
- **Process detection is best-effort** — Zoom/Teams detection enriches metadata only
- **Settings require restart** — audio settings take effect after restarting the app
- **Linux has no tray UI** — CLI only on Linux

## Debug Logging

Set `MEMOFY_DEBUG_RECORDING=true` to enable NDJSON debug logs.

## License

See [LICENSE](LICENSE).
