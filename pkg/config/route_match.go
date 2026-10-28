package config

import (
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
)

// headerMatcher is one compiled entry of RouteMatch.Headers.
type headerMatcher struct {
	name  string         // configured header name, compared case-insensitively
	value string         // exact value, when re is nil
	re    *regexp.Regexp // compiled "~" pattern
}

// CheckRouteMatchers checks and compiles the matchers of every route, and
// refuses a route that sets rateLimit. It is called when a configuration is
// loaded (Validate) and when a gateway is built from one, so each pathRegex
// and "~" header pattern is compiled once, before any request is matched.
//
// A route is refused when:
//   - pathRegex is not a valid Go regular expression;
//   - a methods entry is not an upper-case HTTP method token (methods are
//     compared exactly, so "get" would never match a GET request);
//   - a headers name is not a valid HTTP header name, is Host, or is a
//     framing or hop-by-hop header (Content-Length, Transfer-Encoding,
//     Connection, Keep-Alive, Proxy-Connection, TE, Trailer, Upgrade), or a
//     "~" value is not a valid Go regular expression;
//   - it sets rateLimit, which this gateway does not enforce.
func CheckRouteMatchers(routes []Route) error {
	for i := range routes {
		route := &routes[i]
		if route.RateLimit != nil {
			return fmt.Errorf("route %s sets rateLimit, which is not enforced per route by this gateway: remove it from the route", route.Name)
		}
		for _, m := range route.Match.Methods {
			if !isToken(m) || m != strings.ToUpper(m) {
				return fmt.Errorf("route %s: methods entry %q is not an upper-case HTTP method (methods are compared exactly)", route.Name, m)
			}
		}
		re, headers, err := compileRouteMatch(&route.Match)
		if err != nil {
			return fmt.Errorf("route %s: %w", route.Name, err)
		}
		route.Match.compiledPathRegex = re
		route.Match.compiledHeaders = headers
	}
	return nil
}

// CompileRegex compiles the route's pathRegex and "~" header patterns.
func (r *Route) CompileRegex() error {
	re, headers, err := compileRouteMatch(&r.Match)
	if err != nil {
		return fmt.Errorf("invalid matcher for route %s: %w", r.Name, err)
	}
	r.Match.compiledPathRegex = re
	r.Match.compiledHeaders = headers
	return nil
}

func compileRouteMatch(m *RouteMatch) (*regexp.Regexp, []headerMatcher, error) {
	var re *regexp.Regexp
	if m.PathRegex != "" {
		var err error
		re, err = regexp.Compile(m.PathRegex)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid path regex: %w", err)
		}
	}
	names := make([]string, 0, len(m.Headers))
	for name := range m.Headers {
		names = append(names, name)
	}
	sort.Strings(names)
	headers := make([]headerMatcher, 0, len(names))
	for _, name := range names {
		if !isToken(name) {
			return nil, nil, fmt.Errorf("headers name %q is not a valid HTTP header name", name)
		}
		if strings.EqualFold(name, "Host") {
			// net/http carries the request host in Request.Host, not in
			// the header map, so a Host entry could never match.
			return nil, nil, fmt.Errorf("headers name %q cannot be matched: the request host is not a header here", name)
		}
		if isFramingHeader(name) {
			// The HTTP server consumes or normalises these before routing
			// (identical repeated Content-Length fields become one, for
			// example), so how many times one was sent cannot be judged.
			return nil, nil, fmt.Errorf("headers name %q cannot be matched: it is a framing or hop-by-hop header the HTTP server handles before routing", name)
		}
		value := m.Headers[name]
		hm := headerMatcher{name: name, value: value}
		if strings.HasPrefix(value, "~") {
			hre, err := regexp.Compile(value[1:])
			if err != nil {
				return nil, nil, fmt.Errorf("invalid regex for header %s: %w", name, err)
			}
			hm.re = hre
		}
		headers = append(headers, hm)
	}
	return re, headers, nil
}

// compiled returns the route's compiled matchers. A RouteMatch that was not
// passed through CheckRouteMatchers or CompileRegex (a Route built directly
// in code) is compiled here for this call only; if that fails, the route
// matches nothing.
func (m *RouteMatch) compiled() (*regexp.Regexp, []headerMatcher, bool) {
	if (m.PathRegex == "" || m.compiledPathRegex != nil) && len(m.compiledHeaders) == len(m.Headers) {
		return m.compiledPathRegex, m.compiledHeaders, true
	}
	re, headers, err := compileRouteMatch(m)
	if err != nil {
		return nil, nil, false
	}
	return re, headers, true
}

// Prefix returns PathPrefix without its optional trailing "*".
func (m *RouteMatch) Prefix() string {
	return strings.TrimSuffix(m.PathPrefix, "*")
}

// MatchRequest reports whether a request meets every matcher the route
// sets. path is the request path, already checked and percent-decoded once
// (url.URL.Path). A route that sets no matcher of a kind places no condition
// of that kind:
//   - pathExact: path equals it.
//   - pathPrefix: path starts with it (a trailing "*" is ignored).
//   - pathRegex: the Go regexp matches path (anywhere, unless anchored).
//   - methods: method equals one entry exactly (HTTP methods are
//     case-sensitive; HEAD is not GET).
//   - headers: for each entry, the request carries a header of that name
//     (compared case-insensitively) exactly once, and its value equals the
//     configured value exactly, or, for a value starting with "~", the Go
//     regexp after the "~" matches it (anywhere, unless anchored).
func (r *Route) MatchRequest(path, method string, header http.Header) bool {
	m := &r.Match
	if m.PathExact == "" && m.PathPrefix == "" && m.PathRegex == "" {
		return false
	}
	if m.PathExact != "" && path != m.PathExact {
		return false
	}
	if m.PathPrefix != "" && !strings.HasPrefix(path, m.Prefix()) {
		return false
	}
	re, headers, ok := m.compiled()
	if !ok {
		return false
	}
	if re != nil && !re.MatchString(path) {
		return false
	}
	if len(m.Methods) > 0 {
		found := false
		for _, allowed := range m.Methods {
			if method == allowed {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	for _, hm := range headers {
		values := headerValues(header, hm.name)
		if len(values) != 1 {
			return false
		}
		if hm.re != nil {
			if !hm.re.MatchString(values[0]) {
				return false
			}
		} else if values[0] != hm.value {
			return false
		}
	}
	return true
}

// headerValues returns every value of the header named name, comparing
// names case-insensitively.
func headerValues(header http.Header, name string) []string {
	var values []string
	for k, v := range header {
		if strings.EqualFold(k, name) {
			values = append(values, v...)
		}
	}
	return values
}

// GetRoute finds the first route that matches a request with this path and
// method and no headers.
func (c *Config) GetRoute(path string, method string) *Route {
	return c.GetRouteWithHeaders(path, method, nil)
}

// GetRouteWithHeaders finds the first route that MatchesWithHeaders.
func (c *Config) GetRouteWithHeaders(path string, method string, headers map[string]string) *Route {
	for i := range c.Routes {
		route := &c.Routes[i]
		if route.MatchesWithHeaders(path, method, headers) {
			return route
		}
	}
	return nil
}

// Matches reports whether a request with this path and method, and no
// headers, matches the route (see MatchRequest).
func (r *Route) Matches(path string, method string) bool {
	return r.MatchesWithHeaders(path, method, nil)
}

// MatchesWithHeaders reports whether a request with this path, method and
// headers (one value per name) matches the route (see MatchRequest). A nil
// map is a request with no headers.
func (r *Route) MatchesWithHeaders(path string, method string, headers map[string]string) bool {
	h := make(http.Header, len(headers))
	for k, v := range headers {
		h[k] = append(h[k], v)
	}
	return r.MatchRequest(path, method, h)
}

// framingHeaders are the header names a route cannot match on: message
// framing and hop-by-hop headers, which the HTTP server handles before the
// request is routed.
var framingHeaders = []string{
	"Content-Length", "Transfer-Encoding", "Connection", "Keep-Alive",
	"Proxy-Connection", "TE", "Trailer", "Upgrade",
}

func isFramingHeader(name string) bool {
	for _, h := range framingHeaders {
		if strings.EqualFold(name, h) {
			return true
		}
	}
	return false
}

// isToken reports whether s is a non-empty HTTP token (RFC 9110 5.6.2), the
// syntax of both a method and a header field name.
func isToken(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		case strings.IndexByte("!#$%&'*+-.^_`|~", c) >= 0:
		default:
			return false
		}
	}
	return true
}
