package mcpserver

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// The listener's startup check and default address.

// The listener refuses to start with auth.mode unset and warns for none.
func TestListenAndServeChecksAuthMode(t *testing.T) {
	var buf bytes.Buffer
	prev := log.Logger
	log.Logger = zerolog.New(&buf)
	defer func() { log.Logger = prev }()

	p, _ := newAuthModeProxy(t, "none")
	p.config.Auth.Mode = ""
	err := NewSSEServer(p, "127.0.0.1:0").ListenAndServe(context.Background())
	if err == nil || !strings.Contains(err.Error(), "auth.mode is required") {
		t.Fatalf("unset auth.mode: %v", err)
	}

	cfg := &Config{Server: ServerConfig{Transport: "sse"}, Auth: AuthConfig{Mode: "none"}}
	if err := cfg.checkNetworkAuth("127.0.0.1:9100"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), `"level":"warn"`) || !strings.Contains(buf.String(), "auth.mode is none") {
		t.Fatalf("no startup warning for auth.mode none: %s", buf.String())
	}
	buf.Reset()
	cfg.Auth.Mode = "header"
	if err := cfg.checkNetworkAuth("127.0.0.1:9100"); err != nil || buf.Len() != 0 {
		t.Fatalf("header mode: %v %s", err, buf.String())
	}
	loaded := &Config{Server: ServerConfig{Transport: "sse"}, Auth: AuthConfig{Mode: "none"}, authModeUnset: true}
	if err := loaded.checkNetworkAuth("x"); err == nil {
		t.Fatal("a file without auth.mode passed the listener check")
	}
}

// Network listeners default to 127.0.0.1 unless server.host is set.
func TestListenAddrDefaultsToLoopback(t *testing.T) {
	for _, tc := range []struct {
		host string
		port int
		want string
	}{
		{"", 0, "127.0.0.1:9100"},
		{"", 8443, "127.0.0.1:8443"},
		{"0.0.0.0", 0, "0.0.0.0:9100"},
		{"::", 9200, "[::]:9200"},
		{"[::1]", 0, "[::1]:9100"},
		{"mcp.internal", 7000, "mcp.internal:7000"},
	} {
		c := &Config{Server: ServerConfig{Host: tc.host, Port: tc.port}}
		if got := c.ListenAddr(); got != tc.want {
			t.Fatalf("host=%q port=%d: %q, want %q", tc.host, tc.port, got, tc.want)
		}
	}
	_, err := LoadConfig(writeConfig(t, "server:\n  transport: sse\n  host: 0.0.0.0:9100\nauth:\n  mode: none\n"+testUpstreamYAML))
	if err == nil || !strings.Contains(err.Error(), "server.port") {
		t.Fatalf("host with port: %v", err)
	}
	cfg, err := LoadConfig(writeConfig(t, "server:\n  transport: sse\n  host: 0.0.0.0\n  port: 9300\nauth:\n  mode: none\n"+testUpstreamYAML))
	if err != nil || cfg.ListenAddr() != "0.0.0.0:9300" {
		t.Fatalf("configured host: %v", err)
	}
}
