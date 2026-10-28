package mcpserver

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/satgate-io/satgate/pkg/macaroon"
)

// With the option off (the default), a zero-credit child is minted exactly as
// it was before the hlp- caveat existed. The expected key is built here from the
// caveats the old code wrote (parent link, then the holder's label and scope)
// and compared to the delegator's output byte for byte. This test uses no
// symbol added by the option, so it compiles and passes unchanged on 9c94681e
// (the prod public pin before this branch); that run is in the PR text.
func TestZeroCreditChildWithTheOptionOffIsMintedAsProdMintsIt(t *testing.T) {
	ctx := context.Background()
	svc, err := macaroon.NewService("test-root-key")
	if err != nil {
		t.Fatal(err)
	}
	root, err := svc.Mint("mcp:*", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	mac, err := svc.Delegate(svc.Encode(root), []string{"token_id = parent-row"})
	if err != nil {
		t.Fatal(err)
	}
	d := NewDelegator(svc, NewInMemoryBudgetEnforcer())
	tok := svc.Encode(mac)
	parent, err := (&MacaroonAuthenticator{Service: svc}).Verify(ctx, tok)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []DelegateParams{{Label: "worker", Scope: "api:read"}, {Label: "only-label"}, {}} {
		p := p
		res, err := d.Delegate(ctx, parent, &p)
		if err != nil {
			t.Fatal(err)
		}
		want := []string{"parent = " + parent.TokenID}
		if p.Label != "" {
			want = append(want, "label = "+p.Label)
		}
		if p.Scope != "" {
			want = append(want, "scope = "+p.Scope)
		}
		exp, err := svc.Delegate(tok, want)
		if err != nil {
			t.Fatal(err)
		}
		if res.Token != svc.Encode(exp) {
			t.Fatalf("params %+v: the child key differs from the key prod mints", p)
		}
		got, err := svc.Verify(res.Token)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Join(got.Caveats, "\n") != strings.Join(exp.Caveats, "\n") {
			t.Fatalf("caveat list differs:\n%v\n%v", got.Caveats, exp.Caveats)
		}
		n := 0
		for _, c := range got.Caveats {
			if strings.HasPrefix(c, "token_id = ") {
				n++
			}
		}
		if n != 1 {
			t.Fatalf("%d token_id caveats, want only the parent's", n)
		}
	}
}
