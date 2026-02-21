# Implementation Plan: OBS Auto-Stop Investigation & Safeguard

**Branch**: `002-obs-autostop` | **Date**: 2026-02-20 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/002-obs-autostop/spec.md`

## Summary

Implement a structured NDJSON diagnostic logging framework and a state-machine authority model that permanently prevents manual recordings from being stopped by any automated code path. The investigation phase (logging) and the safeguard phase (state-machine fix) are parallel work streams.

Root cause hypothesis: the OBS WebSocket reconnect handler or the auto-detection polling loop issues a `StopRecord` command without checking whether the current session was started manually. The state machine currently tracks mode (auto/manual/paused) but stores no session origin, so there is no authority check at the stop call site.

## Technical Context

**Language/Version**: Go 1.21
**Primary Dependencies**: `gorilla/websocket` v1.5.3 (existing), `log/slog` (stdlib, Go 1.21), `crypto/rand` (stdlib)
**Storage**: Rolling NDJSON file (`/tmp/memofy-debug.log`, 10 MB cap)
**Testing**: stdlib `testing` + existing `testutil` package; mock OBS WS server (existing pattern in `internal/obsws`)
**Target Platform**: macOS (darwin/arm64 primary; darwin/amd64 secondary)
**Project Type**: Single project — Go monorepo (`cmd/`, `internal/`, `pkg/`)
**Performance Goals**: Log write latency < 1 ms per entry; no measurable impact on recording session throughput
**Constraints**: Zero new external dependencies; Go 1.21 minimum; `--export-diag` completes in < 10 seconds
**Scale/Scope**: Single-user daemon; log volume ~100–500 entries per recording session under debug mode

## Constitution Check

No constitution file found in `.specify/memory/` — gate skipped. No violations to report.

## Project Structure

### Documentation (this feature)

```text
specs/002-obs-autostop/
├── plan.md              ← this file
├── research.md          ← Phase 0 output
├── data-model.md        ← Phase 1 output
├── quickstart.md        ← Phase 1 output
├── contracts/
│   ├── log-entry.md     ← NDJSON schema contract
│   └── cli.md           ← CLI interface contract
└── tasks.md             ← Phase 2 output (/speckit.tasks — NOT created here)
```

### Source Code

```text
internal/
├── diaglog/             # NEW package
│   ├── diaglog.go       # Logger type, Init(), Log(), Close()
│   ├── rolling.go       # rollingWriter (10 MB cap)
│   ├── redact.go        # recursive JSON field redaction
│   ├── export.go        # DiagBundle assembly for --export-diag
│   └── diaglog_test.go  # unit + integration tests
├── statemachine/
│   ├── statemachine.go  # MODIFY: add RecordingOrigin, RecordingSession, StopRequest; authority check in StopRecording
│   └── statemachine_test.go  # MODIFY: add TestManual* authority tests
├── obsws/
│   ├── client.go        # MODIFY: inject *diaglog.Logger; log all WS send/recv; log reconnect; log close code 4009
│   ├── operations.go    # MODIFY: StopRecord accepts reason string; logs with component+reason
│   └── client_test.go   # MODIFY: assert log entries emitted for key events
└── ipc/
    └── status.go        # MODIFY: add RecordingOrigin, SessionID fields to StatusSnapshot

cmd/
└── memofy-core/
    └── main.go          # MODIFY: detect --export-diag flag; call diaglog.Export(); wire diaglog into client + statemachine
```

**Structure Decision**: Single project. All changes are additions to existing packages plus one new `internal/diaglog` package. No new binaries, no new modules, no new top-level directories.

## Complexity Tracking

No constitution violations. No complexity justification required.

---

## Implementation Phases

> Phases are ordered by user story priority (P1 → P4). Each phase is independently deployable and testable.

---

### Phase 1 — Structured NDJSON Logger (US1 / FR-001, FR-002, FR-004, FR-005, FR-013)

**Goal**: Developer can set `MEMOFY_DEBUG_RECORDING=true` and observe a populated NDJSON log.

**New file: `internal/diaglog/diaglog.go`**

```go
// Key exported API:
func New(path string) (*Logger, error)
func (l *Logger) Log(entry LogEntry)
func (l *Logger) Close() error
func IsDebugEnabled() bool  // reads MEMOFY_DEBUG_RECORDING env var
```

- `Logger` wraps a `*rollingWriter` and a `sync.Mutex`.
- `Log()` serialises `LogEntry` to JSON (via `encoding/json`), appends newline, writes to `rollingWriter`, and calls `Sync()`.
- When `IsDebugEnabled()` returns false, `Log()` is a no-op.

**New file: `internal/diaglog/rolling.go`**

```go
type rollingWriter struct {
    path    string
    maxSize int64   // 10 MB default
    f       *os.File
    size    int64
    mu      sync.Mutex
}
func (r *rollingWriter) Write(p []byte) (int, error)
// When size would exceed maxSize: truncate file to 0, reset size counter, then write.
```

**New file: `internal/diaglog/redact.go`**

```go
var sensitiveKeys = []string{"authentication", "password", "secret", "challenge", "salt", "auth"}
func Redact(v interface{}) interface{}
// Traverses map[string]interface{} trees recursively.
// Replaces values of matching keys with "[REDACTED]".
```

**Tests**: `internal/diaglog/diaglog_test.go`
- `TestLogWritesNDJSON`: write 3 entries, read file back, unmarshal each line, assert fields.
- `TestRollingTruncatesAt10MB`: write > 10 MB, assert file size ≤ 10 MB after each write.
- `TestRedactSensitiveFields`: assert `challenge`, `salt`, `auth` replaced in nested payloads.
- `TestNoOpWhenDisabled`: `MEMOFY_DEBUG_RECORDING` unset → file not created.

**Acceptance gate**: US1, scenario 3 (no file created when debug disabled).

---

### Phase 2 — Inject Logging into OBS WebSocket Client (US2 / FR-001, FR-002, FR-003, FR-011)

**Goal**: Every WS send/receive, reconnect attempt, and StopRecord dispatch appears in the log with full attribution.

**Modify `internal/obsws/client.go`**:

- Add `logger *diaglog.Logger` field to `Client`.
- Add `WithLogger(l *diaglog.Logger) *Client` option or pass via `NewClient`.
- In `sendRequest()`: call `l.Log(LogEntry{Component: "obs-ws-client", Event: "ws_send", Payload: redacted(msg)})` before `conn.WriteJSON`.
- In `readMessages()`: log every raw message received as `ws_recv`.
- In `reconnect()` loop: log `ws_reconnect_attempt` (attempt number, delay) and `ws_reconnect_success` / `ws_reconnect_failed`.
- Detect close code 4009 in `readMessages` error handling: log `multi_client_warning`.
- Log `ws_connect` after `authenticate` succeeds; log `ws_disconnect` in `disconnect()` with close reason.

**Modify `internal/obsws/operations.go`**:

- `StopRecord` gains a `reason string` parameter: `func (c *Client) StopRecord(reason string) (string, error)`.
- The reason is written to the `ws_send` log entry for the `StopRecord` request.
- All existing callers of `StopRecord()` must pass an explicit reason (compile-time enforcement).

**Tests**: `internal/obsws/client_test.go`
- `TestLogStopRecordEmitsReason`: mock OBS server, call `StopRecord("user_stop")`, assert log contains entry with `event=ws_send`, `reason=user_stop`.
- `TestLogReconnectAttempt`: simulate disconnect from mock, assert `ws_reconnect_attempt` entry appears.
- `TestLogMultiClientWarning`: mock sends close code 4009, assert `multi_client_warning` entry.

**Acceptance gate**: US2 — developer can trace a stop to a specific component+reason from log alone.

---

### Phase 3 — State Machine Authority Model (US3 / FR-007, FR-008, FR-009)

**Goal**: Manually-started recordings are never stopped by any automated path.

**Modify `internal/statemachine/statemachine.go`**:

Add to `StateMachine` struct:
```go
session         *RecordingSession  // nil when not recording
debounceDur     time.Duration       // from MEMOFY_MANUAL_DEBOUNCE_MS, default 5s
logger          *diaglog.Logger
```

Change `StopRecording()` signature to:
```go
func (sm *StateMachine) StopRecording(req StopRequest) bool
// Returns true if stop was executed, false if rejected.
// Rejection: session.Origin == OriginManual AND req.RequestOrigin != OriginManual → log + return false.
// Debounce (race guard): session.Origin == OriginManual
//   AND time.Since(session.StartedAt) < debounceDur
//   AND req.RequestOrigin != OriginManual → reject.
// Rationale: debounce guard targets only automated signals that were
// queued before the session lock took effect. A user clicking Stop
// within the debounce window always succeeds (spec FR-008).
```

Change `ForceStart()`:
- Sets `session = &RecordingSession{SessionID: newSessionID(), Origin: OriginManual, ...}`
- Logs `recording_start` with `origin=manual`, `session_id`.

Change `StartRecording()`:
- Sets `session = &RecordingSession{..., Origin: OriginAuto}`
- Logs `recording_start` with `origin=auto`.

All callers of `StopRecording` (in `cmd/memofy-core/main.go` and anywhere in `ProcessDetection`) must be updated to pass a `StopRequest`.

**Modify `cmd/memofy-core/main.go`**:
- Detection-loop stop: `sm.StopRecording(StopRequest{RequestOrigin: OriginAuto, Reason: "auto_detection_stop", Component: "auto-detector"})`
- Command-driven stop (user): `sm.StopRecording(StopRequest{RequestOrigin: OriginManual, Reason: "user_stop", Component: "memofy-core"})`

**FR-009 (no reconnect resync)**: Confirm that `reconnect()` in `client.go` does **not** call `GetRecordStatus` or issue any recording command. Add a comment at the reconnect site documenting this is intentional per FR-009.

**Tests**: `internal/statemachine/statemachine_test.go`
- `TestManualSessionBlocksAutoStop`: ForceStart → StopRecording(auto origin) → assert returns false, recording still active.
- `TestManualSessionAllowsUserStop`: ForceStart → StopRecording(manual origin, user_stop) → assert returns true.
- `TestDebounceRejectsAutoStopEarly`: ForceStart → immediately StopRecording(**auto** origin within debounce window) → assert rejected.
- `TestDebounceAllowsUserStopEarly`: ForceStart → immediately StopRecording(manual origin, user_stop within debounce) → assert returns true (debounce does NOT block user stops).
- `TestAutoSessionAllowsAutoStop`: StartRecording(auto) → StopRecording(auto origin) → assert returns true.
- `TestSessionIDGeneratedOnStart`: ForceStart → assert SessionID is 16 non-empty hex chars.

**Acceptance gate**: US3 — all four acceptance scenarios pass.

---

### Phase 4 — Diagnostic Export CLI (US4 / FR-006)

**Goal**: `memofy-core --export-diag` writes a complete NDJSON bundle in < 10 seconds.

**New file: `internal/diaglog/export.go`**

```go
func Export(logPath string, dest string) (path string, lines int, err error)
// Reads logPath, prepends a DiagBundle metadata line, writes to dest/<timestamp>.ndjson.
// Returns the written file path and the number of log lines included.
```

`DiagBundle` is assembled from:
- `runtime.Version()` for Go version
- `runtime.GOOS` + `runtime.GOARCH` for OS/arch
- Memofy version from build-time `ldflags` variable (existing `version` var in `main.go`)
- Line count from scanning logPath

**Modify `cmd/memofy-core/main.go`**:

```go
if len(os.Args) > 1 && os.Args[1] == "--export-diag" {
    path := os.Getenv("MEMOFY_LOG_PATH")
    if path == "" { path = "/tmp/memofy-debug.log" }
    out, n, err := diaglog.Export(path, ".")
    if err != nil { fmt.Fprintln(os.Stderr, "error:", err); os.Exit(exitCodeFor(err)) }
    fmt.Printf("Wrote: %s (%d lines)\n", out, n)
    os.Exit(0)
}
```

**Tests**: `internal/diaglog/export_test.go`
- `TestExportWritesBundleHeader`: write 10 log lines to temp file, call Export, assert line 1 is valid `DiagBundle` JSON with correct entry_count.
- `TestExportContainsAllLines`: assert lines 2–N match source lines verbatim.
- `TestExportMissingFile`: non-existent log path → returns error wrapping `os.ErrNotExist`.
- `TestExportCompletesUnder10s`: large log file (synthetic 10 MB) → Export completes within 10 seconds.

**Acceptance gate**: US4 — both acceptance scenarios pass.

---

## Wiring Summary (main.go changes)

```go
// Startup sequence:
logger, _ := diaglog.New(logPath)
defer logger.Close()

obsClient := obsws.NewClient(url, password)
obsClient.SetLogger(logger)                      // Phase 2

sm := statemachine.NewStateMachine(cfg)
sm.SetLogger(logger)                             // Phase 3
sm.SetDebounceDuration(debounceDur)              // Phase 3

// Write origin to StatusSnapshot on each status update:
status.RecordingOrigin = string(sm.SessionOrigin())
status.SessionID = sm.SessionID()
```

---

## Cross-Cutting: `internal/ipc/status.go`

Add two fields to `StatusSnapshot` (both phases land this):
```go
RecordingOrigin string `json:"recording_origin,omitempty"`
SessionID       string `json:"session_id,omitempty"`
```

Populated in the status-write call inside the main polling loop after each `ProcessDetection` tick.

---

## Dependency Graph

```
Phase 1 (diaglog)
    └── Phase 2 (obsws injection)   — depends on diaglog
    └── Phase 3 (statemachine)      — depends on diaglog
            └── Phase 4 (export CLI) — depends on diaglog.Export only
```

Phases 2 and 3 can proceed in parallel once Phase 1 is complete.
Phase 4 can start in parallel with Phases 2 and 3.

---

## Test Strategy

| Layer | Pattern | Location |
|-------|---------|----------|
| diaglog unit | Table-driven, temp files | `internal/diaglog/*_test.go` |
| obsws integration | Mock OBS WS server (existing) | `internal/obsws/client_test.go` |
| statemachine unit | Table-driven, no I/O | `internal/statemachine/statemachine_test.go` |
| export CLI | Temp log file + subprocess or direct call | `internal/diaglog/export_test.go` |
| regression | `go test ./...` must stay green | CI |

All tests run with `go test ./... -race` to catch the known races identified in research (requestID counter, callback set-after-connect).
