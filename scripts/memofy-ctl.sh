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

# Stop a memofy process gracefully
stop_process() {
    local process_name=$1
    local pid_file="$CACHE_DIR/${process_name}.pid"
    
    if [ -f "$pid_file" ]; then
        local pid=$(cat "$pid_file")
        if ps -p "$pid" > /dev/null 2>&1; then
            print_info "Stopping $process_name (PID $pid)..."
            kill -15 "$pid" 2>/dev/null || true
            
            # Wait up to 5 seconds for graceful shutdown
            for i in {1..10}; do
                if ! ps -p "$pid" > /dev/null 2>&1; then
                    print_info "Process stopped gracefully"
                    rm -f "$pid_file"
                    return 0
                fi
                sleep 0.5
            done
            
            # Force kill if still running
            print_warn "Process did not stop gracefully, forcing..."
            kill -9 "$pid" 2>/dev/null || true
            sleep 1
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
            # Clean stale PID file
            rm -f "$pid_file"
        fi
    fi
    
    print_info "Starting $process_name..."
    "$binary_path" > "/tmp/${process_name}.out.log" 2>&1 &
    sleep 1
    
    if pgrep -q "$process_name"; then
        print_info "✓ $process_name started successfully"
        return 0
    else
        print_error "Failed to start $process_name"
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
    clean)
        stop_process "memofy-ui"
        stop_process "memofy-core"
        clean_pids
        ;;
    *)
        echo "Usage: $0 {start|stop|restart|status|clean} [core|ui]"
        echo ""
        echo "Commands:"
        echo "  start [core|ui]   - Start memofy processes"
        echo "  stop [core|ui]    - Stop memofy processes gracefully"
        echo "  restart [core|ui] - Restart memofy processes"
        echo "  status            - Show process status"
        echo "  clean             - Stop all processes and clean PID files"
        echo ""
        echo "Examples:"
        echo "  $0 status          # Show current status"
        echo "  $0 stop ui         # Stop only the UI"
        echo "  $0 restart         # Restart both processes"
        echo "  $0 clean           # Clean everything"
        exit 1
        ;;
esac
