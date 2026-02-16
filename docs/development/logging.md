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
