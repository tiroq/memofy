# Technical Research: Automatic Meeting Recorder

**Feature**: 001-auto-meeting-recorder  
**Date**: February 12, 2026  
**Purpose**: Resolve technical unknowns before design phase

## Research Areas

### 1. OBS WebSocket v5 API

**Decision**: Use OBS WebSocket v5 protocol with gorilla/websocket client

**Rationale**:
- OBS 28+ ships with obs-websocket v5 built-in (no plugin installation)
- Protocol uses JSON-RPC 2.0 over WebSocket
- Three critical operations needed: GetRecordStatus, StartRecord, StopRecord
- Authentication via SHA-256 challenge-response (if password configured)
- Supports event subscriptions for async notifications (recording state changes)

**Alternatives considered**:
- obs-websocket v4: Deprecated, requires separate plugin installation
- Direct OBS control via scripting: No remote control capability, too coupled
- REST API: OBS doesn't provide one, WebSocket is the official interface

**Implementation notes**:
- Connection URL: `ws://localhost:4455` (default port)
- Must handle reconnection with exponential backoff (5s, 10s, 20s, max 60s)
- Auth flow: Hello → Identify → Identified handshake
- Use gorilla/websocket for standards-compliant client
- Event subscription: Filter to RecordStateChanged events only

---

### 2. macOS Meeting Detection Patterns

**Decision**: Multi-signal detection using NSWorkspace + accessibility APIs

**Rationale**:
- Zoom detection: Check for `zoom.us` process + (CptHost component process OR window title contains "Zoom Meeting")
- Teams detection: Check for `Microsoft Teams` process + window title matching configured hints
- macOS NSWorkspace API provides running application list
- Accessibility AXUIElement APIs for window title inspection (requires Accessibility permission)
- Process detection is lightweight (~1ms), window title inspection adds ~10-50ms per poll

**Alternatives considered**:
- Calendar integration: Unreliable (users join ad-hoc meetings, calendar may not reflect actual join time)
- Network traffic analysis: Too invasive, requires root, complex to implement
- Audio level detection: False positives from music/videos, requires microphone access
- Screen content analysis: Too CPU-intensive, privacy concerns

**Implementation notes**:
- Use `NSWorkspace.sharedWorkspace.runningApplications` for process list
- Process names stable across versions: "zoom.us", "Microsoft Teams"
- Window title hints configurable in JSON (multi-language support): "Meeting", "Reunión", "会議", etc.
- Accessibility permission prompt: System shows dialog, user must enable in Security & Privacy
- Poll interval: 2-3 seconds balances responsiveness vs CPU usage

---

### 3. macOS Menu Bar Implementation with Go

**Decision**: Use progrium/macdriver for Cocoa bridge with cgo

**Rationale**:
- macdriver provides Go bindings to Cocoa frameworks (Foundation, AppKit)
- NSStatusBar for menu bar icon placement
- NSStatusItem for menu construction and state display
- Native look and feel, no frameworks to bundle
- Active maintenance, used in production Go macOS apps

**Alternatives considered**:
- systray library: Limited to basic menus, no advanced state display
- Electron/Wails: Massive bundle size (100MB+), overkill for simple menu bar
- Pure cgo: Too low-level, significant development effort, error-prone
- Swift/Objective-C separate app: Introduces multi-language complexity, harder IPC

**Implementation notes**:
- Template images for icon states: Use SF Symbols or custom PNG (idle/wait/rec/error)
- Menu items: NSMenu with NSMenuItem, checkmarks for mode indicators
- Click handlers: Go functions wrapped with objc.Callback
- Icon updates: Thread-safe via objc.WithAutoreleasePool on main thread
- Build requires macOS SDK, cgo enabled: `CGO_ENABLED=1 GOOS=darwin go build`

---

### 4. macOS Permissions Handling

**Decision**: Declarative permission requests with runtime checks

**Rationale**:
- Screen Recording permission: Required for OBS Display Capture (checked via TCC.db or CGPreflightScreenCaptureAccess)
- Accessibility permission: Required for window title inspection (checked via AXIsProcessTrusted)
- macOS shows system dialogs on first access attempt
- No programmatic grant possible - user must manually enable in System Preferences
- Permission persistence: Stored in TCC (Transparency, Consent, and Control) database

**Alternatives considered**:
- Assume permissions granted: Fails silently, poor UX
- Request permissions at every launch: Annoying for users with existing grants
- Skip permission checks: OBS and detection fail with cryptic errors

**Implementation notes**:
- Check Screen Recording: `CGPreflightScreenCaptureAccess()` (returns bool)
- Check Accessibility: `AXIsProcessTrusted()` (returns bool)
- Show guidance notification if missing: Deep link to System Preferences pane
- Example: `x-apple.systempreferences:com.apple.preference.security?Privacy_ScreenCapture`
- Check at daemon startup and show ERROR state if missing
- Re-check periodically (every 60s) to detect user granted permissions without restart

---

### 5. File-Based IPC for Daemon-UI Communication

**Decision**: JSON status file + plain text command file with fsnotify watchers

**Rationale**:
- Simple, debuggable, no sockets/ports to manage
- status.json: Daemon writes, UI reads (status snapshot every state change)
- cmd.txt: UI writes, daemon reads and clears (one-shot commands)
- fsnotify provides cross-platform file watching with inotify/kqueue
- Atomic write via temp file + rename ensures no partial reads
- Fallback polling if fsnotify events missed (every 1s)

**Alternatives considered**:
- Unix domain socket: Requires connection management, more complex error handling
- Named pipes (FIFO): Blocking behaviors, harder to debug
- DBus: macOS support limited, heavyweight for simple use case
- Shared memory: Complex synchronization, no persistence for debugging

**Implementation notes**:
- Location: `~/.cache/memofy/` (XDG cache convention, won't sync to iCloud)
- Atomic writes: `os.WriteFile(tmpPath); os.Rename(tmpPath, finalPath)`
- File lock not needed (single writer per file, atomic rename)
- status.json updated on: mode change, detection state change, recording start/stop, error
- cmd.txt format: Single line command (`start`, `stop`, `toggle`, `auto`, `pause`, `quit`)
- Daemon clears cmd.txt after reading to prevent re-execution
- UI polls if file watch initialization fails (degraded mode)

---

### 6. macOS LaunchAgent for Background Service

**Decision**: User-level LaunchAgent with plist configuration

**Rationale**:
- LaunchAgent runs as user (not root), necessary for Screen Recording permissions
- Automatic start at user login via launchd
- Restart on crash with ThrottleInterval/StartInterval
- StandardOutPath/StandardErrorPath for logging
- plist configuration declarative and verifiable

**Alternatives considered**:
- Login Items: No automatic restart on crash, deprecated API
- LaunchDaemon (system-level): Runs as root, cannot access user's permissions/screen
- Manual start: Requires user to remember, no crash recovery

**Implementation notes**:
- Plist path: `~/Library/LaunchAgents/com.memofy.core.plist`
- Key properties:
  - `RunAtLoad: true` (start at login)
  - `KeepAlive: true` (restart on crash)
  - `ThrottleInterval: 10` (don't restart too frequently)
  - `StandardOutPath: /tmp/memofy-core.out.log`
  - `StandardErrorPath: /tmp/memofy-core.err.log`
- Install: `cp plist ~/Library/LaunchAgents/ && launchctl load -w ~/Library/LaunchAgents/com.memofy.core.plist`
- Uninstall: `launchctl unload ~/Library/LaunchAgents/com.memofy.core.plist && rm plist`
- Status check: `launchctl list | grep memofy`

---

## Summary of Technology Choices

| Component | Technology | Primary Reason |
|-----------|-----------|----------------|
| Language | Go 1.21+ | System programming, single binary, excellent concurrency |
| OBS Communication | gorilla/websocket + obs-ws v5 | Official protocol, robust client library |
| Meeting Detection | NSWorkspace + Accessibility APIs | Native macOS, minimal overhead, configurable |
| Menu Bar UI | progrium/macdriver (Cocoa) | Native look, no bundle bloat, Go-first API |
| IPC | File-based JSON + fsnotify | Simple, debuggable, atomic updates |
| Background Service | LaunchAgent | User context, auto-start, crash recovery |
| Permissions | Runtime TCC checks | Comply with macOS security model, guide users |

All choices favor simplicity, native integration, and minimal dependencies over framework complexity.
