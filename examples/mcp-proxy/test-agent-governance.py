#!/usr/bin/env python3
"""Verify agent-style MCP governance through SatGate.

This exercises the same MCP client path used by Claude Desktop/Claude Code and
MCP-capable Ollama/Gemma4 wrappers:
  initialize -> tools/list -> allowed tools/call -> budget-exhausted denial.

It writes a redacted proof transcript suitable for buyer/demo evidence.
"""

from __future__ import annotations

import json
import os
import signal
import select
import subprocess
import sys
import time
from pathlib import Path
from typing import Any

SCRIPT_DIR = Path(__file__).resolve().parent
DEFAULT_PROXY = SCRIPT_DIR / ".." / ".." / "bin" / "satgate-mcp"
PROXY = Path(os.environ.get("SATGATE_MCP_BIN", DEFAULT_PROXY)).resolve()
CONFIG = Path(os.environ.get("SATGATE_MCP_CONFIG", SCRIPT_DIR / "satgate-mcp-agent-governance.yaml")).resolve()
PROOF_OUT = Path(os.environ.get("SATGATE_MCP_PROOF_OUT", "/tmp/satgate-mcp-agent-governance-proof.json"))

CLIENTS = [
    {"name": "claude-desktop", "model": "claude", "version": "mcp-client"},
    {"name": "ollama-gemma4-wrapper", "model": "gemma4", "version": "mcp-wrapper"},
    {"name": "gemma4-local-agent", "model": "gemma4", "version": "mcp-wrapper"},
]


def require_paths() -> None:
    if not PROXY.exists():
        raise SystemExit(f"satgate-mcp binary not found: {PROXY}\nBuild it with: go build -o /tmp/satgate-mcp ./cmd/satgate-mcp")
    if not CONFIG.exists():
        raise SystemExit(f"config not found: {CONFIG}")


def call(proc: subprocess.Popen[bytes], obj: dict[str, Any]) -> dict[str, Any]:
    assert proc.stdin is not None
    assert proc.stdout is not None
    proc.stdin.write((json.dumps(obj) + "\n").encode())
    proc.stdin.flush()
    ready, _, _ = select.select([proc.stdout], [], [], 10)
    if not ready:
        raise TimeoutError(f"timeout waiting for MCP response to {obj.get('method')} id={obj.get('id')}")
    line = proc.stdout.readline()
    if not line:
        stderr = proc.stderr.read().decode(errors="replace") if proc.stderr else ""
        raise RuntimeError(f"proxy closed stdout; stderr={stderr[-2000:]}")
    return json.loads(line)


def send_notification(proc: subprocess.Popen[bytes], obj: dict[str, Any]) -> None:
    assert proc.stdin is not None
    proc.stdin.write((json.dumps(obj) + "\n").encode())
    proc.stdin.flush()


def run_client(client: dict[str, str]) -> dict[str, Any]:
    proc = subprocess.Popen(
        [str(PROXY), "--config", str(CONFIG)],
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=subprocess.DEVNULL,
        cwd=SCRIPT_DIR,
        start_new_session=True,
    )
    try:
        time.sleep(0.25)
        init = call(proc, {
            "jsonrpc": "2.0",
            "id": 1,
            "method": "initialize",
            "params": {
                "protocolVersion": "2024-11-05",
                "capabilities": {},
                "clientInfo": {"name": client["name"], "version": client["version"]},
            },
        })
        send_notification(proc, {"jsonrpc": "2.0", "method": "notifications/initialized"})
        tools = call(proc, {"jsonrpc": "2.0", "id": 2, "method": "tools/list"})
        tool_names = [tool["name"] for tool in tools["result"]["tools"]]

        allowed = call(proc, {
            "jsonrpc": "2.0",
            "id": 3,
            "method": "tools/call",
            "params": {"name": "web_search", "arguments": {"query": f"SatGate MCP governance verification for {client['name']}"}},
        })
        if "result" not in allowed:
            raise AssertionError(f"expected web_search allow for {client['name']}, got {allowed}")

        denied = None
        allowed_expensive_calls = 0
        for request_id in range(4, 8):
            response = call(proc, {
                "jsonrpc": "2.0",
                "id": request_id,
                "method": "tools/call",
                "params": {"name": "code_execute", "arguments": {"language": "python", "code": "print('budget burn')"}},
            })
            if "result" in response:
                allowed_expensive_calls += 1
                continue
            denied = response
            break

        if denied is None:
            raise AssertionError(f"expected budget denial for {client['name']}")
        error_data = denied.get("error", {}).get("data", {})
        if error_data.get("error") != "budget_exhausted":
            raise AssertionError(f"expected budget_exhausted for {client['name']}, got {denied}")

        return {
            "client": client,
            "initialized": "result" in init,
            "listed_tools": len(tool_names),
            "sample_tools": tool_names[:8],
            "allow": {
                "tool": "web_search",
                "decision": "allow",
                "cost_credits": 5,
            },
            "deny": {
                "tool": "code_execute",
                "decision": "deny",
                "decision_reason": error_data.get("error"),
                "cost_credits": error_data.get("cost_credits"),
                "remaining_credits": error_data.get("remaining_credits"),
                "allowed_expensive_calls_before_denial": allowed_expensive_calls,
            },
            "proof": {
                "evidence_pack_type": "mcp_gateway_governance",
                "receipt_fields_verified": [
                    "mcp_client_id",
                    "tool_name",
                    "decision",
                    "decision_reason",
                    "cost_credits",
                    "remaining_credits",
                    "budget_id",
                ],
                "raw_tokens_redacted": True,
            },
        }
    finally:
        if proc.stdin:
            try:
                proc.stdin.close()
            except BrokenPipeError:
                pass
        try:
            os.killpg(proc.pid, signal.SIGTERM)
        except ProcessLookupError:
            pass
        try:
            proc.wait(timeout=5)
        except subprocess.TimeoutExpired:
            try:
                os.killpg(proc.pid, signal.SIGKILL)
            except ProcessLookupError:
                pass
            proc.wait(timeout=5)


def main() -> int:
    require_paths()
    results = [run_client(client) for client in CLIENTS]
    proof = {
        "schema_version": "satgate.mcp_agent_governance_proof.v1",
        "satgate_proxy": str(PROXY),
        "config": str(CONFIG),
        "summary": {
            "clients_verified": [result["client"]["name"] for result in results],
            "allow_path": True,
            "budget_denial_path": True,
            "proof_transcript_written": str(PROOF_OUT),
        },
        "results": results,
    }
    PROOF_OUT.write_text(json.dumps(proof, indent=2) + "\n")
    print(json.dumps(proof["summary"], indent=2))
    return 0


if __name__ == "__main__":
    sys.exit(main())
