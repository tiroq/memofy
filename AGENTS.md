# MEMOFY — PROJECT KNOWLEDGE BASE

**Branch:** 002-obs-autostop | **Module:** github.com/tiroq/memofy | **Go:** 1.21

## OVERVIEW

macOS menu bar app that auto-detects and records Zoom/Teams/Google Meet meetings via OBS Studio WebSocket. Two-binary architecture: `memofy-core` (daemon) + `memofy-ui` (status bar GUI).

## STRUCTURE

```
memofy/
├── cmd/
│   ├── memofy-core/    # Background daemon: detection loop + OBS control
│   └── memofy-ui/      # macOS status bar app (AppKit via darwinkit)
├── internal/
│   ├── statemachine/   # Core: debounced recording lifecycle FSM
│   ├── detector/       # Meeting detection (Zoom/Teams/GoogleMeet)
│   ├── obsws/          # OBS WebSocket v5 client + source management
│   ├── ipc/            # File-based IPC between ui↔daemon (~/.cache/memofy/)
│   ├── config/         # Detection rules JSON loader
│   ├── diaglog/        # Structured NDJSON diagnostic logger
│   ├── autoupdate/     # GitHub release checker + one-click updater
│   ├── fileutil/       # OBS recording filename sanitizer/renamer
│   ├── pidfile/        # Single-instance enforcement
│   └── validation/     # OBS version compatibility checks
├── pkg/
│   └── macui/          # Exported macOS UI: status bar, settings window
├── testutil/           # Shared test helpers: MockOBSServer, assertions
├── tests/integration/  # End-to-end integration tests
├── configs/            # default-detection-rules.json
└── specs/              # Feature specs by ticket number (FR-xxx, T0xx)
```

## WHERE TO LOOK

| Task | Location |
|------|----------|
| Recording start/stop logic | `internal/statemachine/statemachine.go` |
| OBS WebSocket calls | `internal/obsws/client.go` |
| Meeting detection signals | `internal/detector/` |
| UI↔daemon communication | `internal/ipc/` (file-based: cmd.txt + status.json) |
| Menu bar + settings UI | `pkg/macui/` |
| Detection thresholds | `configs/default-detection-rules.json` |
| Feature requirements | `specs/` (numbered FR-xxx tickets) |

## ARCHITECTURE

**No HTTP server. No database.** Pure macOS desktop app.

```
memofy-ui  ──IPC──▶  memofy-core  ──WebSocket──▶  OBS Studio
  (AppKit)          (detection loop)               (port 4455)
```

- **IPC**: `~/.cache/memofy/cmd.txt` (commands) + `status.json` (state)
- **Storage**: JSON files only — no SQLite, no Postgres
- **Concurrency**: goroutines + sync.RWMutex; NO channels for business logic
- **Layer flow**: `macui` → `ipc` → `statemachine` → `detector`/`obsws`

## KEY DOMAIN TYPES

| Type | Package | Purpose |
|------|---------|---------|
| `StateMachine` | `statemachine` | Debounce + recording FSM |
| `RecordingSession` | `statemachine` | Active session metadata (ID, origin, app) |
| `RecordingOrigin` | `statemachine` | `manual`/`auto`/`forced` — priority hierarchy |
| `DetectionState` | `detector` | Multi-signal meeting detection snapshot |
| `StatusSnapshot` | `ipc` | Full system state written to status.json |
| `Client` | `obsws` | OBS WebSocket v5 client |
| `OperatingMode` | `ipc` | `auto`/`manual`/`paused` |

## CONVENTIONS

- **No `ctx context.Context`** — not used anywhere; don't add it
- **No custom error types** — use `fmt.Errorf("...: %w", err)` wrapping
- **Constructor pattern**: `NewFoo(cfg) *Foo` with setter injection (`SetLogger`, `SetDebounceDuration`)
- **Ticket references** in comments: `// T014:`, `// FR-003` — keep this style
- **Version injection**: `-ldflags "-X main.Version=..."` at build time; `Version = "dev"` as default
- **`runtime.LockOSThread()`** required in `memofy-ui/main.go` — macOS GUI must run on OS thread

## ANTI-PATTERNS (THIS PROJECT)

- **No gorilla/websocket direct use outside `obsws/`** — all OBS comms go through `obsws.Client`
- **No direct file writes for IPC** — always use `ipc.WriteCommand()` / `ipc.WriteStatus()` (atomic)
- **No status.json reads outside `ipc`** — use `ipc.ReadStatus()`
- **No darwinkit/AppKit calls outside `pkg/macui/` and `cmd/memofy-ui/`** — GUI threading rules
- **No `log.Fatal` outside `main()`** — use error returns
- **Binaries in `bin/` are committed** — do not delete; they are the release artifacts

## BUILD & DEV COMMANDS

```bash
task build          # Build both binaries → build/
task build-core     # Daemon only
task build-ui       # UI only
task test           # Unit tests
task test-integration  # Integration tests (requires OBS running)
task lint           # golangci-lint
task dev-daemon     # Run core in foreground with live reload
task dev-ui         # Run UI binary
task logs           # Tail memofy logs
task status         # Show current status.json
```

> **arm64 Go**: Taskfile prefers `/usr/local/go-arm64/bin/go` over system `go` to avoid Rosetta+darwinkit crashes on macOS 26+.

## NOTES

- `specs/` contains authoritative feature specs — read before implementing anything in `statemachine` or `obsws`
- `diaglog` uses NDJSON format; `--export-diag` subcommand bundles logs for bug reports
- Detection uses **debounce streaks**: default 3 consecutive detections to start, 6 to stop (configurable in `configs/default-detection-rules.json`)
- `RecordingOrigin` has priority: `manual > auto = forced` — manual stops cannot be overridden by auto
