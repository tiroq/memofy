# statemachine — Recording Lifecycle FSM

## OVERVIEW

Debounced finite state machine that governs when recording starts/stops. The single source of truth for recording state transitions; all recording decisions flow through here.

## KEY TYPES

| Type | Purpose |
|------|---------|
| `StateMachine` | Main FSM struct — holds streaks, mode, active session |
| `RecordingSession` | Metadata for active recording: `SessionID` (16-char hex), `Origin`, `App`, `StartedAt` |
| `RecordingOrigin` | `manual` / `auto` / `forced` — encodes who initiated |
| `StopRequest` | Full attribution for a stop signal: origin + reason + component |

## CORE METHOD

```go
ProcessDetection(state DetectionState) (shouldStart, shouldStop bool, app DetectedApp)
```

Called on every detection tick. Returns what action to take — caller (memofy-core) executes.

## DEBOUNCE LOGIC

- **Start**: `detectionStreak >= config.StartThreshold` (default 3)
- **Stop**: `absenceStreak >= config.StopThreshold` (default 6)
- Streaks reset on mode change or manual override

## ORIGIN PRIORITY

```
manual (2) > auto (1) = forced (1)
```

- Manual stop cannot be overridden by auto/forced start
- `StopRequest.RequestOrigin` checked before allowing stop

## CONSTRUCTOR

```go
sm := NewStateMachine(cfg)
sm.SetLogger(logger)           // optional diaglog injection
sm.SetDebounceDuration(d)      // override 5s race guard (tests)
```

## ANTI-PATTERNS

- **Never call OBS directly** — return `shouldStart`/`shouldStop` booleans; let `cmd/memofy-core` execute
- **Never set `currentMode` directly** — use `SetMode()` / command processing
- **Never share `StateMachine` across goroutines without locking** — it has no internal mutex; caller serializes
