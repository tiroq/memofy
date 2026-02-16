# Memofy Startup Sequence

## Overview

This document describes the detailed startup process for Memofy services. Understanding this helps diagnose issues when processes don't start properly.

---

## Timeline: What Should Happen

### Pre-Startup Checks (0-1 seconds)

When you run `memofy-ctl start` or `task release-local`, the system should:

1. **Check existing processes**
   - Verify memofy-core is not already running (check PID file)
   - Verify memofy-ui is not already running (check PID file)
   - If stale PID files exist, clean them and show warning

2. **Check binaries**
   - Ensure `~/.local/bin/memofy-core` exists
   - Ensure `~/.local/bin/memofy-ui` exists
   - Print error if binaries not found

---

## memofy-core Startup Sequence

### Phase 1: Initialization (0-2 seconds)

**Log Output**:
```
=========================================== 
Starting Memofy Core v0.1.1...
PID: 12345
Timestamp: 2026-02-14T15:30:00+07:00
=========================================== 
```

**What's happening**:
- Process creates PID file at `~/.cache/memofy/memofy-core.pid`
- Panic recovery is enabled
- Logging is initialized to `/tmp/memofy-core.out.log` and `/tmp/memofy-core.err.log`

**Check**: No output after 2 seconds = process may have panicked

---

### Phase 2: Permissions Check (2-3 seconds)

**Log Output**:
```
[STARTUP] Checking macOS permissions...
[STARTUP] Permissions check passed
```

**What's happening**:
- Verifies `Screen Recording` permission is granted
- Verifies `Accessibility` permission is granted
- If permissions denied, exits with code 1

**What to do if it fails**:
```bash
# Grant permissions manually:
# System Preferences > Security & Privacy > Screen Recording > Add memofy-core
# System Preferences > Security & Privacy > Accessibility > Add memofy-core
```

---

### Phase 3: Configuration Load (3-4 seconds)

**Log Output**:
```
[STARTUP] Loading detection configuration...
[STARTUP] Loaded detection config: 3 rules, poll_interval=2s, thresholds=3/6
```

**What's happening**:
- Reads `~/.config/memofy/detection-rules.json`
- Parses meeting detection rules (Zoom, Teams, Google Meet, etc.)
- Sets up polling intervals and start/stop thresholds

**What to do if it fails**:
```bash
# Check config file exists:
ls -la ~/.config/memofy/detection-rules.json

# If missing, reinstall will create default:
curl -fsSL https://raw.githubusercontent.com/tiroq/memofy/main/scripts/quick-install.sh | bash
```

---

### Phase 4: OBS Startup Check (4-5 seconds)

**Log Output**:
```
[STARTUP] Checking OBS status...
[STARTUP] Checking OBS status... (if OBS not running, tries to start it)
```

**What's happening**:
- Checks if OBS is already running
- If not running, attempts to launch it with `open -a OBS`
- Waits max 5 seconds for OBS to launch

**Expected**:
- If OBS already running: completes within 1s
- If OBS needs launch: may take 10-30 seconds (OBS is slow)

---

### Phase 5: OBS WebSocket Connection (5-15 seconds)

**Log Output**:
```
[STARTUP] Connecting to OBS WebSocket at ws://localhost:4455...
[STARTUP] Successfully connected to OBS, setting up deferred cleanup...
[STARTUP] Connected to OBS 29.1.3 (WebSocket 5.0.5)
```

**What's happening**:
- Connects to OBS WebSocket on localhost:4455
- Performs 4-way handshake: WebSocket negotiate → Hello → Identify → Identified
- Retrieves OBS and plugin version information
- Sets up auto-reconnection handler (exponential backoff 5s→60s)

**Failure modes**:
- **"Failed to connect to OBS"**: OBS not running or WebSocket disabled
  - Check: `memofy-ctl diagnose`
  - Fix: OBS > Tools > obs-websocket Settings > Enable WebSocket server

- **Timeout after 10s**: OBS not responding
  - May be frozen or very busy
  - Fix: Force restart OBS: `killall OBS; sleep 2; open -a OBS`

---

### Phase 6: Compatibility Check (15-16 seconds)

**Log Output**:
```
[STARTUP] Validating OBS compatibility...
[STARTUP] OBS Health: OBS 29.1 is compatible (requires 28.0+) | WebSocket v5.0.5 is compatible
```

**What's happening**:
- Validates OBS version is 28.0+ (WebSocket v5 requirement)
- Validates WebSocket version is 5.x
- Checks for known incompatibilities

**Failure modes**:
- **"OBS {version} is too old"**: Version < 28.0
  - Fix: Update OBS from https://obsproject.com
  
- **"WebSocket vX.X is incompatible"**: Not v5.x
  - Fix: Update obs-websocket plugin, or update OBS (includes plugin)

---

### Phase 7: Source Validation & Creation (16-20 seconds)

**Log Output**:
```
[STARTUP] Checking OBS recording sources...
[SOURCES] Active scene: 'Collection 1', existing sources: 2
[SOURCE_FOUND] Desktop Audio (type: coreaudio_input_capture, enabled: true)
[SOURCE_FOUND] Display Capture (type: macos_screen_capture, enabled: true)
[SOURCE_CHECK] Audio source: Desktop Audio
[SOURCE_CHECK] Display source: Display Capture
[VERIFY] All required sources are present and enabled
[STARTUP] OBS recording sources validated (audio + display capture ready)
```

**What's happening**:
- Gets active OBS scene name
- Lists existing sources in scene
- Checks for required source types: audio capture + display capture
- Creates missing sources with platform-specific types:
  - macOS: `coreaudio_input_capture` (audio), `macos_screen_capture` (display)
  - Windows: `wasapi_input_capture` (audio), `monitor_capture` (display)  
  - Linux: `pulse_input_capture` (audio), `xshm_input` (display)

**Failure modes**:
- **"Black/silent recordings" warning**: Sources don't exist or are disabled
  - Cause: Scene is empty, or source creation failed
  - Fix: See troubleshooting guide for "Sources Not Ensuring"

- **Code 204 error during creation**: Request type not recognized
  - Cause: OBS version doesn't support the request
  - Fix: Update OBS to 28.0+, ensure plugin is enabled

- **"Audio source exists but is disabled"**: Source exists but `Enabled: false`
  - Fix: In OBS, right-click source > Filter > Toggle Enable
  - Or: Delete and let Memofy recreate it

---

### Phase 8: Event Handlers Registration (20-21 seconds)

**Log Output**:
```
[STARTUP] Setting up OBS event handlers...
[STARTUP] Event handlers registered
```

**What's happening**:
- Registers callback for "OBS recording state changed"
- Registers callback for "OBS disconnected" (triggers auto-reconnect)
- Sets up internal state machine

**Expected**: No visible output unless events occur

---

### Phase 9: Status Directory & Initial Status (21-22 seconds)

**Log Output**:
```
[STARTUP] Creating status directory...
[STARTUP] Writing initial status...
```

**What's happening**:
- Creates `~/.cache/memofy/` directory if missing
- Writes initial status file at `~/.cache/memofy/status.json`
- Status shows: recording state, OBS connection, etc.

**Status file contents**:
```json
{
  "recording": false,
  "start_time": "2026-02-14T15:30:00+07:00",
  "duration_seconds": 0,
  "obs_status": "connected",
  "obs_version": "29.1.3",
  "last_updated": "2026-02-14T15:30:05+07:00"
}
```

---

### Phase 10: Command Watcher Start (22-23 seconds)

**Log Output**:
```
[STARTUP] Starting command file watcher...
```

**What's happening**:
- Starts background goroutine to watch `~/.cache/memofy/cmd.txt`
- Allows CLI to send commands: `start`, `stop`, `auto`, `pause`
- Example: `echo 'start' > ~/.cache/memofy/cmd.txt` to force recording

**Monitoring**: 
```bash
# Watch command processing:
tail -f /tmp/memofy-core.out.log | grep COMMAND
```

---

### Phase 11: State Machine Initialization (23-24 seconds)

**Log Output**:
```
[STARTUP] Initializing state machine...
[STARTUP] State machine initialized in auto mode
```

**What's happening**:
- Creates meeting detection state machine
- Sets to `auto` mode (auto-detects meetings and records)
- Other modes: `manual` (user controls), `disabled` (off)

**Current state**:
```bash
# Check current mode:
memofy-ctl logs | grep "initialized in"

# Valid modes: auto, manual, disabled
```

---

### Phase 12: Main Detection Loop (24-25 seconds+)

**Log Output**:
```
[STARTUP] Starting detection loop (polling every 2s)...
=========================================== 
[RUNNING] Memofy Core is running and monitoring
=========================================== 
```

**What's happening**:
- Starts main event loop
- Every 2 seconds: polls for active application
- Checks if app matches meeting detection rules
- Tracks meeting state (started, stopped, running)
- Auto-starts/stops OBS recording as needed

**Ongoing monitoring**:
```bash
# Watch detection events:
memofy-ctl logs | grep EVENT

# Example output when meeting detected:
# [EVENT] Meeting detected: Zoom (start_count: 1/3)
# [EVENT] Threshold 3 reached - starting recording
# [EVENT] OBS recording state changed: STARTED
```

---

## Total Startup Time

**Expected Timeline**:
- 0-5s: Initialization, permissions, config
- 5-15s: OBS connection & version check
- 15-22s: Source validation & setup
- 22-25s: State machine, ready

**Total: ~25 seconds** from start to "Memofy Core is running"

---

## memofy-ui Startup Sequence

### Complete Timeline (0-10 seconds)

**Log Output**:
```
=========================================== 
Memofy UI starting (version v0.1.1)...
PID: 67890
Timestamp: 2026-02-14T15:30:25+07:00
=========================================== 
[STARTUP] PID file created: ...  
[STARTUP] Initializing macOS application...
[STARTUP] Creating SharedApplication...
[STARTUP] ...UI initialization in progress...
[STARTUP] macOS Application initialized
[STARTUP] Creating status bar app...
[STARTUP] Status bar app created successfully
[STARTUP] UI initialization completed
[STARTUP] Loading initial status...
[STARTUP] Starting status file watcher...
=========================================== 
[RUNNING] Memofy UI is running
=========================================== 
```

**What's happening**:
1. Initializes PID file
2. Creates macOS Application (SharedApplication on main thread)
3. Creates status bar AppKit UI component
4. Loads current meeting/recording status
5. Starts watching status file for updates
6. Displays in menu bar ✓

**Watchdog Timer**: 15-second timeout (increased from 5s for slow Macs)
- If UI init takes >15s, process exits with error
- Added heartbeat logging every 2s during init to show progress

---

## Monitoring Running Services

### Check Status

```bash
# Quick status:
memofy-ctl status

# Shows:
# ✓ memofy-core is running (PID xxxx)
# ✓ memofy-ui is running but no PID file found

# Detailed with diagnostics:
memofy-ctl diagnose
```

### Watch Logs in Real-Time

```bash
# All output:
tail -f /tmp/memofy-core.out.log

# Just events:
tail -f /tmp/memofy-core.out.log | grep EVENT

# Just errors:
tail -f /tmp/memofy-core.err.log

# Core startup:
head -50 /tmp/memofy-core.out.log | grep STARTUP
```

### Verify Meeting Detection

```bash
# When you start a Zoom/Teams/Meet call:
tail -f /tmp/memofy-core.out.log | grep -E "detected|recording"

# Expected flow:
# [EVENT] Meeting detected: Zoom
# [EVENT] Threshold 3 reached - starting recording  
# [EVENT] OBS recording state changed: STARTED
# (recording happens)
# [EVENT] Meeting ended
# [EVENT] Threshold 6 reached - stopping recording
# [EVENT] OBS recording state changed: STOPPED
```

---

## Troubleshooting Specific Stages

| Stage | Problem | Timeout | Check |
|-------|---------|---------|-------|
| Phase 5: OBS Connection | Won't connect | 10-15s | Port 4455 reachable? OBS WebSocket enabled? |  
| Phase 6: Compatibility | Version error | Immediate | OBS >= 28.0? WebSocket v5.x? Run `memofy-ctl diagnose` |
| Phase 7: Sources | Create fails | 10-20s | Code 204? Scene locked? Try manual creation |
| Phase 11: Detection | No events | 5 min of silence | Check detection rules in `~/.config/memofy/` |
| UI Initialization | Timeout error | 15s | Permission denied? macOS issue? Check logs |

---

## Recovery from Startup Failures

### If Processes Don't Start

```bash
# 1. Check diagnostics
memofy-ctl diagnose

# 2. Verify OBS
# OBS > Tools > obs-websocket Settings > Enable WebSocket server

# 3. Check logs for [ERROR] tags
memofy-ctl logs | grep ERROR

# 4. Clean restart
memofy-ctl clean
sleep 1
memofy-ctl start

# 5. Monitor startup
tail -f /tmp/memofy-core.out.log
```

### If Processes Hang

```bash
# Check for timeouts
memofy-ctl logs | grep -i timeout

# Force stop
memofy-ctl clean

# If memofy-ui hangs during init (NSWindow error):
# Known macOS AppKit issue - requires:
# 1. System Preferences > Security & Privacy > Screen Recording > Allow
# 2. System Preferences > Security & Privacy > Accessibility > Allow
# 3. Restart: killall memofy-ui; memofy-ctl start ui
```

### If Code 204 Errors

```bash
# Check OBS version
memofy-ctl diagnose | grep "OBS Health"

# Update OBS if needed
# Download from https://obsproject.com

# Verify WebSocket plugin
# OBS > Tools > obs-websocket Settings > Should show Server port 4455

# Restart with fresh connection
memofy-ctl restart
```
