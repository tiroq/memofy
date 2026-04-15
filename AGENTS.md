# MEMOFY — PROJECT KNOWLEDGE BASE

**Module:** github.com/tiroq/memofy | **Go:** 1.21

## OVERVIEW

Lightweight cross-platform (macOS + Linux) automatic audio recorder. Captures system sound via PortAudio when audio activity is detected. Uses silence-based splitting to create separate WAV files per session. Single binary, CLI-first architecture.

## STRUCTURE

```
memofy/
├── cmd/
│   └── memofy/         # CLI entry point: run, status, doctor, test-audio
├── internal/
│   ├── audio/          # PortAudio capture + platform device detection
│   ├── config/         # YAML configuration loading
│   ├── engine/         # Main recording loop (capture → detect → record → write)
│   ├── statemachine/   # Recording lifecycle FSM
│   ├── metadata/       # JSON sidecar file writer
│   ├── monitor/        # Process detection (Zoom/Teams — metadata only)
│   ├── wav/            # WAV file writer (16-bit PCM)
│   ├── diaglog/        # Structured NDJSON diagnostic logger
│   └── pidfile/        # Single-instance enforcement
└── config.example.yaml # Example configuration
```

## WHERE TO LOOK

| Task | Location |
|------|----------|
| Recording start/stop logic | `internal/statemachine/statemachine.go` |
| Audio capture | `internal/audio/portaudio.go` |
| Device detection (macOS) | `internal/audio/device_darwin.go` |
| Device detection (Linux) | `internal/audio/device_linux.go` |
| RMS level calculation | `internal/audio/rms.go` |
| Main recording loop | `internal/engine/engine.go` |
| Configuration | `internal/config/config.go` |
| CLI commands | `cmd/memofy/main.go` |
| WAV writing | `internal/wav/writer.go` |
| Metadata sidecars | `internal/metadata/metadata.go` |
| Process monitoring | `internal/monitor/monitor.go` |

## ARCHITECTURE

**No HTTP server. No database. No GUI.** CLI-first tool.

```
memofy run  →  Engine  →  PortAudio  →  System Audio Device
                 │
                 ├── RMS Detection → State Machine → WAV Writer
                 │                                      ↓
                 └── Process Monitor (optional)    Metadata JSON
```

- **Audio**: PortAudio via CGo (macOS: CoreAudio + BlackHole, Linux: PulseAudio)
- **Storage**: WAV files + JSON sidecars
- **Concurrency**: goroutines + sync.Mutex; no channels for business logic
- **Config**: YAML file at `~/.config/memofy/config.yaml`

## KEY DOMAIN TYPES

| Type | Package | Purpose |
|------|---------|---------|
| `Engine` | `engine` | Main recording controller |
| `StateMachine` | `statemachine` | Recording lifecycle FSM |
| `State` | `statemachine` | idle/detecting_sound/recording/silence_wait/finalizing |
| `Action` | `statemachine` | none/start_recording/continue/stop_recording |
| `Stream` | `audio` | PortAudio capture stream |
| `DeviceInfo` | `audio` | Audio device descriptor |
| `Config` | `config` | YAML config types |
| `Recording` | `metadata` | JSON sidecar data |
| `Snapshot` | `monitor` | Process detection state |

## CONVENTIONS

- **No `ctx context.Context`** — not used; don't add it
- **No custom error types** — use `fmt.Errorf("...: %w", err)` wrapping
- **Constructor pattern**: `NewFoo(cfg) *Foo`
- **Version injection**: `-ldflags "-X main.Version=..."` at build time
- **CGo build tags**: `//go:build darwin` / `//go:build linux` for platform code

## ANTI-PATTERNS (THIS PROJECT)

- **No direct PortAudio calls outside `internal/audio/`**
- **No `log.Fatal` outside `main()`** — use error returns
- **No GUI or AppKit code** — CLI only
- **No HTTP servers or WebSocket clients**

## BUILD & DEV COMMANDS

```bash
task build          # Build binary → build/memofy
task test           # Unit tests
task lint           # golangci-lint
task run            # Build and run
task clean          # Remove build artifacts
```

## NOTES

- PortAudio requires `libportaudio` installed (brew install portaudio / apt install portaudio19-dev)
- macOS requires BlackHole virtual audio device for system audio capture
- Linux uses PulseAudio/PipeWire monitor source for system audio capture
- Process detection (Zoom/Teams) is best-effort metadata enrichment only
- State machine uses silence threshold (default 60s) for file splitting
