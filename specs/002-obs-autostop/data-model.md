# Data Model: OBS Auto-Stop Investigation & Safeguard

**Feature**: `002-obs-autostop`
**Date**: 2026-02-20

---

## 1. New Types

### 1.1 `RecordingOrigin` — Recording Session Authority

```go
// package: internal/statemachine

type RecordingOrigin string

const (
    OriginUnknown   RecordingOrigin = ""
    OriginManual    RecordingOrigin = "manual"    // user action via UI
    OriginAuto      RecordingOrigin = "auto"      // detection threshold crossed
    OriginForced    RecordingOrigin = "forced"    // programmatic / test path
)
```

**Priority order (highest first)**: `manual` > `auto` > `forced`

A stop signal from origin X is rejected if the active session was started by an origin with higher priority. The comparison is done by the authority check in `StopRecording`.

---

### 1.2 `RecordingSession` — Active Session Snapshot

```go
// package: internal/statemachine
// Embedded in StateMachine

type RecordingSession struct {
    SessionID   string          // 16-char hex, crypto/rand at start
    Origin      RecordingOrigin // who started it
    App         detector.DetectedApp
    StartedAt   time.Time
}
```

**Validation rules**:
- `SessionID` must be non-empty when `recording == true`
- `Origin` must be one of the declared constants; zero value (`""`) is treated as `OriginAuto` for backward compatibility during migration

**State transitions**:
```
idle
  → recording (manual)   via ForceStart()   – origin=manual, new SessionID
  → recording (auto)     via StartRecording() – origin=auto, new SessionID
recording
  → idle                 via StopRecording(requestOrigin)
      IF requestOrigin priority >= session.Origin priority → allowed
      ELSE → rejected, logged with "manual mode override"
```

---

### 1.3 `StopRequest` — Stop Signal with Attribution

```go
// package: internal/statemachine

type StopRequest struct {
    RequestOrigin RecordingOrigin // who is requesting the stop
    Reason        string          // machine-readable reason code (see §2)
    Component     string          // source component label (see §3)
}
```

Used as the parameter to `StopRecording(req StopRequest)` so every stop carries full attribution.

---

### 1.4 `LogEntry` — NDJSON Log Record

```go
// package: internal/diaglog

type LogEntry struct {
    Timestamp string      `json:"ts"`           // RFC3339Nano, e.g. "2026-02-20T13:42:10.213Z"
    Component string      `json:"component"`    // see §3
    Event     string      `json:"event"`        // see §4
    SessionID string      `json:"session_id,omitempty"`
    Reason    string      `json:"reason,omitempty"`
    Payload   interface{} `json:"payload,omitempty"` // redacted before write
}
```

Written as a single JSON object per line (NDJSON). All fields except `ts`, `component`, and `event` are optional.

---

### 1.5 `DiagBundle` — Export Bundle Structure

```go
// package: internal/diaglog

type DiagBundle struct {
    ExportedAt    string `json:"exported_at"`    // RFC3339
    MemofyVersion string `json:"memofy_version"`
    GoVersion     string `json:"go_version"`
    OS            string `json:"os"`
    Arch          string `json:"arch"`
    LogFile       string `json:"log_file"`       // source path
    EntryCount    int    `json:"entry_count"`
    // Followed by the raw NDJSON log entries in the same file
}
```

The export file begins with one `DiagBundle` metadata line (JSON), followed by all log lines from the rolling log. The entire file is valid NDJSON.

---

### 1.6 `StatusSnapshot` additions (`internal/ipc`)

New fields added to the existing `StatusSnapshot` struct:

```go
RecordingOrigin string `json:"recording_origin,omitempty"` // "manual" | "auto" | ""
SessionID       string `json:"session_id,omitempty"`
```

---

## 2. Reason Codes (machine-readable)

| Code | Used on | Meaning |
|------|---------|---------|
| `user_stop` | StopRequest | User explicitly clicked Stop |
| `auto_detection_stop` | StopRequest | Detection threshold: absence streak met |
| `reconnect_sync` | StopRequest | (REJECTED — must never be used; logged as bug) |
| `state_drift` | StopRequest | Local state mismatch with OBS |
| `manual_mode_override` | LogEntry | Stop rejected because session is manual |
| `session_end_normal` | LogEntry | Session ended normally |
| `connect_success` | LogEntry | OBS WS connected |
| `connect_failed` | LogEntry | Connection attempt failed |
| `disconnect_normal` | LogEntry | Graceful disconnect |
| `disconnect_unexpected` | LogEntry | Unexpected close |
| `disconnect_session_invalid` | LogEntry | OBS WS close code 4009 (multi-client) |
| `auth_challenge` | LogEntry | Auth handshake received |
| `auth_success` | LogEntry | Auth handshake completed |

---

## 3. Component Labels

| Label | Owns |
|-------|------|
| `obs-ws-client` | WebSocket send/receive, reconnect loop, auth |
| `state-machine` | Recording start/stop decisions, mode changes |
| `auto-detector` | Detection polling, threshold evaluation |
| `reconnect-handler` | Reconnect attempts and outcomes |
| `diag-export` | `--export-diag` CLI subcommand |
| `memofy-core` | Main process startup/shutdown |

---

## 4. Event Names

| Event | Component | Notes |
|-------|-----------|-------|
| `ws_send` | `obs-ws-client` | Outbound WS message (payload redacted) |
| `ws_recv` | `obs-ws-client` | Inbound WS message (payload redacted) |
| `ws_connect` | `obs-ws-client` | Connection established |
| `ws_disconnect` | `obs-ws-client` | Connection lost |
| `ws_reconnect_attempt` | `reconnect-handler` | Attempt N, delay D |
| `ws_reconnect_success` | `reconnect-handler` | Reconnected after N attempts |
| `ws_reconnect_failed` | `reconnect-handler` | All attempts exhausted |
| `recording_start` | `state-machine` | Session started (origin, session_id) |
| `recording_stop` | `state-machine` | Session ended (origin, reason) |
| `recording_stop_rejected` | `state-machine` | Stop blocked (requesting component, reason) |
| `mode_change` | `state-machine` | Mode transition (from → to) |
| `detection_state` | `auto-detector` | Detection tick (meeting detected, streaks) |
| `multi_client_warning` | `obs-ws-client` | Close code 4009 received |
| `export_start` | `diag-export` | Export initiated |
| `export_complete` | `diag-export` | Export file written |
