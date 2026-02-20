# Quickstart: OBS Auto-Stop Investigation & Safeguard

**Feature**: `002-obs-autostop`
**For**: Developers reproducing or verifying the fix

---

## Enable Debug Logging

```sh
MEMOFY_DEBUG_RECORDING=true memofy-core
```

Or set it permanently in the launchd plist:
```xml
<key>EnvironmentVariables</key>
<dict>
    <key>MEMOFY_DEBUG_RECORDING</key>
    <string>true</string>
</dict>
```

The rolling log appears at `/tmp/memofy-debug.log`.

---

## Watch Events in Real Time

```sh
tail -f /tmp/memofy-debug.log | jq .
```

Filter for stop-related events only:
```sh
tail -f /tmp/memofy-debug.log | jq 'select(.event | test("stop|disconnect|reconnect"))'
```

Find who sent any StopRecord command:
```sh
jq 'select(.event == "ws_send" and .payload.request_type == "StopRecord")' /tmp/memofy-debug.log
```

Find rejected stops:
```sh
jq 'select(.event == "recording_stop_rejected")' /tmp/memofy-debug.log
```

---

## Reproduce the Investigation Scenario

1. Start OBS with WebSocket server enabled (port 4455).
2. Enable debug logging (see above) and start Memofy.
3. Manually start recording from the Memofy menu.
4. Wait for the suspected auto-stop to occur (or simulate it — see below).
5. Stop Memofy.
6. Export the diagnostic bundle:

```sh
memofy-core --export-diag
# Wrote: ./memofy-diag-20260220T134222.ndjson (1842 lines)
```

7. Open the bundle:
```sh
cat memofy-diag-20260220T134222.ndjson | jq .
```

---

## Simulate Auto-Stop Triggers (for testing the safeguard)

**Simulate a reconnect-triggered stop** (send a stop from the reconnect handler):

This is now blocked by the authority check. To verify:
1. Start a manual recording.
2. Disconnect OBS WebSocket (disable the server and re-enable).
3. Confirm in the log that reconnect completes *without* a `StopRecord` command being sent.
4. Confirm the recording is still active.

**Simulate a detection-absence stop**:
1. Start a manual recording.
2. Kill the meeting app process (Zoom, Teams, etc.).
3. Wait for the absence streak to fill.
4. Confirm the log shows `recording_stop_rejected` with reason `manual_mode_override`.

---

## Adjust the Debounce Window

The debounce window protects against race conditions at session start (automated signals arriving in the first N milliseconds):

```sh
MEMOFY_DEBUG_RECORDING=true MEMOFY_MANUAL_DEBOUNCE_MS=10000 memofy-core
```

---

## Run Tests for This Feature

```sh
# State machine safeguard tests
go test ./internal/statemachine/... -v -run TestManual

# Diaglog tests
go test ./internal/diaglog/... -v

# OBS WebSocket logging injection tests
go test ./internal/obsws/... -v -run TestLog

# All feature tests
go test ./... -run "TestManual|TestDiaglog|TestLog|TestExport"
```
