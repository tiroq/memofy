# Memofy Troubleshooting Guide

## Overview

This guide helps you diagnose and fix common Memofy issues. Most problems fall into three categories:
1. **OBS Connection Issues** - Memofy can't reach OBS
2. **Source Configuration Issues** - Missing or disabled audio/display capture
3. **Process Issues** - Services crash, hang, or won't start

---

## Quick Diagnostics

Run this first to see system status:

```bash
memofy-ctl diagnose
```

This shows:
- Running processes and memory usage
- OBS WebSocket connectivity
- System information  
- Recent error logs
- Binary availability

---

## Error Code 204: "InvalidRequest"

### Symptoms

```
ERROR: request failed: OBS rejected request type 'GetCurrentScene' (code 204: InvalidRequest)
```

### Root Cause

Code 204 means OBS doesn't recognize the request. This happens when:
- OBS version is < 28.0 (doesn't support WebSocket v5)
- obs-websocket plugin is disabled or not installed  
- WebSocket server is not enabled

### Solution

1. **Check OBS Version**:
   ```bash
   # In OBS: Help > About OBS Studio
   # Shows version (need 28.0 or later)
   ```

2. **Enable WebSocket Server**:
   ```
   OBS > Tools > obs-websocket Settings > Enable WebSocket server
   Check: Port is 4455
   Click: Apply
   ```

3. **Update OBS if Needed**:
   ```bash
   # Download from https://obsproject.com
   # Requires OBS 28.0+ with obs-websocket v5.0+
   ```

4. **Verify Connection**:
   ```bash
   memofy-ctl logs
   # Should see: [STARTUP] Connected to OBS {version} (WebSocket {version})
   ```

**Recovery**: After fixing, restart Memofy:
```bash
memofy-ctl restart
```

---

## Sources Not Ensuring / Missing Audio/Display

### Symptoms

```
Warning: Could not ensure sources:
  This may cause black/silent recordings
  Please manually add Display Capture and Audio Input sources to your scene
```

Or in logs:
```
[SOURCE_FOUND] Audio source exists but is disabled
[ERROR] Failed to create display source: request failed...
```

### Root Causes

1. **Scene is Empty** - Active scene has no sources
2. **Sources Disabled** - Audio/Display exist but are disabled
3. **OBS Version/Plugin Issue** - Can't create sources (code 204)
4. **Scene Read-Only** - Locked or special scene

### Solutions

#### Option 1: Restart and Let Memofy Create Sources (Preferred)

```bash
# Ensure OBS is running and WebSocket enabled
# Ensure you're in a default scene (not group/studio mode)
memofy-ctl restart core

# Wait 5-10 seconds and check logs
memofy-ctl logs
# Should see: [CREATE] Creating audio source...
#            [SUCCESS] Audio source created
```

#### Option 2: Manually Add Sources in OBS

If automatic creation fails:

1. **Add Audio Source**:
   - OBS > Scenes > [Active Scene] > Sources panel  
   - Click `+` > Input > "Desktop Audio" (or system audio)
   - Select "Default Input Device"
   - Click "Create New" > "OK"

2. **Add Display Source**:
   - Click `+` > Input > "Display Capture" (or "Screen Capture")
   - Select "Primary Display"  
   - Click "Create New" > "OK"

3. **Verify Both Are Enabled**:
   - Both sources should have a checkmark ✓
   - If disabled (no checkmark), click to enable

4. **Restart Memofy**:
   ```bash
   memofy-ctl restart
   ```

---

## Process Not Running / Killed Signal

### Symptoms

```
memofy-ctl status
✗ memofy-core is not running
✗ memofy-ui is not running
```

Or:
```
memofy-ctl start
ℹ Starting memofy-core...
✗ Failed to start memofy-core
```

### Root Causes

1. **Timeout During Startup** - Process takes too long and is killed
2. **Crash on Init** - Process panics during startup
3. **OBS Connection Fails** - Can't reach OBS, immediate exit
4. **Permission Issues** - Missing access to resources

### Solutions

#### Check Logs First

```bash
memofy-ctl logs
# Watch for [STARTUP], [ERROR], [SHUTDOWN] tags
# Last lines show why process died
```

#### If memofy-core Won't Start

```bash
# 1. Verify OBS is running and reachable
memofy-ctl diagnose  # Check "OBS WebSocket: ✓ accessible"

# 2. Check for OBS connection issues
memofy-ctl logs
# Look for: "Failed to connect to OBS"
# Solution: OBS > Tools > obs-websocket Settings > Enable WebSocket

# 3. Look for code 204 errors (OBS version mismatch)
# Solution: See "Error Code 204" section above

# 4. If logs show timeout during startup
# This means OBS is responding slowly - give it time
# Increase polling interval or restart OBS:
killall OBS 2>/dev/null
sleep 2
open -a OBS

# 5. Try starting in debug mode to see more logs
/Users/yourname/.local/bin/memofy-core 2>&1 | grep -E "\[STARTUP\]|\[ERROR\]"
```

#### If memofy-ui Won't Start  

```bash
# UI typically crashes due to macOS threading issues
# Logs should show NSWindow error or "UIinit timeout"

# Try:
memofy-ctl stop ui
sleep 2
memofy-ctl start ui

# If still fails, check for darwinkit/AppKit issues:
tail -50 /tmp/memofy-ui.err.log
# Look for "NSWindow", "main thread", "Exception"

# Usually a system permission issue:
# System Preferences > Security & Privacy > 
#  - Screen Recording (add memofy-ui)
#  - Accessibility (add memofy-ui)
```

#### Handle Stale PID Files

```bash
# If process crashed, PID file may be stale
# Clean command does this automatically:
memofy-ctl clean

# Or manually:
rm ~/.cache/memofy/memofy-core.pid
rm ~/.cache/memofy/memofy-ui.pid

# Then restart:
memofy-ctl start
```

---

## OBS Disconnects / Reconnecting

### Symptoms

```
[EVENT] OBS disconnected - will attempt reconnection
[RECONNECT] Attempt 1, delay 5s, next=10s
[RECONNECT] Attempt 2, delay 10s, next=20s
```

### Root Causes

1. **OBS Crashed** - Restarting OBS drops connection
2. **Network Issue** - Temporary connectivity problem
3. **Port Blocked** - 4455 became unavailable
4. **OBS WebSocket Timeout** - Server stopped responding

### Solutions

```bash
# Memofy auto-reconnects - just wait
# Initial delay is 5s, then 10s, 20s, 40s, 60s max

# Check current status:
memofy-ctl status

# If stuck reconnecting for >2 minutes:
# 1. Restart OBS
killall OBS 2>/dev/null
sleep 2
open -a OBS

# 2. Restart Memofy core  
memofy-ctl restart core

# 3. Check WebSocket is enabled
# OBS > Tools > obs-websocket Settings > Enable WebSocket server

# Verify reconnection:
tail -f /tmp/memofy-core.out.log | grep RECONNECT
# Should count down: attempt 1 (5s), 2 (10s), etc.
```

---

## Port 4455 Already in Use

### Symptoms

```
Failed to connect to OBS: dial tcp [::1]:4455: connect: connection refused
```

Or in OBS when enabling WebSocket:
```
Error: Address already in use  
```

### Solution

```bash
# Find what's using port 4455
lsof -i :4455

# Kill it (if it's not OBS)
kill -9 <PID>

# Or force OBS to use a different port:
# OBS > Tools > obs-websocket Settings > Server port: 4456
# Then tell Memofy:
# (This requires code change - for now, restart everything)

# Clean restart:
memofy-ctl clean
killall OBS 2>/dev/null
sleep 2
# Launch OBS, enable WebSocket port 4455
# Then start Memofy
memofy-ctl start
```

---

## Process Zombies / High Memory Usage

### Symptoms

```
ps aux | grep memofy-ui
# Shows processes with "Z" (zombie) or very high RSS (memory)

memofy-ui not responding
```

### Solution

```bash
# Force clean termination
memofy-ctl clean

# Remove any zombie processes
killall -9 memofy-ui 2>/dev/null
killall -9 memofy-core 2>/dev/null

# Remove stale files
rm -f ~/.cache/memofy/*.pid
rm -f ~/.cache/memofy/*.died

# Restart
memofy-ctl start

# Monitor memory:
# Activity Monitor > Memory tab > Sort by Memory (descending)
# memofy-ui should use <50MB
# memofy-core should use <100MB
```

---

## Common Log Messages Explained

| Log Message | Meaning | Action |
|------------|---------|--------|
| `[STARTUP]` | Service is starting up | Wait for `[RUNNING]` |
| `[EVENT]` | Something happened (recording started/stopped) | Normal, no action needed |
| `[RECONNECT]` | OBS disconnected, retrying | Wait, or check OBS |
| `[SOURCE_CHECK]` | Checking for audio/display sources | Normal part of startup |
| `[CREATE]` | Creating missing source | Normal if source didn't exist |  
| `[ERROR]` | Something failed | Check following messages for details |
| `[SHUTDOWN]` | Service is stopping | Normal during stop/restart |

---

## Still Not Working?

### Collect Full Diagnostics

```bash
# Run comprehensive check
memofy-ctl diagnose

# Capture all logs
mkdir -p ~/memofy-logs
cp /tmp/memofy-core.* ~/memofy-logs/
cp /tmp/memofy-ui.* ~/memofy-logs/
cat ~/.cache/memofy/*.died >> ~/memofy-logs/deaths.log

# Check system logs
system_profiler SPSoftwareDataType > ~/memofy-logs/system.txt
```

### Reset and Reinstall

```bash
# Complete clean
memofy-ctl clean
killall -9 memofy-core memofy-ui 2>/dev/null
rm -rf ~/.cache/memofy
rm -rf ~/.config/memofy

# Reinstall
curl -fsSL https://raw.githubusercontent.com/tiroq/memofy/main/scripts/quick-install.sh | bash

# Start fresh
memofy-ctl status
```

### Contact Support

Include in your support request:
1. `memofy-ctl diagnose` output
2. All logs from `~/memofy-logs/`
3. OBS version and settings screenshot
4. macOS version
5. What were you trying to do when it failed
