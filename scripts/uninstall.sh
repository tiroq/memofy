#!/bin/bash
set -e

# Memofy Uninstallation Script
# Removes memofy-core daemon, memofy-ui, and LaunchAgent

INSTALL_DIR="$HOME/.local/bin"
LAUNCHAGENTS_DIR="$HOME/Library/LaunchAgents"
PLIST_NAME="com.memofy.core.plist"

echo "=== Memofy Uninstallation ==="
echo ""

# Unload LaunchAgent
if [ -f "$LAUNCHAGENTS_DIR/$PLIST_NAME" ]; then
    echo "Stopping memofy-core daemon..."
    launchctl unload "$LAUNCHAGENTS_DIR/$PLIST_NAME" 2>/dev/null || true
    
    echo "Removing LaunchAgent plist..."
    rm "$LAUNCHAGENTS_DIR/$PLIST_NAME"
fi

# Remove binaries
if [ -f "$INSTALL_DIR/memofy-core" ]; then
    echo "Removing memofy-core binary..."
    rm "$INSTALL_DIR/memofy-core"
fi

if [ -f "$INSTALL_DIR/memofy-ui" ]; then
    echo "Removing memofy-ui binary..."
    rm "$INSTALL_DIR/memofy-ui"
fi

# Optional: Remove config and cache (ask user)
echo ""
read -p "Remove configuration and cache files? (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "Removing config directory..."
    rm -rf "$HOME/.config/memofy"
    
    echo "Removing cache directory..."
    rm -rf "$HOME/.cache/memofy"
    
    echo "Removing log files..."
    rm -f /tmp/memofy-core.out.log
    rm -f /tmp/memofy-core.err.log
fi

echo ""
echo "✓ Uninstallation complete!"
echo ""
echo "Note: OBS recordings in ~/Movies are preserved."
