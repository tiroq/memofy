# ipc — UI↔Daemon Communication

## OVERVIEW

File-based IPC via `~/.cache/memofy/`. Two files only: `cmd.txt` (UI→daemon commands) and `status.json` (daemon→UI state). All writes are atomic (temp file + rename).

## FILES

| File | Direction | Writer | Reader |
|------|-----------|--------|--------|
| `~/.cache/memofy/cmd.txt` | UI → daemon | `ipc.WriteCommand()` | daemon poll loop |
| `~/.cache/memofy/status.json` | daemon → UI | `ipc.WriteStatus()` | `ipc.ReadStatus()` |

## COMMANDS (cmd.txt)

```go
CmdStart  = "start"   // Start recording immediately
CmdStop   = "stop"    // Stop recording immediately
CmdToggle = "toggle"  // Toggle recording
CmdAuto   = "auto"    // Switch to auto mode
CmdManual = "manual"  // Manual mode (detect but never auto-start OBS)
CmdPause  = "pause"   // Suspend all detection
CmdQuit   = "quit"    // Shutdown daemon
```

`ReadCommand()` clears the file after reading — each command fires once.

## OPERATING MODES (status.json → OperatingMode)

```go
ModeAuto   = "auto"    // Detection-driven recording
ModeManual = "manual"  // User-controlled only
ModePaused = "paused"  // All detection suspended
```

## STATUS SNAPSHOT

`StatusSnapshot` carries: mode, detection state, recording state, per-app booleans, streaks, last action/error, OBS connection, recording origin + session ID.

## ANTI-PATTERNS

- **Never write cmd.txt directly** — use `ipc.WriteCommand()`
- **Never write status.json directly** — use `ipc.WriteStatus()` (atomic)
- **Never read status.json outside `ipc`** — use `ipc.ReadStatus()`
- **No channels** — UI polls via fsnotify watcher; daemon polls on tick
