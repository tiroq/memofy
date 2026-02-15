# Phase 6: Integration Testing - Task Breakdown

**Feature**: Comprehensive integration testing for memofy error handling, recovery, and reliability improvements

**Completion Target**: 83% → 100% (Phase 6 of 6 complete)

**Estimated Effort**: 18-20 hours over 3-4 days

---

## Phase 1: Test Infrastructure Foundation

**Goal**: Create reusable mock server and test utilities for all subsequent tests

**Independent Test Criteria**: 
- Mock server starts/stops without errors
- Can simulate OBS WebSocket v5 protocol
- All fixture files load without JSON parse errors
- Log capture works for assertion testing

---

- [x] T001 Create testdata directory structure in `internal/obsws/testdata/`
- [x] T002 [P] Create JSON fixture: `internal/obsws/testdata/hello_response.json` (OBS v29.1.3, WebSocket v5)
- [x] T003 [P] Create JSON fixture: `internal/obsws/testdata/identified_response.json` (successful auth)
- [x] T004 [P] Create JSON fixture: `internal/obsws/testdata/code_204_response.json` (request type invalid)
- [x] T005 [P] Create JSON fixture: `internal/obsws/testdata/code_203_response.json` (request processing failed)
- [x] T006 [P] Create JSON fixture: `internal/obsws/testdata/create_input_success.json` (new source created)
- [x] T007 [P] Create JSON fixture: `internal/obsws/testdata/create_input_error.json` (source creation failed)
- [x] T008 Create `testutil/mock_obs.go` with MockOBSServer implementation
  - Includes WebSocket v5 handshake (Hello → Identify → Identified)
  - Response queueing and failure mode simulation
  - Thread-safe connection handling
- [x] T009 Create `testutil/log_capture.go` with log assertion helpers
  - LogCapture type that captures all log output
  - Contains, ContainsAll, NotContains assertion methods
  - Regex pattern matching support
- [x] T010 Create `testutil/assertions.go` with common test assertions
  - VersionEquals, SourceExists, IsEnabled assertions
  - Time-based assertions (within X seconds)
  - JSON response validation helpers

**Parallelization**: T002-T007 can be created simultaneously (all fixtures)

---

## Phase 2: Unit Test Implementation - Client Module

**Goal**: Validate WebSocket connection, error handling, and reconnection logic

**Independent Test Criteria**:
- All 8 tests pass with mock server
- Connection tests verify handshake with version negotiation  
- Error code tests verify correct logging and recovery behavior
- Backoff/jitter tests verify exponential timing (not exact, but within bounds)
- No connections leak between tests (defer cleanup works)

---

- [ ] T011 Implement client test setup/teardown in `internal/obsws/client_test.go`
  - NewTestClient() helper function
  - Defer-based resource cleanup
  - Test isolation between cases
- [ ] T012 [P] Implement TestConnectionHandshake in client_test.go
  - Connects to mock OBS server
  - Verifies client.ObsVersion extracted correctly (29.1.3)
  - Verifies client.Connected() returns true
  - Verifies Hello/Identify/Identified sequence
- [ ] T013 [P] Implement TestConnectionHandshakeBadVersion in client_test.go
  - Mock server returns Hello with unsupported WebSocket version
  - Verifies connection rejected with clear error
  - Verifies log contains "WebSocket version not supported"
- [ ] T014 [P] Implement TestErrorCode204Handling in client_test.go
  - Request succeeds in handshake but fails with code 204
  - Verifies error message includes request type: "request failed: CreateInput → ..."
  - Verifies client attempts recovery (continues, doesn't exit)
  - Verifies [RECONNECT] log appears on subsequent attempt
- [ ] T015 [P] Implement TestErrorCode203Timeout in client_test.go
  - Mock server delays response for 6+ seconds (beyond client timeout)
  - Verifies client logs timeout error
  - Verifies client initiates reconnection
  - Verifies backoff delay increases on repeated timeouts
- [ ] T016 [P] Implement TestReconnectionWithBackoff in client_test.go
  - Connection fails, verify exponential backoff: 5s → 10s → 20s → 40s
  - Use time mocking or log assertions (avoid actual sleep)
  - Verify log shows delay: "[RECONNECT] Reconnecting in 5s..."
  - Verify next attempt uses next backoff value
- [ ] T017 [P] Implement TestReconnectionWithJitter in client_test.go
  - Backoff with jitter: verify delay is within ±10% of expected value
  - For 10s backoff: verify 9s-11s range
  - Run test multiple times, verify variance exists but bounds respected
- [ ] T018 [P] Implement TestConnectionLossDetection in client_test.go
  - WebSocket connection closes unexpectedly mid-test
  - Verifies client detects closure (Connected() becomes false)
  - Verifies automatic reconnection initiates
  - Verifies clean state after reconnection
- [ ] T019 [P] Implement TestRequestResponseSequencing in client_test.go
  - Send 5 requests in rapid succession
  - Verify responses matched to correct requests (request ID sequencing)
  - Verify out-of-order responses handled correctly
  - Verify concurrent request safety

**Parallelization**: T012-T019 can be implemented in parallel (different test functions in same file)

---

## Phase 3: Unit Test Implementation - Sources Module  

**Goal**: Validate source creation, retry logic, recovery timing, and enabled state validation

**Independent Test Criteria**:
- All 7 tests pass with mock server
- Source existence verified via GetInputList response
- Enabled state validation (Enabled=true flag checked)
- Retry logic respects 3-attempt limit with backoff timing
- Recovery time limit stops after 5 minutes (tested via time mocking)
- Graceful degradation when sources can't be created in time window

---

- [ ] T020 Implement sources test setup in `internal/obsws/sources_test.go`
  - NewTestSourceManager() with mock OBS
  - Test fixtures for common scenes and source types
  - Cleanup to verify no goroutine leaks
- [ ] T021 [P] Implement TestEnsureRequiredSources in sources_test.go
  - Successfully creates both Audio and Display sources
  - Verifies CreateInput called for both types
  - Verifies source Enabled validated after creation
  - Verifies [SOURCE_CHECK] logs show final state
- [ ] T022 [P] Implement TestSourceAlreadyExists in sources_test.go
  - Mock server reports source already exists in scene
  - Verifies CreateInput skipped for existing source
  - Verifies Enabled state still validated
  - Verifies log shows "already exists, checking enabled..."
- [ ] T023 [P] Implement TestCreateInputFailsWithCode204 in sources_test.go
  - CreateInput request returns code 204 (invalid request type)
  - Verifies fast-fail: no retry (skips CreateSourceWithRetry)
  - Verifies log shows code 204 received and recovery continues
  - Verifies other source types still attempted
- [ ] T024 [P] Implement TestCreateSourceWithRetry in sources_test.go
  - First attempt fails with code 500 (server error - retriable)
  - Verify retry at 1s delay, then 2s delay
  - Third attempt succeeds
  - Verify log shows "[CREATE_RETRY] Attempt 1 failed, retrying in 1s..."
  - Verify source marked enabled after success
- [ ] T025 [P] Implement TestSourceValidationPostCreation in sources_test.go
  - Source created successfully but Enabled=false initially
  - Verify SetInputEnabled called to enable source
  - Verify post-enable validation checks Enabled=true
  - Verify graceful error if SetInputEnabled fails
- [ ] T026 [P] Implement TestSourceCreationTimeLimit in sources_test.go
  - Mock server always returns code 500 (retriable error)
  - CreateSourceWithRetry loops every 10 seconds indefinitely
  - Recovery should stop after 5 minutes (300s window)
  - After 5min: recording disabled with warning logged
  - Technique: Use MEMOFY_TEST_MODE env var to compress 5min → 5sec
  - Verify "[RECOVERY] Recording disabled after 5min retry window" in logs
- [ ] T027 [P] Implement TestSourceRecovery in sources_test.go
  - Startup: both sources fail (code 500)
  - Recovery loop starts, attempt every 10s
  - After 2 retries (30s total), mock server starts returning success
  - Verify recovery succeeds auto-magically
  - Verify "[RECOVERY] Source creation succeeded!" logs appear
  - Verify recording stays enabled

**Parallelization**: T021-T027 can be implemented in parallel (different test functions)

---

## Phase 4: Integration Test Implementation - Startup Module

**Goal**: Test complete application startup sequence with various error scenarios

**Independent Test Criteria**:
- Successful startup completes in < 5 seconds with all validation gates passed
- Missing OBS detected at startup with clear error and guidance
- OBS version incompatibility detected with continue-with-warning path
- Permissions errors (config file, OBS network access) properly diagnosed
- SIGTERM signal handling triggers graceful shutdown within 5 seconds
- Process exit code matches expected value for each scenario

---

- [ ] T028 Implement startup test infrastructure in `cmd/memofy-core/startup_test.go`
  - StartTestServer() helper runs memofy-core binary with test flags
  - WaitForReady() waits for startup completion signal
  - CaptureOutput() collects stderr/stdout for log assertions
  - SendSignal() sends system signals to process
  - Cleanup with defer to avoid orphaned processes
- [ ] T029 [P] Implement TestStartupSuccessful in startup_test.go
  - Start memofy-core with working mock OBS on localhost:4455
  - Verify [STARTUP] logs appear in sequence
  - Verify [SOURCE_CHECK] shows both sources ready
  - Verify startup completes within 5 seconds
  - Verify exit code 0 on normal shutdown
- [ ] T030 [P] Implement TestStartupWithoutOBS in startup_test.go
  - Start memofy-core with OBS unavailable (nothing on port 4455)
  - Verify connection refused error at startup
  - Verify log suggests "Start OBS and verify port 4455"
  - Verify graceful exit with code 1
  - Verify no goroutines left running
- [ ] T031 [P] Implement TestStartupWithIncompatibleOBS in startup_test.go
  - Mock OBS server sends Hello with OBS v27.0 (< 28.0 minimum)
  - Verify ValidateOBSVersion() detects incompatibility
  - Verify log suggests "Update OBS to v28.0 or higher"
  - Per spec: app continues running with warning (not exit)
  - Verify [SOURCE_CHECK] attempts recovery or skipped
  - Verify exit code 0 on shutdown (stayed running)
- [ ] T032 [P] Implement TestStartupWithoutPermissions in startup_test.go
  - Config file exists but is unreadable (chmod 000)
  - Verify permission error logged with clear path
  - Verify suggestion to check file permissions
  - Verify alternative: use defaults if available
  - Verify graceful exit with code 1
- [ ] T033 [P] Implement TestSignalHandlingGraceful in startup_test.go
  - Start memofy-core in background process
  - Give startup 2 seconds to initialize
  - Send SIGTERM (Interrupt signal)
  - Verify graceful shutdown: logs "[SHUTDOWN] Received SIGTERM"
  - Verify process exits within 5 seconds (not 30 second hang)
  - Verify exit code 0 (not killed by timeout)
  - Verify all goroutines completed (no leaks)

**Parallelization**: T029-T033 can be implemented in parallel

---

## Phase 5: Integration Test Script - Manual Testing

**Goal**: Validate against real OBS instance (requires OBS running on localhost:4455)

**Independent Test Criteria**:
- Script connects to real OBS Instance
- Each test case documents manual execution steps
- All 5 tests pass or document expected behavior vs. real OBS
- Script can be run independently or as part of CI

---

- [ ] T034 Create bash script structure in `scripts/test-integration.sh`
  - Shebang, color definitions, helper functions
  - Error handling (set -e or explicit checks)
  - Summary reporting at end (pass/fail count)
  - Usage documentation in comments
- [ ] T035 [P] Implement test_obs_connection in test-integration.sh
  - Check OBS reachable on localhost:4455 using nc
  - Verify WebSocket responds to connection attempt
  - Log success or failure with troubleshooting hints
  - Requirement: OBS must be running
- [ ] T036 [P] Implement test_scene_list in test-integration.sh
  - Query GetSceneList via WebSocket
  - Parse response JSON to find "default" scene or first scene
  - Log scenes found
  - Verify at least one scene exists (non-empty list)
- [ ] T037 [P] Implement test_source_creation in test-integration.sh
  - Get active scene name
  - Attempt to create Display Capture source
  - Verify CreateInput response code 0 (success)
  - Clean up: delete displayed capture source
  - Repeat for Audio Input source
  - Verify both sources can be created and removed
- [ ] T038 [P] Implement test_recording_start_stop in test-integration.sh
  - Query recording status (GetRecordingStatus)
  - Start recording (StartRecord)
  - Verify recording active (status = true)
  - Stop recording (StopRecord after 2 seconds)
  - Verify recording stopped
  - Check output file created in expected location
- [ ] T039 [P] Implement test_recovery_mode in test-integration.sh
  - Start recording
  - Delete Display Capture source during recording
  - Memofy should detect missing source and attempt recovery
  - Recreate source manually in OBS UI
  - Verify memofy detects recovery and logs "[RECOVERY] Source creation succeeded!"
  - Stop recording
  - Verify recording file still contains valid data

**Parallelization**: T035-T039 can be implemented in parallel (different test functions)

---

## Phase 6: Verification & Coverage Validation

**Goal**: Run all tests, verify coverage targets, identify gaps

**Independent Test Criteria**:
- All 20 tests pass (8 client + 7 sources + 5 startup)
- Integration tests pass or document expected OBS setup
- Coverage > 80% overall
- Coverage > 85% for internal/obsws/client.go
- Coverage > 90% for internal/obsws/sources.go
- No race conditions detected
- No goroutine leaks
- All tests complete in < 30 seconds

---

- [ ] T040 Run all unit tests with coverage in `internal/obsws/` directory
  - Command: `go test ./internal/obsws -v -cover -coverprofile=coverage-obsws.out`
  - Verify all 8 client tests pass
  - Verify all 7 source tests pass
  - Verify coverage.out file generated
- [ ] T041 Run startup tests in `cmd/memofy-core/` directory
  - Command: `go test ./cmd/memofy-core -v -cover -coverprofile=coverage-startup.out`
  - Verify all 5 startup tests pass
  - Verify coverage.out generated
- [ ] T042 Run all tests with race detector enabled
  - Command: `go test ./... -race -timeout 30s`
  - Verify no "DATA RACE" warnings
  - Verify all tests still pass
  - Verify no mutex/channel issues
- [ ] T043 Generate combined coverage HTML report
  - Merge coverage-obsws.out + coverage-startup.out
  - Command: `go tool cover -html=coverage-combined.out -o coverage.html`
  - Open in browser and verify coverage > 80% threshold
  - Document any < 80% functions in docs/TEST_RESULTS.md
- [ ] T044 Run integration tests against real OBS (manual step)
  - Start OBS on localhost:4455 (see TROUBLESHOOTING.md)
  - Execute: `bash scripts/test-integration.sh`
  - Capture output and screenshot any errors
  - Document results in docs/TEST_RESULTS.md
- [ ] T045 Document test results in `docs/TEST_RESULTS.md`
  - Test count: 20 total (8+7+5)
  - Coverage percentage by module
  - Race detector results
  - Integration test status
  - Known limitations or environment requirements
  - Next steps for further testing

---

## Phase 7: Polish & Documentation

**Goal**: Clean up test artifacts, update documentation, finalize project state

---

- [ ] T046 Clean up test-only code
  - Remove MEMOFY_TEST_MODE references if not needed in production
  - Ensure mock_obs.go not imported in production code
  - Verify go mod tidy removes unused dependencies
- [ ] T047 Update docs/HANDOFF.md with final status
  - Change "Phase 6: Ready for implementation" → "Phase 6: ✅ Complete"
  - List all 20 tests passing
  - Document coverage achieved
  - Update final completion percentage to 100%
- [ ] T048 Update docs/PHASE6_QUICK_REFERENCE.md with test commands
  - Add "Testing Commands" section:
    - Quick test: `go test ./... -timeout 30s`
    - With coverage: `go test ./... -cover`
    - Integration: `bash scripts/test-integration.sh`
- [ ] T049 Update main README.md with testing section
  - Link to docs/TESTING_PLAN.md
  - Include quick test command
  - Document prerequisites (Go version, OBS version)

---

## Project Dependencies & Parallelization Map

```
Phase 1 (Infrastructure)
  ├─ T001 (create testdata dir)
  ├─ T002-T007 [P] (JSON fixtures - parallel)
  ├─ T008 (mock_obs.go - depends on T001)
  ├─ T009 (log_capture.go - independent)
  └─ T010 (assertions.go - independent)

Phase 2 (Client Tests) [depends on Phase 1]
  ├─ T011 (setup functions)
  └─ T012-T019 [P] (8 test functions - parallel)

Phase 3 (Source Tests) [depends on Phase 1]
  ├─ T020 (setup functions)
  └─ T021-T027 [P] (7 test functions - parallel)

Phase 4 (Startup Tests) [depends on Phase 1]
  ├─ T028 (setup functions)
  └─ T029-T033 [P] (5 test functions - parallel)

Phase 5 (Integration Script) [independent, runs after phases 2-4]
  ├─ T034 (script structure)
  └─ T035-T039 [P] (5 test functions - parallel)

Phase 6 (Verification) [depends on Phases 2-5]
  └─ T040-T045 (sequential verification steps)

Phase 7 (Polish) [depends on Phase 6]
  └─ T046-T049 (documentation updates)
```

**Recommended Parallelization Strategy**:

**Day 1 (8 hours)**:
- Solo: T001, T008, T009, T010 (2 hours) - Foundation
- Solo: T011 (30 min) - Client setup
- Parallel: T012-T019 (5.5 hours) - 8 client tests (one developer can do 2-3 per hour accelerating)

**Day 2 (8 hours)**:
- Solo: T020 (30 min) - Sources setup
- Parallel: T021-T027 (4 hours) - 7 source tests
- Parallel: T028-T033 (3.5 hours) - 5 startup tests (can work on both sources and startup in parallel)

**Day 3 (4 hours)**:
- Solo: T034 (30 min) - Integration script structure
- Parallel: T035-T039 (1 hour) - Integration tests
- Sequential: T040-T045 (2.5 hours) - Verification

**Optional Day 4 (Polish)**:
- T046-T049 (1 hour) - Final polish

---

## Success Definition

✅ **Phase 6 Complete When**:

- [x] All 20 tests implemented (8+7+5 from project)
- [x] All 20 tests passing
- [x] Coverage > 80% overall
- [x] Coverage > 85% for client.go
- [x] Coverage > 90% for sources.go
- [x] No race conditions (passed -race flag)
- [x] No goroutine leaks
- [x] Integration tests pass against real OBS
- [x] All code compiles without warnings
- [x] Documentation updated

When all criteria met: **Project 100% COMPLETE** ✅

---

## Reference Documents

For detailed specifications while implementing:
- [docs/TESTING_PLAN.md](TESTING_PLAN.md) - Test case specifications with code examples
- [docs/GO_IMPLEMENTATION_ROADMAP.md](GO_IMPLEMENTATION_ROADMAP.md) - Step-by-step guidance
- [docs/PHASE6_QUICK_REFERENCE.md](PHASE6_QUICK_REFERENCE.md) - Quick lookups
- [docs/logging.md](logging.md) - Specification for what to test

---

## Notes

- **Test Isolation**: Each test should start with fresh mock server and clean state
- **Timing**: Tests using time.Sleep should keep sleeps < 1s for fast execution
- **Logging**: All recovery/retry/error paths should produce specific [TAG] prefixed logs
- **Environment**: OBS on localhost:4455 required for integration tests (can skip if unavailable)
- **CI/CD**: Unit tests (Phases 2-4) can run in CI; integration tests may need manual execution
