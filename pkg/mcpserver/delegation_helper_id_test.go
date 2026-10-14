package mcpserver

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/satgate-io/satgate/pkg/macaroon"
)

type captureEvents struct {
	mu     sync.Mutex
	events []Event
}

func (c *captureEvents) Publish(e Event) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, e)
}

// mintWithTokenID mints a key that carries a token_id caveat, as a top-level
// key does (the budget_id seal is the enterprise issuer's and is not needed here).
func mintWithTokenID(t *testing.T, svc *macaroon.Service, id string) *macaroon.Macaroon {
	t.Helper()
	root, err := svc.Mint("api:*", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	mac, err := svc.Delegate(svc.Encode(root), []string{"token_id = " + id})
	if err != nil {
		t.Fatal(err)
	}
	return mac
}

// helperIDRig mints a parent that carries the token_id and budget_id caveats a
// top-level key carries, and a delegator that records its events.
func helperIDRig(t *testing.T) (*macaroon.Service, *Delegator, *captureEvents, *TokenInfo) {
	t.Helper()
	ctx := context.Background()
	svc, err := macaroon.NewService("test-root-key")
	if err != nil {
		t.Fatal(err)
	}
	mac := mintWithTokenID(t, svc, "parent-row")
	budget := NewInMemoryBudgetEnforcer()
	if err := budget.Initialize(ctx, "parent-row", 1000); err != nil {
		t.Fatal(err)
	}
	d := NewDelegator(svc, budget)
	d.EnableHelperRevocationID(true)
	ev := &captureEvents{}
	d.SetEventPublisher(ev)
	tok := svc.Encode(mac)
	info, err := (&MacaroonAuthenticator{Service: svc}).Verify(ctx, tok)
	if err != nil {
		t.Fatal(err)
	}
	info.BudgetID = "parent-row"
	return svc, d, ev, info
}

func tokenIDCaveats(m *macaroon.Macaroon) []string {
	var out []string
	for _, c := range m.Caveats {
		if strings.HasPrefix(c, "token_id = ") {
			out = append(out, strings.TrimPrefix(c, "token_id = "))
		}
	}
	return out
}

// A zero-credit helper carries one server caveat that names it: a second
// token_id caveat, after the parent's, before the holder's own caveats.
func TestZeroCreditHelperCarriesItsOwnRevocationID(t *testing.T) {
	ctx := context.Background()
	svc, d, ev, parent := helperIDRig(t)

	res, err := d.Delegate(ctx, parent, &DelegateParams{Budget: 0, Label: "worker", Scope: "api:read"})
	if err != nil {
		t.Fatal(err)
	}
	mac, err := svc.Verify(res.Token)
	if err != nil {
		t.Fatalf("a new zero-credit helper must verify: %v", err)
	}
	ids := tokenIDCaveats(mac)
	if len(ids) != 2 || ids[0] != "parent-row" || !strings.HasPrefix(ids[1], HelperRevocationIDPrefix) {
		t.Fatalf("token_id caveats = %v, want the parent's first then a %q id", ids, HelperRevocationIDPrefix)
	}

	// The id sits before the holder's caveats, so the holder cannot place one in front of it.
	var posID, posLabel = -1, -1
	for i, c := range mac.Caveats {
		if c == "token_id = "+ids[1] {
			posID = i
		}
		if strings.HasPrefix(c, "label = ") {
			posLabel = i
		}
	}
	if posID < 0 || posLabel < 0 || posID > posLabel {
		t.Fatalf("id at %d, label at %d: the id must come before the holder's caveats", posID, posLabel)
	}

	// The event publishes it. It is an id, not a secret.
	ev.mu.Lock()
	defer ev.mu.Unlock()
	if len(ev.events) != 1 || ev.events[0].Data[EventChildRevocationID] != ids[1] {
		t.Fatalf("child_delegated event data = %v, want %s = %s", ev.events, EventChildRevocationID, ids[1])
	}

	// A copy a holder makes by appending a caveat still carries it.
	other, err := macaroon.NewService("not-the-root-key")
	if err != nil {
		t.Fatal(err)
	}
	cp, err := other.DelegateWithoutVerify(res.Token, []string{"scope = api:read.narrow"})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, id := range tokenIDCaveats(cp) {
		found = found || id == ids[1]
	}
	if !found {
		t.Fatalf("holder-extended copy lost the helper's id: %v", tokenIDCaveats(cp))
	}
}

// Delegating again with the same parent, label and scope gives the same id (so
// the same key and row); a different label, scope or parent gives another id.
func TestZeroCreditHelperIDIsDeterministicPerParentLabelScope(t *testing.T) {
	ctx := context.Background()
	svc, d, _, parent := helperIDRig(t)
	id := func(p *TokenInfo, label, scope string) string {
		t.Helper()
		res, err := d.Delegate(ctx, p, &DelegateParams{Budget: 0, Label: label, Scope: scope})
		if err != nil {
			t.Fatal(err)
		}
		mac, err := svc.Verify(res.Token)
		if err != nil {
			t.Fatal(err)
		}
		ids := tokenIDCaveats(mac)
		return ids[len(ids)-1]
	}
	a, b := id(parent, "worker", "api:read"), id(parent, "worker", "api:read")
	if a != b {
		t.Fatalf("same parent, label and scope gave %s then %s", a, b)
	}
	if c := id(parent, "other", "api:read"); c == a {
		t.Fatal("a different label gave the same id")
	}
	if c := id(parent, "worker", "api:write"); c == a {
		t.Fatal("a different scope gave the same id")
	}
	mac2 := mintWithTokenID(t, svc, "second-parent")
	parent2 := *parent
	parent2.RawToken = svc.Encode(mac2)
	parent2.TokenID = hashToken(mac2.Identifier + mac2.Signature)
	if c := id(&parent2, "worker", "api:read"); c == a {
		t.Fatal("a different parent gave the same id")
	}
}

// Own-credit children are named by their own budget id already; they are unchanged.
func TestOwnCreditChildGetsNoHelperID(t *testing.T) {
	ctx := context.Background()
	svc, d, ev, parent := helperIDRig(t)
	res, err := d.Delegate(ctx, parent, &DelegateParams{Budget: 10, Label: "funded"})
	if err != nil {
		t.Fatal(err)
	}
	mac, err := svc.Verify(res.Token)
	if err != nil {
		t.Fatal(err)
	}
	if ids := tokenIDCaveats(mac); len(ids) != 1 || ids[0] != "parent-row" {
		t.Fatalf("own-credit child token_id caveats = %v, want only the parent's", ids)
	}
	ev.mu.Lock()
	defer ev.mu.Unlock()
	if v, _ := ev.events[0].Data[EventChildRevocationID].(string); v != "" {
		t.Fatalf("own-credit child event carries a helper id %q", v)
	}
}

// With helper ids on, a parent with no token_id caveat of its own cannot have a
// zero-credit helper: no stable id can be derived, so the delegation is refused
// with the fixed sentence and no key. With the option off it mints as before.
func TestHelperOfParentWithoutTokenIDIsRefusedWhenOnAndMintsWhenOff(t *testing.T) {
	ctx := context.Background()
	svc, err := macaroon.NewService("test-root-key")
	if err != nil {
		t.Fatal(err)
	}
	mac, err := svc.Mint("api:*", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	parent := &TokenInfo{TokenID: hashToken(mac.Identifier + mac.Signature), RawToken: svc.Encode(mac)}

	on := NewDelegator(svc, NewInMemoryBudgetEnforcer())
	on.EnableHelperRevocationID(true)
	ev := &captureEvents{}
	on.SetEventPublisher(ev)
	res, err := on.Delegate(ctx, parent, &DelegateParams{Budget: 0, Label: "w"})
	if err == nil || res != nil {
		t.Fatalf("a parent with no token_id gave a helper: res=%v err=%v", res, err)
	}
	if got := delegationClientMessage(err); got != errDelegationNoParentIdentity.msg {
		t.Fatalf("client text = %q, want the fixed sentence", got)
	}
	if len(ev.events) != 0 {
		t.Fatalf("a refused delegation published %d events", len(ev.events))
	}

	off := NewDelegator(svc, NewInMemoryBudgetEnforcer())
	res, err = off.Delegate(ctx, parent, &DelegateParams{Budget: 0, Label: "w"})
	if err != nil {
		t.Fatal(err)
	}
	child, err := svc.Verify(res.Token)
	if err != nil {
		t.Fatal(err)
	}
	if ids := tokenIDCaveats(child); len(ids) != 0 {
		t.Fatalf("token_id caveats = %v, want none", ids)
	}
}

// The MCP verifier accepts a new helper, and the token id it reports for the
// helper is still the hash of the exact key (the row id the gateway stores).
func TestMCPVerifierAcceptsANewZeroCreditHelper(t *testing.T) {
	ctx := context.Background()
	svc, d, _, parent := helperIDRig(t)
	res, err := d.Delegate(ctx, parent, &DelegateParams{Budget: 0, Label: "worker"})
	if err != nil {
		t.Fatal(err)
	}
	info, err := (&MacaroonAuthenticator{Service: svc}).Verify(ctx, res.Token)
	if err != nil {
		t.Fatalf("MCP verifier refused a new zero-credit helper: %v", err)
	}
	if info.TokenID != res.TokenID {
		t.Fatalf("token id %s != result token id %s", info.TokenID, res.TokenID)
	}
	if v := os_getenv("SATGATE_EMIT_HELPER_FIXTURE"); v != "" {
		emitHelperFixture(t, v, res.Token)
	}
}

// Reviewer probe by Astra (astra-cto-review, round 2, public_shadow_test.go),
// kept as a permanent test and changed from "the child has an hlp- id" to the
// design's answer: a holder who appends an empty token_id to a valid parent
// gets a refusal, not a key. The old lookup took the last token_id caveat,
// saw the empty one, and minted a zero-credit child with no stable id.
func TestReviewHolderEmptyTokenIDCannotDisableHelperIdentity(t *testing.T) {
	svc, d, ev, parent := helperIDRig(t)
	changed, e := svc.DelegateWithoutVerify(parent.RawToken, []string{"token_id = "})
	if e != nil {
		t.Fatal(e)
	}
	p, e := (&MacaroonAuthenticator{Service: svc}).Verify(context.Background(), svc.Encode(changed))
	if e != nil {
		t.Fatal(e)
	}
	child, e := d.Delegate(context.Background(), p, &DelegateParams{Label: "holder-empty-token-id"})
	if e == nil || child != nil {
		if child != nil {
			if m, verr := svc.Verify(child.Token); verr == nil {
				for _, c := range m.Caveats {
					if strings.HasPrefix(c, "token_id = hlp-") {
						return // a helper with its id would also close the hole
					}
				}
			}
		}
		t.Fatal("a holder's empty token_id shadow got a zero-credit helper with no stable revocation id")
	}
	if got := delegationClientMessage(e); got != errDelegationNoParentIdentity.msg {
		t.Fatalf("client text = %q, want the fixed sentence", got)
	}
	if len(ev.events) != 0 {
		t.Fatalf("a refused delegation published %d events", len(ev.events))
	}
}

// A later token_id caveat that names something else does not change the id
// either: the helper carries the same id as the unshadowed parent's, because
// the identity is read from the first token_id and the parent's own key hash.
func TestHolderTokenIDAfterTheIssuersDoesNotChangeTheHelperID(t *testing.T) {
	ctx := context.Background()
	svc, d, _, parent := helperIDRig(t)
	changed, err := svc.DelegateWithoutVerify(parent.RawToken, []string{"token_id = something-else"})
	if err != nil {
		t.Fatal(err)
	}
	p, err := (&MacaroonAuthenticator{Service: svc}).Verify(ctx, svc.Encode(changed))
	if err != nil {
		t.Fatal(err)
	}
	shadowed, err := d.Delegate(ctx, p, &DelegateParams{Label: "w", Scope: "api:read"})
	if err != nil {
		t.Fatal(err)
	}
	m, err := svc.Verify(shadowed.Token)
	if err != nil {
		t.Fatal(err)
	}
	ids := tokenIDCaveats(m)
	if len(ids) != 3 || ids[0] != "parent-row" || !strings.HasPrefix(ids[2], HelperRevocationIDPrefix) {
		t.Fatalf("token_id caveats = %v, want parent's first, holder's, then an hlp- id", ids)
	}
}

// A new Delegator has the option off.
func TestHelperRevocationIDIsOffInANewDelegator(t *testing.T) {
	svc, err := macaroon.NewService("test-root-key")
	if err != nil {
		t.Fatal(err)
	}
	if NewDelegator(svc, NewInMemoryBudgetEnforcer()).helperIDs {
		t.Fatal("helper ids are on by default")
	}
}
