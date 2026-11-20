package config

import "testing"

func TestSharedAddressSpaceConfigIP(t *testing.T) {
	for _, host := range []string{"100.64.0.0", "100.64.0.1", "100.127.255.255", "::ffff:100.64.0.1", "::ffff:100.127.255.255"} {
		if !isPrivateIP(host) {
			t.Errorf("shared address accepted: %s", host)
		}
	}
	for _, host := range []string{"100.63.255.255", "100.128.0.0", "::ffff:100.63.255.255", "::ffff:100.128.0.0", "upstream.invalid"} {
		if isPrivateIP(host) {
			t.Errorf("adjacent address/hostname refused: %s", host)
		}
	}
}
