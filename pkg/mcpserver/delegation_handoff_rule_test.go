package mcpserver

import "testing"

func TestHandoffAllowedRule(t *testing.T) {
	for _, tc := range []struct {
		name    string
		limit   int
		limited bool
		depth   int
		want    bool
	}{
		{"unlimited at root", 0, false, 0, true},
		{"unlimited deep", 0, false, 40, true},
		{"zero at root", 0, true, 0, false},
		{"zero deep", 0, true, 3, false},
		{"negative", -1, true, 0, false},
		{"one at root", 1, true, 0, true},
		{"one at depth one", 1, true, 1, false},
		{"three at depth two", 3, true, 2, true},
		{"three at depth three", 3, true, 3, false},
	} {
		if got := handoffAllowed(tc.limit, tc.limited, tc.depth); got != tc.want {
			t.Errorf("%s: handoffAllowed(%d,%v,%d) = %v, want %v", tc.name, tc.limit, tc.limited, tc.depth, got, tc.want)
		}
	}
}
