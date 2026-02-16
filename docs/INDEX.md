# Documentation Index & Navigation Guide

## Welcome! Start Here 👋

You're building **Phase 6: Integration Testing** for Memofy (Go project). This document helps you navigate the complete planning documentation.

---

## What Phase Are We On?

✅ **Phases 1-5: COMPLETE** (83% of plan implemented)
- Phase 1: Enhanced Logging ✓
- Phase 2: Validation Checks ✓
- Phase 3: Automatic Error Recovery ✓
- Phase 4: Code Hardening ✓
- Phase 5: Documentation ✓

📋 **Phase 6: Integration Testing** (Ready to implement)
- 20 automated tests (Go `testing` package)
- 1 integration script (Bash)
- 3-4 days to implement

---

## Documentation Hierarchy

### 🎯 Start With (5 minutes)
1. **This file** - You're reading it
2. [PHASE6_QUICK_REFERENCE.md](PHASE6_QUICK_REFERENCE.md) - One-page summary

### 📚 Understand the Plan (30 minutes)
1. [logging.md](logging.md) - Original specification + clarifications (lines 1-10: decisions)
2. [PHASE6_STATUS.md](PHASE6_STATUS.md) - What's done, what's next (83% complete)
3. [TESTING_PLAN.md](TESTING_PLAN.md) - Detailed test specifications

### 🛠️ Implement Phase 6 (18 hours)
1. [GO_IMPLEMENTATION_ROADMAP.md](GO_IMPLEMENTATION_ROADMAP.md) - Step-by-step guide
   - Step 1: Create test infrastructure (1-2h)
   - Step 2: Implement client tests (3-4h)
   - Step 3: Implement source tests (4-5h)
   - Step 4: Implement startup tests (3-4h)
   - Step 5: Create integration script (2-3h)

### 📖 Reference During Implementation
- [TESTING_PLAN.md](TESTING_PLAN.md) - Detailed test specs (8 + 7 + 5 = 20 tests)
- [PHASE6_QUICK_REFERENCE.md](PHASE6_QUICK_REFERENCE.md) - Test checklist and commands

### 🧠 Understand the System (User docs - already complete)
1. [TROUBLESHOOTING.md](TROUBLESHOOTING.md) - How to diagnose errors
2. [STARTUP_SEQUENCE.md](STARTUP_SEQUENCE.md) - What happens at startup (0-25s timeline)
3. [PROCESS_LIFECYCLE.md](PROCESS_LIFECYCLE.md) - Process state machines
4. [OBS_INTEGRATION.md](OBS_INTEGRATION.md) - WebSocket protocol details

---

## Quick Navigation by Purpose

### "I want to understand what's already done"
→ [PHASE6_STATUS.md](PHASE6_STATUS.md) (Current Completion Overview section)

### "I want to see the test specifications"
→ [TESTING_PLAN.md](TESTING_PLAN.md) (8 pages, organized by test file)

### "I want to implement Phase 6 now"
→ [GO_IMPLEMENTATION_ROADMAP.md](GO_IMPLEMENTATION_ROADMAP.md) (Day-by-day plan)

### "I need a test checklist"
→ [PHASE6_QUICK_REFERENCE.md](PHASE6_QUICK_REFERENCE.md#test-categories)

### "What are the critical decisions?"
→ [logging.md](logging.md#clarifications) (3 key decisions clarified)

### "How do I run the tests?"
→ [PHASE6_QUICK_REFERENCE.md](PHASE6_QUICK_REFERENCE.md#testing-commands)

### "How long will this take?"
→ [GO_IMPLEMENTATION_ROADMAP.md](GO_IMPLEMENTATION_ROADMAP.md#implementation-workflow) (3-day timeline)

### "What's the mock WebSocket pattern?"
→ [GO_IMPLEMENTATION_ROADMAP.md](GO_IMPLEMENTATION_ROADMAP.md#step-1-create-test-infrastructure) (Step 1)

### "What to do if tests hang/fail?"
→ [GO_IMPLEMENTATION_ROADMAP.md](GO_IMPLEMENTATION_ROADMAP.md#debugging-tips)

### "Binary gets killed on macOS?"
→ [CODE_SIGNING.md](CODE_SIGNING.md) (Code signing fix for pipeline binaries)

### "What are the success criteria?"
→ [PHASE6_STATUS.md](PHASE6_STATUS.md#deployment-readiness) or [PHASE6_QUICK_REFERENCE.md](PHASE6_QUICK_REFERENCE.md#success-criteria)

---

## Document Overview Table

| Document | Purpose | Length | Read Time |
|----------|---------|--------|-----------|
| [logging.md](logging.md) | Original 6-phase spec + clarifications | 227 lines | 10 min |
| [TESTING_PLAN.md](TESTING_PLAN.md) | Detailed test specifications (20 tests) | 400 lines | 30 min |
| [GO_IMPLEMENTATION_ROADMAP.md](GO_IMPLEMENTATION_ROADMAP.md) | Day-by-day implementation guide | 300 lines | 20 min |
| [PHASE6_STATUS.md](PHASE6_STATUS.md) | Completion status (83%), what's next | 250 lines | 15 min |
| [PHASE6_QUICK_REFERENCE.md](PHASE6_QUICK_REFERENCE.md) | One-page quick reference | 200 lines | 5 min |
| [TROUBLESHOOTING.md](TROUBLESHOOTING.md) | Error diagnosis (user guide) | 350 lines | 15 min |
| [STARTUP_SEQUENCE.md](STARTUP_SEQUENCE.md) | Startup timeline 0-25s (user guide) | 350 lines | 15 min |
| [PROCESS_LIFECYCLE.md](PROCESS_LIFECYCLE.md) | Process state machines (user guide) | 400 lines | 20 min |
| [OBS_INTEGRATION.md](OBS_INTEGRATION.md) | WebSocket protocol (reference) | 500 lines | 25 min |
| [CODE_SIGNING.md](CODE_SIGNING.md) | macOS code signing for binaries | 200 lines | 10 min |

**Total Reading Time**: 2.5 hours for everything, or 50 minutes for essential docs only

---

## Three Critical Clarifications

### 1. Recovery Mode Duration
**Question**: When sources fail to create, how long should memofy retry?
**Answer**: 5 minutes, then disable recording with warning
**Why**: Prevents CPU waste on indefinite retries; allows user to fix issue manually

### 2. Code 204 Error Behavior
**Question**: When OBS version is incompatible (code 204), should app disable recording or continue?
**Answer**: Continue with warning; let user fix OBS without restart
**Why**: Graceful degradation; user can manually create sources while app runs

### 3. Source Ready Criteria
**Question**: What makes a source "ready to record"?
**Answer**: Must exist in scene AND have Enabled=true flag
**Why**: Recording would be silent/empty if source disabled

---

## Implementation Checklist

### Pre-Implementation
- [ ] Read [PHASE6_QUICK_REFERENCE.md](PHASE6_QUICK_REFERENCE.md) (5 min)
- [ ] Read [GO_IMPLEMENTATION_ROADMAP.md](GO_IMPLEMENTATION_ROADMAP.md) (20 min)
- [ ] Review clarifications in [logging.md](logging.md) (5 min)

### Phase 6A: Infrastructure (1-2 hours)
- [ ] Create `testutil/mock_obs.go` (mock WebSocket server)
- [ ] Create `testutil/log_capture.go` (log assertions)
- [ ] Create `internal/obsws/testdata/` folder
- [ ] Add JSON fixture files (6 files)

### Phase 6B: Client Tests (3-4 hours)
- [ ] TestConnectionHandshake
- [ ] TestConnectionHandshakeBadVersion
- [ ] TestErrorCode204Handling
- [ ] TestErrorCode203Timeout
- [ ] TestReconnectionWithBackoff
- [ ] TestReconnectionWithJitter
- [ ] TestConnectionLossDetection
- [ ] TestRequestResponseSequencing

### Phase 6C: Source Tests (4-5 hours)
- [ ] TestEnsureRequiredSources
- [ ] TestSourceAlreadyExists
- [ ] TestCreateInputFailsWithCode204
- [ ] TestCreateSourceWithRetry
- [ ] TestSourceValidationPostCreation
- [ ] TestSourceCreationTimeLimit
- [ ] TestSourceRecovery

### Phase 6D: Startup Tests (3-4 hours)
- [ ] TestStartupSuccessful
- [ ] TestStartupWithoutOBS
- [ ] TestStartupWithIncompatibleOBS
- [ ] TestStartupWithoutPermissions
- [ ] TestSignalHandlingGraceful

### Phase 6E: Integration Script (2-3 hours)
- [ ] test_obs_connection
- [ ] test_scene_list
- [ ] test_source_creation
- [ ] test_recording_start_stop
- [ ] test_recovery_mode

### Post-Implementation
- [ ] All tests pass: `go test ./... -v -timeout 30s`
- [ ] Coverage check: `go test ./... -cover`
- [ ] Race detector: `go test ./... -race`
- [ ] Integration test: `bash scripts/test-integration.sh` (requires real OBS)
- [ ] Update coverage report
- [ ] Write test results to [TEST_RESULTS.md](TEST_RESULTS.md) (optional)

---

## Success Definition

✅ **Phase 6 is Complete When**:
- [ ] All 20 unit tests run and pass
- [ ] All 5 integration tests documented/passing
- [ ] Overall coverage >80%
- [ ] Client module coverage >85%
- [ ] Sources module coverage >90%
- [ ] No race conditions detected
- [ ] No goroutine leaks
- [ ] Code compiles without warnings
- [ ] All docs updated with results

**Status**: Ready to start! 🚀

---

## Quick Command Reference

```bash
# Run all tests
go test ./internal/obsws ./cmd/memofy-core -v -timeout 30s

# Run specific test file
go test ./internal/obsws -v  # All client + source tests

# Run specific test
go test -run TestConnectionHandshake ./internal/obsws -v

# With coverage
go test ./... -cover

# With race detector
go test ./... -race -v

# Generate HTML coverage report
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out && open coverage.html

# Integration tests (requires real OBS running)
bash scripts/test-integration.sh
```

---

## Architecture Overview

```
Memofy Service (macOS)
├── memofy-core (daemon)
│   ├── Meeting detection
│   ├── OBS WebSocket client [← These are tested in Phase 6]
│   └── Recording coordination
│
├── memofy-ui (menu bar app)
│   └── Status display
│
└── OBS (external)
    ├── WebSocket server (port 4455)
    └── Video/audio recording
```

Phase 6 tests validate all interaction paths between memofy-core and OBS.

---

## Where Things Are Located

### Specification & Planning
```
docs/
├── logging.md                        ← Original spec (with clarifications)
├── TESTING_PLAN.md                  ← Test specifications
├── GO_IMPLEMENTATION_ROADMAP.md      ← Step-by-step implementation
├── PHASE6_STATUS.md                 ← Current progress (83% done)
├── PHASE6_QUICK_REFERENCE.md        ← One-page reference
└── INDEX.md                         ← You are here
```

### User Documentation (Already Complete)
```
docs/
├── TROUBLESHOOTING.md               ← Error diagnosis guide
├── STARTUP_SEQUENCE.md              ← What happens 0-25s
├── PROCESS_LIFECYCLE.md             ← State machines
└── OBS_INTEGRATION.md               ← WebSocket protocol
```

### Implementation (Phase 6)
```
internal/obsws/
├── client_test.go                   ← NEW: 8 tests
├── sources_test.go                  ← NEW: 7 tests
└── testdata/                        ← NEW: JSON fixtures

cmd/memofy-core/
└── startup_test.go                  ← NEW: 5 tests

testutil/
├── mock_obs.go                      ← NEW: Mock WebSocket
└── log_capture.go                   ← NEW: Log helpers

scripts/
└── test-integration.sh              ← NEW: Manual tests (5)
```

### Existing Code (Phases 1-5)
```
internal/
├── obsws/
│   ├── client.go                    ✓ Enhanced logging + reconnection
│   ├── sources.go                   ✓ Retry logic + enabled check
│   └── operations.go
│
└── validation/
    └── obs_compatibility.go         ✓ New validation module

cmd/
├── memofy-core/
│   └── main.go                      ✓ Health check integration
└── memofy-ui/
    └── main.go                      ✓ Timeout increase

scripts/
└── memofy-ctl.sh                    ✓ Diagnose command + death tracking
```

---

## Next Steps

### Option A: Start Implementing (18 hours)
1. Follow [GO_IMPLEMENTATION_ROADMAP.md](GO_IMPLEMENTATION_ROADMAP.md)
2. Implement tests day by day
3. Run `go test ./... -v` to verify each test

### Option B: Understand First (30 minutes)
1. Read [TESTING_PLAN.md](TESTING_PLAN.md) to see all tests
2. Read [logging.md](logging.md) to understand the spec
3. Then start Option A

### Option C: Quick Review (5 minutes)
1. Read [PHASE6_QUICK_REFERENCE.md](PHASE6_QUICK_REFERENCE.md)
2. Jump to implementation

---

## FAQ

**Q: Which document should I read first?**
A: [PHASE6_QUICK_REFERENCE.md](PHASE6_QUICK_REFERENCE.md) (5 min), then [GO_IMPLEMENTATION_ROADMAP.md](GO_IMPLEMENTATION_ROADMAP.md) (20 min)

**Q: How many tests do I need to write?**
A: 20 tests total (8 client + 7 source + 5 startup), plus 5 manual integration tests in bash

**Q: How long will this take?**
A: 18-20 hours spread over 3-4 days, or 1-2 weeks at 2-3 hours/day

**Q: What Go features do I need to know?**
A: `testing` package (standard library), gorilla/websocket (likely already used), goroutine management

**Q: Can I skip tests?**
A: Not recommended - tests validate all 5 previous phases. Aim for 80%+ coverage minimum.

**Q: What if tests fail?**
A: Check [GO_IMPLEMENTATION_ROADMAP.md#debugging-tips](GO_IMPLEMENTATION_ROADMAP.md#debugging-tips)

**Q: Do I need to modify existing code?**
A: No - all code from Phases 1-5 is already implemented and compiling. Phase 6 only adds tests.

---

## Support & Questions

If you have questions during implementation:
1. Check [TESTING_PLAN.md](TESTING_PLAN.md) for detailed test specs
2. Check [GO_IMPLEMENTATION_ROADMAP.md](GO_IMPLEMENTATION_ROADMAP.md) for implementation patterns
3. Check [PHASE6_QUICK_REFERENCE.md](PHASE6_QUICK_REFERENCE.md#common-issues--fixes) for common issues
4. Refer to [logging.md](logging.md#clarifications) for the 3 critical decisions

---

## Document Tree (Visual)

```
memofy/ (Golang project - 83% complete)
│
├── SPECIFICATION & PLANNING
│   ├── logging.md (Original spec + 3 key decisions)
│   ├── TESTING_PLAN.md (20 test specs: 8+7+5)
│   ├── GO_IMPLEMENTATION_ROADMAP.md (3-day timeline)
│   ├── PHASE6_STATUS.md (What's done, what's next)
│   ├── PHASE6_QUICK_REFERENCE.md (One-page ref)
│   └── INDEX.md (← You are here)
│
├── USER GUIDES (Complete)
│   ├── TROUBLESHOOTING.md (Error diagnosis)
│   ├── STARTUP_SEQUENCE.md (0-25s timeline)
│   ├── PROCESS_LIFECYCLE.md (State machines)
│   └── OBS_INTEGRATION.md (Protocol ref)
│
├── IMPLEMENTATION (To be created - Phase 6)
│   ├── testutil/mock_obs.go (Mock server)
│   ├── internal/obsws/client_test.go (8 tests)
│   ├── internal/obsws/sources_test.go (7 tests)
│   ├── cmd/memofy-core/startup_test.go (5 tests)
│   └── scripts/test-integration.sh (5 manual tests)
│
└── EXISTING CODE (Phases 1-5 complete)
    ├── internal/obsws/client.go (Enhanced)
    ├── internal/obsws/sources.go (Enhanced)
    ├── internal/validation/obs_compatibility.go (New)
    ├── cmd/memofy-core/main.go (Enhanced)
    ├── cmd/memofy-ui/main.go (Enhanced)
    └── scripts/memofy-ctl.sh (Enhanced)
```

---

**Ready to start Phase 6?** Pick a path:
- 🏃 **Fast**: Skip straight to [GO_IMPLEMENTATION_ROADMAP.md](GO_IMPLEMENTATION_ROADMAP.md#step-1-create-test-infrastructure)
- 📚 **Thorough**: Start with [TESTING_PLAN.md](TESTING_PLAN.md)
- ⚡ **Quick**: Just read [PHASE6_QUICK_REFERENCE.md](PHASE6_QUICK_REFERENCE.md)

Good luck! 🚀
