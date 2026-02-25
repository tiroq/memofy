# remotewhisper — Remote Whisper API Backend

## OVERVIEW

ASR backend that calls a remote Whisper HTTP API (e.g., faster-whisper-server, OpenAI-compatible). Implements `asr.Backend`. T028 / FR-013.

## STRUCTURE

```
remotewhisper/
├── client.go      # HTTP client: multipart upload, retry with backoff, response parsing
└── client_test.go # Unit tests with httptest server
```

## CONSTRUCTOR

```go
c := remotewhisper.NewClient(remotewhisper.Config{
    BaseURL:        "http://localhost:8000",
    Token:          "optional-bearer-token",
    TimeoutSeconds: 120,
    Retries:        3,
    Model:          "small",
})
c.SetLogger(diagLogger)
```

## KEY TYPES

| Type | Purpose |
|------|---------|
| `Config` | BaseURL, Token, TimeoutSeconds, Retries, Model |
| `Client` | `asr.Backend` implementation — HTTP multipart upload |

## BEHAVIOR

- Uploads audio file via `POST /v1/audio/transcriptions` (multipart)
- Retries on failure with exponential backoff (base × 2^attempt + jitter)
- `HealthCheck()` sends `GET /v1/models` to verify server reachability
- Response parsed as OpenAI-compatible JSON with segments

## ANTI-PATTERNS

- **No direct `http.Client` construction** — use `NewClient()` which sets timeout
- **No `context.Context`** — project convention
- **No streaming** — batch upload only (live mode deferred)
