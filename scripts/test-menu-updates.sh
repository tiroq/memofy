#!/bin/bash
# Test script for menu-click update mechanism

set -e

echo "====================================="
echo "Menu-Click Update Mechanism Test"
echo "====================================="
echo ""

# Kill any existing instance
echo "1. Stopping any existing memofy-ui instances..."
killall memofy-ui 2>/dev/null || true
sleep 1

# Build
echo "2. Building application..."
cd "$(dirname "$0")/.."
go build -o bin/memofy-ui ./cmd/memofy-ui
echo "   ✓ Build successful"
echo ""

# Start app
echo "3. Starting memofy-ui in background..."
rm -f /tmp/memofy-ui-test.log
./bin/memofy-ui > /tmp/memofy-ui-test.log 2>&1 &
APP_PID=$!
echo "   ✓ Started with PID: $APP_PID"
sleep 3

# Check it's running
if ! ps -p $APP_PID > /dev/null; then
    echo "   ✗ Application crashed on startup!"
    echo "   Log output:"
    cat /tmp/memofy-ui-test.log
    exit 1
fi
echo "   ✓ Application running"
echo ""

# Trigger status changes
echo "4. Triggering status changes..."
STATUS_FILE="$HOME/.cache/memofy/status.json"

if [ ! -f "$STATUS_FILE" ]; then
    echo "   ✗ Status file not found: $STATUS_FILE"
    kill $APP_PID 2>/dev/null || true
    exit 1
fi

# Save original mode
ORIGINAL_MODE=$(cat "$STATUS_FILE" | grep -o '"mode": *"[^"]*"' | cut -d'"' -f4)
echo "   Current mode: $ORIGINAL_MODE"

# Change to paused
echo "   Changing mode to 'paused'..."
cat "$STATUS_FILE" | sed 's/"mode": "'$ORIGINAL_MODE'"/"mode": "paused"/' > /tmp/status_temp.json
mv /tmp/status_temp.json "$STATUS_FILE"
sleep 0.5

# Check log for queued update
if ! grep -q "UI update queued" /tmp/memofy-ui-test.log 2>/dev/null; then
    # Uncomment debug log if not found
    echo "   Note: Debug logging is disabled (expected)"
fi
echo "   ✓ Status file updated"
echo ""

# Instructions
echo "5. MANUAL TEST REQUIRED"
echo "   ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "   BEFORE clicking menu bar icon:"
echo "   - Menu bar should show old icon (⏸ or ▶)"
echo ""
echo "   ACTION: Click the Memofy menu bar icon NOW"
echo ""
echo "   AFTER clicking menu bar icon:"
echo "   - Menu bar should show updated icon"
echo "   - Log should show: '✓ Menu click detected'"
echo ""
echo "   Press ENTER when you've clicked the menu bar icon..."
read

# Check for the debug message
sleep 1
if grep -q "Menu click detected" /tmp/memofy-ui-test.log; then
    echo "   ✓ SUCCESS: Update was applied on menu click!"
    echo ""
    echo "   Log output:"
    grep "Menu click detected" /tmp/memofy-ui-test.log | tail -1
else
    echo "   ⚠ Did not find 'Menu click detected' in log"
    echo "   This could mean:"
    echo "   - You didn't click the menu bar icon yet"
    echo "   - Debug logging is disabled"
    echo "   - There's an issue with the implementation"
fi
echo ""

# Restore original mode
echo "6. Restoring original mode..."
cat "$STATUS_FILE" | sed 's/"mode": "paused"/"mode": "'$ORIGINAL_MODE'"/' > /tmp/status_temp.json
mv /tmp/status_temp.json "$STATUS_FILE"
echo "   ✓ Mode restored to: $ORIGINAL_MODE"
echo ""

# Check if still running
if ps -p $APP_PID > /dev/null; then
    echo "7. Application status: ✓ Still running (no crashes)"
else
    echo "7. Application status: ✗ Crashed during test"
    echo "   Last log output:"
    tail -20 /tmp/memofy-ui-test.log
    exit 1
fi
echo ""

# Cleanup prompt
echo "====================================="
echo "Test Complete!"
echo "====================================="
echo ""
echo "To stop the application: killall memofy-ui"
echo "To view full logs: cat /tmp/memofy-ui-test.log"
echo ""
