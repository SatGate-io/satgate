package proxy

import (
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/satgate-io/satgate/pkg/config"
	"github.com/satgate-io/satgate/pkg/macaroon"
)

// What the gateway removes from a request before it forwards it to an
// upstream, and what it adds. The request a client sent is not changed; the
// upstream gets a copy (see proxyRequest).
//
// Removed, from the header block (proxyRequest) and from request trailers
// (upstreamTransport, before the request is sent and again once its body has
// been read to its end):
//   - Authorization, every value, on a route whose policy made the gateway
//     check it: capability (also the default when a route names no policy),
//     l402 and identity. The request reached the upstream only because that
//     credential was accepted, so it was consumed here.
//   - On any other route (public, chargeback), the whole Authorization
//     field (every value) when any of its values carries a token this
//     gateway issued (carriesGatewayToken); a foreign credential in the same
//     field goes with it. When none does, every value is passed on
//     unchanged: the gateway did not check or issue it, and on a route the
//     gateway does not guard it is the upstream's own (for example its API
//     key). The header block and the trailers are each judged on their own
//     Authorization values.
//   - Every header SatGate owns (config.IsGatewayOwnedHeader): X-Admin-Token,
//     X-Tenant-ID and every X-SatGate-* name.
//   - SatGate's own cookies (satgateCookie) from the Cookie header. Other
//     cookies are passed on as written.
//
// From trailers, also every hop-by-hop name and every name the upstream has
// a configured header for, so a trailer cannot add to or contradict a
// configured header.
//
// Added: the upstream's configured headers (config.Upstream.Headers), set by
// upstreamTransport on the request as it leaves, after the reverse proxy has
// made its own changes, so each arrives with exactly its configured value.
// Names the proxy or the HTTP transport manage are refused at load
// (config.CheckUpstreamHeaders).
//
// The removal boundary on public and chargeback routes. A gateway token is
// found in an Authorization value (headers and trailers) in every form the
// gateway accepts one (listed below), anywhere in the value whatever
// surrounds it, and inside exactly one wrapper: a quoted string's backslash
// escapes, one layer of percent-encoding, or one layer of base64 in either
// alphabet, padded or not (Basic credentials among them). Finding is done
// by character class, without parsing quotes, commas or parameters, so
// nothing malformed elsewhere in the value can hide a token (see
// carriesGatewayToken). A token a client wraps in two or more encodings
// (base64 of base64, percent of percent, base64 of percent-encoded text,
// percent-encoded Basic credentials and the like) is not decoded and is
// forwarded like any other client data, as is a token in another header,
// the query or the body: the gateway never accepts a token in those forms.
//
// Authorization forms the gateway accepts a token it issued in:
//   - "Bearer <token>" (scheme exactly "Bearer" and one space) or the bare
//     token, the whole value: capability routes, extractBearerToken
//     (proxy_oss.go), and /api/capability/ping and /api/capability/admin
//     (handleCapabilityPing, handleCapabilityAdmin).
//   - "L402 <macaroon>:<preimage>" and "LSAT <macaroon>:<preimage>": l402
//     routes, extractL402Token and verifyL402Token (proxy_oss.go).
//   - "Bearer <token>" in the agent simulate request's authorization field
//     (agent_loop.go simulate).
//   - The MCP server: "Bearer <token>" or the bare token, with "Bearer "
//     removed up to twice (pkg/mcpserver sse.go extractAuthToken, then
//     auth.go MacaroonAuthenticator.Verify).
//   - Only the first Authorization value is read (Header.Get).
//   - The token itself, in every case: a libmacaroon binary macaroon in
//     standard base64, padded or not, or a SatGate JSON macaroon in URL-safe
//     unpadded or standard base64 (macaroon.go Verify, verifyBinary, Decode;
//     carriesGatewayToken reads candidates with the same decoding through
//     macaroon.TokenBytes).

// authorizationConsumed reports whether a request routed by route reached
// proxyRequest only after the gateway accepted its Authorization header.
func authorizationConsumed(route *config.Route) bool {
	switch route.Policy.Kind {
	case "", "capability", "l402", "identity":
		return true
	}
	return false
}

// removeGatewayCredentials removes from h, a header block or a trailer
// block, what the gateway consumed or owns (see the comment at the top of
// this file). consumed is authorizationConsumed for the request's route.
func (g *Gateway) removeGatewayCredentials(h http.Header, consumed bool) {
	if h == nil {
		return
	}
	canonicalizeKeys(h)
	for name := range h {
		if config.IsGatewayOwnedHeader(name) {
			delete(h, name)
		}
	}
	if consumed {
		delete(h, "Authorization")
	} else {
		g.removeGatewayTokens(h)
	}
	removeGatewayCookies(h)
}

// canonicalizeKeys stores every value of h under its name's canonical form,
// so a later lookup by canonical name sees every spelling. The server and
// the transport already give canonical names; this covers a header built by
// other code.
func canonicalizeKeys(h http.Header) {
	for k, v := range h {
		if c := http.CanonicalHeaderKey(k); c != k {
			delete(h, k)
			if existing, ok := h[c]; ok {
				h[c] = append(existing, v...)
			} else {
				h[c] = v
			}
		}
	}
}

// removeGatewayTokens drops the Authorization field from h, every value of
// it, when any of its values carries a token this gateway issued
// (carriesGatewayToken). When none does, every value is kept byte for byte.
// Nothing is split or rewritten: a value is either forwarded as the client
// sent it or, with the rest of the field, not at all.
func (g *Gateway) removeGatewayTokens(h http.Header) {
	for _, v := range h["Authorization"] {
		if g.carriesGatewayToken(v) {
			delete(h, "Authorization")
			return
		}
	}
}

// minTokenLen is shorter than any macaroon this gateway issues, encoded.
const minTokenLen = 16

// carriesGatewayToken reports whether value, one Authorization line, carries
// a token this gateway issued inside the removal boundary (see the comment
// at the top of this file). It does not parse the value: quotes, commas,
// parameters and schemes are ignored. It looks at three texts:
//
//   - value as written;
//   - value with each "%" and two hex digits decoded once (a "%" that does
//     not start such a sequence is kept as written), when there is one;
//   - each candidate of value, base64-decoded once (the wrapper of Basic
//     credentials "user:<token>" and "<token>:", and of any other single
//     base64 layer).
//
// In each it takes the candidates (eachCandidate): runs of the characters a
// token is written with, found by character class alone. A candidate is a
// gateway token when macaroon.TokenBytes decodes it (the decoding Verify
// uses for both formats, then either alphabet, padded or not) to a
// macaroon signed under this gateway's root key. The macaroon of an L402 or
// LSAT value is the candidate before the ":".
//
// Each candidate yields at most one decoded result (macaroon.TokenBytes),
// which is checked as a token and, for a candidate of value as written,
// searched for candidates; a result is never decoded as a wrapper again.
// The work is linear in the length of value.
func (g *Gateway) carriesGatewayToken(value string) bool {
	svc := g.macaroonSvc
	if svc == nil {
		return false
	}
	// isToken checks a candidate found inside a wrapper.
	isToken := func(c string) bool {
		data, ok := macaroon.TokenBytes(c)
		return ok && svc.SignedByThisKeyBytes(data)
	}
	// isTokenOrWrapper checks a candidate of value as written, and the
	// candidates of its one decoding.
	isTokenOrWrapper := func(c string) bool {
		data, ok := macaroon.TokenBytes(c)
		if !ok {
			return false
		}
		return svc.SignedByThisKeyBytes(data) || eachCandidate(string(data), isToken)
	}
	if eachCandidate(value, isTokenOrWrapper) {
		return true
	}
	if p := percentDecoded(value); p != value {
		return eachCandidate(p, isToken)
	}
	return false
}

// isRunChar reports whether c can be part of a candidate: a character of
// either base64 alphabet, "=" padding, CR and LF (which Go's base64
// decoders, and so Verify, skip) and the backslash of a quoted-pair escape.
func isRunChar(c byte) bool {
	switch {
	case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		return true
	}
	switch c {
	case '+', '/', '-', '_', '=', '\r', '\n', '\\':
		return true
	}
	return false
}

// eachCandidate calls f with each candidate in s, until f returns true, and
// reports whether one did. Candidates are found by character class alone:
//
//   - s is cut into maximal runs of isRunChar characters;
//   - a run is cut at each backslash, and when it has one it is also taken
//     with each backslash escape resolved ("\x" becomes "x"), so a token in
//     a quoted string with escapes is found whatever the quoting around it;
//   - each of those is cut after a padding group ("=", with any CR or LF)
//     that a base64 character follows, since a token has "=" only at its
//     end (this separates "name=<token>");
//   - a piece with CR or LF is tried whole (Verify skips them) and also as
//     each part between them.
//
// Pieces shorter than minTokenLen are skipped. Every byte of s is in a
// fixed number of candidates, so the work is linear in len(s); only a run
// with a backslash is copied.
func eachCandidate(s string, f func(string) bool) bool {
	for i := 0; i < len(s); {
		if !isRunChar(s[i]) {
			i++
			continue
		}
		start := i
		for i < len(s) && isRunChar(s[i]) {
			i++
		}
		run := s[start:i]
		if strings.IndexByte(run, '\\') < 0 {
			if eachPiece(run, f) {
				return true
			}
			continue
		}
		for _, part := range strings.Split(run, `\`) {
			if eachPiece(part, f) {
				return true
			}
		}
		if eachPiece(resolveEscapes(run), f) {
			return true
		}
	}
	return false
}

// eachPiece cuts run after each padding group that a base64 character
// follows and calls f with each piece of at least minTokenLen, and, for a
// piece with CR or LF, with each part between them.
func eachPiece(run string, f func(string) bool) bool {
	start := 0
	for i := 0; i < len(run); i++ {
		if run[i] != '=' {
			continue
		}
		j := i
		for j < len(run) && (run[j] == '=' || run[j] == '\r' || run[j] == '\n') {
			j++
		}
		if j < len(run) {
			if tryPiece(run[start:j], f) {
				return true
			}
			start = j
		}
		i = j
	}
	return tryPiece(run[start:], f)
}

func tryPiece(p string, f func(string) bool) bool {
	if len(p) < minTokenLen {
		return false
	}
	if f(p) {
		return true
	}
	if strings.IndexAny(p, "\r\n") < 0 {
		return false
	}
	for _, part := range strings.FieldsFunc(p, func(r rune) bool { return r == '\r' || r == '\n' }) {
		if len(part) >= minTokenLen && f(part) {
			return true
		}
	}
	return false
}

// resolveEscapes returns run with each backslash escape resolved: "\x"
// becomes "x", and a backslash at the end is dropped.
func resolveEscapes(run string) string {
	b := make([]byte, 0, len(run))
	for i := 0; i < len(run); i++ {
		if run[i] == '\\' {
			i++
			if i == len(run) {
				break
			}
		}
		b = append(b, run[i])
	}
	return string(b)
}

// percentDecoded decodes each "%" followed by two hex digits in s, once,
// and keeps every other byte, a "%" that does not start such a sequence
// included, as written. It returns s itself when nothing was decoded.
func percentDecoded(s string) string {
	var b []byte
	for i := 0; i < len(s); i++ {
		if s[i] == '%' && i+2 < len(s) && isHex(s[i+1]) && isHex(s[i+2]) {
			if b == nil {
				b = make([]byte, 0, len(s))
				b = append(b, s[:i]...)
			}
			b = append(b, unhex(s[i+1])<<4|unhex(s[i+2]))
			i += 2
			continue
		}
		if b != nil {
			b = append(b, s[i])
		}
	}
	if b == nil {
		return s
	}
	return string(b)
}

func isHex(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

func unhex(c byte) byte {
	switch {
	case c >= '0' && c <= '9':
		return c - '0'
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10
	}
	return c - 'A' + 10
}

// satgateCookieNames are the session cookies SatGate's dashboard and sign-in
// set; a gateway served on the same host receives them with every request.
var satgateCookieNames = map[string]bool{
	"cloud_session":        true,
	"__Host-cloud_session": true,
	"__Host-admin_session": true,
	"impersonation":        true,
	"__Host-impersonation": true,
}

// satgateCookie reports whether a cookie name is SatGate's own: one of
// satgateCookieNames, or any name that starts with "satgate" (without regard
// to case).
func satgateCookie(name string) bool {
	return satgateCookieNames[name] || strings.HasPrefix(strings.ToLower(name), "satgate")
}

// removeGatewayCookies removes SatGate's own cookies from every Cookie
// header value. Every other cookie is kept exactly as written, in its order.
// A value left with no cookie is dropped, and the header with it if none is
// left.
func removeGatewayCookies(h http.Header) {
	values := h.Values("Cookie")
	if len(values) == 0 {
		return
	}
	changed := false
	kept := make([]string, 0, len(values))
	for _, v := range values {
		parts := strings.Split(v, ";")
		keep := parts[:0:0]
		for _, part := range parts {
			name, _, _ := strings.Cut(strings.TrimSpace(part), "=")
			if satgateCookie(strings.TrimSpace(name)) {
				changed = true
				continue
			}
			keep = append(keep, part)
		}
		if len(keep) == len(parts) {
			kept = append(kept, v)
			continue
		}
		joined := strings.TrimSpace(strings.Join(keep, ";"))
		if strings.Trim(joined, "; ") != "" {
			kept = append(kept, strings.TrimLeft(joined, "; "))
		}
	}
	if !changed {
		return
	}
	if len(kept) == 0 {
		h.Del("Cookie")
		return
	}
	h["Cookie"] = kept
}

// upstreamHeaders returns an upstream's configured headers with canonical
// names, ready to be set on each request to it, and its configured Host
// ("" when none), which is not a header field to Go's HTTP client. The
// configuration was checked when it was loaded and when the gateway was
// built (config.CheckUpstreamHeaders).
func upstreamHeaders(headers map[string]string) (http.Header, string) {
	if len(headers) == 0 {
		return nil, ""
	}
	out := make(http.Header, len(headers))
	host := ""
	for k, v := range headers {
		if http.CanonicalHeaderKey(k) == "Host" {
			host = v
			continue
		}
		out.Set(k, v)
	}
	return out, host
}

// forwardRulesKey is the context key for a forwarded request's forwardRules.
type forwardRulesKey struct{}

// forwardRules is what upstreamTransport needs to know about the route a
// request was forwarded for.
type forwardRules struct {
	// consumed is authorizationConsumed for the route.
	consumed bool
}

func withForwardRules(ctx context.Context, rules forwardRules) context.Context {
	return context.WithValue(ctx, forwardRulesKey{}, rules)
}

// forwardRulesFrom returns the rules proxyRequest set on ctx. A request the
// gateway sends for itself has none and gets the strictest: every
// Authorization value is removed.
func forwardRulesFrom(ctx context.Context) forwardRules {
	if rules, ok := ctx.Value(forwardRulesKey{}).(forwardRules); ok {
		return rules
	}
	return forwardRules{consumed: true}
}

// upstreamTransport is the last step before a request leaves the gateway
// for an upstream: it runs after the reverse proxy's Director and after the
// reverse proxy has removed hop-by-hop headers and set X-Forwarded-For.
// It sets the upstream's configured headers and Host, so each arrives
// exactly as configured, and removes the gateway's credentials, hop-by-hop
// names and configured names from the request's trailers: before the
// request is sent (trailer names, and values the gateway already read, for
// example when it read an MCP route's body), and again once the body has
// been read to its end, just before the transport writes the trailers. The
// server puts trailer values that arrive while the body is being read into
// the inbound request, which is never sent; the second pass covers any
// value that reaches the outgoing request's trailers by any other path.
type upstreamTransport struct {
	g       *Gateway
	base    http.RoundTripper
	headers http.Header // canonical names; Host is not among them
	host    string      // configured Host, "" when none
}

func (t *upstreamTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	out := req.Clone(req.Context())
	for name, values := range t.headers {
		out.Header[name] = append([]string(nil), values...)
	}
	if t.host != "" {
		out.Host = t.host
	}
	if out.Trailer != nil {
		rules := forwardRulesFrom(out.Context())
		clean := func() { t.cleanTrailer(out.Trailer, rules) }
		clean()
		if out.Body != nil && out.Body != http.NoBody {
			out.Body = &cleanTrailerAtEOF{ReadCloser: out.Body, clean: clean}
		}
		if getBody := out.GetBody; getBody != nil {
			out.GetBody = func() (io.ReadCloser, error) {
				body, err := getBody()
				if err != nil || body == nil || body == http.NoBody {
					return body, err
				}
				return &cleanTrailerAtEOF{ReadCloser: body, clean: clean}, nil
			}
		}
	}
	return t.base.RoundTrip(out)
}

// cleanTrailer removes from a request's trailers what removeGatewayCredentials
// removes from its headers, every hop-by-hop name, Host, and every name the
// upstream has a configured header for.
func (t *upstreamTransport) cleanTrailer(tr http.Header, rules forwardRules) {
	t.g.removeGatewayCredentials(tr, rules.consumed)
	for name := range tr {
		if config.IsHopByHopHeader(name) || name == "Host" {
			delete(tr, name)
			continue
		}
		if _, configured := t.headers[name]; configured {
			delete(tr, name)
		}
	}
}

// cleanTrailerAtEOF runs clean when the body it wraps has been read to its
// end: the server fills in a request's trailers as the body's last bytes are
// read, and the transport writes them after it has read the body.
type cleanTrailerAtEOF struct {
	io.ReadCloser
	clean func()
}

func (b *cleanTrailerAtEOF) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	if err == io.EOF {
		b.clean()
	}
	return n, err
}
