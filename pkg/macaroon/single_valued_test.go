package macaroon

import (
	"strings"
	"testing"
	"time"
)

// Server-written L402 caveats are single-valued: a token that carries one of
// them more than once is refused, and delegation cannot add them.

func mintWithServerCaveats(t *testing.T, svc *Service) *Macaroon {
	t.Helper()
	mac, err := svc.Mint("l402", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	mac.AddCaveat("payment_hash", strings.Repeat("a", 64))
	mac.AddCaveat("route", "cheap")
	mac.AddCaveat("price_sats", "10")
	mac.AddCaveat("invoice_amount_sats", "10")
	mac.Signature = svc.RecalculateSignature(mac)
	return mac
}

func TestVerify_RefusesRepeatedSingleValuedCaveat(t *testing.T) {
	svc, _ := NewService("single-valued-test-key")
	for _, appended := range []string{
		"payment_hash = " + strings.Repeat("b", 64),
		"route = dear",
		"route = cheap",
		"price_sats = 1000",
		"invoice_amount_sats = 1000",
	} {
		t.Run(appended, func(t *testing.T) {
			parent := mintWithServerCaveats(t, svc)
			child, err := svc.DelegateWithoutVerify(svc.Encode(parent), []string{appended})
			if err != nil {
				t.Fatal(err)
			}
			// JSON form.
			if _, err := svc.Verify(svc.Encode(child)); err == nil {
				t.Fatalf("JSON token with a repeated %q was verified", appended)
			}
			// Binary (libmacaroon v2) form.
			bin, err := svc.EncodeBinary(child)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := svc.Verify(bin); err == nil {
				t.Fatalf("binary token with a repeated %q was verified", appended)
			}
		})
	}
}

func TestGetCaveat_SingleValuedKeyRepeatedReturnsEmpty(t *testing.T) {
	m := &Macaroon{Caveats: []string{
		"payment_hash = " + strings.Repeat("a", 64),
		"payment_hash = " + strings.Repeat("b", 64),
	}}
	if got := m.GetCaveat("payment_hash"); got != "" {
		t.Fatalf("GetCaveat on a repeated payment_hash returned a value: %q", got)
	}
}

func TestDelegate_RefusesServerWrittenL402Caveats(t *testing.T) {
	svc, _ := NewService("single-valued-test-key")
	parent := svc.Encode(mintWithServerCaveats(t, svc))
	for _, key := range []string{"payment_hash", "route", "price_sats", "invoice_amount_sats"} {
		cav := key + " = x"
		if _, err := svc.Delegate(parent, []string{cav}); err == nil {
			t.Fatalf("Delegate accepted %q", cav)
		}
		if _, err := svc.Delegate(parent, []string{"scope = l402", "  " + key + "  = x"}); err == nil {
			t.Fatalf("Delegate accepted a spacing variant of %q", key)
		}
		// A holder can append without the root key; the result is refused.
		child, err := svc.DelegateWithoutVerify(parent, []string{"scope = l402", cav})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := svc.Verify(svc.Encode(child)); err == nil {
			t.Fatalf("token with %q appended by a holder was verified", cav)
		}
	}
	// Restrictive caveats that attenuation repeats are still allowed.
	child, err := svc.Delegate(parent, []string{
		"scope = l402",
		"expires = " + time.Now().Add(10*time.Minute).UTC().Format(time.RFC3339),
	})
	if err != nil {
		t.Fatalf("Delegate refused a narrowing scope and expiry: %v", err)
	}
	if _, err := svc.Verify(svc.Encode(child)); err != nil {
		t.Fatalf("attenuated child did not verify: %v", err)
	}
}

func TestVerify_ChecksEveryRepeatedExpiry(t *testing.T) {
	svc, _ := NewService("single-valued-test-key")
	parent := svc.Encode(mintWithServerCaveats(t, svc))
	child, err := svc.Delegate(parent, []string{
		"expires = " + time.Now().Add(-time.Minute).UTC().Format(time.RFC3339),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Verify(svc.Encode(child)); err == nil {
		t.Fatal("token with an expired attenuation caveat was verified")
	}
}
