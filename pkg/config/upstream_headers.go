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

// CheckUpstreamHeaders refuses an upstream whose configured headers cannot be
// applied as written: an empty or malformed header name, a value with a line
// break or another control character, two names that differ only in case,
// or a name SatGate owns (see IsGatewayOwnedHeader). Upstreams are checked
// in name order, so the same configuration always gives the same error.
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
			return fmt.Errorf("header name %q is not a valid HTTP header name", k)
		}
		c := http.CanonicalHeaderKey(k)
		if first, dup := seen[c]; dup {
			return fmt.Errorf("headers %q and %q name the same header", first, k)
		}
		seen[c] = k
		if IsGatewayOwnedHeader(k) {
			return fmt.Errorf("header %q is owned by SatGate and cannot be set on an upstream request", k)
		}
		if !validHeaderValue(headers[k]) {
			return fmt.Errorf("header %q has a value with a line break or control character", k)
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
// than horizontal tab.
func validHeaderValue(value string) bool {
	for i := 0; i < len(value); i++ {
		c := value[i]
		if (c < 0x20 && c != '\t') || c == 0x7f {
			return false
		}
	}
	return true
}
