# Unified ASR + Native Recorder — Implementation Plan

## TL;DR

> **Quick Summary**: Add a unified ASR subsystem (local/remote/Google STT backends with live/batch/hybrid modes) to memofy-core, and lay architectural groundwork for a future Swift-based native recorder that replaces OBS. Phase 1 focuses entirely on the Go-side ASR pipeline — the Swift recorder is scoped as a separate tracked effort (Phase 2).
>
> **Deliverables**: 6 new Go packages (`internal/asr`, `internal/asr/localwhisper`, `internal/asr/remotewhisper`, `internal/asr/googlestt`, `internal/transcript`, `internal/recorder`), updated config system, updated IPC status, ASR integration into the recording lifecycle, and a recorder interface abstraction that decouples `statemachine` from the OBS backend.
>
> **Estimated Effort**: ~15 atomic tasks (Phase 1 ASR + Recorder Interface), ~5 tasks (Phase 2 Swift Recorder — future plan)
>
> **Parallel Execution**: YES — several packages can be built concurrently

---

## Context

### Original Request

Implement the spec at `docs/memofy_unified_recording_and_asr_spec.md`, which defines:
- **Part A**: Unified ASR Architecture — local/remote/Google STT backends, live/batch/hybrid transcription modes, two-pass refinement
- **Part B**: Native Recording Engine — replace OBS with ScreenCaptureKit + AVFoundation via a Swift package (`memofy-recorder`)

### Architecture Decisions (with rationale)

**AD-1: ASR First, Native Recorder Second**
- **Decision**: Implement the ASR subsystem (Part A) as the primary deliverable. The native recorder (Part B) gets an interface abstraction now, but Swift implementation is deferred.
- **Rationale**: ASR adds value immediately on top of existing OBS recordings. The native recorder requires Swift/XPC bridge engineering, code signing changes, and a fundamentally different build pipeline. Shipping ASR with OBS as the backend delivers user value without the multi-language complexity.

**AD-2: Native Recorder is Additive, Not Replacement (When Built)**
- **Decision**: When the Swift recorder is implemented, it will be an opt-in alternative alongside OBS, not an immediate replacement.
- **Rationale**: OBS is battle-tested and already works. Users should be able to choose. The `internal/recorder` interface abstracts both backends, letting `statemachine` and `memofy-core` be backend-agnostic.

**AD-3: ASR is Internal Package Only — No UI Changes in Phase 1**
- **Decision**: ASR integration lives in `internal/asr` and connects to the recording lifecycle via `memofy-core`. UI changes (backend selection, health check display, live transcript view) are deferred to a follow-up plan.
- **Rationale**: The darwinkit GUI is complex (AppKit threading, main thread constraints). Mixing ASR plumbing with UI work in one plan risks scope creep. Phase 1 delivers CLI-verifiable ASR that produces transcript files. UI is Phase 1.5.

**AD-4: Batch Mode First, Live Mode Second**
- **Decision**: Implement batch transcription (post-recording) before live/streaming transcription.
- **Rationale**: Batch is simpler (file in → transcript out), exercises the full backend interface, and delivers the highest value immediately. Live mode requires ring buffers, overlap stitching, segment finalization — significantly more complex. Batch also validates backend connectivity and model quality before investing in real-time infrastructure.

**AD-5: Recorder Interface Abstraction Now**
- **Decision**: Create `internal/recorder.Recorder` interface immediately, with OBS as the first implementation.
- **Rationale**: This decouples `statemachine` and `memofy-core/main.go` from `obsws` specifics, making the future Swift recorder a drop-in implementation. The refactor is modest (wrap existing `obsws.Client` calls) and pays for itself by cleaning up the architecture.

**AD-6: Config Extension via Existing JSON Pattern**
- **Decision**: Extend the existing `~/.config/memofy/detection-rules.json` with an `asr` section rather than creating a new config file.
- **Rationale**: Follows the established pattern. One config file to manage. The `config.DetectionConfig` struct grows but remains backward-compatible (new fields are optional with defaults).

**AD-7: No context.Context**
- **Decision**: Continue the project convention of no `ctx context.Context` anywhere.
- **Rationale**: Codebase explicitly prohibits it (AGENTS.md). ASR operations use timeouts via `time.After` patterns, consistent with `obsws` client.

### Scoping (what is in/out of this plan)

**IN SCOPE (Phase 1)**:
- `internal/recorder` — Backend-agnostic recorder interface + OBS adapter
- `internal/asr` — Unified STT interface, backend registry, config types
- `internal/asr/remotewhisper` — Remote Whisper API backend (most practical first backend)
- `internal/asr/localwhisper` — Local whisper.cpp/faster-whisper backend (stub + exec wrapper)
- `internal/asr/googlestt` — Google Cloud STT backend (stub, opt-in)
- `internal/transcript` — Transcript model, file writer (.txt, .srt, .vtt), segment stitching
- Config system extension for ASR settings
- IPC status extension for ASR state
- Integration into `memofy-core` recording lifecycle (batch transcription after recording stops)
- Sidecar metadata JSON for recordings

**OUT OF SCOPE (deferred)**:
- Swift `memofy-recorder` package (Phase 2 — separate plan)
- XPC/socket bridge between Go and Swift
- Live/streaming transcription mode (Phase 1.5)
- Two-pass refinement (Phase 1.5 — requires live mode first)
- UI changes (backend selector, health check panel, transcript viewer)
- Hybrid mode orchestration
- Audio extraction from video (use existing recording file directly)

---

## Work Objectives

### Must Have
- `internal/recorder.Recorder` interface with `obsws` adapter — decouple recording backend
- `internal/asr.Backend` interface with `TranscribeFile()` and `HealthCheck()` methods
- At least one working backend: `remotewhisper` (POST to Whisper API)
- `internal/transcript` package: segment model, `.txt` and `.srt` output
- ASR config section in detection-rules.json (backward-compatible)
- Batch transcription triggered automatically after recording stops in `memofy-core`
- Sidecar `.json` metadata file written alongside each recording
- All new packages have unit tests
- `task test` passes, `task build` passes
- Zero breaking changes to existing `StatusSnapshot`, `ipc`, or `statemachine` interfaces

### Must NOT Have
- No `context.Context` usage
- No custom error types — use `fmt.Errorf("...: %w", err)` wrapping
- No new goroutine channels for business logic (use mutexes)
- No breaking changes to existing IPC file format (only additive fields)
- No Swift code in this plan
- No UI/darwinkit changes
- No direct file writes for IPC — always use `ipc.WriteStatus()`
- No `log.Fatal` outside `main()`
- No HTTP server — ASR backends are clients, not servers

---

## TODOs

- [ ] T025: Create `internal/recorder` interface and OBS adapter
  **What to do**: Define a `Recorder` interface that abstracts recording backends, then create an `obsadapter` that wraps the existing `obsws.Client` to implement it. This decouples `statemachine` and `memofy-core` from OBS-specific types.

  Interface:
  ```go
  type RecordingResult struct {
      OutputPath string
      Duration   time.Duration
      StartedAt  time.Time
  }

  type RecorderState struct {
      Recording   bool
      Connected   bool
      BackendName string // "obs" | "native" (future)
      OutputPath  string
      StartTime   time.Time
      Duration    int // seconds
  }

  type Recorder interface {
      Connect() error
      Disconnect()
      StartRecording(filename string) error
      StopRecording(reason string) (RecordingResult, error)
      GetState() RecorderState
      IsConnected() bool
      HealthCheck() error
      SetLogger(l *diaglog.Logger)
      OnStateChanged(func(recording bool))
      OnDisconnected(func())
  }
  ```

  The OBS adapter wraps `obsws.Client` and translates its methods to this interface. It must preserve all existing behavior including reconnection, event handlers, and source management.

  **Files to create/modify**:
  - Create `internal/recorder/recorder.go` (interface + types)
  - Create `internal/recorder/obs_adapter.go` (OBS implementation)
  - Create `internal/recorder/obs_adapter_test.go` (tests using testutil MockOBSServer)
  - Create `internal/recorder/AGENTS.md` (knowledge base entry)

  **Must NOT do**:
  - Do not modify `obsws` package — adapter wraps it
  - Do not modify `statemachine` — it doesn't know about recorders
  - Do not add `context.Context`
  - Do not change any existing imports in `cmd/memofy-core/main.go` yet (that's T031)

  **Recommended Agent Profile**: category: deep, skills: []
  **Parallelization**: Can Run In Parallel: YES, Parallel Group: 1, Blocks: [T031], Blocked By: []
  **References**: `internal/obsws/client.go`, `internal/obsws/operations.go`, `testutil/`
  **Acceptance Criteria**:
  - `Recorder` interface compiles and is documented
  - `OBSAdapter` passes tests proving it delegates to `obsws.Client` correctly
  - `task test` passes with new tests included
  - Zero changes to existing packages

---

- [ ] T026: Create `internal/asr` package — unified STT interface and types
  **What to do**: Define the core ASR interfaces, types, and backend registry. This is the foundation all backends implement.

  Key types:
  ```go
  type Segment struct {
      Start    time.Duration
      End      time.Duration
      Text     string
      Language string
      Score    float64 // confidence 0.0–1.0
  }

  type Transcript struct {
      Segments []Segment
      Language string
      Duration time.Duration
      Model    string
      Backend  string
  }

  type TranscribeOptions struct {
      Language    string // "" = auto-detect
      Model       string // backend-specific model name
      Timestamps  bool
      MaxSegLen   int    // max segment length in seconds
  }

  type HealthStatus struct {
      OK       bool
      Backend  string
      Message  string
      Latency  time.Duration
  }

  type Backend interface {
      Name() string
      TranscribeFile(filePath string, opts TranscribeOptions) (*Transcript, error)
      HealthCheck() (*HealthStatus, error)
  }
  ```

  Also create a `Registry` that holds configured backends and supports fallback:
  ```go
  type Registry struct { ... }
  func NewRegistry() *Registry
  func (r *Registry) Register(name string, b Backend)
  func (r *Registry) Get(name string) (Backend, bool)
  func (r *Registry) Primary() Backend
  func (r *Registry) Fallback() Backend
  func (r *Registry) TranscribeWithFallback(filePath string, opts TranscribeOptions) (*Transcript, error)
  ```

  **Files to create/modify**:
  - Create `internal/asr/asr.go` (interfaces + types)
  - Create `internal/asr/registry.go` (backend registry with fallback)
  - Create `internal/asr/registry_test.go` (tests with mock backend)
  - Create `internal/asr/AGENTS.md`

  **Must NOT do**:
  - Do not implement any actual backend — this is interfaces only
  - Do not add streaming/live methods yet (Phase 1.5)
  - Do not add `context.Context`

  **Recommended Agent Profile**: category: deep, skills: []
  **Parallelization**: Can Run In Parallel: YES, Parallel Group: 1, Blocks: [T028, T029, T030, T033], Blocked By: []
  **References**: spec sections 2, 4, 9
  **Acceptance Criteria**:
  - All types and interfaces compile
  - Registry tests pass with mock backends
  - Fallback logic tested: primary fails → fallback succeeds
  - `task test` passes

---

- [ ] T027: Create `internal/transcript` package — segment model and file writers
  **What to do**: Implement transcript data model and output writers for `.txt`, `.srt`, and `.vtt` formats. This package consumes `asr.Transcript` and produces files alongside recordings.

  Key functions:
  ```go
  func WriteText(path string, t *asr.Transcript) error       // Plain text, one segment per line
  func WriteSRT(path string, t *asr.Transcript) error         // SubRip subtitle format
  func WriteVTT(path string, t *asr.Transcript) error         // WebVTT subtitle format
  func WriteAll(basePath string, t *asr.Transcript) error     // Write all formats (basePath without extension)
  ```

  SRT format:
  ```
  1
  00:00:00,000 --> 00:00:05,230
  Hello, welcome to the meeting.

  2
  00:00:05,500 --> 00:00:10,100
  Let's discuss the agenda.
  ```

  VTT format:
  ```
  WEBVTT

  00:00:00.000 --> 00:00:05.230
  Hello, welcome to the meeting.
  ```

  **Files to create/modify**:
  - Create `internal/transcript/transcript.go` (re-exports asr.Transcript or defines writer-specific helpers)
  - Create `internal/transcript/writer.go` (WriteText, WriteSRT, WriteVTT, WriteAll)
  - Create `internal/transcript/writer_test.go` (golden-file or string comparison tests)
  - Create `internal/transcript/AGENTS.md`

  **Must NOT do**:
  - Do not import `asr` backends — this package depends only on `asr` types
  - Do not add audio extraction logic
  - Do not add stitching/merging logic yet (Phase 1.5)

  **Recommended Agent Profile**: category: quick, skills: []
  **Parallelization**: Can Run In Parallel: YES, Parallel Group: 1, Blocks: [T033], Blocked By: []
  **References**: spec section 3B (batch output formats)
  **Acceptance Criteria**:
  - WriteText produces clean plain text output
  - WriteSRT produces valid SRT format with correct timestamps
  - WriteVTT produces valid WebVTT format
  - Edge cases tested: empty transcript, single segment, unicode text
  - `task test` passes

---

- [ ] T028: Implement `internal/asr/remotewhisper` backend
  **What to do**: Implement the Remote Whisper API backend. This is the primary ASR backend — it sends audio files to a Whisper API server (self-hosted or compatible service) via HTTP POST and receives timestamped transcripts.

  API contract (from spec section 4.2):
  - `POST /v1/transcribe` — multipart file upload with options
  - `GET /v1/health` — server health check

  Implementation:
  ```go
  type Config struct {
      BaseURL        string
      Token          string // optional auth token
      TimeoutSeconds int    // default 120
      Retries        int    // default 3
      Model          string // default "small"
  }

  type Client struct { ... }
  func NewClient(cfg Config) *Client
  func (c *Client) Name() string                    // "remote_whisper_api"
  func (c *Client) TranscribeFile(filePath string, opts asr.TranscribeOptions) (*asr.Transcript, error)
  func (c *Client) HealthCheck() (*asr.HealthStatus, error)
  func (c *Client) SetLogger(l *diaglog.Logger)
  ```

  TranscribeFile must:
  1. Open the audio/video file
  2. POST as multipart form to `{BaseURL}/v1/transcribe`
  3. Include model, language, timestamps options
  4. Parse JSON response into `asr.Transcript`
  5. Retry on transient errors with exponential backoff
  6. Respect timeout

  Use `net/http` standard library only (no external HTTP deps).

  **Files to create/modify**:
  - Create `internal/asr/remotewhisper/client.go`
  - Create `internal/asr/remotewhisper/client_test.go` (use `httptest.Server` for testing)

  **Must NOT do**:
  - Do not use `context.Context` — use `http.Client.Timeout` for deadline
  - Do not add WebSocket streaming (Phase 1.5)
  - Do not add gorilla/websocket dependency

  **Recommended Agent Profile**: category: deep, skills: []
  **Parallelization**: Can Run In Parallel: YES, Parallel Group: 2, Blocks: [T033], Blocked By: [T026]
  **References**: spec sections 4.2, 9; `internal/obsws/client.go` for retry/backoff patterns
  **Acceptance Criteria**:
  - Client implements `asr.Backend` interface
  - TranscribeFile works with httptest mock server
  - Retry logic tested (server returns 500 → retry → success)
  - Timeout tested
  - HealthCheck returns correct status
  - `task test` passes

---

- [ ] T029: Implement `internal/asr/localwhisper` backend (exec wrapper)
  **What to do**: Implement a local Whisper backend that shells out to `whisper-cpp` or `faster-whisper` CLI. This provides offline/privacy-first transcription without requiring a server.

  The backend invokes the whisper CLI as a subprocess, captures JSON output, and parses it into `asr.Transcript`.

  ```go
  type Config struct {
      BinaryPath string // path to whisper-cpp or faster-whisper CLI
      ModelPath  string // path to .bin model file
      Model      string // model name (e.g., "small", "base")
      Threads    int    // CPU threads (0 = auto)
  }

  type Backend struct { ... }
  func NewBackend(cfg Config) *Backend
  func (b *Backend) Name() string   // "local_whisper"
  func (b *Backend) TranscribeFile(filePath string, opts asr.TranscribeOptions) (*asr.Transcript, error)
  func (b *Backend) HealthCheck() (*asr.HealthStatus, error)
  ```

  TranscribeFile must:
  1. Validate binary exists at `BinaryPath`
  2. Build CLI args: `--model`, `--output-json`, `--language`, `--threads`
  3. Execute subprocess with timeout (kill after configurable duration)
  4. Parse JSON output into `asr.Transcript`

  HealthCheck must:
  1. Verify binary exists and is executable
  2. Verify model file exists
  3. Run `--help` or `--version` to confirm binary works

  **Files to create/modify**:
  - Create `internal/asr/localwhisper/backend.go`
  - Create `internal/asr/localwhisper/backend_test.go`

  **Must NOT do**:
  - Do not embed or bundle whisper binaries
  - Do not use CGO to link whisper.cpp directly (that's a future optimization)
  - Do not add GPU detection logic
  - Do not add `context.Context` — use `exec.CommandContext` alternative with manual timeout via goroutine + timer

  **Recommended Agent Profile**: category: deep, skills: []
  **Parallelization**: Can Run In Parallel: YES, Parallel Group: 2, Blocks: [T033], Blocked By: [T026]
  **References**: spec section 4.1; whisper.cpp CLI output format
  **Acceptance Criteria**:
  - Backend implements `asr.Backend` interface
  - Tests use a mock script that simulates whisper CLI output
  - HealthCheck detects missing binary gracefully
  - Timeout kills long-running subprocess
  - `task test` passes

---

- [ ] T030: Implement `internal/asr/googlestt` backend (stub)
  **What to do**: Create a minimal Google Cloud Speech-to-Text backend stub. This implements the `asr.Backend` interface but returns a "not configured" error unless Google credentials are present. The stub validates the interface contract and provides a template for future full implementation.

  ```go
  type Config struct {
      CredentialsFile string // path to service account JSON
      LanguageCode    string // e.g., "en-US"
  }

  type Backend struct { ... }
  func NewBackend(cfg Config) *Backend
  func (b *Backend) Name() string   // "google_stt"
  func (b *Backend) TranscribeFile(filePath string, opts asr.TranscribeOptions) (*asr.Transcript, error)
  func (b *Backend) HealthCheck() (*asr.HealthStatus, error)
  ```

  TranscribeFile: return `fmt.Errorf("google_stt: not yet implemented")` with a comment marking it as a stub.
  HealthCheck: check if credentials file exists, return appropriate status.

  **Files to create/modify**:
  - Create `internal/asr/googlestt/backend.go`
  - Create `internal/asr/googlestt/backend_test.go`

  **Must NOT do**:
  - Do not add Google Cloud SDK dependency
  - Do not import any google.golang.org packages
  - Keep this as a pure stub — no actual API calls

  **Recommended Agent Profile**: category: quick, skills: []
  **Parallelization**: Can Run In Parallel: YES, Parallel Group: 2, Blocks: [T033], Blocked By: [T026]
  **References**: spec section 4.3
  **Acceptance Criteria**:
  - Backend implements `asr.Backend` interface
  - TranscribeFile returns clear "not implemented" error
  - HealthCheck correctly reports credential status
  - `task test` passes

---

- [ ] T031: Refactor `memofy-core/main.go` to use `recorder.Recorder` interface
  **What to do**: Replace direct `obsws.Client` usage in `memofy-core/main.go` with the `recorder.Recorder` interface from T025. This is a refactor — behavior must not change.

  Changes:
  1. Import `internal/recorder`
  2. Create `recorder.NewOBSAdapter(obsClient)` after OBS connection
  3. Replace `obs *obsws.Client` parameters in `handleStartRecording`, `handleStopRecording`, `writeStatus`, `updateStatusMode`, `watchCommands`, `handleCommand` with `rec recorder.Recorder`
  4. Update `handleStartRecording` to use `rec.StartRecording(filename)` instead of `obs.StartRecord(filename)`
  5. Update `handleStopRecording` to use `rec.StopRecording(reason)` instead of `obs.StopRecord(reason)`
  6. Update `writeStatus` to use `rec.GetState()` instead of `obs.GetRecordingState()`
  7. Keep OBS-specific initialization (Connect, EnsureRequiredSources, GetVersion) in main() since those are OBS-specific setup — the Recorder interface handles runtime operations only

  The `obsws` package remains unchanged. `obsws.Client` is still created and passed to `NewOBSAdapter`. The OBS-specific startup (sources, version check) stays as-is.

  **Files to create/modify**:
  - Modify `cmd/memofy-core/main.go`

  **Must NOT do**:
  - Do not change `statemachine` package
  - Do not change `obsws` package
  - Do not change `ipc` package
  - Do not change behavior — this is a pure refactor
  - Do not remove OBS-specific startup code (EnsureRequiredSources, GetVersion, validation)

  **Recommended Agent Profile**: category: deep, skills: []
  **Parallelization**: Can Run In Parallel: NO, Parallel Group: 3, Blocks: [T033], Blocked By: [T025]
  **References**: `cmd/memofy-core/main.go`, `internal/recorder/`
  **Acceptance Criteria**:
  - `task build` succeeds
  - `task test` passes (including `cmd/memofy-core/startup_test.go`)
  - All existing functionality preserved — start/stop/mode/toggle/quit commands work identically
  - No `obsws.Client` references remain in function signatures (except initialization in `main()`)

---

- [ ] T032: Extend config system with ASR settings
  **What to do**: Add ASR configuration to the existing `DetectionConfig` struct in `internal/config/detection_rules.go`. The new fields are optional (zero-value defaults to "disabled") so existing config files remain valid without changes.

  New fields:
  ```go
  type ASRConfig struct {
      Enabled        bool   `json:"enabled"`                    // false = ASR disabled entirely
      Mode           string `json:"mode"`                       // "batch" | "live" | "hybrid" (default "batch")
      Backend        string `json:"backend"`                    // "remote_whisper_api" | "local_whisper" | "google_stt"
      FallbackBackend string `json:"fallback_backend,omitempty"` // optional fallback
      DraftModel     string `json:"draft_model,omitempty"`      // for live/hybrid (future)
      RecoveryModel  string `json:"recovery_model,omitempty"`   // for two-pass (future)

      OutputFormats  []string `json:"output_formats,omitempty"` // ["txt", "srt", "vtt"] default ["txt"]

      Remote RemoteWhisperConfig `json:"remote,omitempty"`
      Local  LocalWhisperConfig  `json:"local,omitempty"`
      Google GoogleSTTConfig     `json:"google,omitempty"`
  }

  type RemoteWhisperConfig struct {
      BaseURL        string `json:"base_url"`
      Token          string `json:"token,omitempty"`
      TimeoutSeconds int    `json:"timeout_seconds"` // default 120
      Retries        int    `json:"retries"`          // default 3
      Model          string `json:"model"`            // default "small"
  }

  type LocalWhisperConfig struct {
      BinaryPath string `json:"binary_path"`
      ModelPath  string `json:"model_path"`
      Model      string `json:"model"`   // default "small"
      Threads    int    `json:"threads"` // 0 = auto
  }

  type GoogleSTTConfig struct {
      CredentialsFile string `json:"credentials_file,omitempty"`
      LanguageCode    string `json:"language_code,omitempty"` // default "en-US"
  }
  ```

  Add to `DetectionConfig`:
  ```go
  type DetectionConfig struct {
      // ... existing fields ...
      ASR *ASRConfig `json:"asr,omitempty"` // nil = ASR disabled
  }
  ```

  Add validation for ASR config (valid mode, valid backend name, etc).
  Update `configs/default-detection-rules.json` to include a commented example or a disabled ASR block.

  **Files to create/modify**:
  - Modify `internal/config/detection_rules.go` (add types + validation)
  - Modify `internal/config/detection_rules_test.go` (add tests for ASR config)
  - Modify `configs/default-detection-rules.json` (add disabled ASR example)

  **Must NOT do**:
  - Do not break existing config loading — ASR must be optional
  - Do not create a separate config file
  - Do not add environment variable overrides (keep it simple)

  **Recommended Agent Profile**: category: quick, skills: []
  **Parallelization**: Can Run In Parallel: YES, Parallel Group: 2, Blocks: [T033], Blocked By: []
  **References**: `internal/config/detection_rules.go`, `configs/default-detection-rules.json`, spec section 7
  **Acceptance Criteria**:
  - Existing config files load without error (backward compatible)
  - New ASR config validates correctly (bad mode → error, bad backend → error)
  - Default config has `asr: null` or omitted (disabled by default)
  - `task test` passes

---

- [ ] T033: Integrate ASR batch transcription into recording lifecycle
  **What to do**: Wire ASR batch transcription into `memofy-core` so that after a recording stops, the output file is automatically transcribed (if ASR is enabled). This is the main integration task.

  Changes to `cmd/memofy-core/main.go`:
  1. After config loads, check if `cfg.ASR` is non-nil and enabled
  2. If enabled, create the appropriate `asr.Backend` from config and register in `asr.Registry`
  3. In `handleStopRecording`, after file rename succeeds, trigger batch transcription in a goroutine:
     ```go
     go func(filePath string, registry *asr.Registry, formats []string) {
         transcript, err := registry.TranscribeWithFallback(filePath, asr.TranscribeOptions{...})
         if err != nil {
             errLog.Printf("ASR transcription failed: %v", err)
             return
         }
         basePath := strings.TrimSuffix(filePath, filepath.Ext(filePath))
         if err := transcript.WriteAll(basePath, formats); err != nil {
             errLog.Printf("Failed to write transcript: %v", err)
         }
         outLog.Printf("Transcript written: %s", basePath)
     }(finalPath, asrRegistry, cfg.ASR.OutputFormats)
     ```
  4. Update `writeStatus` to include ASR state in `StatusSnapshot` (new optional field)

  **Files to create/modify**:
  - Modify `cmd/memofy-core/main.go` (ASR initialization + post-recording transcription)
  - Modify `internal/ipc/status.go` — add `ASRState` field to `StatusSnapshot` (optional, additive)

  **Must NOT do**:
  - Do not block the main detection loop — transcription runs in a goroutine
  - Do not modify `statemachine` — it doesn't know about ASR
  - Do not add live transcription yet
  - Do not change existing behavior when ASR is disabled (default)

  **Recommended Agent Profile**: category: deep, skills: []
  **Parallelization**: Can Run In Parallel: NO, Parallel Group: 4, Blocks: [T034], Blocked By: [T025, T026, T027, T028, T031, T032]
  **References**: `cmd/memofy-core/main.go`, all new packages
  **Acceptance Criteria**:
  - With ASR disabled (default): zero behavior change, `task build` + `task test` pass
  - With ASR enabled + remote backend configured: recording stop triggers transcription
  - Transcription failure does not crash daemon or affect detection loop
  - StatusSnapshot gains `asr_state` field (nil when disabled)
  - `task build` passes
  - `task test` passes

---

- [ ] T034: Write sidecar metadata JSON for recordings
  **What to do**: After a recording stops (and optionally after transcription completes), write a `<recording>.meta.json` sidecar file containing recording metadata.

  Metadata structure:
  ```go
  type RecordingMetadata struct {
      Version        string    `json:"version"`          // memofy version
      SessionID      string    `json:"session_id"`
      StartedAt      time.Time `json:"started_at"`
      StoppedAt      time.Time `json:"stopped_at"`
      Duration       string    `json:"duration"`         // human-readable
      DurationMs     int64     `json:"duration_ms"`
      App            string    `json:"app"`              // "zoom" | "teams" | "google_meet" | ""
      WindowTitle    string    `json:"window_title"`
      Origin         string    `json:"recording_origin"` // "manual" | "auto" | "forced"
      RecorderBackend string   `json:"recorder_backend"` // "obs" | "native"
      OutputFile     string    `json:"output_file"`
      ASR            *ASRMeta  `json:"asr,omitempty"`
  }

  type ASRMeta struct {
      Backend     string   `json:"backend"`
      Model       string   `json:"model"`
      Language    string   `json:"language"`
      Formats     []string `json:"formats"`     // ["txt", "srt"]
      Success     bool     `json:"success"`
      Error       string   `json:"error,omitempty"`
      TranscribedAt time.Time `json:"transcribed_at,omitempty"`
  }
  ```

  Write this as `<basepath>.meta.json` alongside the recording file. Use atomic write (temp + rename) consistent with `ipc.atomicWriteJSON`.

  **Files to create/modify**:
  - Create `internal/fileutil/metadata.go` (types + writer)
  - Create `internal/fileutil/metadata_test.go`
  - Modify `cmd/memofy-core/main.go` — call metadata writer in `handleStopRecording`

  **Must NOT do**:
  - Do not block on metadata write failure — log and continue
  - Do not include sensitive data (passwords, tokens) in metadata

  **Recommended Agent Profile**: category: quick, skills: []
  **Parallelization**: Can Run In Parallel: NO, Parallel Group: 5, Blocks: [T035], Blocked By: [T033]
  **References**: spec section 10, `internal/fileutil/filename.go`, `internal/ipc/status.go` (atomic write pattern)
  **Acceptance Criteria**:
  - Metadata file written alongside recording
  - Contains all fields from spec section 10.2
  - Atomic write (no partial files on crash)
  - `task test` passes

---

- [ ] T035: Add ASR health check to diaglog and startup validation
  **What to do**: Add ASR backend health check during `memofy-core` startup (when ASR is enabled). Log results via `diaglog`. If the configured backend is unhealthy, log a warning but don't fail startup (the backend might come online later).

  Changes:
  1. Add new diaglog event constants: `EventASRHealthCheck`, `EventASRTranscribeStart`, `EventASRTranscribeComplete`, `EventASRTranscribeFailed`
  2. Add new diaglog component: `ComponentASR`
  3. During startup, after ASR registry is created, call `HealthCheck()` on each registered backend
  4. Log results via structured diaglog entries
  5. If primary backend is unhealthy but fallback is healthy, log warning

  **Files to create/modify**:
  - Modify `internal/diaglog/diaglog.go` (add new constants)
  - Modify `cmd/memofy-core/main.go` (add health check during startup)

  **Must NOT do**:
  - Do not fail startup on unhealthy ASR backend
  - Do not add HTTP health endpoint (no HTTP server in this project)

  **Recommended Agent Profile**: category: quick, skills: []
  **Parallelization**: Can Run In Parallel: NO, Parallel Group: 6, Blocks: [T036], Blocked By: [T033]
  **References**: `internal/diaglog/diaglog.go`, spec section 11
  **Acceptance Criteria**:
  - Health check runs at startup when ASR enabled
  - Healthy backend: info log
  - Unhealthy backend: warning log with actionable message
  - Missing backend binary (local_whisper): clear error message
  - `task build` + `task test` pass

---

- [ ] T036: Add AGENTS.md knowledge base entries for all new packages
  **What to do**: Create `AGENTS.md` files for each new package following the existing convention. Each file should document the package's purpose, key types, constructor pattern, anti-patterns, and structure.

  Packages needing AGENTS.md:
  - `internal/recorder/AGENTS.md`
  - `internal/asr/AGENTS.md`
  - `internal/asr/remotewhisper/AGENTS.md` (can be brief)
  - `internal/asr/localwhisper/AGENTS.md` (can be brief)
  - `internal/asr/googlestt/AGENTS.md` (can be brief)
  - `internal/transcript/AGENTS.md`

  Note: T025 already creates `internal/recorder/AGENTS.md` and T026 creates `internal/asr/AGENTS.md`. This task covers the remaining ones and ensures consistency.

  **Files to create/modify**:
  - Create/verify `internal/asr/remotewhisper/AGENTS.md`
  - Create/verify `internal/asr/localwhisper/AGENTS.md`
  - Create/verify `internal/asr/googlestt/AGENTS.md`
  - Create/verify `internal/transcript/AGENTS.md`

  **Must NOT do**:
  - Do not overwrite AGENTS.md files already created by earlier tasks
  - Do not change any code — documentation only

  **Recommended Agent Profile**: category: quick, skills: []
  **Parallelization**: Can Run In Parallel: NO, Parallel Group: 6, Blocks: [T037], Blocked By: [T028, T029, T030, T027]
  **References**: Existing AGENTS.md files in `internal/obsws/`, `internal/ipc/`, `internal/statemachine/`
  **Acceptance Criteria**:
  - Every new package has an AGENTS.md
  - Format matches existing AGENTS.md files (OVERVIEW, KEY TYPES, ANTI-PATTERNS, etc.)

---

- [ ] T037: Create ASR feature spec in `specs/`
  **What to do**: Create a feature spec document in `specs/003-asr-transcription/` following the project's existing spec convention. This documents the ASR feature for future reference and serves as the authoritative spec (separate from the original unified doc).

  Content should cover:
  - Feature overview (FR-013: ASR Transcription)
  - Backend interface contract
  - Batch transcription flow
  - Config schema
  - Error handling policy
  - Acceptance criteria
  - Future: live mode, two-pass, UI integration

  **Files to create/modify**:
  - Create `specs/003-asr-transcription/spec.md`

  **Must NOT do**:
  - Do not duplicate the entire original spec — reference it
  - Do not include implementation details (code samples) — that's in AGENTS.md files

  **Recommended Agent Profile**: category: writing, skills: []
  **Parallelization**: Can Run In Parallel: YES, Parallel Group: 1, Blocks: [], Blocked By: []
  **References**: `specs/002-obs-autostop/spec.md` (format reference), `docs/memofy_unified_recording_and_asr_spec.md`
  **Acceptance Criteria**:
  - Spec follows existing format in `specs/`
  - Feature references use FR-013 numbering
  - Ticket references use T025–T037 range

---

- [ ] T038: Full build + test verification
  **What to do**: Final verification that everything compiles, tests pass, and lint is clean.

  Steps:
  1. Run `task build` — verify both binaries compile
  2. Run `task test` — all tests pass (existing + new)
  3. Run `task lint` — no new lint warnings
  4. Verify `configs/default-detection-rules.json` loads correctly
  5. Verify `task build-core` produces a binary that starts (with `--help` or similar non-daemon check)
  6. Check `lsp_diagnostics` on all modified/created files

  **Files to create/modify**: None — verification only

  **Must NOT do**:
  - Do not fix unrelated lint issues (only address issues from this plan's changes)
  - Do not modify code unless tests/build fail due to plan changes

  **Recommended Agent Profile**: category: quick, skills: []
  **Parallelization**: Can Run In Parallel: NO, Parallel Group: 7, Blocks: [], Blocked By: [T033, T034, T035, T036, T037]
  **References**: `Taskfile.yml`
  **Acceptance Criteria**:
  - `task build` exits 0
  - `task test` exits 0 with all tests passing
  - `task lint` exits 0 (or only pre-existing warnings)
  - No LSP errors on new files

---

## Dependency Graph

```
Group 1 (parallel):  T025, T026, T027, T037
Group 2 (parallel):  T028, T029, T030, T032    (T028-T030 blocked by T026)
Group 3:             T031                        (blocked by T025)
Group 4:             T033                        (blocked by T025, T026, T027, T028, T031, T032)
Group 5:             T034                        (blocked by T033)
Group 6 (parallel):  T035, T036                  (blocked by T033 and respective backends)
Group 7:             T038                        (blocked by everything)
```

---

## Success Criteria

### Verification Commands

```bash
# Build both binaries
task build

# Run all unit tests
task test

# Run linter
task lint

# Verify config backward compatibility
cat configs/default-detection-rules.json | python3 -m json.tool

# Verify new packages exist
ls -la internal/recorder/ internal/asr/ internal/transcript/
ls -la internal/asr/remotewhisper/ internal/asr/localwhisper/ internal/asr/googlestt/

# Verify AGENTS.md files
find internal/recorder internal/asr internal/transcript -name "AGENTS.md"

# Verify spec
ls specs/003-asr-transcription/spec.md
```

### Final Checklist

- [ ] `task build` passes (both binaries)
- [ ] `task test` passes (all existing + new tests)
- [ ] `task lint` clean (no new warnings)
- [ ] Existing OBS-based recording works identically (no regression)
- [ ] Config backward compatibility: old config files load without error
- [ ] StatusSnapshot backward compatibility: existing UI reads status.json without error
- [ ] Every new package has AGENTS.md
- [ ] Feature spec exists at `specs/003-asr-transcription/spec.md`
- [ ] No `context.Context` usage anywhere
- [ ] No custom error types
- [ ] No new channels for business logic
- [ ] All constructors follow `NewFoo(cfg) *Foo` pattern
- [ ] Ticket references (T025–T038) in code comments
- [ ] Feature reference FR-013 used consistently

---

## Phase 2 — Native Recorder (Future Plan)

The following tasks are **NOT part of this plan** but are documented for future planning:

1. **T039**: Create Swift Package `memofy-recorder` with ScreenCaptureKit + AVAssetWriter
2. **T040**: Implement XPC bridge between Go (`memofy-core`) and Swift (`memofy-recorder`)
3. **T041**: Create `internal/recorder/native_adapter.go` implementing `Recorder` interface via XPC
4. **T042**: Add recorder backend selection to config (`recorder.backend: "obs" | "native"`)
5. **T043**: Update `memofy-core` startup to select recorder backend from config
6. **T044**: Add Setup Wizard to `memofy-ui` for Screen Recording / Microphone permissions
7. **T045**: Integration tests with native recorder

These require:
- Swift 5.9+ toolchain
- Xcode build integration
- Code signing for screen capture entitlements
- Separate build pipeline (Swift Package Manager → framework → link from Go)
- Estimated effort: 7–10 tasks, separate plan recommended
