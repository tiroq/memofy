# Duplicate Instance Prevention - Implementation Summary

## Overview
Implemented comprehensive PID file management to prevent duplicate instances of both memofy-core and memofy-ui from running simultaneously.

## Implementation Details

### New Files Created

1. **`/internal/pidfile/pidfile.go`** (103 lines)
   - `PIDFile` struct for managing PID files
   - `New()` - Creates PID file and checks for existing instances
   - `Remove()` - Cleans up PID file on exit
   - `isProcessRunning()` - Checks if a process with given PID exists
   - `GetPIDFilePath()` - Returns standard PID file path

2. **`/internal/pidfile/pidfile_test.go`** (186 lines)
   - Comprehensive test suite with 7 test cases
   - Tests: creation, duplicate detection, stale file removal, cleanup, process detection
   - All tests passing ✅

3. **`/docs/duplicate-instance-prevention.md`** (86 lines)
   - Complete documentation for the feature
   - Usage examples and troubleshooting guide
   - Implementation details and testing instructions

### Modified Files

1. **`/cmd/memofy-core/main.go`**
   - Added import for `internal/pidfile`
   - Added PID file creation on startup (before permissions check)
   - Added deferred PID file cleanup
   - Exit with error if duplicate instance detected
   - PID file location: `~/.cache/memofy/memofy-core.pid`

2. **`/cmd/memofy-ui/main.go`**
   - Added import for `internal/pidfile`
   - Added PID file creation on startup (before app initialization)
   - Added deferred PID file cleanup
   - Exit with error if duplicate instance detected
   - PID file location: `~/.cache/memofy/memofy-ui.pid`

3. **`/scripts/quick-install.sh`**
   - Added PID file cleanup in `install_binaries()` function
   - Added PID file cleanup in `start_ui()` function
   - Ensures clean state before installation and startup

## How It Works

### Startup Sequence

1. **Application starts** → Initialize logging
2. **Create PID file** → Check for existing instance
   - If PID file exists and process is running → **EXIT with error**
   - If PID file exists but process is dead → Remove stale file and continue
   - If no PID file → Create new one with current PID
3. **Register cleanup** → `defer pf.Remove()` ensures cleanup on exit
4. **Continue normal startup** → Permissions, OBS connection, etc.

### Error Messages

When duplicate instance is detected:
```
Failed to create PID file: another instance is already running (PID 12345)
Another instance of memofy-core may already be running.
If you're sure no other instance is running, remove: /Users/you/.cache/memofy/memofy-core.pid
```

### PID File Locations

- **memofy-core**: `~/.cache/memofy/memofy-core.pid`
- **memofy-ui**: `~/.cache/memofy/memofy-ui.pid`

## Features

✅ **Automatic stale file recovery** - Removes PID files from crashed processes
✅ **Graceful cleanup** - PID files removed on normal exit
✅ **Clear error messages** - Tells users exactly what's wrong and how to fix it
✅ **Thread-safe** - Uses OS-level atomic file operations
✅ **Cross-platform** - Works on macOS and Linux
✅ **Well-tested** - 7 comprehensive test cases, all passing

## Testing

### Automated Tests
```bash
go test ./internal/pidfile/... -v
```

All 7 tests passing:
- ✅ TestNewPIDFile
- ✅ TestDuplicateInstance  
- ✅ TestStalePIDFile
- ✅ TestRemovePIDFile
- ✅ TestRemoveOnlyOwnPID
- ✅ TestGetPIDFilePath
- ✅ TestIsProcessRunning

### Manual Testing

To test duplicate prevention:

```bash
# Clean state
pkill memofy-core
rm -f ~/.cache/memofy/memofy-core.pid

# Start first instance
./bin/memofy-core &

# Try to start second instance (should fail)
./bin/memofy-core
# Expected: Error message and exit code 1

# Clean up
pkill memofy-core
```

## Benefits

1. **Prevents resource conflicts**
   - Only one daemon connects to OBS WebSocket
   - No race conditions on status.json writes

2. **Prevents UI confusion**
   - Only one menu bar icon appears
   - No duplicate notifications

3. **Improves reliability**
   - Automatic recovery from crashes
   - Clear error diagnostics

4. **Better user experience**
   - Informative error messages
   - Easy manual recovery if needed

## Installation Impact

The installation scripts have been updated to:
- Clean up any stale PID files before installing new binaries
- Clean up PID files when restarting the UI
- Handle edge cases where processes were killed without cleanup

## Backward Compatibility

✅ Fully backward compatible - PID files are created automatically on first run
✅ No configuration required - works out of the box
✅ No breaking changes to existing functionality

## Future Enhancements

Potential improvements for future versions:
- [ ] Add PID file monitoring to detect zombie processes
- [ ] Implement socket-based locking for more robust detection
- [ ] Add auto-restart option if previous instance is detected as dead
- [ ] Create unified cleanup command: `memofy cleanup --all-pids`

## Related Files

- Implementation: `/internal/pidfile/pidfile.go`
- Tests: `/internal/pidfile/pidfile_test.go`
- Core integration: `/cmd/memofy-core/main.go`
- UI integration: `/cmd/memofy-ui/main.go`
- Install script: `/scripts/quick-install.sh`
- Documentation: `/docs/duplicate-instance-prevention.md`

## Build Status

✅ `make build` - Success (both binaries compile)
✅ `go test ./internal/pidfile/...` - All tests pass
✅ `make clean && make build` - Clean build successful
