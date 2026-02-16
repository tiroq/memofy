# Troubleshooting

## Quick Diagnostics

```bash
memofy-ctl diagnose     # if available
pgrep -fl memofy
ps aux | grep OBS
nc -zv localhost 4455
cat /tmp/memofy-core.err.log
```

## Error Code 204

**Cause**: OBS < 28.0 or WebSocket not enabled

**Fix**:
```bash
# Update OBS to 28.0+
# OBS → Tools → obs-websocket Settings → Enable (port 4455)
task restart
```

## Missing Sources

 **Cause**: Scene empty, sources disabled, or OBS version issue

**Fix**:
```bash
# Manually add Display Capture + Audio Input to OBS scene
# Or wait for auto-creation on next start
```

## Daemon Won't Start

```bash
tail -f /tmp/memofy-core.err.log
# Check permissions: System Settings → Privacy & Security
launchctl unload ~/Library/LaunchAgents/com.memofy.core.plist
~/.local/bin/memofy-core    # Run manually to see errors
```

## Black/Silent Recordings

```bash
# Grant Screen Recording permission
# System Settings → Privacy & Security → Screen Recording
# Restart OBS and memofy
```

## OBS Won't Connect

```bash
# Verify WebSocket enabled
nc -zv localhost 4455
# Check OBS logs
# Restart OBS
```

See logs at `/tmp/memofy-core.{out,err}.log` for details.
