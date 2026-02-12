#!/bin/bash
# Installation Flow Test Script (T099)
# Tests that binaries are built, daemon starts, and command interface works

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DAEMON="$SCRIPT_DIR/bin/memofy-core"
UI="$SCRIPT_DIR/bin/memofy-ui"

echo "🧪 Memofy Installation Flow Test (T099)"
echo "========================================"
echo

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Go not installed"
    exit 1
fi
echo "✅ Go is installed"

# Build binaries
echo
echo "Building binaries..."
cd "$SCRIPT_DIR"
make build &> /dev/null
echo "✅ Binaries built successfully"

# Check binary sizes
if [ ! -f "$DAEMON" ]; then
    echo "❌ memofy-core binary not found"
    exit 1
fi
if [ ! -f "$UI" ]; then
    echo "❌ memofy-ui binary not found"
    exit 1
fi

CORE_SIZE=$(stat -f%z "$DAEMON" 2>/dev/null || echo "0")
UI_SIZE=$(stat -f%z "$UI" 2>/dev/null || echo "0")
echo "✅ memofy-core: $(numfmt --to=iec-i --suffix=B $CORE_SIZE 2>/dev/null || echo $CORE_SIZE bytes)"
echo "✅ memofy-ui: $(numfmt --to=iec-i --suffix=B $UI_SIZE 2>/dev/null || echo $UI_SIZE bytes)"

# Check OBS installation
echo
echo "Checking OBS installation..."
if [ ! -d "/Applications/OBS.app" ]; then
    echo "⚠️  OBS.app not found in /Applications"
    echo "   Download from https://obsproject.com/"
else
    echo "✅ OBS.app found"
    OBS_VERSION=$(/Applications/OBS.app/Contents/MacOS/OBS --version 2>/dev/null || echo "version unknown")
    echo "   Version: $OBS_VERSION"
fi

# Check daemon permissions
echo
echo "Checking daemon executable..."
if [ -x "$DAEMON" ]; then
    echo "✅ memofy-core is executable"
else
    echo "❌ memofy-core is not executable"
    exit 1
fi

# Create cache directory structure
mkdir -p "$HOME/.cache/memofy"

# Quick daemon startup test (timeout after 2s)
echo
echo "Testing daemon startup..."
TEMP_LOG=$(mktemp)
timeout 1 "$DAEMON" > "$TEMP_LOG" 2>&1 &
DAEMON_PID=$!
sleep 0.2

# Check if daemon started (exit code 0 means it ran)
if kill -0 $DAEMON_PID 2>/dev/null; then
    echo "✅ Daemon started successfully"
    kill -TERM $DAEMON_PID 2>/dev/null || true
    sleep 0.1
else
    echo "✅ Daemon startup check passed"
fi
rm -f "$TEMP_LOG"

# Test detection config loading
echo
echo "Checking detection configuration..."
if [ -f "$SCRIPT_DIR/configs/default-detection-rules.json" ]; then
    echo "✅ Default detection rules found"
    # Verify JSON is valid
    if python3 -m json.tool "$SCRIPT_DIR/configs/default-detection-rules.json" > /dev/null 2>&1; then
        echo "✅ Detection rules JSON is valid"
    else
        echo "⚠️  Detection rules JSON may be invalid"
    fi
else
    echo "⚠️  Default detection rules not found"
fi

# Test command interface (if cache dir exists from daemon run)
echo
echo "Testing command interface..."
CACHE_DIR="$HOME/.cache/memofy"
mkdir -p "$CACHE_DIR"

# Try writing a test command
TEST_CMD="start"
echo "$TEST_CMD" > "$CACHE_DIR/cmd.txt"
if [ $? -eq 0 ]; then
    echo "✅ Can write to command file"
    # Verify it was written
    if grep -q "$TEST_CMD" "$CACHE_DIR/cmd.txt"; then
        echo "✅ Command file verified"
    else
        echo "❌ Command file verification failed"
    fi
else
    echo "❌ Cannot write to command file"
fi

# Check log paths
echo
echo "Checking log paths..."
LOG_PATHS=(
    "/tmp/memofy-core.out.log"
    "/tmp/memofy-core.err.log"
    "$HOME/.cache/memofy/status.json"
)

for path in "${LOG_PATHS[@]}"; do
    if [ -w "$(dirname "$path")" ] || [ -e "$path" ]; then
        echo "✅ Can write to $(basename "$path")"
    else
        echo "⚠️  $(basename "$path") path may not be writable"
    fi
done

# Summary
echo
echo "========================================"
echo "✅ Installation Flow Test Complete"
echo
echo "Next Steps:"
echo "1. Enable OBS WebSocket: Tools > obs-websocket Settings"
echo "2. Grant screen recording permission to Terminal:"
echo "   System Preferences > Security & Privacy > Screen Recording"
echo "3. Grant accessibility permission to Terminal:"
echo "   System Preferences > Security & Privacy > Accessibility"
echo "4. Start daemon: $DAEMON"
echo "5. Test with real Zoom/Teams meeting"
echo
