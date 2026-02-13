#!/bin/bash
set -e

# Memofy Installation Script
# Installs memofy-core daemon and memofy-ui menu bar app

INSTALL_DIR="$HOME/.local/bin"
LAUNCHAGENTS_DIR="$HOME/Library/LaunchAgents"
PLIST_NAME="com.memofy.core.plist"
CONFIG_DIR="$HOME/.config/memofy"
CACHE_DIR="$HOME/.cache/memofy"

echo "=== Memofy Installation ==="
echo ""

# Check if binaries exist
if [ ! -f "bin/memofy-core" ]; then
    echo "Error: bin/memofy-core not found. Run 'make build' first."
    exit 1
fi

if [ ! -f "bin/memofy-ui" ]; then
    echo "Error: bin/memofy-ui not found. Run 'make build' first."
    exit 1
fi

# Create directories
echo "Creating directories..."
mkdir -p "$INSTALL_DIR"
mkdir -p "$LAUNCHAGENTS_DIR"
mkdir -p "$CONFIG_DIR"
mkdir -p "$CACHE_DIR"

# Copy binaries
echo "Installing binaries to $INSTALL_DIR..."
cp bin/memofy-core "$INSTALL_DIR/"
cp bin/memofy-ui "$INSTALL_DIR/"
chmod +x "$INSTALL_DIR/memofy-core"
chmod +x "$INSTALL_DIR/memofy-ui"

# Copy default config if not exists
if [ ! -f "$CONFIG_DIR/detection-rules.json" ]; then
    echo "Installing default detection rules..."
    cp configs/default-detection-rules.json "$CONFIG_DIR/detection-rules.json"
fi

# Install LaunchAgent
echo "Installing LaunchAgent..."
sed "s|INSTALL_DIR|$INSTALL_DIR|g" scripts/com.memofy.core.plist > "$LAUNCHAGENTS_DIR/$PLIST_NAME"

# Unload existing agent if running
if launchctl list | grep -q com.memofy.core; then
    echo "Stopping existing memofy-core daemon..."
    launchctl unload "$LAUNCHAGENTS_DIR/$PLIST_NAME" 2>/dev/null || true
fi

# Load LaunchAgent
echo "Starting memofy-core daemon..."
launchctl load "$LAUNCHAGENTS_DIR/$PLIST_NAME"

# Check if daemon started
sleep 2
if launchctl list | grep -q com.memofy.core; then
    echo ""
    echo "✓ Installation successful!"
    echo ""
    echo "Daemon: $INSTALL_DIR/memofy-core (running via LaunchAgent)"
    echo "Menu Bar UI: $INSTALL_DIR/memofy-ui"
    echo "Config: $CONFIG_DIR/detection-rules.json"
    echo "Logs: /tmp/memofy-core.{out,err}.log"
    echo ""
    echo "To start the menu bar app, run: $INSTALL_DIR/memofy-ui"
    echo "To check daemon status: launchctl list | grep memofy"
    echo ""
    echo "⚠️  Don't forget to:"
    echo "  1. Grant Screen Recording permission in System Preferences > Security & Privacy"
    echo "  2. Grant Accessibility permission in System Preferences > Security & Privacy"
    echo "  3. Ensure OBS is running with WebSocket server enabled (ws://localhost:4455)"
else
    echo ""
    echo "⚠️  Installation completed but daemon failed to start."
    echo "Check logs: tail -f /tmp/memofy-core.err.log"
    exit 1
fi
