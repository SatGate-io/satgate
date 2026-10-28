package macaroon

import (
	"encoding/base64"
	"errors"
	"slices"
	"testing"
	"time"

	macaroonv2 "gopkg.in/macaroon.v2"
)

func newLineageService(t *testing.T) *Service {
	t.Helper()
	svc, err := NewService("lineage-test-root-key")
	if err != nil {
		t.Fatal(err)
	}
	return svc
}

func verifyEncoded(t *testing.T, svc *Service, mac *Macaroon) *Macaroon {
	t.Helper()
	v, err := svc.Verify(svc.Encode(mac))
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	return v
}

func TestLineage_ContainsEveryAncestorAndEndsAtOwnSignature(t *testing.T) {
	svc := newLineageService(t)
	root, _ := svc.Mint("api:*", time.Now().Add(time.Hour))
	child, err := svc.Delegate(svc.Encode(root), []string{"scope = api:read"})
	if err != nil {
		t.Fatal(err)
	}
	// Offline attenuation, as any holder can do without the root key.
	grandchild, err := svc.DelegateWithoutVerify(svc.Encode(child), []string{"expires = " + time.Now().Add(30*time.Minute).UTC().Format(time.RFC3339), "scope = api:read:x"})
	if err != nil {
		t.Fatal(err)
	}

	lineage, err := svc.Lineage(verifyEncoded(t, svc, grandchild))
	if err != nil {
		t.Fatal(err)
	}
	if lineage[len(lineage)-1] != grandchild.Signature {
		t.Fatal("lineage does not end at the token's own signature")
	}
	for name, sig := range map[string]string{"root": root.Signature, "child": child.Signature} {
		if !slices.Contains(lineage, sig) {
			t.Fatalf("lineage of grandchild does not contain the %s signature", name)
		}
	}
	if len(lineage) != len(grandchild.Caveats)+1 {
		t.Fatalf("lineage length %d, want %d", len(lineage), len(grandchild.Caveats)+1)
	}
}

func TestLineage_ParentAndSiblingDoNotContainChild(t *testing.T) {
	svc := newLineageService(t)
	root, _ := svc.Mint("api:*", time.Now().Add(time.Hour))
	a, _ := svc.Delegate(svc.Encode(root), []string{"scope = api:read"})
	b, _ := svc.Delegate(svc.Encode(root), []string{"scope = api:write"})

	rootLineage, err := svc.Lineage(verifyEncoded(t, svc, root))
	if err != nil {
		t.Fatal(err)
	}
	bLineage, err := svc.Lineage(verifyEncoded(t, svc, b))
	if err != nil {
		t.Fatal(err)
	}
	if slices.Contains(rootLineage, a.Signature) {
		t.Fatal("parent lineage contains the child signature")
	}
	if slices.Contains(bLineage, a.Signature) {
		t.Fatal("sibling lineage contains the other sibling's signature")
	}
	if !slices.Contains(bLineage, root.Signature) {
		t.Fatal("sibling lineage does not contain the shared parent")
	}
}

func TestLineage_BinaryMacaroonAndHolderAttenuation(t *testing.T) {
	svc := newLineageService(t)
	mac, err := svc.MintL402(L402Binding{PaymentHash: "ab", Route: "r", PriceSats: 1}, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	enc, err := svc.EncodeBinary(mac)
	if err != nil {
		t.Fatal(err)
	}
	parent, err := svc.Verify(enc)
	if err != nil {
		t.Fatal(err)
	}

	raw, _ := base64.StdEncoding.DecodeString(enc)
	var m macaroonv2.Macaroon
	if err := m.UnmarshalBinary(raw); err != nil {
		t.Fatal(err)
	}
	if err := m.AddFirstPartyCaveat([]byte("methods = GET")); err != nil {
		t.Fatal(err)
	}
	out, _ := m.MarshalBinary()
	child, err := svc.Verify(base64.StdEncoding.EncodeToString(out))
	if err != nil {
		t.Fatal(err)
	}

	lineage, err := svc.Lineage(child)
	if err != nil {
		t.Fatal(err)
	}
	if lineage[len(lineage)-1] != child.Signature || !slices.Contains(lineage, parent.Signature) {
		t.Fatal("binary lineage does not hold the token and its parent")
	}
}

func TestLineage_DoesNotDependOnTheUnsignedVersionField(t *testing.T) {
	svc := newLineageService(t)
	root, _ := svc.Mint("api:*", time.Now().Add(time.Hour))
	child, _ := svc.Delegate(svc.Encode(root), []string{"scope = api:read"})
	child.Version = 2 // not covered by the signature
	v := verifyEncoded(t, svc, child)
	lineage, err := svc.Lineage(v)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(lineage, root.Signature) {
		t.Fatal("changing the version field hid the parent from the lineage")
	}
}

func TestLineage_RefusesMacaroonFromAnotherRootKey(t *testing.T) {
	svc := newLineageService(t)
	other, _ := NewService("a-different-root-key")
	mac, _ := other.Mint("api:*", time.Now().Add(time.Hour))
	if _, err := svc.Lineage(mac); !errors.Is(err, ErrNotIssuedHere) {
		t.Fatalf("want ErrNotIssuedHere, got %v", err)
	}
}
