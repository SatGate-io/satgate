package config

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// The configuration reference lists exactly the upstream header names
// CheckUpstreamHeaders refuses, states the value rules it applies (the tab
// rule included) and the compatibility note for the refusals.
func TestDocsListRefusedUpstreamHeadersAsCodeRefusesThem(t *testing.T) {
	doc, err := os.ReadFile(filepath.Join(repoRoot(t), "docs", "reference", "configuration.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(doc)
	start := strings.Index(text, "<!-- upstream-headers-refused:start -->")
	end := strings.Index(text, "<!-- upstream-headers-refused:end -->")
	if start < 0 || end < start {
		t.Fatal("configuration.md has no list of refused upstream header names")
	}
	var documented []string
	for _, line := range strings.Split(text[start:end], "\n") {
		if !strings.HasPrefix(line, "| `") {
			continue
		}
		first := strings.SplitN(line, "|", 3)[1]
		for _, m := range regexp.MustCompile("`([^`]+)`").FindAllStringSubmatch(first, -1) {
			documented = append(documented, m[1])
		}
	}
	want := []string{
		"X-Admin-Token", "X-Tenant-ID", "X-SatGate-*",
		"Connection", "Keep-Alive", "Proxy-Authenticate", "Proxy-Authorization", "Proxy-Connection",
		"Te", "Trailer", "Transfer-Encoding", "Upgrade",
		"X-Forwarded-For", "X-Forwarded-Host", "X-Forwarded-Proto", "Forwarded",
		"Content-Length",
	}
	sort.Strings(documented)
	sorted := append([]string(nil), want...)
	sort.Strings(sorted)
	if strings.Join(documented, ",") != strings.Join(sorted, ",") {
		t.Errorf("documented refused names %q, want %q", documented, sorted)
	}
	check := func(name, value string) error {
		return CheckUpstreamHeaders(map[string]Upstream{"u": {Headers: map[string]string{name: value}}})
	}
	for _, name := range want {
		n := strings.Replace(name, "*", "Anything", 1)
		for _, spelling := range []string{n, strings.ToLower(n), strings.ToUpper(n)} {
			if err := check(spelling, "v"); err == nil {
				t.Errorf("documented as refused but accepted: %s", spelling)
			} else if !strings.Contains(err.Error(), "upstream u:") || !strings.Contains(err.Error(), spelling) {
				t.Errorf("refusal does not name the upstream and the entry: %v", err)
			}
		}
	}
	// Names the reference gives as accepted are accepted.
	for _, name := range []string{"Authorization", "Cookie", "User-Agent", "Accept", "Accept-Encoding", "Expect", "Via", "X-Api-Key", "Host"} {
		if err := check(name, "example.com"); err != nil {
			t.Errorf("documented as accepted but refused: %s: %v", name, err)
		}
	}

	// Value rules, as the reference states them.
	for _, sentence := range []string{
		"A horizontal tab (0x09) is accepted\n  between other characters",
		"a value that starts or ends with a space or a horizontal tab",
		"bytes 0x00 to 0x08, 0x0A to 0x1F",
		"An empty value is\n  accepted",
		"No other\nconfiguration that loaded before is refused.",
	} {
		if !strings.Contains(text, sentence) {
			t.Errorf("configuration.md does not say %q", sentence)
		}
	}
	for value, accepted := range map[string]bool{
		"a\tb": true, "": true, "a b": true, "caf\u00e9": true,
		"\ta": false, "a\t": false, " a": false, "a ": false,
		"a\nb": false, "a\rb": false, "a\x00b": false, "a\x08b": false, "a\x1fb": false, "a\x7fb": false,
	} {
		if err := check("X-Value", value); (err == nil) != accepted {
			t.Errorf("value %q: accepted=%v, want %v", value, err == nil, accepted)
		}
	}
	for host, accepted := range map[string]bool{
		"example.com": true, "example.com:8443": true, "[::1]:8080": true,
		"": false, "bad host": false, "[fe80::1%25en0]": false, "ex\u00e4mple.com": false, "a/b": false,
	} {
		if err := check("Host", host); (err == nil) != accepted {
			t.Errorf("Host %q: accepted=%v, want %v", host, err == nil, accepted)
		}
	}
}
