# Feature Specification: ASR Transcription

**Feature ID**: FR-013
**Feature Branch**: `003-asr-transcription`
**Created**: 2026-02-25
**Status**: Draft
**Ticket Range**: T025–T038
**Source**: `docs/memofy_unified_recording_and_asr_spec.md` (full unified spec)

## Overview

Automatic Speech Recognition (ASR) subsystem for Memofy. Transcribes completed meeting recordings into text using pluggable backends (remote Whisper API, local whisper CLI, Google STT stub). Operates in batch mode — transcription runs asynchronously after recording stops and never interferes with the detection loop or recording lifecycle.

This spec covers the **batch transcription** scope only. Live/streaming mode, two-pass refinement, and UI integration are deferred to future work.

---

## Architecture

### Package Layout

```
internal/
├── asr/                    # T026: Core interface, types, registry
│   ├── asr.go              # Backend interface + Transcript/Segment types
│   ├── registry.go         # Primary/fallback backend registry
│   ├── remotewhisper/      # T028: Remote Whisper API client
│   │   └── remotewhisper.go
│   ├── localwhisper/       # T029: Local whisper CLI wrapper
│   │   └── localwhisper.go
│   └── googlestt/          # T030: Google STT stub (future)
│       └── googlestt.go
├── transcript/             # T031: Transcript file writers
│   └── writer.go           # .txt, .srt, .vtt output
├── recorder/               # T025: Backend-agnostic recorder interface
│   ├── recorder.go         # Recorder interface
│   └── obs_adapter.go      # OBS WebSocket adapter
└── statemachine/           # Existing — triggers batch transcription
```

### Data Flow

```
recording stops → fileutil.Rename → goroutine:
  ┌─────────────────────────────────────────────┐
  │ registry.TranscribeWithFallback(audioPath)   │
  │   → primary backend (remote_whisper)         │
  │   → fallback backend (local_whisper) on fail │
  │ transcript.WriteTXT(segments, path)          │
  │ transcript.WriteSRT(segments, path)          │
  │ write .meta.json sidecar                     │
  └─────────────────────────────────────────────┘
```

Transcription runs in a detached goroutine spawned from the state machine's stop handler. Errors are logged via `diaglog` but never propagate to the detection loop.

---

## Backend Interface Contract

Defined in `internal/asr/asr.go` (T026):

| Method | Signature | Purpose |
|--------|-----------|---------|
| `Name()` | `string` | Human-readable backend identifier |
| `TranscribeFile()` | `(filePath string, opts TranscribeOptions) (*Transcript, error)` | Batch-transcribe a completed recording file |
| `HealthCheck()` | `(*HealthStatus, error)` | Verify backend availability |

All backends implement the `Backend` interface. No streaming methods are required for v1.

### Registry (T026)

`Registry` manages backends with primary/fallback support:
- First registered backend becomes primary by default
- `TranscribeWithFallback()` tries primary, falls back on error
- Thread-safe via `sync.RWMutex`

---

## Backend Implementations

### Remote Whisper API (T028)

HTTP client for self-hosted or remote Whisper-compatible API servers.

- Endpoint: `POST {base_url}/v1/transcribe` — multipart file upload
- Health: `GET {base_url}/v1/health`
- Configurable: `base_url`, `token`, `timeout_seconds`, `retries`, `model`
- Returns timestamped segments with confidence scores

### Local Whisper CLI (T029)

Wraps `whisper` or `whisper.cpp` CLI binary.

- Invokes CLI subprocess with `--output-json` flag
- Parses JSON output into `[]Segment`
- Configurable: `binary_path`, `model`, `language`, `device` (cpu/gpu)
- Fully offline, privacy-first

### Google STT Stub (T030)

Placeholder for future Google Cloud Speech-to-Text integration.

- Returns `ErrNotImplemented` for all methods
- Registered in registry but not selectable as primary until implemented

---

## Recorder Interface (T025)

Backend-agnostic recording interface in `internal/recorder/recorder.go`:

| Method | Purpose |
|--------|---------|
| `Connect()` / `Disconnect()` | Lifecycle management |
| `StartRecording(filename)` | Begin recording with given filename |
| `StopRecording(reason)` | Stop and return `RecordingResult` (path, duration, start time) |
| `GetState()` | Current `RecorderState` (recording, connected, backend, path) |
| `HealthCheck()` | Verify backend connectivity |

`OBSAdapter` (T025) implements this interface by delegating to `obsws.Client`. Future native macOS recorder will implement the same interface.

---

## Transcript Output (T031)

Package `internal/transcript` writes transcript files alongside the recording.

### Output Formats

| Format | Extension | Description |
|--------|-----------|-------------|
| Plain text | `.txt` | Concatenated segment text, one per line |
| SubRip | `.srt` | Numbered entries with `HH:MM:SS,mmm --> HH:MM:SS,mmm` timing |
| WebVTT | `.vtt` | `WEBVTT` header + timed cues |

### File Naming Convention

Given recording `2026-02-25_1430_Zoom_Q1-Planning.mp4`:
```
2026-02-25_1430_Zoom_Q1-Planning.txt
2026-02-25_1430_Zoom_Q1-Planning.srt
2026-02-25_1430_Zoom_Q1-Planning.vtt
2026-02-25_1430_Zoom_Q1-Planning.meta.json
```

All transcript files are written to the same directory as the recording.

---

## Sidecar Metadata (T032)

Each completed recording produces a `.meta.json` sidecar file:

```json
{
  "recording_id": "ses_abc123",
  "started_at": "2026-02-25T14:30:00Z",
  "ended_at": "2026-02-25T15:00:00Z",
  "duration_seconds": 1800,
  "application": "zoom",
  "origin": "auto",
  "recording_file": "2026-02-25_1430_Zoom_Q1-Planning.mp4",
  "transcription": {
    "backend": "remote_whisper",
    "model": "large-v3",
    "language": "en",
    "segments": 142,
    "completed_at": "2026-02-25T15:02:30Z",
    "output_files": [".txt", ".srt", ".vtt"]
  }
}
```

If transcription fails, the `transcription` field contains `"error": "..."` instead.

---

## Configuration Schema (T033)

ASR configuration is added to `detection-rules.json` as a top-level `asr` section:

```json
{
  "rules": [ ... ],
  "poll_interval_seconds": 2,
  "start_threshold": 3,
  "stop_threshold": 6,
  "asr": {
    "enabled": true,
    "backend": "remote_whisper",
    "fallback": "local_whisper",
    "language": "",
    "model": "large-v3",
    "output_formats": ["txt", "srt"],
    "remote_whisper": {
      "base_url": "http://10.0.0.50:8080",
      "token": "",
      "timeout_seconds": 120,
      "retries": 3
    },
    "local_whisper": {
      "binary_path": "/usr/local/bin/whisper",
      "model": "small",
      "device": "cpu"
    }
  }
}
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | bool | `false` | Master toggle for ASR |
| `backend` | string | `"remote_whisper"` | Primary backend name |
| `fallback` | string | `""` | Fallback backend (empty = none) |
| `language` | string | `""` | Language hint (empty = auto-detect) |
| `model` | string | `""` | Model override (backend-specific) |
| `output_formats` | []string | `["txt"]` | Which transcript formats to write |

The existing `DetectionConfig` struct gains an optional `ASR *ASRConfig` field. When `asr` is absent or `enabled` is false, no transcription occurs.

---

## Batch Transcription Flow (T034–T035)

### Sequence

1. **Recording stops** — state machine calls `StopRecording(reason)` → receives `RecordingResult` with output path
2. **File rename** — `fileutil.RenameRecording()` produces final filename (T027)
3. **Goroutine spawned** — non-blocking; detection loop continues immediately
4. **Transcribe** — `registry.TranscribeWithFallback(filePath, opts)` → `*Transcript`
5. **Write outputs** — `transcript.WriteTXT()`, `.WriteSRT()`, `.WriteVTT()` per config
6. **Write metadata** — `.meta.json` sidecar with recording + transcription details
7. **Log completion** — `diaglog.Log()` with component `"asr"`, event `"transcription_complete"`

### Failure Handling

- Primary backend fails → automatic fallback via registry
- All backends fail → log error with component `"asr"`, write `.meta.json` with error field
- Transcription goroutine panics → recovered, logged, daemon continues
- Never: crash the daemon, block the detection loop, or retry indefinitely

---

## Error Handling

| Scenario | Behavior |
|----------|----------|
| Backend unreachable | Retry per config, then fallback, then log error |
| Invalid audio file | Log error, skip transcription, write error to `.meta.json` |
| Disk full / write error | Log error, transcription result lost (recording preserved) |
| Binary not found (local) | `HealthCheck()` returns error, backend skipped |
| Config missing ASR section | ASR disabled, no transcription attempted |
| Goroutine panic | Recovered via `defer`, logged, daemon unaffected |

Transcription failures MUST NOT affect the recording lifecycle or detection loop. The ASR subsystem is a best-effort post-processing step.

---

## Acceptance Criteria

- **AC-01**: `asr.Backend` interface is implemented by at least two backends (remote whisper, local whisper) with passing unit tests
- **AC-02**: `Registry.TranscribeWithFallback()` correctly falls back from primary to fallback backend when primary returns an error
- **AC-03**: After recording stops, a `.txt` transcript file appears alongside the recording within the configured timeout period
- **AC-04**: `.srt` and `.vtt` outputs contain valid timing data matching the `Segment.Start`/`End` durations
- **AC-05**: `.meta.json` sidecar is written for every completed recording, regardless of transcription success/failure
- **AC-06**: Transcription failure does not crash the daemon or affect the detection loop — verified by simulating backend errors during active detection
- **AC-07**: ASR is disabled by default; setting `asr.enabled = true` in config activates it
- **AC-08**: `HealthCheck()` returns meaningful status for each backend (reachable, binary found, etc.)
- **AC-09**: `Recorder` interface is implemented by `OBSAdapter` with compile-time interface check
- **AC-10**: Config validation rejects invalid ASR settings (unknown backend, missing base_url for remote) without affecting existing detection config loading

---

## Task Breakdown (T025–T038)

| Ticket | Description | Package |
|--------|-------------|---------|
| T025 | Recorder interface + OBS adapter | `internal/recorder` |
| T026 | ASR Backend interface, types, registry | `internal/asr` |
| T027 | Recording file rename on stop | `internal/fileutil` |
| T028 | Remote Whisper API client | `internal/asr/remotewhisper` |
| T029 | Local Whisper CLI wrapper | `internal/asr/localwhisper` |
| T030 | Google STT stub | `internal/asr/googlestt` |
| T031 | Transcript file writers (.txt, .srt, .vtt) | `internal/transcript` |
| T032 | Sidecar `.meta.json` writer | `internal/transcript` or `internal/asr` |
| T033 | ASR config schema + validation | `internal/config` |
| T034 | Batch transcription orchestrator | `internal/asr` or `internal/statemachine` |
| T035 | Integration: wire ASR into recording stop flow | `cmd/memofy-core` |
| T036 | Unit tests for all ASR packages | `*_test.go` files |
| T037 | Feature spec document (this file) | `specs/003-asr-transcription` |
| T038 | Integration tests with mock backends | `tests/integration` |

---

## Future Work

These are explicitly **out of scope** for the T025–T038 ticket range:

- **Live/streaming mode** — real-time transcription during active recording (sliding window, partial segments, stitching)
- **Two-pass refinement** — draft model during live, recovery model post-meeting
- **Hybrid mode** — combined live draft + post-meeting recovery pass
- **UI integration** — transcript display in memofy-ui settings/status, health check in menu bar
- **Native macOS recorder** — ScreenCaptureKit + AVFoundation backend implementing `Recorder` interface
- **Additional backends** — Deepgram, Azure Speech, AssemblyAI
- **Speaker diarization** — per-speaker labels in transcript segments
- **Audio extraction** — separate audio-only file from video recording for faster transcription

---

## Assumptions

- Recordings are complete media files (MP4) accessible on the local filesystem when transcription starts
- The remote Whisper API conforms to the endpoint schema described in `docs/memofy_unified_recording_and_asr_spec.md` §4.2
- Local whisper binary (if configured) is pre-installed by the user; Memofy does not manage installation
- ASR config lives alongside detection config in `detection-rules.json` to avoid a second config file
- Transcription latency is acceptable in the range of seconds to minutes depending on file size and backend
- No `context.Context` usage — consistent with project conventions
