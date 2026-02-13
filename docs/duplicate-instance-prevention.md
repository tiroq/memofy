# Duplicate Instance Prevention

Memofy prevents duplicate instances of both `memofy-core` and `memofy-ui` from running simultaneously using PID (Process ID) file management.

## How It Works

When each application starts:

1. **PID File Check**: The application checks for an existing PID file at:
   - `~/.cache/memofy/memofy-core.pid` for the core daemon
   - `~/.cache/memofy/memofy-ui.pid` for the menu bar UI

2. **Process Validation**: If a PID file exists:
   - The PID is read from the file
   - The system checks if a process with that PID is actually running
   - If the process exists: **Startup is blocked** with an error message
   - If the process is gone: The stale PID file is automatically removed

3. **PID File Creation**: If no valid instance is running:
   - A new PID file is created with the current process ID
   - The file is automatically removed when the application exits gracefully

## Benefits

- **Prevents resource conflicts**: Avoids multiple daemons competing for the same OBS connection
- **Prevents UI confusion**: Ensures only one menu bar icon appears
- **Auto-recovery**: Stale PID files from crashes are automatically cleaned up
- **Clear error messages**: Users know immediately if another instance is running

## Error Messages

If you try to start a duplicate instance, you'll see:

```
Failed to create PID file: another instance is already running (PID 12345)
Another instance of memofy-core may already be running.
If you're sure no other instance is running, remove: /Users/you/.cache/memofy/memofy-core.pid
```

## Manual Cleanup

If you encounter a stuck PID file after a crash:

```bash
# For memofy-core
rm ~/.cache/memofy/memofy-core.pid

# For memofy-ui
rm ~/.cache/memofy/memofy-ui.pid

# Or clean all PID files
rm ~/.cache/memofy/*.pid
```

## Implementation Details

The PID file management is implemented in `/internal/pidfile/`:

- **Auto-cleanup**: PID files are removed via `defer` when applications exit normally
- **Process detection**: Uses `Signal(0)` to check if a process exists without actually signaling it
- **Thread-safe**: PID file operations are atomic at the OS level
- **Cross-platform**: Works on macOS and Linux (uses Unix syscalls)

## Testing

Run the PID file tests:

```bash
go test ./internal/pidfile/... -v
```

Test scenarios covered:
- Creating a new PID file
- Detecting duplicate instances
- Removing stale PID files
- Cleanup on exit
- Process running detection

## Related Files

- `/internal/pidfile/pidfile.go` - PID file management implementation
- `/internal/pidfile/pidfile_test.go` - Comprehensive test suite
- `/cmd/memofy-core/main.go` - Core daemon integration
- `/cmd/memofy-ui/main.go` - Menu bar UI integration
- `/scripts/quick-install.sh` - Installation script with PID cleanup
