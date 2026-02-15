# Go Implementation Roadmap for Phase 6

## Executive Summary

You have **20 integration tests** to write across 4 files. Using Go's `testing` package with hand-written mock WebSocket servers, you'll achieve 80%+ code coverage and validate all six improvement phases.

---

## Implementation Order

### Step 1: Create Test Infrastructure (Foundation)
**Files to create**: 
- `internal/obsws/testdata/` (folder for JSON fixtures)
- `testutil/mock_obs.go` (mock WebSocket server)

**Time estimate**: 1-2 hours

**Deliverables**:
- Mock OBS WebSocket server that can simulate various OBS responses
- JSON fixture files for common OBS responses (Hello, Identified, error codes)
- Helper functions for assertions (log capture, response validation)

**Key Code Pattern**:
```go
// testutil/mock_obs.go
type MockOBSServer struct {
    listener net.Listener
    conn     *websocket.Conn
    responses map[string]interface{}
}

func (m *MockOBSServer) HandleConnection() {
    // Implement WebSocket v5 handshake (Hello → Identify → Identified)
    // Queue responses based on request type
    // Support failure modes (code 204, timeout, disconnect)
}
```

---

### Step 2: Implement Client Tests (8 tests)
**File**: `internal/obsws/client_test.go`

**Tests in order**:
1. ✓ TestConnectionHandshake
2. ✓ TestConnectionHandshakeBadVersion
3. ✓ TestErrorCode204Handling
4. ✓ TestErrorCode203Timeout
5. ✓ TestReconnectionWithBackoff
6. ✓ TestReconnectionWithJitter
7. ✓ TestConnectionLossDetection
8. ✓ TestRequestResponseSequencing

**Time estimate**: 3-4 hours

**Key dependencies**:
- Mock WebSocket server from Step 1
- Ability to mock OBS versions, error codes, timeouts
- Thread-safe response tracking

**Example test structure**:
```go
func TestConnectionHandshake(t *testing.T) {
    // 1. Start mock server
    mock := testutil.NewMockOBS()
    mock.Start()
    defer mock.Stop()
    
    // 2. Create client
    client := NewClient("localhost:4455", "")
    
    // 3. Verify versions extracted
    if client.ObsVersion != "29.1.3" {
        t.Fatalf("Expected 29.1.3, got %s", client.ObsVersion)
    }
    
    // 4. Verify connection active
    if !client.Connected() {
        t.Fatal("Expected connected")
    }
}
```

---

### Step 3: Implement Source Tests (7 tests)
**File**: `internal/obsws/sources_test.go`

**Tests in order**:
1. ✓ TestEnsureRequiredSources
2. ✓ TestSourceAlreadyExists
3. ✓ TestCreateInputFailsWithCode204
4. ✓ TestCreateSourceWithRetry
5. ✓ TestSourceValidationPostCreation
6. ✓ TestSourceCreationTimeLimit
7. ✓ TestSourceRecovery

**Time estimate**: 4-5 hours

**Key dependencies**:
- Mock WebSocket server (reuse from Step 1)
- Ability to mock CreateInput, GetInputSettings, SetInputEnabled responses
- Log capture utilities
- Time mocking for 5-minute retry window test

**Tricky test** (TestSourceCreationTimeLimit):
```go
func TestSourceCreationTimeLimit(t *testing.T) {
    // Option A: Use time.Sleep(301 * time.Second) - slow but reliable
    // Option B: Mock time.Now() by adjusting internal clock
    // Recommendation: Use environment variable to speed up test
    
    mock := testutil.NewMockOBS()
    mock.SetFailureMode("code500") // Always fail
    mock.Start()
    defer mock.Stop()
    
    client := NewClient("localhost:4455", "")
    
    // Set test mode: 5 min → 5 sec for testing
    os.Setenv("MEMOFY_TEST_MODE", "1")
    
    // Start recovery loop
    go EnsureSourcesWithRecovery(client, "Collection 1")
    
    // Wait for timeout
    time.Sleep(6 * time.Second)
    
    // Verify logging shows recording disabled
    if !logCapture.Contains("Recording disabled") {
        t.Fatal("Expected recording disabled message")
    }
}
```

---

### Step 4: Implement Startup Tests (5 tests)
**File**: `cmd/memofy-core/startup_test.go`

**Tests in order**:
1. ✓ TestStartupSuccessful
2. ✓ TestStartupWithoutOBS
3. ✓ TestStartupWithIncompatibleOBS
4. ✓ TestStartupWithoutPermissions
5. ✓ TestSignalHandlingGraceful

**Time estimate**: 3-4 hours

**Key dependencies**:
- Mock OBS server
- Mock permission check (OS-level)
- Mock config file loading
- Signal handling utilities

**Most complex** (TestSignalHandlingGraceful):
```go
func TestSignalHandlingGraceful(t *testing.T) {
    // 1. Start memofy-core in background
    cmd := exec.Command("go", "run", "./cmd/memofy-core", 
        "--test-mode", "--mock-obs", "localhost:4455")
    cmd.Start()
    defer cmd.Wait()
    
    // 2. Give it time to initialize
    time.Sleep(2 * time.Second)
    
    // 3. Send SIGTERM
    cmd.Process.Signal(os.Interrupt)
    
    // 4. Wait for graceful shutdown
    done := make(chan error)
    go func() { done <- cmd.Wait() }()
    
    select {
    case <-time.After(5 * time.Second):
        t.Fatal("Process did not exit within 5 seconds")
    case err := <-done:
        if cmd.ProcessState.ExitCode() != 0 {
            t.Fatalf("Expected exit code 0, got %d", cmd.ProcessState.ExitCode())
        }
    }
}
```

---

### Step 5: Create Integration Test Script
**File**: `scripts/test-integration.sh`

**Tests to implement**:
1. ✓ test_obs_connection
2. ✓ test_scene_list
3. ✓ test_source_creation
4. ✓ test_recording_start_stop
5. ✓ test_recovery_mode

**Time estimate**: 2-3 hours

**Language**: Bash with WebSocket calls via nc (netcat) or socat

**Example structure**:
```bash
#!/bin/bash

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

# Test: OBS connection
test_obs_connection() {
    echo -n "Testing OBS connection... "
    
    # Use nc to test port
    nc -zv localhost 4455 2>/dev/null
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ PASS${NC}"
    else
        echo -e "${RED}✗ FAIL${NC} (OBS not running on port 4455)"
        return 1
    fi
}

# Test: Create sources
test_source_creation() {
    echo -n "Testing source creation... "
    
    # Send WebSocket request to create Display Capture
    # Parse response for success
    # Cleanup: delete source
    
    # Similar for audio source
}

# Main runner
run_all_tests() {
    test_obs_connection || return 1
    test_scene_list || return 1
    test_source_creation || return 1
    test_recording_start_stop || return 1
    test_recovery_mode || return 1
}

run_all_tests
```

---

## File Structure Checklist

Create these files in this order:

### Phase 6A: Test Infrastructure
- [ ] `internal/obsws/testdata/` (directory)
- [ ] `internal/obsws/testdata/hello_response.json`
- [ ] `internal/obsws/testdata/identified_response.json`
- [ ] `internal/obsws/testdata/code_204_response.json`
- [ ] `internal/obsws/testdata/code_203_response.json`
- [ ] `internal/obsws/testdata/create_input_success.json`
- [ ] `testutil/mock_obs.go`
- [ ] `testutil/log_capture.go`
- [ ] `testutil/assertions.go`

### Phase 6B: Unit Tests - Client
- [ ] `internal/obsws/client_test.go` (8 tests)

### Phase 6C: Unit Tests - Sources
- [ ] `internal/obsws/sources_test.go` (7 tests)

### Phase 6D: Integration Tests - Startup
- [ ] `cmd/memofy-core/startup_test.go` (5 tests)

### Phase 6E: Manual Integration Tests
- [ ] `scripts/test-integration.sh` (5 tests)

### Phase 6F: Documentation
- [ ] `docs/TESTING_PLAN.md` ✓ (already created)
- [ ] `docs/TEST_RESULTS.md` (populate after running tests)

---

## Implementation Workflow

### Day 1: Foundation + Client Tests
```
9:00-10:30  → Create mock_obs.go (WebSocket server)
10:30-11:30 → Create JSON fixtures (testdata/)
11:30-12:00 → Create log_capture.go
12:00-1:00  → Lunch
1:00-4:00   → Implement 8 client tests
4:00-5:00   → Debug, verify all client tests pass
```

### Day 2: Source Tests + Startup Tests
```
9:00-12:00  → Implement 7 source tests
12:00-1:00  → Lunch
1:00-4:30   → Implement 5 startup tests
4:30-5:00   → Debug, verify all tests pass
```

### Day 3: Integration Script + Final Verification
```
9:00-11:00  → Implement test-integration.sh
11:00-12:00 → Run against real OBS instance
12:00-1:00  → Lunch
1:00-3:00   → Bug fixes, coverage gaps
3:00-4:00   → Generate coverage report
4:00-5:00   → Document results, finalize
```

---

## Testing Commands

### Run All Tests
```bash
# Quick test (5 min)
go test ./internal/obsws ./cmd/memofy-core -v -timeout 30s

# With coverage
go test ./... -cover -v

# With race detector (5 min)
go test ./... -race -v -timeout 30s

# Specific test
go test -run TestConnectionHandshake ./internal/obsws -v
```

### Generate Coverage Report
```bash
# Create HTML report
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html

# View in browser
open coverage.html

# View coverage by package
go tool cover -func=coverage.out | tail -20
```

### Manual Integration Testing (requires real OBS)
```bash
# Make sure OBS is running and listening on port 4455
# OBS > Tools > obs-websocket Settings > ✓ Enable WebSocket server

# Run manual tests
bash scripts/test-integration.sh

# Watch logs during test
tail -f /tmp/memofy-*.log
```

---

## Go Testing Best Practices

### 1. Use Table-Driven Tests
```go
func TestMultipleScenarios(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    bool
    }{
        {"Case A", "input_a", true},
        {"Case B", "input_b", false},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := Process(tt.input)
            if got != tt.want {
                t.Errorf("Got %v, want %v", got, tt.want)
            }
        })
    }
}
```

### 2. Defer Cleanup
```go
func TestWithCleanup(t *testing.T) {
    mock := NewMockOBS()
    mock.Start()
    defer mock.Stop()  // Always runs, even on failure
    
    // Test code...
}
```

### 3. Use Sub-tests
```go
t.Run("Connection", func(t *testing.T) {
    // Connection-specific tests
})

t.Run("Reconnection", func(t *testing.T) {
    // Reconnection-specific tests
})
```

### 4. Test Error Cases Explicitly
```go
func TestErrorHandling(t *testing.T) {
    client := New("invalid", "")
    err := client.Connect()
    
    if err == nil {
        t.Fatal("Expected error, got nil")
    }
    
    if err.Error() != "connection refused" {
        t.Errorf("Expected 'connection refused', got '%s'", err.Error())
    }
}
```

### 5. Avoid Global State
```go
// BAD
var mockServer *MockOBS

func TestA(t *testing.T) {
    mockServer = NewMockOBS()  // Global state
}

// GOOD
func TestA(t *testing.T) {
    mock := NewMockOBS()  // Local variable
    defer mock.Stop()
}
```

---

## Debugging Tips

### If Tests Hang
```bash
# Kill hanging tests
pkill -f "go test"

# Run with timeout
go test ./... -timeout 5s

# See goroutines
go test ./... -v 2>&1 | grep goroutine
```

### If Tests Fail
```bash
# Run single failing test with verbose output
go test -run TestNameHere -v ./path/to/package

# Add debug logging
t.Logf("Debug info: %v", variable)  // Only shows on failure unless -v

# Check mock server logs
tail -f /tmp/mock-obs.log
```

### If WebSocket Connection Issues
```bash
# Test port connectivity
nc -zv localhost 4455

# Check what's listening
lsof -i :4455

# TCPdump to inspect WebSocket frames
tcpdump -nn -i lo0 'port 4455'
```

---

## Success Checklist

Before marking Phase 6 complete:

- [ ] All 20 tests run without errors
- [ ] All tests pass (no ❌ marks)
- [ ] Coverage report shows 80%+ overall
- [ ] Coverage shows >85% for client.go
- [ ] Coverage shows >90% for sources.go
- [ ] No race conditions detected (`-race` flag passes)
- [ ] No goroutine leaks detected
- [ ] All cleanup performed (mocks stopped, files deleted)
- [ ] Integration tests pass against real OBS
- [ ] All code builds without warnings
- [ ] Test execution completes in <30 seconds
- [ ] Documentation is updated with test results

---

## Next: Start with Step 1

Ready to begin? Start with creating the mock WebSocket server infrastructure:

**Task 1**: Create `testutil/mock_obs.go` with:
- `MockOBSServer` struct
- `Start()` method (listen on localhost:4455)
- `Stop()` method (cleanup)
- Response queue mechanism
- Failure mode support (code 204, 203, timeout, disconnect)

This is the foundation that all other tests depend on. Estimated time: 1 hour.

Would you like me to create the mock_obs.go file and JSON fixtures now?
