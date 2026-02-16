# OBS Integration

## WebSocket Connection

- **Port**: 4455 (default)
- **Auth**: None required
- **Protocol**: obs-websocket v5.x
- **Min OBS Version**: 28.0

## Auto-Configuration

Sources auto-created on first run:
- **Display Capture**: Screen recording
- **Audio Input**: System audio

Scene must be active for source creation.

## Operations

```
StartRecord         # Start recording
StopRecord          # Stop recording
GetRecordStatus     # Check if recording
GetSceneList        # List scenes
CreateInput         # Create source
SetInputSettings    # Configure source
```

## Reconnection

Connection lost → Exponential backoff (5s → 10s → 20s → 40s → 60s)  
Auto-reconnect for up to 5 minutes

## Error Codes

- **204**: Invalid request (OBS < 28.0 or WebSocket disabled)
- **Timeout**: OBS not reachable

See `/tmp/memofy-core.err.log` for errors.
