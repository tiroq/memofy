# localwhisper — Local Whisper CLI Backend

## OVERVIEW

ASR backend that shells out to a local whisper binary (whisper-cpp or faster-whisper CLI). Implements `asr.Backend`. T029 / FR-013.

## STRUCTURE

```
localwhisper/
├── backend.go      # Exec wrapper: builds CLI args, parses JSON output
└── backend_test.go # Unit tests with mock binary script
```

## CONSTRUCTOR

```go
b := localwhisper.NewBackend(localwhisper.Config{
    BinaryPath:     "/usr/local/bin/whisper-cpp",
    ModelPath:       "/models/ggml-small.bin",
    Model:           "small",
    Threads:         4,        // 0 = auto
    TimeoutSeconds:  300,      // default 5 min
})
```

## KEY TYPES

| Type | Purpose |
|------|---------|
| `Config` | BinaryPath, ModelPath, Model, Threads, TimeoutSeconds |
| `Backend` | `asr.Backend` implementation — exec + JSON parse |

## BEHAVIOR

- Runs whisper binary with `--output-json` flag and captures stdout
- Kills process after `TimeoutSeconds` via `syscall.SIGKILL` to process group
- `HealthCheck()` verifies binary exists at `BinaryPath` and is executable
- Parses whisper JSON output into `asr.Segment` slices

## ANTI-PATTERNS

- **No long-running process** — each `TranscribeFile` call spawns and waits
- **No `context.Context`** — project convention
- **No direct file I/O for transcripts** — returns `*asr.Transcript`; caller uses `transcript.WriteAll`
