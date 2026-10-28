package proxy

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// Unit tests for upstreamTransport and the Authorization splitter in
// upstream_credentials.go.

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// Trailer values that reach the outgoing request while its body is being
// read are removed by the time the body has been read to its end, which is
// when the transport writes the trailers.
func TestUpstreamTransport_TrailerValuesAddedDuringBodyReadAreRemoved(t *testing.T) {
	gw, _ := newTestGatewayWithUpstream(t, newHeaderRecorder(t).srv, "public")
	tok, _ := mintRoot(t, gw)
	for _, consumed := range []bool{true, false} {
		var atEOF http.Header
		tr := &upstreamTransport{
			g:       gw,
			headers: http.Header{"X-Upstream-Key": {"configured"}},
			base: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				// As the body is read, values arrive.
				req.Trailer["Authorization"] = []string{"Bearer " + tok}
				req.Trailer["X-Admin-Token"] = []string{"fixture"}
				req.Trailer["Cookie"] = []string{"cloud_session=fixture; theme=dark"}
				req.Trailer["X-Upstream-Key"] = []string{"from-client"}
				req.Trailer["Proxy-Authorization"] = []string{"Basic Zm9v"}
				req.Trailer["X-Benign"] = []string{"kept"}
				if _, err := io.ReadAll(req.Body); err != nil {
					return nil, err
				}
				atEOF = req.Trailer.Clone()
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody, Request: req}, nil
			}),
		}
		req, _ := http.NewRequest(http.MethodPost, "http://upstream.test/x", strings.NewReader("body"))
		req.Trailer = http.Header{"Authorization": nil, "X-Benign": nil}
		req = req.WithContext(withForwardRules(req.Context(), forwardRules{consumed: consumed}))
		if _, err := tr.RoundTrip(req); err != nil {
			t.Fatal(err)
		}
		for _, k := range []string{"Authorization", "X-Admin-Token", "X-Upstream-Key", "Proxy-Authorization"} {
			if len(atEOF[k]) > 0 {
				t.Errorf("consumed=%v: trailer %s was still set when the body ended", consumed, k)
			}
		}
		if c := atEOF["Cookie"]; len(c) != 1 || c[0] != "theme=dark" {
			t.Errorf("consumed=%v: trailer Cookie = %q, want SatGate's cookie removed", consumed, c)
		}
		if atEOF.Get("X-Benign") != "kept" {
			t.Errorf("consumed=%v: a trailer the gateway does not own was removed", consumed)
		}
	}
}

// The transport sets configured headers and Host on its own copy: the
// request the reverse proxy hands it is not changed.
func TestUpstreamTransport_DoesNotChangeTheRequestItIsGiven(t *testing.T) {
	gw, _ := newTestGatewayWithUpstream(t, newHeaderRecorder(t).srv, "public")
	var sent *http.Request
	tr := &upstreamTransport{
		g:       gw,
		headers: http.Header{"X-Upstream-Key": {"configured"}},
		host:    "configured.example",
		base: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			sent = req
			return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody, Request: req}, nil
		}),
	}
	req, _ := http.NewRequest(http.MethodGet, "http://upstream.test/x", nil)
	req.Header.Set("X-Upstream-Key", "from-client")
	req.Trailer = http.Header{"X-Admin-Token": {"fixture"}}
	if _, err := tr.RoundTrip(req); err != nil {
		t.Fatal(err)
	}
	if sent.Header.Get("X-Upstream-Key") != "configured" || sent.Host != "configured.example" || len(sent.Trailer["X-Admin-Token"]) != 0 {
		t.Error("the outgoing copy does not carry the configured header, the Host or the cleaned trailers")
	}
	if req.Header.Get("X-Upstream-Key") != "from-client" || req.Host != "upstream.test" || req.Trailer.Get("X-Admin-Token") != "fixture" {
		t.Error("the request given to the transport was changed")
	}
}

// A request the gateway sends without forwardRules (its own calls) gets the
// strictest trailer rule: every Authorization value is removed.
func TestUpstreamTransport_NoRulesRemovesEveryAuthorizationTrailer(t *testing.T) {
	gw, _ := newTestGatewayWithUpstream(t, newHeaderRecorder(t).srv, "public")
	var sent http.Header
	tr := &upstreamTransport{g: gw, base: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		sent = req.Trailer
		return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody, Request: req}, nil
	})}
	req, _ := http.NewRequest(http.MethodGet, "http://upstream.test/x", nil)
	req.Trailer = http.Header{"Authorization": {"Basic dXNlcjpwYXNz"}}
	if _, err := tr.RoundTrip(req); err != nil {
		t.Fatal(err)
	}
	if len(sent["Authorization"]) != 0 {
		t.Error("an Authorization trailer was kept without forwarding rules")
	}
}

// credentials splits at commas that start a new credential, never inside a
// quoted string or between a credential's auth-params.
func TestCredentialsSplitsJoinedValues(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"Bearer a", []string{"Bearer a"}},
		{"Bearer a, Basic b", []string{"Bearer a", " Basic b"}},
		{"a,b", []string{"a", "b"}},
		{`Digest username="u", realm="r,s", response="x"`, []string{`Digest username="u", realm="r,s", response="x"`}},
		{`Digest a=1, b=2, Bearer t`, []string{`Digest a=1, b=2`, ` Bearer t`}},
		{"Bearer abc==, Basic d", []string{"Bearer abc==", " Basic d"}},
		{"abc==,def", []string{"abc==", "def"}},
		{"Bearer a,", []string{"Bearer a,"}},
		{`L402 macaroon="m", preimage="p"`, []string{`L402 macaroon="m", preimage="p"`}},
	}
	for _, c := range cases {
		if got := credentials(c.in); strings.Join(got, "|") != strings.Join(c.want, "|") {
			t.Errorf("credentials(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// Recognising a token takes time linear in the value's length, whatever
// the value is made of.
func TestCarriesGatewayTokenIsLinear(t *testing.T) {
	gw, _ := newTestGatewayWithUpstream(t, newHeaderRecorder(t).srv, "public")
	for name, v := range map[string]string{
		"padding splits": strings.Repeat("aaaaaaaaaaaaaaaaaaaa=", 50000),
		"one long run":   strings.Repeat("A", 1<<20),
		"many short":     strings.Repeat("Bearer x, ", 100000),
		"percent":        strings.Repeat("%41", 300000),
	} {
		start := time.Now()
		if gw.carriesGatewayToken(v, true) {
			t.Errorf("%s: recognised a gateway token in a value without one", name)
		}
		if d := time.Since(start); d > 5*time.Second {
			t.Errorf("%s: took %v", name, d)
		}
	}
}
