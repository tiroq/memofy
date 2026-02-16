# Phase 6: Integration Testing Plan for Go

## Overview

This document outlines the comprehensive testing strategy for memofy's six-phase improvement plan. All tests are written in Go using standard `testing` package with mock WebSocket connections to OBS.

---

## Testing Architecture

### Test Pyramid

```
                    ┌─────────────────┐
                    │  End-to-End     │
                    │  (vs real OBS)  │ ← Manual/Integration
                    └─────────────────┘
                           ▲
                    ┌──────────────────┐
                    │  Integration     │
                    │  (mock OBS)      │ ← Automated, ~20 tests
                    └──────────────────┘
                           ▲
                    ┌──────────────────┐
                    │  Unit Tests      │
                    │  (isolated)      │ ← Automated, ~10 tests
                    └──────────────────┘
```

### Test Technologies

**Unit Testing**:
- Framework: `testing` (Go standard library)
- Mocking: Hand-written mock WebSocket server
- Fixtures: JSON response files in `testdata/`

**Integration Testing**:
- Framework: `testing` with mock OBS
- WebSocket: `github.com/gorilla/websocket` (mock server)
- JSON: Marshaling/unmarshaling OBS WebSocket responses

**End-to-End Testing** (Manual):
- Real OBS instance on localhost:4455
- Bash script to automate: start OBS → test scenarios → verify results

---

## File Structure

```
memofy/
├── internal/obsws/
│   ├── client.go
│   ├── client_test.go              ← NEW (8 tests)
│   ├── sources.go
│   ├── sources_test.go             ← NEW (7 tests)
│   └── testdata/                   ← NEW
│       ├── hello_response.json      (OBS Hello frame)
│       ├── identified_response.json (OBS Identified frame)
│       ├── code_204_response.json   (Error 204 response)
│       ├── code_203_response.json   (Timeout 203 response)
│       ├── scene_list_response.json (GetSceneList response)
│       └── create_input_response.json (CreateInput response)
│
├── cmd/memofy-core/
│   ├── main.go
│   └── startup_test.go             ← NEW (5 tests)
│
├── scripts/
│   └── test-integration.sh         ← NEW (Manual test suite)
│
└── docs/
    └── TESTING_PLAN.md             ← THIS FILE
```

---

## Unit Tests: `internal/obsws/client_test.go`

### Purpose
Test WebSocket connection handling, error responses, and reconnection logic in isolation.

### Test Cases

#### Test 1: `TestConnectionHandshake`
**Scenario**: Successful WebSocket connection establishment
```
Client → WebSocket Connect
  ↓
Server → Hello { version, supportedVersion }
  ↓
Client → Identify { auth, etc }
  ↓
Server → Identified { obsVersion: 29.1.3, webSocketVersion: 5.0.5 }
  ↓
Result: ✓ Connection established, versions extracted
```

**Code Outline**:
```go
func TestConnectionHandshake(t *testing.T) {
    // 1. Create mock WebSocket server (listen on localhost:4455)
    // 2. Load testdata/hello_response.json
    // 3. Load testdata/identified_response.json
    // 4. Create client.New("localhost:4455", "password")
    // 5. Verify obsVersion = "29.1.3"
    // 6. Verify wsVersion = "5.0.5"
    // 7. Verify no error
}
```

**Success Criteria**:
- ✓ Client receives OBS and WebSocket versions
- ✓ Versions are correctly parsed and stored
- ✓ No connection error

---

#### Test 2: `TestConnectionHandshakeBadVersion`
**Scenario**: OBS version too old (< 28.0)
```
Client → Connect
  ↓
Server → Hello { version, supportedVersion }
  ↓
Client → Identify
  ↓
Server → Identified { obsVersion: 27.2.1, ... }
  ↓
Result: ✗ Version incompatible (27.2 < 28.0)
```

**Code Outline**:
```go
func TestConnectionHandshakeBadVersion(t *testing.T) {
    // 1. Mock server returns obsVersion: "27.2.1"
    // 2. Create client
    // 3. Verify err != nil
    // 4. Verify err contains "version" or "28.0"
}
```

**Success Criteria**:
- ✓ Error returned immediately on bad version
- ✓ Error message contains helpful text about version requirement
- ✓ Connection closed gracefully

---

#### Test 3: `TestErrorCode204Handling`
**Scenario**: OBS rejects CreateInput request with code 204
```
Client → CreateInput { inputName: "Desktop Audio", ... }
  ↓
Server → { requestStatus: { result: false, code: 204 }, 
           responseData: { comment: "request type is not valid" } }
  ↓
Result: ✗ Code 204 (InvalidRequest) - likely version mismatch
Client logs: "OBS rejected request type 'CreateInput' (code 204)"
Client continues operating (graceful degradation)
```

**Code Outline**:
```go
func TestErrorCode204Handling(t *testing.T) {
    // 1. Create client with v29.1 (but mock server will reject)
    // 2. Call createInput() → code 204 response
    // 3. Verify returned error contains "204"
    // 4. Verify returned error specifies "CreateInput"
    // 5. Verify client is still connected (not closed)
    // 6. Verify no panic or fatal error
}
```

**Success Criteria**:
- ✓ Error code 204 recognized and logged
- ✓ Error message includes request type
- ✓ Client remains operational (can still call other methods)
- ✓ No automatic exit or disable-recording

---

#### Test 4: `TestErrorCode203Timeout`
**Scenario**: OBS response timeout (code 203)
```
Client → Method request
  ↓
(wait 10+ seconds)
  ↓
Server → { code: 203 } (request took too long)
  ↓
Result: ✗ Timeout (slow OBS, high CPU, or network latency)
Client logs: "OBS request timed out (code 203)"
Client will retry (part of reconnection logic)
```

**Code Outline**:
```go
func TestErrorCode203Timeout(t *testing.T) {
    // 1. Create client
    // 2. Inject artificial delay in mock server
    // 3. Call method that times out
    // 4. Verify error contains "203" or "timeout"
    // 5. Verify error is retryable (not fatal)
}
```

**Success Criteria**:
- ✓ Timeout detected and logged
- ✓ Not treated as fatal (client can retry)
- ✓ User warned about slow OBS

---

#### Test 5: `TestReconnectionWithBackoff`
**Scenario**: OBS disconnects, client reconnects with exponential backoff
```
Time 0:  Connection active
Time 1:  OBS socket closes
Time 2:  [RECONNECT] Attempt 1, delay 5s
Time 7:  [RECONNECT] Attempt 2, delay 10s (5 * 2)
Time 17: [RECONNECT] Attempt 3, delay 20s (10 * 2)
Time 37: [RECONNECT] Attempt 4, delay 40s (20 * 2)
Time 77: [RECONNECT] Attempt 5, delay 60s (40 * 2, capped)
Time 137: [RECONNECT] Attempt 6, delay 60s (cap maintained)
```

**Code Outline**:
```go
func TestReconnectionWithBackoff(t *testing.T) {
    // 1. Create client connected to mock server
    // 2. Close server connection (simulate OBS crash)
    // 3. Client detects disconnection
    // 4. Verify [RECONNECT] attempt 1 logged (delay 5s)
    // 5. Verify [RECONNECT] attempt 2 logged (delay 10s)
    // 6. Verify [RECONNECT] attempt 3 logged (delay 20s)
    // 7. Verify delays follow: 5, 10, 20, 40, 60, 60, 60...
    // 8. When server comes back online: verify reconnection succeeds
}
```

**Success Criteria**:
- ✓ Backoff sequence: 5s → 10s → 20s → 40s → 60s (capped)
- ✓ Each attempt logged with attempt number
- ✓ Exponential growth: delay = previous * 2
- ✓ Reconnection succeeds when server available
- ✓ Logs include jitter (±10% variance) explanation

---

#### Test 6: `TestReconnectionWithJitter`
**Scenario**: Multiple clients reconnecting simultaneously use jitter to avoid thundering herd
```
Without jitter (bad):  All clients reconnect at exactly 5, 10, 20, 40s → spike load
With jitter (good):    5 ± 0.5s, 10 ± 1s, 20 ± 2s → spread over time
```

**Code Outline**:
```go
func TestReconnectionWithJitter(t *testing.T) {
    // 1. Create 3 client instances
    // 2. Disconnect all simultaneously
    // 3. Measure reconnection attempt times
    // 4. Verify attempts are NOT at exact intervals (jitter applied)
    // 5. Verify jitter is ±10% (e.g., 5s ± 0.5s = 4.5-5.5s)
    // 6. Verify all clients eventually reconnect successfully
}
```

**Success Criteria**:
- ✓ Jitter applied: delay varied by ±10%
- ✓ Multiple clients don't spike OBS at same time
- ✓ Reconnection still succeeds

---

#### Test 7: `TestConnectionLossDetection`
**Scenario**: Detect when WebSocket is unexpectedly closed
```
Client reading from socket
  ↓
Socket EOF (OBS crashed)
  ↓
Error propagated immediately
  ↓
[ERROR] OBS connection lost
```

**Code Outline**:
```go
func TestConnectionLossDetection(t *testing.T) {
    // 1. Create connected client
    // 2. Abruptly close server socket (simulate OBS crash)
    // 3. Try to call method on client
    // 4. Verify error returned immediately (no hang)
    // 5. Verify error indicates connection loss
    // 6. Verify reconnection logic triggered
}
```

**Success Criteria**:
- ✓ Connection loss detected within 1s
- ✓ Appropriate error returned
- ✓ Automatic reconnection initiated

---

#### Test 8: `TestRequestResponseSequencing`
**Scenario**: Multiple concurrent requests handled correctly (not mixed up)
```
Client → Request A (id: "a1")
Client → Request B (id: "b1")
Server → Response for B (id: "b1")
Server → Response for A (id: "a1")
Result: ✓ Responses matched to correct requesters (via requestId)
```

**Code Outline**:
```go
func TestRequestResponseSequencing(t *testing.T) {
    // 1. Create client
    // 2. Send request A, request B in sequence (don't wait for responses)
    // 3. Send response for B first, then A
    // 4. Verify each goroutine receives correct response
    // 5. Verify responses don't get crossed (A doesn't get B's response)
}
```

**Success Criteria**:
- ✓ Responses correctly routed by requestId
- ✓ No request data corruption
- ✓ Out-of-order responses handled correctly

---

## Unit Tests: `internal/obsws/sources_test.go`

### Purpose
Test source creation, validation, and retry logic in isolation.

### Test Cases

#### Test 1: `TestEnsureRequiredSources`
**Scenario**: Scene is empty, sources must be created
```
Scene "Collection 1"
  ├─ Display Capture? ✗ Missing
  └─ Desktop Audio?   ✗ Missing

Action:
  1. [SOURCES] Checking scene "Collection 1"
  2. [SOURCE_FOUND] ✗ (none exist)
  3. [CREATE] Creating Display Capture (macos_screen_capture)
  4. [SUCCESS] Display Capture created
  5. [CREATE] Creating Desktop Audio (coreaudio_input_capture)
  6. [SUCCESS] Desktop Audio created
  7. [VERIFY] ✓ All sources present and enabled

Result: ✓ Recording ready
```

**Code Outline**:
```go
func TestEnsureRequiredSources(t *testing.T) {
    // 1. Create mock OBS client
    // 2. Mock GetSceneList() → returns scene "Collection 1" but empty
    // 3. Call EnsureRequiredSources("Collection 1")
    // 4. Verify CreateInput called twice (audio + display)
    // 5. Verify sources created with correct inputKind
    // 6. Verify log output shows [CREATE] tags
    // 7. Verify no error returned
    // 8. Verify all sources enabled (Enabled: true)
}
```

**Success Criteria**:
- ✓ Both sources created
- ✓ Correct platform-specific types (macOS: coreaudio_input_capture, macos_screen_capture)
- ✓ Logging shows [CREATE] and [SUCCESS] tags
- ✓ Enabled state verified post-creation
- ✓ Recording enabled signal sent

---

#### Test 2: `TestSourceAlreadyExists`
**Scenario**: Display Capture exists but disabled; Audio missing
```
Scene "Collection 1"
  ├─ Display Capture (enabled: false) ← Disabled
  └─ Desktop Audio?   ✗ Missing

Action:
  1. [SOURCE_FOUND] Display Capture exists but disabled
  2. [ENABLE] Enabling Display Capture
  3. [CREATE] Creating Desktop Audio
  4. [SUCCESS] Desktop Audio created
  5. [VERIFY] ✓ Both sources now enabled

Result: ✓ Recording ready
```

**Code Outline**:
```go
func TestSourceAlreadyExists(t *testing.T) {
    // 1. Create mock client
    // 2. Mock GetSceneList() → returns scene with Display Capture (enabled: false)
    // 3. Call EnsureRequiredSources("Collection 1")
    // 4. Verify SetInputEnabled called for Display (enable it)
    // 5. Verify CreateInput called once (Audio only)
    // 6. Verify log shows [SOURCE_FOUND] and [ENABLE] tags
    // 7. Verify final validation passes
}
```

**Success Criteria**:
- ✓ Disabled source detected (enabled: false)
- ✓ SetInputEnabled called to enable it
- ✓ Missing sources still created
- ✓ Final state: both sources present AND enabled
- ✓ Recording enabled signal sent

---

#### Test 3: `TestCreateInputFailsWithCode204`
**Scenario**: CreateInput request gets code 204 (OBS version incompatible)
```
Client → CreateInput { inputName: "Desktop Audio", ... }
  ↓
Server → { code: 204, comment: "request type is not valid" }
  ↓
Action:
  1. Log: [CREATE_RETRY] Attempting source creation (attempt 1/3)
  2. Log: [ERROR] Code 204 - OBS version incompatible
  3. Try again? No. Code 204 indicates version mismatch (won't help to retry)
  4. Log: [WARN] User must update OBS to 28.0+ or create source manually
  5. Return gracefully (don't crash)

Result: ✗ Source not created, app continues with warning
```

**Code Outline**:
```go
func TestCreateInputFailsWithCode204(t *testing.T) {
    // 1. Create mock client
    // 2. Mock CreateInput to return code 204
    // 3. Call CreateSourceWithRetry("Desktop Audio", "coreaudio_input_capture")
    // 4. Verify error returned with code 204
    // 5. Verify only 1 attempt made (fast-fail, no retry for 204)
    // 6. Verify log contains [CREATE_RETRY] and [ERROR]
    // 7. Verify no crash or panic
}
```

**Success Criteria**:
- ✓ Code 204 detected and reported
- ✓ No wasted retries (code 204 is not retryable)
- ✓ Graceful error return
- ✓ Warning message logged with OBS version hint
- ✓ App continues running (not disabled)

---

#### Test 4: `TestCreateSourceWithRetry`
**Scenario**: CreateInput fails 2x, succeeds on 3rd attempt
```
Attempt 1: Code 500 error → wait 1s
Attempt 2: Code 500 error → wait 2s
Attempt 3: ✓ Success

Result: ✓ Source created after 3 attempts with 3s total backoff
```

**Code Outline**:
```go
func TestCreateSourceWithRetry(t *testing.T) {
    // 1. Create mock client
    // 2. Mock CreateInput: fail twice with code 500, succeed on 3rd
    // 3. Call CreateSourceWithRetry(...)
    // 4. Verify 3 attempts made
    // 5. Verify delays: 1s after attempt 1, 2s after attempt 2
    // 6. Verify log shows [CREATE_RETRY] attempt 1/3, 2/3, 3/3
    // 7. Verify final success
    // 8. Verify no error returned
}
```

**Success Criteria**:
- ✓ Retries on transient failures (code 500)
- ✓ Backoff increases: 1s, 2s, 3s...
- ✓ Succeeds on 3rd attempt
- ✓ Logging shows all attempts
- ✓ Fast-fails on code 204 (doesn't use full 3 attempts)

---

#### Test 5: `TestSourceValidationPostCreation`
**Scenario**: Source created, but we verify it's actually enabled before declaring success
```
After CreateInput succeeds:
  1. Call GetInputSettings("Desktop Audio")
  2. Verify Enabled: true
  3. Verify settings aren't corrupted
  4. Log: [VERIFY] Desktop Audio enabled

Result: ✓ Source is truly ready
```

**Code Outline**:
```go
func TestSourceValidationPostCreation(t *testing.T) {
    // 1. Create mock client
    // 2. Mock successful CreateInput
    // 3. Mock GetInputSettings to return Enabled: true
    // 4. Call EnsureRequiredSources()
    // 5. Verify GetInputSettings called (validation step)
    // 6. Verify Enabled field checked
    // 7. Verify log shows [VERIFY] tag
    // 8. Verify success
}
```

**Success Criteria**:
- ✓ Post-creation validation performed (not just trusting CreateInput)
- ✓ GetInputSettings called to verify
- ✓ Enabled field checked and logged
- ✓ Confidence that source is truly ready to use

---

#### Test 6: `TestSourceCreationTimeLimit`
**Scenario**: Sources fail to create for 5+ minutes, recording should auto-disable
```
Time 0:    Source creation fails
Time 0:    Start 5-minute retry window
Time 10:   Retry, still fails
Time 20:   Retry, still fails
Time 30:   Retry, still fails
...
Time 300:  5 minutes elapsed
Time 300:  [WARN] Recording disabled: sources unavailable for 5 min
Result:    Recording disabled, user warned
```

**Code Outline**:
```go
func TestSourceCreationTimeLimit(t *testing.T) {
    // This test is complex: requires time mocking or real 5-min wait
    // Option 1: Mock time.Now() (using a time.Time interface)
    // Option 2: Use time.Sleep (slow but reliable)
    // 
    // 1. Create mock client with persistent CreateInput failures
    // 2. Call EnsureRequiredSources() → fails
    // 3. Start recovery loop (every 10s retry)
    // 4. Advance time to 5m 1s (via mock or sleep)
    // 5. Verify 30+ retry attempts made
    // 6. Verify [WARN] "Recording disabled" message logged
    // 7. Verify recording disabled flag set
}
```

**Success Criteria**:
- ✓ Retries continue for 5 minutes
- ✓ After 5 min, recording auto-disabled
- ✓ User warned with clear message
- ✓ Can restart to reset timer

---

#### Test 7: `TestSourceRecovery`
**Scenario**: Sources initially missing, then become available (within 5-min window)
```
Time 0:    Sources missing, recovery loop starts
Time 10:   Retry: still missing
Time 20:   Retry: still missing
Time 30:   Retry: ✓ Sources now available!
Time 30:   [RECOVERY] Sources available, recording enabled
Result:    ✓ Recording enabled, user not warned
```

**Code Outline**:
```go
func TestSourceRecovery(t *testing.T) {
    // 1. Create mock client
    // 2. Initial calls to CreateInput → fail
    // 3. Recovery loop starts
    // 4. After 3 retries, mock CreateInput to succeed
    // 5. Verify [RECOVERY] log message
    // 6. Verify recording enabled flag set
    // 7. Verify <5 min elapsed (not auto-disabled)
}
```

**Success Criteria**:
- ✓ Sources created successfully after recovery
- ✓ Recording automatically re-enabled
- ✓ [RECOVERY] message logged
- ✓ No user intervention needed

---

## Integration Tests: `cmd/memofy-core/startup_test.go`

### Purpose
Test the startup sequence with mocked OBS, validating the full initialization flow.

### Test Cases

#### Test 1: `TestStartupSuccessful`
**Scenario**: Normal startup with OBS available and compatible
```
1. Load config
2. Check permissions
3. Launch OBS (already running)
4. Connect to WebSocket
5. Validate version (29.1.3 >= 28.0) ✓
6. Ensure sources (created or verified) ✓
7. Start detection loop ✓
8. [RUNNING] Memofy Core is running

Result: ✓ Full startup completed
```

**Code Outline**:
```go
func TestStartupSuccessful(t *testing.T) {
    // 1. Create mock OBS server
    // 2. Create memofy-core with config pointing to mock
    // 3. Call startup sequence
    // 4. Verify all phases logged with [STARTUP] tags
    // 5. Verify final "[RUNNING]" message
    // 6. Verify detection loop active
    // 7. Verify sources ready (enabled)
    // 8. Cleanup: graceful shutdown
}
```

**Success Criteria**:
- ✓ All startup phases logged
- ✓ No errors or warnings
- ✓ Recording ready
- ✓ Total time <30 seconds

---

#### Test 2: `TestStartupWithoutOBS`
**Scenario**: OBS not running at startup
```
1. Check OBS on port 4455 → Not found
2. Attempt to launch OBS
3. (Mock: OBS starts in 2 seconds)
4. Connect to WebSocket → Success
5. Continue as normal
```

**Code Outline**:
```go
func TestStartupWithoutOBS(t *testing.T) {
    // 1. Don't start mock OBS server yet
    // 2. Call startup sequence (expect attempt to launch)
    // 3. After small delay, start mock OBS
    // 4. Verify connection succeeds after OBS available
    // 5. Verify "[STARTUP] Launching OBS..." logged
    // 6. Continue to normal startup
}
```

**Success Criteria**:
- ✓ OBS launch detected and logged
- ✓ Connection retried until OBS available
- ✓ Continues to normal startup
- ✓ Doesn't fail on startup

---

#### Test 3: `TestStartupWithIncompatibleOBS`
**Scenario**: OBS version too old (27.2.1 < 28.0)
```
1. OBS running on port 4455
2. Connect to WebSocket
3. Receive version: 27.2.1
4. [STARTUP] OBS Health: OBS 27.2.1 is too old (need 28.0+)
5. [ERROR] OBS version incompatible
6. Log suggested fix: Update OBS from obsproject.com
7. Continue with warning (don't exit)
```

**Code Outline**:
```go
func TestStartupWithIncompatibleOBS(t *testing.T) {
    // 1. Create mock OBS server returning version 27.2.1
    // 2. Call startup sequence
    // 3. Verify error logged with version info
    // 4. Verify [STARTUP] OBS Health check logged
    // 5. Verify suggested fix message shown
    // 6. Verify app continues (doesn't exit)
}
```

**Success Criteria**:
- ✓ Version mismatch detected early
- ✓ Clear error message with version requirement
- ✓ Suggested fix provided
- ✓ App continues (graceful degradation)
- ✓ User can update OBS and restart without code changes

---

#### Test 4: `TestStartupWithoutPermissions`
**Scenario**: Screen Recording permission not granted
```
1. Check permission
2. Permission denied ✗
3. [ERROR] Screen Recording permission required
4. Log: Go to System Preferences > Security & Privacy > Screen Recording
5. Exit gracefully with message
```

**Code Outline**:
```go
func TestStartupWithoutPermissions(t *testing.T) {
    // 1. Mock permission check to return denied
    // 2. Call startup sequence
    // 3. Verify error logged about permission
    // 4. Verify helpful message with System Preferences path
    // 5. Verify graceful exit (not panic)
    // 6. Verify exit code is non-zero
}
```

**Success Criteria**:
- ✓ Permission check performed early
- ✓ Clear error message with steps to fix
- ✓ Graceful exit (not panic or hang)
- ✓ No resources leaked

---

#### Test 5: `TestSignalHandlingGraceful`
**Scenario**: Graceful shutdown on SIGTERM
```
1. App running, detection loop active
2. Receive SIGTERM
3. [SHUTDOWN] Graceful shutdown requested
4. Stop detection loop (finish current cycle)
5. Stop recording if active
6. Close OBS connection
7. Remove PID file
8. Exit with code 0
Total time: <5 seconds
```

**Code Outline**:
```go
func TestSignalHandlingGraceful(t *testing.T) {
    // 1. Start memofy-core with mock OBS
    // 2. Let it run detection loop briefly
    // 3. Send SIGTERM via os.Process.Signal()
    // 4. Wait for shutdown
    // 5. Verify [SHUTDOWN] logs present
    // 6. Verify exit code is 0
    // 7. Verify process exits cleanly (no hang >5s)
    // 8. Verify no goroutines leak
}
```

**Success Criteria**:
- ✓ SIGTERM handled immediately
- ✓ Graceful shutdown logged
- ✓ All resources cleaned up
- ✓ Exit within 5 seconds
- ✓ PID file removed
- ✓ No dangling goroutines

---

## Integration Script: `scripts/test-integration.sh`

### Purpose
Manual integration testing against a real OBS instance using WebSocket commands.

### Structure

```bash
#!/bin/bash

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test utilities
assert_exit_code() { ... }
assert_contains() { ... }
start_obs_server() { ... }

# Test functions
test_obs_connection() { ... }
test_scene_list() { ... }
test_source_creation() { ... }
test_recording_start_stop() { ... }
test_recovery_mode() { ... }

# Main
run_all_tests() { ... }
```

### Test Cases

#### Test 1: `test_obs_connection`
```bash
# Purpose: Verify OBS WebSocket is reachable
# Method: nc -zv localhost 4455
# Expected: Connection succeeded
# Fail if: Port unreachable (OBS not running)
```

#### Test 2: `test_scene_list`
```bash
# Purpose: Verify can retrieve scene list from OBS
# Method: Send GetSceneList request via WebSocket
# Expected: Receive list of scenes with names
# Fail if: Malformed response or no scenes
```

#### Test 3: `test_source_creation`
```bash
# Purpose: Verify can create Display Capture and Audio sources
# Method: Send CreateInput requests for both source types
# Expected: Sources created with Enabled: true
# Fail if: Code 204 or creation fails
# Cleanup: Delete created sources
```

#### Test 4: `test_recording_start_stop`
```bash
# Purpose: Verify can start and stop recording
# Method: Send StartRecord → wait → StopRecord
# Expected: Recording file created and closed properly
# Fail if: Recording status incorrect or file not found
# Cleanup: Delete test recording
```

#### Test 5: `test_recovery_mode`
```bash
# Purpose: Verify sources can be created after initial failure
# Method: 
#   1. Try to create source 1 (fails due to mock error)
#   2. Wait 2 seconds
#   3. Try to create source 2 (succeeds)
#   4. Verify recovery logged
# Expected: Both sources eventually available
# Fail if: Recovery doesn't work or takes >5 min
```

---

## Testing Best Practices

### Mocking Strategy

**Mock WebSocket Server**:
```go
// testutils/mock_obs.go
type MockOBSServer struct {
    listener    net.Listener
    responses   map[string]ResponseMock  // Request type → response
    failureMode string                   // "offline", "code204", "code203", etc.
}

func NewMockOBS() *MockOBSServer { ... }
func (m *MockOBSServer) Start() error { ... }
func (m *MockOBSServer) Stop() error { ... }
func (m *MockOBSServer) SetFailureMode(mode string) { ... }
```

**Usage in Tests**:
```go
func TestExample(t *testing.T) {
    mock := NewMockOBS()
    mock.Start()
    defer mock.Stop()
    
    mock.SetFailureMode("code204")  // Next request returns code 204
    
    client := New("localhost:4455", "")
    err := client.CreateInput(...)
    
    if err == nil || !strings.Contains(err.Error(), "204") {
        t.Fatal("Expected code 204 error")
    }
}
```

### Logging Capture

**Capture structured logs**:
```go
// testutils/log_capture.go
type LogCapture struct {
    logs []string
}

func (lc *LogCapture) Write(p []byte) (int, error) {
    lc.logs = append(lc.logs, string(p))
    return len(p), nil
}

func (lc *LogCapture) Contains(substring string) bool { ... }
func (lc *LogCapture) ContainsAll(tags ...string) bool { ... }
```

**Usage**:
```go
func TestLogging(t *testing.T) {
    capture := &LogCapture{}
    oldOutput := log.Writer()
    log.SetOutput(capture)
    defer log.SetOutput(oldOutput)
    
    // Run test...
    
    if !capture.Contains("[STARTUP]") {
        t.Fatal("Missing [STARTUP] in logs")
    }
    if !capture.ContainsAll("[CREATE]", "[SUCCESS]") {
        t.Fatal("Missing source creation logs")
    }
}
```

### Test Data (Fixtures)

**Load JSON responses from files**:
```go
// testdata/hello_response.json
{
  "op": 0,
  "d": {
    "obsWebSocketVersion": "5.0.5",
    "rpcVersion": "1.0"
  }
}

// testdata/identified_response.json
{
  "op": 2,
  "d": {
    "obsVersion": "29.1.3",
    "obsWebSocketVersion": "5.0.5"
  }
}
```

**Load in tests**:
```go
func LoadTestData(filename string) map[string]interface{} {
    data, err := ioutil.ReadFile(filepath.Join("testdata", filename))
    if err != nil {
        panic(err)
    }
    var result map[string]interface{}
    json.Unmarshal(data, &result)
    return result
}
```

### Table-Driven Tests

**For variations**:
```go
func TestErrorCodes(t *testing.T) {
    tests := []struct {
        name        string
        code        int
        expectRetry bool
        expectFail  bool
    }{
        {"Code 204", 204, false, false},    // don't retry, but continue
        {"Code 203", 203, true, false},     // retry
        {"Code 500", 500, true, false},     // retry
        {"Code 600", 600, true, true},      // retry, may fail
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test logic using tt.code, tt.expectRetry, etc.
        })
    }
}
```

---

## Running the Tests

### Command Reference

```bash
# Run all unit tests
go test ./internal/obsws ./cmd/memofy-core -v

# Run specific test
go test -run TestConnectionHandshake ./internal/obsws -v

# Run with coverage
go test ./... -cover

# Run with detailed coverage report
go test ./... -coverprofile=coverage.out && go tool cover -html=coverage.out

# Run integration tests (manual - requires real OBS)
bash scripts/test-integration.sh

# Run with timeout (in case of hangs)
go test ./... -timeout 30s -v

# Run in verbose mode with race detector
go test ./... -race -v
```

### CI/CD Integration

**GitHub Actions** (example .github/workflows/test.yml):
```yaml
name: Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: macos-latest
    steps:
      - uses: actions/checkout@v2
      - uses: actions/setup-go@v2
        with:
          go-version: 1.21
      - run: go test ./... -v -timeout 30s -race
```

---

## Coverage Goals

### Target Coverage Percentages

| Module | Target | Current | Status |
|--------|--------|---------|--------|
| client.go | 85% | TBD | TODO |
| sources.go | 90% | TBD | TODO |
| validation.go | 80% | TBD | TODO |
| main.go (startup) | 70% | TBD | TODO |
| **Overall** | **80%** | TBD | TODO |

---

## Known Limitations & Workarounds

### Time-Based Tests
**Issue**: Testing 5-minute timeout requires either:
1. Actual 5-minute wait (slow)
2. Mock time.Now() (complicated dependency injection)

**Workaround**: Use environment variable to shorten timeout in tests:
```go
if os.Getenv("MEMOFY_TEST") == "1" {
    recoverTimeLimit = 5 * time.Second  // Fast test
} else {
    recoverTimeLimit = 5 * time.Minute  // Production
}
```

### Goroutine Leaks
**Issue**: Detection loop runs in background goroutine; must be cleaned up.

**Workaround**: Add context cancellation:
```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()  // In cleanup after test
```

### macOS-Specific Behavior
**Issue**: Permission checks, AppKit calls, may differ in test environment.

**Workaround**: Mock permission checks and skip platform-specific tests in CI:
```go
func TestPermissions(t *testing.T) {
    if os.Getenv("CI") != "" {
        t.Skip("Skipping permission test in CI environment")
    }
    // ...
}
```

---

## Success Criteria for Phase 6

All tests pass:
```bash
✓ 8 client_test.go tests pass
✓ 7 sources_test.go tests pass
✓ 5 startup_test.go tests pass
✓ Integration script runs without errors against real OBS
✓ Coverage >80% across core modules
✓ No race conditions detected (-race flag)
✓ All resource cleanup verified (no goroutine leaks)
```

Once all tests pass, Phase 6 is complete, and the full improvement plan is implemented.
