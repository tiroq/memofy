#!/bin/bash
# Memofy process management helper script

set -e

CACHE_DIR="$HOME/.cache/memofy"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

print_info() {
    echo -e "${GREEN}ℹ${NC}  $1"
}

print_warn() {
    echo -e "${YELLOW}⚠${NC}  $1"
}

print_error() {
    echo -e "${RED}✗${NC}  $1"
}

# Show process logs
show_logs() {
    echo ""
    echo "=== Memofy Process Logs ==="
    echo ""
    
    echo "--- memofy-core (Output) ---"
    if [ -f /tmp/memofy-core.out.log ]; then
        tail -20 /tmp/memofy-core.out.log
    else
        echo "No output log found"
    fi
    
    echo ""
    echo "--- memofy-core (Errors) ---"
    if [ -f /tmp/memofy-core.err.log ]; then
        tail -20 /tmp/memofy-core.err.log
    else
        echo "No error log found"
    fi
    
    echo ""
    echo "--- memofy-ui (Output) ---"
    if [ -f /tmp/memofy-ui.out.log ]; then
        tail -20 /tmp/memofy-ui.out.log
    else
        echo "No output log found"
    fi
    
    echo ""
    echo "Log files:"
    echo "  /tmp/memofy-core.out.log"
    echo "  /tmp/memofy-core.err.log"
    echo "  /tmp/memofy-ui.out.log"
    echo ""
    echo "View all logs:"
    echo "  tail -f /tmp/memofy-core.out.log"
    echo "  tail -f /tmp/memofy-core.err.log"
    echo ""
}

# Stop a memofy process gracefully
stop_process() {
    local process_name=$1
    local pid_file="$CACHE_DIR/${process_name}.pid"
    
    if [ -f "$pid_file" ]; then
        local pid=$(cat "$pid_file")
        if ps -p "$pid" > /dev/null 2>&1; then
            print_info "Stopping $process_name (PID $pid)..."
            # Send SIGTERM (graceful shutdown - allows defer cleanup)
            kill -15 "$pid" 2>/dev/null || true
            
            # Wait up to 5 seconds for graceful shutdown
            for i in {1..10}; do
                if ! ps -p "$pid" > /dev/null 2>&1; then
                    print_info "Process stopped gracefully"
                    rm -f "$pid_file"
                    # Remove associated died file if it exists
                    rm -f "$CACHE_DIR/${process_name}.died" 2>/dev/null || true
                    return 0
                fi
                sleep 0.5
            done
            
            # Force kill if still running after timeout
            print_warn "Process did not stop gracefully after 5s, forcing with SIGKILL..."
            kill -9 "$pid" 2>/dev/null || true
            sleep 1
            # Log the time of forced termination
            echo "$(date '+%Y-%m-%d %H:%M:%S') - Forced kill (SIGKILL) sent to PID $pid" >> "$CACHE_DIR/${process_name}.died"
        else
            print_warn "PID $pid is not running anymore (stale PID file)"
        fi
        
        # Clean up PID file
        rm -f "$pid_file"
    else
        # Try killall as fallback
        if pgrep -q "$process_name"; then
            print_info "Stopping $process_name (using killall)..."
            killall "$process_name" 2>/dev/null || true
            sleep 2
            rm -f "$pid_file" 2>/dev/null || true
        else
            print_info "$process_name is not running"
        fi
    fi
}

# Start a process
start_process() {
    local process_name=$1
    local binary_path=$2
    
    # Check if already running  
    local pid_file="$CACHE_DIR/${process_name}.pid"
    if [ -f "$pid_file" ]; then
        local pid=$(cat "$pid_file")
        if ps -p "$pid" > /dev/null 2>&1; then
            print_warn "$process_name is already running (PID $pid)"
            return 1
        else
            # Clean stale PID file - log information about previous death
            print_info "Found stale PID file (PID $pid is not running)"
            if [ -f "$CACHE_DIR/${process_name}.died" ]; then
                print_info "Previous death info:"
                tail -1 "$CACHE_DIR/${process_name}.died" | sed 's/^/  /'
            fi
            rm -f "$pid_file"
        fi
    fi
    
    print_info "Starting $process_name..."
    "$binary_path" > "/tmp/${process_name}.out.log" 2>&1 &
    sleep 1
    
    if pgrep -q "$process_name"; then
        print_info "✓ $process_name started successfully"
        # Clear died file on successful start
        rm -f "$CACHE_DIR/${process_name}.died" 2>/dev/null || true
        return 0
    else
        print_error "Failed to start $process_name"
        # Maybe it crashed immediately - check logs
        if [ -s "/tmp/${process_name}.err.log" ]; then
            print_error "Recent error log:"
            tail -5 "/tmp/${process_name}.err.log" | sed 's/^/  /'
        fi
        return 1
    fi
}

# Restart a process
restart_process() {
    local process_name=$1
    local binary_path=$2
    
    stop_process "$process_name"
    sleep 1
    start_process "$process_name" "$binary_path"
}

# Show status
show_status() {
    echo ""
    echo "=== Memofy Process Status ==="
    echo ""

    for process in memofy-core memofy-ui; do
        local pid_file="$CACHE_DIR/${process}.pid"

        if [ -f "$pid_file" ]; then
            local pid=$(cat "$pid_file")
            if ps -p "$pid" > /dev/null 2>&1; then
                echo -e "${GREEN}✓${NC} $process is running (PID $pid)"
            else
                echo -e "${YELLOW}!${NC} $process PID file exists but process is dead (stale PID: $pid)"
            fi
        else
            if pgrep -q "$process"; then
                echo -e "${YELLOW}!${NC} $process is running but no PID file found"
            else
                echo -e "${RED}✗${NC} $process is not running"
            fi
        fi
    done

    echo ""

    # Show launchd service state
    echo "=== launchd Services ==="
    echo ""
    for label in com.memofy.core com.memofy.ui; do
        local svc_status
        svc_status=$(launchctl list 2>/dev/null | grep "$label" || true)
        if [ -n "$svc_status" ]; then
            local pid_col
            pid_col=$(echo "$svc_status" | awk '{print $1}')
            local exit_col
            exit_col=$(echo "$svc_status" | awk '{print $2}')
            if [ "$pid_col" != "-" ]; then
                echo -e "${GREEN}✓${NC} $label loaded (PID $pid_col)"
            else
                echo -e "${YELLOW}!${NC} $label loaded but not running (last exit: $exit_col)"
            fi
        else
            local plist="$HOME/Library/LaunchAgents/${label}.plist"
            if [ -f "$plist" ]; then
                echo -e "${YELLOW}!${NC} $label plist installed but not loaded"
            else
                echo -e "${RED}✗${NC} $label not installed as a service"
            fi
        fi
    done

    echo ""

    # Show current operating mode
    echo "=== Operating Mode ==="
    echo ""
    local status_file="$CACHE_DIR/status.json"
    if [ -f "$status_file" ]; then
        local mode
        mode=$(python3 -c "import json,sys; d=json.load(open('$status_file')); print(d.get('mode','unknown'))" 2>/dev/null || \
               grep -o '"mode":"[^"]*"' "$status_file" | cut -d'"' -f4)
        case "$mode" in
            auto)   echo -e "  Mode: ${GREEN}auto${NC}   (detection active, OBS controlled automatically)" ;;
            manual) echo -e "  Mode: ${YELLOW}manual${NC} (detection active, YOU control OBS recording)" ;;
            paused) echo -e "  Mode: ${RED}paused${NC} (all detection suspended)" ;;
            *)      echo -e "  Mode: ${YELLOW}$mode${NC}" ;;
        esac
        local obs_connected
        obs_connected=$(python3 -c "import json,sys; d=json.load(open('$status_file')); print(d.get('obs_connected',False))" 2>/dev/null || \
                        grep -o '"obs_connected":[^,}]*' "$status_file" | cut -d':' -f2 | tr -d ' ')
        if [ "$obs_connected" = "True" ] || [ "$obs_connected" = "true" ]; then
            echo -e "  OBS: ${GREEN}connected${NC}"
        else
            echo -e "  OBS: ${RED}disconnected${NC}"
        fi
    else
        echo "  Status file not found (is memofy-core running?)"
    fi

    echo ""
}

# Clean all PID files
clean_pids() {
    print_info "Cleaning all PID files..."
    rm -f "$CACHE_DIR"/*.pid 2>/dev/null || true
    print_info "✓ PID files cleaned"
}

# Set operating mode by writing command to IPC file
set_mode() {
    local mode=$1
    local status_file="$CACHE_DIR/status.json"

    if [ -z "$mode" ]; then
        # Show current mode
        if [ -f "$status_file" ]; then
            local current
            current=$(python3 -c "import json; d=json.load(open('$status_file')); print(d.get('mode','unknown'))" 2>/dev/null || \
                      grep -o '"mode":"[^"]*"' "$status_file" | cut -d'"' -f4)
            echo "Current mode: $current"
        else
            print_warn "Status file not found. Is memofy-core running?"
        fi
        echo ""
        echo "Available modes:  auto | manual | paused"
        echo "Usage: memofy-ctl mode <auto|manual|paused>"
        return 0
    fi

    case "$mode" in
        auto|manual|paused)
            mkdir -p "$CACHE_DIR"
            echo -n "$mode" > "$CACHE_DIR/cmd.txt"
            print_info "Command '$mode' sent to memofy-core"
            # Brief wait then confirm
            sleep 0.5
            if [ -f "$status_file" ]; then
                local new_mode
                new_mode=$(python3 -c "import json; d=json.load(open('$status_file')); print(d.get('mode','unknown'))" 2>/dev/null || \
                           grep -o '"mode":"[^"]*"' "$status_file" | cut -d'"' -f4)
                print_info "Mode is now: $new_mode"
            fi
            ;;
        *)
            print_error "Unknown mode: $mode"
            echo "Available modes: auto | manual | paused"
            exit 1
            ;;
    esac
}

# Generate plist content inline (no external file dependency)
_plist_content() {
    local label=$1      # e.g. com.memofy.core
    local binary=$2     # absolute path to binary
    local log_base=$3   # e.g. memofy-core  →  /tmp/memofy-core.{out,err}.log
    local throttle=$4   # seconds between restarts

    cat <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>${label}</string>

    <key>ProgramArguments</key>
    <array>
        <string>${binary}</string>
    </array>

    <key>RunAtLoad</key>
    <true/>

    <key>KeepAlive</key>
    <dict>
        <key>SuccessfulExit</key>
        <false/>
    </dict>

    <key>StandardOutPath</key>
    <string>/tmp/${log_base}.out.log</string>

    <key>StandardErrorPath</key>
    <string>/tmp/${log_base}.err.log</string>

    <key>EnvironmentVariables</key>
    <dict>
        <key>PATH</key>
        <string>/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin</string>
    </dict>

    <key>ProcessType</key>
    <string>Interactive</string>

    <key>ThrottleInterval</key>
    <integer>${throttle}</integer>
</dict>
</plist>
PLIST
}

# Install launchd services for both components
install_services() {
    local component=${1:-all}
    local launchagents_dir="$HOME/Library/LaunchAgents"

    mkdir -p "$launchagents_dir"

    install_plist() {
        local label=$1
        local plist_dst="$launchagents_dir/${label}.plist"

        case "$label" in
            com.memofy.core)
                _plist_content "$label" "$INSTALL_DIR/memofy-core" "memofy-core" 10 > "$plist_dst"
                ;;
            com.memofy.ui)
                _plist_content "$label" "$INSTALL_DIR/memofy-ui"   "memofy-ui"   5  > "$plist_dst"
                ;;
        esac

        print_info "Installed $plist_dst"

        # Unload first if already loaded (ignore error if not loaded)
        launchctl unload "$plist_dst" 2>/dev/null || true
        launchctl load "$plist_dst"
        print_info "✓ $label service loaded"
    }

    case "$component" in
        core) install_plist "com.memofy.core" ;;
        ui)   install_plist "com.memofy.ui"   ;;
        all)
            install_plist "com.memofy.core"
            install_plist "com.memofy.ui"
            ;;
        *)
            print_error "Unknown component: $component"
            echo "Usage: memofy-ctl install [core|ui]"
            exit 1
            ;;
    esac

    print_info ""
    print_info "Services will auto-start on login and restart if they crash."
    print_info "Use 'memofy-ctl status' to verify."
}

# Uninstall launchd services
uninstall_services() {
    local component=${1:-all}
    local launchagents_dir="$HOME/Library/LaunchAgents"

    unload_plist() {
        local label=$1
        local plist_dst="$launchagents_dir/${label}.plist"

        if [ -f "$plist_dst" ]; then
            launchctl unload "$plist_dst" 2>/dev/null || true
            rm -f "$plist_dst"
            print_info "✓ $label service removed"
        else
            print_warn "$label is not installed as a service"
        fi
    }

    case "$component" in
        core) unload_plist "com.memofy.core" ;;
        ui)   unload_plist "com.memofy.ui"   ;;
        all)
            stop_process "memofy-ui"
            stop_process "memofy-core"
            unload_plist "com.memofy.ui"
            unload_plist "com.memofy.core"
            ;;
        *)
            print_error "Unknown component: $component"
            echo "Usage: memofy-ctl uninstall [core|ui]"
            exit 1
            ;;
    esac
}

# Run comprehensive diagnostics
diagnose() {
    echo ""
    echo "╔════════════════════════════════════════╗"
    echo "║       Memofy Diagnostic Report          ║"
    echo "╚════════════════════════════════════════╝"
    echo ""
    
    # Process status
    echo "=== Running Processes ==="
    echo ""
    for process in memofy-core memofy-ui; do
        if pgrep -q "$process"; then
            local pid=$(pgrep "$process" | head -1)
            local mem=$(ps -p "$pid" -o rss= 2>/dev/null | awk '{print int($1/1024) "MB"}')
            echo -e "${GREEN}✓${NC} $process is running (PID $pid, MEM: $mem)"
        else
            echo -e "${RED}✗${NC} $process is not running"
        fi
    done
    echo ""
    
    # OBS connectivity
    echo "=== OBS WebSocket ==="
    echo ""
    if timeout 2 bash -c "echo '' > /dev/tcp/localhost/4455" 2>/dev/null; then
        print_success "OBS WebSocket is accessible on port 4455"
    else
        print_warn "Cannot reach OBS WebSocket on port 4455 (check if OBS is running)"
    fi
    echo ""
    
    # System info
    echo "=== System Information ==="
    echo ""
    echo "macOS Version: $(sw_vers -productVersion)"
    echo "Hostname: $(hostname)"
    echo "User: $USER"
    echo "Home Directory: $HOME"
    echo ""
    
    # Directory status
    echo "=== Directory Status ==="
    echo ""
    echo "Cache Directory: $CACHE_DIR"
    if [ -d "$CACHE_DIR" ]; then
        echo -e "${GREEN}✓${NC} exists"
        echo "  PID files: $(ls -1 "$CACHE_DIR"/*.pid 2>/dev/null | wc -l)"
    else
        echo -e "${RED}✗${NC} does not exist"
    fi
    echo ""
    
    # Log status
    echo "=== Recent Error Logs ==="
    echo ""
    if [ -f /tmp/memofy-core.err.log ]; then
        local error_count=$(wc -l < /tmp/memofy-core.err.log)
        echo "memofy-core error log: $error_count lines"
        if [ "$error_count" -gt 0 ]; then
            echo "Last 5 errors:"
            tail -5 /tmp/memofy-core.err.log | sed 's/^/  /'
        fi
    else
        echo "No error log found"
    fi
    echo ""
    
    # Additional checks
    echo "=== Checks ==="
    echo ""
    
    # Check if memofy binaries exist
    if [ -f "$INSTALL_DIR/memofy-core" ]; then
        echo -e "${GREEN}✓${NC} memofy-core binary found"
    else
        echo -e "${RED}✗${NC} memofy-core binary NOT found at $INSTALL_DIR/memofy-core"
    fi
    
    if [ -f "$INSTALL_DIR/memofy-ui" ]; then
        echo -e "${GREEN}✓${NC} memofy-ui binary found"
    else
        echo -e "${RED}✗${NC} memofy-ui binary NOT found at $INSTALL_DIR/memofy-ui"
    fi
    
    echo ""
    echo "=== Troubleshooting Tips ==="
    echo ""
    echo "1. If processes are not running:"
    echo "   memofy-ctl logs          # Check log output"
    echo "   memofy-ctl start         # Start processes"
    echo ""
    echo "2. If OBS WebSocket unreachable:"
    echo "   - Make sure OBS is running"
    echo "   - Enable: OBS > Tools > obs-websocket Settings > Enable WebSocket server"
    echo "   - Verify port is 4455"
    echo ""
    echo "3. For detailed logs:"
    echo "   tail -f /tmp/memofy-core.out.log"
    echo "   tail -f /tmp/memofy-core.err.log"
    echo ""
}

# Main command dispatcher
COMMAND=${1:-status}
INSTALL_DIR="${HOME}/.local/bin"

case "$COMMAND" in
    stop)
        if [ -n "$2" ]; then
            stop_process "$2"
        else
            stop_process "memofy-ui"
            stop_process "memofy-core"
        fi
        ;;
    start)
        if [ "$2" = "ui" ]; then
            start_process "memofy-ui" "$INSTALL_DIR/memofy-ui"
        elif [ "$2" = "core" ]; then
            start_process "memofy-core" "$INSTALL_DIR/memofy-core"
        else
            start_process "memofy-core" "$INSTALL_DIR/memofy-core"
            start_process "memofy-ui" "$INSTALL_DIR/memofy-ui"
        fi
        ;;
    restart)
        if [ "$2" = "ui" ]; then
            restart_process "memofy-ui" "$INSTALL_DIR/memofy-ui"
        elif [ "$2" = "core" ]; then
            restart_process "memofy-core" "$INSTALL_DIR/memofy-core"
        else
            restart_process "memofy-core" "$INSTALL_DIR/memofy-core"
            restart_process "memofy-ui" "$INSTALL_DIR/memofy-ui"
        fi
        ;;
    status)
        show_status
        ;;
    logs)
        show_logs
        ;;
    diagnose)
        diagnose
        ;;
    clean)
        stop_process "memofy-ui"
        stop_process "memofy-core"
        clean_pids
        ;;
    mode)
        set_mode "$2"
        ;;
    install)
        install_services "${2:-all}"
        ;;
    uninstall)
        uninstall_services "${2:-all}"
        ;;
    enable)
        if [ -z "$2" ]; then
            print_error "Usage: memofy-ctl enable <core|ui>"
            exit 1
        fi
        install_services "$2"
        ;;
    disable)
        if [ -z "$2" ]; then
            print_error "Usage: memofy-ctl disable <core|ui>"
            exit 1
        fi
        uninstall_services "$2"
        ;;
    *)
        echo "Usage: $0 {start|stop|restart|status|mode|install|uninstall|enable|disable|logs|diagnose|clean} [core|ui]"
        echo ""
        echo "Process Management:"
        echo "  start [core|ui]         - Start memofy processes"
        echo "  stop [core|ui]          - Stop memofy processes gracefully"
        echo "  restart [core|ui]       - Restart memofy processes"
        echo ""
        echo "Service Management (launchd auto-restart):"
        echo "  install [core|ui]       - Install and enable launchd service(s)"
        echo "  uninstall [core|ui]     - Disable and remove launchd service(s)"
        echo "  enable <core|ui>        - Enable launchd service for component"
        echo "  disable <core|ui>       - Disable launchd service for component"
        echo ""
        echo "Mode Control:"
        echo "  mode                    - Show current operating mode"
        echo "  mode auto               - Auto mode: detect + control OBS automatically"
        echo "  mode manual             - Manual mode: detection active, YOU control OBS"
        echo "  mode paused             - Paused: all detection suspended"
        echo ""
        echo "Diagnostics:"
        echo "  status                  - Show process & service status + current mode"
        echo "  logs                    - Show process logs (last 20 lines)"
        echo "  diagnose                - Run comprehensive diagnostics"
        echo "  clean                   - Stop all processes and clean PID files"
        echo ""
        echo "Examples:"
        echo "  $0 install               # Install both services (auto-restart on crash)"
        echo "  $0 mode manual           # Stop OBS being auto-controlled"
        echo "  $0 mode auto             # Resume automatic meeting recording"
        echo "  $0 status                # Full status including mode and launchd state"
        echo "  $0 stop ui               # Stop only the UI"
        echo "  $0 restart               # Restart both processes"
        echo "  $0 clean                 # Clean everything"
        echo ""
        exit 1
        ;;
esac
