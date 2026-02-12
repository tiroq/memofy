# OBS Auto-Initialization Features

## Overview

Memofy now includes intelligent OBS initialization that **automatically handles startup and source configuration**. This eliminates manual setup steps and ensures you're ready to record immediately.

## Features

### 1. **Auto-Start OBS** 
When `memofy-core` daemon starts, it checks if OBS is running:
- ✅ If OBS is running → Proceeds to connect
- ❌ If OBS is NOT running → Automatically launches OBS
- ⏱️ Waits 5 seconds for OBS to initialize
- 🔌 Then connects via WebSocket

**Platform Support**:
- **macOS**: `open -a OBS`
- **Windows**: `OBS.exe`
- **Linux**: `obs`

### 2. **Automatic Source Creation**
Once connected, Memofy validates required recording sources:

#### Audio Source
Automatically creates **system audio capture** if missing:
- macOS: `coreaudio_input_capture` (Mac System Audio)
- Windows: `wasapi_input_capture` (WASAPI Audio)
- Linux: `pulse_input_capture` (PulseAudio)

#### Display Source
Automatically creates **screen capture** if missing:
- macOS: `macos_screen_capture` (Screen/Window capture)
- Windows: `monitor_capture` (Monitor capture)
- Linux: `xshm_input` (X11 screen capture)

### 3. **Source Validation**
Before recording, Memofy verifies:
- ✅ Audio input is configured and enabled
- ✅ Display/window capture is configured and enabled
- ❌ If sources are missing → Creates them automatically
- ⚠️ If creation fails → Warns but continues (with manual recovery option)

## Startup Sequence

```
memofy-core starts
    ↓
Check macOS permissions (Screen Recording, Accessibility)
    ↓
Load detection configuration
    ↓
Check if OBS is running
    ├─ If NO → Auto-start OBS
    └─ If YES → Continue
    ↓
Connect to OBS WebSocket
    ├─ If connection fails → Exit with instructions
    └─ If successful → Continue
    ↓
Validate recording sources
    ├─ Check for audio input source
    │  └─ If missing → Create automatically
    ├─ Check for display capture source
    │  └─ If missing → Create automatically
    └─ Continue (warning if creation failed)
    ↓
Start meeting detection loop
```

## Log Output

When daemon starts with auto-initialization:

```
2026-02-12 14:30:00 Starting Memofy Core v0.1...
2026-02-12 14:30:00 Checking macOS permissions...
2026-02-12 14:30:00 Loaded detection config: 3 rules, poll_interval=2s, thresholds=3/6
2026-02-12 14:30:00 Checking OBS status...
2026-02-12 14:30:05 Connecting to OBS WebSocket...
2026-02-12 14:30:05 Connected to OBS 30.0.0 (WebSocket 5.0)
2026-02-12 14:30:05 Checking OBS recording sources...
2026-02-12 14:30:05 OBS recording sources validated (audio + display capture ready)
2026-02-12 14:30:05 State machine initialized in auto mode
2026-02-12 14:30:05 Detection polling started (interval: 2s)
```

## Error Scenarios & Recovery

### Scenario 1: OBS Won't Start
❌ **If auto-start fails:**
```
Failed to start OBS: exit status 1 (continuing anyway)
Failed to connect to OBS: connection refused
Please ensure OBS is running and WebSocket server is enabled
  1. Open OBS Studio
  2. Go to Tools > obs-websocket Settings
  3. Enable 'Enable WebSocket server'
  4. Set port to 4455 (default)
```

**Resolution**: Manually start OBS and enable WebSocket server

### Scenario 2: WebSocket Disabled
❌ **If WebSocket server isn't enabled in OBS:**
```
Failed to connect to OBS: websocket endpoint not available
Please ensure OBS is running and WebSocket server is enabled
  [Instructions provided...]
```

**Resolution**: 
1. Open OBS → Tools → obs-websocket Settings
2. Enable "Enable WebSocket server" checkbox
3. Use port 4455 (default)

### Scenario 3: Source Creation Fails
⚠️ **If automatic sources can't be created:**
```
Warning: Could not ensure sources: Permission denied creating 'Display Capture'
  This may cause black/silent recordings
  Please manually add Display Capture and Audio Input sources to your scene
```

**Resolution** (Manual):
1. Open OBS
2. Go to Sources panel
3. Click "+" icon
4. Add "Audio Input Capture" (for audio)
5. Add "Display Capture" (for video)

## Advanced Configuration

### Manual Source Names
If you have custom source names, Memofy will detect them on first run:

| Source Type | Auto-Created Name | Custom Names Detected |
|---|---|---|
| Audio | "Desktop Audio" | Any enabled audio source type |
| Display | "Display Capture" | Window/Screen/Game capture sources |

### Disabling Auto-Start
To disable automatic OBS launch in future, modify `cmd/memofy-core/main.go`:

```go
// Current behavior: Auto-starts OBS
if err := obsws.StartOBSIfNeeded(); err != nil {
    errLog.Printf("Failed to start OBS: %v (continuing anyway)", err)
}

// To disable: Simply remove or comment out these lines
```

## API Reference

### For Developers

#### `obsws.StartOBSIfNeeded() error`
Launches OBS on current platform if not already running.
- Returns `nil` if OBS starts successfully or was already running
- Returns error if launch fails (but daemon continues)

#### `obsws.Client.EnsureRequiredSources() error`
Validates and creates required sources in active scene:
- Gets current active scene
- Checks for audio input source (creates if missing)
- Checks for display capture source (creates if missing)
- Returns error if validation fails (but daemon continues)

#### `obsws.Client.GetMeetingRecordingSetup() (*RequiredSources, error)`
Checks current setup without creating sources:
```go
setup, err := obsClient.GetMeetingRecordingSetup()
if setup.HasAudioInput && setup.HasDisplayVideo {
  // Ready to record
}
```

#### `obsws.Client.ValidateRequiredSources(sceneName) (*RequiredSources, error)`
Checks if required sources exist without creating:
```go
result, err := obsClient.ValidateRequiredSources("Main")
// Returns:
// - HasAudioInput: bool
// - HasDisplayVideo: bool
// - AudioSourceName: string
// - VideoSourceName: string
```

## Performance Impact

- **OBS Auto-Start**: +5 seconds to daemon startup (one-time, if OBS not running)
- **Source Validation**: +0.2 seconds to daemon startup (checking existing sources)
- **Source Creation**: +1-2 seconds per source created (rare, only on first setup)
- **Recording Impact**: None (validation only happens during startup)

## Troubleshooting

### Sources Keep Getting Deleted
Some OBS configurations auto-load scenes. If sources get deleted:
1. Disable "Auto-start scene collection" in OBS
2. Manually configure and save your scene
3. Restart memofy-core

### Black Screen in Recording
Verify sources were created:
1. Open OBS
2. Check "Sources" panel has "Display Capture" enabled
3. Check "Mixer" panel has audio levels showing

If sources missing, restart daemon:
```bash
# Kill and restart daemon (daemon auto-creates sources)
launchctl stop com.memofy.core
launchctl start com.memofy.core

# Or manually
killall memofy-core
memofy-core
```

### CI/Testing Without GUI
For headless/CI environments, disable source creation:
```go
// In main.go, comment out:
// obsClient.EnsureRequiredSources()

// Then manually create scenes/sources via OBS Websocket API
```

## FAQ

**Q: Does this require manual OBS setup anymore?**
A: Not for basic recording! Audio + display capture sources are auto-created. Advanced setups (multiple scenes, custom sources) still need manual configuration.

**Q: What if I already have custom sources?**
A: Auto-creation detects existing sources and skips creation. Your custom setup is preserved.

**Q: Can I disable auto-start?**
A: Yes, comment out `obsws.StartOBSIfNeeded()` in main.go or remove it.

**Q: Does auto-start work on servers without display?**
A: No. Use manual OBS start or disable auto-start for headless setups.

**Q: What source types are supported?**
A: Game capture, window capture, display capture, screen capture, audio input, WASAPI, PulseAudio, CoreAudio.
