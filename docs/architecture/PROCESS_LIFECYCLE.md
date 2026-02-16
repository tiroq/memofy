# Process Lifecycle

## State Machine

**memofy-core**: STOPPED → STARTING → INITIALIZING → RUNNING → STOPPING  
**memofy-ui**: STOPPED → STARTING → RUNNING → STOPPING  
**OBS**: Auto-launched by core if not running

## Core States

**STARTING** (0-5s): Check permissions, load config, locate OBS  
**INITIALIZING** (5-25s): Connect to OBS, validate version, check/create sources  
**RUNNING**: Monitor meetings every 2s, coordinate recording  
**RECONNECTING**: Auto-reconnect to OBS (5s → 10s → 20s → 40s → 60s backoff)

## Files

```
~/.cache/memofy/memofy-core.pid        # PID file
/tmp/memofy-core.status.json           # Status (updated every 2s)
/tmp/memofy-core.{out,err}.log         # Logs
```

## Health Indicators

- Logs show `[EVENT]` entries when meetings detected
- Status file updates every 2s
- Memory ~30-50MB stable
- OBS reconnect succeeds within 5min

See logs for detailed state transitions.
