package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/satgate-io/satgate/pkg/macaroon"
)

// eaWindowRecorder is an in-memory stand-in for the gateway's window store. It
// keeps one window per tenant, key, tool, code and UTC minute: the first
// refusal signs the opener, the rest are counted, and Close signs one closer
// with the count for every window that has been over for the grace period.
type eaWindowRecorder struct {
	*eaRecorder
	mu      sync.Mutex
	windows map[string]*eaWindow
	order   []string
	calls   int
	// misses are the Observe calls that went through with no receipt, as the
	// recorder was told of them.
	misses []MCPWindowedCall
}

type eaWindow struct {
	opener MCPDecision
	count  int64
	closed bool
}

func newEAWindowRecorder(base *eaRecorder) *eaWindowRecorder {
	return &eaWindowRecorder{eaRecorder: base, windows: map[string]*eaWindow{}}
}

func (w *eaWindowRecorder) RecordWindowedCall(ctx context.Context, call MCPWindowedCall) (*MCPEvidence, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.calls++
	if call.Kind == WindowKindUnrecorded {
		w.misses = append(w.misses, call)
		return nil, nil
	}
	d := call.Decision
	key := strings.Join([]string{d.TenantID, d.TokenID, d.ToolName, d.DenialCode, call.WindowStart.UTC().Format(time.RFC3339)}, "|")
	if win, ok := w.windows[key]; ok {
		win.count++
		return nil, nil
	}
	opener := call.OpenerDecision()
	ev, err := w.eaRecorder.RecordMCPDecision(ctx, opener)
	if err != nil {
		return nil, err
	}
	w.windows[key] = &eaWindow{opener: opener, count: 1}
	w.order = append(w.order, key)
	return ev, nil
}

// Close signs the closer of every window over for the grace period at now.
func (w *eaWindowRecorder) Close(ctx context.Context, now time.Time) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	for _, key := range w.order {
		win := w.windows[key]
		if win.closed {
			continue
		}
		start, err := time.Parse(time.RFC3339, win.opener.RateLimitWindowStart)
		if err != nil {
			return err
		}
		if now.Before(start.Add((RateWindowSeconds + RateWindowGraceSeconds) * time.Second)) {
			continue
		}
		if _, err := w.eaRecorder.RecordMCPDecision(ctx, CloserDecision(win.opener, win.count)); err != nil {
			return err
		}
		win.closed = true
	}
	return nil
}

func init() {
	eaSetSwitch = func(e *eaEnv, on bool) {
		if on && e.window {
			w := newEAWindowRecorder(e.base)
			e.rec = w
			e.proxy.SetEvidenceRecorder(w)
		}
		e.proxy.SetSignEveryAction(on)
	}
}

func eaIsProofUnavailable(r *Response) bool {
	if r == nil || r.Error == nil || r.Error.Code != CodeInternalError {
		return false
	}
	var d map[string]any
	if json.Unmarshal(r.Error.Data, &d) != nil {
		return false
	}
	return d["error"] == "proof_unavailable"
}

// ---- the master table, switch on ----

func TestEveryAction_Matrix(t *testing.T) {
	included, excluded := 0, 0
	for _, row := range eaTable() {
		row := row
		t.Run(row.id, func(t *testing.T) {
			res := eaRun(t, row, true, false)
			if !row.included {
				if len(res.signed) != 0 || res.attempts != 0 {
					t.Fatalf("%s is excluded (%s) but %d receipt(s) were asked for: %v", row.id, row.why, res.attempts, res.signed)
				}
				return
			}
			if len(res.signed) == 0 {
				t.Fatalf("%s (%s) is an action and yielded no receipt", row.id, row.what)
			}
			if !reflect.DeepEqual(res.signed, row.want) {
				t.Fatalf("%s (%s): receipts = %v, want exactly %v", row.id, row.what, res.signed, row.want)
			}
		})
		if row.included {
			included++
		} else {
			excluded++
		}
	}
	t.Logf("master table: %d included rows, %d excluded rows", included, excluded)
	if included != 38 || excluded != 14 {
		t.Fatalf("the table has %d included and %d excluded rows; the PR report says 33 and 14. Update both together", included, excluded)
	}
}

// The same table through the two real transports. An included row flagged
// transports yields exactly the receipts it lists, and with the signer down it
// is stopped with proof_unavailable and nothing debited, however it arrived.
func TestEveryAction_Matrix_Transports(t *testing.T) {
	rows := 0
	for _, row := range eaTable() {
		if !row.transports {
			continue
		}
		rows++
		for _, tr := range eaTransports {
			row, tr := row, tr
			t.Run(row.id+"_"+tr.name, func(t *testing.T) {
				up := eaRunVia(t, row, true, false, tr.via)
				if !reflect.DeepEqual(up.signed, row.want) {
					t.Fatalf("%s (%s) via %s: receipts = %v, want exactly %v", row.id, row.what, tr.name, up.signed, row.want)
				}
				direct := eaRun(t, row, true, false)
				for i := range up.responses {
					if eaShape(up.responses[i]) != eaShape(direct.responses[i]) {
						t.Fatalf("%s via %s answers differently:\n transport %s\n direct    %s", row.id, tr.name, eaShape(up.responses[i]), eaShape(direct.responses[i]))
					}
				}
				if up.forwarded != direct.forwarded || up.debit != direct.debit {
					t.Fatalf("%s via %s: forwarded %d debit %d, direct: forwarded %d debit %d", row.id, tr.name, up.forwarded, up.debit, direct.forwarded, direct.debit)
				}
				down := eaRunVia(t, row, true, true, tr.via)
				last := down.responses[len(down.responses)-1]
				if row.observeOpen {
					if down.forwarded != 1 || last.Error != nil {
						t.Fatalf("%s via %s: Observe with the signer down: forwarded %d, answer %s", row.id, tr.name, down.forwarded, eaShape(last))
					}
					return
				}
				if !eaIsProofUnavailable(last) || down.forwarded != 0 || down.debit != 0 || down.counter != 0 {
					t.Fatalf("%s via %s, signer down: answer %s forwarded %d debit %d window_total %d", row.id, tr.name, eaShape(last), down.forwarded, down.debit, down.counter)
				}
			})
		}
	}
	if rows < 30 {
		t.Fatalf("only %d rows are run through the transports", rows)
	}
}

// Every method the dispatcher knows is either in the table or in the list of
// methods that are decided on purpose. Adding a case to dispatchRequest without
// deciding fails here.
func TestEveryAction_MatrixCoversEverySwitchCase(t *testing.T) {
	consts := map[string]string{}
	fset := token.NewFileSet()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	var proxyFile *ast.File
	for _, ent := range entries {
		name := ent.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		if name == "proxy.go" {
			proxyFile = f
		}
		for _, decl := range f.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || gd.Tok != token.CONST {
				continue
			}
			for _, spec := range gd.Specs {
				vs := spec.(*ast.ValueSpec)
				for i, n := range vs.Names {
					if i < len(vs.Values) {
						if lit, ok := vs.Values[i].(*ast.BasicLit); ok && lit.Kind == token.STRING {
							v, _ := strconv.Unquote(lit.Value)
							consts[n.Name] = v
						}
					}
				}
			}
		}
	}
	var cases []string
	ast.Inspect(proxyFile, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "dispatchRequest" {
			return true
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			cc, ok := n.(*ast.CaseClause)
			if !ok {
				return true
			}
			for _, expr := range cc.List {
				switch x := expr.(type) {
				case *ast.Ident:
					cases = append(cases, consts[x.Name])
				case *ast.BasicLit:
					v, _ := strconv.Unquote(x.Value)
					cases = append(cases, v)
				}
			}
			return true
		})
		return false
	})
	if len(cases) < 8 {
		t.Fatalf("read only %d cases from dispatchRequest: %v", len(cases), cases)
	}
	for _, m := range cases {
		if m == "" {
			t.Fatal("a case in dispatchRequest could not be read; teach this test about it")
		}
		if _, ok := everyActionMethods[m]; !ok {
			t.Errorf("dispatchRequest handles %q but everyActionMethods does not say whether it is an action", m)
		}
	}
	// Every method in the list is sent by some row of the table.
	seen := map[string]bool{}
	for _, row := range eaTable() {
		e := newEAEnv(t, row.opts)
		e.proxy.SetResponseGuard(func(ctx context.Context, req *Request, handle func(context.Context) (*Response, error)) (*Response, error) {
			seen[req.Method] = true
			return handle(ctx)
		})
		row.run(e)
	}
	var missing []string
	for m := range everyActionMethods {
		if !seen[m] {
			missing = append(missing, m)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Fatalf("methods with no row in the table: %v", missing)
	}
}

// ---- a signing failure on every row ----

func TestEveryAction_SignerDown_StopsEveryIncludedRow(t *testing.T) {
	for _, row := range eaTable() {
		row := row
		t.Run(row.id, func(t *testing.T) {
			res := eaRun(t, row, true, true)
			if !row.included {
				if res.attempts != 0 || len(res.signed) != 0 {
					t.Fatalf("excluded row signed: %v", res.signed)
				}
				if res.forwarded != 0 {
					t.Fatalf("excluded row was passed on")
				}
				return
			}
			last := res.responses[len(res.responses)-1]
			switch {
			case row.observeOpen:
				// Today's Observe: the call is still passed on. See
				// TestObserveKeepsTodaysBehaviourWhenAReceiptFails.
				if res.forwarded != 1 {
					t.Fatalf("Observe row: forwarded = %d, today's behaviour is 1", res.forwarded)
				}
			default:
				if !eaIsProofUnavailable(last) {
					t.Fatalf("%s: agent got %s, want proof_unavailable", row.id, eaShape(last))
				}
				if res.forwarded != 0 {
					t.Fatalf("%s: passed on %d time(s) with no receipt", row.id, res.forwarded)
				}
				if res.debit != 0 || res.counter != 0 {
					t.Fatalf("%s: debit=%d window_total=%d, want none", row.id, res.debit, res.counter)
				}
			}
		})
	}
}

// A receipt that fails to store at the last moment (the early check passed)
// still stops a delegation, and the carve goes back to the parent.
func TestEveryAction_Delegate_SignsOrGivesBack(t *testing.T) {
	e := newEAEnv(t, eaOpts{})
	eaSetSwitch(e, true)
	before := e.remaining()
	e.base.mu.Lock()
	e.base.failRecord = true // the preflight still passes
	e.base.mu.Unlock()
	resp := e.send(e.raw(1, MethodSatGateDelegate, `"budget":5,"scope":"free_read","label":"helper"`))
	if !eaIsProofUnavailable(resp) {
		t.Fatalf("answer = %s", eaShape(resp))
	}
	if e.remaining() != before {
		t.Fatalf("the carve was not given back: %d -> %d", before, e.remaining())
	}
	if resp.Result != nil {
		t.Fatal("a key was returned")
	}
	// And with the signer back, the same call delegates once, signed once.
	e.base.setDown(false)
	resp = e.send(e.raw(2, MethodSatGateDelegate, `"budget":5,"scope":"free_read","label":"helper"`))
	if resp.Error != nil {
		t.Fatalf("refused: %+v", resp.Error)
	}
	if got := e.base.signed(); !reflect.DeepEqual(got, []string{"allowed/policy_allowed"}) {
		t.Fatalf("receipts = %v", got)
	}
	if e.remaining() != before-5 {
		t.Fatalf("remaining = %d, want %d", e.remaining(), before-5)
	}
	d := e.base.decisions[0]
	if d.MCPMethod != MethodSatGateDelegate || d.RouteOrTool != "mcp:satgate/delegate" || !d.BudgetNotEvaluated {
		t.Fatalf("decision = %+v", d)
	}
}

// ---- switch off: today's answers ----

func TestEveryAction_SwitchOff_IsToday(t *testing.T) {
	want, err := os.ReadFile("testdata/every_action_switch_off.golden")
	if err != nil {
		t.Fatal(err)
	}
	got := eaGolden(t)
	if got == string(want) {
		return
	}
	wl, gl := strings.Split(string(want), "\n"), strings.Split(got, "\n")
	for i := 0; i < len(wl) && i < len(gl); i++ {
		if wl[i] != gl[i] {
			start := i - 8
			if start < 0 {
				start = 0
			}
			t.Fatalf("with the switch off an answer differs from main at golden line %d:\n  main: %s\n  here: %s\ncontext:\n%s", i+1, wl[i], gl[i], strings.Join(gl[start:i+1], "\n"))
		}
	}
	t.Fatalf("with the switch off the output has %d lines, main's has %d", len(gl), len(wl))
}

// The switch is off in a new proxy and SetSignEveryAction(false) is a no-op.
func TestEveryAction_SwitchDefaultsOff(t *testing.T) {
	e := newEAEnv(t, eaOpts{})
	if e.proxy.SignEveryAction() {
		t.Fatal("a new proxy has the switch on")
	}
	e.proxy.SetSignEveryAction(true)
	e.proxy.SetSignEveryAction(false)
	if e.proxy.SignEveryAction() {
		t.Fatal("the switch did not turn off")
	}
}

// ---- Observe: today's behaviour, pinned ----

func TestObserveKeepsTodaysBehaviourWhenAReceiptFails(t *testing.T) {
	if (&Proxy{}).allowReceiptFailureBlocks("observe") || !(&Proxy{}).allowReceiptFailureBlocks("control") {
		t.Fatal("Observe must not stop a call whose receipt fails, and Control must")
	}
	rows := map[string]bool{"T04": true, "T05": true}
	for _, row := range eaTable() {
		if !rows[row.id] {
			continue
		}
		for _, on := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s_switch_on_%v", row.id, on), func(t *testing.T) {
				res := eaRun(t, row, on, true)
				if res.forwarded != 1 {
					t.Fatalf("Observe call with the signer down: forwarded = %d, today it is passed on", res.forwarded)
				}
				if eaIsProofUnavailable(res.responses[0]) || res.responses[0].Error != nil {
					t.Fatalf("Observe call with the signer down got %s", eaShape(res.responses[0]))
				}
			})
		}
	}
	// A delegation in Observe is the same: signed when it can be, passed on when not.
	for _, on := range []bool{false, true} {
		e := newEAEnv(t, eaOpts{observe: true})
		if on {
			eaSetSwitch(e, true)
		}
		e.base.setDown(true)
		if resp := e.send(e.raw(1, MethodSatGateDelegate, `"budget":5,"scope":"free_read","label":"helper"`)); resp.Error != nil {
			t.Fatalf("switch %v: Observe delegation with the signer down was stopped: %+v", on, resp.Error)
		}
	}
}

// ---- one receipt per decision ----

// A call the spend path signed has one receipt, never two, even when the
// receipt it asked for could not be made.
func TestEveryAction_NoDoubleReceipt_Spend(t *testing.T) {
	for _, id := range []string{"T01", "T03", "T16", "T21", "T23"} {
		for _, row := range eaTable() {
			if row.id != id {
				continue
			}
			res := eaRun(t, row, true, false)
			if len(res.signed) != 1 || res.signed[0] != "allowed/budget_authorized" || res.attempts != 1 {
				t.Errorf("%s: signed=%v asked=%d, want one budget_authorized", id, res.signed, res.attempts)
			}
		}
	}
	// Observe: the spend path's receipt fails. No second, different receipt is
	// tried for the same call.
	row := eaRow{opts: eaOpts{observe: true}, run: eaToolCall(1, "paid_read", "")}
	res := eaRun(t, row, true, true)
	if res.attempts != 1 {
		t.Fatalf("receipts asked for = %d, want 1", res.attempts)
	}
}

func TestEveryAction_HeldThenApproved(t *testing.T) {
	for _, row := range eaTable() {
		if row.id != "T24" {
			continue
		}
		e := newEAEnv(t, row.opts)
		eaSetSwitch(e, true)
		resps := row.run(e)
		if resps[0].Error == nil || resps[1].Error != nil {
			t.Fatalf("responses = %s / %s", eaShape(resps[0]), eaShape(resps[1]))
		}
		if e.rr.forwarded() != 1 {
			t.Fatalf("forwarded = %d, want 1 (the approved call)", e.rr.forwarded())
		}
		ds := e.base.decisions
		if len(ds) != 2 || ds[0].Decision != "denied" || ds[1].Decision != "allowed" {
			t.Fatalf("receipts = %v", e.base.signed())
		}
		if ds[1].ApprovalID == "" || ds[1].ApprovalID != ds[0].ApprovalID {
			t.Fatalf("the approval id is on the second receipt: %q vs %q", ds[1].ApprovalID, ds[0].ApprovalID)
		}
	}
}

// A signer that is down before the approved call is sent again does not use
// up the approval: the owner does not have to approve twice.
func TestEveryAction_EarlyPreflight_DoesNotUseApproval(t *testing.T) {
	e := newEAEnv(t, eaOpts{approvals: true, caveats: []func(*testing.T) string{eaControl(askAbove50)}})
	eaSetSwitch(e, true)
	if r := e.call(1, "place_crypto_order", order("75")); r.Error == nil {
		t.Fatal("not held")
	}
	rows := e.approvals.Rows()
	if len(rows) != 1 {
		t.Fatalf("approvals = %d", len(rows))
	}
	if err := e.approvals.Approve(eaTenant, rows[0].ID, "owner@example.com", e.clock); err != nil {
		t.Fatal(err)
	}
	e.base.setDown(true)
	if r := e.call(2, "place_crypto_order", order("75")); !eaIsProofUnavailable(r) {
		t.Fatalf("answer = %s", eaShape(r))
	}
	if e.rr.forwarded() != 0 {
		t.Fatal("passed on with the signer down")
	}
	e.base.setDown(false)
	if r := e.call(3, "place_crypto_order", order("75")); r.Error != nil {
		t.Fatalf("the approved call was refused after the signer came back: %+v", r.Error)
	}
	if e.rr.forwarded() != 1 {
		t.Fatalf("forwarded = %d", e.rr.forwarded())
	}
}

// A call whose receipt failed leaves nothing in the spending-limit window.
func TestEveryAction_SignFailureReleasesGate(t *testing.T) {
	e := newEAEnv(t, eaOpts{gate: true, caveats: []func(*testing.T) string{eaControl(spendDoc200Day)}})
	eaSetSwitch(e, true)
	// The early check passes; the receipt itself fails, after the gate admitted
	// the call and reserved its amount.
	e.base.mu.Lock()
	e.base.failRecord = true
	e.base.mu.Unlock()
	r := e.call(1, "place_crypto_order", order("100"))
	if !eaIsProofUnavailable(r) {
		t.Fatalf("answer = %s", eaShape(r))
	}
	if e.counter() != 0 {
		t.Fatalf("window total after a failed receipt = %d, want 0", e.counter())
	}
	if e.remaining() != eaCredits {
		t.Fatalf("budget after a failed receipt = %d, want %d", e.remaining(), eaCredits)
	}
	if e.rr.forwarded() != 0 {
		t.Fatal("passed on")
	}
}

// ---- what a receipt holds ----

func TestEveryAction_ReceiptHoldsNoArguments(t *testing.T) {
	const marker = "ARG-MARKER-7731-never-in-a-receipt"
	e := newEAEnv(t, eaOpts{})
	eaSetSwitch(e, true)
	args := `{"q":"` + marker + `"}`
	_ = e.call(1, "free_read", args)
	_ = e.call(2, "paid_read", args)
	_ = e.call(3, "big_order", args)
	_ = e.send(e.raw(4, "resources/read", `"uri":"`+marker+`"`))
	_ = e.send(e.raw(5, MethodSatGateDelegate, `"budget":1,"label":"`+marker+`","scope":"free_read"`))
	_ = e.send(e.raw(6, MethodSatGateDelegate, `"budget":9999,"label":"`+marker+`"`))
	e.base.mu.Lock()
	defer e.base.mu.Unlock()
	if len(e.base.attempts) < 5 {
		t.Fatalf("only %d receipts were asked for", len(e.base.attempts))
	}
	for _, d := range e.base.attempts {
		raw, _ := json.Marshal(d)
		if strings.Contains(string(raw), marker) {
			t.Fatalf("a receipt carries call data: %s", raw)
		}
		if strings.Contains(string(raw), e.token) {
			t.Fatalf("a receipt carries the key: %s", raw)
		}
	}
}

// The allow receipt makes no balance claim: no cost, no remaining, no limit.
func TestEveryAction_AllowReceiptMakesNoBalanceClaim(t *testing.T) {
	e := newEAEnv(t, eaOpts{})
	eaSetSwitch(e, true)
	_ = e.call(1, "free_read", "")
	if len(e.base.decisions) != 1 {
		t.Fatalf("receipts = %d", len(e.base.decisions))
	}
	d := e.base.decisions[0]
	if d.Decision != "allowed" || d.DecisionReason != "policy_allowed" || !d.BudgetNotEvaluated {
		t.Fatalf("decision = %+v", d)
	}
	if d.CostCredits != 0 || d.RemainingCredits != 0 || d.RemainingBeforeCredits != 0 || d.BudgetLimitCredits != 0 {
		t.Fatalf("a balance was claimed: %+v", d)
	}
	if d.ToolName != "free_read" || d.RouteOrTool != "mcp:tools/call:free_read" || d.MCPMethod != MethodToolsCall {
		t.Fatalf("tool not named: %+v", d)
	}
	if d.TokenID == "" || d.TenantID != eaTenant || d.PolicyMode != "control" || d.RequestID == "" || d.JSONRPCID != "1" {
		t.Fatalf("identity missing: %+v", d)
	}
}

// The agent is told the receipt: the handles ride on the answer.
func TestEveryAction_AnswerCarriesTheReceiptHandles(t *testing.T) {
	e := newEAEnv(t, eaOpts{})
	eaSetSwitch(e, true)
	r := e.call(1, "free_read", "")
	var result map[string]any
	if err := json.Unmarshal(r.Result, &result); err != nil {
		t.Fatal(err)
	}
	meta, _ := result["_meta"].(map[string]any)
	ev, _ := meta["satgate_evidence"].(map[string]any)
	if ev["receipt_id"] != "rcpt_1" {
		t.Fatalf("answer = %s", r.Result)
	}
	r = e.send(e.raw(2, "resources/list", ""))
	if d := errData(t, r); d["receipt_id"] != "rcpt_2" {
		t.Fatalf("refusal data = %v", d)
	}
}

// ---- no key, no receipt ----

func TestEveryAction_NoKey_NoReceipt(t *testing.T) {
	for _, o := range []eaOpts{{noKey: true}, {garbage: true}, {expired: true}} {
		e := newEAEnv(t, o)
		eaSetSwitch(e, true)
		for i, m := range []string{MethodToolsCall, MethodSatGateBudget, MethodSatGateDelegate, "budget/check", "resources/list"} {
			resp := e.send(e.raw(i+1, m, `"name":"free_read"`))
			if resp == nil || resp.Error == nil {
				t.Fatalf("%v %s: answered %v", o, m, resp)
			}
		}
		if len(e.base.attempts) != 0 {
			t.Fatalf("%v: %d receipt(s) for requests with no key SatGate can name", o, len(e.base.attempts))
		}
	}
}

// With the switch on and no recorder, nothing that would be passed on is.
func TestEveryAction_OnWithoutRecorder_FailsClosed(t *testing.T) {
	e := newEAEnv(t, eaOpts{})
	e.proxy.SetEvidenceRecorder(nil)
	e.proxy.SetSignEveryAction(true)
	for _, r := range []*Response{
		e.call(1, "free_read", ""),
		e.call(2, "paid_read", ""),
		e.send(e.raw(3, MethodSatGateBudget, "")),
		e.send(e.raw(4, MethodSatGateDelegate, `"budget":1`)),
	} {
		if !eaIsProofUnavailable(r) {
			t.Fatalf("answer = %s", eaShape(r))
		}
	}
	if e.rr.forwarded() != 0 || e.remaining() != eaCredits {
		t.Fatal("something was passed on or debited")
	}
	// A refusal of a verified key is signed or it is not given: with no
	// recorder it is proof_unavailable, never the plain refusal.
	for i, r := range []*Response{
		e.send(e.raw(5, "resources/list", "")),
		e.send(e.raw(6, MethodToolsCall, `"name":123`)),
		e.send(e.raw(7, MethodSatGateDelegate, `"budget":"many"`)),
		e.call(8, "argrules:v1:abc", ""),
		e.call(9, "not_in_scope_tool", ""),
	} {
		if !eaIsProofUnavailable(r) {
			t.Fatalf("refusal %d = %s", i, eaShape(r))
		}
	}
}

// ---- budget/check ----

func TestEveryAction_BudgetCheckAlias(t *testing.T) {
	e := newEAEnv(t, eaOpts{})
	// Off: today's answer, an internal error.
	off := e.send(e.raw(1, "budget/check", ""))
	if off.Error == nil || off.Error.Code != CodeInternalError || off.Error.Message != "unreachable" {
		t.Fatalf("switch off: %s", eaShape(off))
	}
	eaSetSwitch(e, true)
	on := e.send(e.raw(2, "budget/check", ""))
	if on.Error != nil {
		t.Fatalf("switch on: %+v", on.Error)
	}
	var res map[string]any
	if err := json.Unmarshal(on.Result, &res); err != nil || res["remaining"] != float64(eaCredits) {
		t.Fatalf("result = %s (%v)", on.Result, err)
	}
	same := e.send(e.raw(3, MethodSatGateBudget, ""))
	var res2 map[string]any
	_ = json.Unmarshal(same.Result, &res2)
	delete(res, "_meta")
	delete(res2, "_meta")
	if !reflect.DeepEqual(res, res2) {
		t.Fatalf("the alias answers differently: %v vs %v", res, res2)
	}
}

// ---- refusals that repeat: windows ----

func TestEveryAction_RateWindow_OpenerCloserAndCount(t *testing.T) {
	e := newEAEnv(t, eaOpts{limited: true, window: true})
	eaSetSwitch(e, true)
	w := e.rec.(*eaWindowRecorder)
	for i := 0; i < 7; i++ {
		r := e.call(i+1, "paid_read", "")
		if r.Error == nil || r.Error.Code != CodeRateLimited {
			t.Fatalf("answer %d = %s", i, eaShape(r))
		}
	}
	// A second tool is a second window.
	_ = e.call(20, "free_read", "")
	if w.calls != 8 {
		t.Fatalf("the recorder was told of %d refused calls, want 8", w.calls)
	}
	if got := e.base.signed(); len(got) != 2 {
		t.Fatalf("receipts so far = %v, want one opener per tool", got)
	}
	opener := e.base.decisions[0]
	if opener.RateLimitPhase != RateWindowOpened || opener.RateLimitWindowSeconds != 60 || opener.RateLimitRefusedCalls != 0 || opener.DenialCode != "RATE_LIMITED" {
		t.Fatalf("opener = %+v", opener)
	}
	start, err := time.Parse(time.RFC3339, opener.RateLimitWindowStart)
	if err != nil || start.Second() != 0 || start.Location() != time.UTC {
		t.Fatalf("window start %q is not a UTC minute", opener.RateLimitWindowStart)
	}
	if opener.ToolName != "paid_read" || opener.RouteOrTool != "mcp:tools/call:paid_read" || !opener.BudgetNotEvaluated {
		t.Fatalf("opener = %+v", opener)
	}
	// Too early: no closer. After the grace period: one closer per window.
	if err := w.Close(context.Background(), start.Add(70*time.Second)); err != nil {
		t.Fatal(err)
	}
	if len(e.base.decisions) != 2 {
		t.Fatalf("a closer was signed before the grace period was over")
	}
	if err := w.Close(context.Background(), start.Add(80*time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(context.Background(), start.Add(90*time.Second)); err != nil {
		t.Fatal(err)
	}
	if len(e.base.decisions) != 4 {
		t.Fatalf("receipts = %v, want two openers and two closers", e.base.signed())
	}
	var closers []MCPDecision
	for _, d := range e.base.decisions {
		if d.RateLimitPhase == RateWindowClosed {
			closers = append(closers, d)
		}
	}
	if len(closers) != 2 || closers[0].ToolName != "paid_read" || closers[0].RateLimitRefusedCalls != 7 || closers[1].RateLimitRefusedCalls != 1 {
		t.Fatalf("closers = %+v", closers)
	}
	if closers[0].RateLimitWindowStart != opener.RateLimitWindowStart || closers[0].DenialCode != "RATE_LIMITED" {
		t.Fatalf("closer does not name the window it closes: %+v", closers[0])
	}
}

// A refusal that cannot be recorded is not given as if it had been: with the
// signer down the agent gets proof_unavailable, not the rate-limit answer. A
// call counted inside a window that is already open and signed is a success and
// gets the ordinary answer.
func TestEveryAction_RateWindow_ProofUnavailableWhenTheSignerIsDown(t *testing.T) {
	e := newEAEnv(t, eaOpts{limited: true, window: true})
	eaSetSwitch(e, true)
	e.base.setDown(true)
	r := e.call(1, "paid_read", "")
	if !eaIsProofUnavailable(r) {
		t.Fatalf("answer = %s", eaShape(r))
	}
	if strings.Contains(string(r.Error.Data), "retry_after") {
		t.Fatalf("the proof_unavailable answer carries the rate-limit data: %s", r.Error.Data)
	}
	// The signer comes back: the first refusal opens the window.
	e.base.setDown(false)
	if r := e.call(2, "paid_read", ""); r.Error == nil || r.Error.Code != CodeRateLimited {
		t.Fatalf("answer = %s", eaShape(r))
	}
	// The signer goes down again. The window is open and signed, so the next
	// refusal is only counted: that needs no signature and is not a failure.
	e.base.setDown(true)
	r = e.call(3, "paid_read", "")
	if r.Error == nil || r.Error.Code != CodeRateLimited || !strings.Contains(string(r.Error.Data), `"retry_after_seconds":3`) {
		t.Fatalf("a call counted in an open window got %s", eaShape(r))
	}
	// A different tool is a different window: it must be opened, and cannot be.
	if r := e.call(4, "free_read", ""); !eaIsProofUnavailable(r) {
		t.Fatalf("answer = %s", eaShape(r))
	}
}

// A recorder with no windows cannot sign a repeated refusal as a window. The
// switch says so when it is turned on, and every window refusal is then
// proof_unavailable: nothing is signed one per call behind the wiring's back.
func TestEveryAction_RateLimit_RecorderWithoutWindows(t *testing.T) {
	e := newEAEnv(t, eaOpts{limited: true})
	err := e.proxy.SetSignEveryAction(true)
	if !errors.Is(err, ErrSignEveryActionNeedsWindowRecorder) {
		t.Fatalf("SetSignEveryAction with a recorder that keeps no windows returned %v", err)
	}
	if !errors.Is(e.proxy.SignEveryActionReady(), ErrSignEveryActionNeedsWindowRecorder) {
		t.Fatal("SignEveryActionReady does not say the wiring is incomplete")
	}
	for i := 1; i <= 2; i++ {
		if r := e.call(i, "paid_read", ""); !eaIsProofUnavailable(r) {
			t.Fatalf("answer %d = %s", i, eaShape(r))
		}
	}
	if got := e.base.signed(); len(got) != 0 {
		t.Fatalf("receipts = %v, want none", got)
	}
}

// SetSignEveryAction reports a wiring that cannot keep the promise: no
// recorder, or one without windows. A recorder with windows is ready, and the
// switch off is always ready.
func TestEveryAction_SetupError(t *testing.T) {
	e := newEAEnv(t, eaOpts{})
	if err := e.proxy.SetSignEveryAction(false); err != nil {
		t.Fatalf("switch off: %v", err)
	}
	e.proxy.SetEvidenceRecorder(nil)
	if err := e.proxy.SetSignEveryAction(true); !errors.Is(err, ErrSignEveryActionNeedsWindowRecorder) {
		t.Fatalf("no recorder: %v", err)
	}
	if !e.proxy.SignEveryAction() {
		t.Fatal("the switch was not set; a wiring error must stop traffic, not leave it unsigned")
	}
	e.proxy.SetEvidenceRecorder(e.base)
	if err := e.proxy.SignEveryActionReady(); !errors.Is(err, ErrSignEveryActionNeedsWindowRecorder) {
		t.Fatalf("recorder without windows: %v", err)
	}
	e.proxy.SetEvidenceRecorder(newEAWindowRecorder(e.base))
	if err := e.proxy.SetSignEveryAction(true); err != nil {
		t.Fatalf("recorder with windows: %v", err)
	}
	if err := e.proxy.SetSignEveryAction(false); err != nil {
		t.Fatalf("switch off again: %v", err)
	}
}

// The limiter can say why it refused; the reason is signed, and only if it is a code.
type reasonedLimiter struct {
	scriptedLimiter
	reason string
}

func (l *reasonedLimiter) RefusalReason(context.Context) string { return l.reason }

func TestEveryAction_LimiterReason(t *testing.T) {
	for _, c := range []struct{ reason, want string }{
		{"TRIAL_ENDED_STOPPED", "TRIAL_ENDED_STOPPED"},
		{"", "RATE_LIMITED"},
		{"your trial ended, " + "buy now", "RATE_LIMITED"}, // not a code: ignored
		{"lower_case", "RATE_LIMITED"},
	} {
		e := newEAEnv(t, eaOpts{window: true})
		e.proxy.SetRequestLimiter(&reasonedLimiter{reason: c.reason})
		eaSetSwitch(e, true)
		r := e.call(1, "paid_read", "")
		if r.Error == nil || r.Error.Code != CodeRateLimited {
			t.Fatalf("answer = %s", eaShape(r))
		}
		want := "denied/policy_denied/" + c.want
		if got := e.base.signed(); !reflect.DeepEqual(got, []string{want}) {
			t.Fatalf("reason %q: receipts = %v, want %v", c.reason, got, want)
		}
	}
}

// ---- expired keys ----

// The error text an agent reads for an expired key is the same with the
// switch on or off; only the error's type carries the mark.
func TestEveryAction_ExpiredKey_TextUnchanged(t *testing.T) {
	var texts []string
	for _, on := range []bool{false, true} {
		e := newEAEnv(t, eaOpts{expired: true, tenantCtx: true, window: true})
		if on {
			eaSetSwitch(e, true)
		}
		r := e.call(1, "paid_read", "")
		if r.Error == nil || r.Error.Code != CodePolicyDenied {
			t.Fatalf("answer = %s", eaShape(r))
		}
		texts = append(texts, r.Error.Message)
	}
	if texts[0] != texts[1] || !strings.Contains(texts[0], "token expired") {
		t.Fatalf("texts differ: %q vs %q", texts[0], texts[1])
	}
	auth := &MacaroonAuthenticator{Service: mustService(t)}
	e := newEAEnv(t, eaOpts{expired: true})
	_, err := auth.Verify(context.Background(), e.token)
	var exp ExpiredCredential
	if !errors.As(err, &exp) || !exp.CredentialExpired() {
		t.Fatalf("the authenticator did not mark an expired key: %v", err)
	}
	// A key that does not verify is never marked expired.
	_, err = auth.Verify(context.Background(), "not-a-token")
	if errors.As(err, &exp) {
		t.Fatalf("garbage was marked expired: %v", err)
	}
	wrong, _ := macaroon.NewService("another-root-key")
	tok, _ := wrong.Mint("*", time.Now().Add(-time.Hour))
	_, err = auth.Verify(context.Background(), wrong.Encode(tok))
	if errors.As(err, &exp) {
		t.Fatalf("a key signed by another root was marked expired: %v", err)
	}
}

func mustService(t *testing.T) *macaroon.Service {
	t.Helper()
	svc, err := macaroon.NewService(argTestRoot)
	if err != nil {
		t.Fatal(err)
	}
	return svc
}

// ---- Observe: a call whose receipt cannot be made goes through and is counted ----

// An Observe call with a forced signing failure is passed on, and the recorder
// is told of exactly one miss: tenant, tool, key and the UTC minute. With the
// switch off nothing is reported (today's behaviour).
func TestObserveMiss_PassedOnAndReportedOnce(t *testing.T) {
	cases := []struct {
		name   string
		tool   string
		method string
		run    func(e *eaEnv) *Response
	}{
		{"free tool", "free_read", MethodToolsCall, func(e *eaEnv) *Response { return e.call(1, "free_read", "") }},
		{"priced tool, receipt of the spend fails", "paid_read", MethodToolsCall, func(e *eaEnv) *Response { return e.call(1, "paid_read", "") }},
		{"over budget", "big_order", MethodToolsCall, func(e *eaEnv) *Response { return e.call(1, "big_order", "") }},
		{"satgate/budget", MethodSatGateBudget, MethodSatGateBudget, func(e *eaEnv) *Response { return e.send(e.raw(1, MethodSatGateBudget, "")) }},
		{"satgate/delegate", MethodSatGateDelegate, MethodSatGateDelegate, func(e *eaEnv) *Response {
			return e.send(e.raw(1, MethodSatGateDelegate, `"budget":5,"scope":"free_read","label":"helper"`))
		}},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			e := newEAEnv(t, eaOpts{observe: true, window: true})
			eaSetSwitch(e, true)
			w := e.rec.(*eaWindowRecorder)
			e.base.setDown(true)
			resp := c.run(e)
			if resp.Error != nil {
				t.Fatalf("an Observe call was stopped: %+v", resp.Error)
			}
			if len(w.misses) != 1 {
				t.Fatalf("miss reports = %d, want exactly 1", len(w.misses))
			}
			m := w.misses[0]
			d := m.Decision
			if m.Kind != WindowKindUnrecorded || m.WindowSeconds != 60 || !m.WindowStart.Equal(RateWindowStart(m.WindowStart)) {
				t.Fatalf("miss = %+v", m)
			}
			if d.TenantID != eaTenant || d.TokenID == "" || d.ToolName != c.tool || d.MCPMethod != c.method || d.PolicyMode != "observe" {
				t.Fatalf("miss names %+v", d)
			}
			if d.RequestID != "" || d.JSONRPCID != "" {
				t.Fatalf("a miss report carries the request's ids: %+v", d)
			}
			// When the signer is back, one gap receipt carries the count.
			g := UnrecordedDecision(m, 3)
			if g.Decision != "allowed" || g.DecisionReason != "observe_unrecorded" || g.UnrecordedCalls != 3 ||
				g.UnrecordedWindowSeconds != 60 || g.UnrecordedWindowStart != m.WindowStart.UTC().Format(time.RFC3339) || !g.BudgetNotEvaluated {
				t.Fatalf("gap decision = %+v", g)
			}
		})
	}

	t.Run("switch off reports nothing", func(t *testing.T) {
		e := newEAEnv(t, eaOpts{observe: true})
		w := newEAWindowRecorder(e.base)
		e.proxy.SetEvidenceRecorder(w) // the switch stays off
		e.base.setDown(true)
		for _, r := range []*Response{e.call(1, "free_read", ""), e.call(2, "paid_read", "")} {
			if r.Error != nil {
				t.Fatalf("stopped: %+v", r.Error)
			}
		}
		if len(w.misses) != 0 || w.calls != 0 {
			t.Fatalf("with the switch off the recorder was told of %d call(s)", w.calls)
		}
	})

	t.Run("signer up reports nothing", func(t *testing.T) {
		e := newEAEnv(t, eaOpts{observe: true, window: true})
		eaSetSwitch(e, true)
		w := e.rec.(*eaWindowRecorder)
		_ = e.call(1, "free_read", "")
		if len(w.misses) != 0 {
			t.Fatal("a call that was signed was reported as a miss")
		}
	})

	t.Run("Control is never a miss", func(t *testing.T) {
		e := newEAEnv(t, eaOpts{window: true})
		eaSetSwitch(e, true)
		w := e.rec.(*eaWindowRecorder)
		e.base.setDown(true)
		if r := e.call(1, "free_read", ""); !eaIsProofUnavailable(r) {
			t.Fatalf("answer = %s", eaShape(r))
		}
		if len(w.misses) != 0 {
			t.Fatal("a stopped Control call was reported as a miss")
		}
	})

	t.Run("a recorder with no windows does not break the call", func(t *testing.T) {
		e := newEAEnv(t, eaOpts{observe: true})
		eaSetSwitch(e, true)
		e.base.setDown(true)
		if r := e.call(1, "free_read", ""); r.Error != nil {
			t.Fatalf("stopped: %+v", r.Error)
		}
	})
}
