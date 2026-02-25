# asr — Automatic Speech Recognition

## OVERVIEW

Unified ASR interface and backend registry for transcription. Defines the `Backend` interface that all ASR implementations must satisfy, plus a `Registry` with primary/fallback support. T026 / FR-013.

## KEY TYPES

| Type | Purpose |
|------|---------|
| `Backend` | Interface: `Name()`, `TranscribeFile()`, `HealthCheck()` |
| `Segment` | Single transcribed segment with start/end timing and confidence |
| `Transcript` | Complete transcription result (segments, language, model, backend) |
| `TranscribeOptions` | Request config: language, model, timestamps, max segment length |
| `HealthStatus` | Backend health report: OK, message, latency |
| `Registry` | Manages backends with primary/fallback transcription |

## CONSTRUCTOR

```go
reg := asr.NewRegistry()
reg.Register("whisper-local", localBackend)
reg.Register("whisper-api", apiBackend)
reg.SetPrimary("whisper-local")
reg.SetFallback("whisper-api")
```

## FALLBACK BEHAVIOR

`TranscribeWithFallback(filePath, opts)` tries primary first. On error, retries with fallback. Error messages include both backend names and failure reasons.

## ANTI-PATTERNS

- **No concrete backends here** — this package defines interfaces only; implementations go in separate packages (e.g., `internal/asr/whisper/`)
- **No `context.Context`** — project convention
- **No streaming/live transcription** — deferred to Phase 1.5
- **No channels** — uses `sync.RWMutex` for concurrent access
