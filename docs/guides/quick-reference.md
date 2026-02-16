# Quick Reference

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/tiroq/memofy/main/scripts/quick-install.sh | bash
```

## Files

```
~/.local/bin/memofy-{core,ui}
~/.config/memofy/detection-rules.json
~/.cache/memofy/{status.json,cmd.txt}
~/Library/LaunchAgents/com.memofy.core.plist
/tmp/memofy-core.{out,err}.log
```

## Build

```bash
task build|test|lint|install
```

## Release

```bash
task release-{major|minor|patch}     # Auto-bump
task release-{alpha|beta|rc}         # Pre-releases
task release-stable                  # Promote to stable
```

## Control

```bash
echo 'start|stop|auto' > ~/.cache/memofy/cmd.txt
cat ~/.cache/memofy/status.json | jq
tail -f /tmp/memofy-core.out.log
```

## OBS

```bash
brew install --cask obs
# Tools → WebSocket Server → Enable (port 4455)
```

## Troubleshoot

```bash
pgrep -fl memofy|OBS
nc -zv localhost 4455
launchctl unload|load ~/Library/LaunchAgents/com.memofy.core.plist
```
