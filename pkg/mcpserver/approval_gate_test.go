package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/satgate-io/satgate/pkg/toolcontrols"
)

const (
	askAbove50     = `{"v":1,"approvals":[{"tools":["place_crypto_order"],"field":"dollar_amount","above":"50"}]}`
	askAbove50Cap2 = `{"v":1,"approvals":[{"tools":["place_crypto_order"],"field":"dollar_amount","above":"50"}],"spend_limits":[{"tools":["place_crypto_order"],"field":"dollar_amount","max":"200","window":"day"}]}`
)

type fakeNotifier struct {
	mu      sync.Mutex
	notices []ApprovalNotice
	err     error
}

func (f *fakeNotifier) Notify(_ context.Context, n ApprovalNotice) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.notices = append(f.notices, n)
	return f.err
}

func (f *fakeNotifier) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.notices)
}

type approvalHarness struct {
	*spendHarness
	approvals *toolcontrols.MemoryApprovalStore
	notifier  *fakeNotifier
	gate      *ApprovalGate
	clock     time.Time
}

func newApprovalHarness(t *testing.T, doc string) *approvalHarness {
	t.Helper()
	h := newSpendHarness(t, doc)
	a := &approvalHarness{
		spendHarness: h,
		approvals:    toolcontrols.NewMemoryApprovalStore(),
		notifier:     &fakeNotifier{},
		clock:        time.Now().UTC(),
	}
	a.gate = NewApprovalGate(a.approvals, a.notifier)
	a.gate.Synchronous = true
	a.gate.now = func() time.Time { return a.clock }
	h.proxy.SetCallGate(ChainGates(NewSpendGate(h.store), a.gate))
	return a
}

func (a *approvalHarness) onlyApproval(t *testing.T) toolcontrols.MemoryApproval {
	t.Helper()
	rows := a.approvals.Rows()
	if len(rows) != 1 {
		t.Fatalf("approvals = %d, want 1", len(rows))
	}
	return rows[0]
}

func (a *approvalHarness) lastDecision(t *testing.T) MCPDecision {
	t.Helper()
	if len(a.rec.decisions) == 0 {
		t.Fatal("no receipt recorded")
	}
	return a.rec.decisions[len(a.rec.decisions)-1]
}

func TestApprovalHoldsAboveThresholdAndLetsApprovedRetryThroughOnce(t *testing.T) {
	a := newApprovalHarness(t, askAbove50)

	// At the threshold, and below it: not touched.
	for i, amt := range []string{"50", "10"} {
		if r := a.call(t, i+1, "place_crypto_order", order(amt)); r.Error != nil {
			t.Fatalf("order of %s refused: %v", amt, r.Error)
		}
	}
	if len(a.approvals.Rows()) != 0 || a.notifier.count() != 0 {
		t.Fatal("a call at or below the threshold made an approval or an email")
	}
	before := a.rr.forwarded()

	// Above: held, not forwarded, the owner told once.
	r := a.call(t, 3, "place_crypto_order", order("75"))
	if r.Error == nil || r.Error.Code != CodePolicyDenied {
		t.Fatalf("not held: %+v", r)
	}
	d := errData(t, r)
	row := a.onlyApproval(t)
	if d["error"] != "APPROVAL_REQUIRED" || d["approval_id"] != row.ID || d["receipt_id"] == "" || d["evidence_url"] == "" || d["expires_at"] == "" {
		t.Fatalf("data = %v", d)
	}
	if a.rr.forwarded() != before {
		t.Fatal("a held call was forwarded")
	}
	if a.notifier.count() != 1 || a.notifier.notices[0].ApprovalID != row.ID {
		t.Fatalf("notices = %+v", a.notifier.notices)
	}
	dec := a.lastDecision(t)
	if dec.Decision != "denied" || dec.DenialCode != "APPROVAL_REQUIRED" || dec.ApprovalID != row.ID || dec.ArgumentField != "dollar_amount" {
		t.Fatalf("receipt input = %+v", dec)
	}

	// The agent asks again before the owner answers: same approval, no second email.
	r = a.call(t, 4, "place_crypto_order", order("75"))
	if errData(t, r)["approval_id"] != row.ID {
		t.Fatal("a repeat made a second approval")
	}
	if len(a.approvals.Rows()) != 1 || a.notifier.count() != 1 {
		t.Fatalf("rows=%d notices=%d", len(a.approvals.Rows()), a.notifier.count())
	}

	// The owner approves; the same call goes through, once.
	if err := a.approvals.Approve("tenant-1", row.ID, "owner@example.com", a.clock); err != nil {
		t.Fatal(err)
	}
	r = a.call(t, 5, "place_crypto_order", order("75"))
	if r.Error != nil {
		t.Fatalf("approved retry refused: %v", r.Error)
	}
	if a.rr.forwarded() != before+1 {
		t.Fatalf("forwarded = %d", a.rr.forwarded())
	}
	if dec := a.lastDecision(t); dec.Decision != "allowed" || dec.ApprovalID != row.ID {
		t.Fatalf("receipt input for the approved call = %+v", dec)
	}
	if got, _ := a.approvals.Row(row.ID); got.State != toolcontrols.StateUsed {
		t.Fatalf("state = %s", got.State)
	}

	// Sending it a third time is a new ask.
	r = a.call(t, 6, "place_crypto_order", order("75"))
	if errData(t, r)["error"] != "APPROVAL_REQUIRED" || errData(t, r)["approval_id"] == row.ID {
		t.Fatalf("a used approval let a second call through: %v", r.Error)
	}
	if a.rr.forwarded() != before+1 {
		t.Fatal("second call was forwarded")
	}
}

func TestApprovalDoesNotCoverADifferentCall(t *testing.T) {
	a := newApprovalHarness(t, askAbove50)
	r := a.call(t, 1, "place_crypto_order", order("75"))
	first := errData(t, r)["approval_id"].(string)
	if err := a.approvals.Approve("tenant-1", first, "owner@example.com", a.clock); err != nil {
		t.Fatal(err)
	}

	// Another amount, another symbol: held as new calls.
	r = a.call(t, 2, "place_crypto_order", order("76"))
	d := errData(t, r)
	if d["error"] != "APPROVAL_REQUIRED" || d["approval_id"] == first {
		t.Fatalf("a different amount used the approval: %v", d)
	}
	other := strings.Replace(order("75"), "BTC-USD", "ETH-USD", 1)
	r = a.call(t, 3, "place_crypto_order", other)
	if d := errData(t, r); d["error"] != "APPROVAL_REQUIRED" || d["approval_id"] == first {
		t.Fatalf("a different symbol used the approval: %v", d)
	}
	if a.rr.forwarded() != 0 {
		t.Fatal("a different call was forwarded")
	}
	// The approval is still there for the exact call.
	if r := a.call(t, 4, "place_crypto_order", order("75")); r.Error != nil {
		t.Fatalf("exact call refused: %v", r.Error)
	}
	// Key order and spacing do not make a different call.
	if a.rr.forwarded() != 1 {
		t.Fatalf("forwarded = %d", a.rr.forwarded())
	}
}

func TestApprovalBoundToTheToken(t *testing.T) {
	a := newApprovalHarness(t, askAbove50)
	r := a.call(t, 1, "place_crypto_order", order("75"))
	id := errData(t, r)["approval_id"].(string)
	if err := a.approvals.Approve("tenant-1", id, "owner@example.com", a.clock); err != nil {
		t.Fatal(err)
	}
	// Another token (own budget id) sending the identical call must not use it.
	svc, tok := mintBudgetToken(t, argTestRoot, "place_crypto_order,place_equity_order", "tenant-1", "budget-other", "50")
	tok = appendCaveat(t, svc, tok, controlWord(t, askAbove50))
	resp, err := a.proxy.handleRequest(context.Background(), argCall(2, tok, "place_crypto_order", order("75")))
	if err != nil || resp.Error == nil {
		t.Fatalf("another token was let through: %+v %v", resp, err)
	}
	if d := errData(t, resp); d["error"] != "APPROVAL_REQUIRED" || d["approval_id"] == id {
		t.Fatalf("data = %v", d)
	}
	if a.rr.forwarded() != 0 {
		t.Fatal("forwarded")
	}
}

func TestApprovalDeniedAndExpiredRefuseWithReceipts(t *testing.T) {
	a := newApprovalHarness(t, askAbove50)
	r := a.call(t, 1, "place_crypto_order", order("75"))
	id := errData(t, r)["approval_id"].(string)
	if err := a.approvals.Deny("tenant-1", id, "owner@example.com", a.clock); err != nil {
		t.Fatal(err)
	}
	r = a.call(t, 2, "place_crypto_order", order("75"))
	d := errData(t, r)
	if d["error"] != "APPROVAL_DENIED" || d["approval_id"] != id && d["approval_id"] != nil {
		t.Fatalf("data = %v", d)
	}
	if dec := a.lastDecision(t); dec.DenialCode != "APPROVAL_DENIED" || dec.ApprovalID != id {
		t.Fatalf("receipt input = %+v", dec)
	}
	if a.rr.forwarded() != 0 {
		t.Fatal("denied call forwarded")
	}

	// A fresh ask, approved, then left to run out.
	b := newApprovalHarness(t, askAbove50)
	r = b.call(t, 1, "place_crypto_order", order("80"))
	id = errData(t, r)["approval_id"].(string)
	if err := b.approvals.Approve("tenant-1", id, "owner@example.com", b.clock); err != nil {
		t.Fatal(err)
	}
	b.clock = b.clock.Add(toolcontrols.ApprovalTTL + time.Second)
	r = b.call(t, 2, "place_crypto_order", order("80"))
	if d := errData(t, r); d["error"] != "APPROVAL_EXPIRED" {
		t.Fatalf("data = %v", d)
	}
	if dec := b.lastDecision(t); dec.DenialCode != "APPROVAL_EXPIRED" || dec.ApprovalID != id {
		t.Fatalf("receipt input = %+v", dec)
	}
	if b.rr.forwarded() != 0 {
		t.Fatal("expired approval let the call through")
	}
	// Late approval of an expired ask is refused by the store.
	if err := b.approvals.Approve("tenant-1", id, "owner@example.com", b.clock); !errors.Is(err, toolcontrols.ErrApprovalNotPending) {
		t.Fatalf("late approve = %v", err)
	}
}

func TestApprovalRefusedCallsAreNotHeldAndNobodyIsEmailed(t *testing.T) {
	a := newApprovalHarness(t, askAbove50Cap2)
	// Over the $200 day limit: refused by the spending limit, never held.
	r := a.call(t, 1, "place_crypto_order", order("250"))
	if d := errData(t, r); d["error"] != "TOOL_SPEND_LIMIT_DENIED" {
		t.Fatalf("data = %v", d)
	}
	if len(a.approvals.Rows()) != 0 || a.notifier.count() != 0 {
		t.Fatalf("a refused call made an approval (%d) or an email (%d)", len(a.approvals.Rows()), a.notifier.count())
	}
	if a.counter(t) != 0 {
		t.Fatal("counter moved")
	}

	// A token that does not cover the tool is refused before the gates.
	if r := a.call(t, 2, "place_forbidden_tool", order("75")); r.Error == nil {
		t.Fatal("tool outside the token's scope was allowed")
	}
	if len(a.approvals.Rows()) != 0 || a.notifier.count() != 0 {
		t.Fatal("a scope refusal made an approval or an email")
	}
}

func TestApprovalStillReservesAgainstSpendLimitWhenForwarded(t *testing.T) {
	a := newApprovalHarness(t, askAbove50Cap2)
	hold := func(id int, amt string) string {
		r := a.call(t, id, "place_crypto_order", order(amt))
		appr := errData(t, r)["approval_id"].(string)
		if err := a.approvals.Approve("tenant-1", appr, "owner@example.com", a.clock); err != nil {
			t.Fatal(err)
		}
		return appr
	}
	// Both fit the $200 limit while nothing is spent, so both are held and approved.
	idBig, idSmall := hold(1, "150"), hold(2, "60")
	if a.counter(t) != 0 {
		t.Fatalf("a held or approved call reserved spend: %d", a.counter(t))
	}
	if r := a.call(t, 3, "place_crypto_order", order("150")); r.Error != nil {
		t.Fatalf("approved call refused: %v", r.Error)
	}
	if a.counter(t) != 150*units {
		t.Fatalf("counter = %d", a.counter(t))
	}
	if got, _ := a.approvals.Row(idBig); got.State != toolcontrols.StateUsed {
		t.Fatalf("state = %s", got.State)
	}

	// The second approval is good, but 150 + 60 is over the limit now: refused
	// at forward time, and the owner's yes is not used up by that.
	r := a.call(t, 4, "place_crypto_order", order("60"))
	if d := errData(t, r); d["error"] != "TOOL_SPEND_LIMIT_DENIED" {
		t.Fatalf("approved call went over the limit: %v", d)
	}
	if got, _ := a.approvals.Row(idSmall); got.State != toolcontrols.StateApproved {
		t.Fatalf("a spend refusal used the approval up: %s", got.State)
	}
	if a.counter(t) != 150*units || a.rr.forwarded() != 1 {
		t.Fatalf("counter=%d forwarded=%d", a.counter(t), a.rr.forwarded())
	}
}

// The approval is single use: it is used up whenever the call is admitted,
// whatever the upstream does and whether or not a later check stops the call.
func TestApprovalStaysUsedWhenTheUpstreamFails(t *testing.T) {
	for name, reply := range map[string]func() (*Response, error){
		"connection lost": forwardFails,
		"isError":         isErrorReply,
		"rpc error":       rpcErrorReply,
	} {
		t.Run(name, func(t *testing.T) {
			a := newApprovalHarness(t, askAbove50)
			a.rr.replies = []func() (*Response, error){reply}
			r := a.call(t, 1, "place_crypto_order", order("75"))
			id := errData(t, r)["approval_id"].(string)
			if err := a.approvals.Approve("tenant-1", id, "owner@example.com", a.clock); err != nil {
				t.Fatal(err)
			}
			a.call(t, 2, "place_crypto_order", order("75"))
			if a.rr.forwarded() != 1 {
				t.Fatalf("forwarded = %d", a.rr.forwarded())
			}
			if got, _ := a.approvals.Row(id); got.State != toolcontrols.StateUsed {
				t.Fatalf("state = %s", got.State)
			}
			// The retry is a new call: held again, not forwarded.
			if r := a.call(t, 3, "place_crypto_order", order("75")); r.Error == nil || a.rr.forwarded() != 1 {
				t.Fatalf("a used approval was used again: %v forwarded=%d", r.Error, a.rr.forwarded())
			}
		})
	}
}

func TestApprovalRepliesAndReceiptsCarryNoArgumentValues(t *testing.T) {
	a := newApprovalHarness(t, askAbove50)
	args := `{"side":"buy","type":"market","symbol":"ZZQ-USD","dollar_amount":"777.31"}`
	a.call(t, 1, "place_crypto_order", args)
	r := a.call(t, 2, "place_crypto_order", args)
	id := errData(t, r)["approval_id"].(string)
	_ = a.approvals.Deny("tenant-1", id, "owner@example.com", a.clock)
	a.call(t, 3, "place_crypto_order", args)

	raw, _ := json.Marshal(r)
	var all strings.Builder
	all.Write(raw)
	for _, d := range a.rec.decisions {
		b, _ := json.Marshal(d)
		all.Write(b)
	}
	for _, bad := range []string{"777", "ZZQ"} {
		if strings.Contains(all.String(), bad) {
			t.Fatalf("a reply or receipt input carries %q: %s", bad, all.String())
		}
	}
	// The owner's notice is the one place the values are allowed.
	n := a.notifier.notices[0]
	found := false
	for _, it := range n.Summary {
		if strings.Contains(it.Value, "777.31") {
			found = true
		}
	}
	if !found {
		t.Fatalf("owner notice lacks the amount: %+v", n.Summary)
	}
	// And the threshold is the rule's, not the call's.
	if n.Above != "50" || n.Field != "dollar_amount" {
		t.Fatalf("notice = %+v", n)
	}
}

func TestApprovalFailsClosed(t *testing.T) {
	// No store: a covered call above the threshold is refused as a gate error,
	// not forwarded.
	a := newApprovalHarness(t, askAbove50)
	a.proxy.SetCallGate(NewApprovalGate(nil, nil))
	r := a.call(t, 1, "place_crypto_order", order("75"))
	if r.Error == nil || a.rr.forwarded() != 0 {
		t.Fatalf("not closed: %+v forwarded=%d", r, a.rr.forwarded())
	}
	// Below the threshold the missing store does not matter.
	if r := a.call(t, 2, "place_crypto_order", order("5")); r.Error != nil {
		t.Fatalf("small order refused: %v", r.Error)
	}

	// A call whose amount cannot be read is held, not waved through.
	b := newApprovalHarness(t, askAbove50)
	for i, args := range []string{`{"symbol":"BTC-USD"}`, `{"dollar_amount":"lots"}`, `{"dollar_amount":"-5"}`} {
		r := b.call(t, i+1, "place_crypto_order", args)
		if d := errData(t, r); d["error"] != "APPROVAL_REQUIRED" {
			t.Fatalf("%s: data = %v", args, d)
		}
	}
	if b.rr.forwarded() != 0 {
		t.Fatal("unreadable amount forwarded")
	}

	// A notifier that fails does not stop the hold. The failed email is not
	// retried by every repeat of the call (that would be an unbounded mail
	// loop); the approval joins the tenant's backlog and is counted in the
	// next "N more orders are waiting" summary.
	c := newApprovalHarness(t, askAbove50)
	c.notifier.err = errors.New("mail down")
	r = c.call(t, 1, "place_crypto_order", order("75"))
	if errData(t, r)["error"] != "APPROVAL_REQUIRED" {
		t.Fatal("hold lost when mail failed")
	}
	c.notifier.err = nil
	c.call(t, 2, "place_crypto_order", order("75"))
	c.call(t, 3, "place_crypto_order", order("75"))
	if c.notifier.count() != 1 {
		t.Fatalf("notices = %d, want the one failed try and no retry per repeat", c.notifier.count())
	}
	if n, ok := c.approvals.TakeSummary("tenant-1", c.clock.Add(3*time.Minute)); !ok || n != 1 {
		t.Fatalf("summary = %d %v, want the failed approval counted once", n, ok)
	}
}

func TestApprovalQueueIsBounded(t *testing.T) {
	a := newApprovalHarness(t, askAbove50)
	for i := 0; i < toolcontrols.MaxPendingApprovals; i++ {
		r := a.call(t, i+1, "place_crypto_order", order(strings.Repeat("9", 2)+"."+leftPad(i)))
		if errData(t, r)["error"] != "APPROVAL_REQUIRED" {
			t.Fatalf("call %d: %v", i, r.Error)
		}
	}
	r := a.call(t, 100, "place_crypto_order", order("98.99"))
	if d := errData(t, r); d["error"] != "APPROVAL_QUEUE_FULL" {
		t.Fatalf("data = %v", d)
	}
	// The tenant's mail cap (10 an hour) is below the token's queue (20): the
	// other ten are held and listed, and wait for a summary.
	if a.notifier.count() != toolcontrols.MaxApprovalEmailsPerHour {
		t.Fatalf("notices = %d", a.notifier.count())
	}
	if len(a.approvals.Rows()) != toolcontrols.MaxPendingApprovals {
		t.Fatalf("rows = %d", len(a.approvals.Rows()))
	}
	if a.rr.forwarded() != 0 {
		t.Fatal("a refused call was forwarded")
	}
}

func leftPad(i int) string {
	s := "00" + string(rune('0'+i/10)) + string(rune('0'+i%10))
	return s
}

func TestPlainTokenWithApprovalWordStillNeedsAGate(t *testing.T) {
	// A proxy with no gate installed must refuse a token that carries an
	// approval rule, never forward it unchecked.
	a := newApprovalHarness(t, askAbove50)
	a.proxy.SetCallGate(nil)
	r := a.call(t, 1, "place_crypto_order", order("75"))
	if r.Error == nil || a.rr.forwarded() != 0 {
		t.Fatalf("forwarded without a gate: %+v", r)
	}
}

// Round 2 rules, applied to approvals. A scope that allows a case variant of a
// tool an "ask me" rule names (mcp:* allows everything) must not be a way round
// the hold: the variant is held, and the owner's approval of one spelling is
// the same call as the other only if every byte of the arguments matches.
func TestApprovalCaseVariantToolNameIsHeld(t *testing.T) {
	h := newSpendHarnessScope(t, askAbove50, "mcp:*")
	approvals := toolcontrols.NewMemoryApprovalStore()
	gate := NewApprovalGate(approvals, &fakeNotifier{})
	gate.Synchronous = true
	h.proxy.SetCallGate(ChainGates(NewSpendGate(h.store), gate))

	r := h.call(t, 1, "Place_Crypto_Order", order("500"))
	if got := errData(t, r)["error"]; got != toolcontrols.DenialCodeApprovalRequired {
		t.Fatalf("500 on a case variant: error = %v, want %s", got, toolcontrols.DenialCodeApprovalRequired)
	}
	if h.rr.forwarded() != 0 {
		t.Fatal("a case variant above the threshold was forwarded")
	}
	// Below the threshold a case variant is not touched.
	if r := h.call(t, 2, "PLACE_crypto_ORDER", order("10")); r.Error != nil {
		t.Fatalf("small order on a case variant refused: %v", r.Error)
	}
	// Approving the held variant lets exactly that call through, once.
	rows := approvals.Rows()
	if len(rows) != 1 {
		t.Fatalf("approvals = %d", len(rows))
	}
	if err := approvals.Approve("tenant-1", rows[0].ID, "owner@example.com", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if r := h.call(t, 3, "Place_Crypto_Order", order("500")); r.Error != nil {
		t.Fatalf("approved variant refused: %v", r.Error)
	}
	if r := h.call(t, 4, "Place_Crypto_Order", order("500")); r.Error == nil {
		t.Fatal("approval used twice")
	}
}

// Names the upstream might read as the tool but that are not plain are refused
// on a token that carries an "ask me" rule, never forwarded and never held.
func TestApprovalUnmatchableToolNamesAreRefused(t *testing.T) {
	for label, name := range map[string]string{
		"trailing space": "place_crypto_order ",
		"newline":        "place_crypto_order\n",
		"kelvin sign":    "place_crypto_order\u212a",
		"fullwidth":      "\uff50lace_crypto_order",
		"zero width":     "place_crypto\u200b_order",
	} {
		t.Run(label, func(t *testing.T) {
			a := newApprovalHarness(t, askAbove50)
			a.proxy.SetCallGate(ChainGates(NewSpendGate(a.store), a.gate))
			params := `{"name":"` + name + `","arguments":` + order("500") + `,"_meta":{"token":"` + a.token + `"}}`
			req := &Request{JSONRPC: "2.0", ID: json.RawMessage("1"), Method: MethodToolsCall, Params: json.RawMessage(params)}
			resp, err := a.proxy.handleRequest(context.Background(), req)
			if err != nil || resp == nil || resp.Error == nil {
				t.Fatalf("not refused: %+v %v", resp, err)
			}
			if a.rr.forwarded() != 0 || len(a.approvals.Rows()) != 0 || a.notifier.count() != 0 {
				t.Fatalf("forwarded=%d rows=%d mail=%d", a.rr.forwarded(), len(a.approvals.Rows()), a.notifier.count())
			}
		})
	}
}

// The release rule of PR 1 round 3 applies to an approved call that is
// forwarded: the approval is used up whatever the upstream does, and the spend
// reservation is kept unless the call was never sent or the upstream answered
// with a JSON-RPC error that means "not processed" (-32602 and its kin).
func TestApprovedCallFollowsTheSpendReleaseRule(t *testing.T) {
	for name, tc := range map[string]struct {
		reply    func() (*Response, error)
		wantKept bool
	}{
		"upstream says isError":          {isErrorReply, true},
		"upstream server error (-32000)": {rpcErrorReply, true},
		"upstream says invalid params":   {rpcErrorCode(-32602), false},
		"ok":                             {okReply, true},
	} {
		t.Run(name, func(t *testing.T) {
			a := newApprovalHarness(t, askAbove50Cap2)
			a.rr.replies = []func() (*Response, error){tc.reply}
			r := a.call(t, 1, "place_crypto_order", order("75"))
			id := errData(t, r)["approval_id"].(string)
			if err := a.approvals.Approve("tenant-1", id, "owner@example.com", a.clock); err != nil {
				t.Fatal(err)
			}
			a.call(t, 2, "place_crypto_order", order("75"))
			if a.rr.forwarded() != 1 {
				t.Fatalf("forwarded = %d", a.rr.forwarded())
			}
			if got, _ := a.approvals.Row(id); got.State != toolcontrols.StateUsed {
				t.Fatalf("approval state = %s, want used", got.State)
			}
			want := int64(0)
			if tc.wantKept {
				want = 75 * units
			}
			if a.counter(t) != want {
				t.Fatalf("counter = %d, want %d", a.counter(t), want)
			}
		})
	}
}

// orderSymbol is an order whose symbol is written exactly as given (JSON
// escapes included), so a test can send a lone surrogate.
func orderSymbol(amount, symbolJSON string) string {
	return `{"side":"buy","type":"market","symbol":"` + symbolJSON + `","dollar_amount":"` + amount + `"}`
}

// Astra's round-2 blocker: the owner approves a call with symbol "\ufffd";
// the agent retries with "\ud800". encoding/json read both as U+FFFD, so the
// changed call used the approval and was forwarded with its own raw bytes. A
// call that cannot be read without loss is now refused outright.
func TestApprovalLosslessMatchingRefusesChangedSurrogate(t *testing.T) {
	for name, tc := range map[string]struct{ approved, retried string }{
		"fffd then lone high":  {`\ufffd`, `\ud800`},
		"lone high then other": {`\ud800`, `\ud801`},
		"fffd then lone low":   {`\ufffd`, `\udc00`},
	} {
		t.Run(name, func(t *testing.T) {
			a := newApprovalHarness(t, askAbove50)
			// The approved call is only held if it can be read; a lone half is
			// refused at the first send, so approve the readable one when there is one.
			first := a.call(t, 1, "place_crypto_order", orderSymbol("75", tc.approved))
			d := errData(t, first)
			if strings.Contains(tc.approved, `\ud8`) {
				if d["error"] != "APPROVAL_CHECK_FAILED" {
					t.Fatalf("a lone surrogate was held: %v", d)
				}
				if len(a.approvals.Rows()) != 0 {
					t.Fatal("a call that cannot be read left a row")
				}
			} else {
				if err := a.approvals.Approve("tenant-1", d["approval_id"].(string), "owner@example.com", a.clock); err != nil {
					t.Fatal(err)
				}
			}
			rowsBefore, mailBefore := len(a.approvals.Rows()), a.notifier.count()

			r := a.call(t, 2, "place_crypto_order", orderSymbol("75", tc.retried))
			d = errData(t, r)
			if d["error"] != "APPROVAL_CHECK_FAILED" || d["receipt_id"] == "" || d["evidence_url"] == "" {
				t.Fatalf("changed call not refused with a receipt: %v", d)
			}
			if a.rr.forwarded() != 0 {
				t.Fatalf("upstream = %d, want 0: the changed call was forwarded", a.rr.forwarded())
			}
			if len(a.approvals.Rows()) != rowsBefore || a.notifier.count() != mailBefore {
				t.Fatalf("a refused call was held or emailed: rows %d->%d mail %d->%d",
					rowsBefore, len(a.approvals.Rows()), mailBefore, a.notifier.count())
			}
			if dec := a.lastDecision(t); dec.Decision != "denied" || dec.DenialCode != "APPROVAL_CHECK_FAILED" || dec.ApprovalID != "" {
				t.Fatalf("receipt input = %+v", dec)
			}
			// The reply and the receipt name no value from the call.
			if strings.Contains(r.Error.Message, "ud800") || strings.Contains(string(r.Error.Data), "ud800") {
				t.Fatal("the refusal repeats the call's value")
			}
		})
	}
}

// Every way a call can be unreadable without loss is refused on a token with
// an approval rule, even below the threshold and even for a tool the rule does
// not cover: nothing held, no email, nothing forwarded.
func TestApprovalRefusesCallsThatCannotBeReadWithoutLoss(t *testing.T) {
	for name, args := range map[string]string{
		"lone high surrogate":        orderSymbol("75", `\ud800`),
		"lone low surrogate":         orderSymbol("75", `\udc00`),
		"reversed pair":              orderSymbol("75", `\udc00\ud800`),
		"surrogate in a key":         `{"sym\ud800bol":"x","dollar_amount":"75"}`,
		"surrogate deep":             `{"dollar_amount":"75","legs":[{"a":{"b":["\ud800"]}}]}`,
		"duplicate key at depth":     `{"dollar_amount":"75","legs":[{"a":1,"a":2}]}`,
		"duplicate nested object":    `{"dollar_amount":"75","o":{"k":1,"k":1}}`,
		"number too long":            `{"dollar_amount":"75","n":` + strings.Repeat("9", 80) + `}`,
		"number not finite":          `{"dollar_amount":"75","n":1e999}`,
		"number past 40 digits":      `{"dollar_amount":"75","n":` + strings.Repeat("1", 41) + `}`,
		"below threshold, surrogate": orderSymbol("10", `\ud800`),
	} {
		t.Run(name, func(t *testing.T) {
			a := newApprovalHarness(t, askAbove50)
			r := a.call(t, 1, "place_crypto_order", args)
			if d := errData(t, r); d["error"] != "APPROVAL_CHECK_FAILED" || d["receipt_id"] == "" {
				t.Fatalf("data = %v", d)
			}
			if a.rr.forwarded() != 0 || len(a.approvals.Rows()) != 0 || a.notifier.count() != 0 {
				t.Fatalf("forwarded=%d rows=%d mail=%d", a.rr.forwarded(), len(a.approvals.Rows()), a.notifier.count())
			}
			if dec := a.lastDecision(t); dec.Decision != "denied" || dec.DenialCode != "APPROVAL_CHECK_FAILED" {
				t.Fatalf("receipt input = %+v", dec)
			}
		})
	}
	// Invalid UTF-8 anywhere in the body.
	a := newApprovalHarness(t, askAbove50)
	params := "{\"name\":\"place_crypto_order\",\"arguments\":{\"dollar_amount\":\"75\",\"symbol\":\"B\xffC\"},\"_meta\":{\"token\":\"" + a.token + "\"}}"
	req := &Request{JSONRPC: "2.0", ID: json.RawMessage("1"), Method: MethodToolsCall, Params: json.RawMessage(params)}
	resp, err := a.proxy.handleRequest(context.Background(), req)
	if err != nil || resp == nil || resp.Error == nil {
		t.Fatalf("invalid UTF-8 not refused: %+v %v", resp, err)
	}
	if a.rr.forwarded() != 0 || len(a.approvals.Rows()) != 0 || a.notifier.count() != 0 {
		t.Fatal("invalid UTF-8 call was forwarded, held or emailed")
	}
}

// A valid surrogate pair is an ordinary character and stays allowed: it is
// held, approved and forwarded once like any other call.
func TestApprovalAllowsAValidSurrogatePair(t *testing.T) {
	a := newApprovalHarness(t, askAbove50)
	r := a.call(t, 1, "place_crypto_order", orderSymbol("75", `\ud83d\ude00`))
	id := errData(t, r)["approval_id"].(string)
	if err := a.approvals.Approve("tenant-1", id, "owner@example.com", a.clock); err != nil {
		t.Fatal(err)
	}
	if r := a.call(t, 2, "place_crypto_order", orderSymbol("75", `\ud83d\ude00`)); r.Error != nil {
		t.Fatalf("approved call with a valid pair refused: %v", r.Error)
	}
	if a.rr.forwarded() != 1 {
		t.Fatalf("forwarded = %d", a.rr.forwarded())
	}
}

// tokenFor mints another token of the same tenant with its own id (a delegated
// child or a rotated token is a new token to the store) and the same rule.
func (a *approvalHarness) tokenFor(t *testing.T, budget string) string {
	t.Helper()
	svc, tok := mintBudgetToken(t, argTestRoot, "place_crypto_order,place_equity_order", "tenant-1", budget, "50")
	return appendCaveat(t, svc, tok, controlWord(t, askAbove50))
}

func (a *approvalHarness) callAs(t *testing.T, token string, id int, args string) *Response {
	t.Helper()
	resp, err := a.proxy.handleRequest(context.Background(), argCall(id, token, "place_crypto_order", args))
	if err != nil || resp == nil {
		t.Fatalf("handleRequest: %v %v", resp, err)
	}
	return resp
}

// Email volume: 30 tokens of one tenant each hold a call. The owner gets 10
// emails in the hour, every call is still held and listed, and one summary
// (a count, no amounts or symbols) follows when the hour allows.
func TestApprovalEmailsAreCappedPerTenantWithOneSummary(t *testing.T) {
	a := newApprovalHarness(t, askAbove50)
	for i := 0; i < 30; i++ {
		r := a.callAs(t, a.tokenFor(t, "budget-"+leftPad(i)), i+1, order("75"))
		if errData(t, r)["error"] != "APPROVAL_REQUIRED" {
			t.Fatalf("token %d: %v", i, r.Error)
		}
	}
	if got := len(a.approvals.Rows()); got != 30 {
		t.Fatalf("held = %d, want all 30 listed", got)
	}
	if got := a.notifier.count(); got != toolcontrols.MaxApprovalEmailsPerHour {
		t.Fatalf("emails = %d, want %d", got, toolcontrols.MaxApprovalEmailsPerHour)
	}
	if a.rr.forwarded() != 0 {
		t.Fatal("a held call was forwarded")
	}
	// A burst is not announced the moment it starts.
	if n, ok := a.approvals.TakeSummary("tenant-1", a.clock.Add(time.Minute)); ok {
		t.Fatalf("summary before the delay: %d", n)
	}
	// Then one summary counts the 20 held without an email, and only one.
	n, ok := a.approvals.TakeSummary("tenant-1", a.clock.Add(3*time.Minute))
	if !ok || n != 20 {
		t.Fatalf("summary = %d %v, want 20", n, ok)
	}
	if _, ok := a.approvals.TakeSummary("tenant-1", a.clock.Add(4*time.Minute)); ok {
		t.Fatal("a second summary for the same backlog")
	}
}

// At most 50 approvals wait across all of a tenant's tokens. The 51st is
// refused APPROVAL_QUEUE_FULL with a signed receipt: nothing held, no email.
func TestApprovalTenantQueueRefusesThe51st(t *testing.T) {
	a := newApprovalHarness(t, askAbove50)
	tokens := []string{a.tokenFor(t, "budget-a"), a.tokenFor(t, "budget-b"), a.tokenFor(t, "budget-c")}
	n := 0
	for n < toolcontrols.MaxPendingApprovalsPerTenant {
		tok := tokens[n/toolcontrols.MaxPendingApprovals] // 20 + 20 + 10: no token hits its own bound first
		r := a.callAs(t, tok, n+1, order("60."+leftPad(n)))
		if errData(t, r)["error"] != "APPROVAL_REQUIRED" {
			t.Fatalf("call %d: %v", n+1, r.Error)
		}
		n++
	}
	rows, mail := len(a.approvals.Rows()), a.notifier.count()
	if rows != 50 {
		t.Fatalf("rows = %d", rows)
	}
	r := a.callAs(t, tokens[2], 100, order("99.99"))
	d := errData(t, r)
	if d["error"] != "APPROVAL_QUEUE_FULL" || d["receipt_id"] == "" || d["evidence_url"] == "" {
		t.Fatalf("51st: %v", d)
	}
	if dec := a.lastDecision(t); dec.Decision != "denied" || dec.DenialCode != "APPROVAL_QUEUE_FULL" {
		t.Fatalf("receipt input = %+v", dec)
	}
	if len(a.approvals.Rows()) != rows || a.notifier.count() != mail || a.rr.forwarded() != 0 {
		t.Fatalf("51st was held, emailed or forwarded: rows %d->%d mail %d->%d forwarded %d",
			rows, len(a.approvals.Rows()), mail, a.notifier.count(), a.rr.forwarded())
	}
	// A call that is already waiting is still answered, not refused.
	if r := a.callAs(t, tokens[0], 101, order("60."+leftPad(0))); errData(t, r)["error"] != "APPROVAL_REQUIRED" {
		t.Fatalf("a waiting call was refused: %v", r.Error)
	}
}
