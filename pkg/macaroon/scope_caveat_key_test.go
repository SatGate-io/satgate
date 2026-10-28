package macaroon

import (
	"strings"
	"testing"
	"time"
)

// ScopeCaveatValue finds a scope caveat by the key reading Verify uses. The
// default Service refuses a scope caveat that carries argument rules, so
// "Verify refuses a rule word under this key" observes Verify's reading.
func TestScopeCaveatValue_ReadsKeyAsVerify(t *testing.T) {
	svc, err := NewService("scope-key-test-secret")
	if err != nil {
		t.Fatal(err)
	}
	const rule = ArgumentRulesScopePrefix + "v1:x"
	for _, tc := range []struct {
		caveat string
		want   bool
	}{
		{"scope = " + rule, true},
		{" scope = " + rule, true},
		{"scope  = " + rule, true},
		{"\tscope = " + rule, true},
		{"scope\t = " + rule, true},
		{"scope =  " + rule, true},
		{"scopes = " + rule, false},
		{"Scope = " + rule, false},
		{"x scope = " + rule, false},
	} {
		for _, binary := range []bool{false, true} {
			tok := expiresToken(t, svc, binary, tc.caveat)
			_, verr := svc.Verify(tok)
			verifyReadsScope := verr != nil
			value, ok := ScopeCaveatValue(tc.caveat)
			if ok != tc.want || verifyReadsScope != tc.want {
				t.Errorf("%q binary=%v: ScopeCaveatValue ok=%v, Verify reads a scope=%v, want %v", tc.caveat, binary, ok, verifyReadsScope, tc.want)
			}
			if ok && strings.TrimSpace(value) != rule {
				t.Errorf("%q: value %q", tc.caveat, value)
			}
		}
	}
	// The value is returned as written, so HasScope sees the spaces Verify
	// would see and refuses a malformed scope.
	if v, _ := ScopeCaveatValue("scope = api:read "); v != "api:read " {
		t.Errorf("value trimmed: %q", v)
	}
	mac, _ := svc.Mint("", time.Now().Add(time.Hour))
	mac.Caveats = append(mac.Caveats, "scope = api:*", " scope = api:read")
	if mac.HasScope("api:write") || !mac.HasScope("api:read") {
		t.Error("HasScope does not enforce a scope caveat with a padded key")
	}
}
