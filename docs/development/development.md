# Development Guide

## Prerequisites

- Go 1.21+
- macOS 11.0+
- OBS Studio 28.0+

## Build

```bash
git clone https://github.com/tiroq/memofy.git
cd memofy
task build              # or: make build
task install
```

## Structure

```
cmd/                    # Entrypoints
  ├── memofy-core/      # Daemon
  └── memofy-ui/        # Menu bar UI
internal/
  ├── config/           # Config management
  ├── detector/         # Meeting detection
  ├── ipc/              # File-based IPC
  ├── obsws/            # OBS WebSocket
  └── statemachine/     # Recording state
pkg/macui/              # macOS UI
```

## Testing

```bash
task test               # All tests
task test-coverage      # With coverage
go test -v ./internal/detector/...    # Specific package
```

## Architecture

**memofy-core**: Detects meetings → Controls OBS via WebSocket (3/6 debounce)  
**memofy-ui**: Menu bar → Status display → Command IPC  
**obsws**: WebSocket client → Auto-start OBS → Source management

## Development

```bash
task dev-daemon         # Run daemon
task dev-ui             # Run UI
task fmt                # Format
task lint               # Lint
```

See [Architecture docs](../architecture/) for detailed design.
