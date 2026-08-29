package mcpserver

import "testing"

func TestMatchScopeTenantPrefixedWildcard(t *testing.T) {
	tests := []struct {
		name     string
		scope    string
		toolName string
		want     bool
	}{
		{name: "tenant slug wildcard", scope: "demo-82712860:*", toolName: "ping", want: true},
		{name: "tenant slug mcp wildcard", scope: "demo-82712860:mcp:*", toolName: "ping", want: true},
		{name: "tenant slug namespaced wildcard", scope: "demo-82712860:db:*", toolName: "db:query", want: true},
		{name: "uuid wildcard", scope: "123e4567-e89b-12d3-a456-426614174000:*", toolName: "ping", want: true},
		{name: "unrelated exact colon tool remains exact", scope: "db:query", toolName: "db:query", want: true},
		{name: "unrelated exact colon tool does not widen", scope: "db:query", toolName: "query", want: false},
		{name: "tenant prefix does not widen exact unrelated", scope: "demo-82712860:db:query", toolName: "query", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchScope(tt.scope, tt.toolName); got != tt.want {
				t.Fatalf("matchScope(%q, %q) = %v, want %v", tt.scope, tt.toolName, got, tt.want)
			}
		})
	}
}
