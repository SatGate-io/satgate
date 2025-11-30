#!/usr/bin/env bash
# =============================================================================
# SatGate Aperture Watchdog
# =============================================================================
# This script monitors Aperture and restarts it if:
#   1. It's not running
#   2. It's been running for more than MAX_UPTIME_HOURS
#   3. Health check fails
#
# Usage:
#   ./aperture-watchdog.sh
#
# Cron (check every 5 minutes):
#   */5 * * * * /path/to/aperture-watchdog.sh >> /tmp/aperture-watchdog.log 2>&1
#
# =============================================================================

set -euo pipefail

# Configuration
APERTURE_BIN="${APERTURE_BIN:-/Users/waynewonder/go/bin/aperture}"
APERTURE_CONFIG="${APERTURE_CONFIG:-$HOME/.aperture/aperture.yaml}"
APERTURE_PORT="${APERTURE_PORT:-8081}"
MAX_UPTIME_HOURS="${MAX_UPTIME_HOURS:-12}"  # Restart after 12 hours
LOG_PREFIX="[$(date '+%Y-%m-%d %H:%M:%S')] WATCHDOG:"

# LNC Credentials (set these or export before running)
LNC_PASSPHRASE="${LNC_PASSPHRASE:-}"
LNC_MAILBOX="${LNC_MAILBOX:-mailbox.terminal.lightning.today:443}"
LNC_NETWORK="${LNC_NETWORK:-mainnet}"

# Check if LNC passphrase is set
if [ -z "$LNC_PASSPHRASE" ]; then
    echo "$LOG_PREFIX ERROR: LNC_PASSPHRASE not set. Export it before running."
    exit 1
fi

# Function to get Aperture PID
get_aperture_pid() {
    pgrep -f "aperture.*--configfile" 2>/dev/null || echo ""
}

# Function to get uptime in seconds
get_uptime_seconds() {
    local pid=$1
    if [ -n "$pid" ]; then
        # macOS: use ps with etime
        local etime=$(ps -p "$pid" -o etime= 2>/dev/null | tr -d ' ')
        if [ -n "$etime" ]; then
            # Parse etime format: [[dd-]hh:]mm:ss
            local days=0 hours=0 mins=0 secs=0
            if [[ "$etime" == *-* ]]; then
                days=$(echo "$etime" | cut -d'-' -f1)
                etime=$(echo "$etime" | cut -d'-' -f2)
            fi
            IFS=':' read -ra parts <<< "$etime"
            if [ ${#parts[@]} -eq 3 ]; then
                hours=${parts[0]}
                mins=${parts[1]}
                secs=${parts[2]}
            elif [ ${#parts[@]} -eq 2 ]; then
                mins=${parts[0]}
                secs=${parts[1]}
            fi
            echo $(( days*86400 + hours*3600 + mins*60 + secs ))
        else
            echo "0"
        fi
    else
        echo "0"
    fi
}

# Function to start Aperture
start_aperture() {
    echo "$LOG_PREFIX Starting Aperture..."
    nohup "$APERTURE_BIN" \
        --configfile="$APERTURE_CONFIG" \
        --authenticator.network="$LNC_NETWORK" \
        --authenticator.passphrase="$LNC_PASSPHRASE" \
        --authenticator.mailboxaddress="$LNC_MAILBOX" \
        > /tmp/aperture.log 2>&1 &
    
    sleep 3
    local new_pid=$(get_aperture_pid)
    if [ -n "$new_pid" ]; then
        echo "$LOG_PREFIX Aperture started with PID $new_pid"
    else
        echo "$LOG_PREFIX ERROR: Failed to start Aperture"
        exit 1
    fi
}

# Function to stop Aperture
stop_aperture() {
    local pid=$1
    echo "$LOG_PREFIX Stopping Aperture (PID: $pid)..."
    kill "$pid" 2>/dev/null || true
    sleep 2
    # Force kill if still running
    if kill -0 "$pid" 2>/dev/null; then
        kill -9 "$pid" 2>/dev/null || true
    fi
}

# Function to check health
check_health() {
    local response=$(curl -s -o /dev/null -w "%{http_code}" "http://127.0.0.1:$APERTURE_PORT/health" 2>/dev/null || echo "000")
    if [ "$response" == "200" ] || [ "$response" == "402" ] || [ "$response" == "404" ]; then
        return 0  # Aperture is responding
    else
        return 1  # Aperture is not responding
    fi
}

# Main logic
main() {
    local pid=$(get_aperture_pid)
    local max_uptime_secs=$((MAX_UPTIME_HOURS * 3600))
    
    if [ -z "$pid" ]; then
        echo "$LOG_PREFIX Aperture not running. Starting..."
        start_aperture
        exit 0
    fi
    
    local uptime_secs=$(get_uptime_seconds "$pid")
    local uptime_hours=$((uptime_secs / 3600))
    
    echo "$LOG_PREFIX Aperture running (PID: $pid, Uptime: ${uptime_hours}h)"
    
    # Check if uptime exceeds max
    if [ "$uptime_secs" -gt "$max_uptime_secs" ]; then
        echo "$LOG_PREFIX Uptime exceeds ${MAX_UPTIME_HOURS}h. Restarting to refresh macaroon keys..."
        stop_aperture "$pid"
        start_aperture
        exit 0
    fi
    
    # Health check
    if ! check_health; then
        echo "$LOG_PREFIX Health check failed. Restarting..."
        stop_aperture "$pid"
        start_aperture
        exit 0
    fi
    
    echo "$LOG_PREFIX Aperture healthy."
}

main "$@"

