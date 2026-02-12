#!/usr/bin/env python3
"""Mock MCP server for testing SatGate MCP proxy.

Exposes 5 tools with varying costs:
- web_search (cheap)
- database_query (cheap)
- gpt4_summarize (moderate)
- dalle_generate (expensive)
- code_execute (moderate)

Communicates over stdio using newline-delimited JSON-RPC 2.0.
"""

import json
import sys
import time
import random


TOOLS = [
    {
        "name": "web_search",
        "description": "Search the web for information",
        "inputSchema": {
            "type": "object",
            "properties": {
                "query": {"type": "string", "description": "Search query"}
            },
            "required": ["query"]
        }
    },
    {
        "name": "database_query",
        "description": "Execute a read-only database query",
        "inputSchema": {
            "type": "object",
            "properties": {
                "sql": {"type": "string", "description": "SQL query"}
            },
            "required": ["sql"]
        }
    },
    {
        "name": "gpt4_summarize",
        "description": "Summarize text using GPT-4",
        "inputSchema": {
            "type": "object",
            "properties": {
                "text": {"type": "string", "description": "Text to summarize"}
            },
            "required": ["text"]
        }
    },
    {
        "name": "dalle_generate",
        "description": "Generate an image with DALL-E",
        "inputSchema": {
            "type": "object",
            "properties": {
                "prompt": {"type": "string", "description": "Image description"}
            },
            "required": ["prompt"]
        }
    },
    {
        "name": "code_execute",
        "description": "Execute code in a sandboxed environment",
        "inputSchema": {
            "type": "object",
            "properties": {
                "language": {"type": "string", "description": "Programming language"},
                "code": {"type": "string", "description": "Code to execute"}
            },
            "required": ["language", "code"]
        }
    }
]


def handle_request(req):
    method = req.get("method", "")
    req_id = req.get("id")

    if method == "initialize":
        return {
            "jsonrpc": "2.0",
            "id": req_id,
            "result": {
                "protocolVersion": "2024-11-05",
                "capabilities": {"tools": {"listChanged": True}},
                "serverInfo": {"name": "mock-mcp-server", "version": "1.0.0"}
            }
        }

    if method == "notifications/initialized":
        return None  # notification, no response

    if method == "ping":
        return {"jsonrpc": "2.0", "id": req_id, "result": {}}

    if method == "tools/list":
        return {"jsonrpc": "2.0", "id": req_id, "result": {"tools": TOOLS}}

    if method == "tools/call":
        params = req.get("params", {})
        tool_name = params.get("name", "unknown")
        arguments = params.get("arguments", {})

        # Simulate some work
        time.sleep(random.uniform(0.05, 0.2))

        result_text = simulate_tool(tool_name, arguments)
        return {
            "jsonrpc": "2.0",
            "id": req_id,
            "result": {
                "content": [{"type": "text", "text": result_text}],
                "isError": False
            }
        }

    # Unknown method
    return {
        "jsonrpc": "2.0",
        "id": req_id,
        "error": {"code": -32601, "message": f"Method not found: {method}"}
    }


def simulate_tool(name, args):
    if name == "web_search":
        q = args.get("query", "")
        return f"Found 3 results for '{q}': [1] Example article... [2] Wikipedia... [3] Stack Overflow..."
    if name == "database_query":
        return "| id | name | status |\n| 1 | Agent Alpha | active |\n| 2 | Agent Beta | paused |"
    if name == "gpt4_summarize":
        text = args.get("text", "")[:50]
        return f"Summary: The text discusses {text}... (key points: efficiency, governance, cost control)"
    if name == "dalle_generate":
        return "Generated image: https://example.com/generated/img_abc123.png (1024x1024)"
    if name == "code_execute":
        lang = args.get("language", "python")
        return f"Execution complete ({lang}). Output: 42\nExit code: 0"
    return f"Tool '{name}' executed successfully"


def main():
    print(f"Mock MCP server starting...", file=sys.stderr)
    for line in sys.stdin:
        line = line.strip()
        if not line:
            continue
        try:
            req = json.loads(line)
        except json.JSONDecodeError:
            print(f"Invalid JSON: {line}", file=sys.stderr)
            continue

        resp = handle_request(req)
        if resp is not None:
            print(json.dumps(resp), flush=True)

    print("Mock MCP server shutting down", file=sys.stderr)


if __name__ == "__main__":
    main()
