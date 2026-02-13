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

### Initial Check (Pre-Research)

**✅ PASS** - Simplicity: Two binaries (daemon + UI) is minimal for background service + menu bar pattern  
**✅ PASS** - Dependencies: 4 external packages, all well-established and maintained  
**✅ PASS** - File structure: Single Go project with clear separation (cmd/, internal/, pkg/)  
**✅ PASS** - Testing: State machine is isolated and testable with standard Go tests  
**✅ PASS** - Documentation: Planning follows specification + research + data model + contracts approach  

**No violations** - Project structure is appropriately scoped for the feature requirements.

### Post-Design Re-Evaluation

**✅ PASS** - Design remains simple: Data model has 6 core entities, all necessary and minimal  
**✅ PASS** - No additional dependencies introduced during design phase  
**✅ PASS** - File structure remains single project (cmd/, internal/, pkg/ as planned)  
**✅ PASS** - Contracts documented: OBS WebSocket API contract complete  
**✅ PASS** - Research complete: All technical unknowns resolved without adding complexity

**Final Status**: ✅ ALL GATES PASSED - Ready to proceed to task breakdown (`/speckit.tasks`)

## Project Structure

### Documentation (this feature)

```text
specs/001-auto-meeting-recorder/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
│   └── obs-websocket-api.md
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
memofy/
├── cmd/
│   ├── memofy-core/      # Daemon service binary
│   │   └── main.go
│   └── memofy-ui/        # Menu bar app binary
│       └── main.go
├── internal/
│   ├── detector/         # Meeting detection logic
│   │   ├── detector.go
│   │   ├── zoom.go
│   │   └── teams.go
│   ├── statemachine/     # Debounce state machine
│   │   ├── machine.go
│   │   └── machine_test.go
│   ├── obsws/           # OBS WebSocket client
│   │   ├── client.go
│   │   └── commands.go
│   ├── ipc/             # Inter-process communication
│   │   ├── status.go
│   │   └── commands.go
│   └── config/          # Configuration management
│       ├── config.go
│       └── detection_rules.go
├── pkg/
│   └── macui/           # macOS UI helpers (cgo/Objective-C)
│       ├── statusbar.go
│       ├── statusbar.h
│       ├── statusbar.m
│       └── notifications.go
├── tests/
│   ├── integration/     # Integration tests
│   │   └── state_transitions_test.go
│   └── fixtures/        # Test data
│       └── mock_obs_responses.json
├── scripts/
│   └── install-launchagent.sh
├── configs/
│   └── default-detection-rules.json
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

**Structure Decision**: Standard Go project layout using `cmd/` for binaries, `internal/` for private application code, and `pkg/` for reusable macOS UI components. Two separate binaries in `cmd/` allow independent deployment and testing of daemon vs UI. The `internal/` packages are organized by domain concern (detection, state machine, OBS communication, IPC).

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

**N/A** - No constitution violations. Project structure is appropriately minimal for requirements.
