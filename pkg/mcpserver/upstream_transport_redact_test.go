package mcpserver

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const echoMarker = "ECHOED-BY-UPSTREAM-5d27"

// rawReply writes bytes to the client verbatim after reading its request head,
// so the HTTP client's own parser sees whatever the test wants.
type rawReply func(w *bufio.Writer)

func rawUpstream(t *testing.T, reply rawReply) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				br := bufio.NewReader(c)
				for {
					line, err := br.ReadString('\n')
					if err != nil || line == "\r\n" {
						break
					}
				}
				bw := bufio.NewWriter(c)
				reply(bw)
				bw.Flush()
			}(conn)
		}
	}()
	return "http://" + ln.Addr().String()
}

var malformedReplies = map[string]rawReply{
	"malformed status line": func(w *bufio.Writer) {
		fmt.Fprintf(w, "HTTP/1.1 %s nonsense\r\n\r\n", echoMarker)
	},
	"malformed HTTP version": func(w *bufio.Writer) {
		fmt.Fprintf(w, "HTTP/9 200 %s\r\n\r\n", echoMarker)
	},
	"malformed header line": func(w *bufio.Writer) {
		fmt.Fprintf(w, "HTTP/1.1 200 OK\r\nstatus 401 %s\r\n\r\n", echoMarker)
	},
	"malformed header with 401 text": func(w *bufio.Writer) {
		fmt.Fprintf(w, "HTTP/1.1 200 OK\r\nBad(Name): %s status 401\r\n\r\n", echoMarker)
	},
	"bad content length": func(w *bufio.Writer) {
		fmt.Fprintf(w, "HTTP/1.1 200 OK\r\nContent-Length: %s\r\n\r\n", echoMarker)
	},
}

func streamableWriteErr(t *testing.T, url string, redact bool) error {
	t.Helper()
	tr := NewStreamableHTTPTransport(url, map[string]string{"X-Stored": "stored-header-value"}, false, true)
	tr.SetRedactUpstreamText(redact)
	defer tr.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return tr.WriteMessage(ctx, []byte(`{"jsonrpc":"2.0","id":1,"method":"ping"}`))
}

// With redaction on, a transport or protocol failure names no byte the
// upstream sent: not a parser error, not a status line, not a header line.
func TestRedactTransportFailureCarriesNoUpstreamBytes(t *testing.T) {
	for name, reply := range malformedReplies {
		t.Run(name, func(t *testing.T) {
			url := rawUpstream(t, reply)
			err := streamableWriteErr(t, url, true)
			if err == nil {
				t.Fatal("expected an error")
			}
			if strings.Contains(err.Error(), echoMarker) || strings.Contains(err.Error(), "status 401") {
				t.Fatalf("upstream bytes in redacted error: %v", err)
			}
			if _, ok := UpstreamHTTPStatus(err); ok {
				t.Fatalf("a malformed reply must not carry an HTTP status: %v", err)
			}
		})
	}
}

// Default off: the messages are the ones before this change (the parser text
// is still there).
func TestRedactOffKeepsTransportErrorText(t *testing.T) {
	url := rawUpstream(t, malformedReplies["malformed status line"])
	err := streamableWriteErr(t, url, false)
	if err == nil || !strings.Contains(err.Error(), echoMarker) {
		t.Fatalf("default behaviour changed: %v", err)
	}
}

// A dial failure and a cancelled request keep their kind, with a fixed text.
func TestRedactTransportErrorKinds(t *testing.T) {
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	addr := ln.Addr().String()
	ln.Close() // nothing listens here now
	err := streamableWriteErr(t, "http://"+addr+"/secret-path?key=abc", true)
	if err == nil || strings.Contains(err.Error(), "secret-path") || strings.Contains(err.Error(), "key=abc") {
		t.Fatalf("dial error: %v", err)
	}
	if !strings.Contains(err.Error(), "could not connect to upstream") {
		t.Fatalf("dial error lost its kind: %v", err)
	}

	tr := NewStreamableHTTPTransport("http://127.0.0.1:1", nil, false, true)
	tr.SetRedactUpstreamText(true)
	defer tr.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err = tr.WriteMessage(ctx, []byte(`{}`))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("a canceled request must still match context.Canceled: %v", err)
	}

	// A private address is refused by the dialer; the text is fixed.
	tr2 := NewStreamableHTTPTransport("http://127.0.0.1:9", nil, false, false)
	tr2.SetRedactUpstreamText(true)
	defer tr2.Close()
	err = tr2.WriteMessage(context.Background(), []byte(`{}`))
	if err == nil || !strings.Contains(err.Error(), "not allowed") {
		t.Fatalf("SSRF refusal: %v", err)
	}
}

// A chunked body that breaks off names nothing the upstream sent.
func TestRedactBrokenBodyCarriesNoUpstreamBytes(t *testing.T) {
	url := rawUpstream(t, func(w *bufio.Writer) {
		fmt.Fprintf(w, "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nTransfer-Encoding: chunked\r\n\r\n%s\r\n", echoMarker)
	})
	err := streamableWriteErr(t, url, true)
	if err != nil && strings.Contains(err.Error(), echoMarker) {
		t.Fatalf("chunk framing text in redacted error: %v", err)
	}
}

// --- typed HTTP status ------------------------------------------------------

func TestUpstreamStatusErrorIsTypedForRealStatusesOnly(t *testing.T) {
	cases := []struct {
		name   string
		handle http.HandlerFunc
		want   int
	}{
		{"401", func(w http.ResponseWriter, r *http.Request) { http.Error(w, "no", 401) }, 401},
		{"403", func(w http.ResponseWriter, r *http.Request) { http.Error(w, "no", 403) }, 403},
		{"500 whose body says status 401", func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "status 401 "+echoMarker, 500)
		}, 500},
	}
	for _, c := range cases {
		for _, redact := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/redact=%v", c.name, redact), func(t *testing.T) {
				srv := httptest.NewServer(c.handle)
				defer srv.Close()
				err := streamableWriteErr(t, srv.URL, redact)
				got, ok := UpstreamHTTPStatus(err)
				if !ok || got != c.want {
					t.Fatalf("status = %d, %v for %v", got, ok, err)
				}
				var se *UpstreamStatusError
				if !errors.As(err, &se) || se.Status != c.want {
					t.Fatalf("errors.As: %v", err)
				}
			})
		}
	}
	// A JSON-RPC error is a response, never a typed error.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"error":{"code":-1,"message":"status 401"}}`)
	}))
	defer srv.Close()
	if err := streamableWriteErr(t, srv.URL, true); err != nil {
		t.Fatalf("a JSON-RPC error body is not a transport failure: %v", err)
	}
}

// --- SSE: the server-chosen POST endpoint -------------------------------------

func sseUpstream(t *testing.T, endpoint string, post http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "text/event-stream")
			fmt.Fprintf(w, "event: endpoint\ndata: %s\n\n", endpoint)
			w.(http.Flusher).Flush()
			<-r.Context().Done()
			return
		}
		post(w, r)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestRedactSSEEndpointIsNotInErrors(t *testing.T) {
	endpoint := "/messages-" + echoMarker + "?echo=" + echoMarker
	posts := map[string]http.HandlerFunc{
		"HTTP 500": func(w http.ResponseWriter, r *http.Request) { http.Error(w, "x", 500) },
		"HTTP 401": func(w http.ResponseWriter, r *http.Request) { http.Error(w, "x", 401) },
		"malformed reply": func(w http.ResponseWriter, r *http.Request) {
			conn, bw, _ := w.(http.Hijacker).Hijack()
			fmt.Fprintf(bw, "HTTP/1.1 200 OK\r\nstatus 401 %s\r\n\r\n", echoMarker)
			bw.Flush()
			conn.Close()
		},
		"connection dropped": func(w http.ResponseWriter, r *http.Request) {
			conn, _, _ := w.(http.Hijacker).Hijack()
			conn.Close()
		},
	}
	for name, post := range posts {
		for _, redact := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s/redact=%v", name, redact), func(t *testing.T) {
				srv := sseUpstream(t, endpoint, post)
				tr := NewSSETransport(srv.URL, map[string]string{"X-Stored": "stored-header-value"}, false, true)
				tr.SetRedactUpstreamText(redact)
				defer tr.Close()
				buf := captureLogs(t)
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := tr.Connect(ctx); err != nil {
					t.Fatal(err)
				}
				err := tr.WriteMessage(ctx, []byte(`{"jsonrpc":"2.0","id":1,"method":"ping"}`))
				if err == nil {
					t.Fatal("expected an error")
				}
				has := strings.Contains(err.Error(), echoMarker)
				if redact && (has || strings.Contains(buf.String(), echoMarker) || strings.Contains(buf.String(), "messages-")) {
					t.Fatalf("server-chosen endpoint reached the error or a log: %v\nlogs: %s", err, buf.String())
				}
				if redact {
					if strings.Contains(buf.String(), srv.URL) && strings.Contains(buf.String(), "received message endpoint") {
						t.Fatalf("accepted endpoint origin logged with redaction on: %s", buf.String())
					}
				}
				if !redact && !has && name != "HTTP 401" && name != "HTTP 500" && name != "malformed reply" && name != "connection dropped" {
					t.Fatalf("default behaviour changed: %v", err)
				}
				if !redact && name == "HTTP 500" && !has {
					t.Fatalf("default behaviour changed (endpoint missing): %v", err)
				}
				if st, ok := UpstreamHTTPStatus(err); (name == "HTTP 500" && (!ok || st != 500)) || (name == "HTTP 401" && (!ok || st != 401)) {
					t.Fatalf("SSE POST status not typed: %d %v %v", st, ok, err)
				}
				if name == "malformed reply" || name == "connection dropped" {
					if _, ok := UpstreamHTTPStatus(err); ok {
						t.Fatalf("a failed exchange must not carry a status: %v", err)
					}
				}
			})
		}
	}
}

// --- the response guard -------------------------------------------------------

func TestResponseGuardWrapsEveryAnswerAndIsOffByDefault(t *testing.T) {
	srv, proxy := newStreamableTestServer(t, 5)
	defer srv.Close()

	// Off by default: the tool call is answered as before.
	sid := mustInitSession(t, srv.URL, "")
	status, _, body := streamablePost(t, srv.URL+"/mcp", sid, "", toolCallBody(2, "echo"))
	if status != 200 || !strings.Contains(body, `"result"`) || strings.Contains(body, "GUARDED") {
		t.Fatalf("default: status=%d body=%s", status, body)
	}

	var seen []string
	proxy.SetResponseGuard(func(ctx context.Context, req *Request, handle func(context.Context) (*Response, error)) (*Response, error) {
		seen = append(seen, req.Method)
		resp, err := handle(ctx)
		if resp != nil && resp.Error == nil {
			resp.Result = []byte(`{"content":[{"type":"text","text":"GUARDED"}]}`)
		}
		return resp, err
	})
	for _, m := range []map[string]any{
		{"jsonrpc": "2.0", "id": 3, "method": "ping"},
		{"jsonrpc": "2.0", "id": 4, "method": "tools/list"},
		toolCallBody(5, "echo"),
	} {
		status, _, body = streamablePost(t, srv.URL+"/mcp", sid, "", m)
		if status != 200 || !strings.Contains(body, "GUARDED") {
			t.Fatalf("guard skipped for %v: status=%d body=%s", m["method"], status, body)
		}
	}
	if strings.Join(seen, ",") != "ping,tools/list,tools/call" {
		t.Fatalf("guard saw %v", seen)
	}
}
