# recorder — Backend-Agnostic Recording Interface

## OVERVIEW
Abstracts recording backends behind a common `Recorder` interface. Current implementation: `OBSAdapter` wrapping `obsws.Client`. Future: native macOS recorder via ScreenCaptureKit.

## STRUCTURE

```
recorder/
├── recorder.go          # Recorder interface, RecordingResult, RecorderState
├── obs_adapter.go       # OBSAdapter: wraps obsws.Client
├── obs_adapter_test.go  # Interface compliance + state mapping tests
└── AGENTS.md
```

## KEY TYPES

| Type | Purpose |
|------|---------|
| `Recorder` | Interface all backends implement |
| `RecordingResult` | Outcome of a completed recording |
| `RecorderState` | Current backend state snapshot |
| `OBSAdapter` | Wraps `obsws.Client` |

## CONSTRUCTOR

`NewOBSAdapter(client *obsws.Client) *OBSAdapter`

## METHOD MAPPING (OBSAdapter → obsws.Client)

| Recorder method | obsws.Client method |
|----------------|---------------------|
| `Connect()` | `Connect()` |
| `Disconnect()` | `Disconnect()` |
| `StartRecording(filename)` | `StartRecord(filename)` |
| `StopRecording(reason)` | `StopRecord(reason)` |
| `GetState()` | `GetRecordingState()` + `IsConnected()` |
| `IsConnected()` | `IsConnected()` |
| `HealthCheck()` | `GetRecordStatus()` |
| `SetLogger(l)` | `SetLogger(l)` |
| `OnStateChanged(fn)` | `OnRecordStateChanged(fn)` |
| `OnDisconnected(fn)` | `OnDisconnected(fn)` |

## ANTI-PATTERNS

- Never import `obsws` outside this package and `cmd/memofy-core` init
- Never call OBS WebSocket directly — use the Recorder interface
- No context.Context
