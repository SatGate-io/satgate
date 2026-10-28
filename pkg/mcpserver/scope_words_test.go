package mcpserver

import (
	"strings"
	"testing"
)

// The words matchScope reads are the words macaroon.ScopeGrants reads: no
// trimming, and the tenant prefix only for a UUID.
func TestScopeWords_NoTrimTenantUUIDOnly(t *testing.T) {
	const tenant = "0f2b7c4e-1d3a-4b5c-8e9f-0a1b2c3d4e5f"
	for _, tc := range []struct {
		in   string
		want []string
	}{
		{"a,b", []string{"a", "b"}},
		{" a , b ", []string{" a ", " b "}},
		{tenant + ":mcp:*," + tenant + ":x", []string{"mcp:*", "x"}},
		{" " + tenant + ":mcp:*", []string{" " + tenant + ":mcp:*"}},
		{"api:xxxx-xxxx-" + strings.Repeat("x", 22) + ":mcp:*", []string{"api:xxxx-xxxx-" + strings.Repeat("x", 22) + ":mcp:*"}},
	} {
		got := scopeWords(tc.in)
		if strings.Join(got, "|") != strings.Join(tc.want, "|") {
			t.Errorf("scopeWords(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
