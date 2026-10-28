package config

import (
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
)

// ErrRequestPath is returned by CheckRequestPath for a path that does not
// pass the request path rule.
var ErrRequestPath = errors.New("request path refused")

// CheckRequestPath is the request path rule. The gateway applies it to the
// path of every request, and to the final path it would send upstream.
// u.EscapedPath() is the path as it is sent; u.Path is that path
// percent-decoded once. Each of the two forms is refused when it
//   - does not begin with "/",
//   - has a segment that is "." or "..",
//   - contains "%2F" or "%5C" (an encoded slash or backslash, any case),
//   - contains a backslash or a NUL byte, or
//   - contains a "%" that is not followed by two hex digits.
//
// A URL with Opaque set is refused. Nothing is repaired.
func CheckRequestPath(u *url.URL) error {
	if u.Opaque != "" {
		return ErrRequestPath
	}
	raw := u.EscapedPath()
	if err := checkPathForm(raw); err != nil {
		return err
	}
	decoded, err := url.PathUnescape(raw)
	if err != nil || decoded != u.Path {
		return ErrRequestPath
	}
	return checkPathForm(decoded)
}

func checkPathForm(p string) error {
	if !strings.HasPrefix(p, "/") {
		return ErrRequestPath
	}
	for i := 0; i < len(p); i++ {
		switch p[i] {
		case 0, '\\':
			return ErrRequestPath
		case '%':
			if i+2 >= len(p) || !isHexDigit(p[i+1]) || !isHexDigit(p[i+2]) {
				return ErrRequestPath
			}
			pair := strings.ToUpper(p[i+1 : i+3])
			if pair == "2F" || pair == "5C" {
				return ErrRequestPath
			}
		}
	}
	for _, seg := range strings.Split(p, "/") {
		if seg == "." || seg == ".." {
			return ErrRequestPath
		}
	}
	return nil
}

func isHexDigit(c byte) bool {
	return c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F'
}

// RewritePath is the URL path a route's static rewrite forwards: Path and
// RawPath are both the configured value, as the gateway sets them.
func RewritePath(rewrite string) *url.URL {
	return &url.URL{Path: rewrite, RawPath: rewrite}
}

// CheckForwardPaths refuses a route whose static rewrite, or an upstream
// whose URL has a path that is not empty, when that path does not pass
// CheckRequestPath on its own. Both become part of the path the gateway
// sends upstream. The error names the route or the upstream.
func CheckForwardPaths(routes []Route, upstreams map[string]Upstream) error {
	for i := range routes {
		if r := routes[i].Rewrite; r != "" && CheckRequestPath(RewritePath(r)) != nil {
			return fmt.Errorf("route %s: rewrite does not pass the request path rule", routes[i].Name)
		}
	}
	names := make([]string, 0, len(upstreams))
	for name := range upstreams {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if !upstreamBasePathOK(upstreams[name].URL) {
			return fmt.Errorf("upstream %s: URL path does not pass the request path rule", name)
		}
	}
	return nil
}

// upstreamBasePathOK reports whether the path of an upstream URL is empty
// or passes CheckRequestPath. When the URL does not parse, only its path
// (from the first "/" after "scheme://host", up to "?" or "#") is judged,
// so a URL that fails to parse for another reason is left to the caller.
func upstreamBasePathOK(raw string) bool {
	if u, err := url.Parse(raw); err == nil {
		if u.Path == "" && u.RawPath == "" {
			return true
		}
		return CheckRequestPath(&url.URL{Path: u.Path, RawPath: u.RawPath}) == nil
	}
	rest := raw
	if i := strings.Index(rest, "://"); i >= 0 {
		rest = rest[i+3:]
	}
	if i := strings.IndexAny(rest, "?#"); i >= 0 {
		rest = rest[:i]
	}
	i := strings.Index(rest, "/")
	if i < 0 {
		return true
	}
	p, err := url.ParseRequestURI(rest[i:])
	return err == nil && CheckRequestPath(&url.URL{Path: p.Path, RawPath: p.RawPath}) == nil
}
