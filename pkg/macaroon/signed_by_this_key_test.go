package macaroon

import (
	"testing"
	"time"
)

// SignedByThisKey tells a token this service signed (in either format,
// expired or attenuated) from any other value, and checks no caveat.
func TestSignedByThisKey(t *testing.T) {
	svc, err := NewService("signed-by-this-key-root")
	if err != nil {
		t.Fatal(err)
	}
	other, err := NewService("a-different-root-key")
	if err != nil {
		t.Fatal(err)
	}
	live, err := svc.Mint("api:read", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	expired, err := svc.Mint("api:read", time.Now().Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	child, err := svc.Delegate(svc.Encode(live), []string{"scope = api:read:narrow"})
	if err != nil {
		t.Fatal(err)
	}
	binary, err := svc.EncodeBinary(expired)
	if err != nil {
		t.Fatal(err)
	}
	foreign, err := other.Mint("api:read", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	foreignBinary, err := other.EncodeBinary(foreign)
	if err != nil {
		t.Fatal(err)
	}
	tampered := *live
	tampered.Caveats = append(append([]string{}, live.Caveats...), "scope = api:*")

	for name, tok := range map[string]string{
		"live": svc.Encode(live), "expired": svc.Encode(expired), "delegated": svc.Encode(child),
		"binary": binary, "padded with space": "  " + svc.Encode(live) + " ",
	} {
		if !svc.SignedByThisKey(tok) {
			t.Errorf("%s: not recognised as signed by this key", name)
		}
	}
	for name, tok := range map[string]string{
		"foreign": other.Encode(foreign), "foreign binary": foreignBinary, "tampered": svc.Encode(&tampered),
		"empty": "", "api key": "upstream-own-api-key", "jwt": "eyJhbGciOiJFZERTQSJ9.eyJzdWIiOiJ4In0.c2ln",
	} {
		if svc.SignedByThisKey(tok) {
			t.Errorf("%s: recognised as signed by this key", name)
		}
	}
}
