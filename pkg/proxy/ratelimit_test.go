package proxy

import (
	"net/http"
	"testing"
)

func TestRateLimiter_AllowsUnderLimit(t *testing.T) {
	rl := newRateLimiter(10)
	for i := 0; i < 10; i++ {
		if !rl.allow("test-ip") {
			t.Fatalf("request %d should be allowed (limit 10)", i+1)
		}
	}
}

func TestRateLimiter_BlocksOverLimit(t *testing.T) {
	rl := newRateLimiter(5)
	for i := 0; i < 5; i++ {
		rl.allow("test-ip")
	}
	if rl.allow("test-ip") {
		t.Fatal("6th request should be blocked (limit 5)")
	}
}

func TestRateLimiter_SeparateKeys(t *testing.T) {
	rl := newRateLimiter(2)
	rl.allow("ip-a")
	rl.allow("ip-a")

	// ip-a is exhausted
	if rl.allow("ip-a") {
		t.Fatal("ip-a should be blocked")
	}

	// ip-b should still work
	if !rl.allow("ip-b") {
		t.Fatal("ip-b should be allowed (separate bucket)")
	}
}

func TestRateLimiter_UnlimitedWhenZero(t *testing.T) {
	rl := newRateLimiter(0)
	for i := 0; i < 1000; i++ {
		if !rl.allow("test") {
			t.Fatal("should be unlimited when rate=0")
		}
	}
}

func TestExtractIP(t *testing.T) {
	tests := []struct {
		name     string
		xff      string
		remote   string
		expected string
	}{
		{"no xff", "", "192.168.1.1:12345", "192.168.1.1"},
		{"xff single", "10.0.0.1", "192.168.1.1:12345", "10.0.0.1"},
		{"xff chain", "10.0.0.1, 172.16.0.1, 192.168.1.1", "proxy:8080", "10.0.0.1"},
		{"no port", "", "192.168.1.1", "192.168.1.1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &http.Request{
				RemoteAddr: tt.remote,
				Header:     http.Header{},
			}
			if tt.xff != "" {
				r.Header.Set("X-Forwarded-For", tt.xff)
			}
			got := extractIP(r)
			if got != tt.expected {
				t.Errorf("extractIP() = %q, want %q", got, tt.expected)
			}
		})
	}
}
