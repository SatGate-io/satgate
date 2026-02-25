package mcpserver

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"syscall"
	"time"
)

// SSRFSafeTransport creates an http.Transport that blocks connections to
// private/internal IPs, preventing SSRF via DNS rebinding on upstream URLs.
// When allowPrivate is true, the check is bypassed (for local dev/testing).
func SSRFSafeTransport(tlsSkipVerify, allowPrivate bool) *http.Transport {
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	if !allowPrivate {
		dialer.Control = func(network, address string, c syscall.RawConn) error {
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				return err
			}
			ip := net.ParseIP(host)
			if ip != nil && (ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified()) {
				return fmt.Errorf("SSRF blocked: upstream resolved to private/internal IP %s", host)
			}
			return nil
		}
	}

	t := &http.Transport{
		DialContext:         dialer.DialContext,
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	}
	if tlsSkipVerify {
		t.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	return t
}
