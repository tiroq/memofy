# Contract: CLI Interface

**Feature**: `002-obs-autostop`
**Binary**: `memofy-core`

---

## New Flags / Subcommands

### `--export-diag`

**Invocation**:
```sh
memofy-core --export-diag
```

**Behaviour**:
1. Reads the rolling log from `MEMOFY_LOG_PATH` if set, otherwise `/tmp/memofy-debug.log`.
2. If the log file does not exist, exits with a non-zero code and prints a message to stderr.
3. Writes a NDJSON export bundle to `./memofy-diag-<YYYYMMDDTHHmmss>.ndjson`.
4. Prints the path of the written file to stdout.
5. Exits with code `0` on success, non-zero on failure.

**Exit codes**:

| Code | Meaning |
|------|---------|
| 0 | Export written successfully |
| 1 | Log file not found |
| 2 | Log file unreadable |
| 3 | Output file could not be created |

**Example output (stdout)**:
```
Wrote: /home/user/memofy-diag-20260220T134222.ndjson (4312 lines)
```

**Example error (stderr)**:
```
error: log file not found at /tmp/memofy-debug.log
hint: run with MEMOFY_DEBUG_RECORDING=true to enable logging
```

---

## New Environment Variables

| Variable | Type | Default | Purpose |
|----------|------|---------|---------|
| `MEMOFY_DEBUG_RECORDING` | bool (`"true"` / `"false"`) | unset (disabled) | Enable verbose NDJSON logging |
| `MEMOFY_LOG_PATH` | file path | `/tmp/memofy-debug.log` | Override log file location |
| `MEMOFY_MANUAL_DEBOUNCE_MS` | integer (ms) | `5000` | Manual-mode race-condition guard window |

---

## Existing CLI (unchanged)

The existing daemon invocation is unchanged:
```sh
memofy-core              # runs as daemon (default)
memofy-core --export-diag   # export mode (new): exits after writing bundle
```

The `--export-diag` flag causes the process to exit immediately after writing the bundle; it does not start the daemon.
