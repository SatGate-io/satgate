package proxy

import (
	"net/http"
	"strings"

	"github.com/satgate-io/satgate/pkg/config"
)

// What the gateway removes from a request before it forwards it to an
// upstream, and what it adds. The request a client sent is not changed; the
// upstream gets a copy (see proxyRequest).
//
// Removed:
//   - Authorization, every value, on a route whose policy made the gateway
//     check it: capability (also the default when a route names no policy),
//     l402 and identity. The request reached the upstream only because that
//     credential was accepted, so it was consumed here.
//   - On any other route (public, chargeback), each Authorization value that
//     carries a token this gateway issued: a macaroon signed under this
//     gateway's root key, sent as "Bearer <macaroon>", as the bare macaroon,
//     or as "L402 <macaroon>:<preimage>" / "LSAT <macaroon>:<preimage>". Any
//     other value is passed on unchanged: the gateway did not check or issue
//     it, and on a route the gateway does not guard it is the upstream's own
//     credential (for example its API key).
//   - Every header SatGate owns (config.IsGatewayOwnedHeader): X-Admin-Token,
//     X-Tenant-ID and every X-SatGate-* name.
//   - SatGate's own cookies (satgateCookie) from the Cookie header. Other
//     cookies are passed on as written.
//
// Added: the upstream's configured headers (config.Upstream.Headers), set
// after the removal, so a configured header replaces a client header of the
// same name.

// authorizationConsumed reports whether a request routed by route reached
// proxyRequest only after the gateway accepted its Authorization header.
func authorizationConsumed(route *config.Route) bool {
	switch route.Policy.Kind {
	case "", "capability", "l402", "identity":
		return true
	}
	return false
}

// removeGatewayCredentials removes from h what the gateway consumed or owns
// (see the comment at the top of this file). consumed is
// authorizationConsumed for the request's route.
func (g *Gateway) removeGatewayCredentials(h http.Header, consumed bool) {
	for name := range h {
		if config.IsGatewayOwnedHeader(name) {
			delete(h, name)
		}
	}

	if consumed {
		h.Del("Authorization")
	} else if values := h.Values("Authorization"); len(values) > 0 {
		kept := values[:0:0]
		for _, v := range values {
			if !g.isGatewayToken(v) {
				kept = append(kept, v)
			}
		}
		if len(kept) == 0 {
			h.Del("Authorization")
		} else if len(kept) != len(values) {
			h["Authorization"] = kept
		}
	}

	removeGatewayCookies(h)
}

// isGatewayToken reports whether an Authorization value carries a token this
// gateway issued.
func (g *Gateway) isGatewayToken(value string) bool {
	if g.macaroonSvc == nil {
		return false
	}
	value = strings.TrimSpace(value)
	scheme, rest, hasScheme := strings.Cut(value, " ")
	if !hasScheme {
		return g.macaroonSvc.SignedByThisKey(value)
	}
	rest = strings.TrimSpace(rest)
	switch {
	case strings.EqualFold(scheme, "Bearer"):
		return g.macaroonSvc.SignedByThisKey(rest)
	case strings.EqualFold(scheme, "L402"), strings.EqualFold(scheme, "LSAT"):
		mac, _, _ := strings.Cut(rest, ":")
		return g.macaroonSvc.SignedByThisKey(mac)
	}
	return false
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

// keepConfiguredHeaders removes from h's Connection header every name of a
// configured header. The reverse proxy drops the headers a Connection header
// names after the Director has run, so without this a client could keep a
// configured header from reaching the upstream.
func keepConfiguredHeaders(h, configured http.Header) {
	values := h.Values("Connection")
	if len(values) == 0 || len(configured) == 0 {
		return
	}
	kept := make([]string, 0, len(values))
	for _, v := range values {
		var names []string
		for _, f := range strings.Split(v, ",") {
			f = strings.TrimSpace(f)
			if f == "" {
				continue
			}
			if _, isConfigured := configured[http.CanonicalHeaderKey(f)]; !isConfigured {
				names = append(names, f)
			}
		}
		if len(names) > 0 {
			kept = append(kept, strings.Join(names, ", "))
		}
	}
	if len(kept) == 0 {
		h.Del("Connection")
		return
	}
	h["Connection"] = kept
}

// upstreamHeaders returns headers with canonical names, ready to be set on
// each request to an upstream. The configuration was checked when it was
// loaded and when the gateway was built (config.CheckUpstreamHeaders).
func upstreamHeaders(headers map[string]string) http.Header {
	if len(headers) == 0 {
		return nil
	}
	out := make(http.Header, len(headers))
	for k, v := range headers {
		out.Set(k, v)
	}
	return out
}
