package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/satgate-io/satgate/pkg/macaroon"
)

// Round 4: a token that carries delegation_depth = 0 anywhere in its chain
// allows no further handoffs on satgate/delegate, whether the issuer wrote the
// zero or a holder appended it, and whether or not a stored limit is wired.
// /v1/mint/delegate has refused such a token all along; the two routes must
// give the same answer for the same token.

// countingLimitStore is a limitStore that also counts HandoffRecord reads, so a
// test can prove the zero was refused before the stored row was read.
type countingLimitStore struct {
	*limitStore
	reads int32
}

func (s *countingLimitStore) HandoffRecord(ctx context.Context, parent *TokenInfo) (string, int, int, bool, error) {
	atomic.AddInt32(&s.reads, 1)
	return s.limitStore.HandoffRecord(ctx, parent)
}

type zeroDepthRig struct {
	t      *testing.T
	svc    *macaroon.Service
	auth   *MacaroonAuthenticator
	d      *Delegator
	budget *recordingBudget
	store  *countingLimitStore // nil when no store is wired
}

// newZeroDepthRig wires a Delegator the way an embedded gateway does: with a
// HandoffLimiter-capable event consumer (withStore), or with no consumer at all
// (a self-hosted gateway without the database).
func newZeroDepthRig(t *testing.T, withStore bool) *zeroDepthRig {
	t.Helper()
	svc, err := macaroon.NewService("zero-depth-root")
	if err != nil {
		t.Fatal(err)
	}
	rb := &recordingBudget{InMemoryBudgetEnforcer: NewInMemoryBudgetEnforcer()}
	d := NewDelegator(svc, rb)
	r := &zeroDepthRig{t: t, svc: svc, auth: &MacaroonAuthenticator{Service: svc}, d: d, budget: rb}
	if withStore {
		r.store = &countingLimitStore{limitStore: newLimitStore()}
		d.SetEventPublisher(r.store)
	}
	return r
}

// token mints a root token with the issuer's caveats (written before the
// signature, like a policy mint), then appends the holder's (after it).
func (r *zeroDepthRig) token(issuer []string, holder []string, storedLimit *int) *TokenInfo {
	r.t.Helper()
	mac, err := r.svc.Mint("mcp:*", time.Now().Add(time.Hour))
	if err != nil {
		r.t.Fatal(err)
	}
	for _, c := range issuer {
		k, v, _ := strings.Cut(c, " = ")
		mac.AddCaveat(k, v)
	}
	mac.Signature = r.svc.RecalculateSignature(mac)
	tok := r.svc.Encode(mac)
	if len(holder) > 0 {
		tok = appendCaveat(r.t, r.svc, tok, holder...)
	}
	info, err := r.auth.Verify(context.Background(), tok)
	if err != nil {
		r.t.Fatal(err)
	}
	info.TenantID = "tenant-secret-zero"
	if err := r.budget.Initialize(context.Background(), info.BudgetID, 1000); err != nil {
		r.t.Fatal(err)
	}
	if r.store != nil {
		r.store.setRow(info.BudgetID, storedLimit)
	}
	return info
}

func (r *zeroDepthRig) handoff(parent *TokenInfo, budget int64) *Response {
	r.t.Helper()
	raw, _ := json.Marshal(DelegateParams{Budget: budget, Label: "helper"})
	resp, err := r.d.HandleDelegate(context.Background(), &Request{JSONRPC: "2.0", ID: json.RawMessage(`1`), Method: MethodSatGateDelegate, Params: raw}, parent)
	if err != nil || resp == nil {
		r.t.Fatalf("HandleDelegate: %+v %v", resp, err)
	}
	return resp
}

func (r *zeroDepthRig) wantRefusedAndUntouched(parent *TokenInfo, resp *Response, msg string, label string) {
	r.t.Helper()
	if resp.Error == nil {
		r.t.Fatalf("%s: want a refusal, got a result: %s", label, resp.Result)
	}
	if resp.Error.Code != CodeBudgetExhausted || resp.Error.Message != msg {
		r.t.Fatalf("%s: code %d message %q, want code %d message %q", label, resp.Error.Code, resp.Error.Message, CodeBudgetExhausted, msg)
	}
	data, _ := json.Marshal(resp.Error.Data)
	if string(data) != `{"error":"delegation_failed"}` {
		r.t.Fatalf("%s: data %s", label, data)
	}
	if len(r.budget.spends) != 0 {
		r.t.Fatalf("%s: the parent was carved: %v", label, r.budget.spends)
	}
	if len(r.budget.initialized) != 1 {
		r.t.Fatalf("%s: a child budget was created: %v", label, r.budget.initialized)
	}
	if n, _ := r.budget.Remaining(context.Background(), parent.BudgetID); n != 1000 {
		r.t.Fatalf("%s: parent remaining %d, want 1000", label, n)
	}
	if r.store != nil {
		if rows := r.store.children(); len(rows) != 0 {
			r.t.Fatalf("%s: child rows written: %v", label, rows)
		}
		if r.store.events != 0 {
			r.t.Fatalf("%s: %d events published for a refused handoff", label, r.store.events)
		}
	}
}

// DEFECT TEST: the same token, with and without a stored limit wired, with and
// without a stored row for it, is refused before any read, mint or carve.
func TestDelegateRefusesZeroDelegationDepthWithAndWithoutStore(t *testing.T) {
	cases := []struct {
		name   string
		issuer []string
		holder []string
	}{
		{"issuer wrote zero", []string{"delegation_depth = 0"}, nil},
		{"holder appended zero to an unlimited token", nil, []string{"delegation_depth = 0"}},
		{"holder appended zero under a cap of 2", []string{"delegation_depth = 2"}, []string{"delegation_depth = 0"}},
		{"zero then a negative value", nil, []string{"delegation_depth = 0", "delegation_depth = -1"}},
		{"zero then a wider value", nil, []string{"delegation_depth = 0", "delegation_depth = 9"}},
		{"zero after a wider value", []string{"delegation_depth = 9"}, []string{"delegation_depth = 0"}},
		{"zero written +0", nil, []string{"delegation_depth = +0"}},
		{"zero written 00", nil, []string{"delegation_depth = 00"}},
	}
	for _, withStore := range []bool{false, true} {
		for _, tc := range cases {
			name := tc.name + " / no store"
			if withStore {
				name = tc.name + " / store"
			}
			t.Run(name, func(t *testing.T) {
				for _, budget := range []int64{0, 25} {
					r := newZeroDepthRig(t, withStore)
					// A stored row that would permit it (NULL = unlimited).
					parent := r.token(tc.issuer, tc.holder, nil)
					resp := r.handoff(parent, budget)
					r.wantRefusedAndUntouched(parent, resp, wantNoHandoff, name)
					if r.store != nil && atomic.LoadInt32(&r.store.reads) != 0 {
						t.Fatalf("%s: the stored row was read %d times before the zero was refused", name, r.store.reads)
					}
				}
			})
		}
	}
}

// DEFECT TEST: the refusal logs one line that says the token allows no further
// handoffs and names neither the token nor the tenant.
func TestDelegateZeroDepthRefusalLogsOneLineWithoutIdentity(t *testing.T) {
	var buf bytes.Buffer
	prev := log.Logger
	log.Logger = zerolog.New(&buf)
	defer func() { log.Logger = prev }()

	r := newZeroDepthRig(t, false)
	parent := r.token(nil, []string{"delegation_depth = 0"}, nil)
	r.handoff(parent, 25)

	out := buf.String()
	if n := strings.Count(out, "allows no further handoffs"); n != 1 {
		t.Fatalf("want exactly one line saying the token allows no further handoffs, got %d in:\n%s", n, out)
	}
	for _, secret := range []string{parent.RawToken, parent.TenantID, parent.TokenID, parent.BudgetID} {
		if secret != "" && strings.Contains(out, secret) {
			t.Fatalf("the refusal log carries %q:\n%s", secret, out)
		}
	}
}

// CONTROL: a positive cap, and a token with no delegation_depth caveat, keep
// today's behaviour, with and without a stored limit.
func TestDelegatePositiveAndAbsentDepthKeepTodaysBehaviour(t *testing.T) {
	for _, withStore := range []bool{false, true} {
		// No caveat: a child and a grandchild.
		r := newZeroDepthRig(t, withStore)
		parent := r.token(nil, nil, nil)
		child := r.childOf(parent)
		_ = r.childOf(child)

		// A cap of 2: two levels, then today's depth-limit refusal.
		r = newZeroDepthRig(t, withStore)
		parent = r.token([]string{"delegation_depth = 2"}, nil, intp(2))
		child = r.childOf(parent)
		grandchild := r.childOf(child)
		// Today's answers: the stored row decides first when one is wired.
		want := "delegation depth limit reached (2)"
		if withStore {
			want = "this token has reached its handoff limit (2 levels)"
		}
		resp := r.handoff(grandchild, 5)
		if resp.Error == nil || resp.Error.Code != CodeBudgetExhausted || resp.Error.Message != want {
			t.Fatalf("store=%v: the third handoff under a cap of 2: %+v, want %q", withStore, resp.Error, want)
		}
	}
}

// childOf makes a child through the real handler and reads it back through the
// real authenticator, as the next caller.
func (r *zeroDepthRig) childOf(parent *TokenInfo) *TokenInfo {
	r.t.Helper()
	resp := r.handoff(parent, 10)
	if resp.Error != nil {
		r.t.Fatalf("handoff refused: %d %q", resp.Error.Code, resp.Error.Message)
	}
	var res DelegateResult
	if err := json.Unmarshal(resp.Result, &res); err != nil {
		r.t.Fatal(err)
	}
	info, err := r.auth.Verify(context.Background(), res.Token)
	if err != nil {
		r.t.Fatal(err)
	}
	info.TenantID = parent.TenantID
	return info
}
