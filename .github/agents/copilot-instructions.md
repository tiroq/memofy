# memofy Development Guidelines

Auto-generated from all feature plans. Last updated: 2026-02-12

## Active Technologies
- Go 1.21 + `gorilla/websocket` v1.5.3 (existing), `log/slog` (stdlib, Go 1.21), `crypto/rand` (stdlib) (002-obs-autostop)
- Rolling NDJSON file (`/tmp/memofy-debug.log`, 10 MB cap) (002-obs-autostop)

- Go 1.21+ (001-auto-meeting-recorder)

## Project Structure

```text
cmd/
internal/
pkg/
tests/
```

## Commands

# Add commands for Go 1.21+

## Code Style

Go 1.21+: Follow standard conventions

## Recent Changes
- 002-obs-autostop: Added Go 1.21 + `gorilla/websocket` v1.5.3 (existing), `log/slog` (stdlib, Go 1.21), `crypto/rand` (stdlib)

- 001-auto-meeting-recorder: Added Go 1.21+

<!-- MANUAL ADDITIONS START -->
<!-- MANUAL ADDITIONS END -->
