package mcp

import (
	"net/http"
	"strings"

	"github.com/rs/zerolog/log"
)

// Middleware returns an HTTP middleware that inspects the request body for
// MCP (JSON-RPC 2.0) traffic and stores parsed metadata in the request
// context.
//
// The middleware:
//   - Only activates for POST requests with a JSON content type.
//   - Respects maxBodySize to avoid buffering huge payloads.
//   - Replays the body so the upstream proxy can read it unmodified.
//   - Falls through transparently for non-MCP traffic.
//
// Per-tool cost attribution is available in SatGate Enterprise.
func Middleware(maxBodySize int64) func(http.Handler) http.Handler {
	if maxBodySize <= 0 {
		maxBodySize = DefaultMaxBodySize
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only inspect POST with JSON content type.
			if r.Method != http.MethodPost || !isJSONContent(r) {
				next.ServeHTTP(w, r)
				return
			}

			info := Parse(r, maxBodySize)
			if info == nil {
				next.ServeHTTP(w, r)
				return
			}

			// Log the parsed MCP request.
			log.Debug().
				Str("mcp.method", info.Method).
				Str("mcp.tool", info.ToolName).
				Bool("mcp.batch", info.IsBatch).
				Msg("MCP request detected")

			// Stash in context for downstream handlers and telemetry.
			ctx := WithInfo(r.Context(), info)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// isJSONContent returns true if the request Content-Type indicates JSON.
func isJSONContent(r *http.Request) bool {
	ct := r.Header.Get("Content-Type")
	return strings.HasPrefix(ct, "application/json")
}
