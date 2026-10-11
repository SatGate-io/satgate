package macaroon

import (
	"strings"
	"testing"
	"time"
)

// Two kinds of primitive handle a token's caveats.
//
// Holder attenuation (Delegate, DelegateWithoutVerify) and AppendBudgetBind add
// caveats to the token's own chain. The chain keeps the key that signed it: a
// token signed under one key still verifies under that key and under no other.
//
// RecalculateSignature and EncodeBinary sign the caveats afresh under the key of
// the Service that calls them (see TestRootKeyWritersReissueUnderTheirOwnKey).
// They take a trusted caller that holds the key.
//
// This test covers the first kind: a hosted sign-in token (signed under its own
// derived key) stays a sign-in token through holder attenuation and
// AppendBudgetBind, and the server's ordinary key refuses it with the typed
// ErrInvalidSignature.
func TestAttenuationKeepsTheChainsKey(t *testing.T) {
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
}

// RecalculateSignature and EncodeBinary re-sign a token's caveats under the key
// of the Service that calls them. A service holding the ordinary key that is
// given a sign-in token's caveats therefore produces an ordinary-key token. The
// primitives do not check where the caveats came from, so the caller must be
// trusted and must hold the key; no request path reaches them with a presented
// token. Callers in this module: Gateway.issueL402Challenge in pkg/proxy
// (proxy_oss.go), which mints the token it re-signs.
func TestRootKeyWritersReissueUnderTheirOwnKey(t *testing.T) {
	signIn, err := NewService("hosted-sign-in-derived-key")
	if err != nil {
		t.Fatal(err)
	}
	ordinary, err := NewService("the-servers-ordinary-key")
	if err != nil {
		t.Fatal(err)
	}
	newSignIn := func() *Macaroon {
		m, err := signIn.Mint("api:read", time.Now().Add(time.Hour))
		if err != nil {
			t.Fatal(err)
		}
		return m
	}

	// RecalculateSignature: assigning the ordinary service's signature to the
	// token makes a token the ordinary key accepts, and the sign-in key no
	// longer accepts it.
	m := newSignIn()
	if _, err := ordinary.Verify(signIn.Encode(m)); err == nil {
		t.Fatal("the ordinary key verified a sign-in token before any re-signing")
	}
	m.Signature = ordinary.RecalculateSignature(m)
	if _, err := ordinary.Verify(ordinary.Encode(m)); err != nil {
		t.Fatalf("a token re-signed with the ordinary key must verify under it: %v", err)
	}
	if _, err := signIn.Verify(signIn.Encode(m)); err == nil {
		t.Fatal("a token re-signed with the ordinary key still verified under the sign-in key")
	}

	// EncodeBinary: the binary form is built from the encoding service's own
	// root key. It verifies under that service and not under the original one.
	bin, err := ordinary.EncodeBinary(newSignIn())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ordinary.Verify(bin); err != nil {
		t.Fatalf("the ordinary service's binary output must verify under the ordinary key: %v", err)
	}
	if _, err := signIn.Verify(bin); err == nil {
		t.Fatal("the ordinary service's binary output verified under the sign-in key")
	}
}
