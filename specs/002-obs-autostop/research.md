# Research: OBS Auto-Stop Investigation & Safeguard

**Feature**: `002-obs-autostop`
**Date**: 2026-02-20

---

## 1. Existing Codebase Findings

### 1.1 State Machine (`internal/statemachine`)

**Decision**: Add a `RecordingOrigin` enum and `sessionID` field to `StateMachine`. Enforce authority check inside `StopRecording`.

**Rationale**: The state machine already distinguishes mode (`auto`/`manual`/`paused`) and has `ForceStart` which switches mode to `ModePaused` post-start — but this is the *only* protection and it is not declarative. There is no stored `origin`, no `sessionID`, and `StopRecording()` performs no authority check. Two code paths may independently call `StopRecording()` with no indication of who called it.

**Key gap confirmed**: `StartRecording()` and `ForceStart()` both produce identical internal state. Nothing downstream knows whether a session was started manually or automatically.

**Alternatives considered**:
- Relying purely on `ModePaused` to block auto-stop: rejected — the mode can be changed by other code paths and does not protect the `StopRecording()` call itself.
- Wrapping via a boolean `manualLock` flag: rejected — origin and session semantics are richer than a bool; an enum is cleaner and more extensible.

---

### 1.2 OBS WebSocket Client (`internal/obsws`)

**Decision**: Inject the new `diaglog` logger into `Client`; add a `reason` parameter to `sendRequest` wrappers for `StopRecord`.

**Rationale**: All OBS commands flow through a single `sendRequest()` method, making it the ideal single injection point for outbound logging. Inbound messages are all handled in `readMessages()` and `handleEvent()` — two additional injection points for inbound logging. There is no existing structured logging; only `log.Printf` and `fmt.Printf` scattered through `sources.go`.

**Reconnection finding**: The reconnect loop in `readMessages` does **not** re-query recording state after a successful `Connect()`. This is the correct default under FR-009 (no automatic state sync on reconnect). No change needed here — but this must be documented explicitly so future maintainers don't add resync.

**Alternatives considered**:
- Wrapping `Client` in a decorator/proxy type: rejected — adds indirection for no gain; direct injection is idiomatic Go.
- Using `slog.SetDefault` global logger: rejected — the diaglog must be independently enabled/disabled via env var without affecting process-level logging.

---

### 1.3 IPC / StatusSnapshot (`internal/ipc`)

**Decision**: Add `RecordingOrigin` and `SessionID` fields to `StatusSnapshot`.

**Rationale**: `StatusSnapshot` is the shared state snapshot read by the UI and other consumers. Making origin and session ID visible at this level ensures the UI can display meaningful recording status and allows the diagnostic export to capture the full session context.

---

### 1.4 Logging Infrastructure

**Decision**: New package `internal/diaglog` implementing a structured NDJSON logger backed by a rolling file, gated by `MEMOFY_DEBUG_RECORDING=true`.

**Rationale**: Go 1.21 (the project's declared minimum) ships `log/slog` with a built-in JSON handler. This eliminates any external logging dependency. The rolling file behaviour (10 MB cap) requires a thin custom writer wrapping `os.File` with size tracking — a ~50-line implementation.

**Alternatives considered**:
- `github.com/rs/zerolog` or `github.com/uber-go/zap`: rejected — both add an external dependency for functionality achievable with stdlib. The project currently has only 3 dependencies; adding a fourth for logging is unjustified.
- Writing to stderr: rejected — debug output must not intermix with process stderr (used for error reporting) and must be retained across process restarts.
- Single global logger: accepted with the caveat that the logger must be initialised once at startup and injected (not accessed via a package-level var called from deep in hot paths).

---

### 1.5 Multi-Client Detection (FR-010)

**Decision**: Detect via PID file + process enumeration at startup (existing `internal/pidfile` pattern). When a second Memofy instance connects to the same OBS WS endpoint, the first will receive a `disconnect` event with reason `websocket: close 4009 session invalidated`. Trap this specific close code and surface a warning.

**Rationale**: OBS WebSocket v5 does support multiple simultaneous clients. The protocol does not provide a "list connected clients" API. The most reliable detection method is the OBS-v5 behaviour of revoking earlier sessions when a new client connects — close code `4009`. Memofy already has a PID file mechanism in `internal/pidfile`.

**Alternatives considered**:
- Querying OBS for a client list: not available in OBS WS v5 protocol.
- Generating a session token sent in `Identify.eventSubscriptions`: not visible to other clients; not viable.

---

### 1.6 Diagnostic Export (`--export-diag`)

**Decision**: Add a `--export-diag` flag to `cmd/memofy-core/main.go` using stdlib `os.Args` parsing (no `flag` package subcommand needed). The command reads the rolling log from `/tmp/memofy-debug.log` (or `MEMOFY_LOG_PATH` override), bundles it with version and OS info into a single NDJSON file, and writes it to the current working directory as `memofy-diag-<timestamp>.ndjson`.

**Rationale**: Simple subcommand implementation with no new dependencies. Output as NDJSON keeps the format consistent with the rolling log, so the same `jq` queries work on both.

---

### 1.7 Go Version & Dependencies

| Item | Decision |
|------|----------|
| Go version | 1.21 (existing minimum) |
| Structured logging | `log/slog` (stdlib, Go 1.21+) |
| Rolling file | Custom `rollingWriter` (~50 lines) using `os.File` |
| Session ID | `crypto/rand` + `encoding/hex` (stdlib) — 8 random bytes → 16-char hex string |
| New external deps | **None** |
| Testing | stdlib `testing` + existing `testutil` package |
| WebSocket | `gorilla/websocket` (already present, v1.5.3) |

---

## 2. Open Questions (Resolved)

All questions from the clarification session are resolved:

| Question | Answer |
|----------|--------|
| OBS WS protocol version | v5 (OBS 28+) |
| Log format | NDJSON — one JSON object per line |
| Export trigger | `memofy-core --export-diag` CLI subcommand |
| Manual session protection duration | Full session; debounce window (5 s) is race-condition guard only |
| Sensitive field redaction | Always redact; replace with `"[REDACTED]"` |

---

## 3. Risk Register

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| `requestID` counter race under concurrent `sendRequest` calls | Low (existing single-goroutine model) | Document; add mutex in same PR |
| `onRecordStateChanged` callback set after `Connect` races with `handleEvent` | Low | Callbacks registered before `Connect` in all current callers; document contract |
| Rolling log file write fails silently on disk full | Medium | `diaglog` must check write errors and downgrade gracefully (log to stderr only) |
| OBS v5 close code `4009` not always sent on multi-client | Medium | Add a startup PID-file check as primary guard; close code detection as secondary |
| Debounce window (5 s) too short for slow OBS startup | Low | Window is configurable; default can be tuned without code change |
