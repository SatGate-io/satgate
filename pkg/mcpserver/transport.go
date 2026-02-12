package mcpserver

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/rs/zerolog/log"
)

// Transport reads JSON-RPC messages from a stream and writes responses back.
type Transport interface {
	// ReadMessage blocks until a message is available, then returns it.
	ReadMessage(ctx context.Context) (json.RawMessage, error)

	// WriteMessage sends a JSON-RPC message.
	WriteMessage(ctx context.Context, msg json.RawMessage) error

	// Close shuts down the transport.
	Close() error
}

// --- stdio transport ---

// StdioTransport implements MCP over stdio (newline-delimited JSON).
type StdioTransport struct {
	reader  *bufio.Scanner
	writer  io.Writer
	writeMu sync.Mutex
	closer  io.Closer
}

// NewStdioTransport creates a transport over stdin/stdout (or any reader/writer pair).
func NewStdioTransport(r io.Reader, w io.Writer, closer io.Closer) *StdioTransport {
	scanner := bufio.NewScanner(r)
	// MCP messages can be large (tool results with full documents)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024) // 10MB max

	return &StdioTransport{
		reader: scanner,
		writer: w,
		closer: closer,
	}
}

func (t *StdioTransport) ReadMessage(ctx context.Context) (json.RawMessage, error) {
	// Scanner.Scan is blocking — we use a channel to respect context cancellation
	type result struct {
		data []byte
		err  error
	}
	ch := make(chan result, 1)

	go func() {
		if t.reader.Scan() {
			line := t.reader.Bytes()
			// Make a copy since scanner reuses the buffer
			cpy := make([]byte, len(line))
			copy(cpy, line)
			ch <- result{data: cpy}
		} else {
			err := t.reader.Err()
			if err == nil {
				err = io.EOF
			}
			ch <- result{err: err}
		}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case r := <-ch:
		if r.err != nil {
			return nil, r.err
		}
		return json.RawMessage(r.data), nil
	}
}

func (t *StdioTransport) WriteMessage(_ context.Context, msg json.RawMessage) error {
	t.writeMu.Lock()
	defer t.writeMu.Unlock()

	// Write message followed by newline (NDJSON)
	if _, err := t.writer.Write(msg); err != nil {
		return err
	}
	_, err := t.writer.Write([]byte("\n"))
	return err
}

func (t *StdioTransport) Close() error {
	if t.closer != nil {
		return t.closer.Close()
	}
	return nil
}

// --- Upstream stdio transport (manages a subprocess) ---

// ProcessTransport wraps a subprocess stdin/stdout as a Transport.
type ProcessTransport struct {
	*StdioTransport
	cmd interface{ Wait() error } // *exec.Cmd
}

// NewProcessTransport creates a transport connected to a subprocess.
func NewProcessTransport(stdin io.Writer, stdout io.Reader, closer io.Closer, cmd interface{ Wait() error }) *ProcessTransport {
	return &ProcessTransport{
		StdioTransport: NewStdioTransport(stdout, stdin, closer),
		cmd:            cmd,
	}
}

func (t *ProcessTransport) Close() error {
	err := t.StdioTransport.Close()
	if t.cmd != nil {
		if waitErr := t.cmd.Wait(); waitErr != nil {
			log.Debug().Err(waitErr).Msg("upstream process exited")
			if err == nil {
				err = waitErr
			}
		}
	}
	return err
}

// --- Message routing helper ---

// ParseMessage attempts to parse raw JSON into a Request or Response.
// Returns (request, nil) for requests, (nil, response) for responses.
func ParseMessage(data json.RawMessage) (*Request, *Response, error) {
	// Quick check: does it have a "method" field? → request
	// Does it have "result" or "error"? → response
	var probe struct {
		Method string          `json:"method"`
		Result json.RawMessage `json:"result"`
		Error  json.RawMessage `json:"error"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return nil, nil, fmt.Errorf("invalid JSON-RPC message: %w", err)
	}

	if probe.Method != "" {
		var req Request
		if err := json.Unmarshal(data, &req); err != nil {
			return nil, nil, err
		}
		return &req, nil, nil
	}

	if probe.Result != nil || probe.Error != nil {
		var resp Response
		if err := json.Unmarshal(data, &resp); err != nil {
			return nil, nil, err
		}
		return nil, &resp, nil
	}

	return nil, nil, fmt.Errorf("message is neither request nor response")
}
