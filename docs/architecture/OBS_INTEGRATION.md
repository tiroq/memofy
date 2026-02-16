# OBS Integration Guide

## Overview

Memofy integrates with OBS (Open Broadcaster Software) via the WebSocket protocol. This guide explains the integration, how it works, and how to troubleshoot issues.

---

## System Architecture

### Components

```
┌──────────────────────────────────────────┐
│ memofy-core (Darwin/macOS service)       │
│                                          │
│  - Detects active meetings (Zoom, Teams) │
│  - Controls OBS recording via WebSocket  │
│  - Manages audio/display sources         │
└─────────────────┬────────────────────────┘
                  │
        WebSocket Connection
        (JSON-RPC over WS)
                  │
                  ↓
┌──────────────────────────────────────────┐
│ OBS (Open Broadcaster Software v28.0+)  │
│                                          │
│  - Captures display (screen recording)   │
│  - Captures system audio                 │
│  - Encodes video (H.264, VP8, etc.)      │
│  - Writes .mkv/.mp4 files                │
│                                          │
│  obs-websocket plugin (v5.x required)    │
│  - Exposes WebSocket server on port 4455 │
│  - Handles method calls (start/stop)     │
│  - Sends event notifications             │
└──────────────────────────────────────────┘
```

### Protocol Details

**WebSocket Connection**
- Protocol: `ws://localhost:4455`
- Format: JSON-RPC 2.0 over WebSocket
- Requires password authentication (configurable in OBS)
- Default password: Empty (no auth required)

**Connection Sequence**
1. Client connects to `ws://localhost:4455`
2. Server sends `Hello` frame (server info)
3. Client sends `Identify` frame (client info, auth)
4. Server acknowledges with `Identified` frame
5. Connection established, bidirectional messaging begins

---

## Prerequisites

### OBS Installation

**Minimum version**: OBS 28.0
- Older versions lack WebSocket v5 protocol
- memofy requires v5 for compatibility

**Installation**:
```bash
# macOS
brew install obs
# OR download from https://obsproject.com/download

# Verify installation
/Applications/OBS.app/Contents/MacOS/OBS --version
# Output should be: 28.0.0 or higher
```

### obs-websocket Plugin

**Status**: OBS 28.0+ includes obs-websocket built-in (no separate install needed)

**Verification**:
```bash
# Verify plugin is loaded:
# 1. OBS > Tools menu shows "obs-websocket Settings"
# 2. Check plugin version: OBS > Help > Check for Updates
```

**Enable Server**:
```
OBS > Tools > obs-websocket Settings
→ Server Control
  ✓ Enable WebSocket server (must be checked)
  ✓ Port: 4455 (default)
  ✓ Use authentication (optional, default: off)
```

---

## Initial Setup

### Create Recording Profile

**Step 1: Create Output Profile**
```
OBS > File > Settings > Output
Format: mkv (or mp4)
Encoder: Hardware-accelerated if available (VideoToolbox on macOS)
Video Bitrate: 8000-10000 kbps
Audio Bitrate: 128 kbps
Container: MKV (preserves stream if capture interrupted)
```

**Step 2: Configure Scenes**

OBS requires at least one scene for recording:
```
Scenes Panel (left sidebar):
  + Click "+" to add scene "Default" or "Collection 1"
  + Right-click → Duplicate to create backup
```

**Step 3: Add Sources**

For memofy to record correctly, scene must contain:

1. **Display Capture (Screen Recording)**
   ```
   Scenes > Your Scene > Sources
   + (add source) > Screen Capture > Create new
   → Screen: [Your Primary Display]
   → Capture Cursor: Yes (optional)
   ```

2. **Audio Input (System Audio)**
   ```
   Scenes > Your Scene > Sources  
   + (add source) > Audio Input Capture > Create new
   → Device: [System Audio - Built-in Output]
   ```

If sources don't already exist, memofy will create them automatically on startup.

### Quick Setup Script

Instead of manual setup, memofy provides a helper:

```bash
# FUTURE: One-command setup (not yet implemented)
# memofy-ctl setup-obs

# For now, use manual steps above or let memofy create sources automatically
```

---

## WebSocket Protocol Details

### Authentication

**Default (No Auth)**
```json
{
  "op": 1,
  "d": {
    "rpcVersion": "1.0",
    "authentication": null,
    "eventSubscriptions": 33
  }
}
```

**With Password**
```json
{
  "op": 1,
  "d": {
    "rpcVersion": "1.0",
    "authentication": "BASE64(SHA256(password + salt))",
    "eventSubscriptions": 33
  }
}
```

For memofy, password is empty by default (no auth needed).

### Method Calls (Request/Response)

**Request Format**
```json
{
  "op": 6,
  "d": {
    "requestType": "StartRecord",
    "requestId": "request-id-12345",
    "requestData": {}
  }
}
```

**Response Format**
```json
{
  "op": 7,
  "d": {
    "requestid": "request-id-12345",
    "requestType": "StartRecord",
    "requestStatus": {
      "result": true,
      "code": 0
    },
    "responseData": {}
  }
}
```

**Error Response (Code 204)**
```json
{
  "op": 7,
  "d": {
    "requestid": "request-id-12345",
    "requestType": "CreateInput",
    "requestStatus": {
      "result": false,
      "code": 204
    },
    "responseData": {
      "comment": "The request type is not valid for the client's version"
    }
  }
}
```

### Event Subscriptions

memofy subscribes to:
- 0x01: General events (OBS_FRONTEND_EVENT_*)
- 0x20: Recording state changes

Bitwise OR: `0x01 | 0x20 = 0x21 = 33` (eventSubscriptions)

**Event Example: Recording Started**
```json
{
  "op": 8,
  "d": {
    "eventType": "RecordStateChanged",
    "eventIntent": 0,
    "eventData": {
      "outputActive": true
    }
  }
}
```

---

## Methods Used by Memofy

### Scene Management

**GetSceneList**
```
Purpose: Retrieve all scenes
Request: { "requestType": "GetSceneList" }
Response: { "scenes": [...], "currentProgramSceneName": "..." }
```

**GetSceneItemList**
```
Purpose: Get all sources (items) in a scene
Request: { 
  "requestType": "GetSceneItemList",
  "requestData": { "sceneName": "Collection 1" }
}
Response: { "sceneItems": [...] }
```

### Source Management

**CreateInput**
```
Purpose: Create a new source/input
Request: {
  "requestType": "CreateInput",
  "requestData": {
    "sceneName": "Collection 1",
    "inputName": "Desktop Audio",
    "inputKind": "coreaudio_input_capture",
    "inputSettings": { "device": "...UUID..." }
  }
}
Response: { "sceneItemId": 1 }
```

**GetInputList**
```
Purpose: List all available inputs
Request: { "requestType": "GetInputList", "requestData": {} }
Response: { "inputs": [...] }
```

**GetInputSettings**
```
Purpose: Get input properties (enabled, etc.)
Request: {
  "requestType": "GetInputSettings",
  "requestData": { "inputName": "Desktop Audio" }
}
Response: { "inputSettings": {...}, "inputKind": "..." }
```

**SetInputEnabled**
```
Purpose: Enable/disable a source
Request: {
  "requestType": "SetInputEnabled",
  "requestData": { "inputName": "Desktop Audio", "inputEnabled": true }
}
Response: {}
```

### Recording Control

**StartRecord**
```
Purpose: Start recording
Request: { "requestType": "StartRecord" }
Response: { "outputPath": "/path/to/recording.mkv" }
```

**StopRecord**
```
Purpose: Stop recording
Request: { "requestType": "StopRecord" }
Response: { "outputPath": "/path/to/recording.mkv" }
```

**GetRecordStatus**
```
Purpose: Check if recording is active
Request: { "requestType": "GetRecordStatus" }
Response: { 
  "outputActive": true,
  "outputPath": "/path/to/recording.mkv",
  "outputDuration": 123456 (milliseconds),
  "outputBytes": 987654321 (total bytes written)
}
```

---

## Source Types by Platform

Memofy automatically creates platform-specific source types:

### macOS

**Display Capture**
- Type: `macos_screen_capture`
- Requires: Screen Recording permission
- Captures: Active display at monitor resolution

**Audio Capture**
- Type: `coreaudio_input_capture`
- Requires: Microphone permission (or system audio loopback)
- Captures: System audio (speakers/headphones output)

### Windows

**Display Capture**
- Type: `monitor_capture` (or `dxgi_output_duplication`)
- Captures: Selected monitor

**Audio Capture**
- Type: `wasapi_input_capture`
- Captures: Selected audio device output (loopback)

### Linux

**Display Capture**
- Type: `xshm_input` (X11) or `pipewire_screen` (Wayland)
- Captures: X11 display or PipeWire output

**Audio Capture**
- Type: `pulse_input_capture` or `jack_input_capture`
- Captures: PulseAudio or JACK audio

---

## Error Codes

### HTTP Status Codes (WebSocket)

| Code | Meaning | Memofy Response |
|------|---------|-----------------|
| 0 | Success | Continue normally |
| 203 | Timeout | Mark OBS as "slow response", may retry |
| 204 | Invalid request type | Likely version mismatch, log error + suggest update |
| 205 | Missing request data | Likely malformed request (should not occur) |
| 500-600 | Server error | Log error, suggest OBS restart |

### Code 204: Invalid Request Type

**Root Causes**
1. OBS version < 28.0 (missing WebSocket v5)
2. Method name changed between versions
3. Incorrect parameter format for OBS version

**Symptoms**
```
[ERROR] OBS rejected request type 'CreateInput' (code 204: InvalidRequest)
[ERROR] This likely indicates an OBS version or plugin compatibility issue.
```

**Solutions**
1. Check OBS version
   ```bash
   /Applications/OBS.app/Contents/MacOS/OBS --version
   # If < 28.0 → Update OBS
   ```

2. Verify WebSocket server enabled
   ```
   OBS > Tools > obs-websocket Settings > Server Control > ✓ Enable
   ```

3. Restart OBS
   ```bash
   killall OBS; sleep 1; open -a OBS
   memofy-ctl restart
   ```

---

## Connection Troubleshooting

### Port Not Reachable

**Check if port 4455 is open**
```bash
# Test connectivity
nc -zv localhost 4455
# Expected: Connection succeeded / connected to 4455

# If failed: Check what's using port 4455
lsof -i :4455
# If nothing: OBS not running
# If other process: Change OBS port in obs-websocket settings
```

**Solutions**:
1. Ensure OBS is running: `ps aux | grep OBS`
2. Ensure WebSocket server enabled: OBS > Tools > obs-websocket Settings > ✓ Enable
3. Try different port: OBS > obs-websocket Settings > Change Port → 4456 (restart OBS)

### Connection Timeout

**What happens**:
```
[STARTUP] Connecting to OBS WebSocket...
[ERROR] Failed to connect to OBS after 10s timeout
```

**Causes**:
1. OBS not responding (frozen, high CPU)
2. High network latency (unlikely on localhost)
3. Port blocked by firewall

**Solutions**:
```bash
# 1. Force-restart OBS
killall -9 OBS
sleep 2
open -a OBS

# 2. Monitor CPU - if high, close other apps
top -o %CPU

# 3. Check firewall
System Preferences > Security & Privacy > Firewall Options
Add /Applications/OBS.app to allowed list
```

### Authentication Failure

**Symptoms**:
```
[ERROR] WebSocket authentication failed
[ERROR] Check OBS password setting
```

**Causes**:
1. OBS has password enabled but memofy not configured with it
2. Wrong password in memofy config

**Solutions**:
```
OBS > Tools > obs-websocket Settings > Server Control
→ Disable "Use authentication" (unless you need it)
→ Or set password to blank

If you need authentication:
→ Copy password from OBS
→ Set MEMOFY_OBS_PASSWORD environment variable
→ export MEMOFY_OBS_PASSWORD="your-password"
→ memofy-ctl restart
```

---

## Scene Configuration

### Recommended Setup

**Scene 1: "Meeting Recording"**
```
Sources:
  1. Desktop Audio (coreaudio_input_capture)
  2. Display Capture (macos_screen_capture)
  3. Mic Input (optional, if recording voiceover)

Output:
  Format: MKV (preserves even if interrupted)
  Bitrate: 8000 kbps video, 128 kbps audio
  Resolution: 1920x1080 or 3840x2160 (if 4K)
  FPS: 30 or 60 (depends on system)
```

**Scene 2: "Backup Scene"** (optional)
- Duplicate of Scene 1 for fallback

### Verify Sources Are Enabled

Memofy checks source "enabled" state before recording:

```
Scenes > Your Scene > Sources
→ Click source → Properties
→ Verify checkbox ✓ Source is enabled

If greyed out or disabled:
→ Right-click > Enable Source
```

---

## Recording Output

### File Location

Default recording location (configurable in OBS):
```
OBS > Settings > Output > Recording > Recording Path
Default: ~/Videos/Memofy/
```

### File Format

Memofy uses MKV container by default:
```
Filename: meeting-2026-02-14-15-30-00.mkv
Container: Matroska Video (.mkv)
Video Codec: H.264 (hardware-accelerated)
Audio Codec: AAC or Opus
```

### Post-Processing

Records are NOT converted automatically. If you need MP4:
```bash
# Convert MKV to MP4 (requires ffmpeg)
brew install ffmpeg

ffmpeg -i meeting-2026-02-14-15-30-00.mkv \
  -c:v libx264 -preset medium \
  -c:a aac \
  meeting-2026-02-14-15-30-00.mp4
```

---

## Performance Tuning

### CPU Usage

**If memofy-core using >20% CPU**:
1. Reduce polling frequency (contact support, not user-configurable)
2. Reduce OBS resolution (FPS affects CPU most)
3. Close other desktop capture apps

**If memofy-ui using >5% CPU**:
1. Check for status file thrashing
2. Restart UI: `memofy-ctl restart ui`

### OBS CPU Usage

**If OBS using >50% CPU while recording**:
1. Reduce video resolution: OBS > Settings > Video
2. Reduce framerate: 60 FPS → 30 FPS
3. Use hardware encoder: 
   ```
   OBS > Settings > Output > Encoder
   Select: Hardware (VideoToolbox on macOS)
   ```

### Network (if using remote OBS)

Memofy is NOT designed for remote OBS (different machine). It requires localhost connection:
```
SUPPORTED: ws://localhost:4455
NOT SUPPORTED: ws://192.168.1.100:4455
```

If you need remote OBS, use OBS browser plugin instead.

---

## Monitoring OBS Health

### Via memofy-ctl

```bash
# Quick health check
memofy-ctl diagnose | grep OBS
# Shows: OBS Health check status, port reachability, version info
```

### Via OBS Interface

```
OBS > Tools > obs-websocket Settings
→ Server Port: Shows 4455
→ Server Connected: Shows if any client connected
```

### Via Logs

```bash
# Watch for OBS connection events
tail -f /tmp/memofy-core.out.log | grep -E "OBS|STARTUP"

# Expected output:
# [STARTUP] Connected to OBS 29.1.3 (WebSocket 5.0.5)
# [EVENT] OBS recording state changed: STARTED
# [RECONNECT] Reconnection successful
```

---

## Updating OBS

### Check Current Version

```bash
/Applications/OBS.app/Contents/MacOS/OBS --version
# Output: 29.1.3
```

### Update Process

```bash
# Method 1: Homebrew
brew upgrade obs

# Method 2: Manual download
# Download from https://obsproject.com/download
# Drag OBS.app to Applications folder
# Restart: killall OBS; open -a OBS

# After update, restart memofy
memofy-ctl restart
```

### Verify Update

```bash
# Check version again
/Applications/OBS.app/Contents/MacOS/OBS --version

# Verify WebSocket plugin version
# OBS > Tools > obs-websocket Settings > Should show v5.x

# Run diagnostics
memofy-ctl diagnose
```

---

## Advanced Configuration

### Custom OBS Password

```bash
# Set password in OBS
OBS > Tools > obs-websocket Settings > Set Server Password > "my-secure-password"

# Configure memofy to use it
export MEMOFY_OBS_PASSWORD="my-secure-password"

# Only needed once, then restart:
memofy-ctl restart

# Verify connection works:
memofy-ctl logs | grep -i "identified"
```

### Custom Recording Path

```bash
# OBS Settings > Output > Recording Path
# Set to: /Volumes/External/Meetings/
# Make sure external drive is mounted

# Verify memofy can write to it:
touch /Volumes/External/Meetings/test.txt && rm /Volumes/External/Meetings/test.txt
```

### Multiple Scenes

If you want different scenes for different meetings:

```bash
# Memofy uses active scene at time of start
# To switch before recording:
# OBS > Scenes > Click desired scene

# Or use WebSocket method directly:
# (Requires custom integration)
```

---

## Known Limitations

### macOS Specific

| Limitation | Impact | Workaround |
|------------|--------|-----------|
| No ALSA support | Can't capture hardware audio on some systems | Use built-in mic or create audio loopback |
| Permission dialog | First run prompts "Screen Recording access" | Grant permission in System Preferences |
| Audio loopback | System audio capture requires 3rd party driver | Use Blackhole or SoundFlower |

### OBS Specific

| Limitation | Impact | Workaround |
|------------|--------|-----------|
| One output at a time | Can't stream and record simultaneously | Use OBS virtual camera instead |
| Scene switching during record | Audio/video sync issues if scene changed | Keep same scene during entire recording |
| WebSocket v5 requirement | Old OBS versions won't work | Update OBS to 28.0+ |

---

## Support & Debugging

### Collect OBS Information

```bash
# Export OBS config
tar -czf ~/obs-config.tar.gz ~/.config/obs-studio/

# Collect memofy logs
memofy-ctl logs > ~/memofy-diag.log

# Get OBS version and plugins
/Applications/OBS.app/Contents/MacOS/OBS --version > ~/obs-version.txt

# Get system info
system_profiler SPSoftwareDataType > ~/system-info.txt

# Pack for support
tar -czf ~/memofy-support.tar.gz \
  ~/obs-config.tar.gz \
  ~/memofy-diag.log \
  ~/obs-version.txt \
  ~/system-info.txt

# Share ~/memofy-support.tar.gz with support team
```

### Common Issues

**Issue: "OBS Disconnected" message persists**

```bash
# 1. Check OBS is running
ps aux | grep OBS

# 2. Restart OBS
killall OBS; sleep 1; open -a OBS

# 3. Verify WebSocket enabled
# OBS > Tools > obs-websocket Settings > ✓ Enable Server

# 4. Test connectivity
nc -zv localhost 4455

# 5. Restart memofy
memofy-ctl restart
```

**Issue: Sources not created automatically**

```bash
# 1. Check logs
memofy-ctl logs | grep -i "source\|create"

# 2. Manual creation
# OBS > Scenes > Sources > + > Screen Capture / Audio Input Capture

# 3. Verify sources enabled
# Right-click source > Enable

# 4. Restart memofy
memofy-ctl restart
```

**Issue: Code 204 error on startup**

```bash
# 1. Update OBS
brew upgrade obs

# 2. Verify version >= 28.0
/Applications/OBS.app/Contents/MacOS/OBS --version

# 3. Restart
memofy-ctl restart
```
