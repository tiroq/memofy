# Implementation Tasks: Automatic Meeting Recorder (Memofy v0.1)

**Feature**: 001-auto-meeting-recorder  
**Generated**: February 12, 2026  
**Spec**: [spec.md](spec.md) | **Plan**: [plan.md](plan.md)

## Overview

This task breakdown organizes work by user story to enable independent implementation and testing. Each user story represents a complete, testable increment of functionality.

**MVP Scope**: User Story 1 (P1) delivers core automatic recording functionality.

---

## Phase 1: Project Setup

**Goal**: Initialize Go project structure, dependencies, and configuration files.

**Independent Test**: Project builds successfully with `go build ./...`

### Tasks

- [X] T001 Initialize Go module in project root with `go mod init github.com/tiroq/memofy`
- [X] T002 [P] Create directory structure: cmd/, internal/, pkg/, tests/, scripts/, configs/
- [X] T003 [P] Create subdirectories: cmd/memofy-core/, cmd/memofy-ui/, internal/{detector,statemachine,obsws,ipc,config}, pkg/macui/, tests/{integration,fixtures}
- [X] T004 [P] Install gorilla/websocket dependency with `go get github.com/gorilla/websocket@latest`
- [X] T005 [P] Install progrium/darwinkit (formerly macdriver) dependency with `go get github.com/progrium/darwinkit@latest`
- [X] T006 [P] Install fsnotify dependency with `go get github.com/fsnotify/fsnotify@latest`
- [X] T007 Create Makefile in project root with build, clean, test, run-core, run-ui targets
- [X] T008 Create default detection rules JSON config in configs/default-detection-rules.json
- [X] T009 Create .gitignore with Go patterns, bin/, .cache/, .DS_Store
- [X] T010 Run `go mod tidy` and verify all dependencies resolve

---

## Phase 2: Foundational Infrastructure

**Goal**: Implement core components needed by all user stories (state machine, OBS client, IPC, detection framework).

**Independent Test**: State machine and OBS client can be unit tested independently. IPC can write/read status.json.

### Tasks

- [X] T011 Define StatusSnapshot struct in internal/ipc/status.go with all fields from data-model.md
- [X] T012 [P] Define DetectionState struct in internal/detector/detector.go
- [X] T013 [P] Define RecordingState struct in internal/obsws/client.go
- [X] T014 [P] Define OperatingMode constants (ModeAuto, ModeManual, ModePaused) in internal/ipc/status.go
- [X] T015 [P] Define Command constants (CmdStart, CmdStop, CmdToggle, CmdAuto, CmdPause, CmdQuit) in internal/ipc/commands.go
- [X] T016 Implement atomic file write function (temp + rename) in internal/ipc/status.go
- [X] T017 Implement WriteStatus function to persist StatusSnapshot to ~/.cache/memofy/status.json
- [X] T018 [P] Implement ReadStatus function to load StatusSnapshot from file
- [X] T019 [P] Implement WriteCommand function to write command to ~/.cache/memofy/cmd.txt
- [X] T020 [P] Implement ReadCommand function to read and clear cmd.txt
- [X] T021 Define DetectionRule and DetectionConfig structs in internal/config/detection_rules.go
- [X] T022 Implement LoadDetectionRules function to read from ~/.config/memofy/detection-rules.json
- [X] T023 [P] Implement SaveDetectionRules function with validation (thresholds >= 1, stop >= start)
- [X] T024 Create state machine struct in internal/statemachine/machine.go with startStreak, stopStreak counters
- [X] T025 Implement state machine Update(detected bool) method with 3/6 threshold logic
- [X] T026 Implement state machine Reset() method to clear counters
- [X] T027 Write table-driven tests for state machine in internal/statemachine/machine_test.go covering all transitions
- [X] T028 Define OBS WebSocket client struct in internal/obsws/client.go with connection state tracking
- [X] T029 Implement OBS WebSocket Connect() with handshake (Hello → Identify → Identified)
- [X] T030 [P] Implement GetRecordStatus() request/response handling with 5s timeout
- [X] T031 [P] Implement StartRecord() request/response handling
- [X] T032 [P] Implement StopRecord() request/response handling
- [X] T033 Implement reconnection logic with exponential backoff (5s, 10s, 20s, max 60s)
- [X] T034 [P] Implement RecordStateChanged event subscription and handler
- [X] T035 Create mock OBS responses fixture in tests/fixtures/mock_obs_responses.json

---

## Phase 3: User Story 1 - Automatic Meeting Detection and Recording (P1)

**Goal**: Implement core automatic recording functionality - detect meetings and control OBS recording with debounce.

**Independent Test**: Start Zoom/Teams meeting → recording begins within 6-15s → end meeting → recording stops within 12-25s. No additional UI needed for testing (check status.json and log files).

### Tasks

- [X] T036 [US1] Define Detector interface in internal/detector/detector.go with Detect() method
- [X] T037 [P] [US1] Implement ZoomDetector in internal/detector/zoom.go checking zoom.us process + CptHost/window
- [X] T038 [P] [US1] Implement TeamsDetector in internal/detector/teams.go checking Microsoft Teams process + window
- [X] T039 [US1] Implement macOS process detection using NSWorkspace.runningApplications in internal/detector/detector.go
- [X] T040 [US1] Implement window title detection using Accessibility APIs (AXUIElement) in internal/detector/detector.go
- [X] T041 [US1] Implement DetectMeeting aggregator function that runs all detectors and returns DetectionState
- [X] T042 [US1] Create daemon main loop in cmd/memofy-core/main.go with 2s ticker for detection polling
- [X] T043 [US1] Integrate state machine into daemon: call Update() on each detection poll result
- [X] T044 [US1] Implement recording start action: when state machine returns START_RECORDING action, call OBS StartRecord()
- [X] T045 [US1] Implement recording stop action: when state machine returns STOP_RECORDING action, call OBS StopRecord()
- [X] T046 [US1] Implement status update: write StatusSnapshot to file after each state change
- [X] T047 [US1] Add startup checks for OBS connection and macOS permissions (Screen Recording, Accessibility)
- [X] T048 [US1] Implement permission check functions using CGPreflightScreenCaptureAccess and AXIsProcessTrusted
- [X] T049 [US1] Add logging to /tmp/memofy-core.out.log and /tmp/memofy-core.err.log with detection reasoning
- [ ] T050 [US1] Test automatic recording with Zoom meeting (start → wait 3 detections → record → end → wait 6 → stop)
- [ ] T051 [US1] Test automatic recording with Teams meeting (same flow as Zoom)
- [ ] T052 [US1] Test fragmentation prevention: verify short interruption (5-10s) doesn't create multiple files

---

## Phase 4: User Story 2 - Manual Recording Control Override (P2)

**Goal**: Add manual control via file-based commands, allowing users to override automatic detection.

**Independent Test**: Write commands to cmd.txt → daemon executes within 2s → status.json reflects new state. Works independently of User Story 3 (menu bar UI).

### Tasks

- [X] T053 [US2] Implement command file watcher using fsnotify in cmd/memofy-core/main.go
- [X] T054 [US2] Add fallback polling (1s interval) for command file if fsnotify fails
- [X] T055 [US2] Implement command handler switch statement for all commands (start, stop, toggle, auto, pause, quit)
- [X] T056 [P] [US2] Implement CmdStart handler: set mode to manual, call StartRecord() immediately
- [X] T057 [P] [US2] Implement CmdStop handler: call StopRecord() immediately
- [X] T058 [P] [US2] Implement CmdToggle handler: if recording, stop; else start
- [X] T059 [P] [US2] Implement CmdAuto handler: set mode to auto, resume detection-based control
- [X] T060 [P] [US2] Implement CmdPause handler: set mode to paused, suspend all detection
- [X] T061 [P] [US2] Implement CmdQuit handler: stop recording if active, cleanup, exit gracefully
- [X] T062 [US2] Update detection loop to skip actions when mode is Manual or Paused
- [X] T063 [US2] Test manual start command while no meeting detected (should start immediately)
- [X] T064 [US2] Test manual stop command while recording (should stop immediately)
- [X] T065 [US2] Test mode switching: auto → manual → auto (detection resumes after switching back)
- [X] T066 [US2] Test pause mode: verify detection doesn't run when paused

---

## Phase 5: User Story 3 - Status Monitoring and Configuration (P3)

**Goal**: Build native macOS menu bar UI for status display, manual controls, and settings.

**Independent Test**: Menu bar app displays correct status from status.json, controls write to cmd.txt, settings UI updates detection-rules.json. Fully independent from daemon.

### Tasks

- [X] T067 [US3] Create menu bar app entry point in cmd/memofy-ui/main.go
- [X] T068 [US3] Implement NSStatusBar initialization using progrium/macdriver in pkg/macui/statusbar.go
- [X] T069 [US3] Create status bar icon loading function with 4 states: IDLE (gray), WAIT (yellow), REC (red), ERROR (orange)
- [X] T070 [P] [US3] Implement menu construction in pkg/macui/statusbar.go with sections: Status, Controls, Actions, Settings
- [X] T071 [US3] Implement status file watcher using fsnotify to detect status.json changes
- [X] T072 [US3] Implement menu update function to refresh icon and menu labels based on StatusSnapshot
- [X] T073 [P] [US3] Add menu item: "Start Recording" → writes "start" to cmd.txt
- [X] T074 [P] [US3] Add menu item: "Stop Recording" → writes "stop" to cmd.txt  
- [X] T075 [P] [US3] Add menu item: "Auto Mode" with checkmark when mode == auto → writes "auto" to cmd.txt
- [ ] T076 [P] [US3] Add menu item: "Manual Mode" → writes "start" then switches tracking
- [X] T077 [P] [US3] Add menu item: "Pause" → writes "pause" to cmd.txt
- [X] T078 [P] [US3] Add menu item: "Open Recordings Folder" → opens OBS recording directory in Finder
- [X] T079 [P] [US3] Add menu item: "Open Logs" → opens /tmp/ directory in Finder showing log files
- [ ] T080 [US3] Implement macOS notification sending using NSUserNotificationCenter in pkg/macui/notifications.go
- [ ] T081 [US3] Add notification for ERROR state with actionable guidance (deep link to System Preferences)
- [ ] T082 [US3] Implement Settings window using NSWindow and NSView for detection rules configuration
- [ ] T083 [US3] Create Settings UI form: text fields for window hints (Teams, Zoom), sliders for thresholds
- [ ] T084 [US3] Implement Settings Save button: validates and writes to detection-rules.json
- [ ] T085 [US3] Add status display in menu: show mode, detected app, recording duration, last error
- [ ] T086 [US3] Test menu bar icon state changes: manually update status.json → icon updates within 500ms
- [ ] T087 [US3] Test control commands: click "Start Recording" → verify cmd.txt contains "start"
- [ ] T088 [US3] Test Settings UI: modify window hints → save → verify detection-rules.json updated
- [ ] T089 [US3] Test error notification: set error in status.json → verify macOS notification appears

---

## Phase 6: Deployment and Polish

**Goal**: Production installation, LaunchAgent setup, documentation, and final testing.

**Independent Test**: Install script successfully installs both binaries and LaunchAgent, daemon starts at login, recordings have correct filenames.

### Tasks

- [X] T090 Create LaunchAgent plist template in scripts/com.memofy.core.plist with RunAtLoad, KeepAlive, log paths
- [X] T091 Create install script in scripts/install-launchagent.sh that copies binaries, plist, loads with launchctl
- [X] T092 [P] Create uninstall script in scripts/uninstall.sh that unloads and removes all files
- [X] T093 Implement filename renaming after recording stops: OBS path → YYYY-MM-DD_HHMM_App_Title.mp4 format
- [X] T094 [P] Implement meeting title extraction from window title (best effort, fallback to "Meeting")
- [X] T095 Add comprehensive logging: detection reasoning, state transitions, OBS commands, errors
- [X] T096 [P] Add log rotation or size limits to prevent unbounded growth in /tmp/
- [X] T097 Create README.md in project root with installation, usage, troubleshooting sections
- [ ] T098 [P] Create user guide document with screenshots of menu bar states and settings
- [ ] T099 Test full installation flow: install → verify daemon starts at login → verify menu bar appears
- [ ] T100 Test end-to-end: meeting start → auto record → manual stop → verify filename format correct
- [ ] T101 Test error recovery: kill OBS during recording → verify reconnection → verify status shows error
- [ ] T102 Test permissions: revoke Screen Recording → verify error notification with guidance
- [ ] T103 Create integration test in tests/integration/state_transitions_test.go for full state machine flows
- [ ] T104 Run full test suite: `go test ./...` → verify all tests pass
- [ ] T105 Build release binaries with optimizations: `make build` → verify both binaries under 10MB

---

## Task Summary

**Total Tasks**: 105

### Tasks by Phase
- Phase 1 (Setup): 10 tasks
- Phase 2 (Foundational): 25 tasks (T011-T035)
- Phase 3 (User Story 1 - P1): 17 tasks (T036-T052)
- Phase 4 (User Story 2 - P2): 14 tasks (T053-T066)
- Phase 5 (User Story 3 - P3): 23 tasks (T067-T089)
- Phase 6 (Polish): 16 tasks (T090-T105)

### Parallelizable Tasks
41 tasks marked with [P] can be executed in parallel with other tasks in the same phase (different files, no dependencies).

### Dependencies

**Critical Path**:
1. Phase 1 (Setup) → Phase 2 (Foundational) MUST complete first
2. Phase 3 (US1) requires: T011-T035 (foundational infrastructure)
3. Phase 4 (US2) requires: Phase 3 complete (adds commands to existing daemon)
4. Phase 5 (US3) requires: Phase 2 complete (reads status.json, writes cmd.txt)
5. Phase 6 (Polish) requires: All user stories complete

**User Story Independence**:
- US1 (P1): Can be fully implemented and tested immediately after Phase 2
- US2 (P2): Depends on US1 (extends daemon with command handling)
- US3 (P3): Can be developed in parallel with US2 (only needs Phase 2 complete)

### Parallel Execution Examples

**Phase 2 Parallelization** (after T015 complete):
- Group A: T016-T020 (IPC file operations)
- Group B: T021-T023 (Config loading)
- Group C: T024-T027 (State machine)
- Group D: T028-T034 (OBS client)

**Phase 3 Parallelization** (after T036 complete):
- T037-T038 can run in parallel (independent detector implementations)
- T039-T041 are sequential (build up detection framework)

**Phase 5 Parallelization** (after T070 complete):
- T073-T079 can run in parallel (independent menu items)

---

## Implementation Strategy

### MVP First (Complete P1 User Story)
1. Complete Phase 1: Setup (T001-T010)
2. Complete Phase 2: Foundational (T011-T035)
3. Complete Phase 3: User Story 1 (T036-T052)
4. **MVP COMPLETE** - Can automatically record meetings without UI

### Incremental Delivery
After MVP, deliver user stories independently:
- **Release 1**: MVP only (automatic recording, file-based control)
- **Release 2**: MVP + US2 (manual override commands)
- **Release 3**: Full feature + US3 (menu bar UI and settings)
- **Release 4**: Production deployment (Phase 6 polish)

### Testing Strategy
- **Unit tests**: Write alongside implementation (state machine, detectors)
- **Integration tests**: After each user story complete
- **End-to-end tests**: Phase 6 before release

---

## Notes

- All [US1], [US2], [US3] labels map to the user stories from spec.md
- Tasks are designed to be independently verifiable (each has a clear done criterion)
- The [P] marker indicates tasks that can be parallelized within their phase
- File paths are exact - implementation should match the structure in plan.md exactly
- Tests are integrated throughout (not deferred to end) for continuous validation
