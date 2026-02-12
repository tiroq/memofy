# Quick API Reference: OBS Source Management

## Functions Available

### OBS Launch
```go
// Auto-start OBS if not running (5 second wait for init)
// Returns nil if OBS is running or starts successfully
// Returns error if launch fails (daemon continues anyway)
StartOBSIfNeeded() error

// Check if OBS process is currently active
// Returns true if OBS detected, false otherwise
isOBSRunning() bool
```

### Scene Management
```go
// Get the name of the currently active scene
// Returns: "Main", "Scene 2", etc.
GetActiveScene() (string, error)

// Get all sources in a specific scene
// Returns array of SourceInfo structs
GetSceneSources(sceneName string) ([]SourceInfo, error)
```

### Source Creation
```go
// Create a new source in a scene
// Example: CreateSource("Main", "Mic", "coreaudio_input_capture", settings)
CreateSource(sceneName, sourceName, sourceType string, settings interface{}) error

// Auto-create audio input if missing (platform-aware)
// macOS: coreaudio_input_capture
// Windows: wasapi_input_capture  
// Linux: pulse_input_capture
CheckAndCreateAudioSource(sceneName string) (string, error)

// Auto-create display capture if missing (platform-aware)
// macOS: macos_screen_capture
// Windows: monitor_capture
// Linux: xshm_input
CheckAndCreateDisplaySource(sceneName string) (string, error)
```

### Source Validation
```go
// Check if required sources exist (audio + display)
// Returns: &RequiredSources{
//   HasAudioInput: bool,
//   HasDisplayVideo: bool,
//   AudioSourceName: "Desktop Audio",
//   VideoSourceName: "Display Capture"
// }
ValidateRequiredSources(sceneName string) (*RequiredSources, error)

// Master function: validate all required sources, create if missing
// Automatically gets active scene and ensures both sources exist
// Continues on error (safe for daemon startup)
EnsureRequiredSources() error

// Get current recording setup status (read-only)
// Returns what sources currently exist
GetMeetingRecordingSetup() (*RequiredSources, error)
```

---

## Data Types

```go
// Source information
type SourceInfo struct {
    SourceName string  // "Desktop Audio", "Display Capture"
    SourceType string  // "coreaudio_input_capture", "macos_screen_capture"
    SourceKind string  // "input" or "scene"
    Enabled    bool    // true if source is enabled
}

// Required sources status
type RequiredSources struct {
    HasAudioInput   bool   // true if audio input exists and enabled
    HasDisplayVideo bool   // true if display capture exists and enabled
    AudioSourceName string // e.g., "Desktop Audio"
    VideoSourceName string // e.g., "Display Capture"
}
```

---

## Platform-Specific Source Types

### Audio Input
| Platform | Type | Description |
|----------|------|-------------|
| macOS | `coreaudio_input_capture` | System audio via CoreAudio |
| Windows | `wasapi_input_capture` | System audio via WASAPI |
| Linux | `pulse_input_capture` | System audio via PulseAudio |

### Display Capture
| Platform | Type | Description |
|----------|------|-------------|
| macOS | `macos_screen_capture` | Display/window capture |
| Windows | `monitor_capture` | Monitor capture |
| Linux | `xshm_input` | X11 screen capture |

### Other Supported Types (Auto-detected)
- `window_capture` - Window capture (cross-platform)
- `game_capture` - Game capture (if enabled)
- `monitor_capture` - Monitor capture (Windows)
- `av_audio_input` - Generic audio input
- `av_input_device_capture` - Camera/device capture
- `audio_line` - Audio line/mixer

---

## Typical Usage Pattern

```go
package main

import (
    "log"
    "github.com/tiroq/memofy/internal/obsws"
)

func main() {
    // Try to start OBS if needed
    if err := obsws.StartOBSIfNeeded(); err != nil {
        log.Printf("Warning: Could not auto-start OBS: %v", err)
    }

    // Connect to OBS
    client := obsws.NewClient("ws://localhost:4455", "")
    if err := client.Connect(); err != nil {
        log.Fatalf("Failed to connect: %v", err)
    }
    defer client.Disconnect()

    // Ensure all required sources exist (create if missing)
    if err := client.EnsureRequiredSources(); err != nil {
        log.Printf("Warning: Could not ensure sources: %v", err)
        // Continue anyway - sources might already exist
    }

    // Check current setup
    setup, err := client.GetMeetingRecordingSetup()
    if err != nil {
        log.Printf("Error checking setup: %v", err)
    } else {
        if setup.HasAudioInput {
            log.Printf("✓ Audio configured: %s", setup.AudioSourceName)
        }
        if setup.HasDisplayVideo {
            log.Printf("✓ Display configured: %s", setup.VideoSourceName)
        }
    }

    // Now ready to record
    client.StartRecord("meeting.mp4")
    // ... recording happens ...
    client.StopRecord()
}
```

---

## Daemon Integration (main.go)

```go
// In cmd/memofy-core/main.go around startup:

// Auto-start OBS if needed
if err := obsws.StartOBSIfNeeded(); err != nil {
    errLog.Printf("Failed to start OBS: %v (continuing anyway)", err)
}

// Connect to OBS WebSocket
obsClient := obsws.NewClient(obsWebSocketURL, obsPassword)
if err := obsClient.Connect(); err != nil {
    errLog.Printf("Failed to connect to OBS: %v", err)
    os.Exit(1)
}

// Validate and create required sources
if err := obsClient.EnsureRequiredSources(); err != nil {
    errLog.Printf("Warning: Could not ensure sources: %v", err)
}
```

---

## Error Handling

```go
// Graceful error handling pattern

// Auto-start (optional, might already be running)
obsws.StartOBSIfNeeded()  // Errors are non-fatal

// Connect (required, must succeed)
if err := client.Connect(); err != nil {
    return err  // Fatal error
}

// Ensure sources (recommended but non-fatal)
if err := client.EnsureRequiredSources(); err != nil {
    log.Printf("Note: %v", err)
    log.Println("Continuing anyway, sources might exist or user can create manually")
}
```

---

## Testing

```go
// Run tests
go test -v ./internal/obsws/sources_test.go

// Run benchmarks
go test -bench=. ./internal/obsws/

// Run specific test
go test -run TestRequiredSourcesDetection ./internal/obsws/
```

---

## Environment Variables (Future)

Currently not implemented, but could be added:
- `MEMOFY_DISABLE_OBS_AUTOSTART=1` - Skip OBS launch
- `MEMOFY_OBS_LOCATION=/custom/path/OBS` - Custom OBS path
- `MEMOFY_SOURCE_AUDIO_TYPE=pulse_input_capture` - Override audio type
- `MEMOFY_SOURCE_VIDEO_TYPE=monitor_capture` - Override video type

---

## Return Values

### Success Patterns
```go
// Sources already exist
setup, _ := client.ValidateRequiredSources("Main")
// Returns: &RequiredSources{
//   HasAudioInput: true,
//   HasDisplayVideo: true,
//   AudioSourceName: "Desktop Audio",
//   VideoSourceName: "Display Capture"
// }

// Sources don't exist but were created
err := client.EnsureRequiredSources()  // nil (success)

// OBS auto-start successful
err := obsws.StartOBSIfNeeded()  // nil (success or already running)
```

### Error Patterns
```go
// Sources don't exist and couldn't be created
err := client.EnsureRequiredSources()
// err: "failed to create source 'Display Capture': permission denied"

// OBS not installed/can't launch
err := obsws.StartOBSIfNeeded()
// err: "failed to start OBS: exit status 1"

// WebSocket permission denied
err := client.Connect()
// err: "failed to connect: connection refused"
```

---

## Performance Considerations

- `StartOBSIfNeeded()`: ~50ms (check) + 5000ms (wait) if not running
- `ValidateRequiredSources()`: ~100-200ms (API call)
- `CreateSource()`: ~500-1000ms per source (API call + OBS processing)
- `EnsureRequiredSources()`: ~1-2 seconds total (first run, if sources created)

All calls are synchronous and can be awaited before continuing.

---

## FAQ

**Q: What if OBS crashes after auto-start?**
A: Daemon will notice the next time it checks. The reconnect logic will handle it.

**Q: Can I use custom source names?**
A: Yes! The functions will detect them and use them instead of creating new ones.

**Q: Will it work with multiple scenes?**
A: Currently works with the active scene only. Multiple scenes would require additional logic.

**Q: What if user deletes sources manually?**
A: Restart the daemon and they'll be re-created automatically.

**Q: Can I disable auto-initialization?**
A: Yes, comment out `StartOBSIfNeeded()` and `EnsureRequiredSources()` calls in main.go.
