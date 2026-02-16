# Memofy Process Lifecycle

## State Machine Overview

Memofy consists of two interlinked processes with distinct lifecycles:

```
┌─────────────────────────────────────────────────────────────┐
│ memofy-core                                                 │
│ (Meeting detection + OBS recording orchestration)           │
│                                                             │
│  STOPPED → STARTING → INITIALIZING → RUNNING → STOPPING    │
│            (0-5s)     (5-25s)        (auto)   (1-5s)        │
│                                      ↓                       │
│                                   RECONNECTING (on OBS loss) │
│                                   (exponential backoff)      │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│ memofy-ui                                                   │
│ (macOS menu bar UI)                                         │
│                                                             │
│  STOPPED → STARTING → RUNNING → STOPPING                    │
│           (0-10s)    (always)   (1-2s)                      │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│ OBS                                                         │
│ (External recording application)                           │
│                                                             │
│ If not running: memofy-core launches it automatically      │
│ If disconnected: memofy-core reconnects with backoff       │
│ If version incompatible: memofy-core warns at startup      │
└─────────────────────────────────────────────────────────────┘
```

---

## Process States Explained

### memofy-core States

#### STOPPED
- **What it means**: Process is not running
- **PID file state**: Doesn't exist or is stale
- **Log location**: Last logs at `/tmp/memofy-core.out.log`
- **What user will see**: `memofy-ctl status` shows "memofy-core not running"

#### STARTING
- **Duration**: 0-5 seconds
- **What happens**: 
  - Checks for existing process
  - Verifies permissions
  - Loads configuration
  - Attempts to locate/launch OBS
- **Log output**: `[STARTUP]` messages for each phase
- **Danger zone**: If this takes >5s, may indicate permission issues or slow system
- **What can go wrong**:
  - Permission denied → Log shows permission error → Process exits
  - Config file missing → Creates default → Continues

#### INITIALIZING  
- **Duration**: 5-25 seconds (expected ~20s)
- **What happens**:
  1. Connects to OBS WebSocket (5-15s)
  2. Validates OBS version compatibility (1s)
  3. Checks/creates scene sources (5-10s)
  4. Sets up event handlers (1s)
  5. Initializes detection state machine (1s)
- **Log output**: `[STARTUP]` messages with status
- **Critical path**: 
  - OBS not running = auto-launch (adds 10-30s to timeline)
  - OBS version incompatible = warns but continues
  - Source creation fails with code 204 = retries 3x
- **What can go wrong**:
  - OBS WebSocket timeout → Exits with error
  - Permission denied to screen record → Exits with error
  - Source creation fails 3x → Warns but continues

#### RUNNING
- **What it means**: Process is fully initialized and monitoring
- **PID file**: Located at `~/.cache/memofy/memofy-core.pid` (contains PID number)
- **What it does**: Every 2 seconds, checks active application for meeting presence
- **Log output**: Only `[EVENT]` and status updates when things change
- **Expected stability**: Can run indefinitely until stopped or crash
- **Health indicators**:
  - Logs should show `[EVENT]` entries regularly (when meeting detected)
  - Status file `/tmp/memofy-core.status.json` updated every 2s
  - Memory usage stable (typically 30-50 MB)

#### RECONNECTING (Automatic)
- **When it happens**: OBS connection drops (network issue, OBS crashed, etc.)
- **Duration**: Up to 5 minutes (exponential backoff)
- **Backoff sequence**: 5s → 10s → 20s → 40s → 60s (then repeats 60s)
- **What happens**:
  - Detects connection lost
  - Logs reconnection attempt with [RECONNECT] tag
  - Waits before retry (increasing delay)
  - On reconnect success: Validates sources again, resumes recording coordination
- **Log output**:
  ```
  [RECONNECT] Attempting to reconnect to OBS (attempt 1/unlimited)
  [RECONNECT] Reconnecting in 5s...
  [RECONNECT] Reconnection successful
  ```
- **What user sees in memofy-ui**: Status shows "OBS Disconnected" → "OBS Reconnecting" → "OBS Connected"
- **Automatic behavior**:
  - If memofy-ui was showing status, updates to show disconnection
  - Recording state preserved (if recording, continues when reconnected)
  - Detection continues (no impact on meeting detection logic)

#### STOPPING
- **Duration**: 1-5 seconds
- **When it happens**: User runs `memofy-ctl stop` or system signals termination
- **Signal sequence**:
  1. Receives SIGTERM (graceful shutdown request)
  2. Stops detection loop (max 2 seconds to finish current cycle)
  3. Stops any active recording
  4. Closes OBS connection gracefully
  5. Removes PID file
  6. Exits with code 0 (success)
- **Log output**:
  ```
  [SHUTDOWN] Graceful shutdown requested
  [SHUTDOWN] Stopping detection loop...
  [SHUTDOWN] Closing OBS connection...
  [SHUTDOWN] Shutdown complete
  ```
- **If timeout (>5s)**:
  - System sends SIGKILL (force terminate)
  - Records death info in `.died` file
  - Next startup will show "Previous death: killed at 2026-02-14 15:35:00"

---

### memofy-ui States

#### STOPPED
- **What it means**: Menu bar app is not running
- **Platform specifics**: On macOS, check "Cmd+Tab" app switcher
- **Expected**: If memofy-core running but memofy-ui stopped, recording still happens (ui is optional)

#### STARTING
- **Duration**: 0-10 seconds
- **What happens**:
  1. Initializes macOS SharedApplication
  2. Creates status bar menu and icon
  3. Loads current status from file
  4. Starts watching status file for updates
- **Log output**: `[STARTUP]` messages with progress
- **Watchdog**: 15-second timeout
  - If initialization takes >15s → Exits with error
  - Problem typically: App stuck waiting for screen permission
- **Progress indicators**: Every 2 seconds during init, logs "...UI initialization in progress..."

#### RUNNING
- **What it means**: Menu bar icon visible and operational
- **UI location**: Top-right macOS menu bar (next to clock, battery, etc.)
- **Click behaviors**:
  - Click icon: Shows current status (connected, recording, error, etc.)
  - Click "Pause": Temporarily disables recording
  - Click "Resume": Re-enables recording
- **Status updates**: Watches `~/.cache/memofy/status.json` file
  - When file changes: Immediately updates UI
  - Latency: <100ms from status change to UI update
- **Expected stability**: Lightweight, uses <10MB RAM
- **Error states**: Shows error icon if memofy-core disconnected

#### STOPPING
- **Duration**: 1-2 seconds
- **When it happens**: User quits app or system shutdown
- **What happens**:
  1. Closes menu bar window
  2. Removes PID file
  3. Exits gracefully
- **Note**: Stopping memofy-ui does NOT stop memofy-core or recording

---

## OBS Integration Lifecycle

### OBS Not Running

**Timeline**:
1. memofy-core starts (sees OBS not running)
2. Attempts to launch OBS with `open -a OBS`
3. OBS startup takes 10-30 seconds
4. Once OBS window appears, continues with WebSocket connection

**Log output**:
```
[STARTUP] Checking OBS status...
[STARTUP] OBS is not running, attempting to launch...
[STARTUP] Waiting for OBS to start...
... wait 10-30 seconds ...
[STARTUP] OBS process detected, connecting WebSocket...
[STARTUP] Connected to OBS 29.1.3...
```

**User experience**:
- `memofy-ctl start` may take 30-40 seconds total (instead of 25s)
- OBS window pops up automatically
- Nothing shown in memofy-ui until OBS is ready

### OBS Crashes

**Timeline**:
1. memofy-core detects socket disconnection
2. Logs `[RECONNECT]` attempt
3. Retries every 5s initially (exponential backoff)
4. If user restarts OBS manually, reconnects immediately

**Log output**:
```
[ERROR] OBS connection lost: read error
[RECONNECT] Attempting to reconnect to OBS (attempt 1)
[RECONNECT] Reconnecting in 5s...
... OBS restarted by user ...
[RECONNECT] Reconnection successful after attempt 2
```

**User experience**:
- memofy-ui shows "OBS Disconnected ⚠️"
- Recording stops automatically
- When OBS restarts, reconnects and resumes
- No data loss if recording was in progress

### OBS WebSocket Method Error (Code 204)

**What triggers it**: 
- OBS version < 28.0 (too old, missing WebSocket v5)
- obs-websocket plugin not installed or disabled
- Request uses deprecated syntax

**Timeline**:
1. memofy-core attempts to create source
2. OBS returns 204 error
3. Logs special-case error: "OBS rejected request type 'CreateInput' (code 204)"
4. Indicates likely version issue
5. CreateSourceWithRetry sees 204, fast-fails (stops retrying)

**Log output**:
```
[CREATE_RETRY] Attempting source creation (attempt 1/3)...
[ERROR] OBS rejected request type 'CreateInput' (code 204: InvalidRequest)
[ERROR] This likely indicates an OBS version or plugin compatibility issue.
[WARN] Suggesting action: Update OBS to 28.0+
```

**Recovery**:
1. User runs `memofy-ctl diagnose`
2. Confirms OBS version < 28.0
3. Downloads new OBS from obsproject.com
4. Restarts: `memofy-ctl restart`

---

## Graceful Shutdown vs Forced Termination

### Graceful Shutdown (SIGTERM)

**Timeline**:
```
User: memofy-ctl stop
         ↓
System sends SIGTERM (-15)
         ↓
Process receives signal (1-2s)
         ↓
Finishes current detection cycle (max 2 more seconds)
         ↓
Stops recording if active
         ↓
Closes OBS connection gracefully
         ↓
Removes PID file
         ↓
Process exits with code 0
         ↓
Total: 1-5 seconds
```

**Log output**:
```
[SHUTDOWN] Graceful shutdown requested (SIGTERM)
[SHUTDOWN] Stopping detection loop...
[SHUTDOWN] Current recording state: STOPPED
[SHUTDOWN] Closing OBS connection...
[SHUTDOWN] OBS disconnected cleanly
[SHUTDOWN] Removing PID file...
[SHUTDOWN] Shutdown complete
```

**What's saved**:
- Recording files are finalized cleanly
- OBS state saved
- No corruption expected

### Forced Termination (SIGKILL)

**When it happens**:
- User force-quits process: `kill -9 <pid>`
- Watchdog timeout on shutdown (>5s)
- System runs out of memory
- OOM killer activates

**Timeline**:
```
System sends SIGKILL (-9)
         ↓
Process terminates IMMEDIATELY (no cleanup possible)
         ↓
Partial data may be lost
         ↓
Recording file may be incomplete
         ↓
OCS connection not closed gracefully
```

**Death tracking**:
- memofy-ctl records why process died
- Writes to `~/.cache/memofy/memofy-core.died` file
- Contents: timestamp + reason
- On next startup, displays "Previous death info" to user

**Example death file**:
```
Timestamp: 2026-02-14T15:35:42+07:00
Reason: Forced kill (SIGKILL) sent to PID 12345
Cause: Watchdog timeout during shutdown (UI hang detected)
```

---

## State Transition Rules

### memofy-core

```
STOPPED ──(start command)──→ STARTING ──(init success)──→ INITIALIZING ──(ready)──→ RUNNING
  ↑                                          ↓                    ↓
  └────────────(stop signal)────────────────┴────────────(STOPPING)────→ (exit)

RUNNING ──(OBS lost)──→ RECONNECTING ──(backoff wait)──→ RECONNECTING ---(reconnect success)--→ RUNNING
         ──(stop signal)──→ STOPPING ──→ exit
         ──(crash)──→ (force exit)
```

### memofy-ui

```
STOPPED ──(start command)──→ STARTING ──(init success)──→ RUNNING
  ↑                              ↓
  └────────────(stop signal)─────┴──→ STOPPING → (exit)

RUNNING ──(stop signal)──→ STOPPING → exit
        ──(crash)──→ (force exit)
```

### Cross-Process State

```
memofy-core RUNNING + memofy-ui RUNNING
  ↓
Both handling user interactions, status synced via file

memofy-core RUNNING + memofy-ui STOPPED
  ↓
Recording still works, just no menu bar icon
  ↓
User can press Cmd+Space → type "memofy-ui" to restart UI

memofy-core STOPPED + memofy-ui RUNNING
  ↓
UI shows "memofy-core disconnected ⚠️"
  ↓
No recording happens until memofy-core restarted
```

---

## Log File Lifecycle

### Log Rotation

**Core logs**:
- Stdout: `/tmp/memofy-core.out.log` (append-only)
- Stderr: `/tmp/memofy-core.err.log` (append-only)
- Max size: Grows indefinitely (manual truncation recommended)
- Persistence: "/tmp" clears on macOS reboot

**UI logs**:
- Stdout: `/tmp/memofy-ui.out.log` (append-only)
- No stderr separate file
- Max size: Grows indefinitely

**Status file**:
- Location: `~/.cache/memofy/status.json`
- Updated every 2 seconds by memofy-core
- Read every <100ms by memofy-ui
- Format: JSON with timestamp

### Cleanup

**Manual cleanup** (recommended monthly):
```bash
# Clear all logs (processes keep running)
> /tmp/memofy-core.out.log
> /tmp/memofy-core.err.log
> /tmp/memofy-ui.out.log

# Or use memofy-ctl
memofy-ctl logs --clear
```

**Automatic cleanup** on clean:
```bash
# memofy-ctl clean removes:
# - PID files
# - Death files
# - Stale socket connections
# (Does NOT remove logs)

memofy-ctl clean
```

---

## PID File Management

### PID File Locations

```
~/.cache/memofy/memofy-core.pid
~/.cache/memofy/memofy-ui.pid
```

### PID File Content

```
12345
```

Just the process ID, one per line.

### Stale PID Files

**What makes a PID "stale"**:
- File exists but process no longer running
- System rebooted (PID reused by different process)
- Process was killed with SIGKILL

**Detection**:
```bash
# Automatic on `memofy-ctl start`:
# Checks if PID in file is actually running
# If not: Shows "stale PID" warning + removes file

# Manual check:
ps -p $(cat ~/.cache/memofy/memofy-core.pid) 2>/dev/null || echo "stale"
```

**Death file companion**:
- When SIGKILL used: Creates `.died` file
- On next startup: Shows "Previous death info"
- Example:
  ```
  Previous death info: Forced kill (SIGKILL) sent to PID 12345
  Recommending graceful restart...
  ```

---

## Memory and Resource Management

### Expected Memory Usage

| Process | Idle RAM | Active Recording | Notes |
|---------|----------|------------------|-------|
| memofy-core | 30-50 MB | 35-60 MB | Polling detection every 2s |
| memofy-ui | 5-10 MB | 5-10 MB | Very lightweight |
| OBS | 200-400 MB | 400-800 MB | Depends on resolution |

### Memory Spikes

**Normal**:
- Startup: Temporary spike to 100MB then drops to baseline
- Source creation: Brief spike when creating audio/display capture

**Abnormal**:
- Steady climb: Possible memory leak
- Jump >200MB: Check OBS (recording at high resolution?)

**Recovery**:
```bash
# Force restart to reset memory
memofy-ctl restart
```

### Process Zombies

**What they are**: Process declared dead but not fully cleaned by OS

**Signs**:
```bash
# If you see "<defunct>" in process list:
ps aux | grep memofy

# Output like:
# 501 12345 0.0 0.0 0 0 ?? Z 3:15PM 0:00.00 memofy-core <defunct>
```

**Cleanup**:
```bash
# Force clean
memofy-ctl clean

# If still stuck, restart system
reboot
```

---

## Monitoring Process Health

### One-Command Diagnostic

```bash
memofy-ctl diagnose
```

Output shows:
- ✓/✗ Process status with memory
- ✓/✗ OBS connectivity (port 4455 reachable)
- ✓/✗ System info (macOS version, hostname)
- Last 5 errors from logs
- Troubleshooting suggestions

### Continuous Monitoring

```bash
# Watch logs for events
watch 'tail -20 /tmp/memofy-core.out.log'

# Watch status file updates
watch 'cat ~/.cache/memofy/status.json'

# Watch process memory
watch 'ps aux | grep memofy | grep -v grep'
```

### Health Checks

| Check | Success Sign | Failure Sign | Recovery |
|-------|--------------|--------------|----------|
| Process running | `memofy-ctl status` shows PID | "not running" | `memofy-ctl start` |
| OBS connected | `[RUNNING]` in logs | `[RECONNECT]` logs | Start OBS manually |
| Sources created | `[VERIFY]` All required found | `[ERROR]` code 204 | Wait for backoff, or restart |
| Detection active | `[EVENT]` logs appear | Only `[STARTUP]` | Check detection rules config |

---

## Troubleshooting Process Issues

### Process Hangs (No Output >30s)

```bash
# 1. Check if process exists
ps aux | grep memofy-core

# 2. Force stop if hung
memofy-ctl stop --force

# 3. Clean stale files
memofy-ctl clean

# 4. Restart
memofy-ctl start
```

### Process Crashes Immediately

```bash
# 1. Check why
memofy-ctl logs | tail -50 | grep ERROR

# 2. Common causes:
# - Permission denied: Check System Preferences > Security & Privacy
# - Config missing: Will auto-recreate on restart
# - OBS missing: Will auto-launch on restart

# 3. Restart
memofy-ctl start
```

### Process Leaking Memory

```bash
# 1. Monitor for 1 hour
watch -n 5 'ps aux | grep memofy | grep -v grep | awk "{print \$6}"'

# 2. If memory continuously grows:
# - Known issue: Possible bug in detection logic
# - Workaround: Restart daily

# 3. Report with logs:
memofy-ctl logs > ~/memofy-memleak.log
# Attach ~/memofy-memleak.log to issue report
```

---

## Performance Expectations

### Startup Performance

| Component | Expected Time | Slow System Time | Bottleneck |
|-----------|----------------|-----------------|-----------|
| memofy-core total | ~25 seconds | ~35 seconds | OBS launch (if not running) |
| OBS connect | ~5 seconds | ~10 seconds | Network/system load |
| Source validation | ~5 seconds | ~10 seconds | Scene size |
| memofy-ui total | ~5 seconds | ~10 seconds | AppKit initialization |

### Runtime Performance

- **CPU**: Minimal (<1% idle, <5% during recording)
- **Memory**: Stable 30-50 MB for memofy-core, 5-10 MB for memofy-ui
- **Disk I/O**: None except log writes (>100 KB/hour)
- **Network**: Minimal (<100 bytes/s to OBS WebSocket)

### Detection Latency

- Poll frequency: Every 2 seconds
- Start threshold: 3 consecutive detections (6 seconds to confirm)
- Stop threshold: 6 consecutive non-detections (12 seconds to confirm)
- **Total**: Meeting start detected within ~6-8 seconds, stop within ~12-15 seconds
