# Duplicate Instance Prevention

## Mechanism

**PID File**: `~/.cache/memofy/memofy-core.pid`

On start:
1. Check if PID file exists  
2. Read PID from file
3. Check if process alive (`kill -0 $PID`)
4. If alive → Exit with error
5. If stale → Remove file and continue
6. Write current PID to file

On exit:
- Remove PID file

## Testing

```bash
# Start daemon
~/.local/bin/memofy-core &

# Try to start again (should fail)
~/.local/bin/memofy-core
# Error: Another instance is already running (PID: xxxx)

# Kill daemon
kill $(cat ~/.cache/memofy/memofy-core.pid)
```

Prevents multiple daemons from controlling OBS simultaneously.
