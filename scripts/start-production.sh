#!/bin/bash
# =============================================================================
# SatGate - Production Startup Script
# =============================================================================
#
# This script starts all components needed for SatGate:
#   1. Backend Node.js server
#   2. Aperture L402 proxy (requires LNC credentials)
#
# Usage:
#   ./scripts/start-production.sh
#
# Environment Variables:
#   LNC_PASSPHRASE        - Lightning Node Connect pairing phrase (required)
#   LNC_NETWORK           - Bitcoin network (default: mainnet)
#   LNC_MAILBOX           - Mailbox server address (default: mailbox.terminal.lightning.today:443)
#   BACKEND_PORT          - Backend server port (default: 8083)
#   APERTURE_PORT         - Aperture proxy port (default: 8081)
#
# =============================================================================

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
APERTURE_CONFIG="$HOME/.aperture/aperture.yaml"

BACKEND_PORT="${BACKEND_PORT:-8083}"
APERTURE_PORT="${APERTURE_PORT:-8081}"
LNC_NETWORK="${LNC_NETWORK:-mainnet}"
LNC_MAILBOX="${LNC_MAILBOX:-mailbox.terminal.lightning.today:443}"

# Logging functions
log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Check if a port is in use
check_port() {
    local port=$1
    if lsof -i :$port > /dev/null 2>&1; then
        return 0  # Port is in use
    else
        return 1  # Port is free
    fi
}

# Kill process on port
kill_port() {
    local port=$1
    local pids=$(lsof -t -i :$port 2>/dev/null)
    if [ -n "$pids" ]; then
        echo "$pids" | xargs kill -9 2>/dev/null || true
        sleep 1
    fi
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check Node.js
    if ! command -v node &> /dev/null; then
        log_error "Node.js is not installed"
        exit 1
    fi
    
    # Check Aperture
    if ! command -v aperture &> /dev/null; then
        log_error "Aperture is not installed. Run: cd ~/aperture && make install"
        exit 1
    fi
    
    # Check LNC passphrase
    if [ -z "$LNC_PASSPHRASE" ]; then
        log_error "LNC_PASSPHRASE environment variable is not set"
        log_info "Usage: LNC_PASSPHRASE='your phrase' ./scripts/start-production.sh"
        exit 1
    fi
    
    # Check aperture.yaml exists
    if [ ! -f "$APERTURE_CONFIG" ]; then
        log_warn "Aperture config not found at $APERTURE_CONFIG"
        log_info "Creating config directory..."
        mkdir -p "$(dirname "$APERTURE_CONFIG")"
        cp "$PROJECT_DIR/aperture.yaml" "$APERTURE_CONFIG"
    fi
    
    log_info "Prerequisites OK"
}

# Start backend
start_backend() {
    log_info "Starting backend on port $BACKEND_PORT..."
    
    if check_port $BACKEND_PORT; then
        log_warn "Port $BACKEND_PORT is already in use"
        read -p "Kill existing process? (y/n) " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            kill_port $BACKEND_PORT
        else
            log_error "Cannot start backend - port in use"
            exit 1
        fi
    fi
    
    cd "$PROJECT_DIR"
    NODE_ENV=production BACKEND_PORT=$BACKEND_PORT node backend/server.js &
    BACKEND_PID=$!
    
    sleep 2
    
    if ! kill -0 $BACKEND_PID 2>/dev/null; then
        log_error "Backend failed to start"
        exit 1
    fi
    
    log_info "Backend started (PID: $BACKEND_PID)"
}

# Start Aperture
start_aperture() {
    log_info "Starting Aperture on port $APERTURE_PORT..."
    
    if check_port $APERTURE_PORT; then
        log_warn "Port $APERTURE_PORT is already in use"
        read -p "Kill existing process? (y/n) " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            kill_port $APERTURE_PORT
        else
            log_error "Cannot start Aperture - port in use"
            exit 1
        fi
    fi
    
    aperture \
        --configfile="$APERTURE_CONFIG" \
        --authenticator.network="$LNC_NETWORK" \
        --authenticator.passphrase="$LNC_PASSPHRASE" \
        --authenticator.mailboxaddress="$LNC_MAILBOX" &
    APERTURE_PID=$!
    
    sleep 5
    
    if ! kill -0 $APERTURE_PID 2>/dev/null; then
        log_error "Aperture failed to start"
        exit 1
    fi
    
    log_info "Aperture started (PID: $APERTURE_PID)"
}

# Health check
health_check() {
    log_info "Running health checks..."
    
    # Check backend
    if curl -sf "http://127.0.0.1:$BACKEND_PORT/health" > /dev/null; then
        log_info "✓ Backend health check passed"
    else
        log_error "✗ Backend health check failed"
        return 1
    fi
    
    # Check Aperture (should return 402 or 200)
    local status=$(curl -sf -o /dev/null -w "%{http_code}" "http://127.0.0.1:$APERTURE_PORT/api/free/ping")
    if [ "$status" = "200" ]; then
        log_info "✓ Aperture health check passed"
    else
        log_error "✗ Aperture health check failed (status: $status)"
        return 1
    fi
    
    return 0
}

# Cleanup on exit
cleanup() {
    log_info "Shutting down..."
    
    if [ -n "$BACKEND_PID" ]; then
        kill $BACKEND_PID 2>/dev/null || true
    fi
    
    if [ -n "$APERTURE_PID" ]; then
        kill $APERTURE_PID 2>/dev/null || true
    fi
    
    log_info "Shutdown complete"
}

# Main
main() {
    trap cleanup EXIT INT TERM
    
    echo "=============================================="
    echo "  ⚡ SatGate - Production Start              "
    echo "=============================================="
    echo
    
    check_prerequisites
    start_backend
    start_aperture
    
    if health_check; then
        echo
        log_info "=============================================="
        log_info "  ⚡ SatGate is running!                     "
        log_info "=============================================="
        log_info ""
        log_info "  Frontend:  http://127.0.0.1:$BACKEND_PORT"
        log_info "  Aperture:  http://127.0.0.1:$APERTURE_PORT"
        log_info "  Health:    http://127.0.0.1:$BACKEND_PORT/health"
        log_info ""
        log_info "  Press Ctrl+C to stop"
        log_info "=============================================="
        
        # Wait for interrupt
        wait
    else
        log_error "Health checks failed - shutting down"
        exit 1
    fi
}

main "$@"


