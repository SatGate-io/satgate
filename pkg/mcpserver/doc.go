// Package mcpserver implements an MCP (Model Context Protocol) proxy gateway.
//
// It sits between an MCP client (e.g., Claude Code, Agent Zero, Cursor) and
// one or more upstream MCP servers, providing:
//
//   - Transparent relay of all MCP JSON-RPC methods (tools/list, tools/call, etc.)
//   - Per-tool cost attribution and budget enforcement
//   - Request/response correlation with cancellation propagation
//   - stdio and SSE transport support (client-facing); stdio upstream (HTTP upstream planned)
//
// The OSS version provides in-memory budget enforcement with a single budget counter.
// SatGate Enterprise adds Redis-backed budgets, delegation hierarchies, spend ledgers,
// multi-tenant policy, and dashboard integration.
//
// Architecture:
//
//	Agent → [stdio/SSE] → SatGate MCP Proxy → [stdio] → Upstream MCP Server(s)
//	                            │
//	                     BudgetEnforcer
//	                     (in-memory / Redis)
//
// Quick start:
//
//	proxy, err := mcpserver.New(mcpserver.Config{...})
//	if err != nil { log.Fatal(err) }
//	proxy.Run(ctx) // blocks until context cancelled
package mcpserver
