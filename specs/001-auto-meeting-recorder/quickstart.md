# Quick Start Guide: Memofy Development

**Feature**: Automatic Meeting Recorder (v0.1)  
**Target**: macOS 11+ (Big Sur and later)  
**Language**: Go 1.21+

## Prerequisites

### System Requirements

- macOS 11.0 (Big Sur) or later
- Xcode Command Line Tools installed
- Go 1.21 or later
- OBS Studio 28.0+ (includes obs-websocket v5)

### Install Dependencies

```bash
# Install Xcode Command Line Tools (if not already installed)
xcode-select --install

# Install Go (if not already installed)
brew install go

# Install OBS Studio
brew install --cask obs

# Verify installations
go version      # Should show 1.21+
obs --version   # Should show 28.0+
```

---

## Project Setup

### Clone and Initialize

```bash
# Clone repository
git clone https://github.com/tiroq/memofy.git
cd memofy

# Initialize Go module
go mod init github.com/tiroq/memofy

# Install dependencies
go get github.com/gorilla/websocket@latest
go get github.com/progrium/macdriver@latest
go get github.com/fsnotify/fsnotify@latest

# Download all dependencies
go mod download
go mod tidy
```

### Project Structure

```bash
# Create directory structure
mkdir -p cmd/memofy-core
mkdir -p cmd/memofy-ui
mkdir -p internal/{detector,statemachine,obsws,ipc,config}
mkdir -p pkg/macui
mkdir -p tests/{integration,fixtures}
mkdir -p configs
mkdir -p scripts
```

---

## Configuration

### OBS Setup

1. **Launch OBS Studio**
   ```bash
   open -a OBS
   ```

2. **Enable WebSocket Server**
   - OBS → Preferences → WebSocket Server Settings
   - Enable "Enable WebSocket server" checkbox
   - Default port: 4455
   - Password: Leave empty for development (optional for production)
   - Click "Apply" and "OK"

3. **Configure Recording Settings**
   - Settings → Output → Recording
   - Recording Path: `~/Movies/Memofy/` (or your preferred location)
   - Recording Format: mp4
   - Encoder: Hardware (H264) or Software (x264)

4. **Add Display Capture Source**
   - Sources → Add → Display Capture
   - Name: "Screen"
   - Display: Select your main display
   - Click "OK"

### macOS Permissions

Memofy requires two macOS permissions:

1. **Screen Recording** (for OBS to capture display)
   - System Preferences → Security & Privacy → Privacy → Screen Recording
   - Grant permission to OBS Studio
   - Grant permission to memofy-core (after first run)

2. **Accessibility** (for window title detection)
   - System Preferences → Security & Privacy → Privacy → Accessibility
   - Grant permission to memofy-core (after first run)

**Note**: macOS will prompt for these permissions on first use. Application must be restarted after granting.

---

## Build

### Build Both Binaries

```bash
# Build daemon
CGO_ENABLED=1 GOOS=darwin go build -o bin/memofy-core cmd/memofy-core/main.go

# Build menu bar UI
CGO_ENABLED=1 GOOS=darwin go build -o bin/memofy-ui cmd/memofy-ui/main.go

# Or use Makefile
make build
```

### Makefile (create in project root)

```makefile
.PHONY: build clean test run-core run-ui install

BINARY_CORE=bin/memofy-core
BINARY_UI=bin/memofy-ui
GO=CGO_ENABLED=1 GOOS=darwin go

build:
	mkdir -p bin
	$(GO) build -o $(BINARY_CORE) cmd/memofy-core/main.go
	$(GO) build -o $(BINARY_UI) cmd/memofy-ui/main.go

clean:
	rm -rf bin/
	rm -rf ~/.cache/memofy/

test:
	go test -v ./internal/...
	go test -v ./tests/integration/...

run-core:
	$(BINARY_CORE)

run-ui:
	$(BINARY_UI)

install:
	./scripts/install-launchagent.sh
```

---

## Development Workflow

### Running Locally

#### Terminal 1: Start OBS (if not already running)
```bash
open -a OBS
# Ensure WebSocket is enabled (see OBS Setup above)
```

#### Terminal 2: Run Daemon
```bash
# Run daemon in foreground with logging
make run-core

# Or with verbose logging
DEBUG=1 make run-core
```

Expected output:
```
2026-02-12 14:30:00 [INFO] Memofy daemon starting...
2026-02-12 14:30:00 [INFO] Connecting to OBS WebSocket at ws://localhost:4455
2026-02-12 14:30:01 [INFO] Connected to OBS v30.0.0
2026-02-12 14:30:01 [INFO] Detection polling started (interval: 2s)
2026-02-12 14:30:01 [INFO] Monitoring mode: auto
```

#### Terminal 3: Run Menu Bar UI
```bash
make run-ui
```

Expected: Menu bar icon appears in top-right of macOS menu bar

### Testing Detection

1. **Start Zoom/Teams**
   ```bash
   open -a "zoom.us"
   # Join a meeting or start a test meeting
   ```

2. **Watch Daemon Logs**
   ```
   2026-02-12 14:31:00 [DEBUG] Detection: Zoom process running=true, window match=true
   2026-02-12 14:31:02 [DEBUG] Detection: Start streak 1/3
   2026-02-12 14:31:04 [DEBUG] Detection: Start streak 2/3
   2026-02-12 14:31:06 [INFO] Detection: Start streak 3/3 - THRESHOLD MET
   2026-02-12 14:31:06 [INFO] OBS: Sending StartRecord command
   2026-02-12 14:31:07 [INFO] Recording started: ~/Movies/Memofy/2026-02-12_1431_Zoom_Meeting.mp4
   ```

3. **Check Menu Bar UI**
   - Icon should change from IDLE (gray) → WAIT (yellow) → REC (red)
   - Click menu bar icon to see status details

### Manual Testing Commands

Use file-based commands to test daemon:

```bash
# Start recording manually
echo "start" > ~/.cache/memofy/cmd.txt

# Stop recording
echo "stop" > ~/.cache/memofy/cmd.txt

# Toggle mode
echo "auto" > ~/.cache/memofy/cmd.txt
echo "pause" > ~/.cache/memofy/cmd.txt

# Check status
cat ~/.cache/memofy/status.json | jq .
```

---

## Testing

### Unit Tests

```bash
# Test state machine
go test -v ./internal/statemachine/

# Test all packages
go test -v ./...

# With coverage
go test -cover ./...
```

### Integration Tests

```bash
# Requires OBS running
go test -v ./tests/integration/

# Specific test
go test -v ./tests/integration/ -run TestRecordingFlow
```

### Table-Driven State Machine Tests

Example test structure (internal/statemachine/machine_test.go):

```go
func TestDebounceLogic(t *testing.T) {
    tests := []struct {
        name           string
        detections     []bool
        expectedState  string
        expectedAction string
    }{
        {
            name:           "Three consecutive detections trigger start",
            detections:     []bool{true, true, true},
            expectedState:  "WAIT",
            expectedAction: "prepare_start",
        },
        {
            name:           "Two detections then false resets counter",
            detections:     []bool{true, true, false, true, true},
            expectedState:  "IDLE",
            expectedAction: "none",
        },
        // ... more test cases
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

---

## Installation (Production)

### Install as LaunchAgent

```bash
# Build release binaries
make build

# Install LaunchAgent
sudo make install
```

The install script:
1. Copies binaries to `/usr/local/bin/`
2. Creates plist at `~/Library/LaunchAgents/com.memofy.core.plist`
3. Loads the agent with `launchctl load`
4. Creates config directory at `~/.config/memofy/`

### Verify Installation

```bash
# Check if daemon is running
launchctl list | grep memofy

# Expected output:
# 12345  0  com.memofy.core

# Check logs
tail -f /tmp/memofy-core.out.log
tail -f /tmp/memofy-core.err.log
```

### Uninstall

```bash
# Stop and unload
launchctl unload ~/Library/LaunchAgents/com.memofy.core.plist

# Remove files
rm ~/Library/LaunchAgents/com.memofy.core.plist
rm /usr/local/bin/memofy-core
rm /usr/local/bin/memofy-ui

# Clean config
rm -rf ~/.config/memofy/
rm -rf ~/.cache/memofy/
```

---

## Configuration Files

### Default Detection Rules

Location: `~/.config/memofy/detection-rules.json`

```json
{
  "rules": [
    {
      "application": "zoom",
      "process_names": ["zoom.us", "CptHost"],
      "window_hints": ["Zoom Meeting", "Zoom Webinar"],
      "enabled": true
    },
    {
      "application": "teams",
      "process_names": ["Microsoft Teams"],
      "window_hints": ["Meeting", "Call", "Reunión", "会議"],
      "enabled": true
    }
  ],
  "poll_interval_seconds": 2,
  "start_threshold": 3,
  "stop_threshold": 6
}
```

**Customization**: Edit this file to add language-specific window title hints or adjust thresholds.

---

## Troubleshooting

### OBS Connection Issues

**Problem**: `ERROR: Failed to connect to OBS`

**Solutions**:
1. Verify OBS is running: `ps aux | grep obs`
2. Check WebSocket enabled: OBS → Preferences → WebSocket Server
3. Verify port: Default is 4455
4. Check firewall: Allow local connections on port 4455

### Detection Not Working

**Problem**: Meetings not detected automatically

**Solutions**:
1. Check permissions: System Preferences → Security & Privacy → Accessibility
2. View detection logs: `tail -f /tmp/memofy-core.out.log`
3. Verify process names: `ps aux | grep -i zoom` or `ps aux | grep -i teams`
4. Update window hints in `detection-rules.json` for your language

### Black Screen Recordings

**Problem**: Recordings show black screen

**Solutions**:
1. Grant Screen Recording permission to OBS: System Preferences → Security & Privacy → Screen Recording
2. Add Display Capture source in OBS
3. Restart OBS after granting permissions

### Menu Bar Icon Not Showing

**Problem**: UI doesn't appear in menu bar

**Solutions**:
1. Check if process is running: `ps aux | grep memofy-ui`
2. Run from terminal to see errors: `./bin/memofy-ui`
3. Verify CGO_ENABLED=1 during build
4. Check macOS version (requires 11.0+)

---

## Development Tips

### Debugging State Machine

Add debug logging in `internal/statemachine/machine.go`:

```go
func (m *Machine) Update(detected bool) Action {
    log.Printf("[DEBUG] Update: detected=%v, startStreak=%d, stopStreak=%d", 
        detected, m.startStreak, m.stopStreak)
    // ... rest of logic
}
```

### Hot Reload During Development

```bash
# Use air for hot reloading
go install github.com/cosmtrek/air@latest

# Create .air.toml in project root
air

# Daemon will rebuild and restart on file changes
```

### Logging Best Practices

- INFO: State transitions, OBS operations, mode changes
- WARN: Recoverable errors (OBS disconnect, permission denied)
- ERROR: Unrecoverable errors (invalid config, fatal crashes)
- DEBUG: Detection details, counter values, raw events

---

## Next Steps

1. **Implement Core Daemon** (`cmd/memofy-core/main.go`)
   - OBS WebSocket client
   - Meeting detector
   - State machine
   - File-based IPC

2. **Implement Menu Bar UI** (`cmd/memofy-ui/main.go`)
   - Status bar icon with states
   - Menu with controls
   - Settings panel
   - Notification handling

3. **Write Tests**
   - State machine unit tests
   - Detection logic tests
   - Integration tests with mock OBS

4. **Documentation**
   - User guide
   - API documentation (godoc)
   - Architecture diagrams

---

## Resources

- [OBS WebSocket Protocol](https://github.com/obsproject/obs-websocket/blob/master/docs/generated/protocol.md)
- [macdriver Documentation](https://github.com/progrium/macdriver)
- [Go Testing Best Practices](https://go.dev/doc/tutorial/add-a-test)
- [macOS Permissions Guide](https://developer.apple.com/documentation/bundleresources/information_property_list)
