# Feature Specification: OBS Auto-Stop Investigation & Safeguard

**Feature Branch**: `002-obs-autostop`
**Created**: 2026-02-20
**Status**: Draft
**Input**: User description: "OBS Auto-Stop Investigation: Structured logging and diagnostic framework to reproduce, trace, and eliminate silent automatic recording stops in Memofy during manual recording mode"

## Clarifications

### Session 2026-02-20

- Q: Which OBS WebSocket protocol version is the integration target? → A: OBS WebSocket v5 (OBS 28+, current)
- Q: What format must the structured log use? → A: JSON / NDJSON — one JSON object per line
- Q: How is the diagnostic export triggered? → A: CLI flag / subcommand (e.g., `memofy-core --export-diag`)
- Q: How long does manual-session protection last against automated stops? → A: Full session — only explicit user stop is accepted
- Q: Must sensitive fields (OBS WS auth tokens, secrets) be redacted from log entries? → A: Yes — always redacted, replaced with `"[REDACTED]"`

## Background

During manual recording mode, OBS recordings have been observed stopping automatically without any user action. The problem disappears when the OBS WebSocket server is disabled, strongly implicating the WebSocket layer or Memofy's interaction with it. The root cause has not yet been confirmed — suspected triggers include a WebSocket reconnection loop, recording state drift, multiple competing clients, or a race condition between manual mode and auto-detection.

This feature covers:
1. A structured diagnostic logging framework to trace the full event chain leading to a silent stop
2. A permanent state-machine safeguard that prevents manual recordings from being stopped by any automated path

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Enable Debug Tracing for Recording Events (Priority: P1)

A developer suspects the auto-stop is caused by an event somewhere in the WebSocket or state-machine layer. They want to turn on verbose tracing, reproduce the issue, and read a complete timeline of every relevant event — all without modifying source code or restarting OBS.

**Why this priority**: Without a reliable event trace, root cause analysis is guesswork. This is the foundation all other stories depend on.

**Independent Test**: Can be fully tested by setting the debug flag, starting a recording session, and verifying that a structured log file is produced containing timestamped WebSocket and state-machine events.

**Acceptance Scenarios**:

1. **Given** the debug flag is enabled, **When** Memofy connects to OBS, **Then** every WebSocket message sent and received is written to the log with a millisecond-precise timestamp and source label.
2. **Given** a recording session is started manually, **When** any recording state change occurs, **Then** the log entry includes the triggering component, the new state, and the reason for the transition.
3. **Given** the debug flag is disabled, **When** Memofy runs normally, **Then** verbose WebSocket traffic is not written to the log so normal operation is unaffected.

---

### User Story 2 - Identify Who Sent the StopRecord Command (Priority: P2)

A developer wants to look at a captured log from a session where the auto-stop occurred and immediately determine which code path issued the StopRecord command and why.

**Why this priority**: Attributing the stop to a specific source is the single most important question the investigation must answer. This story is independently valuable as soon as logging is in place.

**Independent Test**: Can be fully tested by simulating a programmatic stop through each suspected code path (reconnect handler, state-drift poller, auto-detection logic) and confirming the log clearly identifies the source and reason for each.

**Acceptance Scenarios**:

1. **Given** a StopRecord command is about to be sent, **When** it is dispatched, **Then** the log entry contains the originating component (e.g., reconnect handler, state-drift check, auto-detection, explicit user action) and a machine-readable reason code.
2. **Given** no explicit stop was requested by the user, **When** reviewing the log after an unexpected stop, **Then** the developer can trace a single unbroken event chain from trigger to stop command in under five minutes.
3. **Given** multiple WebSocket clients are connected simultaneously, **When** any client sends a stop, **Then** the log identifies which client and its session identifier.

---

### User Story 3 - Protect Manual Recordings from Silent Auto-Stop (Priority: P3)

A user manually starts recording a meeting. They expect it to keep recording until they explicitly stop it, regardless of what auto-detection or WebSocket reconnections do in the background.

**Why this priority**: This is the permanent fix that prevents the bug from recurring after the root cause is addressed. Safe to implement in parallel with investigation.

**Independent Test**: Can be fully tested by starting a recording manually, then synthetically triggering each suspected auto-stop pathway (reconnect, state-drift, auto-detection), and verifying that the recording continues in all cases.

**Acceptance Scenarios**:

1. **Given** recording was started manually by the user, **When** a WebSocket reconnection occurs and Memofy re-syncs state, **Then** the recording is not stopped.
2. **Given** recording was started manually, **When** auto-detection logic determines no meeting is active, **Then** the recording continues and a warning is logged instead of a stop command being issued.
3. **Given** recording was started manually, **When** a stop signal arrives from any automated source at any point during the session, **Then** the stop is rejected and logged with reason "manual mode override".
4. **Given** recording was started manually, **When** the user explicitly clicks Stop, **Then** the recording stops normally.

---

### User Story 4 - Export Diagnostic Logs for Sharing (Priority: P4)

A developer or support contributor wants to export a clean, self-contained diagnostic bundle from a session by running a CLI subcommand, then share the output file for analysis — without needing to open the Memofy UI.

**Why this priority**: Enables asynchronous collaboration on hard-to-reproduce bugs without requiring the original environment.

**Independent Test**: Can be fully tested by triggering the export action and verifying the output file contains all required log sections and is human-readable.

**Acceptance Scenarios**:

1. **Given** a debug session has produced logs, **When** the user runs `memofy-core --export-diag`, **Then** a single file is produced containing the full structured event log, the Memofy version, and OS information.
2. **Given** the log file has grown beyond its maximum rolling size, **When** the user runs `memofy-core --export-diag`, **Then** the most recent complete session is included without truncation.

---

### Edge Cases

- What happens when the debug log file cannot be written (e.g., disk full, permissions error)? Memofy must continue operating normally and surface a single warning.
- What happens when a stop signal arrives exactly at the boundary of the debounce window (the 5-second race-condition guard at session start)? The system must treat the boundary as exclusive — a signal arriving at exactly t=5s is rejected.
- What happens when OBS crashes and restarts while a manual recording session is active? The manual-mode flag and session ownership must survive the reconnect.
- What happens when two Memofy instances are running simultaneously and both connect to OBS? Each must log its own session ID; the second instance must warn the user about a potential conflict.
- What happens if debug mode is active but the recording session ends before the log is flushed? All buffered entries must be written to disk on shutdown.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST write all WebSocket events (sent and received) conforming to the OBS WebSocket v5 protocol to a structured log when debug mode is active, with millisecond-precise ISO timestamps.
- **FR-002**: System MUST label every log entry with the originating component (e.g., `obs-ws-client`, `state-machine`, `auto-detector`, `reconnect-handler`).
- **FR-003**: System MUST include a machine-readable reason code on every StopRecord command dispatched, regardless of the originating path.
- **FR-004**: System MUST expose a debug mode toggle via an environment variable (`MEMOFY_DEBUG_RECORDING=true`) that enables verbose tracing without requiring a rebuild or OBS restart.
- **FR-005**: System MUST write logs in NDJSON format (one JSON object per line) to a rolling file capped at 10 MB, preserving the most recent entries on overflow.
- **FR-006**: System MUST support a `--export-diag` CLI subcommand on `memofy-core` that bundles the current NDJSON log, Memofy version, and OS details into a single portable file written to the current working directory.
- **FR-007**: Recording state machine MUST enforce the authority hierarchy for the full duration of a session — manual start takes precedence over auto-detection, which takes precedence over WebSocket sync — so no lower-priority source may stop a recording started by a higher-priority source at any point until the higher-priority source explicitly ends it.
- **FR-008**: For the entire duration of a manually-started recording session, system MUST reject any stop signal originating from an automated source and log the rejection with reason "manual mode override". In addition, a configurable debounce window (default: 5 seconds) after a manual start provides a race-condition guard at session initialisation, ensuring automated signals queued before the session lock took effect are also rejected.
- **FR-009**: System MUST NOT automatically synchronize recording state on WebSocket reconnection; an explicit intent must be required before any start or stop is issued post-reconnect.
- **FR-010**: System MUST detect when more than one WebSocket client session is active and surface a visible warning to the user.
- **FR-011**: System MUST log all WebSocket reconnection attempts, including the reason for disconnect and the outcome of each reconnect.
- **FR-012**: System MUST assign a unique session ID to each recording session and include it on all related log entries for that session.
- **FR-013**: System MUST redact sensitive fields from all log entries and diagnostic export bundles — specifically OBS WebSocket authentication tokens, derived secrets, and any password-bearing connection parameters — replacing their values with the literal string `"[REDACTED]"`.

### Key Entities

- **Recording Session**: Represents one recording interval — has an origin (manual or auto), a session ID, a start timestamp, and an end timestamp with attributed source and reason.
- **Log Entry**: A structured event record serialised as a single-line JSON object with fields: `timestamp` (ISO 8601, millisecond precision), `component`, `event`, `session_id`, and optional `payload`. Sensitive payload fields (auth tokens, secrets, passwords) are always replaced with `"[REDACTED]"` before writing.
- **Debug Export Bundle**: A self-contained diagnostic artifact containing the rolling log, Memofy version metadata, and OS info.
- **Stop Signal**: An instruction to end recording, carrying a required source component and reason code.

## Assumptions

- The integration targets OBS WebSocket v5 (bundled with OBS 28+); OBS WebSocket v4 (legacy plugin) is explicitly out of scope.
- Log entries are written as NDJSON (one JSON object per line); no other log format is supported.
- Sensitive fields (OBS WebSocket auth tokens, derived secrets, password-bearing connection parameters) are always redacted in log output and export bundles; there is no flag to disable redaction.
- A manually-started session is fully protected from automated stops for its entire duration; the debounce window (default: 5 seconds, configurable) serves only as a race-condition guard at session initialisation — it is not the sole protection mechanism.
- Debug mode is intended for developers and power users — verbose logging may produce large files during extended sessions.
- "Manual start" is any recording start initiated directly by the user through Memofy's UI; recordings started programmatically are treated as auto-detection starts unless explicitly flagged otherwise.
- Log file default path is `/tmp/memofy-debug.log`; this path should be configurable for environments where `/tmp` is restricted.
- The investigation phase (logging + reproduction) and the safeguard phase (state-machine fix) are treated as parallel work streams under this feature.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A developer can enable debug mode, reproduce the auto-stop scenario, and identify the exact triggering code path from the log alone — within a single debugging session, without modifying source code.
- **SC-002**: Every StopRecord command in the log is accompanied by a source component and reason code; zero ambiguous or unlabelled stop events are present.
- **SC-003**: Manually-started recordings never stop automatically after the safeguard is in place — validated by running all suspected auto-stop triggers against a live manual session with zero unexpected stops observed.
- **SC-004**: Running `memofy-core --export-diag` produces a complete, human-readable NDJSON bundle in under 10 seconds.
- **SC-005**: No new reports of silent recording stops occur in production for at least 30 days following the safeguard release.
- **SC-006**: The root cause of the observed auto-stop is identified and documented, with a confirmed reproduction path and a confirmed fix path.
