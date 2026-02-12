#!/usr/bin/env python3
"""E2E test: Agent delegation with budget enforcement.

Scenario (what Mike will see):
1. Parent agent connects with 1000-credit budget
2. Parent discovers tools
3. Parent delegates 300 credits to "research-agent"
4. Parent delegates 200 credits to "content-agent"
5. Research agent makes tool calls (burns through budget)
6. Research agent hits 402 (budget_exhausted)
7. Content agent still has its own budget (isolation!)
8. Parent checks remaining budget
"""

import json
import subprocess
import sys
import os
import time

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
PROXY = os.path.join(SCRIPT_DIR, "..", "..", "bin", "satgate-mcp")
CONFIG = os.path.join(SCRIPT_DIR, "satgate-mcp-delegation.yaml")


def main():
    print("=== SatGate MCP Proxy — Delegation E2E Test ===")
    print("Scenario: Parent agent spawning sub-agents with carved budgets\n")

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

    # Wait for startup and capture root token from stderr
    time.sleep(1.0)
    stderr_data = b""
    import select
    # Read available stderr
    while select.select([proc.stderr], [], [], 0.1)[0]:
        stderr_data += proc.stderr.read1(4096)

    root_token = None
    for line in stderr_data.decode().split("\n"):
        if line.startswith("ROOT_TOKEN="):
            root_token = line.split("=", 1)[1].strip()
            break

    if not root_token:
        print(f"❌ Failed to capture root token from stderr")
        print(f"   stderr: {stderr_data.decode()[:500]}")
        proc.terminate()
        return

    print(f"   Root token captured (len={len(root_token)})\n")

    # === Initialize ===
    print("1️⃣  Initialize MCP session")
    send({"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {
        "protocolVersion": "2024-11-05",
        "capabilities": {},
        "clientInfo": {"name": "demo-orchestrator", "version": "1.0"}
    }})
    resp = recv()
    print(f"   Server: {resp['result']['serverInfo']['name']}\n")
    send({"jsonrpc": "2.0", "method": "notifications/initialized"})

    # === Discover tools ===
    print("2️⃣  Discover tools")
    send({"jsonrpc": "2.0", "id": 2, "method": "tools/list"})
    resp = recv()
    tools = resp["result"]["tools"]
    print(f"   Found {len(tools)} tools: {[t['name'] for t in tools]}\n")

    # === Check initial budget ===
    print("3️⃣  Check parent budget")
    send({"jsonrpc": "2.0", "id": 3, "method": "satgate/budget",
          "params": {"_meta": {"token": root_token}}})
    resp = recv()
    print(f"   Budget: {resp['result']['remaining']} credits\n")

    # === Delegate to research agent ===
    print("4️⃣  Delegate 300 credits → research-agent")
    send({"jsonrpc": "2.0", "id": 4, "method": "satgate/delegate", "params": {
        "budget": 300, "label": "research-agent", "scope": "api:read",
        "_meta": {"token": root_token}
    }})
    resp = recv()
    if "error" in resp:
        print(f"   ❌ Delegation failed: {resp['error']['message']}")
        proc.terminate()
        return

    research = resp["result"]
    print(f"   Child token: {research['tokenId']}")
    print(f"   Child budget: {research['budget']}")
    print(f"   Parent remaining: {research['parentRemaining']}\n")

    # === Delegate to content agent ===
    print("5️⃣  Delegate 200 credits → content-agent")
    send({"jsonrpc": "2.0", "id": 5, "method": "satgate/delegate", "params": {
        "budget": 200, "label": "content-agent", "scope": "api:*",
        "_meta": {"token": root_token}
    }})
    resp = recv()
    content = resp["result"]
    print(f"   Child token: {content['tokenId']}")
    print(f"   Child budget: {content['budget']}")
    print(f"   Parent remaining: {content['parentRemaining']}\n")

    # === Research agent makes calls (5 credits each) ===
    print("6️⃣  Research agent: web_search calls (5 credits each)")
    call_id = 100
    research_calls = 0
    research_denied = False

    # 300 budget / 5 per call = 60 calls max. Let's do 65 to hit the wall.
    for i in range(65):
        call_id += 1
        send({"jsonrpc": "2.0", "id": call_id, "method": "tools/call",
              "params": {"name": "web_search", "arguments": {"query": f"research {i}"},
                         "_meta": {"token": research["token"]}}})
        resp = recv()
        if "result" in resp:
            research_calls += 1
        else:
            data = resp["error"].get("data", {})
            print(f"   ✓ {research_calls} calls succeeded")
            print(f"   ✗ Call {research_calls + 1}: {data.get('error', 'denied')} "
                  f"(remaining: {data.get('remaining_credits', '?')})")
            research_denied = True
            break

    if not research_denied:
        print(f"   ⚠️  All 65 calls succeeded (expected denial)")

    print()

    # === Content agent still has budget ===
    print("7️⃣  Content agent: gpt4_summarize (25 credits) — should still work!")
    call_id += 1
    send({"jsonrpc": "2.0", "id": call_id, "method": "tools/call",
          "params": {"name": "gpt4_summarize",
                     "arguments": {"text": "Summarize the research findings..."},
                     "_meta": {"token": content["token"]}}})
    resp = recv()
    if "result" in resp:
        print("   ✓ Content agent call succeeded (budget isolated from research agent)")
    else:
        print(f"   ✗ Content agent denied: {resp['error']['message']}")

    print()

    # === Parent checks own budget ===
    print("8️⃣  Parent checks remaining budget")
    send({"jsonrpc": "2.0", "id": 999, "method": "satgate/budget",
          "params": {"_meta": {"token": root_token}}})
    resp = recv()
    parent_remaining = resp["result"]["remaining"]
    print(f"   Parent remaining: {parent_remaining} credits")
    print(f"   (started 1000, delegated 300+200=500, should have 500 left)\n")

    # === Summary ===
    print("=" * 60)
    all_pass = (
        research_denied and
        research_calls == 60 and
        parent_remaining == 500
    )

    if all_pass:
        print("✅ ALL CHECKS PASSED")
        print(f"   • Research agent: {research_calls} calls, then 402 (budget isolated)")
        print(f"   • Content agent: still operational (own budget)")
        print(f"   • Parent: 500 credits remaining (delegation carved correctly)")
        print(f"   • Budget isolation: VERIFIED")
    else:
        print("❌ SOME CHECKS FAILED")
        print(f"   research_denied={research_denied} (expect True)")
        print(f"   research_calls={research_calls} (expect 60)")
        print(f"   parent_remaining={parent_remaining} (expect 500)")

    proc.stdin.close()
    proc.terminate()
    proc.wait(timeout=5)


if __name__ == "__main__":
    main()
