# googlestt — Google Cloud Speech-to-Text Backend (Stub)

## OVERVIEW

Stub ASR backend for Google Cloud Speech-to-Text. Implements `asr.Backend` but `TranscribeFile` returns an error ("not yet implemented"). T030 / FR-013.

## STRUCTURE

```
googlestt/
├── backend.go      # Stub implementation with credentials health check
└── backend_test.go # Unit tests
```

## CONSTRUCTOR

```go
b := googlestt.NewBackend(googlestt.Config{
    CredentialsFile: "/path/to/service-account.json",
    LanguageCode:    "en-US",
})
```

## KEY TYPES

| Type | Purpose |
|------|---------|
| `Config` | CredentialsFile, LanguageCode |
| `Backend` | `asr.Backend` stub — transcription not implemented |

## BEHAVIOR

- `TranscribeFile()` always returns error (stub)
- `HealthCheck()` verifies credentials file exists and is readable
- Compile-time interface check: `var _ asr.Backend = (*Backend)(nil)`

## ANTI-PATTERNS

- **Do not add Google Cloud SDK dependency** — this is a stub; real implementation is future work
- **No `context.Context`** — project convention
