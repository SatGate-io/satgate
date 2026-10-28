package proxy

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/satgate-io/satgate/pkg/config"
)

// rawUpstream is an upstream that records the header lines of each request
// exactly as they arrive on the connection, before any parsing.
type rawUpstream struct {
	url   string
	lines chan []string
}

func newRawUpstream(t *testing.T) *rawUpstream {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	u := &rawUpstream{url: "http://" + ln.Addr().String(), lines: make(chan []string, 1)}
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				r := bufio.NewReader(c)
				var lines []string
				for {
					line, err := r.ReadString('\n')
					if err != nil {
						return
					}
					line = strings.TrimSuffix(line, "\r\n")
					if line == "" {
						break
					}
					lines = append(lines, line)
				}
				u.lines <- lines
				_, _ = io.WriteString(c, "HTTP/1.1 200 OK\r\nContent-Length: 0\r\nConnection: close\r\n\r\n")
			}(c)
		}
	}()
	return u
}

// fieldValues returns the values of every header line named name (without
// regard to case) in lines, exactly as written after "name:" and the
// optional whitespace HTTP allows there.
func fieldValues(lines []string, name string) []string {
	var out []string
	for _, l := range lines[1:] { // lines[0] is the request line
		k, v, ok := strings.Cut(l, ":")
		if ok && strings.EqualFold(k, name) {
			out = append(out, strings.TrimLeft(v, " \t"))
		}
	}
	return out
}

// Every upstream header name the configuration accepts, with an empty and a
// non-empty value, arrives on the wire as exactly one field with exactly
// the configured value, whatever the client sends under the same name; the
// entries that cannot arrive so (an empty User-Agent or Accept-Encoding,
// any Expect) are refused.
func TestUpstreamHeaders_EveryAcceptedNameArrivesExactlyOnTheWire(t *testing.T) {
	names := []string{
		"Authorization", "Cookie", "User-Agent", "Accept", "Accept-Encoding", "Accept-Language",
		"Accept-Charset", "Cache-Control", "Content-Type", "Content-Encoding", "Content-Language",
		"Date", "From", "If-Match", "If-None-Match", "If-Modified-Since", "If-Unmodified-Since",
		"If-Range", "Range", "Max-Forwards", "Origin", "Pragma", "Priority", "Referer", "Via",
		"Warning", "Dnt", "Sec-Fetch-Mode", "Upgrade-Insecure-Requests", "Traceparent", "Baggage",
		"X-Api-Key", "X-Request-Id", "X-Real-Ip", "X-Forwarded-Port", "X-Custom", "Expect", "Host",
	}
	mustRefuse := map[string]bool{
		"User-Agent/": true, "Accept-Encoding/": true, "Host/": true,
		"Expect/": true, "Expect/fixture-value": true, "Expect/100-continue": true,
	}
	for _, name := range names {
		for _, value := range []string{"", "fixture-value", "100-continue"} {
			if value == "100-continue" && name != "Expect" {
				continue
			}
			if name == "Host" && value != "" {
				value = "configured.example"
			}
			key := name + "/" + value
			t.Run(key, func(t *testing.T) {
				headers := map[string]string{name: value}
				if err := config.CheckUpstreamHeaders(map[string]config.Upstream{"u": {Headers: headers}}); err != nil {
					if !mustRefuse[key] {
						t.Fatalf("refused: %v", err)
					}
					return
				}
				if mustRefuse[key] {
					t.Fatal("accepted, but the HTTP client does not send it as configured")
				}
				up := newRawUpstream(t)
				gw := newTestGateway(t, &config.Config{
					Upstreams: map[string]config.Upstream{"u": {URL: up.url, Timeout: 10 * time.Second, Headers: headers}},
					Routes: []config.Route{{Name: "r", Match: config.RouteMatch{PathPrefix: "/"}, Upstream: "u",
						Policy: config.RoutePolicy{Kind: "public"}}},
				})
				req := httptest.NewRequest(http.MethodGet, "http://gateway.test/data", nil)
				if name != "Host" {
					req.Header.Add(name, "from-client-a")
					req.Header.Add(name, "from-client-b")
				}
				rec := httptest.NewRecorder()
				gw.ServeHTTP(rec, req)
				if rec.Code != http.StatusOK {
					t.Fatalf("status %d", rec.Code)
				}
				got := fieldValues(<-up.lines, name)
				if len(got) != 1 || got[0] != value {
					t.Errorf("arrived as %d field(s), exact=%v; want one field with the configured value", len(got), len(got) == 1 && got[0] == value)
				}
			})
		}
	}
}

// The refusals of an empty User-Agent or Accept-Encoding and of Expect
// apply on every load path (ParseYAML, Load from a file, Validate, New),
// and each error names the upstream and the entry, never the value.
func TestUpstreamHeaders_EmptyUserAgentAcceptEncodingAndExpectRefusedOnEveryLoadPath(t *testing.T) {
	const secret = "unique-configured-value-fixture"
	for _, c := range []struct{ name, value string }{
		{"User-Agent", ""}, {"user-agent", ""}, {"Accept-Encoding", ""}, {"ACCEPT-ENCODING", ""},
		{"Expect", secret}, {"expect", "100-continue"}, {"Expect", ""},
	} {
		t.Run(c.name+"/"+fmt.Sprint(len(c.value)), func(t *testing.T) {
			raw := fmt.Sprintf("upstreams:\n  fixture-upstream:\n    url: https://example.com\n    headers:\n      %q: %q\n"+
				"routes:\n  - name: r\n    match: {pathPrefix: /}\n    upstream: fixture-upstream\n    policy: {kind: public}\n", c.name, c.value)
			file := filepath.Join(t.TempDir(), "config.yaml")
			if err := os.WriteFile(file, []byte(raw), 0o600); err != nil {
				t.Fatal(err)
			}
			cfg := &config.Config{
				Upstreams: map[string]config.Upstream{"fixture-upstream": {URL: "https://example.com", Headers: map[string]string{c.name: c.value}}},
				Routes: []config.Route{{Name: "r", Match: config.RouteMatch{PathPrefix: "/"}, Upstream: "fixture-upstream",
					Policy: config.RoutePolicy{Kind: "public"}}},
			}
			_, errParse := config.ParseYAML([]byte(raw))
			_, errLoad := config.Load(file)
			errValidate := cfg.Validate()
			_, errNew := New(Options{Config: cfg, Macaroon: newTestMacaroonService(t)})
			for path, err := range map[string]error{"ParseYAML": errParse, "Load": errLoad, "Validate": errValidate, "New": errNew} {
				if err == nil {
					t.Errorf("%s accepted the entry", path)
					continue
				}
				msg := err.Error()
				if !strings.Contains(msg, "upstream fixture-upstream") || !strings.Contains(msg, `"`+c.name+`"`) {
					t.Errorf("%s: error does not name the upstream and the entry: %v", path, err)
				}
				if strings.Contains(msg, secret) {
					t.Errorf("%s: error carries the configured value", path)
				}
			}
		})
	}
}
