# PID File Management - Known Issues and Solutions

## Issue: Process killed but PID file remains

### Symptom
```
Failed to create PID file: another instance is already running (PID 12345)
```
But when you check, no process is actually running.

### Cause
When a process is killed with `SIGKILL` (kill -9), the `defer pf.Remove()` cleanup doesn't execute, leaving behind a stale PID file.

### Solution

**Option 1: Use the management script**
```bash
# Install the helper script
cp scripts/memofy-ctl.sh ~/.local/bin/memofy-ctl
chmod +x ~/.local/bin/memofy-ctl

# Check status
memofy-ctl status

# Stop gracefully (SIGTERM allows cleanup)
memofy-ctl stop ui

# Clean everything
memofy-ctl clean
```

**Option 2: Manual cleanup**
```bash
# Check if process is actually running
ps aux | grep memofy-ui

# If not running, remove stale PID file
rm ~/.cache/memofy/memofy-ui.pid

# For core daemon
rm ~/.cache/memofy/memofy-core.pid
```

**Option 3: Use graceful kill**
```bash
# Use SIGTERM instead of SIGKILL
killall memofy-ui  # This is SIGTERM by default

# Wait for cleanup
sleep 2

# Verify PID file is gone
ls ~/.cache/memofy/*.pid
```

## Issue: Quick-install script kills process immediately

### Symptom
```
Killed: 9 nohup "$INSTALL_DIR/memofy-ui" > /tmp/memofy-ui.out.log 2>&1
✓ Memofy is running in menu bar
```

### Cause
The install script kills existing processes but doesn't wait long enough for:
1. The `defer pf.Remove()` to execute
2. The previous process to fully terminate
3. The PID file to be cleaned up

### Solution
The quick-install.sh script has been updated to:
1. Use SIGTERM (graceful) instead of SIGKILL
2. Wait 2 seconds for cleanup
3. Remove PID files manually as a safety measure
4. Use the memofy-ctl helper script when available

## Recommendations

### For Users
1. **Always use graceful shutdown when possible**
   ```bash
   killall memofy-ui      # Correct (SIGTERM)
   killall -9 memofy-ui   # Avoid (SIGKILL)
   ```

2. **Use the management script**
   ```bash
   memofy-ctl stop ui     # Handles cleanup automatically
   ```

3. **If stuck, clean manually**
   ```bash
   rm ~/.cache/memofy/*.pid
   ```

### For Developers

1. **Signal handling is implemented** in both binaries to catch SIGTERM and SIGINT
2. **Always test with graceful shutdown** during development
3. **The PID file check works correctly** - if it says another instance is running, one probably is!

## Testing PID File Behavior

```bash
# Test graceful shutdown
./bin/memofy-ui &
PID=$!
sleep 2
cat ~/.cache/memofy/memofy-ui.pid  # Should show PID
kill -15 $PID  # SIGTERM
sleep 1
ls ~/.cache/memofy/memofy-ui.pid  # Should be gone ✓

# Test forced kill (leaves stale file)
./bin/memofy-ui &
PID=$!
sleep 2
cat ~/.cache/memofy/memofy-ui.pid  # Should show PID
kill -9 $PID  # SIGKILL
sleep 1
ls ~/.cache/memofy/memofy-ui.pid  # Still exists ✗ (expected)
rm ~/.cache/memofy/memofy-ui.pid   # Manual cleanup needed
```

## Files Updated

- **cmd/memofy-ui/main.go** - Added signal handler for SIGTERM/SIGINT
- **scripts/quick-install.sh** - Updated to use graceful shutdown
- **scripts/memofy-ctl.sh** - New process management helper
- **docs/PID_FILE_ISSUES.md** - This documentation

## Future Improvements

Potential enhancements:
- [ ] Add timestamp to PID file to detect very old stale files
- [ ] Implement flock() for atomic locking
- [ ] Add auto-recovery check on startup (detect and clean stale files automatically)
- [ ] Create systemd/launchd integration for proper process supervision
