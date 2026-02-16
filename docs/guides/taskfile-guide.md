# Taskfile Guide

## Install Task

```bash
brew install go-task     # macOS
# or: sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d -b ~/.local/bin
```

## Common Tasks

```bash
task                    # List all tasks
task build              # Build both binaries
task test               # Run tests
task lint               # Run linter
task install            # Install to ~/.local/bin
```

## Development

```bash
task build-core         # Build daemon only
task build-ui           # Build UI only
task dev-daemon         # Run daemon in foreground
task dev-ui             # Run UI in foreground
task fmt                # Format code
task test-coverage      # Coverage report
```

## Daemon Control

```bash
task status             # Check status
task start|stop|restart # Control daemon
task logs               # View logs
task logs-ui            # UI logs
task logs-error         # Error logs
```

## Release

```bash
task release-patch      # 0.2.0 → 0.2.1
task release-minor      # 0.2.0 → 0.3.0
task release-major      # 0.2.0 → 1.0.0
task release-alpha      # Auto alpha
task release-beta       # Auto beta
task release-rc         # Auto RC
task release-stable     # Promote to stable
task release-list       # List tags
task release-status     # Check CI status
```

## Utilities

```bash
task clean              # Remove build artifacts
task deps               # Download deps
task deps-update        # Update deps
task ci-test            # Simulate CI
```

See [RELEASE_PIPELINE.md](RELEASE_PIPELINE.md) for detailed release workflow.
