# detector — Meeting Detection

## OVERVIEW

Multi-signal detection for Zoom, Teams, and Google Meet. Each app has its own detector file; `multi.go` aggregates them into a single `DetectionState`.

## STRUCTURE

```
detector/
├── detector.go        # Interfaces + shared types (DetectionState, RawDetection, ConfidenceLevel)
├── multi.go           # MultiDetector: runs all detectors, returns merged state
├── zoom.go            # Zoom: process + window title signals
├── teams.go           # Teams: process + window title signals
├── google_meet.go     # Google Meet: browser process + window title
├── process_darwin.go  # macOS process/window inspection (build tag: darwin)
└── process_stub.go    # No-op stub for non-darwin builds (tests on Linux CI)
```

## CORE INTERFACE

```go
type Detector interface {
    Detect() (*DetectionState, error)
    Name() string
}
```

## DETECTION SIGNALS (per app)

| Signal | Weight |
|--------|--------|
| Process running | Low confidence |
| Process + window match | Medium confidence |
| Process + window + host process | High confidence |

`ConfidenceLevel`: `none` → `low` → `medium` → `high`

## KEY TYPES

- `DetectionState` — snapshot: `MeetingDetected`, `DetectedApp`, `WindowTitle`, `RawDetections`, `ConfidenceLevel`, `EvaluatedAt`
- `RawDetection` — individual boolean signals (process, window, host)
- `DetectedApp` — string enum: `"zoom"` / `"teams"` / `"google_meet"` / `""`

## ANTI-PATTERNS

- **No build constraints on detector.go / multi.go** — only `process_darwin.go` has `//go:build darwin`
- **No direct syscalls outside `process_darwin.go`** — all OS-level inspection lives there
- **Detection is read-only** — detectors never modify state, write files, or call OBS
