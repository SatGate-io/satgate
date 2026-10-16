package mcpserver

import (
	"strings"
	"testing"
)

// A session id names the token it was issued to; nothing else matches it.
func TestStreamSessionIDIsBoundToItsToken(t *testing.T) {
	id := streamSessionIDFor("token-alice")
	if !validSessionID(id) || !strings.HasPrefix(id, "mcp-") || len(id) != 4+32+1+32 {
		t.Fatalf("id %q has the wrong shape", id)
	}
	if !streamSessionIDBoundTo(id, "token-alice") {
		t.Fatal("id not bound to its own token")
	}
	for _, other := range []string{"token-bob", "", "token-alice ", "Token-alice"} {
		if streamSessionIDBoundTo(id, other) {
			t.Fatalf("id bound to %q", other)
		}
	}
	if streamSessionIDFor("token-alice") == id {
		t.Fatal("two ids for one token are equal")
	}
	// Tampering with either half breaks the binding.
	flip := func(s string, i int) string {
		b := []byte(s)
		if b[i] == '0' {
			b[i] = '1'
		} else {
			b[i] = '0'
		}
		return string(b)
	}
	for _, i := range []int{4, 20, 37, len(id) - 1} {
		if streamSessionIDBoundTo(flip(id, i), "token-alice") {
			t.Fatalf("tampered id (byte %d) still bound", i)
		}
	}
	// Ids in the old shape, upper-case hex, or other shapes bind to nothing.
	for _, bad := range []string{
		"mcp-0123456789abcdef0123456789abcdef",
		strings.ToUpper(id),
		"mcp-" + strings.Repeat("a", 32) + "-" + strings.Repeat("b", 31),
		"x" + id,
		id + "-",
	} {
		if streamSessionIDBoundTo(bad, "token-alice") {
			t.Fatalf("%q bound", bad)
		}
	}
}
