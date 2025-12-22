#!/bin/bash
set -e

echo "=== phoenixd Starting ==="
echo "Time: $(date)"
echo "Chain: ${PHOENIX_CHAIN:-mainnet}"
echo "Data dir: ${PHOENIX_DATADIR:-/data/.phoenix}"

# Create config if custom password is provided
CONFIG_FILE="${PHOENIX_DATADIR}/phoenix.conf"
if [ -n "$PHOENIXD_PASSWORD" ]; then
    echo "Setting custom HTTP password..."
    mkdir -p "$(dirname $CONFIG_FILE)"
    cat > "$CONFIG_FILE" << EOF
# Auto-generated phoenixd config
http-password=$PHOENIXD_PASSWORD
http-bind-ip=0.0.0.0
http-bind-port=${PHOENIX_HTTP_BIND_PORT:-9740}
EOF
fi

# Determine chain argument
CHAIN_ARG=""
case "${PHOENIX_CHAIN}" in
    mainnet) CHAIN_ARG="--chain=mainnet" ;;
    testnet) CHAIN_ARG="--chain=testnet" ;;
    *) CHAIN_ARG="--chain=mainnet" ;;
esac

echo ""
echo "Starting phoenixd..."
echo "Note: First start may take a few minutes to sync."
echo ""

# Run phoenixd
exec phoenixd \
    --datadir="${PHOENIX_DATADIR}" \
    --http-bind-ip=0.0.0.0 \
    --http-bind-port="${PHOENIX_HTTP_BIND_PORT:-9740}" \
    --agree-to-terms-of-service \
    $CHAIN_ARG

