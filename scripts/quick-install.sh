#!/bin/bash
set -e

# Memofy Quick Install Script
# One-command installation from GitHub releases or local build

REPO_URL="https://github.com/tiroq/memofy"
RELEASES_URL="$REPO_URL/releases/download"
LATEST_RELEASE_URL="$REPO_URL/releases/latest"
INSTALL_DIR="$HOME/.local/bin"
CONFIG_DIR="$HOME/.config/memofy"
CACHE_DIR="$HOME/.cache/memofy"
LAUNCHAGENTS_DIR="$HOME/Library/LaunchAgents"

MEMOFY_VERSION="0.1.0"
OS=$(uname -s)

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
print_info() {
    echo -e "${BLUE}ℹ${NC}  $1"
}

print_success() {
    echo -e "${GREEN}✓${NC}  $1"
}

print_warn() {
    echo -e "${YELLOW}⚠${NC}  $1"
}

print_error() {
    echo -e "${RED}✗${NC}  $1"
}

# Check prerequisites
check_prerequisites() {
    print_info "Checking prerequisites..."
    
    local missing_tools=()
    
    if ! command -v brew &> /dev/null; then
        missing_tools+=("Homebrew")
    fi
    
    if ! command -v obs &> /dev/null && [ ! -d "/Applications/OBS.app" ]; then
        missing_tools+=("OBS Studio")
    fi
    
    if [ ${#missing_tools[@]} -gt 0 ]; then
        print_warn "Missing tools: ${missing_tools[*]}"
        print_info "Installing missing tools..."
        
        if [[ " ${missing_tools[*]} " =~ " Homebrew " ]]; then
            print_info "Installing Homebrew..."
            /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
        fi
        
        if [[ " ${missing_tools[*]} " =~ " OBS Studio " ]]; then
            print_info "Installing OBS Studio..."
            brew install --cask obs
        fi
    fi
    
    print_success "All prerequisites installed"
}

# Get latest release version
get_latest_version() {
    curl -s "$LATEST_RELEASE_URL" | grep '"tag_name":' | cut -d'"' -f4 | sed 's/^v//'
}

# Download binary from release
download_release_binary() {
    local version=$1
    local binary=$2
    local temp_file="/tmp/$binary"
    
    print_info "Downloading $binary v$version..."
    
    local download_url="$RELEASES_URL/v$version/$binary-darwin-arm64.zip"
    
    if curl -sfo "$temp_file.zip" "$download_url"; then
        unzip -o "$temp_file.zip" -d /tmp
        rm "$temp_file.zip"
        print_success "Downloaded $binary"
        echo "$temp_file"
    else
        return 1
    fi
}

# Build from source
build_from_source() {
    print_info "Building from source..."
    
    if ! command -v go &> /dev/null; then
        print_warn "Go not installed. Installing..."
        brew install go
    fi
    
    # Ensure we're in the repo directory
    if [ ! -f "Makefile" ]; then
        print_error "Not in memofy repository directory"
        exit 1
    fi
    
    print_info "Building binaries..."
    make clean
    make build
    print_success "Build complete"
}

# Install binaries
install_binaries() {
    print_info "Installing binaries..."
    
    mkdir -p "$INSTALL_DIR"
    mkdir -p "$CONFIG_DIR"
    mkdir -p "$CACHE_DIR"
    mkdir -p "$LAUNCHAGENTS_DIR"
    
    local core_binary="${1:-bin/memofy-core}"
    local ui_binary="${2:-bin/memofy-ui}"
    
    if [ ! -f "$core_binary" ]; then
        print_error "$core_binary not found"
        exit 1
    fi
    
    if [ ! -f "$ui_binary" ]; then
        print_error "$ui_binary not found"
        exit 1
    fi
    
    cp "$core_binary" "$INSTALL_DIR/memofy-core"
    cp "$ui_binary" "$INSTALL_DIR/memofy-ui"
    chmod +x "$INSTALL_DIR/memofy-core"
    chmod +x "$INSTALL_DIR/memofy-ui"
    
    print_success "Binaries installed to $INSTALL_DIR"
}

# Install config and LaunchAgent
install_config() {
    print_info "Installing configuration..."
    
    # Copy default config if not exists
    if [ ! -f "$CONFIG_DIR/detection-rules.json" ]; then
        if [ -f "configs/default-detection-rules.json" ]; then
            cp configs/default-detection-rules.json "$CONFIG_DIR/detection-rules.json"
        fi
    fi
    
    # Install LaunchAgent
    if [ -f "scripts/com.memofy.core.plist" ]; then
        sed "s|INSTALL_DIR|$INSTALL_DIR|g" scripts/com.memofy.core.plist > "$LAUNCHAGENTS_DIR/com.memofy.core.plist"
        launchctl load "$LAUNCHAGENTS_DIR/com.memofy.core.plist" 2>/dev/null || true
        print_success "LaunchAgent installed and loaded"
    fi
    
    print_success "Configuration installed"
}

# Request macOS permissions
setup_permissions() {
    print_info "Setting up macOS permissions..."
    
    # These need user interaction, so just inform
    echo ""
    print_warn "Please grant permissions when prompted by macOS:"
    echo "  1. Screen Recording: System Preferences > Security & Privacy > Screen Recording"
    echo "  2. Accessibility: System Preferences > Security & Privacy > Accessibility"
    echo ""
    
    # Try to request in background (may not work, but worth trying)
    osascript -e 'tell application "System Preferences"
        activate
        set current pane to pane id "com.apple.preference.security"
    end tell' 2>/dev/null || true
    
    print_success "Permission setup instructions displayed"
}

# Enable WebSocket in OBS
setup_obs() {
    print_info "Checking OBS setup..."
    
    # Check if OBS is already running
    if pgrep -x "OBS" > /dev/null; then
        print_warn "OBS is running. Please configure manually:"
        echo "  1. Open OBS"
        echo "  2. Go to: Tools > obs-websocket Settings"
        echo "  3. Enable 'Enable WebSocket server'"
        echo "  4. Set port to 4455"
    else
        print_info "Starting OBS..."
        open -a OBS &
        sleep 2
        
        print_warn "Please configure OBS when it opens:"
        echo "  1. Go to: Tools > obs-websocket Settings"
        echo "  2. Enable 'Enable WebSocket server'"
        echo "  3. Set port to 4455"
        echo "  4. Close OBS when done (Memofy will auto-start it)"
    fi
}

# Start menu bar UI
start_ui() {
    print_info "Starting Memofy menu bar UI..."
    
    # Kill any existing instances
    killall memofy-ui 2>/dev/null || true
    
    # Start daemon if not running
    launchctl start com.memofy.core 2>/dev/null || true
    
    # Start UI in background
    "$INSTALL_DIR/memofy-ui" &
    
    sleep 1
    print_success "Memofy is running in menu bar"
}

# Parse arguments
INSTALL_FROM_SOURCE=false
INSTALL_FROM_RELEASE=false
VERSION=""

while [[ $# -gt 0 ]]; do
    case $1 in
        --source) INSTALL_FROM_SOURCE=true; shift ;;
        --release) INSTALL_FROM_RELEASE=true; VERSION="$2"; shift 2 ;;
        --version) echo "$MEMOFY_VERSION"; exit 0 ;;
        --help)
            echo "Usage: ./quick-install.sh [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --source              Build from source (default if releases not available)"
            echo "  --release <version>   Install specific release version"
            echo "  --version             Show version"
            echo "  --help                Show this help"
            echo ""
            echo "Examples:"
            echo "  ./quick-install.sh                  # Smart install (release or source)"
            echo "  ./quick-install.sh --source         # Build from source"
            echo "  ./quick-install.sh --release 0.1.0  # Install release v0.1.0"
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Main installation flow
main() {
    echo ""
    echo "╔════════════════════════════════════════╗"
    echo "║       Memofy Quick Install - v0.1       ║"
    echo "║  Automatic Meeting Recorder for macOS   ║"
    echo "╚════════════════════════════════════════╝"
    echo ""
    
    # Check prerequisites
    check_prerequisites
    echo ""
    
    # Get binaries (source or release)
    if [ "$INSTALL_FROM_SOURCE" = true ]; then
        build_from_source
    elif [ "$INSTALL_FROM_RELEASE" = true ] && [ -n "$VERSION" ]; then
        print_error "Release download not yet implemented"
        print_info "Falling back to source build..."
        build_from_source
    else
        # Smart detection: try release, fall back to source
        if command -v curl &> /dev/null; then
            print_info "Attempting to download pre-built release..."
            latest_version=$(get_latest_version 2>/dev/null || echo "")
            if [ -n "$latest_version" ]; then
                print_success "Found release v$latest_version"
                download_release_binary "$latest_version" "memofy-core" || build_from_source
                download_release_binary "$latest_version" "memofy-ui" || build_from_source
            else
                build_from_source
            fi
        else
            build_from_source
        fi
    fi
    echo ""
    
    # Install binaries
    install_binaries
    install_config
    echo ""
    
    # Setup permissions
    setup_permissions
    echo ""
    
    # Setup OBS
    setup_obs
    echo ""
    
    # Start UI
    start_ui
    echo ""
    
    print_success "Installation complete!"
    echo ""
    echo "Next steps:"
    echo "  1. Configure OBS: Tools > obs-websocket Settings > Enable WebSocket server"
    echo "  2. Grant macOS permissions when prompted"
    echo "  3. Start a Zoom/Teams/Google Meet meeting"
    echo "  4. Memofy will detect and auto-record!"
    echo ""
    echo "View logs: tail -f /tmp/memofy-core.out.log"
    echo "Settings: Click menu bar icon > Settings"
    echo ""
}

main
