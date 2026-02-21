# Tasks: OBS Auto-Stop Investigation & Safeguard

**Feature Branch**: `002-obs-autostop`
**Input**: Design documents from `/specs/002-obs-autostop/`
**Prerequisites**: plan.md ✓, spec.md ✓, research.md ✓, data-model.md ✓, contracts/ ✓

**Format**: `- [ ] [T###] [P?] [Story?] Description — file path`
- **[P]** marks tasks that can run in parallel (touch different files, no partial-result dependencies)
- **[Story]** marks tasks belonging to a specific user story phase

---

## Phase 1: Setup

**Purpose**: Create the new package skeleton so all user story phases can reference real import paths.

- [x] T001 Create `internal/diaglog/` package with empty Go source files: `diaglog.go`, `rolling.go`, `redact.go`, `export.go`, `diaglog_test.go`, `export_test.go`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Cross-cutting data changes that both US2 and US3 consume. Must land before those phases begin.

**⚠️ CRITICAL**: US2 and US3 both read `StatusSnapshot` fields added here.

- [x] T002 Add `RecordingOrigin string` and `SessionID string` fields to `StatusSnapshot` struct in `internal/ipc/status.go`

**Checkpoint**: Foundation ready — user story phases 3–6 can now proceed (3 and 4/5/6 can overlap once T006 is done).

---

## Phase 3: User Story 1 — Enable Debug Tracing (Priority: P1) 🎯 MVP

**Goal**: Developer sets `MEMOFY_DEBUG_RECORDING=true` and gets a populated NDJSON rolling log at `/tmp/memofy-debug.log`.

**Independent Test**: `MEMOFY_DEBUG_RECORDING=true go run ./cmd/memofy-core` → connect to OBS → verify `/tmp/memofy-debug.log` contains valid NDJSON lines with `ts`, `component`, `event` fields. Disable flag → verify no file is created.

- [x] T003 [US1] Define `LogEntry` struct and exported event/component name constants in `internal/diaglog/diaglog.go`
- [x] T004 [P] [US1] Implement `rollingWriter` with 10 MB cap (truncate-on-overflow, immediate `Sync` after each write, mutex-guarded) in `internal/diaglog/rolling.go`
- [x] T005 [P] [US1] Implement `Redact(v interface{}) interface{}` recursive deep-redaction of sensitive keys (`authentication`, `password`, `secret`, `challenge`, `salt`, `auth`) in `internal/diaglog/redact.go`
- [x] T006 [US1] Implement `Logger` type with `New(path string) (*Logger, error)`, `Log(entry LogEntry)` (no-op when debug disabled), `Close() error`, and package-level `IsDebugEnabled() bool` reading `MEMOFY_DEBUG_RECORDING` env var in `internal/diaglog/diaglog.go`
- [x] T007 [P] [US1] Write `TestLogWritesNDJSON`, `TestRollingTruncatesAt10MB`, `TestRedactSensitiveFields`, `TestNoOpWhenDisabled` in `internal/diaglog/diaglog_test.go`

**Checkpoint**: US1 complete — `MEMOFY_DEBUG_RECORDING=true` produces a valid NDJSON log. US2, US3, US4 can now start in parallel.

---

## Phase 4: User Story 2 — Identify Who Sent StopRecord (Priority: P2)

**Goal**: Every WS send/receive, reconnect event, and StopRecord command appears in the log with the originating component and a machine-readable reason code.

**Independent Test**: Run mock OBS server, call `StopRecord("user_stop")`, assert log contains `{"event":"ws_send","reason":"user_stop",...}`. Simulate reconnect, assert `ws_reconnect_attempt` entry. Confirm all entries have `component` field.

- [x] T008 [P] [US2] Add `logger *diaglog.Logger` field and `SetLogger(l *diaglog.Logger)` method to `Client` struct in `internal/obsws/client.go`
- [x] T009 [US2] Log `ws_send` (with redacted payload) in `sendRequest()` and `ws_recv` (with redacted payload) in `readMessages()` in `internal/obsws/client.go`
- [x] T010 [US2] Log `ws_connect` after auth success, `ws_disconnect` with close reason in `disconnect()`, `ws_reconnect_attempt` / `ws_reconnect_success` / `ws_reconnect_failed` in `reconnect()`, and `multi_client_warning` on close code 4009 in `internal/obsws/client.go`
- [x] T011 [US2] Add `reason string` parameter to `StopRecord()` in `internal/obsws/operations.go`; update all callers in `cmd/memofy-core/main.go` to pass an explicit reason string
- [x] T012 [P] [US2] Write `TestLogStopRecordEmitsReason`, `TestLogReconnectAttempt`, `TestLogMultiClientWarning` in `internal/obsws/client_test.go`

**Checkpoint**: US2 complete — every StopRecord in the log has a named source component and reason code.

---

## Phase 5: User Story 3 — Protect Manual Recordings (Priority: P3)

**Goal**: A recording started manually can only be stopped by an explicit user action; all automated stop signals are rejected and logged with `"manual_mode_override"`.

**Independent Test**: `ForceStart()` → send auto-origin `StopRecording` request → assert recording still active and log shows `recording_stop_rejected`. Then send user-origin stop → assert recording ends normally.

- [x] T013 [P] [US3] Add `RecordingOrigin` string enum (`OriginManual`, `OriginAuto`, `OriginForced`), `RecordingSession` struct, and `StopRequest` struct to `internal/statemachine/statemachine.go`
- [x] T014 [US3] Add `session *RecordingSession`, `debounceDur time.Duration`, `logger *diaglog.Logger` fields to `StateMachine`; add `SetLogger(l *diaglog.Logger)` and `SetDebounceDuration(d time.Duration)` setter methods; update `ForceStart()` to set `session` with `OriginManual` and a new `crypto/rand` session ID; update `StartRecording()` to set `session` with `OriginAuto` in `internal/statemachine/statemachine.go`
- [x] T015 [US3] Implement `StopRecording(req StopRequest) bool` — reject (log + return false) if `session.Origin == OriginManual` and `req.RequestOrigin != OriginManual`; apply debounce guard on session start; clear session and return true on allowed stop in `internal/statemachine/statemachine.go`
- [x] T016 [US3] Update all existing `StopRecording()` call sites in `cmd/memofy-core/main.go` to pass a `StopRequest` with appropriate `RequestOrigin`, `Reason`, and `Component`
- [x] T017 [P] [US3] Add `SessionOrigin() RecordingOrigin` and `SessionID() string` accessor methods to `StateMachine` in `internal/statemachine/statemachine.go`
- [x] T018 [P] [US3] Write `TestManualSessionBlocksAutoStop`, `TestManualSessionAllowsUserStop`, `TestDebounceRejectsAutoStopEarly` (auto-origin stop within debounce → rejected), `TestDebounceAllowsUserStopEarly` (manual-origin stop within debounce → allowed, per FR-008), `TestAutoSessionAllowsAutoStop`, `TestSessionIDGeneratedOnStart` in `internal/statemachine/statemachine_test.go`

**Checkpoint**: US3 complete — manual recordings are fully protected from automated stops.

---

## Phase 6: User Story 4 — Export Diagnostic Logs (Priority: P4)

**Goal**: `memofy-core --export-diag` writes a complete NDJSON bundle (metadata header + all log lines) to the current directory in under 10 seconds.

**Independent Test**: Seed a temp log file with 10 NDJSON lines → run `memofy-core --export-diag` pointing at it → assert output file line 1 is valid `DiagBundle` JSON with correct `entry_count`, lines 2–11 match source lines exactly.

- [x] T019 [US4] Implement `DiagBundle` struct and `Export(logPath, dest string) (path string, lines int, err error)` (reads source log, counts lines, prepends metadata line, writes `memofy-diag-<timestamp>.ndjson` to dest; returns file path and line count) in `internal/diaglog/export.go`
- [x] T020 [US4] Add `--export-diag` flag detection at top of `main()` in `cmd/memofy-core/main.go`: read `MEMOFY_LOG_PATH`, call `diaglog.Export`, print path to stdout, exit with codes from `contracts/cli.md`
- [x] T021 [P] [US4] Write `TestExportWritesBundleHeader`, `TestExportContainsAllLines`, `TestExportMissingFile`, `TestExportCompletesUnder10s` in `internal/diaglog/export_test.go`

**Checkpoint**: US4 complete — `memofy-core --export-diag` produces a shareable diagnostic bundle.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Wire all components together, fix known latent races, and update documentation.

- [x] T022 Wire `diaglog` logger into `cmd/memofy-core/main.go` startup: read `MEMOFY_LOG_PATH` and `MEMOFY_MANUAL_DEBOUNCE_MS` env vars; call `diaglog.New()`; pass logger to `obsClient.SetLogger()`, `sm.SetLogger()`, `sm.SetDebounceDuration()`
- [x] T023 [P] Populate `StatusSnapshot.RecordingOrigin` and `StatusSnapshot.SessionID` from `sm.SessionOrigin()` and `sm.SessionID()` on each poll tick in `cmd/memofy-core/main.go`
- [x] T024 [P] Add `sync.Mutex` guard to `requestID` counter increment in `sendRequest()` to fix data race in `internal/obsws/client.go`
- [x] T025 [P] Add `// FR-009: intentional — do not add GetRecordStatus or any recording command here` comment to `reconnect()` goroutine in `internal/obsws/client.go`
- [x] T026 [P] Run `go test ./... -race` and resolve any remaining data races surfaced by the race detector
- [x] T027 [P] Update `docs/development/logging.md` with debug mode instructions and `jq` query examples from `specs/002-obs-autostop/quickstart.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately
- **Foundational (Phase 2)**: Requires Phase 1
- **US1 (Phase 3)**: Requires Phase 1; can start in parallel with Phase 2
- **US2 (Phase 4)**: Requires T006 (Logger) from Phase 3 — can start once `Logger.Log()` is importable
- **US3 (Phase 5)**: Requires T006 (Logger) from Phase 3 — can start in parallel with US2
- **US4 (Phase 6)**: Requires T006 (Logger) + T019 (Export) — can start once `diaglog` package compiles
- **Polish (Phase 7)**: Requires all preceding phases

### User Story Dependencies

- **US1 (P1)**: Standalone — only requires package skeleton from Phase 1
- **US2 (P2)**: Requires US1 Logger to be importable; independent of US3
- **US3 (P3)**: Requires US1 Logger to be importable; independent of US2
- **US4 (P4)**: Requires `Export()` from US1 work; independent of US2 and US3

### Parallel Opportunities Per Story

```
# US1 (after T003 defines LogEntry):
T004 (rolling.go) ──┐
T005 (redact.go)  ──┤──→ T006 (Logger) → T007 (tests)
                    │
# US2 + US3 (after T006):
T008 (SetLogger)  ──┐
T009 (ws_send)    ──┤ (sequential within client.go)   T013 (types) ──┐
T010 (reconnect)  ──┤                                 T014 (session)─┤
T011 (StopRecord) ──┤                                 T015 (authority)┤
T012 (tests) [P]  ──┘                                 T016 (callers) ─┤
                                                       T017 (accessors)┤[P]
# US4 (after T006):                                   T018 (tests) [P]┘
T019 → T020 → T021 [P]
```

---

## Implementation Strategy

**Suggested MVP scope**: Complete Phase 1 + Phase 2 + Phase 3 (US1) first. This delivers working diagnostic logging that can immediately be used to investigate the live bug — without waiting for the safeguard (US3) to be complete.

**Parallel stream recommendation** (if two developers):
- Stream A: Phase 1 → Phase 3 (US1) → Phase 4 (US2) → Phase 7 wiring
- Stream B: Phase 2 → Phase 5 (US3) → Phase 6 (US4) → Phase 7 polish

Both streams converge at Phase 7.

---

## Task Count Summary

| Phase | Tasks | User Story |
|-------|-------|-----------|
| Phase 1: Setup | 1 | — |
| Phase 2: Foundational | 1 | — |
| Phase 3: US1 Debug Tracing | 5 | P1 |
| Phase 4: US2 Stop Attribution | 5 | P2 |
| Phase 5: US3 Manual Protection | 6 | P3 |
| Phase 6: US4 Diagnostic Export | 3 | P4 |
| Phase 7: Polish | 6 | — |
| **Total** | **27** | |

**Parallel opportunities**: 14 of 27 tasks marked [P]
