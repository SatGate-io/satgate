package proxy

import (
	"context"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/satgate-io/satgate/pkg/config"
)

var sharedAddressSpaceHosts = []string{"100.64.0.0", "100.64.0.1", "100.127.255.255", "::ffff:100.64.0.1", "::ffff:100.127.255.255"}

// Baseline dial probes must run in a network-disabled container.
func checkSharedAddressSpaceDial(t *testing.T, transport *http.Transport, marker string) {
	t.Helper()
	defer transport.CloseIdleConnections()
	for _, host := range sharedAddressSpaceHosts {
		t.Run(host, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
			defer cancel()
			conn, err := transport.DialContext(ctx, "tcp", net.JoinHostPort(host, "9"))
			if conn != nil {
				conn.Close()
			}
			if err == nil || !strings.Contains(err.Error(), marker) {
				t.Fatalf("expected SSRF refusal before connect, got %v", err)
			}
		})
	}
}

func TestSharedAddressSpaceIdentityDial(t *testing.T) {
	g := &Gateway{config: &config.Config{}}
	checkSharedAddressSpaceDial(t, g.identityHTTPClient().Transport.(*http.Transport), "identity_jwks_invalid")
}

func TestSharedAddressSpaceIdentityURL(t *testing.T) {
	g := &Gateway{config: &config.Config{}}
	for _, host := range sharedAddressSpaceHosts {
		if err := g.identityURLAllowed("https://" + net.JoinHostPort(host, "443") + "/jwks"); err == nil {
			t.Errorf("shared address accepted: %s", host)
		}
	}
	for _, host := range []string{"100.63.255.255", "100.128.0.0", "::ffff:100.63.255.255", "::ffff:100.128.0.0"} {
		if err := g.identityURLAllowed("https://" + net.JoinHostPort(host, "443") + "/jwks"); err != nil {
			t.Errorf("adjacent public address refused: %s", host)
		}
	}
	g.config.Cloud = &config.CloudConfig{SSRF: &config.CloudSSRFConfig{AllowPrivateIPs: true}}
	for _, host := range sharedAddressSpaceHosts {
		if err := g.identityURLAllowed("https://" + net.JoinHostPort(host, "443") + "/jwks"); err != nil {
			t.Errorf("explicit private-address opt-in changed: %s", host)
		}
	}
}

func TestSharedAddressSpaceProxyDial(t *testing.T) {
	g := &Gateway{config: &config.Config{}}
	proxy, err := g.createProxy("https://upstream.invalid", nil)
	if err != nil {
		t.Fatal(err)
	}
	checkSharedAddressSpaceDial(t, proxy.Transport.(*upstreamTransport).base.(*http.Transport), "SSRF blocked:")
}
