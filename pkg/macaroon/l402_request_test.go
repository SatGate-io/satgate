package macaroon

import (
	"strings"
	"testing"
	"time"
)

func TestHasScope_ChecksScopeCaveatsInEverySpacing(t *testing.T) {
	for _, caveat := range []string{"scope = forbidden", "scope  = forbidden", " scope = forbidden", "\tscope\t = forbidden"} {
		m := &Macaroon{Caveats: []string{"scope = l402", caveat}}
		if m.HasScope(L402Scope) {
			t.Fatalf("HasScope ignored %q", caveat)
		}
	}
	m := &Macaroon{Caveats: []string{"scope = l402", "Scope = forbidden"}}
	if !m.HasScope(L402Scope) {
		t.Fatal("a case variant of the key is a different key; HasScope must not read it")
	}
}

func TestCheckL402Request(t *testing.T) {
	svc, err := NewService("test-secret-key-for-tests")
	if err != nil {
		t.Fatal(err)
	}
	base, err := svc.MintL402(L402Binding{PaymentHash: strings.Repeat("a", 64), Route: "cheap", PriceSats: 10, InvoiceAmountSats: 10}, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	req := L402Request{Method: "GET", Path: "/cheap/x", Route: "cheap"}
	if err := base.CheckL402Request(req); err != nil {
		t.Fatalf("unrestricted challenge macaroon refused: %v", err)
	}
	with := func(caveats ...string) *Macaroon {
		return &Macaroon{Identifier: base.Identifier, Caveats: append(append([]string{}, base.Caveats...), caveats...)}
	}
	for _, tc := range []struct {
		caveats []string
		ok      bool
	}{
		{[]string{"scope = l402"}, true},
		{[]string{"scope = *"}, true},
		{[]string{"scope = api:read"}, false},
		{[]string{"scope = l402:x"}, false},
		{[]string{"routes = *"}, true},
		{[]string{"routes = /cheap/*"}, true},
		{[]string{"routes = /cheap/x"}, true},
		{[]string{"routes = cheap"}, true},
		{[]string{"routes = /dear/*, dear"}, false},
		{[]string{"routes = /cheap"}, false},
		{[]string{"routes = ,"}, false},
		{[]string{"routes = cheap", "routes = dear"}, false},
		{[]string{"methods = GET"}, true},
		{[]string{"methods = post, get"}, true},
		{[]string{"methods = POST"}, false},
		{[]string{"expires = 5m"}, false},
		{[]string{"expires = 0"}, false},
		{[]string{"expires = " + time.Now().Add(time.Minute).UTC().Format(time.RFC3339)}, true},
		{[]string{"budget = 1 USD"}, false},
		{[]string{"label:team = x"}, false},
		{[]string{"no separator"}, false},
	} {
		err := with(tc.caveats...).CheckL402Request(req)
		if (err == nil) != tc.ok {
			t.Errorf("caveats %q: ok=%v, err=%v", tc.caveats, tc.ok, err)
		}
	}
}
