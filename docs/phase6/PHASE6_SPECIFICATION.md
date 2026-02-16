# Phase 6: Integration Testing - Feature Specification

**Document Version**: 1.0  
**Created**: 2026-02-14  
**Status**: Ready for Implementation  
**Completion Target**: 83% → 100% (Final phase)

---

## Executive Summary

### What This Feature Delivers

A comprehensive integration testing framework for the memofy application that validates error handling, automatic recovery mechanisms, and system reliability improvements implemented in Phases 1-5. This testing suite ensures production-ready quality with 80%+ code coverage and validates all error scenarios through automated and manual testing.

### User Value

**For Developers**:
- Confidence that error recovery works as specified
- Automated regression prevention for all 6 improvement phases
- Clear test specifications with code examples for maintenance
- Fast test execution (< 30 seconds for full suite)

**For End Users**:
- Reliable application behavior even when OBS is unavailable or misconfigured
- Predictable recovery from connection failures and missing sources
- Clear error messages with actionable troubleshooting steps
- Stable recording functionality without unexpected crashes

### Success Metrics

- ✅ All 20 automated tests passing
- ✅ Code coverage > 80% overall
- ✅ Coverage > 85% for `internal/obsws/client.go`
- ✅ Coverage > 90% for `internal/obsws/sources.go`
- ✅ Zero race conditions detected
- ✅ Zero goroutine leaks
- ✅ Integration tests validate against real OBS

---

## Feature Scope

### In Scope

**Test Infrastructure**:
- Mock OBS WebSocket server supporting v5 protocol
- JSON fixture files for common OBS responses
- Log capture utilities for assertion testing
- Test helper functions for common patterns

**Unit Tests** (15 tests):
- 8 client tests: WebSocket connection, error codes, reconnection logic
- 7 source tests: Creation, retry, validation, recovery timing

**Integration Tests** (5 tests):
- Startup sequence validation with various error scenarios
- Signal handling (SIGTERM graceful shutdown)
- OBS version compatibility detection

**Manual Testing** (5 tests):
- Bash script against real OBS instance
- Connection, scene listing, source creation
- Recording start/stop, recovery mode validation

**Documentation**:
- Test results summary document
- Updated handoff with 100% completion status
- Testing commands in quick reference guide

### Out of Scope

**Not Included in Phase 6**:
- Performance benchmarking or load testing
- UI automation testing (manual verification only)
- Cross-platform testing (macOS focus)
- Network failure simulation (WebSocket disconnect covered)
- Multi-user or concurrent session testing
- OBS plugin compatibility testing beyond v28.0+

---

## Functional Requirements

### FR1: Mock OBS Server Infrastructure

**Requirement**: Create reusable mock WebSocket server that simulates OBS v29.1.3 behavior

**Acceptance Criteria**:
- [ ] Mock server implements WebSocket v5 handshake sequence (Hello → Identify → Identified)
- [ ] Supports response queueing for predictable test behavior
- [ ] Can simulate failure modes: code 204, code 203, connection timeout, unexpected disconnect
- [ ] Thread-safe connection handling for concurrent test execution
- [ ] Starts/stops cleanly without port conflicts between tests
- [ ] Validates all JSON responses parse correctly with no errors

**Technical Details**:
- File: `testutil/mock_obs.go`
- Methods: `Start()`, `Stop()`, `SetResponseMode()`, `QueueResponse()`
- Protocol: OBS WebSocket v5 (JSON-RPC 2.0)
- Port: Dynamic allocation to avoid conflicts

---

### FR2: JSON Test Fixtures

**Requirement**: Provide realistic OBS response fixtures for test isolation

**Acceptance Criteria**:
- [ ] `hello_response.json` - OBS v29.1.3, WebSocket v5.0.5
- [ ] `identified_response.json` - Successful authentication
- [ ] `code_204_response.json` - Request type invalid error
- [ ] `code_203_response.json` - Request processing timeout
- [ ] `create_input_success.json` - New source created with Enabled=true
- [ ] `create_input_error.json` - Source creation failed with reason
- [ ] All fixtures validate against JSON schema

**Technical Details**:
- Location: `internal/obsws/testdata/*.json`
- Format: Valid JSON-RPC 2.0 responses
- Versioning: Match OBS WebSocket v5 specification

---

### FR3: Client Unit Tests (8 tests)

**Requirement**: Validate WebSocket client behavior for all connection scenarios

**Test Cases**:

#### T3.1: TestConnectionHandshake
**Scenario**: Successful connection with version extraction
- Given: Mock OBS server responding with Hello/Identified
- When: Client connects to localhost:4455
- Then: 
  - Connection establishes successfully
  - `client.ObsVersion` == "29.1.3"
  - `client.WebSocketVersion` == "5.0.5"
  - `client.Connected()` returns `true`

#### T3.2: TestConnectionHandshakeBadVersion
**Scenario**: Reject unsupported WebSocket version
- Given: Mock server returns WebSocket v4.x in Hello
- When: Client attempts connection
- Then:
  - Connection rejected with error
  - Log contains "WebSocket version not supported"
  - `client.Connected()` returns `false`

#### T3.3: TestErrorCode204Handling
**Scenario**: Continue operation on OBS version incompatibility
- Given: Mock server returns code 204 for CreateInput request
- When: Client sends CreateInput
- Then:
  - Error message includes request type: "request failed: CreateInput → code 204"
  - Client does NOT exit (continues running)
  - Log shows recovery attempt: "[RECONNECT]" appears in output
  - **Per Spec**: App continues with warning (doesn't disable recording)

#### T3.4: TestErrorCode203Timeout
**Scenario**: Handle request processing timeout
- Given: Mock server delays response > 6 seconds
- When: Client waits for response
- Then:
  - Timeout error logged after 6 seconds
  - Client initiates reconnection
  - Backoff delay increases on repeated timeouts

#### T3.5: TestReconnectionWithBackoff
**Scenario**: Exponential backoff on connection failure
- Given: Connection fails repeatedly
- When: Client attempts reconnection
- Then:
  - Delays follow sequence: 5s → 10s → 20s → 40s → 60s (max)
  - Log shows delay: "[RECONNECT] Reconnecting in 10s..."
  - Each retry uses next backoff value

#### T3.6: TestReconnectionWithJitter
**Scenario**: Jitter prevents thundering herd
- Given: Backoff delay is 10 seconds
- When: Random jitter applied (±10%)
- Then:
  - Actual delay is within 9s-11s range (90%-110%)
  - Multiple test runs show variance in delays
  - All delays respect ±10% bounds

#### T3.7: TestConnectionLossDetection
**Scenario**: Detect and recover from unexpected disconnect
- Given: Active WebSocket connection
- When: Server closes connection unexpectedly
- Then:
  - `client.Connected()` returns `false` immediately
  - Automatic reconnection initiates
  - Clean state restored after reconnection

#### T3.8: TestRequestResponseSequencing
**Scenario**: Verify request/response matching with concurrent requests
- Given: 5 rapid requests sent simultaneously
- When: Responses arrive (possibly out of order)
- Then:
  - Each response matched to correct request via request ID
  - Out-of-order responses handled correctly
  - No race conditions detected
  - All requests complete successfully

**Coverage Target**: > 85% for `internal/obsws/client.go`

---

### FR4: Source Unit Tests (7 tests)

**Requirement**: Validate source creation, retry logic, and recovery mechanisms

**Test Cases**:

#### T4.1: TestEnsureRequiredSources
**Scenario**: Successfully create both required sources
- Given: Mock OBS with empty scene
- When: `EnsureRequiredSources()` called
- Then:
  - `CreateInput` called for Display Capture
  - `CreateInput` called for Audio Input
  - Both sources have `Enabled=true` validated
  - Log shows "[SOURCE_CHECK] Audio: exists=true, enabled=true"
  - Log shows "[SOURCE_CHECK] Display: exists=true, enabled=true"

#### T4.2: TestSourceAlreadyExists
**Scenario**: Skip creation if source already exists
- Given: Mock OBS scene already has "Memofy Display Capture"
- When: `EnsureRequiredSources()` called
- Then:
  - `CreateInput` skipped for existing source
  - `Enabled` state still validated (via `GetInputSettings`)
  - Log shows "already exists, checking enabled..."

#### T4.3: TestCreateInputFailsWithCode204
**Scenario**: Fast-fail on OBS version incompatibility
- Given: Mock server returns code 204 for CreateInput
- When: Source creation attempted
- Then:
  - No retry attempted (fast-fail for code 204)
  - Log shows "code 204: OBS version incompatible"
  - **Per Spec**: App continues running (doesn't disable recording)
  - Other source types still attempted

#### T4.4: TestCreateSourceWithRetry
**Scenario**: Retry on temporary server errors
- Given: First 2 attempts return code 500, 3rd succeeds
- When: `CreateSourceWithRetry()` called
- Then:
  - Attempt 1 fails, retry after 1s
  - Attempt 2 fails, retry after 2s
  - Attempt 3 succeeds
  - Log shows "[CREATE_RETRY] Attempt 1 failed, retrying in 1s..."
  - Source marked enabled after success

#### T4.5: TestSourceValidationPostCreation
**Scenario**: Enable source if created but disabled
- Given: Source created successfully but `Enabled=false`
- When: Post-creation validation runs
- Then:
  - `SetInputEnabled` called to enable source
  - Validation checks `Enabled=true` after enablement
  - Graceful error if `SetInputEnabled` fails

#### T4.6: TestSourceCreationTimeLimit
**Scenario**: Stop retry after 5-minute recovery window
- Given: Mock server always returns code 500 (retriable)
- When: Recovery loop runs for 5+ minutes
- Then:
  - **Per Spec**: Retries every 10s for 5 minutes max
  - After 5 min (300s), recording disabled with warning
  - Log shows "[RECOVERY] Recording disabled after 5min retry window"
  - **Test Mode**: Use `MEMOFY_TEST_MODE=1` to compress 5min → 5sec

#### T4.7: TestSourceRecovery
**Scenario**: Auto-recovery when sources become available
- Given: Startup fails (code 500), recovery loop started
- When: After 30s, mock server returns success
- Then:
  - Recovery succeeds automatically
  - Log shows "[RECOVERY] Source creation succeeded!"
  - Recording stays enabled
  - No manual intervention required

**Coverage Target**: > 90% for `internal/obsws/sources.go`

---

### FR5: Startup Integration Tests (5 tests)

**Requirement**: Validate end-to-end application startup with error scenarios

**Test Cases**:

#### T5.1: TestStartupSuccessful
**Scenario**: Clean startup with working OBS
- Given: Mock OBS running on localhost:4455
- When: `memofy-core` starts
- Then:
  - [STARTUP] logs appear in sequence
  - [SOURCE_CHECK] shows both sources ready
  - Startup completes within 5 seconds
  - Exit code 0 on shutdown

#### T5.2: TestStartupWithoutOBS
**Scenario**: Detect missing OBS at startup
- Given: No service on port 4455
- When: `memofy-core` starts
- Then:
  - Connection refused error logged
  - Message: "Start OBS and verify port 4455"
  - Graceful exit with code 1
  - No goroutines left running

#### T5.3: TestStartupWithIncompatibleOBS
**Scenario**: Detect OBS version < 28.0
- Given: Mock OBS returns version 27.0 in Hello
- When: `memofy-core` validates OBS
- Then:
  - `ValidateOBSVersion()` detects incompatibility
  - Log suggests "Update OBS to v28.0 or higher"
  - **Per Spec**: App continues running with warning (doesn't exit)
  - [SOURCE_CHECK] attempts recovery or skipped
  - Exit code 0 on shutdown (stayed running)

#### T5.4: TestStartupWithoutPermissions
**Scenario**: Handle config file permission errors
- Given: Config file exists but unreadable (chmod 000)
- When: `memofy-core` loads config
- Then:
  - Permission error logged with file path
  - Suggestion: "Check file permissions"
  - Alternative: Use defaults if available
  - Graceful exit with code 1

#### T5.5: TestSignalHandlingGraceful
**Scenario**: SIGTERM triggers clean shutdown
- Given: `memofy-core` running in background
- When: SIGTERM sent after 2s of initialization
- Then:
  - Log shows "[SHUTDOWN] Received SIGTERM"
  - Process exits within 5 seconds (not 30s timeout)
  - Exit code 0 (not killed by force)
  - All goroutines completed (no leaks detected)

**Coverage Target**: > 70% for `cmd/memofy-core/main.go`

---

### FR6: Manual Integration Testing Script

**Requirement**: Bash script for manual validation against real OBS

**Test Cases**:

#### T6.1: test_obs_connection
**Scenario**: Verify OBS reachable on localhost:4455
- Command: `nc -zv localhost 4455`
- Expected: Connection succeeds
- Failure: "OBS not running on port 4455"

#### T6.2: test_scene_list
**Scenario**: Query scenes from running OBS
- Request: `GetSceneList` via WebSocket
- Expected: JSON response with ≥1 scene
- Validation: Parse JSON, log scene names

#### T6.3: test_source_creation
**Scenario**: Create and delete test sources
- Steps:
  1. Get active scene name
  2. Create Display Capture source
  3. Verify `CreateInput` response code 0
  4. Delete source (cleanup)
  5. Repeat for Audio Input
- Expected: Both sources create/delete successfully

#### T6.4: test_recording_start_stop
**Scenario**: Validate recording lifecycle
- Steps:
  1. Query `GetRecordingStatus`
  2. Start recording (`StartRecord`)
  3. Verify status = true
  4. Wait 2 seconds
  5. Stop recording (`StopRecord`)
  6. Verify output file created
- Expected: All operations succeed, file exists

#### T6.5: test_recovery_mode
**Scenario**: Validate auto-recovery when source restored
- Steps:
  1. Start recording
  2. Delete Display Capture source manually
  3. Memofy detects missing source
  4. Recreate source in OBS UI
  5. Verify memofy logs "[RECOVERY] Source creation succeeded!"
  6. Stop recording
- Expected: Recording continues, file contains valid data

**File**: `scripts/test-integration.sh`
**Prerequisites**: OBS running on localhost:4455
**Exit Codes**: 0 = all pass, 1 = any failure

---

### FR7: Test Verification & Coverage

**Requirement**: Automated verification of test quality and coverage

**Acceptance Criteria**:

#### T7.1: Unit Test Execution
- Command: `go test ./internal/obsws -v -cover -coverprofile=coverage-obsws.out`
- Expected:
  - All 15 tests pass (8 client + 7 sources)
  - Coverage file generated
  - No test timeouts (< 30s total)

#### T7.2: Integration Test Execution
- Command: `go test ./cmd/memofy-core -v -cover -coverprofile=coverage-startup.out`
- Expected:
  - All 5 tests pass
  - Coverage file generated
  - Clean process shutdown in each test

#### T7.3: Race Condition Detection
- Command: `go test ./... -race -timeout 30s`
- Expected:
  - Zero "DATA RACE" warnings
  - All tests still pass with race detector enabled
  - No mutex/channel deadlocks

#### T7.4: Coverage Report Generation
- Commands:
  ```bash
  # Merge coverage files
  go tool covdata merge -i=coverage-obsws.out,coverage-startup.out -o=coverage-combined.out
  
  # Generate HTML
  go tool cover -html=coverage-combined.out -o=coverage.html
  ```
- Expected:
  - Combined coverage > 80%
  - `client.go` coverage > 85%
  - `sources.go` coverage > 90%
  - Report opens in browser without errors

#### T7.5: Integration Test Execution
- Command: `bash scripts/test-integration.sh`
- Prerequisites: OBS running on localhost:4455
- Expected:
  - All 5 tests pass or document expected behavior
  - Clear pass/fail output with colors
  - Results documented in `docs/TEST_RESULTS.md`

---

## User Scenarios & Testing

### Scenario 1: Developer Runs Full Test Suite

**Context**: Developer preparing to commit code changes

**Steps**:
1. Run unit tests: `go test ./... -v -timeout 30s`
2. Check for races: `go test ./... -race`
3. Generate coverage: `go test ./... -coverprofile=coverage.out`
4. View report: `go tool cover -html=coverage.out`

**Expected Outcome**:
- All 20 tests pass in < 30 seconds
- No race conditions detected
- Coverage report shows > 80% overall
- Clear indication of which tests passed/failed

**Success Criteria**:
- Exit code 0 from all test commands
- Coverage HTML opens without errors
- No goroutine leak warnings

---

### Scenario 2: CI/CD Pipeline Runs Automated Tests

**Context**: Pull request triggers automated testing

**Steps**:
1. CI runs: `go test ./internal/obsws ./cmd/memofy-core -v -race -timeout 30s`
2. Coverage calculated: `go test ./... -coverprofile=coverage.out`
3. Results uploaded to coverage service

**Expected Outcome**:
- Pipeline passes if all tests pass
- Coverage badge shows > 80%
- Failed tests display clear error messages

**Success Criteria**:
- CI job completes in < 2 minutes
- Artifacts (coverage.out) available for download
- Clear failure diagnostics if any test fails

---

### Scenario 3: QA Runs Manual Integration Tests

**Context**: Pre-release validation against real OBS

**Steps**:
1. Start OBS on localhost:4455
2. Run: `bash scripts/test-integration.sh`
3. Review output for any failures
4. Document results in `docs/TEST_RESULTS.md`

**Expected Outcome**:
- All 5 integration tests pass
- OBS connection verified
- Source creation/deletion works
- Recording lifecycle validated

**Success Criteria**:
- Script exits with code 0
- No manual intervention required during tests
- Clear summary: "5 tests passed, 0 failed"

---

## Technical Architecture

### Test Infrastructure Components

```
┌─────────────────────────────────────────────────┐
│          Test Execution Layer                    │
│  (go test runner, table-driven tests)           │
└────────────────┬────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────┐
│        Test Utilities Layer                      │
│  ├─ testutil/mock_obs.go    (WebSocket mock)   │
│  ├─ testutil/log_capture.go (Log assertions)   │
│  └─ testutil/assertions.go  (Common helpers)   │
└────────────────┬────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────┐
│         Test Fixtures Layer                      │
│  internal/obsws/testdata/*.json                 │
│  (Realistic OBS WebSocket responses)            │
└────────────────┬────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────┐
│      System Under Test (SUT)                     │
│  ├─ internal/obsws/client.go                    │
│  ├─ internal/obsws/sources.go                   │
│  └─ cmd/memofy-core/main.go                     │
└─────────────────────────────────────────────────┘
```

### Mock OBS Server Design

**Purpose**: Simulate OBS WebSocket v5 protocol for isolated testing

**Key Methods**:
```go
type MockOBSServer struct {
    listener   net.Listener
    conn       *websocket.Conn
    responses  map[string]interface{}
    mode       FailureMode
}

func (m *MockOBSServer) Start() error
func (m *MockOBSServer) Stop() error
func (m *MockOBSServer) SetFailureMode(mode string)
func (m *MockOBSServer) QueueResponse(requestType string, response interface{})
func (m *MockOBSServer) HandleConnection()
```

**Failure Modes**:
- `"normal"` - Standard responses
- `"code204"` - Return error 204 for all requests
- `"timeout"` - Delay response > 6 seconds
- `"disconnect"` - Close connection unexpectedly

---

## Dependencies & Assumptions

### Dependencies

**Required**:
- Go 1.21+ (generics support)
- `github.com/gorilla/websocket` (WebSocket implementation)
- Standard library `testing` package
- macOS (for integration tests with OBS)

**Optional**:
- OBS Studio 28.0+ (for manual integration tests only)
- `nc` (netcat) for connectivity checks
- `jq` for JSON parsing in bash scripts

### Assumptions

**Test Environment**:
- Tests run on developer machine or CI environment
- Network access to localhost available
- No firewall blocking localhost:4455
- Sufficient disk space for coverage files (< 10MB)

**OBS Configuration** (Manual Tests Only):
- OBS running with WebSocket server enabled
- Port 4455 accessible
- At least one scene available
- Recording output directory writable

**Code State**:
- Phases 1-5 already implemented and compiling
- No breaking changes to `client.go` or `sources.go` interfaces
- Log format uses `[TAG]` prefix convention

---

## Edge Cases & Error Handling

### Edge Case 1: Port Already in Use
**Scenario**: Mock server can't bind to port (previous test didn't clean up)
**Handling**: 
- Use dynamic port allocation (`listener.Addr()`)
- Defer cleanup in every test: `defer mock.Stop()`
- Verify port released before next test starts

### Edge Case 2: JSON Fixture Parse Error
**Scenario**: Malformed JSON in testdata/*.json file
**Handling**:
- Validate all fixtures in `TestFixturesValid()` test
- Use `json.Unmarshal()` with strict error checking
- Fail fast with clear file path in error message

### Edge Case 3: Test Timeout
**Scenario**: Test hangs waiting for WebSocket response
**Handling**:
- Set test timeout: `go test -timeout 30s`
- Use `context.WithTimeout()` for all WebSocket operations
- Log "Test timed out waiting for X" before context cancellation

### Edge Case 4: Goroutine Leak
**Scenario**: Test spawns goroutine that doesn't exit
**Handling**:
- Use `goleak` package: `defer goleak.VerifyNone(t)`
- Ensure all background tasks have cancellation contexts
- Verify `client.Close()` called in defer statement

### Edge Case 5: Flaky Test (Timing-Dependent)
**Scenario**: Test passes/fails inconsistently due to timing
**Handling**:
- Avoid `time.Sleep()` in tests, use channels for synchronization
- For backoff tests: validate delay range, not exact value
- Run tests multiple times: `go test -count=10`

---

## Success Criteria

### Phase 6 Complete When:

**Code Quality**:
- [ ] All 20 tests implemented (8 client + 7 sources + 5 startup)
- [ ] All tests passing consistently (no flakiness)
- [ ] Code compiles without warnings
- [ ] No race conditions detected (`-race` flag clean)
- [ ] No goroutine leaks (verified with `goleak`)

**Coverage Metrics**:
- [ ] Overall coverage > 80%
- [ ] `internal/obsws/client.go` coverage > 85%
- [ ] `internal/obsws/sources.go` coverage > 90%
- [ ] Coverage report generated in HTML format

**Test Performance**:
- [ ] All unit tests complete in < 30 seconds total
- [ ] Individual tests complete in < 3 seconds each
- [ ] Integration tests complete in < 60 seconds

**Documentation**:
- [ ] `docs/TEST_RESULTS.md` created with results
- [ ] `docs/HANDOFF.md` updated to 100% completion
- [ ] Testing commands added to quick reference
- [ ] README.md includes testing section

**Integration**:
- [ ] Manual integration tests pass against real OBS
- [ ] Bash script executes without errors
- [ ] All 5 manual test scenarios validated

---

## Acceptance Testing

### Pre-Deployment Checklist

Before marking Phase 6 complete, verify:

1. **Test Execution**:
   ```bash
   # All tests pass
   go test ./... -v -timeout 30s
   echo "Exit code: $?"  # Must be 0
   
   # No races
   go test ./... -race
   echo "Exit code: $?"  # Must be 0
   ```

2. **Coverage Verification**:
   ```bash
   # Generate report
   go test ./... -coverprofile=coverage.out
   go tool cover -func=coverage.out | grep total
   # Verify: total coverage > 80%
   ```

3. **Integration Tests** (with OBS running):
   ```bash
   bash scripts/test-integration.sh
   echo "Exit code: $?"  # Must be 0
   ```

4. **Documentation**:
   - [ ] `docs/TEST_RESULTS.md` exists and populated
   - [ ] `docs/HANDOFF.md` shows 100% completion
   - [ ] All links in documentation work

5. **Final Build**:
   ```bash
   go build ./...
   echo "Exit code: $?"  # Must be 0
   ```

---

## Implementation Timeline

### Day 1: Foundation (8 hours)
- **Hours 1-2**: Create test infrastructure
  - T001: testdata directory
  - T002-T007: JSON fixtures (parallel)
  - T008: mock_obs.go
- **Hours 3-4**: Test utilities
  - T009: log_capture.go
  - T010: assertions.go
  - T011: Client test setup
- **Hours 5-8**: Client tests
  - T012-T019: 8 client tests (parallel implementation)

### Day 2: Tests (8 hours)
- **Hours 1-5**: Source tests
  - T020: Sources test setup
  - T021-T027: 7 source tests (parallel)
- **Hours 6-8**: Startup tests
  - T028: Startup test infrastructure
  - T029-T033: 5 startup tests (parallel)

### Day 3: Integration & Verification (4 hours)
- **Hour 1**: Integration script
  - T034: Bash script structure
  - T035-T039: 5 integration tests
- **Hours 2-4**: Verification
  - T040-T045: Run all tests, generate coverage, document results

### Day 4: Polish (Optional, 1 hour)
- T046: Clean up test-only code
- T047-T049: Update documentation

**Total Effort**: 18-20 hours over 3-4 days

---

## Risks & Mitigation

### Risk 1: Timing-Dependent Test Failures
**Probability**: Medium  
**Impact**: High (flaky tests reduce confidence)  
**Mitigation**:
- Use synchronization primitives (channels) instead of `time.Sleep()`
- Validate ranges for jitter tests, not exact values
- Run tests multiple times in CI: `go test -count=10`

### Risk 2: Mock Server Port Conflicts
**Probability**: Low  
**Impact**: Medium (tests fail locally)  
**Mitigation**:
- Use dynamic port allocation
- Verify port released in cleanup
- Add retry logic for port binding

### Risk 3: Incomplete Coverage
**Probability**: Low  
**Impact**: High (gaps in error handling)  
**Mitigation**:
- Monitor coverage incrementally after each test
- Identify uncovered branches with `go tool cover -html`
- Add targeted tests for low-coverage areas

### Risk 4: Integration Tests Require Manual Setup
**Probability**: High  
**Impact**: Low (expected, documented)  
**Mitigation**:
- Clear documentation of OBS prerequisites
- Graceful failure with actionable error messages
- Skip integration tests in CI if OBS unavailable

---

## Glossary

**Terms**:
- **SUT**: System Under Test (code being tested)
- **Mock**: Simulated component replacing real dependency
- **Fixture**: Static test data (JSON files)
- **Coverage**: Percentage of code executed by tests
- **Race Condition**: Bug where timing affects correctness
- **Goroutine Leak**: Background task that doesn't exit
- **Flaky Test**: Test that passes/fails inconsistently

**OBS Terms**:
- **Hello**: Initial WebSocket handshake message
- **Identified**: Authentication confirmation message
- **CreateInput**: OBS request to create new source
- **Code 204**: OBS error "request type invalid"
- **Code 203**: OBS error "request processing failed"

---

## References

**Related Documents**:
- [docs/TESTING_PLAN.md](TESTING_PLAN.md) - Detailed test specifications
- [docs/GO_IMPLEMENTATION_ROADMAP.md](GO_IMPLEMENTATION_ROADMAP.md) - Implementation guide
- [docs/PHASE6_QUICK_REFERENCE.md](PHASE6_QUICK_REFERENCE.md) - Quick lookups
- [docs/tasks.md](tasks.md) - Task breakdown (49 tasks)
- [docs/logging.md](logging.md) - Phases 1-5 specification

**External References**:
- OBS WebSocket v5 Protocol: https://github.com/obsproject/obs-websocket/blob/master/docs/generated/protocol.md
- Go Testing Guide: https://golang.org/pkg/testing/
- Gorilla WebSocket: https://github.com/gorilla/websocket

---

## Appendix: Key Decision Log

**Decision 1**: Use hand-written mock server instead of library
- **Rationale**: Full control over failure modes, simpler dependencies
- **Alternatives Considered**: `gomock`, `testify/mock`
- **Trade-off**: More boilerplate, but clearer test intent

**Decision 2**: Compress 5-minute timeout to 5 seconds in test mode
- **Rationale**: Fast test execution without changing production behavior
- **Implementation**: `MEMOFY_TEST_MODE` environment variable
- **Risk**: Test doesn't validate exact production timing

**Decision 3**: Manual integration tests in bash script
- **Rationale**: Real OBS validation, complex to automate in Go
- **Alternatives Considered**: Go-based integration tests
- **Trade-off**: Manual execution, but validates real-world behavior

**Decision 4**: Separate coverage targets per file
- **Rationale**: Critical files (client, sources) need higher coverage
- **Targets**: 85% client, 90% sources, 80% overall
- **Justification**: These files contain error recovery logic

**Decision 5**: Test timeout set to 30 seconds
- **Rationale**: Balance between catching hangs and allowing slow CI
- **Individual Limit**: 3 seconds per test
- **Total Suite**: < 30 seconds for fast feedback

---

**End of Specification**

This document is ready for implementation. See [docs/tasks.md](tasks.md) for the 49-task breakdown and [docs/GO_IMPLEMENTATION_ROADMAP.md](GO_IMPLEMENTATION_ROADMAP.md) for step-by-step coding guidance.
