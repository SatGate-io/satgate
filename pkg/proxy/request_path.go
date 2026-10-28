package proxy

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/satgate-io/satgate/pkg/config"
)

// checkRequestPath applies the request path rule (config.CheckRequestPath)
// to a request or outgoing URL. The gateway matches, checks and forwards
// only paths that pass it.
func checkRequestPath(u *url.URL) error {
	return config.CheckRequestPath(u)
}

// strippedPath returns the Path and RawPath of u with prefix removed from
// a path that passed checkRequestPath and starts with prefix (decoded
// form), removing the same characters from the escaped form, so the rest
// keeps the escaping it had. A result that does not begin with "/" gets
// one. It reports false when the result would not pass checkRequestPath.
func strippedPath(u *url.URL, prefix string) (path, rawPath string, ok bool) {
	if !strings.HasPrefix(u.Path, prefix) {
		return "", "", false
	}
	raw := u.EscapedPath()
	// Find where the first len(prefix) decoded bytes end in raw. Each
	// "%XX" in a checked path is one decoded byte.
	n, i := 0, 0
	for n < len(prefix) && i < len(raw) {
		if raw[i] == '%' {
			i += 3
		} else {
			i++
		}
		n++
	}
	if n != len(prefix) || i > len(raw) {
		return "", "", false
	}
	newPath, newRaw := u.Path[len(prefix):], raw[i:]
	if !strings.HasPrefix(newPath, "/") {
		newPath, newRaw = "/"+newPath, "/"+newRaw
	}
	out := url.URL{Path: newPath}
	if newRaw != out.EscapedPath() {
		out.RawPath = newRaw
	}
	if checkRequestPath(&out) != nil || out.EscapedPath() != newRaw {
		return "", "", false
	}
	return out.Path, out.RawPath, true
}

// forwardPath returns the Path and RawPath the gateway forwards for a
// request to route: the request path, or the route's static rewrite, or
// the request path with the route's prefix stripped (a rewrite takes
// precedence over stripPrefix). It reports false when that path, or the
// final outgoing path after the reverse proxy for the route's upstream
// joins it with the upstream URL's base path, does not pass
// checkRequestPath. It is called when the route is matched, before the
// route's policy runs, and again when the request is forwarded.
func (g *Gateway) forwardPath(route *config.Route, u *url.URL) (path, rawPath string, ok bool) {
	path, rawPath = u.Path, u.RawPath
	switch {
	case route.Rewrite != "":
		rw := config.RewritePath(route.Rewrite)
		path, rawPath = rw.Path, rw.RawPath
	case route.StripPrefix && route.Match.PathPrefix != "":
		if path, rawPath, ok = strippedPath(u, route.Match.Prefix()); !ok {
			return "", "", false
		}
	}
	fwd := &url.URL{Path: path, RawPath: rawPath}
	if checkRequestPath(fwd) != nil {
		return "", "", false
	}
	g.proxyMu.RLock()
	proxy := g.proxies[route.Upstream]
	g.proxyMu.RUnlock()
	if proxy != nil {
		// Run the upstream's own Director on a copy, so the path checked
		// is the one it will send.
		out := &http.Request{URL: fwd, Header: make(http.Header)}
		proxy.Director(out)
		if checkRequestPath(out.URL) != nil {
			return "", "", false
		}
	}
	return path, rawPath, true
}
