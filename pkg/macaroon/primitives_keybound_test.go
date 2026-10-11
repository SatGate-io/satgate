package macaroon

import (
	"strings"
	"testing"
	"time"
)

// The macaroon primitives that attenuate, re-chain or serialise a token. None of
// them authenticates, and none turns a token signed under one key into a token
// that verifies under another. A hosted sign-in token (signed under its own
// derived key) therefore stays a sign-in token through every one of them, and
// the server's ordinary key refuses it with the typed ErrInvalidSignature.
func TestPrimitivesNeverMakeATokenVerifyUnderAnotherKey(t *testing.T) {
	signIn, err := NewService("hosted-sign-in-derived-key")
	if err != nil {
		t.Fatal(err)
	}
	ordinary, err := NewService("the-servers-ordinary-key")
	if err != nil {
		t.Fatal(err)
	}
	base, err := signIn.Mint("api:read", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	token := signIn.Encode(base)

	refused := func(label, tok string) {
		t.Helper()
		_, err := ordinary.Verify(tok)
		if err == nil {
			t.Fatalf("%s: the ordinary key verified a sign-in token", label)
		}
		if !strings.Contains(err.Error(), "invalid signature") {
			t.Fatalf("%s: %v", label, err)
		}
	}
	refused("the sign-in token itself", token)

	// DelegateWithoutVerify (decode + re-chain; the holder needs no key).
	child, err := ordinary.DelegateWithoutVerify(token, []string{"scope = api:read"})
	if err != nil {
		t.Fatal(err)
	}
	refused("DelegateWithoutVerify under the ordinary key", ordinary.Encode(child))
	if _, err := signIn.Verify(ordinary.Encode(child)); err != nil {
		t.Fatalf("the child of a sign-in token must still verify under the sign-in key: %v", err)
	}

	// AppendBudgetBind (budget_bind.go): a seal made with the ordinary key is a
	// caveat on the same chain; the chain's signature is still the sign-in one.
	sealed, err := ordinary.AppendBudgetBind(token)
	if err != nil {
		t.Fatal(err)
	}
	refused("AppendBudgetBind with the ordinary key", ordinary.Encode(sealed))

	// RecalculateSignature is the ordinary key's chain: applying it to a sign-in
	// token's caveats makes a different token (the holder cannot do this without
	// the key), and it is the caller's job to use it only on tokens it minted.
	mac, _ := signIn.Decode(token)
	if ordinary.RecalculateSignature(mac) == mac.Signature {
		t.Fatal("RecalculateSignature under the ordinary key reproduced the sign-in signature")
	}

	// The V2 binary writer (EncodeBinary, AddFirstPartyCaveat) re-chains from the
	// encoding service's own root key: the output is that key's token.
	bin, err := ordinary.EncodeBinary(mac)
	if err != nil {
		t.Fatal(err)
	}
	back, err := ordinary.DecodeBinary(bin)
	if err != nil {
		t.Fatal(err)
	}
	if back.Signature == mac.Signature {
		t.Fatal("the V2 writer under the ordinary key kept the sign-in signature")
	}
	if _, err := signIn.Verify(signIn.Encode(back)); err == nil {
		t.Fatal("a token re-chained under the ordinary key verified under the sign-in key")
	}
}
