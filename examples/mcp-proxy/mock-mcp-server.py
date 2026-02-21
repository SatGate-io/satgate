#!/usr/bin/env python3
"""Mock MCP server for SatGate MCP proxy — enterprise demo.

Simulates a realistic enterprise tool surface:
- LLM inference (multiple models/tiers)
- Code generation & analysis
- Data & search
- Image & media generation
- DevOps & infrastructure
- Communication & collaboration
- Security & compliance
- Document processing

31 tools across 8 categories with realistic cost profiles.
Communicates over stdio using newline-delimited JSON-RPC 2.0.
"""

import json
import sys
import time
import random


TOOLS = [
    # ── LLM Inference ───────────────────────────────────────────────────
    {
        "name": "gpt4_chat",
        "description": "Send a prompt to GPT-4 Turbo for completion",
        "inputSchema": {
            "type": "object",
            "properties": {
                "prompt": {"type": "string", "description": "User prompt"},
                "system": {"type": "string", "description": "System instruction"},
                "temperature": {"type": "number", "description": "Sampling temperature (0-2)"},
                "max_tokens": {"type": "integer", "description": "Max output tokens"}
            },
            "required": ["prompt"]
        }
    },
    {
        "name": "gpt4_summarize",
        "description": "Summarize long text using GPT-4",
        "inputSchema": {
            "type": "object",
            "properties": {
                "text": {"type": "string", "description": "Text to summarize"},
                "style": {"type": "string", "enum": ["brief", "detailed", "bullets"], "description": "Summary style"}
            },
            "required": ["text"]
        }
    },
    {
        "name": "claude_analyze",
        "description": "Deep analysis using Claude 3.5 Sonnet",
        "inputSchema": {
            "type": "object",
            "properties": {
                "content": {"type": "string", "description": "Content to analyze"},
                "task": {"type": "string", "description": "Analysis objective"}
            },
            "required": ["content"]
        }
    },
    {
        "name": "embeddings_generate",
        "description": "Generate vector embeddings for text (ada-002)",
        "inputSchema": {
            "type": "object",
            "properties": {
                "text": {"type": "string", "description": "Input text"},
                "model": {"type": "string", "description": "Embedding model"}
            },
            "required": ["text"]
        }
    },

    # ── Code Generation & Analysis ──────────────────────────────────────
    {
        "name": "code_generate",
        "description": "Generate code from a natural language specification",
        "inputSchema": {
            "type": "object",
            "properties": {
                "spec": {"type": "string", "description": "What to build"},
                "language": {"type": "string", "description": "Target language"},
                "framework": {"type": "string", "description": "Framework (e.g. React, FastAPI)"}
            },
            "required": ["spec", "language"]
        }
    },
    {
        "name": "code_review",
        "description": "AI-powered code review with security and quality checks",
        "inputSchema": {
            "type": "object",
            "properties": {
                "code": {"type": "string", "description": "Code to review"},
                "language": {"type": "string", "description": "Programming language"},
                "focus": {"type": "string", "enum": ["security", "performance", "quality", "all"]}
            },
            "required": ["code"]
        }
    },
    {
        "name": "code_execute",
        "description": "Execute code in a sandboxed environment",
        "inputSchema": {
            "type": "object",
            "properties": {
                "language": {"type": "string", "description": "Programming language"},
                "code": {"type": "string", "description": "Code to execute"},
                "timeout_seconds": {"type": "integer", "description": "Execution timeout"}
            },
            "required": ["language", "code"]
        }
    },
    {
        "name": "code_translate",
        "description": "Translate code from one language to another",
        "inputSchema": {
            "type": "object",
            "properties": {
                "code": {"type": "string", "description": "Source code"},
                "from_language": {"type": "string"},
                "to_language": {"type": "string"}
            },
            "required": ["code", "from_language", "to_language"]
        }
    },

    # ── Data & Search ───────────────────────────────────────────────────
    {
        "name": "web_search",
        "description": "Search the web for real-time information",
        "inputSchema": {
            "type": "object",
            "properties": {
                "query": {"type": "string", "description": "Search query"},
                "num_results": {"type": "integer", "description": "Number of results (1-10)"}
            },
            "required": ["query"]
        }
    },
    {
        "name": "database_query",
        "description": "Execute a read-only SQL query against the data warehouse",
        "inputSchema": {
            "type": "object",
            "properties": {
                "sql": {"type": "string", "description": "SQL query (read-only)"},
                "database": {"type": "string", "description": "Target database"}
            },
            "required": ["sql"]
        }
    },
    {
        "name": "vector_search",
        "description": "Semantic search across the knowledge base using embeddings",
        "inputSchema": {
            "type": "object",
            "properties": {
                "query": {"type": "string", "description": "Natural language query"},
                "collection": {"type": "string", "description": "Vector collection name"},
                "top_k": {"type": "integer", "description": "Number of results"}
            },
            "required": ["query"]
        }
    },
    {
        "name": "data_transform",
        "description": "Transform and aggregate data (CSV, JSON, Parquet)",
        "inputSchema": {
            "type": "object",
            "properties": {
                "input_url": {"type": "string", "description": "Data source URL or S3 path"},
                "operations": {"type": "string", "description": "Transform operations (filter, group, pivot)"},
                "output_format": {"type": "string", "enum": ["csv", "json", "parquet"]}
            },
            "required": ["input_url", "operations"]
        }
    },

    # ── Image & Media Generation ────────────────────────────────────────
    {
        "name": "dalle_generate",
        "description": "Generate images with DALL-E 3",
        "inputSchema": {
            "type": "object",
            "properties": {
                "prompt": {"type": "string", "description": "Image description"},
                "size": {"type": "string", "enum": ["1024x1024", "1792x1024", "1024x1792"]},
                "quality": {"type": "string", "enum": ["standard", "hd"]}
            },
            "required": ["prompt"]
        }
    },
    {
        "name": "image_analyze",
        "description": "Analyze an image using GPT-4 Vision",
        "inputSchema": {
            "type": "object",
            "properties": {
                "image_url": {"type": "string", "description": "URL of the image to analyze"},
                "question": {"type": "string", "description": "What to look for"}
            },
            "required": ["image_url"]
        }
    },
    {
        "name": "tts_generate",
        "description": "Convert text to speech audio",
        "inputSchema": {
            "type": "object",
            "properties": {
                "text": {"type": "string", "description": "Text to speak"},
                "voice": {"type": "string", "enum": ["alloy", "echo", "fable", "onyx", "nova", "shimmer"]},
                "speed": {"type": "number", "description": "Playback speed (0.25-4.0)"}
            },
            "required": ["text"]
        }
    },
    {
        "name": "whisper_transcribe",
        "description": "Transcribe audio to text using Whisper",
        "inputSchema": {
            "type": "object",
            "properties": {
                "audio_url": {"type": "string", "description": "URL of audio file"},
                "language": {"type": "string", "description": "Language hint (ISO 639-1)"}
            },
            "required": ["audio_url"]
        }
    },

    # ── DevOps & Infrastructure ─────────────────────────────────────────
    {
        "name": "k8s_query",
        "description": "Query Kubernetes cluster state (pods, services, deployments)",
        "inputSchema": {
            "type": "object",
            "properties": {
                "resource": {"type": "string", "description": "Resource type (pods, services, deployments)"},
                "namespace": {"type": "string", "description": "Kubernetes namespace"},
                "selector": {"type": "string", "description": "Label selector"}
            },
            "required": ["resource"]
        }
    },
    {
        "name": "terraform_plan",
        "description": "Generate and validate a Terraform plan",
        "inputSchema": {
            "type": "object",
            "properties": {
                "config": {"type": "string", "description": "Terraform HCL configuration"},
                "workspace": {"type": "string", "description": "Terraform workspace"}
            },
            "required": ["config"]
        }
    },
    {
        "name": "ci_trigger",
        "description": "Trigger a CI/CD pipeline run",
        "inputSchema": {
            "type": "object",
            "properties": {
                "pipeline": {"type": "string", "description": "Pipeline name"},
                "branch": {"type": "string", "description": "Git branch"},
                "params": {"type": "object", "description": "Pipeline parameters"}
            },
            "required": ["pipeline"]
        }
    },
    {
        "name": "log_search",
        "description": "Search application logs (Elasticsearch/CloudWatch)",
        "inputSchema": {
            "type": "object",
            "properties": {
                "query": {"type": "string", "description": "Search query"},
                "service": {"type": "string", "description": "Service name"},
                "time_range": {"type": "string", "description": "Time range (e.g. '1h', '24h', '7d')"},
                "severity": {"type": "string", "enum": ["debug", "info", "warn", "error", "fatal"]}
            },
            "required": ["query"]
        }
    },

    # ── Communication & Collaboration ───────────────────────────────────
    {
        "name": "slack_send",
        "description": "Send a message to a Slack channel",
        "inputSchema": {
            "type": "object",
            "properties": {
                "channel": {"type": "string", "description": "Channel name or ID"},
                "message": {"type": "string", "description": "Message text (supports Markdown)"},
                "thread_ts": {"type": "string", "description": "Thread timestamp for replies"}
            },
            "required": ["channel", "message"]
        }
    },
    {
        "name": "email_send",
        "description": "Send an email via corporate SMTP",
        "inputSchema": {
            "type": "object",
            "properties": {
                "to": {"type": "string", "description": "Recipient email"},
                "subject": {"type": "string", "description": "Email subject"},
                "body": {"type": "string", "description": "Email body (HTML supported)"},
                "cc": {"type": "string", "description": "CC recipients"}
            },
            "required": ["to", "subject", "body"]
        }
    },
    {
        "name": "jira_create",
        "description": "Create a Jira ticket",
        "inputSchema": {
            "type": "object",
            "properties": {
                "project": {"type": "string", "description": "Project key"},
                "summary": {"type": "string", "description": "Ticket title"},
                "description": {"type": "string", "description": "Ticket description"},
                "type": {"type": "string", "enum": ["bug", "story", "task", "epic"]},
                "priority": {"type": "string", "enum": ["critical", "high", "medium", "low"]}
            },
            "required": ["project", "summary", "type"]
        }
    },
    {
        "name": "calendar_check",
        "description": "Check calendar availability and schedule meetings",
        "inputSchema": {
            "type": "object",
            "properties": {
                "action": {"type": "string", "enum": ["check", "schedule", "cancel"]},
                "attendees": {"type": "string", "description": "Comma-separated email addresses"},
                "date": {"type": "string", "description": "Date (YYYY-MM-DD)"},
                "duration_minutes": {"type": "integer", "description": "Meeting duration"}
            },
            "required": ["action"]
        }
    },

    # ── Security & Compliance ───────────────────────────────────────────
    {
        "name": "vulnerability_scan",
        "description": "Scan a codebase or container image for vulnerabilities",
        "inputSchema": {
            "type": "object",
            "properties": {
                "target": {"type": "string", "description": "Repository URL or container image"},
                "scan_type": {"type": "string", "enum": ["sast", "dast", "container", "dependency"]},
                "severity_threshold": {"type": "string", "enum": ["critical", "high", "medium", "low"]}
            },
            "required": ["target", "scan_type"]
        }
    },
    {
        "name": "secrets_read",
        "description": "Read a secret from the vault (HashiCorp Vault / AWS Secrets Manager)",
        "inputSchema": {
            "type": "object",
            "properties": {
                "path": {"type": "string", "description": "Secret path"},
                "version": {"type": "integer", "description": "Secret version (latest if omitted)"}
            },
            "required": ["path"]
        }
    },
    {
        "name": "policy_check",
        "description": "Validate a configuration against OPA/Rego policies",
        "inputSchema": {
            "type": "object",
            "properties": {
                "config": {"type": "string", "description": "Configuration to validate (JSON/YAML)"},
                "policy_set": {"type": "string", "description": "Policy set name"}
            },
            "required": ["config"]
        }
    },

    # ── Source Control ───────────────────────────────────────────────────
    {
        "name": "github_pr",
        "description": "Create or review a GitHub pull request",
        "inputSchema": {
            "type": "object",
            "properties": {
                "action": {"type": "string", "enum": ["create", "review", "merge", "list"]},
                "repo": {"type": "string", "description": "Repository (owner/repo)"},
                "title": {"type": "string", "description": "PR title (for create)"},
                "branch": {"type": "string", "description": "Source branch"}
            },
            "required": ["action", "repo"]
        }
    },

    # ── Document Processing ─────────────────────────────────────────────
    {
        "name": "pdf_extract",
        "description": "Extract text and tables from a PDF document",
        "inputSchema": {
            "type": "object",
            "properties": {
                "url": {"type": "string", "description": "PDF URL or S3 path"},
                "pages": {"type": "string", "description": "Page range (e.g. '1-5', 'all')"},
                "extract_tables": {"type": "boolean", "description": "Extract tables as structured data"}
            },
            "required": ["url"]
        }
    },
    {
        "name": "spreadsheet_analyze",
        "description": "Analyze an Excel/CSV spreadsheet with natural language queries",
        "inputSchema": {
            "type": "object",
            "properties": {
                "url": {"type": "string", "description": "Spreadsheet URL or S3 path"},
                "question": {"type": "string", "description": "Analysis question in natural language"},
                "sheet": {"type": "string", "description": "Sheet name (for multi-sheet workbooks)"}
            },
            "required": ["url", "question"]
        }
    },
    {
        "name": "doc_generate",
        "description": "Generate a formatted document (PDF, DOCX, Markdown)",
        "inputSchema": {
            "type": "object",
            "properties": {
                "content": {"type": "string", "description": "Document content (Markdown)"},
                "format": {"type": "string", "enum": ["pdf", "docx", "markdown", "html"]},
                "template": {"type": "string", "description": "Template name (optional)"}
            },
            "required": ["content", "format"]
        }
    },
]


TOOL_RESPONSES = {
    "gpt4_chat": lambda args: f"GPT-4 response: Based on your query about '{args.get('prompt', '')[:60]}...', here's my analysis: The key considerations are scalability, security posture, and cost optimization across the deployment pipeline.",
    "gpt4_summarize": lambda args: f"Summary: The document covers {args.get('text', '')[:40]}... Key points: (1) operational efficiency improved 23%, (2) security incidents reduced by 41%, (3) recommended next steps include automated remediation.",
    "claude_analyze": lambda args: f"Analysis complete. Task: {args.get('task', 'general analysis')}. Findings: 3 critical insights identified. Confidence: 94%. The primary pattern suggests a shift toward distributed architecture with centralized governance.",
    "embeddings_generate": lambda args: f"Generated 1536-dimensional embedding for input text ({len(args.get('text', ''))} chars). Model: text-embedding-ada-002. Usage: 42 tokens.",
    "code_generate": lambda args: f"Generated {args.get('language', 'Python')} code for: {args.get('spec', '')[:50]}...\n\n```{args.get('language', 'python').lower()}\n# Auto-generated by SatGate AI\ndef main():\n    print('Implementation ready')\n```\n\nFiles: 1 | Lines: 47 | Tests: 12 generated",
    "code_review": lambda args: f"Code review complete. Focus: {args.get('focus', 'all')}.\n⚠️ 2 issues found:\n  1. [MEDIUM] SQL injection risk at line 23\n  2. [LOW] Unused import 'os'\n✅ 14 checks passed. Overall: B+",
    "code_execute": lambda args: f"Execution complete ({args.get('language', 'python')}).\nOutput: 42\nRuntime: 0.34s | Memory: 12MB | Exit code: 0",
    "code_translate": lambda args: f"Translated from {args.get('from_language', '?')} to {args.get('to_language', '?')}. Lines: 84 → 91. Confidence: 97%. 2 manual review points flagged.",
    "web_search": lambda args: f"Found 5 results for '{args.get('query', '')}':\n[1] Enterprise AI Governance — Gartner Report 2026\n[2] MCP Protocol Specification — modelcontextprotocol.io\n[3] API Cost Management Best Practices — AWS Blog\n[4] Agent Economy Forecast — McKinsey Digital\n[5] Zero Trust for AI Agents — Deloitte Insights",
    "database_query": lambda args: "Query executed (read-only). 847 rows returned.\n| department | agent_count | monthly_spend | budget_util |\n| Engineering | 12 | $4,230 | 78% |\n| Marketing | 8 | $2,890 | 92% |\n| Research | 5 | $6,120 | 61% |",
    "vector_search": lambda args: f"Semantic search: '{args.get('query', '')[:40]}...'\nTop 3 matches (collection: {args.get('collection', 'default')}):\n  1. [0.94] Internal security policy v2.3\n  2. [0.89] Incident response playbook\n  3. [0.85] Agent onboarding checklist",
    "data_transform": lambda args: f"Transform complete. Input: {args.get('input_url', 'data.csv')} → Output: {args.get('output_format', 'json')}. Rows processed: 12,847. Operations applied: {args.get('operations', 'filter, aggregate')}.",
    "dalle_generate": lambda args: f"Generated image: https://cdn.example.com/generated/img_{random.randint(10000,99999)}.png\nPrompt: {args.get('prompt', '')[:50]}...\nSize: {args.get('size', '1024x1024')} | Quality: {args.get('quality', 'standard')}",
    "image_analyze": lambda args: f"Image analysis complete.\nObjects detected: 4 (laptop, whiteboard, person, coffee mug)\nText found: 'Q3 Revenue Target'\nScene: Office/meeting room setting\nConfidence: 91%",
    "tts_generate": lambda args: f"Audio generated. Duration: {len(args.get('text', '')) * 0.06:.1f}s | Voice: {args.get('voice', 'nova')} | Format: mp3 | Size: {len(args.get('text', '')) * 12}B",
    "whisper_transcribe": lambda args: f"Transcription complete.\nLanguage: {args.get('language', 'en')} (confidence: 99%)\nDuration: 3m 42s | Words: 487\nText: 'Welcome to the quarterly security review. Today we'll cover three main topics...'",
    "k8s_query": lambda args: f"Namespace: {args.get('namespace', 'default')} | Resource: {args.get('resource', 'pods')}\n  agent-worker-7f8b9 — Running (cpu: 120m, mem: 256Mi)\n  agent-worker-3c4d2 — Running (cpu: 95m, mem: 198Mi)\n  gateway-proxy-a1b2c — Running (cpu: 340m, mem: 512Mi)\n3 resources found.",
    "terraform_plan": lambda args: f"Terraform plan generated.\nWorkspace: {args.get('workspace', 'default')}\n+ 3 to add, ~ 1 to change, - 0 to destroy\n  + aws_lambda_function.agent_handler\n  + aws_iam_role.agent_exec\n  + aws_cloudwatch_log_group.agent_logs\n  ~ aws_api_gateway_rest_api.main (tags updated)\nEstimated cost: +$12.40/mo",
    "ci_trigger": lambda args: f"Pipeline '{args.get('pipeline', 'default')}' triggered.\nBranch: {args.get('branch', 'main')}\nRun ID: #2847\nStatus: queued → running\nEstimated duration: 4m 30s",
    "log_search": lambda args: f"Found 23 entries matching '{args.get('query', '')}' in {args.get('service', 'all services')} ({args.get('time_range', '1h')}):\n  [ERROR] 09:41:12 — Budget exceeded for token tk_8f2a... (agent: research-bot)\n  [WARN]  09:38:45 — Rate limit approaching (82%) for /api/v1/chat\n  [ERROR] 09:22:03 — Connection timeout to upstream model provider",
    "slack_send": lambda args: f"Message sent to #{args.get('channel', 'general')}.\nTimestamp: {time.time():.0f}\nDelivered to 24 channel members.",
    "email_send": lambda args: f"Email sent to {args.get('to', 'recipient')}.\nSubject: {args.get('subject', '(no subject)')}\nMessage-ID: <{random.randint(100000, 999999)}@corp.example.com>\nStatus: delivered",
    "jira_create": lambda args: f"Created {args.get('project', 'PROJ')}-{random.randint(1000, 9999)}: {args.get('summary', 'New ticket')}\nType: {args.get('type', 'task')} | Priority: {args.get('priority', 'medium')}\nAssignee: auto-assigned via round-robin\nURL: https://jira.example.com/browse/{args.get('project', 'PROJ')}-{random.randint(1000, 9999)}",
    "calendar_check": lambda args: f"Calendar {args.get('action', 'check')}: {args.get('date', 'today')}\nAvailable slots: 10:00-10:30, 14:00-15:00, 16:30-17:00\nAttendees checked: {len(args.get('attendees', '').split(','))} people\nBest slot: 14:00-15:00 (all available)",
    "vulnerability_scan": lambda args: f"Scan complete: {args.get('target', 'unknown')}\nType: {args.get('scan_type', 'dependency')}\nFindings: 2 critical, 5 high, 12 medium, 31 low\nCritical: CVE-2026-1234 (jsonwebtoken < 9.0.3), CVE-2026-5678 (lodash prototype pollution)\nRecommendation: Upgrade dependencies immediately.",
    "secrets_read": lambda args: f"Secret retrieved: {args.get('path', '/unknown')}\nVersion: {args.get('version', 'latest')}\nType: kv-v2\nLast rotated: 2026-02-14T08:00:00Z\nValue: [REDACTED — 32 bytes]",
    "policy_check": lambda args: f"Policy validation: {args.get('policy_set', 'default')}\n✅ 18 rules passed\n⚠️ 2 warnings (non-blocking)\n❌ 1 violation: 'require_encryption_at_rest' — S3 bucket missing server-side encryption\nOverall: FAIL (1 violation must be resolved)",
    "github_pr": lambda args: f"PR {args.get('action', 'list')} on {args.get('repo', 'org/repo')}.\nBranch: {args.get('branch', 'feature/update')}\n#2847 — '{args.get('title', 'Update dependencies')}'\nStatus: open | Reviews: 1/2 approved | CI: passing\nURL: https://github.com/{args.get('repo', 'org/repo')}/pull/2847",
    "pdf_extract": lambda args: f"Extracted from: {args.get('url', 'document.pdf')}\nPages: {args.get('pages', 'all')} | Total: 24 pages\nText: 12,847 words extracted\nTables: 3 tables found (exported as structured data)\nImages: 7 embedded images detected",
    "spreadsheet_analyze": lambda args: f"Analysis: {args.get('question', 'summary')}\nSource: {args.get('url', 'data.xlsx')}\nAnswer: Total Q4 revenue was $2.4M across 3 regions. APAC grew 34% QoQ while EMEA declined 8%. Top contributor: Enterprise segment ($1.1M, 46% of total).",
    "doc_generate": lambda args: f"Document generated.\nFormat: {args.get('format', 'pdf')}\nPages: 4 | Size: 847KB\nTemplate: {args.get('template', 'corporate-default')}\nURL: https://docs.example.com/generated/doc_{random.randint(10000,99999)}.{args.get('format', 'pdf')}",
}


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
                "serverInfo": {"name": "enterprise-tool-suite", "version": "2.0.0"}
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

        # Simulate realistic latency by tool type
        latency = {
            "gpt4_chat": (0.5, 2.0), "gpt4_summarize": (0.3, 1.5),
            "claude_analyze": (0.5, 2.5), "dalle_generate": (2.0, 8.0),
            "code_generate": (0.5, 3.0), "code_execute": (0.2, 1.0),
            "vulnerability_scan": (1.0, 5.0), "terraform_plan": (1.0, 4.0),
        }.get(tool_name, (0.05, 0.3))
        time.sleep(random.uniform(*latency))

        handler = TOOL_RESPONSES.get(tool_name)
        if handler:
            result_text = handler(arguments)
        else:
            result_text = f"Tool '{tool_name}' executed successfully with args: {json.dumps(arguments)[:100]}"

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


def main():
    print("Enterprise MCP server starting (31 tools)...", file=sys.stderr)
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

    print("Enterprise MCP server shutting down", file=sys.stderr)


if __name__ == "__main__":
    main()
