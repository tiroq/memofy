# Memofy

Memofy is a lightweight automatic audio recorder that captures system sound when activity is detected. It runs in the background, starts recording when audio appears, and splits recordings when silence exceeds a configurable threshold.

## Features

- **Automatic recording** — starts when system audio is detected, stops on silence
- **Silence-based splitting** — creates separate files per audio session
- **Cross-platform** — macOS (via BlackHole) and Linux (via PulseAudio/PipeWire)
- **WAV output** — produces standard 16-bit PCM WAV files
- **Metadata sidecars** — JSON files with timestamps, duration, and process info
- **Process detection** — optional Zoom/Teams detection enriches metadata
- **Simple CLI** — `run`, `status`, `doctor`, `test-audio`
- **Low resource usage** — single process, no server, no database

## How It Works

Memofy captures audio from a virtual loopback device (BlackHole on macOS, PulseAudio monitor on Linux). It continuously measures the RMS level of incoming audio. When audio exceeds the threshold, recording starts. When silence lasts longer than the configured duration (default: 60 seconds), the recording file is finalized and a new session begins on next sound.

```
Audio In → RMS Detection → State Machine → WAV Writer → File + Metadata
                                ↑
                          Silence Timer
```

## Prerequisites

### macOS

1. Install [BlackHole](https://existential.audio/blackhole/) (virtual audio driver):
   ```
   brew install blackhole-2ch
   ```
2. Set up a Multi-Output Device in Audio MIDI Setup:
   - Open **Audio MIDI Setup** (Applications → Utilities)
   - Click **+** → **Create Multi-Output Device**
   - Check both your speakers/headphones AND **BlackHole 2ch**
   - Set this as your system output device

3. Install PortAudio:
   ```
   brew install portaudio
   ```

### Linux

1. Ensure PulseAudio or PipeWire is running (default on most distributions)
2. Install PortAudio development files:
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

# Or with version tag
go build -ldflags "-X main.Version=1.0.0" -o memofy ./cmd/memofy/
```

## Usage

### Start recording

```bash
memofy run
```

Memofy runs in the foreground. Use Ctrl+C to stop. For background operation, use a process manager or systemd.

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

### Show version

```bash
memofy version
```

## Configuration

Create `~/.config/memofy/config.yaml`:

```yaml
audio:
  device: auto            # "auto" or device name substring
  threshold: 0.02         # RMS level for sound detection (0.0 - 1.0)
  silence_seconds: 60     # seconds of silence before splitting
  sample_rate: 44100
  channels: 2

output:
  dir: ~/Recordings/Memofy

platform:
  macos_device: "BlackHole"   # device name hint for macOS
  linux_device: "default"     # device name hint for Linux
```

All settings have sensible defaults. The config file is optional.

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
  "duration": 1800,
  "mic_active": false,
  "zoom_running": true,
  "teams_running": false,
  "platform": "darwin"
}
```

## Architecture

```
cmd/memofy/         CLI entry point (run, status, doctor, test-audio)
internal/
  audio/            PortAudio capture + device detection (macOS/Linux)
  config/           YAML configuration
  engine/           Main recording loop
  statemachine/     Recording lifecycle FSM
  metadata/         JSON sidecar writer
  monitor/          Process detection (Zoom/Teams)
  wav/              WAV file writer
  diaglog/          Structured NDJSON debug logging
  pidfile/          Single-instance enforcement
```

### State machine

```
idle → detecting_sound → recording → silence_wait → finalizing → idle
                              ↑            |
                              └── sound ───┘
```

## Limitations

- **System audio only** — requires a virtual audio device (BlackHole on macOS)
- **No video** — audio recording only
- **Process detection is best-effort** — Zoom/Teams detection enriches metadata but does not control recording
- **Microphone detection** — currently best-effort, may not be accurate on all systems
- **PortAudio dependency** — requires libportaudio installed on the system

## Debug logging

Set `MEMOFY_DEBUG_RECORDING=true` to enable NDJSON debug logs to `/tmp/memofy-debug.log`.

## License

See [LICENSE](LICENSE).
