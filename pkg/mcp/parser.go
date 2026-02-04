package mcp

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"sync"
)

// DefaultMaxBodySize is the default maximum request body size (1 MB)
// that the parser will buffer for MCP detection.
const DefaultMaxBodySize int64 = 1 << 20

// bufPool reuses byte buffers to reduce GC pressure on high-traffic gateways.
var bufPool = sync.Pool{
	New: func() interface{} {
		b := make([]byte, 0, 4096)
		return &b
	},
}

// Parse reads up to maxSize bytes from the request body, attempts to parse
// the content as a JSON-RPC 2.0 MCP request, and returns the parsed info.
//
// On success or failure the request body is always replayed so the upstream
// proxy handler can read it again.
//
// Returns nil (not an error) when the body is not valid MCP traffic — the
// caller should treat it as a pass-through.
func Parse(r *http.Request, maxSize int64) *RequestInfo {
	if r.Body == nil || r.Body == http.NoBody {
		return nil
	}

	if maxSize <= 0 {
		maxSize = DefaultMaxBodySize
	}

	// Read up to maxSize+1 bytes. If we get more than maxSize we know the
	// body exceeds the limit and we skip parsing.
	bufp := bufPool.Get().(*[]byte)
	buf := (*bufp)[:0]
	defer func() {
		// Return buffer to pool if it hasn't grown too large.
		if cap(buf) <= 1<<20 {
			*bufp = buf
			bufPool.Put(bufp)
		}
	}()

	limited := io.LimitReader(r.Body, maxSize+1)
	tmp := bytes.NewBuffer(buf)
	n, _ := io.Copy(tmp, limited)
	buf = tmp.Bytes()

	// Replay body: prepend what we read back onto the remaining body.
	r.Body = io.NopCloser(io.MultiReader(bytes.NewReader(buf), r.Body))

	// Body exceeds limit — skip parsing, let it pass through.
	if n > maxSize {
		return nil
	}

	// Quick sniff: must start with '{' or '[' (ignoring leading whitespace).
	trimmed := bytes.TrimLeft(buf, " \t\r\n")
	if len(trimmed) == 0 {
		return nil
	}

	switch trimmed[0] {
	case '{':
		return parseSingle(buf)
	case '[':
		return parseBatch(buf)
	default:
		return nil
	}
}

// parseSingle parses a single JSON-RPC request.
func parseSingle(data []byte) *RequestInfo {
	var req JSONRPCRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil
	}
	if req.JSONRPC != "2.0" || req.Method == "" {
		return nil
	}
	info := &RequestInfo{
		Method: req.Method,
	}
	switch req.Method {
	case MethodToolsCall:
		info.ToolName = req.Params.Name
	case MethodResourceRead:
		info.ResourceURI = req.Params.URI
	}
	return info
}

// parseBatch parses a JSON-RPC batch (array of requests).
// It returns info for the first tools/call in the batch, or the first request
// if no tools/call is found.
func parseBatch(data []byte) *RequestInfo {
	var reqs []JSONRPCRequest
	if err := json.Unmarshal(data, &reqs); err != nil {
		return nil
	}
	if len(reqs) == 0 {
		return nil
	}

	// Find first tools/call, fall back to first request.
	var best *RequestInfo
	for i := range reqs {
		r := &reqs[i]
		if r.JSONRPC != "2.0" || r.Method == "" {
			continue
		}
		info := &RequestInfo{
			Method:    r.Method,
			IsBatch:   true,
			BatchSize: len(reqs),
		}
		if r.Method == MethodToolsCall {
			info.ToolName = r.Params.Name
			return info
		}
		if best == nil {
			best = info
		}
	}
	return best
}
