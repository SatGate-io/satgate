#!/bin/bash
set -e

# SatGate Cloud - Local Development Script
# Runs all services locally without Docker

echo "╔══════════════════════════════════════════════════════════════════╗"
echo "║              SatGate Cloud - Local Development                    ║"
echo "╚══════════════════════════════════════════════════════════════════╝"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$SCRIPT_DIR/.."

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

info() { echo -e "${GREEN}[INFO]${NC} $1"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }

# Check for required tools
command -v pnpm >/dev/null 2>&1 || { echo "pnpm required. Run: npm install -g pnpm"; exit 1; }
command -v psql >/dev/null 2>&1 || warn "psql not found - you may need to apply schema manually"

# Default environment
export NODE_ENV=development
export DATABASE_URL="${DATABASE_URL:-postgres://satgate:satgate_dev@localhost:5432/satgate}"
export SECRETS_ENCRYPTION_KEY="${SECRETS_ENCRYPTION_KEY:-dev_encryption_key_32_chars_long!}"
export INTERNAL_AUTH_TOKEN="${INTERNAL_AUTH_TOKEN:-dev_internal_token_for_testing}"
export L402_ROOT_KEY="${L402_ROOT_KEY:-0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef}"
export APP_URL="${APP_URL:-http://localhost:3000}"
export DATA_PLANE_URL="${DATA_PLANE_URL:-http://localhost:8080}"
export DATA_PLANE_INTERNAL_TOKEN="${DATA_PLANE_INTERNAL_TOKEN:-dev_internal_token_for_testing}"
export CORS_ORIGINS="${CORS_ORIGINS:-http://localhost:3000}"
export LIGHTNING_ENABLED="${LIGHTNING_ENABLED:-false}"

cd "$ROOT_DIR"

case "${1:-dev}" in
    install)
        info "Installing dependencies..."
        pnpm install
        ;;
    
    build)
        info "Building all packages..."
        pnpm run build
        ;;
    
    db)
        info "Starting Postgres (Docker)..."
        docker run --rm -d \
            --name satgate-postgres \
            -e POSTGRES_DB=satgate \
            -e POSTGRES_USER=satgate \
            -e POSTGRES_PASSWORD=satgate_dev \
            -p 5432:5432 \
            postgres:15-alpine
        
        echo "Waiting for Postgres..."
        sleep 3
        
        info "Applying schema..."
        psql "$DATABASE_URL" < db/schema.sql
        
        info "Postgres ready on localhost:5432"
        ;;
    
    db-stop)
        info "Stopping Postgres..."
        docker stop satgate-postgres || true
        ;;
    
    control-plane)
        info "Starting Control Plane on :3001..."
        export PORT=3001
        cd apps/control-plane
        pnpm run dev
        ;;
    
    data-plane)
        info "Starting Data Plane on :8080..."
        export PORT=8080
        cd apps/data-plane
        pnpm run dev
        ;;
    
    dashboard)
        info "Starting Dashboard on :3000..."
        cd apps/dashboard
        NEXT_PUBLIC_API_URL=http://localhost:3001 pnpm run dev
        ;;
    
    dev)
        info "Starting all services..."
        
        # Check if tmux is available for split panes
        if command -v tmux >/dev/null 2>&1; then
            tmux new-session -d -s satgate
            tmux split-window -h
            tmux split-window -v
            tmux select-pane -t 0
            tmux split-window -v
            
            tmux send-keys -t 0 "cd $ROOT_DIR && ./scripts/dev-local.sh control-plane" Enter
            tmux send-keys -t 1 "cd $ROOT_DIR && ./scripts/dev-local.sh data-plane" Enter
            tmux send-keys -t 2 "cd $ROOT_DIR && ./scripts/dev-local.sh dashboard" Enter
            tmux send-keys -t 3 "docker logs -f satgate-postgres 2>/dev/null || echo 'Postgres not running'" Enter
            
            tmux attach-session -t satgate
        else
            warn "tmux not found. Start services manually in separate terminals:"
            echo ""
            echo "Terminal 1: ./scripts/dev-local.sh db"
            echo "Terminal 2: ./scripts/dev-local.sh control-plane"
            echo "Terminal 3: ./scripts/dev-local.sh data-plane"
            echo "Terminal 4: ./scripts/dev-local.sh dashboard"
        fi
        ;;
    
    typecheck)
        info "Running type checks..."
        pnpm run typecheck
        ;;
    
    *)
        echo "Usage: $0 {install|build|db|db-stop|control-plane|data-plane|dashboard|dev|typecheck}"
        echo ""
        echo "Commands:"
        echo "  install       - Install all dependencies"
        echo "  build         - Build all packages"
        echo "  db            - Start Postgres in Docker and apply schema"
        echo "  db-stop       - Stop Postgres"
        echo "  control-plane - Start control plane on :3001"
        echo "  data-plane    - Start data plane on :8080"
        echo "  dashboard     - Start dashboard on :3000"
        echo "  dev           - Start all services (requires tmux)"
        echo "  typecheck     - Run type checks"
        exit 1
        ;;
esac

