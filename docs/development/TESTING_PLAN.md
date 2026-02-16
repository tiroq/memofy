# Testing Plan

## Unit Tests

```bash
task test                           # All tests
task test-coverage                   # With coverage
go test -v ./internal/detector/...  # Specific package
```

## Integration Tests

```bash
task test-integration
```

Tests:
- Meeting detection logic
- State machine transitions
- OBS WebSocket communication
- IPC file operations

## Manual Testing

1. Start daemon: `task dev-daemon`
2. Open meeting app (Zoom/Teams/Meet)
3. Verify auto-start recording
4. Close meeting → verify auto-stop
5. Check logs for errors

## CI

GitHub Actions runs:
- Unit tests
- Linter
- Build for all platforms

See `.github/workflows/` for configuration.
