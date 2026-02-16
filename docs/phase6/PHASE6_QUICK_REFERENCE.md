# Phase 6: Quick Reference Card

## What's Done (✅ Phases 1-5)

### Logging (Phase 1)
```go
// In codebase: 50+ log statements with [TAG] prefixes
[STARTUP]     - Initialization phases
[EVENT]       - Meeting detection events
[RECONNECT]   - WebSocket reconnection attempts
[SOURCE_CHECK] - Source validation steps
[CREATE_RETRY] - Source creation retry attempts
[SHUTDOWN]    - Graceful shutdown sequence
```

**Files Modified**: client.go, sources.go, main.go, memofy-ctl.sh

### Validation (Phase 2)
```go
// New file: internal/validation/obs_compatibility.go
ValidateOBSVersion()     // Check version >= 28.0
CheckOBSHealth()         // Combined validation
SuggestedFixes()         // Error-specific fixes
```

**Called in**: cmd/memofy-core/main.go startup

### Recovery (Phase 3)
```go
// Enhancement: client.go reconnection
Exponential backoff: 5s → 10s → 20s → 40s → 60s (capped)
Jitter: ±10% variance
Logging: Each attempt [RECONNECT] tagged

// New: sources.go CreateSourceWithRetry()
3 attempts with 1s → 2s → 3s backoff
Special code 204 handling: fast-fail
```

**Files Modified**: client.go, sources.go

### Hardening (Phase 4)
```go
// UI timeout: 5s → 15s
// Heartbeat logging every 2s during init
// Death file tracking (why did process die?)
// Stale PID detection with previous death info
```

**Files Modified**: memofy-ui/main.go, memofy-ctl.sh

### Documentation (Phase 5)
```
docs/TROUBLESHOOTING.md       (350 lines) - Error diagnosis
docs/STARTUP_SEQUENCE.md      (350 lines) - 12-phase timeline
docs/PROCESS_LIFECYCLE.md     (400 lines) - State machines
docs/OBS_INTEGRATION.md       (500 lines) - WebSocket protocol
```

---

## What's Next (Phase 6)

### Test File Structure
```
internal/obsws/
├── client_test.go         (8 tests)
├── sources_test.go        (7 tests)
└── testdata/              (JSON fixtures)

cmd/memofy-core/
└── startup_test.go        (5 tests)

scripts/
└── test-integration.sh    (5 manual tests)

testutil/
├── mock_obs.go            (Mock WebSocket server)
└── log_capture.go         (Log assertion helpers)
```

### Test Categories

**Client Tests** (8):
1. Connection handshake
2. Bad OBS version detection
3. Code 204 error handling
4. Code 203 timeout handling
5. Reconnection with backoff
6. Reconnection with jitter
7. Connection loss detection
8. Request/response sequencing

**Source Tests** (7):
1. Ensure required sources
2. Source already exists (disabled)
3. CreateInput fails with code 204
4. Source creation with retry
5. Source validation post-creation
6. Source creation time limit (5 min)
7. Source recovery

**Startup Tests** (5):
1. Successful startup
2. Startup without OBS (launches it)
3. Startup with incompatible OBS
4. Startup without permissions
5. Graceful shutdown on SIGTERM

**Integration Tests** (5):
1. OBS connection reachable
2. Scene list retrieval
3. Source creation
4. Recording start/stop
5. Recovery mode

---

## Key Decisions (Clarified 2026-02-14)

| Decision | Value | Reason |
|----------|-------|--------|
| Recovery time limit | 5 minutes | Prevent CPU waste on indefinite retries |
| Code 204 behavior | Continue with warning | Allow user to fix OBS without restart |
| Source ready = | Exist + Enabled | Recording won't start with disabled sources |

---

## How to Implement Phase 6

### Step 1: Setup (1-2 hours)
```bash
# Create mock infrastructure
touch testutil/mock_obs.go
touch testutil/log_capture.go
mkdir internal/obsws/testdata
# Add JSON fixture files
```

### Step 2: Implement Client Tests (3-4 hours)
```bash
# TestConnectionHandshake
# TestConnectionHandshakeBadVersion
# TestErrorCode204Handling
# TestErrorCode203Timeout
# TestReconnectionWithBackoff
# TestReconnectionWithJitter
# TestConnectionLossDetection
# TestRequestResponseSequencing
```

**Run**: `go test ./internal/obsws -run TestConnection -v`

### Step 3: Implement Source Tests (4-5 hours)
```bash
# TestEnsureRequiredSources
# TestSourceAlreadyExists
# TestCreateInputFailsWithCode204
# TestCreateSourceWithRetry
# TestSourceValidationPostCreation
# TestSourceCreationTimeLimit
# TestSourceRecovery
```

**Run**: `go test ./internal/obsws -run TestSource -v`

### Step 4: Implement Startup Tests (3-4 hours)
```bash
# TestStartupSuccessful
# TestStartupWithoutOBS
# TestStartupWithIncompatibleOBS
# TestStartupWithoutPermissions
# TestSignalHandlingGraceful
```

**Run**: `go test ./cmd/memofy-core -run TestStartup -v`

### Step 5: Integration Script (2-3 hours)
```bash
# Create scripts/test-integration.sh
# Implement 5 tests using real OBS
# Requires: OBS running on localhost:4455
```

**Run**: `bash scripts/test-integration.sh`

---

## Testing Commands

```bash
# Quick: Run all tests
go test ./internal/obsws ./cmd/memofy-core -v -timeout 30s

# Coverage report
go test ./... -cover -v

# Specific test
go test -run TestConnectionHandshake ./internal/obsws -v

# With race detector (detects concurrency issues)
go test ./... -race -v

# Generate HTML coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out

# Run integration tests (requires real OBS)
bash scripts/test-integration.sh
```

---

## Success Criteria

✅ **Phase 6 Complete When**:
- [ ] All 20 tests run without errors
- [ ] All tests pass (exit code 0)
- [ ] Coverage >80% overall
- [ ] Coverage >85% for client.go
- [ ] Coverage >90% for sources.go
- [ ] No race conditions detected (`-race` passes)
- [ ] No goroutine leaks (cleanup verified)
- [ ] Integration script passes against real OBS
- [ ] Code compiles without warnings
- [ ] All tests complete in <30 seconds
- [ ] Documentation updated with results

---

## Reference Documentation

**For Implementation**:
- [GO_IMPLEMENTATION_ROADMAP.md](GO_IMPLEMENTATION_ROADMAP.md) - Step-by-step implementation guide
- [TESTING_PLAN.md](TESTING_PLAN.md) - Detailed test specifications

**For Understanding**:
- [PHASE6_STATUS.md](PHASE6_STATUS.md) - Current progress (83% complete)
- [logging.md](logging.md) - Original spec with clarifications
- [TROUBLESHOOTING.md](TROUBLESHOOTING.md) - Error diagnosis guide
- [STARTUP_SEQUENCE.md](STARTUP_SEQUENCE.md) - Expected timelines

**For Users**:
- [PROCESS_LIFECYCLE.md](PROCESS_LIFECYCLE.md) - How processes work
- [OBS_INTEGRATION.md](OBS_INTEGRATION.md) - WebSocket protocol details

---

## Mock WebSocket Server Pattern

```go
// testutil/mock_obs.go
type MockOBSServer struct {
    listener net.Listener
    conn     *websocket.Conn
    responses map[string]interface{}
    failureMode string
}

func (m *MockOBSServer) Start() error {
    m.listener, _ = net.Listen("tcp", "localhost:4455")
    go m.accept()
    return nil
}

func (m *MockOBSServer) accept() {
    conn, _ := m.listener.Accept()
    ws, _ := websocket.Upgrade(conn, nil, nil, 0, 0)
    // Handle WebSocket v5 handshake
    // Queue responses based on failure mode
}
```

**Usage in tests**:
```go
func TestExample(t *testing.T) {
    mock := testutil.NewMockOBS()
    mock.SetFailureMode("code204")
    mock.Start()
    defer mock.Stop()
    
    client := New("localhost:4455", "")
    err := client.CreateInput(...)
    
    if !strings.Contains(err.Error(), "204") {
        t.Fatal("Expected code 204 error")
    }
}
```

---

## Common Issues & Fixes

| Issue | Solution |
|-------|----------|
| Tests hang for 5 minutes | Use `MEMOFY_TEST_MODE=1` to reduce timeout to 5 seconds |
| Port already in use | Kill existing processes: `killall memofy-core; sleep 1` |
| Connection refused | Check mock server started: `nc -zv localhost 4455` |
| Goroutine leaks | Add `defer cancel()` in test cleanup for context |
| Race conditions | Run with `-race` flag: `go test -race ./...` |
| Coverage gaps | Check detailed coverage: `go tool cover -func=coverage.out` |

---

## Estimated Timeline

| Task | Hours | Cumulative |
|------|-------|-----------|
| Setup infrastructure | 1-2 | 1-2 |
| Client tests (8) | 3-4 | 4-6 |
| Source tests (7) | 4-5 | 8-11 |
| Startup tests (5) | 3-4 | 11-15 |
| Integration script | 2-3 | 13-18 |
| Debug & coverage | 2-3 | 15-21 |
| **Total** | **~18 hours** | **3-4 days** |

**Recommended**: 2-3 hours/day spread over 1-2 weeks for thorough testing

---

## Ready to Start?

1. Read [GO_IMPLEMENTATION_ROADMAP.md](GO_IMPLEMENTATION_ROADMAP.md) (15 minutes)
2. Create `testutil/mock_obs.go` with mock WebSocket server (1 hour)
3. Start with `TestConnectionHandshake` in `client_test.go` (30 min)
4. Run first test: `go test -run TestConnectionHandshake -v ./internal/obsws`
5. Expand from there following the test checklist

**Questions?** All docs are cross-referenced. Check [TESTING_PLAN.md](TESTING_PLAN.md) for detailed test specifications.
