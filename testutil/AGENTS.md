# testutil — Shared Test Helpers

## OVERVIEW

Shared helpers for unit and integration tests: a fake OBS WebSocket server and assertion utilities. Import as `github.com/tiroq/memofy/testutil`.

## STRUCTURE

```
testutil/
├── mock_obs.go    # MockOBSServer: in-process OBS WS v5 stub
└── assertions.go  # Custom test assertion helpers
```

## MockOBSServer

Spins up a real `net/http` WebSocket server implementing the OBS WS v5 handshake. Use it to test `obsws.Client` without a real OBS instance.

```go
srv := testutil.NewMockOBSServer(t)
defer srv.Close()

client := obsws.NewClient(srv.URL(), "")
err := client.Connect()
```

Key controls:
- `srv.SetRecordingState(bool)` — make server report recording on/off
- `srv.SimulateDisconnect()` — force a mid-session disconnect
- `srv.RequestCount()` — assert how many requests were sent

## assertions.go

Project-specific assertion wrappers over `testing.T`. Prefer these over bare `t.Fatal` for consistent failure messages.

## ANTI-PATTERNS

- **`MockOBSServer` is for unit tests only** — integration tests in `tests/integration/` use a real OBS instance
- **Never import `testutil` from non-test code** — it imports `testing` package
- **Always `defer srv.Close()`** — leaks goroutines otherwise
