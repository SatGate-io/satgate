package argrules

import (
	"strconv"
	"strings"
)

const maxArgumentURLBytes = 2048

// splitCleanPath splits an absolute path ("/a/b") into whole segments. It
// refuses an empty segment, a trailing slash, a dot segment, a backslash and
// anything that is not a clean path. "/" has no segments.
func splitCleanPath(p string) ([]string, bool) {
	if p == "" || p[0] != '/' {
		return nil, false
	}
	if p == "/" {
		return nil, true
	}
	segs := strings.Split(p[1:], "/")
	for _, s := range segs {
		if s == "" || s == "." || s == ".." || strings.ContainsRune(s, '\\') {
			return nil, false
		}
	}
	return segs, true
}

// rawPathEscapesOK refuses a percent escape that hides a separator, a dot or
// a percent sign, and any malformed escape. Nothing is decoded.
func rawPathEscapesOK(p string) bool {
	for i := 0; i < len(p); i++ {
		if p[i] != '%' {
			continue
		}
		if i+2 >= len(p) {
			return false
		}
		h1, h2 := p[i+1], p[i+2]
		if !isHex(h1) || !isHex(h2) {
			return false
		}
		switch strings.ToLower(p[i+1 : i+3]) {
		case "2f", "5c", "2e", "25":
			return false
		}
		i += 2
	}
	return true
}

func isHex(c byte) bool {
	return c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F'
}

// matchArgumentURL reports whether raw is an absolute https URL that satisfies
// the condition. Anything unusual is refused rather than normalised.
func matchArgumentURL(raw string, c *URLCondition) bool {
	if c == nil || raw == "" || len(raw) > maxArgumentURLBytes {
		return false
	}
	// No control byte, space, DEL or backslash anywhere. Non-ASCII bytes are
	// refused in the host and path below; a query or fragment may carry them.
	for i := 0; i < len(raw); i++ {
		b := raw[i]
		if b <= 0x20 || b == 0x7f || b == '\\' {
			return false
		}
	}
	const scheme = "https://"
	if len(raw) < len(scheme) || !strings.EqualFold(raw[:len(scheme)], scheme) {
		return false
	}
	rest := raw[len(scheme):]
	end := strings.IndexAny(rest, "/?#")
	authority, tail := rest, ""
	if end >= 0 {
		authority, tail = rest[:end], rest[end:]
	}
	if authority == "" || strings.ContainsAny(authority, "@[]%") {
		return false // empty host, userinfo, IP literal, encoded host
	}
	host, portText := authority, ""
	if i := strings.IndexByte(authority, ':'); i >= 0 {
		host, portText = authority[:i], authority[i+1:]
		if strings.IndexByte(portText, ':') >= 0 {
			return false
		}
	}
	host = asciiLower(host)
	if !ruleHostOK(host) {
		return false // trailing dot, empty label, IP literal, odd character
	}
	if !containsString(c.Hosts, host) {
		return false
	}
	if strings.IndexByte(authority, ':') >= 0 {
		if portText == "" || len(portText) > 5 || (len(portText) > 1 && portText[0] == '0') {
			return false
		}
		port, err := strconv.Atoi(portText)
		if err != nil || port < 1 || port > 65535 || !containsInt(c.Ports, port) {
			return false
		}
		for i := 0; i < len(portText); i++ {
			if portText[i] < '0' || portText[i] > '9' {
				return false
			}
		}
	}
	path := tail
	if i := strings.IndexAny(path, "?#"); i >= 0 {
		path = path[:i]
	}
	if path == "" {
		path = "/"
	}
	for i := 0; i < len(path); i++ {
		if path[i] >= 0x80 {
			return false
		}
	}
	if !rawPathEscapesOK(path) {
		return false
	}
	segs, ok := splitCleanPath(path)
	if !ok {
		return false
	}
	if len(c.PathPrefixes) > 0 {
		matched := false
		for _, p := range c.PathPrefixes {
			ps, _ := splitCleanPath(p)
			if hasSegmentPrefix(segs, ps) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	if len(c.PathSuffixes) > 0 {
		matched := false
		for _, p := range c.PathSuffixes {
			ps, _ := splitCleanPath(p)
			if hasSegmentSuffix(segs, ps) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	return true
}

func hasSegmentPrefix(segs, prefix []string) bool {
	if len(prefix) > len(segs) {
		return false
	}
	for i := range prefix {
		if segs[i] != prefix[i] {
			return false
		}
	}
	return true
}

// hasSegmentSuffix needs at least one segment more than nothing: the suffix
// "/" (no segments) is refused at parse time, so len(suffix) >= 1 here.
func hasSegmentSuffix(segs, suffix []string) bool {
	if len(suffix) == 0 || len(suffix) > len(segs) {
		return false
	}
	off := len(segs) - len(suffix)
	for i := range suffix {
		if segs[off+i] != suffix[i] {
			return false
		}
	}
	return true
}

func asciiLower(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 32
		}
	}
	return string(b)
}

func containsString(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}

func containsInt(list []int, v int) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}
