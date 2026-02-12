#!/usr/bin/env bash
# End-to-end test for SatGate MCP Proxy
# Uses named pipes to communicate with the proxy process.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROXY="$SCRIPT_DIR/../../bin/satgate-mcp"
CONFIG="$SCRIPT_DIR/satgate-mcp.yaml"

if [[ ! -f "$PROXY" ]]; then
    echo "ERROR: satgate-mcp binary not found. Run 'go build -o bin/satgate-mcp ./cmd/satgate-mcp/' first."
    exit 1
fi

echo "=== SatGate MCP Proxy E2E Test ==="
echo ""

# Create named pipes
PIPE_IN=$(mktemp -u /tmp/mcp-in.XXXXX)
PIPE_OUT=$(mktemp -u /tmp/mcp-out.XXXXX)
mkfifo "$PIPE_IN" "$PIPE_OUT"

cleanup() {
    kill "$PROXY_PID" 2>/dev/null || true
    rm -f "$PIPE_IN" "$PIPE_OUT"
}
trap cleanup EXIT

# Start proxy
cd "$SCRIPT_DIR"
"$PROXY" --config "$CONFIG" < "$PIPE_IN" > "$PIPE_OUT" 2>/dev/null &
PROXY_PID=$!

# Open write fd (keep pipe open)
exec 3>"$PIPE_IN"

sleep 1

send() {
    echo "$1" >&3
}

recv() {
    read -t 5 line < "$PIPE_OUT"
    echo "$line"
}

# 1. Initialize
echo "--- initialize ---"
send '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}'
RESP=$(recv)
echo "  OK (server: $(echo "$RESP" | python3 -c "import sys,json; print(json.load(sys.stdin)['result']['serverInfo']['name'])" 2>/dev/null || echo 'parse error'))"
send '{"jsonrpc":"2.0","method":"notifications/initialized"}'

# 2. List tools
echo "--- tools/list ---"
send '{"jsonrpc":"2.0","id":2,"method":"tools/list"}'
RESP=$(recv)
TOOL_COUNT=$(echo "$RESP" | python3 -c "import sys,json; print(len(json.load(sys.stdin)['result']['tools']))" 2>/dev/null || echo '?')
echo "  Found $TOOL_COUNT tools"

# 3. Cheap calls (5 credits each x3 = 15 spent, 485 remaining)
echo "--- web_search x3 (5 credits each = 15 total) ---"
for i in 3 4 5; do
    send "{\"jsonrpc\":\"2.0\",\"id\":$i,\"method\":\"tools/call\",\"params\":{\"name\":\"web_search\",\"arguments\":{\"query\":\"test $i\"}}}"
    RESP=$(recv)
    STATUS=$(echo "$RESP" | python3 -c "import sys,json; r=json.load(sys.stdin); print('✓' if 'result' in r else '✗ '+r['error']['message'])" 2>/dev/null || echo '?')
    echo "  Call $i: $STATUS"
done

# 4. Expensive calls (50 credits each) — 485 remaining, so 9 calls = 450, leaves 35
echo "--- dalle_generate x9 (50 credits each = 450 total) ---"
for i in $(seq 6 14); do
    send "{\"jsonrpc\":\"2.0\",\"id\":$i,\"method\":\"tools/call\",\"params\":{\"name\":\"dalle_generate\",\"arguments\":{\"prompt\":\"test $i\"}}}"
    RESP=$(recv)
    STATUS=$(echo "$RESP" | python3 -c "
import sys,json
r=json.load(sys.stdin)
if 'result' in r:
    print('✓ allowed')
elif 'error' in r:
    d=r['error'].get('data',{})
    print(f'✗ {d.get(\"error\",\"denied\")} (remaining: {d.get(\"remaining_credits\",\"?\")})')
" 2>/dev/null || echo '?')
    echo "  Call $i: $STATUS"
done

# 5. One more expensive call — 35 remaining < 50 cost → DENIED
echo "--- dalle_generate (expect budget_exhausted) ---"
send '{"jsonrpc":"2.0","id":99,"method":"tools/call","params":{"name":"dalle_generate","arguments":{"prompt":"should fail"}}}'
RESP=$(recv)
ERROR_CODE=$(echo "$RESP" | python3 -c "import sys,json; r=json.load(sys.stdin); print(r.get('error',{}).get('data',{}).get('error','none'))" 2>/dev/null || echo "none")

echo ""
if [[ "$ERROR_CODE" == "budget_exhausted" ]]; then
    echo "✅ PASS — Budget enforcement working!"
    echo "   500 credits → 3 cheap (15) + 9 expensive (450) = 465 spent, 35 remaining"
    echo "   Next 50-credit call correctly denied with budget_exhausted"
else
    echo "❌ FAIL — Expected budget_exhausted, got: $ERROR_CODE"
    echo "   Response: $RESP"
fi
