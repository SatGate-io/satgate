package macaroon

import (
	"testing"
	"time"
)

func TestService_Mint(t *testing.T) {
	svc, err := NewService("test-secret")
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	mac, err := svc.Mint("api:read", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("Failed to mint macaroon: %v", err)
	}

	if mac.Version != 1 {
		t.Errorf("Expected version 1, got %d", mac.Version)
	}

	if mac.Signature == "" {
		t.Error("Expected non-empty signature")
	}

	if len(mac.Caveats) < 2 {
		t.Errorf("Expected at least 2 caveats, got %d", len(mac.Caveats))
	}
}

func TestService_Verify(t *testing.T) {
	svc, err := NewService("test-secret")
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Mint a token
	mac, err := svc.Mint("api:read", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("Failed to mint macaroon: %v", err)
	}

	// Encode and decode
	token := svc.Encode(mac)
	verified, err := svc.Verify(token)
	if err != nil {
		t.Fatalf("Failed to verify macaroon: %v", err)
	}

	if verified.Signature != mac.Signature {
		t.Error("Signature mismatch after verification")
	}
}

func TestService_Verify_Expired(t *testing.T) {
	svc, err := NewService("test-secret")
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Mint an expired token
	mac, err := svc.Mint("api:read", time.Now().Add(-time.Hour))
	if err != nil {
		t.Fatalf("Failed to mint macaroon: %v", err)
	}

	token := svc.Encode(mac)
	_, err = svc.Verify(token)
	if err == nil {
		t.Error("Expected error for expired token")
	}
}

func TestService_Verify_InvalidSignature(t *testing.T) {
	svc, err := NewService("test-secret")
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Mint a token
	mac, err := svc.Mint("api:read", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("Failed to mint macaroon: %v", err)
	}

	// Tamper with signature
	mac.Signature = "invalid-signature"
	token := svc.Encode(mac)

	_, err = svc.Verify(token)
	if err == nil {
		t.Error("Expected error for invalid signature")
	}
}

func TestService_Delegate(t *testing.T) {
	svc, err := NewService("test-secret")
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Mint parent token
	parent, err := svc.Mint("api:*", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("Failed to mint parent: %v", err)
	}

	parentToken := svc.Encode(parent)

	// Delegate to child with restricted scope
	child, err := svc.Delegate(parentToken, []string{"scope = api:read"})
	if err != nil {
		t.Fatalf("Failed to delegate: %v", err)
	}

	if child.Signature == parent.Signature {
		t.Error("Child signature should differ from parent")
	}

	if len(child.Caveats) <= len(parent.Caveats) {
		t.Error("Child should have more caveats than parent")
	}
}

func TestMacaroon_HasScope(t *testing.T) {
	tests := []struct {
		tokenScope    string
		requiredScope string
		expected      bool
	}{
		{"api:*", "api:read", true},
		{"api:*", "api:write", true},
		{"api:read", "api:read", true},
		{"api:read", "api:write", false},
		{"api:users:*", "api:users:read", true},
		{"api:users:*", "api:orders:read", false},
		{"*", "anything", true},
	}

	for _, tt := range tests {
		t.Run(tt.tokenScope+"->"+tt.requiredScope, func(t *testing.T) {
			mac := &Macaroon{
				Caveats: []string{"scope = " + tt.tokenScope},
			}
			got := mac.HasScope(tt.requiredScope)
			if got != tt.expected {
				t.Errorf("HasScope(%s, %s) = %v, want %v", tt.tokenScope, tt.requiredScope, got, tt.expected)
			}
		})
	}
}

func TestService_EncodeDecode(t *testing.T) {
	svc, err := NewService("test-secret")
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	mac := &Macaroon{
		Version:    1,
		Location:   "https://satgate.io",
		Identifier: "test-id",
		Caveats:    []string{"scope = api:read"},
		Signature:  "abc123",
	}

	encoded := svc.Encode(mac)
	decoded, err := svc.Decode(encoded)
	if err != nil {
		t.Fatalf("Failed to decode: %v", err)
	}

	if decoded.Identifier != mac.Identifier {
		t.Errorf("Identifier mismatch: got %s, want %s", decoded.Identifier, mac.Identifier)
	}

	if decoded.Signature != mac.Signature {
		t.Errorf("Signature mismatch: got %s, want %s", decoded.Signature, mac.Signature)
	}
}

func BenchmarkMint(b *testing.B) {
	svc, _ := NewService("bench-secret")
	expires := time.Now().Add(time.Hour)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		svc.Mint("api:read", expires)
	}
}

func BenchmarkVerify(b *testing.B) {
	svc, _ := NewService("bench-secret")
	mac, _ := svc.Mint("api:read", time.Now().Add(time.Hour))
	token := svc.Encode(mac)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		svc.Verify(token)
	}
}



