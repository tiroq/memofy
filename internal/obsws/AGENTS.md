# obsws — OBS WebSocket v5 Client

## OVERVIEW

Full OBS WebSocket v5 protocol client. Handles auth, request/response correlation, event subscriptions, reconnection, and source management. **All OBS communication in the project goes through this package.**

## STRUCTURE

```
obsws/
├── client.go      # Core client: connect, auth, request/response, events, reconnect
├── sources.go     # Source management: create/configure audio + display sources
├── client_test.go # Unit tests against MockOBSServer
└── sources_test.go
```

## CLIENT LIFECYCLE

```go
c := obsws.NewClient(url, password)
c.SetLogger(logger)
c.SetOnRecordStateChanged(func(recording bool) { ... })
c.SetOnDisconnected(func() { ... })
err := c.Connect()
defer c.Disconnect()
```

## KEY METHODS

| Method | Purpose |
|--------|---------|
| `Connect()` | Dial WS, authenticate (SHA256 challenge), identify |
| `StartRecording()` / `StopRecording()` | OBS recording control |
| `GetRecordingState()` | Returns `RecordingState` (cached + refreshed) |
| `IsConnected()` | Thread-safe connection check |
| `SetReconnectEnabled(bool, delay)` | Auto-reconnect on disconnect |

## PROTOCOL DETAILS

- **Auth**: `SHA256(password + salt)` then `SHA256(secret + challenge)`, base64-encoded
- **Request IDs**: monotonic int, per-request `chan *Response` stored in `responses` map
- **Event loop**: separate goroutine reads WS frames, dispatches to response channels or event handlers
- **Identified gate**: `identifiedChan` ensures no requests sent before handshake completes

## CONCURRENCY

```
mu sync.RWMutex         — guards conn, connected, identified
responseMu sync.RWMutex — guards responses map
stateMu sync.RWMutex    — guards recordingState cache
requestIDMu sync.Mutex  — guards requestID increment (T024)
loggerMu sync.RWMutex   — guards logger injection
```

## ANTI-PATTERNS

- **Never import `gorilla/websocket` outside this package** — use `obsws.Client` exclusively
- **Never call `conn.WriteMessage` directly** — use `sendRequest()` which handles ID + mutex
- **Never read `recordingState` directly** — use `GetRecordingState()` for thread-safe access
