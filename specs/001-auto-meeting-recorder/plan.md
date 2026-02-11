# Implementation Plan: Automatic Meeting Recorder (Memofy v0.1)

**Branch**: `001-auto-meeting-recorder` | **Date**: February 12, 2026 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/001-auto-meeting-recorder/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Primary requirement: macOS application that automatically records Teams/Zoom meetings via OBS using intelligent detection and stable state control, requiring 3 consecutive detections (6-9 seconds) to start and 6 consecutive non-detections (12-18 seconds) to stop recording, preventing file fragmentation.

Technical approach: Two-component Go architecture: (1) background daemon service for OBS WebSocket communication, meeting detection, and debounce state machine; (2) native macOS menu bar UI using cgo/Objective-C bridge for status display and user controls. Communication via file-based IPC (status.json, cmd.txt). Detection via macOS process/window APIs.

## Technical Context

**Language/Version**: Go 1.21+  
**Primary Dependencies**: 
- `github.com/gorilla/websocket` - OBS WebSocket v5 client
- `github.com/progrium/macdriver` - Native macOS APIs (NSStatusBar, NSWorkspace)
- `github.com/fsnotify/fsnotify` - File watching for command interface
- Standard library: `encoding/json`, `os/exec`, `time`, `sync`

**Storage**: File-based (JSON for status, text for commands, recordings managed by OBS)  
**Testing**: Go's built-in testing framework (`go test`), table-driven tests for state machine  
**Target Platform**: macOS 11+ (Big Sur and later) for Screen Recording permissions API  
**Project Type**: Single project with two binaries (daemon + menu bar app)  
**Performance Goals**: 
- Detection polling: 2-3 second intervals
- Command response: <2 seconds
- State transitions: 6-9s start, 12-18s stop
- Menu bar UI updates: <500ms

**Constraints**: 
- Must run as user-level process (not system daemon) for Screen Recording permissions
- Menu bar app must use macOS native UI (no cross-platform frameworks)
- Daemon must survive OBS crashes/restarts
- Zero CPU usage when idle (event-driven, not busy-wait polling)

**Scale/Scope**: 
- Single-user desktop application
- 2 binaries: memofy-core (daemon), memofy-ui (menu bar)
- ~10-15 source files total
- Expected to run 24/7 with minimal resource usage

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

**✅ PASS** - Simplicity: Two binaries (daemon + UI) is minimal for background service + menu bar pattern  
**✅ PASS** - Dependencies: 4 external packages, all well-established and maintained  
**✅ PASS** - File structure: Single Go project with clear separation (cmd/, internal/, pkg/)  
**✅ PASS** - Testing: State machine is isolated and testable with standard Go tests  
**✅ PASS** - Documentation: Planning follows specification + research + data model + contracts approach  

**No violations** - Project structure is appropriately scoped for the feature requirements.

## Project Structure

### Documentation (this feature)

```text
specs/[###-feature]/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)
<!--
  ACTION REQUIRED: Replace the placeholder tree below with the concrete layout
  for this feature. Delete unused options and expand the chosen structure with
  real paths (e.g., apps/admin, packages/something). The delivered plan must
  not include Option labels.
-->

```text
# [REMOVE IF UNUSED] Option 1: Single project (DEFAULT)
src/
├── models/
├── services/
├── cli/
└── lib/

tests/
├── contract/
├── integration/
└── unit/

# [REMOVE IF UNUSED] Option 2: Web application (when "frontend" + "backend" detected)
backend/
├── src/
│   ├── models/
│   ├── services/
│   └── api/
└── tests/

frontend/
├── src/
│   ├── components/
│   ├── pages/
│   └── services/
└── tests/

# [REMOVE IF UNUSED] Option 3: Mobile + API (when "iOS/Android" detected)
api/
└── [same as backend above]

ios/ or android/
└── [platform-specific structure: feature modules, UI flows, platform tests]
```

**Structure Decision**: [Document the selected structure and reference the real
directories captured above]

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| [e.g., 4th project] | [current need] | [why 3 projects insufficient] |
| [e.g., Repository pattern] | [specific problem] | [why direct DB access insufficient] |
