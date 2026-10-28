package config

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
)

// Request headers SatGate owns. The gateway reads them for itself (admin
// authentication, tenant selection) or reserves them for its own use, so it
// never passes a client's value on to an upstream, and an upstream's
// configured headers may not set them.
//
//   - X-Admin-Token: admin authentication of the gateway's /api/ endpoints.
//   - X-Tenant-ID: the tenant a request names to SatGate.
//   - X-SatGate-*: SatGate's own header names (for example X-SatGate-Tenant,
//     which the SatGate CLI sends to name a tenant). A client's value would
//     otherwise reach the upstream looking like one SatGate set.
var gatewayOwnedHeaders = []string{"X-Admin-Token", "X-Tenant-Id"}

// gatewayOwnedHeaderPrefix is the canonical form of SatGate's header prefix.
const gatewayOwnedHeaderPrefix = "X-Satgate-"

// IsGatewayOwnedHeader reports whether name (compared without regard to
// case) is a request header SatGate owns: X-Admin-Token, X-Tenant-ID, or any
// name that starts with X-SatGate-.
func IsGatewayOwnedHeader(name string) bool {
	c := http.CanonicalHeaderKey(strings.TrimSpace(name))
	for _, owned := range gatewayOwnedHeaders {
		if c == owned {
			return true
		}
	}
	return strings.HasPrefix(c, gatewayOwnedHeaderPrefix)
}

// hopByHopHeaders are the header names that apply to a single connection
// (RFC 9110 section 7.6.1 and the names Go's reverse proxy treats the same
// way). The reverse proxy removes them from every request it forwards and
// the HTTP transport writes its own, so a configured value would not arrive.
var hopByHopHeaders = []string{
	"Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Proxy-Connection",
	"Te",
	"Trailer",
	"Transfer-Encoding",
	"Upgrade",
}

// IsHopByHopHeader reports whether name (compared without regard to case)
// is one of hopByHopHeaders.
func IsHopByHopHeader(name string) bool {
	c := http.CanonicalHeaderKey(strings.TrimSpace(name))
	for _, h := range hopByHopHeaders {
		if c == h {
			return true
		}
	}
	return false
}

// proxyManagedHeaders are request headers that describe the client's
// connection to the gateway. The reverse proxy adds the client's address to
// X-Forwarded-For on every request, and X-Forwarded-Host,
// X-Forwarded-Proto and Forwarded carry the same kind of per-request
// information; a fixed configured value would misstate it.
var proxyManagedHeaders = []string{
	"X-Forwarded-For",
	"X-Forwarded-Host",
	"X-Forwarded-Proto",
	"Forwarded",
}

// transportComputedHeaders are request headers the HTTP transport writes
// from the request itself and ignores in the header map.
var transportComputedHeaders = []string{
	"Content-Length",
}

func inCanonical(list []string, name string) bool {
	c := http.CanonicalHeaderKey(name)
	for _, h := range list {
		if c == h {
			return true
		}
	}
	return false
}

// CheckUpstreamHeaders refuses an upstream whose configured headers cannot
// be delivered exactly as written. The error names the upstream and the
// entry, never the value. Refused:
//   - an empty name, or one that is not an RFC 9110 token;
//   - two names that differ only in case;
//   - a name SatGate owns (IsGatewayOwnedHeader);
//   - a hop-by-hop name (IsHopByHopHeader);
//   - a name the proxy manages for each request (X-Forwarded-For,
//     X-Forwarded-Host, X-Forwarded-Proto, Forwarded);
//   - a name the HTTP transport computes (Content-Length);
//   - a value with a control character other than horizontal tab, or one
//     that starts or ends with a space or a horizontal tab (HTTP does not
//     deliver leading or trailing whitespace; a tab between other
//     characters is delivered);
//   - a Host that is empty or not a plain ASCII host or host:port.
//
// Upstreams are checked in name order and entries in name order, so the
// same configuration always gives the same error.
func CheckUpstreamHeaders(upstreams map[string]Upstream) error {
	names := make([]string, 0, len(upstreams))
	for name := range upstreams {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if err := checkHeaderMap(upstreams[name].Headers); err != nil {
			return fmt.Errorf("upstream %s: %w", name, err)
		}
	}
	return nil
}

func checkHeaderMap(headers map[string]string) error {
	keys := make([]string, 0, len(headers))
	for k := range headers {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	seen := make(map[string]string, len(keys))
	for _, k := range keys {
		if !validHeaderName(k) {
			return fmt.Errorf("header %q is not a valid HTTP header name", k)
		}
		c := http.CanonicalHeaderKey(k)
		if first, dup := seen[c]; dup {
			return fmt.Errorf("headers %q and %q name the same header", first, k)
		}
		seen[c] = k
		switch {
		case IsGatewayOwnedHeader(k):
			return fmt.Errorf("header %q is owned by SatGate and cannot be set on an upstream request", k)
		case IsHopByHopHeader(k):
			return fmt.Errorf("header %q is a hop-by-hop header: it applies to one connection and is not forwarded", k)
		case inCanonical(proxyManagedHeaders, k):
			return fmt.Errorf("header %q is managed by the proxy for each request and cannot be configured", k)
		case inCanonical(transportComputedHeaders, k):
			return fmt.Errorf("header %q is computed by the HTTP transport from the request and cannot be configured", k)
		}
		v := headers[k]
		if !validHeaderValue(v) {
			return fmt.Errorf("header %q has a value with a line break or control character", k)
		}
		if v != strings.Trim(v, " \t") {
			return fmt.Errorf("header %q has a value that starts or ends with a space or tab, which HTTP does not deliver", k)
		}
		if c == "Host" && !validHost(v) {
			return fmt.Errorf("header %q must be a host or host:port in ASCII letters, digits and URI host characters", k)
		}
	}
	return nil
}

// validHeaderName reports whether name is an RFC 9110 token.
func validHeaderName(name string) bool {
	if name == "" {
		return false
	}
	for i := 0; i < len(name); i++ {
		c := name[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		case strings.IndexByte("!#$%&'*+-.^_`|~", c) >= 0:
		default:
			return false
		}
	}
	return true
}

// validHeaderValue reports whether value has no control character other
// than horizontal tab (bytes 0x00-0x08, 0x0A-0x1F and 0x7F are refused).
func validHeaderValue(value string) bool {
	for i := 0; i < len(value); i++ {
		c := value[i]
		if (c < 0x20 && c != '\t') || c == 0x7f {
			return false
		}
	}
	return true
}

// validHost reports whether value is a Host the HTTP transport sends exactly
// as written: not empty, and only ASCII letters, digits and the characters
// "-._~!$&'()*+,;=:[]". The transport would convert a non-ASCII name to
// punycode and remove an IPv6 zone ("%..."), so neither is accepted.
func validHost(value string) bool {
	if value == "" {
		return false
	}
	for i := 0; i < len(value); i++ {
		c := value[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		case strings.IndexByte("-._~!$&'()*+,;=:[]", c) >= 0:
		default:
			return false
		}
	}
	return true
}
