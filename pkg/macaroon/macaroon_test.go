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

func TestService_Delegate_ChainedVerification(t *testing.T) {
	svc, err := NewService("test-secret")
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Mint root token
	root, err := svc.Mint("api:*", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("Failed to mint root: %v", err)
	}
	rootToken := svc.Encode(root)

	// Delegate: root → child (restrict to api:read)
	child, err := svc.Delegate(rootToken, []string{"scope = api:read"})
	if err != nil {
		t.Fatalf("Failed to delegate to child: %v", err)
	}
	childToken := svc.Encode(child)

	// Verify child token with root key — chained signature must verify
	verified, err := svc.Verify(childToken)
	if err != nil {
		t.Fatalf("Failed to verify delegated child: %v", err)
	}

	// Child must have parent's caveats plus new one
	if len(verified.Caveats) != len(root.Caveats)+1 {
		t.Errorf("Expected %d caveats, got %d", len(root.Caveats)+1, len(verified.Caveats))
	}
}

func TestService_Delegate_MultiLevel(t *testing.T) {
	svc, err := NewService("test-secret")
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Mint root with wide scope
	root, err := svc.Mint("api:*", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("Failed to mint root: %v", err)
	}
	rootToken := svc.Encode(root)

	// Level 1: restrict to read
	level1, err := svc.Delegate(rootToken, []string{"scope = api:read"})
	if err != nil {
		t.Fatalf("Failed L1 delegation: %v", err)
	}
	level1Token := svc.Encode(level1)

	// Level 2: add rate limit
	level2, err := svc.Delegate(level1Token, []string{"rate_limit = 100"})
	if err != nil {
		t.Fatalf("Failed L2 delegation: %v", err)
	}
	level2Token := svc.Encode(level2)

	// Level 3: add IP restriction
	level3, err := svc.Delegate(level2Token, []string{"ip = 192.168.1.0/24"})
	if err != nil {
		t.Fatalf("Failed L3 delegation: %v", err)
	}
	level3Token := svc.Encode(level3)

	// All levels must verify against root key
	for name, token := range map[string]string{
		"root":   rootToken,
		"level1": level1Token,
		"level2": level2Token,
		"level3": level3Token,
	} {
		if _, err := svc.Verify(token); err != nil {
			t.Errorf("Failed to verify %s: %v", name, err)
		}
	}

	// Level 3 should have root caveats + 3 additional
	if len(level3.Caveats) != len(root.Caveats)+3 {
		t.Errorf("Expected %d caveats at level 3, got %d", len(root.Caveats)+3, len(level3.Caveats))
	}
}

func TestService_Delegate_PreservesIdentifier(t *testing.T) {
	svc, err := NewService("test-secret")
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	root, err := svc.Mint("api:*", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("Failed to mint: %v", err)
	}
	rootToken := svc.Encode(root)

	child, err := svc.Delegate(rootToken, []string{"scope = api:read"})
	if err != nil {
		t.Fatalf("Failed to delegate: %v", err)
	}

	// Child must keep parent's identifier (same chain)
	if child.Identifier != root.Identifier {
		t.Errorf("Child identifier %q should match parent %q", child.Identifier, root.Identifier)
	}
}

func TestService_Delegate_TamperedCaveatFails(t *testing.T) {
	svc, err := NewService("test-secret")
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	root, err := svc.Mint("api:*", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("Failed to mint: %v", err)
	}
	rootToken := svc.Encode(root)

	child, err := svc.Delegate(rootToken, []string{"scope = api:read"})
	if err != nil {
		t.Fatalf("Failed to delegate: %v", err)
	}

	// Tamper: try to widen scope by modifying a caveat
	child.Caveats[len(child.Caveats)-1] = "scope = api:*"
	tamperedToken := svc.Encode(child)

	_, err = svc.Verify(tamperedToken)
	if err == nil {
		t.Error("Tampered caveat should fail verification — attenuation must be enforced cryptographically")
	}
}

func TestService_Delegate_RemovedCaveatFails(t *testing.T) {
	svc, err := NewService("test-secret")
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	root, err := svc.Mint("api:*", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("Failed to mint: %v", err)
	}
	rootToken := svc.Encode(root)

	child, err := svc.Delegate(rootToken, []string{"scope = api:read", "rate_limit = 50"})
	if err != nil {
		t.Fatalf("Failed to delegate: %v", err)
	}

	// Tamper: remove the rate limit caveat (try to widen permissions)
	child.Caveats = child.Caveats[:len(child.Caveats)-1]
	tamperedToken := svc.Encode(child)

	_, err = svc.Verify(tamperedToken)
	if err == nil {
		t.Error("Removing a caveat should fail verification — caveats are cryptographically bound")
	}
}

func TestService_DelegateWithoutVerify(t *testing.T) {
	svc, err := NewService("test-secret")
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Mint and delegate normally
	root, err := svc.Mint("api:*", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("Failed to mint: %v", err)
	}
	rootToken := svc.Encode(root)

	// DelegateWithoutVerify should produce same result as Delegate
	child1, err := svc.Delegate(rootToken, []string{"scope = api:read"})
	if err != nil {
		t.Fatalf("Delegate failed: %v", err)
	}

	child2, err := svc.DelegateWithoutVerify(rootToken, []string{"scope = api:read"})
	if err != nil {
		t.Fatalf("DelegateWithoutVerify failed: %v", err)
	}

	if child1.Signature != child2.Signature {
		t.Error("DelegateWithoutVerify should produce same signature as Delegate")
	}

	// And it should verify
	childToken := svc.Encode(child2)
	if _, err := svc.Verify(childToken); err != nil {
		t.Fatalf("DelegateWithoutVerify token should verify: %v", err)
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

func BenchmarkDelegateMultiLevel(b *testing.B) {
	svc, _ := NewService("bench-secret")
	root, _ := svc.Mint("api:*", time.Now().Add(time.Hour))
	rootToken := svc.Encode(root)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l1, _ := svc.Delegate(rootToken, []string{"scope = api:read"})
		l1Token := svc.Encode(l1)
		l2, _ := svc.Delegate(l1Token, []string{"rate_limit = 100"})
		svc.Verify(svc.Encode(l2))
	}
}
