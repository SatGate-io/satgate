package mcpserver

// Nothing ages out with both switches on, the store's not-written failure, and
// what a kept call holds:
//
//	age  with both switches on, a stored, never-published prepared row is not
//	     aged out at PendingHorizon: it is published and settled however late,
//	     in ordinary recovery and in the unknown outcome's recovery alike. With
//	     either switch off it ages out exactly as on main.
//	save with either switch off, the store's not-written failure
//	     (ErrPendingNotWritten) takes main's ambiguous path; with both on it is
//	     the definite not-stored save D8 settled.
//	kept the context a kept call carries holds no key, raw token or request.

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
	"weak"

	"github.com/satgate-io/satgate/pkg/macaroon"
)

// ageKeptAndRow moves the original minute of every kept unknown outcome, and of
// its journal row, back by d, as the passage of time would (a process that
// survives a journal outage that long). Nothing else changes.
func ageKeptAndRow(t *testing.T, u *unknownRig, d time.Duration) {
	t.Helper()
	s := u.proxy.every().pending
	s.mu.Lock()
	defer s.mu.Unlock()
	u.b.mu.Lock()
	defer u.b.mu.Unlock()
	if len(s.entries) != 1 {
		t.Fatalf("want one kept unknown outcome, have %d", len(s.entries))
	}
	for _, e := range s.entries {
		w := &e.unknown.work
		w.Record.Minute = w.Record.Minute.Add(-d)
		e.Minute = w.Record.Minute
		raw, err := json.Marshal(*w)
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := u.b.rows[w.Record.ID]; !ok {
			t.Fatal("the save did not land: no journal row to age")
		}
		u.b.rows[w.Record.ID] = raw
	}
}

// forgetKept drops the in-memory retry, as a restart of the process does: only
// the journal row is left, for the host's ordinary recovery (flushDurable).
func forgetKept(u *unknownRig) {
	s := u.proxy.every().pending
	s.mu.Lock()
	var ids []string
	for id := range s.entries {
		ids = append(ids, id)
	}
	s.mu.Unlock()
	for _, id := range ids {
		s.remove(id)
	}
}

// assertPublishedAndSettledOnce: the prepared call's one receipt is published
// (the stored bytes), its row completed (not aged out), its money settled once
// and nothing refused, counted, released, sent or issued.
func assertPublishedAndSettledOnce(t *testing.T, u *unknownRig, c unknownCase, r *Response) {
	t.Helper()
	receipts, completed, cancelled := u.published()
	if receipts != 1 || completed != 1 || cancelled != 0 {
		t.Fatalf("receipts %d completed %d cancelled %d, want 1 1 0 (disposition %s)", receipts, completed, cancelled, u.dispositions())
	}
	if d := u.dispositions(); d != string(PendingCompleted) {
		t.Fatalf("the row ended %q, want %q", d, PendingCompleted)
	}
	u.b.mu.Lock()
	parent, child, debits, funded, reconciled := u.b.parent, u.b.child, len(u.b.debit), len(u.b.funded), len(u.b.reconciled)
	u.b.mu.Unlock()
	want := struct{ parent, child int64 }{100, 0}
	if c.method == MethodSatGateDelegate {
		want = struct{ parent, child int64 }{95, 5}
	}
	if parent != want.parent || child != want.child || reconciled != 1 || debits > 1 || funded > 1 {
		t.Fatalf("settlement parent=%d child=%d debits=%d funded=%d reconciled=%d, want parent=%d child=%d once", parent, child, debits, funded, reconciled, want.parent, want.child)
	}
	if _, _, commits, releases := u.store.counts(); commits != 0 || releases != 0 {
		t.Fatalf("hold: commits %d releases %d, want none (the journal settles it)", commits, releases)
	}
	if len(u.decisions()) != 0 || u.misses() != 0 || u.out(r) != 0 {
		t.Fatalf("signed %v misses %d out %d: a found prepared row is published, nothing else", u.base.signed(), u.misses(), u.out(r))
	}
}

// Nothing ages out, over the actual handlers (satgate/delegate and a priced tools/call,
// Observe and Control, direct, Streamable HTTP and SSE): the journal holds the
// work, its save answer was lost, and the journal answers again 49 hours later.
// The receipt is published and the money settled once, however it is recovered.
func TestHorizon_FoundPreparedIsPublishedAndSettledOnce(t *testing.T) {
	for _, c := range unknownCases()[:4] {
		for _, road := range eaRoads {
			for _, via := range []string{"unknown-recovery", "ordinary-recovery"} {
				t.Run(c.name+"/"+road.name+"/"+via, func(t *testing.T) {
					u, done := newUnknownRig(t, c, true, true, true, road, nil)
					defer done()
					r := c.act(u.gapsRig)
					assertUnknownAnswer(t, u, r)
					ageKeptAndRow(t, u, 49*time.Hour)
					ctx := context.Background()
					if via == "ordinary-recovery" {
						forgetKept(u)
					}
					u.up()
					if via == "ordinary-recovery" {
						u.proxy.every().flushDurable(ctx)
					} else {
						u.proxy.every().flush(ctx)
					}
					assertPublishedAndSettledOnce(t, u, c, r)
					for i := 0; i < 3; i++ {
						u.proxy.FlushPending(ctx)
					}
					assertPublishedAndSettledOnce(t, u, c, r)
					if n := u.kept(); n != 0 {
						t.Fatalf("%d entries still kept", n)
					}
				})
			}
		}
	}
}

// With either switch off a 49-hour-old row ages out exactly as on main (the
// snapshot rows, TestNotStored_SwitchOffBytes, hold the bytes): nothing
// published, counted or settled, the row completed as aged out.
func TestHorizon_EitherSwitchOffAgesOutAsOnMain(t *testing.T) {
	for _, c := range unknownCases()[:4] {
		for _, road := range eaRoads {
			t.Run(c.name+"/"+road.name, func(t *testing.T) {
				u, done := newUnknownRig(t, c, true, true, false, road, nil)
				defer done()
				r := c.act(u.gapsRig)
				if outageEvidence(r)["error"] != "outcome_unknown" {
					t.Fatalf("want main's outcome_unknown: %s", eaShape(r))
				}
				if n := u.kept(); n != 0 {
					t.Fatalf("kept %d entries with a switch off", n)
				}
				ageJournalRows(t, u, 49*time.Hour)
				u.up()
				u.proxy.FlushPending(context.Background())
				u.proxy.FlushPending(context.Background())
				receipts, completed, _ := u.published()
				u.b.mu.Lock()
				settled := len(u.b.reconciled) + len(u.b.debit) + len(u.b.funded)
				u.b.mu.Unlock()
				if receipts != 0 || completed != 1 || u.dispositions() != string(PendingAgedOut) || settled != 0 || u.misses() != 0 || len(u.decisions()) != 0 {
					t.Fatalf("receipts %d completed %d disposition %s settled %d misses %d signed %v: not main's aged-out",
						receipts, completed, u.dispositions(), settled, u.misses(), u.base.signed())
				}
			})
		}
	}
}

// A prepared row 49 hours old whose hold has lapsed (the budget holds nothing
// for it any more) settles by the settlement contract of PendingBudget and
// charges at most once, however recovery is interrupted: the receipt is
// published, a delegation's parent is debited and its child funded once, a
// Recovery carve gets its delivery-unconfirmed reconciliation row, and the row
// completes only when all have landed. With either switch off it ages out.
func TestHorizon_Prepared49hOldRowSettlesByTheContractAndChargesOnce(t *testing.T) {
	steps := map[string][]string{
		"uncharged": {"", "published", "completion", "reconciled"},
		"carve":     {"", "published", "completion", "parent_debited", "child_funded", "reconciled"},
	}
	for _, kind := range []string{"uncharged", "carve"} {
		for _, step := range steps[kind] {
			t.Run(kind+"/crash="+step, func(t *testing.T) {
				b := newP2Backend()
				w := p2Work(kind)
				w.Record.Minute = time.Now().Add(-49 * time.Hour)
				if err := (p2Store{b}).Save(context.Background(), w); err != nil {
					t.Fatal(err)
				}
				p := p2Proxy(b)
				p.every().refusals.Store(true) // both switches on
				b.stop = step
				for i := 0; i < 5; i++ {
					p.FlushPending(context.Background())
				}
				wantParent, wantChild := int64(100), int64(0)
				if kind == "carve" {
					wantParent, wantChild = 90, 10
				}
				b.mu.Lock()
				defer b.mu.Unlock()
				if b.completed[w.Record.ID] != PendingCompleted || !b.receipts["fixture-call"] || !b.reconciled["fixture-call"] ||
					b.parent != wantParent || b.child != wantChild || len(b.debit) > 1 || len(b.funded) > 1 {
					t.Fatalf("disposition %q receipt %v reconciled %v parent=%d child=%d debits=%d funded=%d, want completed, published, reconciled, parent=%d child=%d once",
						b.completed[w.Record.ID], b.receipts["fixture-call"], b.reconciled["fixture-call"], b.parent, b.child, len(b.debit), len(b.funded), wantParent, wantChild)
				}
			})
		}
	}
	// With either switch off, the same row is the final no-op, as on main.
	for _, sw := range []struct {
		name         string
		on, refusals bool
	}{{"sign-only", true, false}, {"refusals-only", false, true}, {"both-off", false, false}} {
		t.Run("switch-off/"+sw.name, func(t *testing.T) {
			b := newP2Backend()
			w := p2Work("carve")
			w.Record.Minute = time.Now().Add(-49 * time.Hour)
			if err := (p2Store{b}).Save(context.Background(), w); err != nil {
				t.Fatal(err)
			}
			p := p2Proxy(b)
			p.every().on.Store(sw.on)
			p.every().refusals.Store(sw.refusals)
			p.every().flushDurable(context.Background())
			b.mu.Lock()
			defer b.mu.Unlock()
			if b.completed[w.Record.ID] != PendingAgedOut || len(b.receipts)+len(b.reconciled)+len(b.debit)+len(b.funded) != 0 || b.parent != 100 || b.child != 0 {
				t.Fatalf("not main's final no-op: disposition %q", b.completed[w.Record.ID])
			}
		})
	}
}

// ---- the not-written failure ----

func TestNotWritten_IsNotStoredAndNotTheOtherWayAround(t *testing.T) {
	err := fixtureNotWritten(errors.New("fixture: pool closed"))
	if !errors.Is(err, ErrPendingNotWritten) || !errors.Is(err, ErrPendingNotStored) || PendingSaveResult(err) != PendingNotStored {
		t.Fatal("a not-written failure must stay a definite not-stored one")
	}
	if errors.Is(ErrPendingNotStored, ErrPendingNotWritten) {
		t.Fatal("the plain not-stored outcome must not be a not-written one")
	}
}

// nwOutcome is everything one call and the journal's later answer leave behind.
type nwOutcome struct {
	Shape                               string
	Signed                              string
	Out                                 int
	Debit                               int64
	Misses, Journaled, Kept             int
	Holds, Commits, Releases, Cancelled int
}

func runNotWritten(t *testing.T, c unknownCase, kind string, resolveUp, sign, refusals bool, road eaRoad) nwOutcome {
	t.Helper()
	u, done := newNotWrittenRig(t, c, kind, resolveUp, sign, refusals, road)
	defer done()
	before := u.remaining()
	r := c.act(u.gapsRig)
	o := nwOutcome{Shape: eaShape(r), Signed: strings.Join(u.base.signed(), ","), Out: u.out(r), Kept: u.kept()}
	u.up()
	u.proxy.FlushPending(context.Background())
	u.proxy.FlushPending(context.Background())
	o.Debit, o.Misses = before-u.remaining(), u.misses()
	u.b.mu.Lock()
	o.Journaled, o.Cancelled = len(u.b.rows), len(u.b.cancelled)
	u.b.mu.Unlock()
	_, o.Holds, o.Commits, o.Releases = u.store.counts()
	return o
}

func notWrittenRoads(c unknownCase) []eaRoad {
	if c.site == "U1" || c.site == "U2" {
		return eaRoads
	}
	return eaRoads[:1]
}

// With either switch off the store's not-written failure reaches no
// answer, hold, release, record or count that main (where it was an unmarked
// error) would not reach; with the journal's answer down it is outcome_unknown
// with the hold left, and with it answering absent it is main's too.
func TestNotWritten_EitherSwitchOffIsMains(t *testing.T) {
	for _, c := range unknownCases() {
		for _, road := range notWrittenRoads(c) {
			for _, resolveUp := range []bool{false, true} {
				for _, sw := range offSwitches[1:] { // sign-only, refusals-only; both-off journals nothing
					name := c.name + "/" + road.name + "/" + sw.name
					if resolveUp {
						name += "/resolve-up"
					}
					t.Run(name, func(t *testing.T) {
						main := runNotWritten(t, c, "plain", resolveUp, sw.sign, sw.refusals, road)
						got := runNotWritten(t, c, "marked", resolveUp, sw.sign, sw.refusals, road)
						if !reflect.DeepEqual(main, got) {
							t.Fatalf("with a switch off the not-written failure differs from main's:\n got %+v\nwant %+v", got, main)
						}
						if !resolveUp && sw.sign && (c.site == "U1" || c.site == "U2") && !strings.Contains(got.Shape, "outcome_unknown") {
							t.Fatalf("want main's outcome_unknown while the journal cannot tell: %s", got.Shape)
						}
						if !resolveUp && sw.sign && c.held && got.Releases != 0 {
							t.Fatalf("the hold was released (%d) while the journal could not tell", got.Releases)
						}
					})
				}
			}
		}
	}
}

// D8, unchanged: with both switches on the same failure is the definite
// not-stored save: it is answered and recorded exactly as a plain
// ErrPendingNotStored, without asking the journal again.
func TestNotWritten_BothOnIsTheDefiniteNotStoredSave(t *testing.T) {
	for _, c := range unknownCases() {
		for _, road := range notWrittenRoads(c) {
			t.Run(c.name+"/"+road.name, func(t *testing.T) {
				want := runNotWritten(t, c, "notstored", false, true, true, road)
				got := runNotWritten(t, c, "marked", false, true, true, road)
				if !reflect.DeepEqual(want, got) {
					t.Fatalf("both on: the not-written failure is not the definite not-stored save:\n got %+v\nwant %+v", got, want)
				}
				if strings.Contains(got.Shape, "outcome_unknown") || got.Cancelled != 0 {
					t.Fatalf("both on: answered unknown or asked the journal (cancelled %d): %s", got.Cancelled, got.Shape)
				}
			})
		}
	}
}

// ---- what a kept call holds ----

// retention walks everything reachable from v (unexported fields, interfaces,
// pointers, maps, slices and the context chain included; closures cannot be
// walked, which is why the weak-pointer test below also runs) and names what
// must not be reachable from a kept call.
type visited struct {
	p uintptr
	t reflect.Type
}

type retention struct {
	markers []string
	seen    map[visited]bool
	hits    []string
}

func (w *retention) walk(v reflect.Value, path string, depth int) {
	if !v.IsValid() || depth > 120 || len(w.hits) > 10 {
		return
	}
	switch v.Kind() {
	case reflect.String:
		for _, m := range w.markers {
			if m != "" && strings.Contains(v.String(), m) {
				w.hits = append(w.hits, path+": a marker string")
			}
		}
	case reflect.Interface:
		if !v.IsNil() {
			w.walk(v.Elem(), path, depth+1)
		}
	case reflect.Pointer:
		if v.IsNil() {
			return
		}
		k := visited{v.Pointer(), v.Type()}
		if w.seen[k] {
			return
		}
		w.seen[k] = true
		switch v.Type().Elem() {
		case reflect.TypeOf(TokenInfo{}):
			ti := v.Elem()
			if ti.FieldByName("RawToken").String() != "" || !ti.FieldByName("Raw").IsNil() {
				w.hits = append(w.hits, path+": a TokenInfo carrying the key")
			}
		case reflect.TypeOf(Request{}), reflect.TypeOf(macaroon.Macaroon{}):
			w.hits = append(w.hits, path+": a "+v.Type().Elem().String())
		}
		w.walk(v.Elem(), path, depth+1)
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			w.walk(v.Field(i), path+"."+v.Type().Field(i).Name, depth+1)
		}
	case reflect.Slice:
		if v.Type().Elem().Kind() == reflect.Uint8 {
			for _, m := range w.markers {
				if m != "" && strings.Contains(string(v.Bytes()), m) {
					w.hits = append(w.hits, path+": a marker in bytes")
				}
			}
			return
		}
		fallthrough
	case reflect.Array:
		for i := 0; i < v.Len(); i++ {
			w.walk(v.Index(i), path+"[]", depth+1)
		}
	case reflect.Map:
		it := v.MapRange()
		for it.Next() {
			w.walk(it.Key(), path+"{k}", depth+1)
			w.walk(it.Value(), path+"{v}", depth+1)
		}
	}
}

func keptEntries(u *unknownRig) []*pendingEntry {
	s := u.proxy.every().pending
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []*pendingEntry
	for _, e := range s.entries {
		out = append(out, e)
	}
	return out
}

func assertRetainsNothingOfTheRequest(t *testing.T, u *unknownRig, markers ...string) {
	t.Helper()
	es := keptEntries(u)
	if len(es) != 1 {
		t.Fatalf("want one kept call, have %d", len(es))
	}
	w := &retention{markers: markers, seen: map[visited]bool{}}
	for _, e := range es {
		w.walk(reflect.ValueOf(e), "entry", 0)
	}
	if len(w.hits) != 0 {
		t.Fatalf("the kept call reaches what it must not hold: %v", w.hits)
	}
}

// The kept Control call of the real handlers (both priced paths, every road)
// reaches no key, raw token or request body: the walk starts at the kept entry.
func TestRetainedRetry_RealHandlersHoldNoKey(t *testing.T) {
	for _, c := range unknownCases()[:4] {
		if c.observe {
			continue
		}
		for _, road := range eaRoads {
			t.Run(c.name+"/"+road.name, func(t *testing.T) {
				u, done := newUnknownRig(t, c, false, true, true, road, nil)
				defer done()
				r := c.act(u.gapsRig)
				assertUnknownAnswer(t, u, r)
				if es := keptEntries(u); len(es) != 1 || es[0].unknown.stopCtx == nil {
					t.Fatal("a Control call's stop context is not kept: the walk would prove nothing")
				}
				assertRetainsNothingOfTheRequest(t, u, u.token)
			})
		}
	}
}

//go:noinline
func keepSyntheticCall(u *unknownRig, mode string, markers []string) (weak.Pointer[TokenInfo], weak.Pointer[macaroon.Macaroon], weak.Pointer[Request]) {
	mac := &macaroon.Macaroon{Identifier: "fixture-identifier", Signature: markers[0]}
	info := &TokenInfo{TokenID: "fixture-token", BudgetID: "fixture-budget", TenantID: "fixture-tenant", Scope: "*", RawToken: markers[0], Raw: mac,
		ArgumentRules: nil, BudgetLimit: 100}
	req := &Request{JSONRPC: "2.0", Method: MethodToolsCall, Params: json.RawMessage(`{"_meta":{"token":"` + markers[0] + `"},"body":"` + markers[2] + `"}`)}
	ctx := context.WithValue(context.Background(), CtxTokenInfo, info)
	ctx = context.WithValue(ctx, CtxTenantID, "fixture-tenant")
	ctx = context.WithValue(ctx, ctxEnforcementMode, mode)
	ctx = withTransportToken(ctx, markers[1])
	ctx = context.WithValue(ctx, delegateRequestKey{}, req)
	prep := &pendingPreparation{work: PendingWork{Version: 1, Receipt: json.RawMessage(`{}`), Record: PendingRecord{
		ID: "fixture-work", RequestID: "fixture-call", Kind: PendingUncharged, TenantID: "fixture-tenant", TokenID: "fixture-token",
		BudgetID: "fixture-budget", Method: MethodToolsCall, Tool: "paid_read", Amount: 1, Minute: RateWindowStart(time.Now()), Since: time.Now().UTC()}}}
	u.proxy.keepUnknownPrepared(ctx, info, MethodToolsCall, "paid_read", mode, prep, func(context.Context) {})
	return weak.Make(info), weak.Make(mac), weak.Make(req)
}

// The synthetic key, connection credential and request body are planted in a
// request context; once the call is kept nothing of them is reachable from the
// kept entry (reflection walk) and the garbage collector frees the original
// TokenInfo, macaroon and request while the call is still kept (which also
// covers what a closure holds, where a walk cannot see). Both modes: Control
// keeps a stop context, Observe a miss to count.
func TestRetainedRetry_HoldsNoKeyConnectionCredentialOrBody(t *testing.T) {
	markers := []string{"fixture-synthetic-key-marker", "fixture-synthetic-conn-marker", "fixture-synthetic-body-marker"}
	for _, mode := range []string{"control", "observe"} {
		t.Run(mode, func(t *testing.T) {
			c := unknownCases()[3] // U2/tools-call/control: its journal is down, so the call is kept
			if mode == "observe" {
				c = unknownCases()[2]
			}
			u, done := newUnknownRig(t, c, false, true, true, eaRoad{name: "direct"}, nil)
			defer done()
			ti, mac, req := keepSyntheticCall(u, mode, markers)
			if mode == "control" {
				if es := keptEntries(u); len(es) != 1 || es[0].unknown.stopCtx == nil {
					t.Fatal("the stop context is not kept")
				}
			}
			assertRetainsNothingOfTheRequest(t, u, markers...)
			for i := 0; i < 5; i++ {
				runtime.GC()
			}
			if ti.Value() != nil || mac.Value() != nil || req.Value() != nil {
				t.Fatalf("the kept call still holds the original: TokenInfo %v macaroon %v request %v", ti.Value() != nil, mac.Value() != nil, req.Value() != nil)
			}
			if n := u.kept(); n != 1 {
				t.Fatalf("the call is not kept any more (%d): the check proves nothing", n)
			}
		})
	}
}

// What a kept stop needs is still there: the identity (without the key), the
// tenant and the mode the call was decided under.
func TestRetainedRetry_KeepsWhatTheStopNeeds(t *testing.T) {
	ctx := context.WithValue(context.Background(), CtxTenantID, "fixture-tenant")
	ctx = context.WithValue(ctx, ctxEnforcementMode, "control")
	ctx = context.WithValue(ctx, CtxTokenInfo, &TokenInfo{RawToken: "fixture-synthetic-key-marker"})
	d := detachedCtx(ctx)
	if TenantFromContext(d) != "fixture-tenant" {
		t.Fatal("the tenant is not kept")
	}
	if m, ok := EnforcementModeFromContext(d); !ok || m != "control" {
		t.Fatal("the mode is not kept")
	}
	if TokenInfoFromContext(d) != nil || d.Done() != nil {
		t.Fatal("a token or a cancellation was kept")
	}
	who := keptIdentity(&TokenInfo{TokenID: "a", BudgetID: "b", BudgetLimit: 7, TenantID: "c", Scope: "d", DelegationDepth: 1, DelegationBudget: 2, Depth: 3, ParentTokenID: "e", RawToken: "fixture-synthetic-key-marker", Raw: &macaroon.Macaroon{}})
	if who.RawToken != "" || who.Raw != nil || who.TokenID != "a" || who.BudgetID != "b" || who.BudgetLimit != 7 || who.TenantID != "c" || who.Scope != "d" || who.DelegationDepth != 1 || who.DelegationBudget != 2 || who.Depth != 3 || who.ParentTokenID != "e" {
		t.Fatalf("not the identity without the key: %+v", who)
	}
}
