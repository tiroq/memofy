#!/bin/bash
# Installation flow test - quick checks only

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo "🧪 Memofy Installation Flow Test"
echo "=================================="
echo

# Check binaries
cd "$SCRIPT_DIR"
make build > /dev/null 2>&1 && echo "✅ Build successful" || echo "❌ Build failed"

[ -f bin/memofy-core ] && echo "✅ memofy-core exists" || echo "❌ memofy-core missing"
[ -f bin/memofy-ui ] && echo "✅ memofy-ui exists" || echo "❌ memofy-ui missing"
[ -x bin/memofy-core ] && echo "✅ memofy-core executable" || echo "❌ memofy-core not executable"

# Check OBS
[ -d "/Applications/OBS.app" ] && echo "✅ OBS.app found" || echo "⚠️  OBS.app not found"

# Check config
[ -f "configs/default-detection-rules.json" ] && echo "✅ Config file exists" || echo "⚠️  Config missing"

# Check cache paths
mkdir -p ~/.cache/memofy
echo "test" > ~/.cache/memofy/cmd.txt && echo "✅ Cache directory writable" || echo "❌ Cache not writable"

# Check log paths
touch /tmp/memofy-test.log 2>/dev/null && echo "✅ /tmp writable for logs" || echo "⚠️  /tmp may not be writable"
rm -f /tmp/memofy-test.log

echo
echo "❤️  Installation checks complete!"
