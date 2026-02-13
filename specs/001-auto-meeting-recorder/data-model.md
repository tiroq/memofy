# Data Model: Automatic Meeting Recorder

**Feature**: 001-auto-meeting-recorder  
**Date**: February 12, 2026  
**Purpose**: Define data structures and state management

## Core Entities

### 1. Status Snapshot

Represents the complete system state at a point in time. Written by daemon to `~/.cache/memofy/status.json`, read by UI.

```go
type StatusSnapshot struct {
    Mode              OperatingMode     `json:"mode"`               // Current operating mode
    DetectionState    DetectionState    `json:"detection_state"`    // Raw detection state
    RecordingState    RecordingState    `json:"recording_state"`    // Actual recording state
    TeamsDetected     bool              `json:"teams_detected"`     // Teams meeting active
    ZoomDetected      bool              `json:"zoom_detected"`      // Zoom meeting active
    StartStreak       int               `json:"start_streak"`       // Consecutive detections
    StopStreak        int               `json:"stop_streak"`        // Consecutive non-detections
    LastAction        string            `json:"last_action"`        // Last action taken
    LastError         string            `json:"last_error"`         // Last error message
    Timestamp         time.Time         `json:"timestamp"`          // Snapshot time
    OBSConnected      bool              `json:"obs_connected"`      // OBS connection status
}
```

**Validation Rules**:
- `Timestamp` must be updated on every state change
- `StartStreak` resets to 0 when detection becomes false
- `StopStreak` resets to 0 when detection becomes true
- `LastError` is empty string when no error
- Only one of `TeamsDetected`/`ZoomDetected` should be true at a time (first detected wins)

**State Transitions**:
- Detection → StartStreak increments → Recording starts when StartStreak >= 3
- No detection → StopStreak increments → Recording stops when StopStreak >= 6

---

### 2. Operating Mode

Represents user control mode for recording behavior.

```go
type OperatingMode string

const (
    ModeAuto   OperatingMode = "auto"     // Automatic detection-based recording
    ModeManual OperatingMode = "manual"   // User-controlled recording only
    ModePaused OperatingMode = "paused"   // All detection suspended
)
```

```go
type ModeTransition struct {
    FromMode    OperatingMode  `json:"from_mode"`
    ToMode      OperatingMode  `json:"to_mode"`
    Command     string         `json:"command"`        // Command that triggered transition
    Timestamp   time.Time      `json:"timestamp"`
}
```

**Validation Rules**:
- Valid transitions: Any mode → Any mode (no restrictions)
- In `ModeManual`: Detection runs but doesn't trigger actions
- In `ModePaused`: Detection doesn't run at all
- In `ModeAuto`: Detection controls recording

---

### 3. Detection State

Represents current meeting detection evaluation.

```go
type DetectionState struct {
    MeetingDetected    bool              `json:"meeting_detected"`    // Stable detection result
    DetectedApp        DetectedApp       `json:"detected_app"`        // Which app triggered
    RawDetections      RawDetection      `json:"raw_detections"`      // Individual signal checks
    ConfidenceLevel    ConfidenceLevel   `json:"confidence"`          // Detection confidence
    EvaluatedAt        time.Time         `json:"evaluated_at"`        // When evaluated
}

type DetectedApp string

const (
    AppNone  DetectedApp = ""
    AppZoom  DetectedApp = "zoom"
    AppTeams DetectedApp = "teams"
)

type RawDetection struct {
    ZoomProcessRunning  bool   `json:"zoom_process_running"`
    ZoomHostRunning     bool   `json:"zoom_host_running"`         // CptHost process
    ZoomWindowMatch     bool   `json:"zoom_window_match"`         // Window title hint match
    TeamsProcessRunning bool   `json:"teams_process_running"`
    TeamsWindowMatch    bool   `json:"teams_window_match"`        // Window title hint match
}

type ConfidenceLevel string

const (
    ConfidenceNone   ConfidenceLevel = "none"     // No meeting detected
    ConfidenceLow    ConfidenceLevel = "low"      // Process only
    ConfidenceMedium ConfidenceLevel = "medium"   // Process + window OR host
    ConfidenceHigh   ConfidenceLevel = "high"     // Process + window + host
)
```

**Validation Rules**:
- Zoom detection requires: `ZoomProcessRunning AND (ZoomHostRunning OR ZoomWindowMatch)`
- Teams detection requires: `TeamsProcessRunning AND TeamsWindowMatch`
- `DetectedApp` is `AppNone` when `MeetingDetected` is false
- Confidence:
  - High: Zoom with all three signals
  - Medium: Zoom with 2 signals, or Teams with both
  - Low: Process running but no confirmation
  - None: No detection

---

### 4. Recording State

Represents actual OBS recording status.

```go
type RecordingState struct {
    Recording       bool       `json:"recording"`           // Currently recording
    StartedAt       time.Time  `json:"started_at"`          // Recording start time
    Duration        int64      `json:"duration_seconds"`    // Current duration in seconds
    OutputPath      string     `json:"output_path"`         // Recording file path
    OBSStatus       string     `json:"obs_status"`          // Last OBS response
}
```

**Validation Rules**:
- `Recording` false → `StartedAt` is zero time, `Duration` is 0, `OutputPath` is empty
- `Recording` true → All fields must be populated
- `Duration` updated every status write when recording is true
- `OBSStatus` contains last confirmation from GetRecordStatus

**File Naming**:
- Format: `YYYY-MM-DD_HHMM_Application_Title.mp4`
- Example: `2026-02-12_1430_Zoom_Q1-Planning.mp4`
- Fallback when title unavailable: `2026-02-12_1430_Zoom_Meeting.mp4`

---

### 5. Detection Rule

Represents configurable meeting detection criteria.

```go
type DetectionRule struct {
    Application     string   `json:"application"`       // "zoom" or "teams"
    ProcessNames    []string `json:"process_names"`     // Process name patterns
    WindowHints     []string `json:"window_hints"`      // Window title substrings
    Enabled         bool     `json:"enabled"`           // Rule active
}

type DetectionConfig struct {
    Rules          []DetectionRule  `json:"rules"`
    PollInterval   int              `json:"poll_interval_seconds"`   // Detection polling interval
    StartThreshold int              `json:"start_threshold"`         // Consecutive detections to start
    StopThreshold  int              `json:"stop_threshold"`          // Consecutive non-detections to stop
}
```

**Default Configuration** (from spec clarifications):
```json
{
  "rules": [
    {
      "application": "zoom",
      "process_names": ["zoom.us", "CptHost"],
      "window_hints": ["Zoom Meeting", "Zoom Webinar"],
      "enabled": true
    },
    {
      "application": "teams",
      "process_names": ["Microsoft Teams"],
      "window_hints": ["Meeting", "Call", "Reunión", "Anruf", "会議"],
      "enabled": true
    }
  ],
  "poll_interval_seconds": 2,
  "start_threshold": 3,
  "stop_threshold": 6
}
```

**Validation Rules**:
- `PollInterval` must be >= 1 second, <= 10 seconds
- `StartThreshold` must be >= 1, <= 10
- `StopThreshold` must be >= `StartThreshold`
- `WindowHints` case-insensitive matching
- At least one rule must be enabled

---

### 6. Command

Represents user commands from UI to daemon.

```go
type Command string

const (
    CmdStart  Command = "start"   // Start recording immediately
    CmdStop   Command = "stop"    // Stop recording immediately
    CmdToggle Command = "toggle"  // Toggle recording state
    CmdAuto   Command = "auto"    // Switch to auto mode
    CmdPause  Command = "pause"   // Switch to paused mode
    CmdQuit   Command = "quit"    // Shutdown daemon
)
```

Written to `~/.cache/memofy/cmd.txt` as single line, cleared by daemon after reading.

**Validation Rules**:
- Invalid commands are logged and ignored
- Commands processed with <2 second latency
- File cleared immediately after processing to prevent re-execution

---

## State Machine

### Detection State Machine

```
[IDLE] --[3 consecutive detections]--> [WAIT] --[start recording]--> [RECORDING]
[RECORDING] --[6 consecutive non-detections]--> [STOPPING] --[stop recording]--> [IDLE]
```

**State Transitions**:

| Current State | Event | Next State | Action |
|--------------|-------|------------|--------|
| IDLE | Detection TRUE | Increment StartStreak | None |
| IDLE | StartStreak >= 3 | WAIT | Prepare to record |
| WAIT | OBS confirms ready | RECORDING | StartRecord |
| RECORDING | Detection FALSE | Increment StopStreak | None |
| RECORDING | StopStreak >= 6 | STOPPING | Prepare to stop |
| STOPPING | OBS confirms stopped | IDLE | Reset streaks |
| Any | Manual command | Override state | Execute command |

**Invariants**:
- Only one counter (StartStreak or StopStreak) increments at a time
- Counters reset when detection flips (TRUE↔FALSE)
- Manual commands bypass debounce logic

---

## File Formats

### status.json (Written by daemon, read by UI)

```json
{
  "mode": "auto",
  "detection_state": {
    "meeting_detected": true,
    "detected_app": "zoom",
    "raw_detections": {
      "zoom_process_running": true,
      "zoom_host_running": true,
      "zoom_window_match": true,
      "teams_process_running": false,
      "teams_window_match": false
    },
    "confidence": "high",
    "evaluated_at": "2026-02-12T14:30:15Z"
  },
  "recording_state": {
    "recording": true,
    "started_at": "2026-02-12T14:30:00Z",
    "duration_seconds": 900,
    "output_path": "/Users/user/Videos/2026-02-12_1430_Zoom_Q1-Planning.mp4",
    "obs_status": "recording"
  },
  "teams_detected": false,
  "zoom_detected": true,
  "start_streak": 0,
  "stop_streak": 0,
  "last_action": "Started recording (auto)",
  "last_error": "",
  "timestamp": "2026-02-12T14:45:15Z",
  "obs_connected": true
}
```

### cmd.txt (Written by UI, read and cleared by daemon)

Single line containing one command:
```
start
```

### detection-rules.json (User configuration)

```json
{
  "rules": [
    {
      "application": "zoom",
      "process_names": ["zoom.us", "CptHost"],
      "window_hints": ["Zoom Meeting", "Zoom Webinar"],
      "enabled": true
    },
    {
      "application": "teams",
      "process_names": ["Microsoft Teams"],
      "window_hints": ["Meeting", "Call"],
      "enabled": true
    }
  ],
  "poll_interval_seconds": 2,
  "start_threshold": 3,
  "stop_threshold": 6
}
```

---

## Summary

| Entity | Persistence | Owner | Purpose |
|--------|-------------|-------|---------|
| StatusSnapshot | File (JSON) | Daemon writes, UI reads | System state visibility |
| DetectionState | Memory + StatusSnapshot | Daemon | Meeting detection evaluation |
| RecordingState | Memory + StatusSnapshot | Daemon | OBS recording tracking |
| OperatingMode | Memory + StatusSnapshot | Daemon | User control mode |
| DetectionRule | File (JSON) | User configures, both read | Detection customization |
| Command | File (text) | UI writes, daemon reads | User control actions |

All file-based entities use atomic write (temp + rename) for consistency.
