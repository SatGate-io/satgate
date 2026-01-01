#!/bin/sh
# SatGate Startup Script
# Updated: 2024-12-22
set -eu

echo "=== SatGate Starting ==="
echo "Time: $(date)"
echo "PORT env: ${PORT:-not set}"
echo "BACKEND_PORT: ${BACKEND_PORT:-8083}"

# Start backend in background
echo "Starting backend on port ${BACKEND_PORT:-8083}..."
cd /app/backend
BACKEND_PORT="${BACKEND_PORT:-8083}" NODE_ENV="${NODE_ENV:-production}" node server.js &
BACKEND_PID=$!
echo "Backend PID: $BACKEND_PID"

# Wait for backend to be ready
echo "Waiting for backend to start..."
sleep 3

# Start Aperture in foreground so we see all output
echo ""
echo "=== Starting Aperture ==="
echo "LNC Network: ${LNC_NETWORK}"
echo "LNC Mailbox: ${LNC_MAILBOX}"
echo "LNC Passphrase length: ${#LNC_PASSPHRASE} chars"
echo "Config file: /app/aperture.yaml"
echo ""
echo "Note: LNC connection may take 60-120 seconds on first connection..."
echo ""

cd /app
exec aperture \
  --configfile=/app/aperture.yaml \
  --authenticator.passphrase="${LNC_PASSPHRASE}" \
  --authenticator.mailboxaddress="${LNC_MAILBOX}" \
  --authenticator.network="${LNC_NETWORK}"
