# PID File Issues

## Stale PID File

**Symptom**: Daemon won't start, claims another instance running

**Cause**: PID file exists but process is dead

**Fix**:
```bash
rm ~/.cache/memofy/memofy-core.pid
task restart
```

## Permission Denied

**Symptom**: Can't write PID file

**Fix**:
```bash
mkdir -p ~/.cache/memofy
chmod 755 ~/.cache/memofy
```

## Multiple Instances

**Symptom**: Multiple daemons running

**Check**:
```bash
pgrep -fl memofy-core
```

**Fix**:
```bash
killall memofy-core
rm ~/.cache/memofy/memofy-core.pid
task start
```

PID file location: `~/.cache/memofy/memofy-core.pid`
