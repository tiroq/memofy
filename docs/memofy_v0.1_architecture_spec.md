# Memofy v0.1 -- Architecture Specification (Concise)

## Vision

Memofy is a macOS menu bar application that automatically records
Teams/Zoom meetings via OBS using intelligent detection and stable state
control.

------------------------------------------------------------------------

## Core Components

### 1. memofy-core (Daemon Service)

-   Persistent OBS WebSocket connection
-   Meeting detection (Teams/Zoom)
-   Debounce state machine (anti-flap)
-   Start/Stop recording logic
-   Writes runtime status file
-   Accepts control commands (manual override)

### 2. memofy-ui (Menu Bar App)

-   Displays status: IDLE / WAIT / REC / ERROR
-   Controls: Start / Stop / Auto / Pause
-   Shows last trigger (Teams/Zoom)
-   Opens logs and recordings folder
-   Settings (thresholds, hints, OBS config)

### 3. OBS Backend

-   Requires OBS v30+ with obs-websocket v5 enabled
-   Uses:
    -   GetRecordStatus
    -   StartRecord
    -   StopRecord

------------------------------------------------------------------------

## Detection Strategy (v0.1)

### Zoom

-   zoom.us process running
-   AND (CptHost process OR window title hint)

### Teams

-   Microsoft Teams process running
-   Window title matching configurable hints (via System Events)

Detection must be configurable due to localization.

------------------------------------------------------------------------

## State Machine

Poll interval: 2--5 seconds

Parameters: - START_AFTER = 3 polls - STOP_AFTER = 6 polls

Logic: - Raw detection → Debounce → Stable Intent - Only act on stable
transitions - Prevent file fragmentation

------------------------------------------------------------------------

## Runtime Files

### Status

\~/.cache/memofy/status.json Contains: - mode (auto/manual/paused) -
need (raw) - recording (actual) - teams_hit / zoom_hit - streak
counters - last_action - last_error

### Command

\~/.cache/memofy/cmd.txt Supported: - start - stop - toggle - auto -
pause - quit

------------------------------------------------------------------------

## Logging

-   /tmp/memofy-core.out.log
-   /tmp/memofy-core.err.log

Logs must include: - detection reasoning - debounce counters - OBS
actions - reconnect events

------------------------------------------------------------------------

## Packaging

-   LaunchAgent: com.memofy.core
-   Core starts at login
-   UI optional auto-start

------------------------------------------------------------------------

## Risks & Mitigations

1.  Teams detection instability → Configurable window hints

2.  Black screen in recordings → Enforce Screen Recording permission +
    Display Capture check

3.  Recording fragmentation → Debounce + cooldown logic

------------------------------------------------------------------------

## Future Extensions (v2+)

-   Audio activity detection
-   Automatic transcription
-   Indexed meeting archive
-   AI summaries
-   Calendar-based triggers
