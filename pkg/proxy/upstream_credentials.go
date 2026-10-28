package proxy

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"strings"

	"github.com/satgate-io/satgate/pkg/config"
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
//   - On any other route (public, chargeback), each credential in an
//     Authorization value that carries a token this gateway issued
//     (carriesGatewayToken). Any other credential is passed on unchanged:
//     the gateway did not check or issue it, and on a route the gateway does
//     not guard it is the upstream's own (for example its API key).
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
// removed from Authorization (headers and trailers) in every form the
// gateway accepts one (listed below) and inside exactly one wrapper: a
// quoted string (backslash escapes resolved), an auth-param value, Basic
// credentials, one layer of percent-encoding, or base64 in either alphabet,
// padded or not. Each comma-joined credential is judged on its own; a part
// that does not decode is left as written and stops nothing else. A token a
// client wraps in two or more encodings (base64 of base64, percent of
// percent, base64 of percent-encoded text, percent-encoded Basic
// credentials and the like) is not decoded and is forwarded like any other
// client data, as is a token in another header, the query or the body: the
// gateway never accepts a token in those forms. See withoutGatewayTokens
// and carriesGatewayToken.
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
//     unpadded or standard base64 (macaroon.go Verify, verifyBinary, Decode).

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

// removeGatewayTokens removes from every Authorization value of h each
// credential that carries a token this gateway issued. A value with no such
// credential is kept byte for byte; a value left with none is dropped.
func (g *Gateway) removeGatewayTokens(h http.Header) {
	values := h["Authorization"]
	if len(values) == 0 {
		return
	}
	kept := make([]string, 0, len(values))
	changed := false
	for _, v := range values {
		rest, removed := g.withoutGatewayTokens(v)
		if !removed {
			kept = append(kept, v)
			continue
		}
		changed = true
		if rest != "" {
			kept = append(kept, rest)
		}
	}
	if !changed {
		return
	}
	if len(kept) == 0 {
		delete(h, "Authorization")
		return
	}
	h["Authorization"] = kept
}

// withoutGatewayTokens splits an Authorization value into the credentials it
// carries (credentials), drops each one that carries a token this gateway
// issued (carriesGatewayToken) and joins the others once with ", ". Each
// credential is judged on its own, so nothing in one credential can stop
// another from being removed. A value that cannot be split reliably (a
// quoted string that is not closed) is judged whole: it is dropped if it
// carries a gateway token anywhere, and kept as written otherwise. removed
// reports whether anything was dropped; when nothing was, rest is value
// unchanged.
func (g *Gateway) withoutGatewayTokens(value string) (rest string, removed bool) {
	creds, ok := credentials(value)
	if !ok {
		if g.carriesGatewayToken(value) {
			return "", true
		}
		return value, false
	}
	keep := make([]string, 0, len(creds))
	for _, c := range creds {
		if g.carriesGatewayToken(c) {
			removed = true
			continue
		}
		if c = strings.Trim(c, " \t"); c != "" {
			keep = append(keep, c)
		}
	}
	if !removed {
		return value, false
	}
	return strings.Join(keep, ", "), true
}

// credentials splits an Authorization value into the credentials a client
// joined into it with commas. A comma inside a quoted string does not
// split, and an element that is an auth-param (name=value), or empty,
// belongs to the credential before it, so "Digest a=1, b=2" stays one
// credential while "Bearer x, Basic y" is two. Each credential is a slice
// of value (its bytes as written); the split is one pass over value and
// copies nothing. ok is false when value has a quoted string that is not
// closed, so where its credentials end cannot be told.
func credentials(value string) (creds []string, ok bool) {
	credStart, credEnd := -1, 0 // the credential being built: value[credStart:credEnd]
	element := func(start, end int) {
		e := strings.Trim(value[start:end], " \t")
		if credStart >= 0 && (e == "" || isAuthParam(e)) {
			credEnd = end
			return
		}
		if credStart >= 0 {
			creds = append(creds, value[credStart:credEnd])
		}
		credStart, credEnd = start, end
	}
	start, quoted, escaped := 0, false, false
	for i := 0; i < len(value); i++ {
		c := value[i]
		switch {
		case escaped:
			escaped = false
		case quoted && c == '\\':
			escaped = true
		case c == '"':
			quoted = !quoted
		case c == ',' && !quoted:
			element(start, i)
			start = i + 1
		}
	}
	if quoted {
		return nil, false
	}
	element(start, len(value))
	return append(creds, value[credStart:credEnd]), true
}

// isAuthParam reports whether s starts as an RFC 9110 auth-param does: a
// token, optional whitespace, "=", optional whitespace, then a character
// other than "=". A token68 never matches: its "=" are only trailing
// padding.
func isAuthParam(s string) bool {
	i := 0
	for i < len(s) && isTokenChar(s[i]) {
		i++
	}
	if i == 0 {
		return false
	}
	for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	if i >= len(s) || s[i] != '=' {
		return false
	}
	i++
	for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	return i < len(s) && s[i] != '='
}

func isTokenChar(c byte) bool {
	switch {
	case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		return true
	}
	return strings.IndexByte("!#$%&'*+-.^_`|~", c) >= 0
}

// minTokenLen is shorter than any macaroon this gateway issues, encoded.
const minTokenLen = 16

// carriesGatewayToken reports whether s, one credential (or a whole value
// that cannot be split), carries a token this gateway issued inside the
// removal boundary (see the comment at the top of this file):
//
//   - as written: anywhere in s, under any scheme or none, after any
//     separator, as a parameter value, before or after a ":"
//     (tokenAsWritten);
//   - the same with each quoted string's backslash escapes resolved
//     (unquoted);
//   - after exactly one decoding: one base64 decoding, in either alphabet,
//     padded or not, of one candidate (this covers Basic credentials
//     "user:token" and "token:"), or one percent-decoding of s.
//
// The result of a decoding is only looked at as written; it is never
// decoded again, so a token wrapped in two or more encodings is not found.
// A part that does not decode (a malformed percent sequence, invalid
// base64, a non-UTF-8 byte) is left as written and affects no other part.
// The work is linear in the length of s.
func (g *Gateway) carriesGatewayToken(s string) bool {
	if g.macaroonSvc == nil {
		return false
	}
	if g.tokenInOneLayer(s) {
		return true
	}
	if u := unquoted(s); u != s {
		return g.tokenInOneLayer(u)
	}
	return false
}

// tokenInOneLayer reports whether s carries a gateway token as written,
// inside one base64-encoded candidate, or after one percent-decoding of s.
func (g *Gateway) tokenInOneLayer(s string) bool {
	if g.tokenAsWritten(s) {
		return true
	}
	if eachCandidate(s, func(part string) bool {
		data, ok := base64Decoded(part)
		return ok && g.tokenAsWritten(string(data))
	}) {
		return true
	}
	if p := percentDecoded(s); p != s && g.tokenAsWritten(p) {
		return true
	}
	return false
}

// tokenAsWritten reports whether a candidate in s (eachCandidate) is a
// token signed under this gateway's root key, in either macaroon format and
// either base64 alphabet (macaroon.Service.SignedByThisKey).
func (g *Gateway) tokenAsWritten(s string) bool {
	return eachCandidate(s, g.macaroonSvc.SignedByThisKey)
}

func isBase64Char(c byte) bool {
	switch {
	case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		return true
	}
	return c == '+' || c == '/' || c == '-' || c == '_' || c == '='
}

// eachCandidate calls f with each part of s that could be a base64 token,
// in order, until f returns true, and reports whether one did. A candidate
// is a maximal run of characters a base64 token can contain (letters,
// digits, "+", "/", "-", "_", "="), cut after a run of "=" that more such
// characters follow (a token has "=" only at its end), and at least
// minTokenLen long. One pass over s; nothing is copied.
func eachCandidate(s string, f func(string) bool) bool {
	i := 0
	for i < len(s) {
		if !isBase64Char(s[i]) {
			i++
			continue
		}
		start := i
		for i < len(s) && isBase64Char(s[i]) {
			if s[i] != '=' {
				i++
				continue
			}
			j := i
			for j < len(s) && s[j] == '=' {
				j++
			}
			if j < len(s) && isBase64Char(s[j]) {
				if j-start >= minTokenLen && f(s[start:j]) {
					return true
				}
				start = j
			}
			i = j
		}
		if i-start >= minTokenLen && f(s[start:i]) {
			return true
		}
	}
	return false
}

// base64Decoded decodes part as base64 in either alphabet, padded or not.
// The bytes need not be text: Basic credentials may carry a user name in
// any character set.
func base64Decoded(part string) ([]byte, bool) {
	trimmed := strings.TrimRight(part, "=")
	enc := base64.RawStdEncoding
	if strings.ContainsAny(trimmed, "-_") {
		enc = base64.RawURLEncoding
	}
	data, err := enc.DecodeString(trimmed)
	return data, err == nil && len(data) > 0
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

// unquoted returns s with each backslash escape inside a quoted string
// resolved ("\x" becomes "x"); the quotes and everything outside quoted
// strings are kept. It returns s itself when s has no backslash.
func unquoted(s string) string {
	if strings.IndexByte(s, '\\') < 0 {
		return s
	}
	b := make([]byte, 0, len(s))
	quoted := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if quoted && c == '\\' && i+1 < len(s) {
			i++
			b = append(b, s[i])
			continue
		}
		if c == '"' {
			quoted = !quoted
		}
		b = append(b, c)
	}
	return string(b)
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
