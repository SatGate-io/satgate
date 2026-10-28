package mcpserver

import (
	"encoding/json"
	"testing"
)

// presentedToken and extractMetaToken: blank values are no token, every
// other value is returned unchanged (no trimming of a real token).
func TestPresentedTokenBlankOnly(t *testing.T) {
	for _, blank := range blankTokenValues {
		if got := presentedToken(blank); got != "" {
			t.Fatalf("%q: got %q", blank, got)
		}
	}
	for _, v := range []string{staticTestToken, " " + staticTestToken, staticTestToken + "\t", "Bearer"} {
		if got := presentedToken(v); got != v {
			t.Fatalf("%q changed to %q", v, got)
		}
	}
	if got := extractMetaToken(json.RawMessage(`{"_meta":{"token":" \u00a0 "}}`)); got != "" {
		t.Fatalf("blank _meta.token: %q", got)
	}
}
