package proxy

import (
	"errors"
	"net/url"
	"strings"

	"github.com/satgate-io/satgate/pkg/config"
)

// errRequestPath is returned by checkRequestPath for a path the gateway
// refuses with 400.
var errRequestPath = errors.New("request path refused")

// checkRequestPath enforces the one form of request path the gateway
// matches, checks and forwards. u.EscapedPath() is the path as it will be
// sent upstream; u.Path is that path percent-decoded once, the form routes
// are matched against. Each of the two forms is refused when it
//   - does not begin with "/",
//   - has a segment that is "." or "..",
//   - contains "%2F" or "%5C" (an encoded slash or backslash, any case),
//   - contains a backslash or a NUL byte, or
//   - contains a "%" that is not followed by two hex digits.
//
// Nothing is repaired: a path that passes is used unchanged.
func checkRequestPath(u *url.URL) error {
	if u.Opaque != "" {
		return errRequestPath
	}
	raw := u.EscapedPath()
	if err := checkPathForm(raw); err != nil {
		return err
	}
	decoded, err := url.PathUnescape(raw)
	if err != nil || decoded != u.Path {
		return errRequestPath
	}
	return checkPathForm(decoded)
}

func checkPathForm(p string) error {
	if !strings.HasPrefix(p, "/") {
		return errRequestPath
	}
	for i := 0; i < len(p); i++ {
		switch p[i] {
		case 0, '\\':
			return errRequestPath
		case '%':
			if i+2 >= len(p) || !isHex(p[i+1]) || !isHex(p[i+2]) {
				return errRequestPath
			}
			pair := strings.ToUpper(p[i+1 : i+3])
			if pair == "2F" || pair == "5C" {
				return errRequestPath
			}
		}
	}
	for _, seg := range strings.Split(p, "/") {
		if seg == "." || seg == ".." {
			return errRequestPath
		}
	}
	return nil
}

// stripPathPrefix removes prefix from a path that passed checkRequestPath
// and starts with prefix (decoded form), removing the same characters from
// the encoded form, so the upstream receives the checked path minus the
// prefix, encoded as the client sent it. A result that does not begin with
// "/" gets one. It reports false, leaving u unchanged, when the result would
// not pass checkRequestPath.
func stripPathPrefix(u *url.URL, prefix string) bool {
	path, rawPath, ok := strippedPath(u, prefix)
	if !ok {
		return false
	}
	u.Path, u.RawPath = path, rawPath
	return true
}

// strippedPath returns the Path and RawPath stripPathPrefix would set.
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

// forwardPathOK reports whether the path route would forward for u passes
// the path rule. It is checked when the route is matched, before the
// route's policy runs, so a request whose stripped path would be refused
// is refused before any credential or payment proof is checked.
func forwardPathOK(route *config.Route, u *url.URL) bool {
	if route.Rewrite != "" || !route.StripPrefix || route.Match.PathPrefix == "" {
		return true
	}
	_, _, ok := strippedPath(u, route.Match.Prefix())
	return ok
}
