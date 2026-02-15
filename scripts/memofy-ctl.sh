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
}

# Clean all PID files
clean_pids() {
    print_info "Cleaning all PID files..."
    rm -f "$CACHE_DIR"/*.pid 2>/dev/null || true
    print_info "✓ PID files cleaned"
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
    *)
        echo "Usage: $0 {start|stop|restart|status|logs|diagnose|clean} [core|ui]"
        echo ""
        echo "Commands:"
        echo "  start [core|ui]   - Start memofy processes"
        echo "  stop [core|ui]    - Stop memofy processes gracefully"
        echo "  restart [core|ui] - Restart memofy processes"
        echo "  status            - Show process status"
        echo "  logs              - Show process logs (last 20 lines)"
        echo "  diagnose          - Run comprehensive diagnostics"
        echo "  clean             - Stop all processes and clean PID files"
        echo ""
        echo "Examples:"
        echo "  $0 status          # Show current status"
        echo "  $0 logs            # Show recent logs"
        echo "  $0 stop ui         # Stop only the UI"
        echo "  $0 restart         # Restart both processes"
        echo "  $0 clean           # Clean everything"
        echo ""
        echo "Troubleshooting:"
        echo "  If core won't start:"
        echo "    1. Check logs: $0 logs"
        echo "    2. Ensure OBS is running"
        echo "    3. Enable WebSocket: OBS > Tools > obs-websocket Settings"
        echo "    4. Check port 4455 is available"
        echo ""
        exit 1
        ;;
esac
