#!/usr/bin/env python3
"""End-to-end test for SatGate MCP Proxy with budget enforcement."""

import json
import subprocess
import sys
import os
import time

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
PROXY = os.path.join(SCRIPT_DIR, "..", "..", "bin", "satgate-mcp")
CONFIG = os.path.join(SCRIPT_DIR, "satgate-mcp.yaml")


def main():
    print("=== SatGate MCP Proxy E2E Test ===\n")

    proc = subprocess.Popen(
        [PROXY, "--config", CONFIG],
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        cwd=SCRIPT_DIR,
    )

    def send(obj):
        line = json.dumps(obj) + "\n"
        proc.stdin.write(line.encode())
        proc.stdin.flush()

    def recv():
        line = proc.stdout.readline()
        if not line:
            raise RuntimeError("proxy closed stdout")
        return json.loads(line)

    time.sleep(0.5)  # let upstream start

    # 1. Initialize
    print("--- initialize ---")
    send({"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {
        "protocolVersion": "2024-11-05",
        "capabilities": {},
        "clientInfo": {"name": "test", "version": "1.0"}
    }})
    resp = recv()
    print(f"  Server: {resp['result']['serverInfo']['name']}")

    send({"jsonrpc": "2.0", "method": "notifications/initialized"})

    # 2. List tools
    print("--- tools/list ---")
    send({"jsonrpc": "2.0", "id": 2, "method": "tools/list"})
    resp = recv()
    tools = resp["result"]["tools"]
    print(f"  Found {len(tools)} tools: {[t['name'] for t in tools]}")

    # 3. Cheap calls (5 credits each)
    print("--- web_search x3 (5 credits each = 15 total) ---")
    for i in range(3, 6):
        send({"jsonrpc": "2.0", "id": i, "method": "tools/call",
              "params": {"name": "web_search", "arguments": {"query": f"test {i}"}}})
        resp = recv()
        if "result" in resp:
            print(f"  Call {i}: ✓ allowed")
        else:
            print(f"  Call {i}: ✗ {resp['error']['message']}")

    # 4. Expensive calls (50 credits each) — 485 remaining
    print("--- dalle_generate x9 (50 credits each = 450 total) ---")
    for i in range(6, 15):
        send({"jsonrpc": "2.0", "id": i, "method": "tools/call",
              "params": {"name": "dalle_generate", "arguments": {"prompt": f"test {i}"}}})
        resp = recv()
        if "result" in resp:
            print(f"  Call {i}: ✓ allowed")
        else:
            data = resp["error"].get("data", {})
            print(f"  Call {i}: ✗ {data.get('error', 'denied')} (remaining: {data.get('remaining_credits', '?')})")

    # 5. One more expensive call — should be DENIED
    print("--- dalle_generate (expect budget_exhausted) ---")
    send({"jsonrpc": "2.0", "id": 99, "method": "tools/call",
          "params": {"name": "dalle_generate", "arguments": {"prompt": "should fail"}}})
    resp = recv()

    error_code = resp.get("error", {}).get("data", {}).get("error", "none")

    print()
    if error_code == "budget_exhausted":
        remaining = resp["error"]["data"]["remaining_credits"]
        print("✅ PASS — Budget enforcement working!")
        print(f"   500 credits → 15 (searches) + 450 (images) = 465 spent, {remaining} remaining")
        print("   Next 50-credit call correctly denied with budget_exhausted")
    else:
        print(f"❌ FAIL — Expected budget_exhausted, got: {error_code}")
        print(f"   Response: {json.dumps(resp, indent=2)}")

    # Cleanup
    proc.stdin.close()
    proc.terminate()
    proc.wait(timeout=5)


if __name__ == "__main__":
    main()
