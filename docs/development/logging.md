I'll create a detailed plan to investigate the root causes of the service failures you're experiencing. Let me start by gathering context about the codebase and the specific errors.

## Documentation References

For Phase 6 implementation guidance, see:
- **Testing Plan**: [docs/TESTING_PLAN.md](TESTING_PLAN.md) - Detailed test specifications (20 tests across 4 files)
- **Implementation Roadmap**: [docs/GO_IMPLEMENTATION_ROADMAP.md](GO_IMPLEMENTATION_ROADMAP.md) - Step-by-step Go implementation guide
- **Phase Status**: [docs/PHASE6_STATUS.md](PHASE6_STATUS.md) - Current completion status (83% done)

## Clarifications

### Session 2026-02-14

- Q: When sources fail to create and memofy retries in background every 10s, what should happen? → A: Stop after 5 minutes of failed retries (then disable recording with user warning)
- Q: When error 204 (OBS version incompatible) occurs, should memofy disable recording or continue? → A: Continue running, warn user, allow manual override/source creation
- Q: For sources to be 'ready to record', must they exist only or also be enabled? → A: Must exist AND have Enabled=true flag 

Research root causes of service failures

Searched codebase for "OBS WebSocket connection setup error handling", 9 results

Searched codebase for "request type is not valid error code 204", 24 results

Searched codebase for "scene setup sources display capture audio input", 14 results

Searched codebase for "SIGKILL process killed startup shutdown", 13 results

Read [](file:///Users/mysterx/dev/memofy/internal/obsws/client.go#1-1), lines 1 to 450

Read [](file:///Users/mysterx/dev/memofy/internal/obsws/operations.go)

Read [](file:///Users/mysterx/dev/memofy/internal/obsws/sources.go#1-1), lines 1 to 300

Read [](file:///Users/mysterx/dev/memofy/cmd/memofy-core/main.go#1-1), lines 1 to 150

Read [](file:///Users/mysterx/dev/memofy/scripts/quick-install.sh#200-200), lines 200 to 450

Read [](file:///Users/mysterx/dev/memofy/scripts/quick-install.sh#1-1), lines 1 to 350

Read [](file:///Users/mysterx/dev/memofy/cmd/memofy-ui/main.go#1-1), lines 1 to 100

Searched for regex `code.*204|error.*204|InvalidRequest` (`**/specs/001-auto-meeting-recorder/contracts/**`), no results

Read [](file:///Users/mysterx/dev/memofy/specs/001-auto-meeting-recorder/contracts/obs-websocket-api.md#200-200), lines 200 to 330

Read [](file:///Users/mysterx/dev/memofy/specs/001-auto-meeting-recorder/contracts/obs-websocket-api.md#1-1), lines 1 to 280

Perfect! Now I have comprehensive insight. Let me ask clarifying questions to determine the exact improvements you need: 

Asked 3 questions (AllErrors, Improvemts, Testing)

Excellent! Based on your answers, here's a comprehensive improvement plan:

## Plan: Comprehensive Error Investigation & System Hardening

**TL;DR**: You're experiencing three interconnected failures: (1) OBS WebSocket protocol mismatch (error 204), (2) abrupt process termination preventing graceful cleanup, and (3) missing scene sources due to failed operations. The plan addresses all three with enhanced diagnostics, automatic recovery, validation gates, documentation, signal handling hardening, and integration tests to prevent regressions. This ensures production-ready reliability with clear troubleshooting paths for users.

---

### **Root Cause Summary**

1. **Code 204 Error** → OBS version or request type mismatch; app then can't ensure sources
2. **SIGKILL Signals** → Hang/timeout during startup; lack of timeout protection; signal handlers not invoked
3. **Missing Sources** → Failed `CreateInput` requests cascade into unusable recording setup

These are **cascading failures**: if (1) fails, (2) happens faster, preventing (3) from being fixed.

---

### **Steps**

#### **Phase 1: Enhanced Logging & Diagnostics**

1. **main.go** — Add structured startup logging with clear phase markers:
   - Log OBS version after connection: `[STARTUP] Connected to OBS v{version}, plugin v{plugin-version}`
   - Log each source check with results: `[SOURCE_CHECK] Audio: exists={bool}, enabled={bool}`
   - Log reconnection attempts with delays: `[RECONNECT] Attempt {n}, delay {s}s, next={s}s`
   - Log signal handling: `[SHUTDOWN] Received {signal}, stopping gracefully...`

2. **client.go** — Enhance error messages:
   - Include request type in error: `"request failed: {requestType} → {error} (code {code})"`
   - Log request/response JSON for code 204: `"[DEBUG] Request: {json}, Response status: {code}"`
   - Log version negotiation: `[CONNECTION] OBS supports: {supported-apis}`

3. **sources.go** — Track source operations:
   - Log scene info: `[SOURCES] Active scene: {name}, existing-sources: {count}`
   - Log creation attempts: `[CREATE] Creating {source-type}, response code: {code}`
   - Log validation after creation: `[VERIFY] {source-name} created={bool}, enabled={bool}`

4. **memofy-ctl.sh** — Add diagnostic command:
   - New `memofy-ctl diagnose` command that:
     - Lists all processes with PIDs and memory usage
     - Shows OBS WebSocket connectivity (can reach port 4455)
     - Checks OBS version via API
     - Lists active OBS scene and sources
     - Displays recent error logs with timestamps

---

#### **Phase 2: Validation Checks (Pre-Failure Gates)**

1. **main.go** — Add pre-startup validation:
   - Verify OBS is running on `localhost:4455` before connecting (timeout 2s)
   - Check OBS version >= 28.x required for WebSocket v5
   - Validate scene has at least one source or is empty (OK to populate)
   - Fail fast with helpful message instead of hanging

2. **sources.go** — Improve source handling:
   - After creating source, verify it's enabled (`Enabled: true`)
   - If source exists but disabled, attempt enable via `SetInputSettings`
   - Add retry logic: if creation fails with code 204, log OBS version incompatibility but **continue operation**
   - Add validation: before considering success, verify source is enabled (not just created)
   - User can manually create/enable sources in OBS at any time; memofy will detect them

3. **Add new file internal/validation/obs_compatibility.go**:
   - Function `ValidateOBSCompatibility(client)` → checks version, plugin, API support
   - Function `ValidateScene(client, sceneName)` → ensures scene exists and is writable
   - Function `SuggestedFixes(error)` → returns user-friendly troubleshooting steps

---

#### **Phase 3: Automatic Error Recovery**

1. **client.go** — Enhance reconnection:
   - On code 204 (invalid request): log OBS version, **do NOT disable recording**, log with "version incompatibility hint"
   - Continue app operation; user may create sources manually or update OBS without restart
   - On connection timeout: increase initial heartbeat check, add 30s total timeout
   - Add backoff jitter: `delay = delay * 2 + random(0, 5)` to avoid thundering herd

2. **sources.go** — Retry source creation:
   - Implement 3-attempt retry on `CreateInput` with 1s backoff
   - On failure, log: `"Failed to create {source} after 3 attempts. Reason: {code}. Enable manually in OBS."`
   - Parse code 204 specially: suggest "OBS version mismatch" rather than generic error

3. **main.go** — Add recovery mode with time limit:
   - If source creation fails but OBS is connected, continue with warning (graceful degradation)
   - Periodically (every 10s) retry missing sources **for up to 5 minutes only**
   - When 5 min elapsed: stop retrying, disable recording, log warning to user
   - When sources become available within 5 min, log: `[RECOVERY] Sources now available, recording enabled`
   - User can restart memofy to reset 5-min counter if needed

---

#### **Phase 4: Code Hardening Against SIGKILL**

1. **main.go** — Prevent timeout-induced SIGKILL:
   - Increase UI init timeout from 5s to 15s (slower Macs need more time)
   - Add periodic "heartbeat" logging during AppKit init: log every 2s "UI init in progress..."
   - Wrap AppKit calls with thread dispatcher: `dispatch_async(dispatch_get_main_queue(), ^{...})`
   - Add pre-checks: verify darwinkit is initialized before attempting UI

2. **memofy-ctl.sh** — Improve PID file cleanup:
   - Before killing process, save state to *.died file (not just PID)
   - Died file includes: PID, signal sent, timestamp, reason
   - On next start, check died file and unlock if previous process fully gone (e.g., via `kill -0`)
   - Log: `[CLEANUP] Removing stale PID and died files from previous crash`

3. **Add new file internal/pidfile/recovery.go**:
   - Function `RecoverStaleProcess(pidFile)` → validates PID is truly dead before cleanup
   - Function `LogProcessCrash(pidfile, signal)` → records why process died
   - Prevent double-start on fast restarts

---

#### **Phase 5: Comprehensive Documentation**

1. **Create docs/TROUBLESHOOTING.md** with sections:
   - **"Process Killed with Signal 9"** → explains SIGKILL, when it happens, solutions
   - **"Error Code 204"** → OBS compatibility matrix (version requirements)
   - **"Sources Not Ensuring"** → decision tree for missing Display/Audio
   - **"Connection Drops"** → reconnection logic explained, timeouts
   - **"Startup Hangs"** → how to detect, when to force-kill, recovery steps
   - Each section includes: symptoms, root cause, solutions, commands to verify

2. **Create docs/STARTUP_SEQUENCE.md**:
   - Detailed timeline of what should happen during `memofy-ctl start`
   - Expected log lines and what they mean
   - Where each error can occur and what to check next

3. **Update README.md** with Troubleshooting section:
   - Link to new docs
   - Quick reference: 3-step diagnostic (`memofy-ctl diagnose`)

---

#### **Phase 6: Integration Tests**

1. **Create new test file client_test.go**:
   - `TestConnectionHandshake()` → mock WebSocket, verify Hello/Identify sequence
   - `TestErrorCode204Handling()` → mock 204 response, verify proper logging/exit
   - `TestReconnectionWithBackoff()` → verify delays double: 5s, 10s, 20s, 40s, 60s
   - `TestSourceCreationRetry()` → mock 3 failures then success, verify retries

2. **Create new test file sources_test.go**:
   - `TestEnsureRequiredSources()` → mock scene with missing sources, verify creation
   - `TestSourceAlreadyExists()` → mock source exists but disabled, verify enable
   - `TestCreateInputFailsWithCode204()` → verify error logged, recovery attempted

3. **Create new test file cmd/memofy-core/startup_test.go**:
   - `TestStartupWithoutOBS()` → fail fast, don't hang
   - `TestStartupWithIncompatibleOBS()` → detect version mismatch early
   - `TestSignalHandlingGraceful()` → SIGTERM → stop recording → disconnect → exit

4. **Create scripts/test-integration.sh**:
   - Manual test suite to run against real OBS:
     - `test_obs_connection` — can we reach OBS?
     - `test_scene_list` — can we list scenes?
     - `test_source_creation` — can we create audio/display sources?
     - `test_recording` — start/stop recording works?
   - Each test logs results in structured format for debugging

---

---

### **Verification**

**Success Criteria — Sources 'Ready to Record'**:
- Audio source: exists AND `Enabled: true`
- Display source: exists AND `Enabled: true`
- Both sources in scene AND responding to `GetInputSettings` calls
- Logging shows: `[VERIFY] All required sources present and enabled`

**Automatic Tests**:
```bash
go test ./internal/obsws -v
go test ./cmd/memofy-core -v
```
Expected: All tests pass, code 204 and timeout scenarios handled gracefully

**Manual Integration Tests** (against running OBS):
```bash
bash scripts/test-integration.sh
memofy-ctl diagnose
memofy-ctl start core
tail -f /tmp/memofy-core.out.log  # Should see [STARTUP], [SOURCE_CHECK], [RECOVERY] logs
```
Expected: Clear phase markers, no hangs, graceful shutdown on SIGTERM

**Regression Prevention**:
- `memofy-ctl clean` → removes PID files properly
- `memofy-ctl start core` twice → second attempt gracefully detects running process
- Kill OBS → memofy reconnects with 5s initial backoff
- Send `SIGTERM` to memofy-core → exits within 2 seconds with cleanup
- Source creation fails → retries for 5 min, then disables recording with warning

---

### **Decisions**

- **Logging format**: Structured `[TAG] message` format (consistent with existing startup logs)
- **Timeouts**: UI 15s (was 5s), OBS connection 10s total (verify alive + handshake)
- **Retry strategy**: 3 attempts with 1s backoff for sources, exponential (5s→60s) for reconnect
- **Exit behavior**: Code 204 → **continue with warning** (don't disable), connection error → retry, signal → graceful shutdown
- **Recovery mode**: Retry failed sources every 10s **for 5 minutes max**, then disable recording with user warning
- **Source ready criteria**: Must exist in scene **AND have Enabled=true** to be considered ready for recording
- **PID cleanup**: Always valid before re-use; log why process died (signal, timeout, panic)
- **Graceful degradation**: If sources can't be created, app continues running but disables auto-record (user can manually create sources)
