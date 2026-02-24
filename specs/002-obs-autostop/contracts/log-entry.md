# Contract: NDJSON Log Entry Schema

**Feature**: `002-obs-autostop`
**Type**: Structured log format contract
**Stability**: Internal (diaglog package)

---

## Log Line Schema

Every line in `memofy-debug.log` (and in diagnostic export bundles) is a single JSON object conforming to this schema.

```jsonc
{
  "ts":          "<RFC3339Nano timestamp>",   // required; e.g. "2026-02-20T13:42:10.213Z"
  "component":   "<label>",                   // required; see Component Labels in data-model.md
  "event":       "<event_name>",              // required; see Event Names in data-model.md
  "session_id":  "<16-char hex>",             // optional; present when event relates to a recording session
  "reason":      "<reason_code>",             // optional; present when event involves a decision
  "payload":     { ... }                      // optional; sensitive fields ALWAYS replaced with "[REDACTED]"
}
```

---

## Example Log Lines

**WebSocket connect:**
```json
{"ts":"2026-02-20T13:42:10.213Z","component":"obs-ws-client","event":"ws_connect","payload":{"obs_version":"30.2.2","ws_version":"5.5.1"}}
```

**Outbound command with reason:**
```json
{"ts":"2026-02-20T13:43:22.014Z","component":"obs-ws-client","event":"ws_send","session_id":"a3f1b2c9d4e50001","reason":"user_stop","payload":{"request_type":"StopRecord","request_id":"7"}}
```

**Stop rejected:**
```json
{"ts":"2026-02-20T13:43:22.015Z","component":"state-machine","event":"recording_stop_rejected","session_id":"a3f1b2c9d4e50001","reason":"manual_mode_override","payload":{"requesting_component":"reconnect-handler","requesting_reason":"reconnect_sync"}}
```

**Reconnect attempt:**
```json
{"ts":"2026-02-20T13:43:27.001Z","component":"reconnect-handler","event":"ws_reconnect_attempt","payload":{"attempt":1,"delay_ms":5000,"disconnect_reason":"websocket: close 1006 unexpected EOF"}}
```

**Auth (redacted):**
```json
{"ts":"2026-02-20T13:42:10.100Z","component":"obs-ws-client","event":"ws_recv","payload":{"op":0,"authentication":{"challenge":"[REDACTED]","salt":"[REDACTED]"}}}
```

---

## Redacted Fields

The following JSON keys are **always** replaced with the string `"[REDACTED]"` regardless of nesting depth:

- `authentication`
- `password`
- `secret`
- `challenge`
- `salt`
- `auth`

Redaction is applied recursively to the entire payload tree before serialisation.

---

## Rolling File Behaviour

| Property | Value |
|----------|-------|
| Default path | `/tmp/memofy-debug.log` |
| Env override | `MEMOFY_LOG_PATH` |
| Max size | 10 MB |
| Overflow policy | Truncate to empty and continue writing (not rotate) |
| Flush policy | Each line flushed immediately (`os.File.Sync` after each write) |
| Concurrent writes | Serialised by a mutex inside `rollingWriter` |

---

## Diagnostic Export Bundle (NDJSON)

The file produced by `memofy-core --export-diag` is also NDJSON:

- **Line 1**: metadata object (`DiagBundle` struct)
- **Lines 2–N**: raw log lines copied verbatim from the rolling log

All lines in the export file conform to this same schema. The file is named:

```
memofy-diag-<YYYYMMDDTHHmmss>.ndjson
```

Written to the current working directory at the time of the `--export-diag` invocation.
