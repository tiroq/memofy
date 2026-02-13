# Development Guide

## Building from Source

### Prerequisites

- Go 1.21+
- macOS 11.0+
- OBS Studio 28.0+

### Clone & Build

```bash
git clone https://github.com/tiroq/memofy.git
cd memofy

# Build
make build

# Install locally
make install

# Run
~/.local/bin/memofy-ui
```

---

## Using Task (Recommended)

[Task](https://taskfile.dev/) provides convenient commands:

```bash
# Install Task
brew install go-task

# Build
task build

# Run tests
task test

# Run with race detector
task test-race

# Lint
task lint

# Install locally
task install

# Clean build artifacts
task clean
```

See [Taskfile Guide](taskfile-guide.md) for all commands.

---

## Project Structure

```
memofy/
├── cmd/
│   ├── memofy-core/      # Daemon entrypoint
│   └── memofy-ui/        # Menu bar UI entrypoint
├── internal/
│   ├── config/           # Configuration management
│   ├── detector/         # Meeting detection logic
│   ├── ipc/              # File-based IPC
│   ├── obsws/            # OBS WebSocket client
│   ├── recorder/         # Recording state machine
│   └── ui/               # Menu bar UI (macOS)
├── pkg/
│   └── models/           # Shared data models
├── configs/              # Default configuration
└── scripts/              # Build/install scripts
```

---

## Testing

### Run All Tests

```bash
# Quick tests
make test

# With race detector
make test-race

# Coverage
make test-coverage
```

### Run Specific Tests

```bash
# Test specific package
go test ./internal/detector/...

# Verbose output
go test -v ./internal/obsws/...

# Run specific test
go test -run TestStateTransition ./internal/recorder/...
```

---

## Linting

```bash
# Run golangci-lint
make lint

# Fix issues automatically
golangci-lint run --fix
```

---

## Architecture

### Core Components

**1. Daemon (`memofy-core`)**
- Detects active meetings via process/window monitoring
- Controls OBS recording via WebSocket
- State machine with 3/6 debounce thresholds
- File-based status reporting

**2. Menu Bar UI (`memofy-ui`)**
- Native macOS menu bar app
- Reads status from daemon
- Sends commands via IPC
- Displays notifications

**3. OBS WebSocket Client**
- Connects to OBS Studio
- Auto-starts OBS if not running
- Creates missing sources automatically
- Handles reconnection

### Detection Logic

Meeting detection uses **3/6 debounce**:
- 3 consecutive detections → Start recording
- 6 consecutive non-detections → Stop recording

Prevents false starts from quick window switches.

### State Machine

```
Idle → Detected (1-2 frames) → Recording
Recording → Lost (1-5 frames) → Idle
```

---

## Contributing

### Code Style

- Follow [Effective Go](https://go.dev/doc/effective_go)
- Run `gofmt` before committing
- Add tests for new features
- Update documentation

### Adding Meeting Platform Support

1. Add detection config to `configs/default-detection-rules.json`
2. Add process/window matching logic
3. Update tests
4. Update README features list

### Pull Requests

1. Fork the repository
2. Create feature branch: `git checkout -b feature/my-feature`
3. Make changes with tests
4. Run: `make test lint`
5. Commit: `git commit -m "feat: add my feature"`
6. Push and create PR

---

## Debugging

### Enable Debug Logging

```bash
# Set log level in daemon
export MEMOFY_LOG_LEVEL=debug
~/.local/bin/memofy-core
```

### View Logs

```bash
# Daemon logs
tail -f /tmp/memofy-core.out.log
tail -f /tmp/memofy-core.err.log

# UI logs
tail -f /tmp/memofy-ui.out.log
```

### Common Issues

**OBS WebSocket connection fails**:
- Check OBS is running: `pgrep OBS`
- Verify WebSocket enabled and port 4455
- Test connection: `nc -zv localhost 4455`

**Race condition detected**:
- Run tests with `-race` flag
- Fix concurrent access issues
- Ensure proper mutex usage

---

**Next**: See [Release Process](release-process.md) for maintainers
