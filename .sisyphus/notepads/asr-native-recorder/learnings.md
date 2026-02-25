# Learnings — asr-native-recorder

## [2026-02-25] Orchestrator: Initial state audit

### Group 1 — COMPLETED (packages already exist, all tests pass)
- T025: `internal/recorder/` — Recorder interface + OBSAdapter. Tests pass.
- T026: `internal/asr/` — Backend interface, Segment/Transcript types, Registry with fallback. Tests pass.
- T027: `internal/transcript/` — WriteText/WriteSRT/WriteVTT/WriteAll. Tests pass.
- T037: `specs/003-asr-transcription/spec.md` — FR-013 spec written.

### Key interface facts (from reading source)
- `asr.Backend`: `Name() string`, `TranscribeFile(filePath string, opts TranscribeOptions) (*Transcript, error)`, `HealthCheck() (*HealthStatus, error)`
- `asr.Registry`: `NewRegistry()`, `Register(name, backend)`, `SetPrimary(name)`, `SetFallback(name)`, `TranscribeWithFallback(filePath, opts)`
- `recorder.Recorder`: `Connect()`, `Disconnect()`, `StartRecording(filename)`, `StopRecording(reason)`, `GetState()`, `IsConnected()`, `HealthCheck()`, `SetLogger()`, `OnStateChanged(fn)`, `OnDisconnected(fn)`
- `recorder.OBSAdapter`: wraps `obsws.Client`, created via `NewOBSAdapter(client *obsws.Client)`

### Project build state
- `go build ./...` — passes (only harmless `ld: warning: ignoring duplicate libraries: '-lobjc'`)
- `go test ./...` — ALL packages pass
- Module: `github.com/tiroq/memofy`, Go 1.21

### Config state
- `internal/config/detection_rules.go` — NO ASR fields yet (T032 still needed)
- `cmd/memofy-core/main.go` — still imports `obsws` directly, NO recorder.Recorder usage (T031 still needed)
- No `internal/asr/remotewhisper/`, `localwhisper/`, `googlestt/` (T028, T029, T030 still needed)

### Go module path for new packages
- `github.com/tiroq/memofy/internal/asr/remotewhisper`
- `github.com/tiroq/memofy/internal/asr/localwhisper`
- `github.com/tiroq/memofy/internal/asr/googlestt`

## [2026-02-25] T029: localwhisper backend implementation

### Implementation details
- `cmd.Output()` blocks on pipe drain even after `Process.Kill()` — must use `cmd.Start()` + `cmd.Wait()` for timeout kill to work
- Process group killing via `syscall.Kill(-pid, SIGKILL)` + `SysProcAttr{Setpgid: true}` needed to kill child processes (e.g., shell scripts spawning `sleep`)
- `bytes.Buffer` as `cmd.Stdout` works well for capturing output when using Start/Wait pattern
- HealthCheck returns `{OK: false}` (not error) for expected failures (missing binary, model, not executable) — errors reserved for unexpected failures
- `--help` exit code varies by binary; `*exec.ExitError` is acceptable, only non-ExitError means binary truly can't run
- Compile-time interface check: `var _ asr.Backend = (*Backend)(nil)`


## [2026-02-25] T028: remotewhisper backend implementation

### Implementation details
- Multipart form POST via `io.Pipe` + goroutine — avoids buffering entire audio file in memory
- `retryableError` wrapper type used to distinguish retryable (5xx, network) from non-retryable (4xx) errors
- `backoffBase` field (unexported) allows tests to inject 1ms backoff instead of 1s default — keeps test suite fast (~2.4s)
- HealthCheck returns `*HealthStatus` (not error) for expected failures (HTTP errors, bad JSON) — errors reserved for unexpected issues
- `floatSecToDuration` converts API's float seconds to `time.Duration` accurately
- Bearer token sent via `Authorization` header only when `Config.Token` is non-empty
- `TranscribeOptions.Model` overrides `Config.Model` when provided (opts take priority)