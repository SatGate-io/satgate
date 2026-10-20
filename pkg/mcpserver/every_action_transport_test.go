package mcpserver

// The master table, carried by the two real transports: Streamable HTTP
// (POST /mcp) and SSE (POST /message). The agent's key travels in the
// Authorization header, as it does in production, and not in params.

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// eaWire is a JSON-RPC request as it goes on the wire. Params is sent as given,
// so a test can send one that is not an object.
type eaWire struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// eaWithoutToken removes params._meta.token from an object when it is the
// environment's own key, so the key reaches the proxy only by the header.
// Anything else is returned as it is.
func eaWithoutToken(params json.RawMessage, token string) json.RawMessage {
	var m map[string]json.RawMessage
	if json.Unmarshal(params, &m) != nil || m == nil {
		return params
	}
	var meta map[string]json.RawMessage
	if raw, ok := m["_meta"]; ok && json.Unmarshal(raw, &meta) == nil {
		if string(meta["token"]) == `"`+token+`"` {
			delete(meta, "token")
		}
		if len(meta) == 0 {
			delete(m, "_meta")
		} else {
			m["_meta"], _ = json.Marshal(meta)
		}
	}
	out, _ := json.Marshal(m)
	return out
}

func eaWireOf(req *Request, token string) eaWire {
	return eaWire{JSONRPC: "2.0", ID: req.ID, Method: req.Method, Params: eaWithoutToken(req.Params, token)}
}

// eaViaStreamable sends each request of e through POST /mcp.
func eaViaStreamable(t *testing.T, e *eaEnv) func() {
	t.Helper()
	srv := httptest.NewServer(NewSSEServer(e.proxy, "").Handler())
	sid := mustInitSession(t, srv.URL, e.token)
	e.sender = func(req *Request) *Response {
		t.Helper()
		status, _, body := streamablePost(t, srv.URL+"/mcp", sid, e.token, eaWireOf(req, e.token))
		if status != http.StatusOK {
			t.Fatalf("POST /mcp %s: status %d body %s", req.Method, status, body)
		}
		var resp Response
		if err := json.Unmarshal([]byte(body), &resp); err != nil {
			t.Fatalf("POST /mcp %s: answer %q: %v", req.Method, body, err)
		}
		return &resp
	}
	return srv.Close
}

// eaSSEConn is an open SSE stream and the messages that come down it.
type eaSSEConn struct {
	t        *testing.T
	baseURL  string
	token    string
	session  string
	messages chan json.RawMessage
	body     interface{ Close() error }
}

func eaOpenSSE(t *testing.T, baseURL, token string) *eaSSEConn {
	t.Helper()
	req, _ := http.NewRequest("GET", baseURL+"/sse", nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("SSE connect: %v", err)
	}
	c := &eaSSEConn{t: t, baseURL: baseURL, token: token, messages: make(chan json.RawMessage, 16), body: resp.Body}
	rd := bufio.NewReader(resp.Body)
	for {
		line, err := rd.ReadString('\n')
		if err != nil {
			t.Fatalf("SSE endpoint event: %v", err)
		}
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "data: /message?sessionId=") {
			c.session = strings.TrimPrefix(line, "data: /message?sessionId=")
			break
		}
	}
	go func() {
		for {
			line, err := rd.ReadString('\n')
			if err != nil {
				return
			}
			if line = strings.TrimSpace(line); strings.HasPrefix(line, "data: {") {
				c.messages <- json.RawMessage(strings.TrimPrefix(line, "data: "))
			}
		}
	}()
	return c
}

// post sends a raw body to POST /message and returns the HTTP status.
func (c *eaSSEConn) post(body []byte) int {
	c.t.Helper()
	req, _ := http.NewRequest("POST", c.baseURL+"/message?sessionId="+c.session, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.t.Fatalf("POST /message: %v", err)
	}
	resp.Body.Close()
	return resp.StatusCode
}

// next waits for the next message on the stream.
func (c *eaSSEConn) next() *Response {
	c.t.Helper()
	select {
	case raw := <-c.messages:
		var resp Response
		if err := json.Unmarshal(raw, &resp); err != nil {
			c.t.Fatalf("SSE message %q: %v", raw, err)
		}
		return &resp
	case <-time.After(5 * time.Second):
		c.t.Fatal("timeout waiting for the SSE message")
		return nil
	}
}

// eaViaSSE sends each request of e through POST /message.
func eaViaSSE(t *testing.T, e *eaEnv) func() {
	t.Helper()
	srv := httptest.NewServer(NewSSEServer(e.proxy, "").Handler())
	conn := eaOpenSSE(t, srv.URL, e.token)
	e.sender = func(req *Request) *Response {
		t.Helper()
		body, _ := json.Marshal(eaWireOf(req, e.token))
		if status := conn.post(body); status != http.StatusAccepted {
			t.Fatalf("POST /message %s: status %d", req.Method, status)
		}
		return conn.next()
	}
	return func() { conn.body.Close(); srv.Close() }
}

var eaTransports = []struct {
	name string
	via  func(*testing.T, *eaEnv) func()
}{
	{"streamable_http", eaViaStreamable},
	{"sse", eaViaSSE},
}
