# Logging

## Log Files

```
/tmp/memofy-core.out.log        # Stdout (info)
/tmp/memofy-core.err.log        # Stderr (errors)
~/.cache/memofy/memofy-ui.log   # UI logs
```

## View Logs

```bash
task logs           # Daemon output
task logs-error     # Daemon errors
task logs-ui        # UI logs

# Or directly
tail -f /tmp/memofy-core.out.log
```

## Log Tags

```
[STARTUP]           # Initialization
[EVENT]             # Meeting detection events
[RECONNECT]         # OBS reconnection
[ERROR]             # Errors
[SOURCE_FOUND]      # Source configuration
```

## Log Rotation

Logs not rotated automatically. Clean manually if needed:
```bash
> /tmp/memofy-core.out.log
> /tmp/memofy-core.err.log
```

## Debug Mode

Run daemon in foreground:
```bash
task dev-daemon
```

---

## Diagnostic Recording Log (002-obs-autostop)

A structured NDJSON diagnostic log is available for deep debugging of recording-lifecycle events (WS traffic, reconnects, stop authority decisions).

### Enable

```bash
MEMOFY_DEBUG_RECORDING=true memofy-core
```

Or set permanently in the launchd plist:
```xml
<key>EnvironmentVariables</key>
<dict>
    <key>MEMOFY_DEBUG_RECORDING</key>
    <string>true</string>
</dict>
```

Default log file: `/tmp/memofy-debug.log` (10 MB rolling — oldest entries truncated on overflow).

Override the path:
```bash
MEMOFY_LOG_PATH=/var/log/memofy-debug.log memofy-core
```

### Watch Events in Real Time

```bash
tail -f /tmp/memofy-debug.log | jq .
```

Filter for stop-related events only:
```bash
tail -f /tmp/memofy-debug.log | jq 'select(.event | test("stop|disconnect|reconnect"))'
```

Find who sent any `StopRecord` command:
```bash
jq 'select(.event == "ws_send" and .payload.request_type == "StopRecord")' /tmp/memofy-debug.log
```

Find rejected stops:
```bash
jq 'select(.event == "recording_stop_rejected")' /tmp/memofy-debug.log
```

### Export a Diagnostic Bundle

Bundles the current log with system metadata into a single NDJSON file for sharing:

```bash
memofy-core --export-diag
# Wrote: ./memofy-diag-20260220T134222.ndjson (1842 lines)
```

Or specify a custom output directory:
```bash
MEMOFY_LOG_PATH=/var/log/memofy-debug.log memofy-core --export-diag /tmp/diag-out/
```

Exit codes: `0` success, `1` no log file found, `2` file unreadable, `3` output directory error.

### Adjust the Debounce Window

The debounce window blocks automated stop signals arriving within the first N ms of a session start (guards against race conditions):

```bash
MEMOFY_DEBUG_RECORDING=true MEMOFY_MANUAL_DEBOUNCE_MS=10000 memofy-core
```

Default: `5000` ms (5 seconds). Set to `0` to disable.

### Logged Events

| Event | Component | Description |
|-------|-----------|-------------|
| `ws_connect` | `obs-client` | OBS WebSocket authentication succeeded |
| `ws_disconnect` | `obs-client` | WebSocket disconnected |
| `ws_reconnect_attempt` | `obs-reconnect` | Reconnect attempt started |
| `ws_reconnect_success` | `obs-reconnect` | Reconnect succeeded |
| `ws_reconnect_failed` | `obs-reconnect` | All reconnect attempts exhausted |
| `ws_send` | `obs-client` | OBS request sent (includes `reason` for StopRecord) |
| `ws_recv` | `obs-client` | OBS message received |
| `multi_client_warning` | `obs-client` | OBS close code 4009 — another client connected |
| `recording_start` | `state-machine` | Recording session started |
| `recording_stop` | `state-machine` | Recording session stopped |
| `recording_stop_rejected` | `state-machine` | Stop rejected by authority/debounce guard |

