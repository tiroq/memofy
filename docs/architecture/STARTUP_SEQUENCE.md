# Startup Sequence

## memofy-core

1. Check for existing PID file
2. Load config (`~/.config/memofy/detection-rules.json`)
3. Connect to OBS WebSocket (port 4455)
4. Validate OBS version (require 28.0+)
5. Check/create scene sources (Display + Audio)
6. Initialize state machine
7. Start detection loop (every 2s)

## memofy-ui

1. Initialize macOS menu bar
2. Load status from `~/.cache/memofy/status.json`
3. Start status polling (every 1s)
4. Display menu bar icon

## OBS Auto-Launch

If OBS not running:
1. Core attempts to launch via `open -a OBS`
2. Waits up to 30s for WebSocket
3. Continues initialization

Logs at `/tmp/memofy-core.out.log`
