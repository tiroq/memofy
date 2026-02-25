# Memofy – Unified ASR Architecture Specification
## Local + Remote + Live STT + Context Recovery

---

# 1. Overview

Memofy supports a flexible Speech-to-Text (STT) architecture designed for:

- Live real-time transcription
- Post-meeting batch transcription
- Two-pass quality refinement
- Local and remote model execution
- Privacy-first operation

Supported backends:

- local_whisper (faster-whisper / whisper.cpp)
- remote_whisper_api (self-hosted GPU/CPU server or remote api)
- google_stt (optional managed backend)

---

# 2. Unified STT Interface

All backends must implement:

- TranscribeFile(filePath, options) -> Transcript
- StartStreaming(sessionConfig) -> StreamHandle
- StopStreaming(sessionId)
- HealthCheck() -> Status

---

# 3. Operating Modes

## A) Live Mode (Real-Time)

Goal:
- ≤ 2–3 second latency
- Stable partial updates
- Segment finalization logic

### Live Pipeline

1. Audio capture → ring buffer
2. Every STEP_SECONDS → send chunk
3. Receive partial transcript
4. Apply overlap + stitching
5. Finalize old segments
6. Update UI + write live transcript

Recommended defaults:

- STEP_SECONDS = 2
- WINDOW_SECONDS = 16
- OVERLAP_SECONDS = 10
- FINALIZE_LAG_SECONDS = 10

---

## B) Batch Mode (Post-Meeting)

1. Extract audio from recording
2. Submit full file to backend
3. Receive timestamped transcript
4. Save:
   - .txt
   - optional .srt / .vtt
5. Optional Recovery Pass

---

## C) Hybrid Mode

- Live Draft during meeting
- Strong Recovery model after meeting
- Automatic fallback support

---

# 4. Backends

## 4.1 Local Whisper

- Fully offline
- Sliding window inference
- CPU/GPU dependent
- Privacy-first

## 4.2 Remote Whisper API

Endpoints:

- POST /v1/transcribe
- POST /v1/stream (optional WebSocket)
- GET  /v1/health

Supports:
- model selection
- timestamps
- language selection
- retries + timeout

## 4.3 Google STT (Optional)

Used for:
- Enterprise reliability
- Minimal setup
- Low local resource usage

---

# 5. Two-Pass Transcription (Context Recovery)

## Pass 1: Draft (Low Latency)
- Smaller model
- Used in Live Mode
- Provisional segments

## Pass 2: Recovery / Refine
- Stronger model (e.g., large-v3)
- Runs on:
  - on_end (default)
  - periodic
  - manual
- Larger windows
- Higher decoding quality

---

# 6. Stitching & Correction Strategy

Preferred: Timestamp-Based Replacement

- Maintain global segment timeline
- Replace overlapping tail with latest inference
- Freeze segments older than FINALIZE_LAG_SECONDS
- Merge adjacent segments when appropriate

Fallback: Text Overlap Matching

---

# 7. Configuration Example

asr:
  mode: hybrid                 # realtime | batch | hybrid
  backend: remote_whisper_api  # local_whisper | remote_whisper_api | google_stt
  fallback: local_whisper
  draft_model: "small"
  recovery_model: "large-v3"
  realtime:
    step_seconds: 2
    window_seconds: 16
    overlap_seconds: 10
    finalize_lag_seconds: 10
  remote:
    base_url: "http://10.0.0.50:8080"
    token: ""
    timeout_seconds: 120
    retries: 3

---

# 8. Performance Targets

| Backend | Live Latency | CPU Load | Network |
|----------|-------------|----------|---------|
| Local Whisper | 2–6 sec | Medium–High | No |
| Remote Whisper | 1–3 sec | Low | Yes |
| Google STT | 1–2 sec | Very Low | Yes |

---

# 9. Error Handling

If backend fails:

1. Retry (exponential backoff)
2. Switch to fallback if enabled
3. Show macOS notification + menu bar error
4. Log to /tmp/memofy-core.err.log

---

# 10. Acceptance Criteria

- User can select backend (Local/Remote/Google)
- Live transcript updates with stable latency
- Recovery pass improves transcript quality
- Fallback works automatically
- Both live and batch outputs are valid and consistent
- Health check accessible via Settings UI

---

# Strategic Positioning

This unified ASR architecture ensures:

- Parity with Google STT-based assistants
- Offline-first capability
- GPU-accelerated remote option
- High-quality two-pass transcription
- Future extensibility (Deepgram, Azure, etc.)

Memofy becomes not just a recorder, but a structured meeting memory engine.

---

# Part B — Native Recording Engine (macOS)
## ScreenCaptureKit + AVFoundation/AVAssetWriter

# Memofy Native Recording Engine (macOS) — Detailed Architecture Spec (v0.1)

> **Goal:** Replace OBS with a **lightweight, native** macOS recording engine using **ScreenCaptureKit + AVFoundation**.
> This spec defines the recorder subsystem (“memofy-recorder”) as a first‑class backend for Memofy.

---

## 1) Objectives

### Must
- Capture **video** (display or window) and **audio** (system audio + microphone).
- Produce a single media file (default **MP4/H.264 + AAC**) with stable A/V sync.
- Support **start/stop** programmatically (no UI).
- Run reliably as a background service with minimal CPU/RAM.
- Provide strong diagnostics and explicit permission handling.

### Should
- Support **HEVC** optionally (better compression on Apple Silicon).
- Support **MKV** output (optional) for crash resilience (MP4 is not crash‑safe without finalization).
- Support recording **single window** (Teams/Zoom) or **full display**.
- Support dynamic device changes (mic switched, AirPods, etc.).

### Non‑Goals (v0.1)
- Multi-track audio export
- Live streaming to RTMP/WebRTC
- Built‑in editing
- Automatic transcription (handled by ASR subsystem)

---

## 2) Platform Constraints

### Minimum OS
- **macOS 13+** recommended.
  - ScreenCaptureKit is available since macOS 12.3, but stability/features improve in 13+.
- Apple Silicon + Intel supported (encode path differs).

### Permissions
- **Screen Recording** permission (required for video and system audio capture).
- **Microphone** permission (required if mic capture enabled).

Recorder must detect missing permissions and return actionable errors.

---

## 3) High‑Level Architecture

### Modules
1. **memofy-recorder (native engine)** — Swift package / framework
2. **memofy-core (daemon)** — business logic (meeting detection, state machine)
3. **memofy-ui (menu bar)** — user controls + setup checks

### Communication
- Preferred: **XPC** between `memofy-core` and `memofy-recorder` (native, secure, lifecycle-friendly).
- Alternative (simpler): localhost HTTP or UNIX socket IPC.

### Data Flow
```
ScreenCaptureKit (video+system audio)  ┐
                                       ├──> Sample Buffer Router ──> Encoders ──> AVAssetWriter ──> File
Microphone (AVAudioEngine)             ┘
```

---

## 4) Recorder Responsibilities

### 4.1 Capture Targets
- **Display** capture (primary use)
- **Window** capture (optional, depends on app compatibility and user preference)

### 4.2 Audio Sources
- **System audio** via ScreenCaptureKit audio output.
- **Microphone** via AVAudioEngine (or via ScreenCaptureKit audio if Apple exposes mic directly — prefer AVAudioEngine for control).

### 4.3 Output
Default container: **MP4**
- Video: **H.264** (hardware accelerated when available)
- Audio: **AAC**
Optional:
- **HEVC** video
- WAV/FLAC “audio-only” sibling output (future, or separate module)

---

## 5) APIs and Control Plane

### 5.1 Public API (Swift)
```swift
struct RecorderConfig {
  enum Target { case display(id: CGDirectDisplayID), window(id: CGWindowID) }
  var target: Target

  var includeSystemAudio: Bool
  var includeMicrophone: Bool

  var outputDirectory: URL
  var filenameTemplate: String   // e.g. "YYYY-MM-DD_HHMM_{App}_{Title}.mp4"

  var video: VideoConfig
  var audio: AudioConfig
}

struct VideoConfig {
  var codec: VideoCodec          // .h264 (default), .hevc
  var width: Int?                // nil = native
  var height: Int?
  var fps: Int                   // e.g. 30
  var bitrate: Int?              // optional
}

struct AudioConfig {
  var sampleRate: Double         // 48000 typical
  var channels: Int              // 2 typical
  var aacBitrate: Int            // e.g. 128_000
  var micDeviceID: String?       // optional specific device
}

protocol RecorderEvents {
  func onStateChanged(_ state: RecorderState)
  func onError(_ err: RecorderError)
  func onStats(_ stats: RecorderStats)     // cpu, dropped frames, drift, etc.
}

final class MemofyRecorder {
  init(config: RecorderConfig, events: RecorderEvents)
  func start() async throws -> RecordingHandle
  func stop() async throws -> RecordingResult
  func pause() async throws
  func resume() async throws
}
```

### 5.2 Control Commands from Core
- StartRecording(config)
- StopRecording()
- Pause/Resume (optional v0.1)
- HealthCheck()
- GetStats()

---

## 6) Capture Layer (ScreenCaptureKit)

### 6.1 Discovery
Use `SCShareableContent` to list displays/windows:
- displays: `SCShareableContent.current.displays`
- windows: `SCShareableContent.current.windows`

### 6.2 Stream Configuration
- `SCStreamConfiguration`:
  - width/height (optional)
  - minFrameInterval (fps)
  - captureAudio = true (if system audio enabled)
  - queueDepth tuned for latency vs stability

### 6.3 Outputs
Add two outputs:
- `SCStreamOutputType.screen` → video `CMSampleBuffer`
- `SCStreamOutputType.audio` → system audio `CMSampleBuffer`

Implementation:
- Create a dedicated dispatch queue per output.
- Route buffers into a unified “sample router” with timestamps.

### 6.4 Key Requirement: Timestamp Discipline
All buffers are timestamped (PTS). Recorder must:
- preserve monotonic PTS
- detect discontinuities
- handle stream restarts gracefully

---

## 7) Microphone Capture (AVAudioEngine)

### 7.1 Approach
- Use `AVAudioEngine` with `inputNode`
- Convert to desired format (sampleRate/channels)
- Produce `CMSampleBuffer` or `AVAudioPCMBuffer` then convert into `CMSampleBuffer` for AssetWriter

### 7.2 Device Selection
- Default: system input device
- Optional: select by device UID
- React to route changes (AirPods connected/disconnected)

### 7.3 Sync Strategy
- Align mic audio timestamps to the same clock domain as ScreenCaptureKit.
- If direct clock alignment is unstable, implement drift correction:
  - periodically compare mic sample timestamps vs system audio / video PTS
  - insert/remove tiny silent frames or resample minimally (advanced; can be v0.2)

---

## 8) Sample Buffer Router

A central component to:
- Accept buffers from (video, system audio, mic)
- Validate and normalize timestamps
- Feed buffers to the writer inputs in correct order
- Track dropped frames, backpressure, and drift

### 8.1 Backpressure Handling
If `AVAssetWriterInput.isReadyForMoreMediaData == false`:
- Prefer dropping **video** frames (keep audio continuous) after a threshold
- Never block capture queues indefinitely
- Track counters + emit stats

### 8.2 Ordering
Writer expects monotonically increasing PTS per track.
Router maintains per-track last PTS and rejects out-of-order buffers.

---

## 9) Encoding + File Writing (AVAssetWriter)

### 9.1 Writer Setup
- `AVAssetWriter(outputURL:fileType:)`
  - `.mp4` by default
- Create inputs:
  - `AVAssetWriterInput(mediaType: .video, outputSettings: ...)`
  - `AVAssetWriterInput(mediaType: .audio, outputSettings: ...)` for system audio
  - Optional second audio track for mic (v0.2) or mixdown into one track (v0.1)

### 9.2 Audio Mix Strategy (v0.1)
To keep v0.1 simple and compatible:
- **Mix system + mic into a single AAC track**
  - Pros: simplest playback and sync
  - Cons: cannot adjust levels later

Implementation options:
- Use `AVAudioMixerNode` + render into PCM then encode
- If using only system audio OR only mic, skip mixing

### 9.3 Start/Stop Semantics
- On `start()`:
  - `writer.startWriting()`
  - `writer.startSession(atSourceTime: firstPTS)`
- On `stop()`:
  - mark inputs as finished
  - `writer.finishWriting` and return result with file URL and duration

### 9.4 Crash Safety
MP4 requires finalization. If the process dies mid-recording, file may be unplayable.
Mitigations:
- Keep recordings short (split files) OR
- Implement “periodic segmentation” (rotate every N minutes) OR
- Offer MKV container in v0.2 via different muxer (non‑AVAssetWriter).

---

## 10) File Naming & Metadata

### 10.1 Filename Template (chosen)
`YYYY-MM-DD_HHMM_{Application}_{Meeting-Title}.mp4`

Rules:
- sanitize title for filesystem safety (ASCII fallback, remove slashes, trim length)
- if title unknown: `{Application}_Unknown`
- ensure uniqueness (append `_1`, `_2`)

### 10.2 Sidecar Metadata (JSON)
Write `<file>.json`:
- start/end timestamps
- capture target (display/window)
- audio sources enabled
- fps/codec settings
- dropped frames statistics
- app detection result (Teams/Zoom)
- recorder version

---

## 11) Health Checks & Setup Wizard

### 11.1 HealthCheck API
- Can we list shareable content?
- Do we have Screen Recording permission?
- Can we access mic (if enabled)?
- Can we create output directory?
- Can we write a tiny test file?

### 11.2 UI Setup Wizard (memofy-ui)
- Step 1: Request Screen Recording permission
- Step 2: Request Microphone permission
- Step 3: Select capture target (display/window)
- Step 4: Test record 3 seconds → play back

---

## 12) Error Handling

### Error categories
- PermissionDenied(ScreenRecording/Microphone)
- CaptureTargetUnavailable
- AssetWriterFailure(details)
- AudioEngineFailure(details)
- DiskFull / WriteDenied
- TimestampDiscontinuity

### UX policy
- Menu bar error state + actionable notification
- “Open Settings” deep link when permissions missing
- Automatic retry only for transient failures (not permissions)

---

## 13) Performance Targets

- Idle daemon CPU: near 0%
- Recording CPU: depends on codec; target < 15% on Apple Silicon for 1080p30 H.264
- Memory: < 200MB steady state
- Dropped frames: < 0.5% typical
- A/V drift: < 100ms per hour (goal; tune later)

---

## 14) Testing Plan

### 14.1 Unit Tests
- filename sanitization
- timestamp ordering logic
- router backpressure behavior
- metadata generation

### 14.2 Integration Tests (manual + automated harness)
- 1–5 minute recordings with system audio only
- mic only
- system+mic mix
- window capture vs display capture
- sleep/wake during recording
- device change (Bluetooth headset connect/disconnect)

### 14.3 Diagnostics Bundle
One command in UI:
- export logs + status + config + last 60s stats

---

## 15) Roadmap (Recorder Only)

### v0.1 (MVP Native Recorder)
- Display capture + system audio
- Optional mic capture + mixdown
- MP4 output, H.264
- Start/Stop, stats, health checks

### v0.2
- HEVC option
- segmentation/rotation for crash resilience
- separate audio tracks (system vs mic)
- improved drift correction

### v0.3
- window capture hardening
- audio VAD, auto pause on silence
- “safe mode” fallback settings

---

## Critical Evaluation (Weakest Link)
**A/V sync** and **system+mic mixing** are the most failure-prone areas.
Mitigate by:
- shipping v0.1 with system audio only (mic optional),
- adding rigorous stats and drift detection,
- and supporting segmentation to avoid long corrupt recordings.
