package mcpserver

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"
)

// Run baseline probes in a network-disabled container: a broken guard must
// never send a packet to shared address space from a developer's host.
func TestSharedAddressSpaceMCPDial(t *testing.T) {
	transport := SSRFSafeTransport(false, false)
	defer transport.CloseIdleConnections()
	for _, host := range []string{"100.64.0.0", "100.64.0.1", "100.127.255.255", "::ffff:100.64.0.1", "::ffff:100.127.255.255"} {
		t.Run(host, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
			defer cancel()
			conn, err := transport.DialContext(ctx, "tcp", net.JoinHostPort(host, "9"))
			if conn != nil {
				conn.Close()
			}
			var blocked *ssrfBlockedError
			if !errors.As(err, &blocked) {
				t.Fatalf("expected SSRF refusal before connect, got %v", err)
			}
		})
	}
}
