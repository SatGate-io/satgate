package macaroon

import (
	"errors"
	"strconv"
	"testing"
	"time"
)

// Tests of ScopeGrants and ParseExpires, the two readers every scope and
// expiry check uses. Behaviour through Verify, HasScope and GetExpiry is
// tested in scope_expiry_test.go.

func TestScopeGrants(t *testing.T) {
	for _, tc := range []struct {
		granted, required string
		want              bool
	}{
		// Exact match.
		{"api:read", "api:read", true},
		{"l402", "l402", true},
		{"api:read", "api:write", false},
		{"api:read", "api:read:users", false},
		{"API:read", "api:read", false},

		// Same-namespace wildcard.
		{"api:*", "api:read", true},
		{"api:*", "api:users:read", true},
		{"api:*", "api:capability:admin", true},
		{"api:users:*", "api:users:read", true},
		{"api:users:*", "api:orders:read", false},
		{"api:users:*", "api:users", false},
		{"api:*", "api", false},
		{"api:*", "apix:read", false},

		// A wildcard never leaves its namespace, in either direction.
		{"api:*", "mcp:search", false},
		{"api:*", "mcp:*", false},
		{"api:*", "l402", false},
		{"api:*", "read", false},
		{"mcp:*", "mcp:search", true},
		{"mcp:*", "api:read", false},
		{"mcp:*", "api:capability:admin", false},
		{"db:*", "db_query", false},

		// Bare wildcard.
		{"*", "api:read", true},
		{"*", "mcp:search", true},
		{"*", "l402", true},

		// Empty scopes.
		{"", "api:read", false},
		{"api:read", "", false},
		{"", "", false},
		{"*", "", false},

		// Malformed granted scopes.
		{" api:*", "api:read", false},
		{"api:* ", "api:read", false},
		{"api:read,api:write", "api:read", false},
		{"api:*:x", "api:read:x", false},
		{"api*", "apix", false},
		{"api:r*", "api:read", false},
		{"*:*", "api:read", false},
		{":*", "api:read", false},
		{"api::*", "api::read", false},
		{"**", "api:read", false},
		{"api:\x00", "api:\x00", false},

		// Malformed required scopes, including a wildcard.
		{"*", "*", false},
		{"api:*", "api:*", false},
		{"*", "api:", false},
		{"*", ":read", false},
		{"api:*", "api::read", false},
		{"api:*", "api:read users", false},
		{"api:*", "api:read,api:write", false},
		{"api:read ", "api:read ", false},
	} {
		if got := ScopeGrants(tc.granted, tc.required); got != tc.want {
			t.Errorf("ScopeGrants(%q, %q) = %v, want %v", tc.granted, tc.required, got, tc.want)
		}
	}
}

func TestParseExpires(t *testing.T) {
	future := time.Now().Add(time.Hour).Truncate(time.Second)
	for _, tc := range []struct {
		value string
		want  time.Time
		ok    bool
	}{
		{strconv.FormatInt(future.UnixMilli(), 10), future, true},
		{" " + strconv.FormatInt(future.UnixMilli(), 10) + " ", future, true},
		{future.UTC().Format(time.RFC3339), future, true},
		{future.Format(time.RFC3339Nano), future, true},
		{"0", time.UnixMilli(0), true},
		{"24h", time.Time{}, false},
		{"5m", time.Time{}, false},
		{"30s", time.Time{}, false},
		{"0s", time.Time{}, false},
		{"-1h", time.Time{}, false},
		{"", time.Time{}, false},
		{"expired", time.Time{}, false},
		{"tomorrow", time.Time{}, false},
		{"-5", time.Time{}, false},
		{"+5", time.Time{}, false},
		{"12.5", time.Time{}, false},
		{"1e12", time.Time{}, false},
		{"0x10", time.Time{}, false},
		{"99999999999999999999", time.Time{}, false},
		{"2026-11-01", time.Time{}, false},
		{"2026-11-01T00:00:00", time.Time{}, false},
		{"2026-11-01 00:00:00Z", time.Time{}, false},
		{"Sun, 01 Nov 2026 00:00:00 GMT", time.Time{}, false},
	} {
		got, err := ParseExpires(tc.value)
		if tc.ok {
			if err != nil || !got.Equal(tc.want) {
				t.Errorf("ParseExpires(%q) = %v, %v; want %v", tc.value, got, err, tc.want)
			}
			continue
		}
		if err == nil || !errors.Is(err, ErrExpiresNotAbsolute) {
			t.Errorf("ParseExpires(%q) = %v, %v; want ErrExpiresNotAbsolute", tc.value, got, err)
		}
	}
}
