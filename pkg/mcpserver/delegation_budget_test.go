package mcpserver

// Round 2, blocker B1: delegation never adds or seals a budget the holder chose.
//
// The HTTP tests run a real /mcp server (httptest) over a real Streamable HTTP
// upstream (a disposable localhost order server that stands in for a brokerage;
// it never talks to a real one). Counters are read from the spend store; "no
// counter change" means the store's keys and totals are the same before and
// after, and "zero upstream calls" is the upstream's own call count.

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/satgate-io/satgate/pkg/macaroon"
	"github.com/satgate-io/satgate/pkg/toolcontrols"
)

const (
	// A word that must never reach an event or a log line.
	scopeMarker = "RAW-SCOPE-MARKER-5521"
	parentCreds = 50
)

type delegHarness struct {
	t      *testing.T
	proxy  *Proxy
	store  *toolcontrols.MemoryStore
	up     *orderUpstream
	srv    *httptest.Server
	svc    *macaroon.Service
	events *ChannelPublisher
	logs   *bytes.Buffer
}

func newDelegHarness(t *testing.T) *delegHarness {
	t.Helper()
	h := &delegHarness{t: t, logs: &bytes.Buffer{}}
	prev := log.Logger
	log.Logger = zerolog.New(h.logs).Level(zerolog.DebugLevel)
	t.Cleanup(func() { log.Logger = prev })

	h.up = newOrderUpstream(t)
	proxy, _, events := newArgProxy(t)
	h.proxy, h.events = proxy, events
	cfg := proxy.config
	cfg.AllowPrivateUpstreams = true
	cfg.DefaultUpstream = "u"
	cfg.Upstreams = map[string]UpstreamConfig{"u": {Transport: "streamable", URL: h.up.srv.URL, Timeout: 2 * time.Second}}
	mgr := NewUpstreamManager(cfg.Upstreams, nil, "u", true)
	ctx, cancel := context.WithCancel(context.Background())
	if err := mgr.Start(ctx); err != nil {
		cancel()
		t.Fatalf("start upstream: %v", err)
	}
	t.Cleanup(func() { cancel(); mgr.Close() })
	proxy.upstream = mgr
	proxy.SetUpstreamRouter(&defaultRouter{mgr: mgr})
	proxy.SetEvidenceRecorder(&lockedRecorder{})
	h.store = toolcontrols.NewMemoryStore(nil)
	proxy.SetCallGate(NewSpendGate(h.store))
	h.srv = httptest.NewServer(NewSSEServer(proxy, ":0").Handler())
	t.Cleanup(h.srv.Close)
	return h
}

// parent mints a token in budget A the way the server does, and gives its own
// credit budget room so a delegation can carve from it.
func (h *delegHarness) parent(own string) string {
	h.t.Helper()
	svc, tok := sealedBudgetToken(h.t, "tenant-1", own, bothTools, budgetDoc(budgetA, "200", ""))
	h.svc = svc
	if err := h.proxy.budget.Initialize(context.Background(), own, parentCreds); err != nil {
		h.t.Fatal(err)
	}
	return tok
}

// rpc sends one JSON-RPC request over the real /mcp endpoint with the token as
// the bearer, in a session of its own.
func (h *delegHarness) rpc(token, method string, params any) *Response {
	h.t.Helper()
	sid := mustInitSession(h.t, h.srv.URL, token)
	body := map[string]any{"jsonrpc": "2.0", "id": 2, "method": method, "params": params}
	status, _, raw := streamablePost(h.t, h.srv.URL+"/mcp", sid, token, body)
	if status != http.StatusOK {
		h.t.Fatalf("%s: status %d body %s", method, status, raw)
	}
	var resp Response
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &resp); err != nil {
		h.t.Fatalf("decode %q: %v", raw, err)
	}
	return &resp
}

func (h *delegHarness) delegate(token string, budget int64, scope, label string) *Response {
	h.t.Helper()
	p := map[string]any{"budget": budget}
	if scope != "" {
		p["scope"] = scope
	}
	if label != "" {
		p["label"] = label
	}
	return h.rpc(token, MethodSatGateDelegate, p)
}

func (h *delegHarness) order(token, sym, amount string) *Response {
	h.t.Helper()
	return h.rpc(token, MethodToolsCall, map[string]any{
		"name":      "place_crypto_order",
		"arguments": map[string]any{"side": "buy", "type": "market", "symbol": sym, "dollar_amount": amount},
	})
}

func childToken(t *testing.T, r *Response) string {
	t.Helper()
	if r.Error != nil {
		t.Fatalf("delegation refused: %+v", r.Error)
	}
	var res DelegateResult
	if err := json.Unmarshal(r.Result, &res); err != nil || res.Token == "" {
		t.Fatalf("no child token in %s (%v)", r.Result, err)
	}
	return res.Token
}

// snapshot is every counter the gate holds and the parent's credit balance.
type delegSnap struct {
	totals  map[string]int64
	credits int64
	calls   int64
}

func (h *delegHarness) snap(own string) delegSnap {
	h.t.Helper()
	s := delegSnap{totals: map[string]int64{}, calls: h.up.calls.Load()}
	for _, k := range h.store.Keys() {
		s.totals[k] = h.store.TotalOf(k)
	}
	rem, err := h.proxy.budget.Remaining(context.Background(), own)
	if err != nil {
		h.t.Fatal(err)
	}
	s.credits = rem
	return s
}

func (a delegSnap) equal(b delegSnap) bool {
	if a.credits != b.credits || a.calls != b.calls || len(a.totals) != len(b.totals) {
		return false
	}
	for k, v := range a.totals {
		if b.totals[k] != v {
			return false
		}
	}
	return true
}

func (h *delegHarness) delegationEvents() []Event {
	var out []Event
	for {
		select {
		case ev := <-h.events.Events():
			if ev.Type == EventDelegation {
				out = append(out, ev)
			}
		default:
			return out
		}
	}
}

func (h *delegHarness) wantRefused(r *Response, what string) {
	h.t.Helper()
	if r.Error == nil {
		h.t.Fatalf("%s: a child token was issued: %s", what, r.Result)
	}
	if r.Error.Code != CodePolicyDenied {
		h.t.Fatalf("%s: code = %d, want %d (%s)", what, r.Error.Code, CodePolicyDenied, r.Error.Message)
	}
	if !strings.Contains(r.Error.Message, "delegation refused") {
		h.t.Fatalf("%s: message = %q", what, r.Error.Message)
	}
	var d map[string]any
	if err := json.Unmarshal(r.Error.Data, &d); err != nil || d["error"] != "delegation_failed" {
		h.t.Fatalf("%s: data = %s", what, r.Error.Data)
	}
}

func scopeWordOf(t *testing.T, doc string) string {
	t.Helper()
	return strings.TrimPrefix(controlWord(t, doc), "scope = ")
}

// ---- B1, path 1: a budget word in the delegate scope ----------------------

func TestDelegateRefusesABudgetWordInTheScope(t *testing.T) {
	h := newDelegHarness(t)
	parent := h.parent("own-1")
	otherWord := scopeWordOf(t, budgetDoc(budgetB, "1000000", ""))
	sameWord := scopeWordOf(t, budgetDoc(budgetA, "9999999", ""))
	capWord := scopeWordOf(t, budgetDoc(budgetB, "1000000", "999999"))

	for name, scope := range map[string]string{
		"only the word":            otherWord,
		"after a tool":             "place_crypto_order," + otherWord,
		"before a tool":            otherWord + ",place_crypto_order",
		"space separated":          "place_crypto_order " + otherWord,
		"the parent's own budget":  sameWord,
		"a symbol cap inside it":   capWord,
		"marker next to the word":  scopeMarker + "," + otherWord,
		"unreadable control word":  "argrules:ctl:!!!not-base64!!!",
		"control word, odd casing": strings.ToUpper(otherWord[:12]) + otherWord[12:],
		"two words, one a budget":  scopeWordOf(t, `{"v":1,"spend_limits":[{"tools":["place_crypto_order"],"field":"dollar_amount","max":"5","window":"day"}]}`) + "," + otherWord,
	} {
		for _, budget := range []int64{0, 5} {
			before := h.snap("own-1")
			resp := h.delegate(parent, budget, scope, "child")
			h.wantRefused(resp, name)
			if after := h.snap("own-1"); !before.equal(after) {
				t.Fatalf("%s (budget %d): a counter, the parent's credits or the upstream moved: %+v -> %+v", name, budget, before, after)
			}
		}
	}
	if evs := h.delegationEvents(); len(evs) != 0 {
		t.Fatalf("a refused delegation published %d delegation event(s)", len(evs))
	}
}

// A parent with no budget at all cannot be given one through the scope either.
func TestDelegateScopeCannotGiveAPlainTokenABudget(t *testing.T) {
	h := newDelegHarness(t)
	svc, plain := sealedBudgetToken(t, "tenant-1", "own-9", bothTools,
		`{"v":1,"spend_limits":[{"tools":["place_crypto_order"],"field":"dollar_amount","max":"200","window":"day"}]}`)
	h.svc = svc
	_ = h.proxy.budget.Initialize(context.Background(), "own-9", parentCreds)
	before := h.snap("own-9")
	h.wantRefused(h.delegate(plain, 5, scopeWordOf(t, budgetDoc(budgetB, "1000000", "")), "child"), "plain parent")
	if after := h.snap("own-9"); !before.equal(after) {
		t.Fatalf("state moved: %+v -> %+v", before, after)
	}
}

// ---- B1, path 2: a budget word appended offline to the parent -------------

func TestDelegateRefusesAParentWithAnOfflineBudgetWord(t *testing.T) {
	h := newDelegHarness(t)
	parent := h.parent("own-1")

	evil := map[string]string{
		"another budget":          appendCaveat(t, h.svc, parent, controlWord(t, budgetDoc(budgetB, "1000000", ""))),
		"the same budget, raised": appendCaveat(t, h.svc, parent, controlWord(t, budgetDoc(budgetA, "9999999", ""))),
		"a budget with a BTC cap": appendCaveat(t, h.svc, parent, controlWord(t, budgetDoc(budgetB, "1000000", "999999"))),
		"after a plain narrowing": appendCaveat(t, h.svc, parent, "scope = place_crypto_order", controlWord(t, budgetDoc(budgetB, "1000000", ""))),
	}
	for name, tok := range evil {
		for _, scope := range []string{"", "place_crypto_order"} {
			for _, budget := range []int64{0, 5} {
				before := h.snap("own-1")
				h.wantRefused(h.delegate(tok, budget, scope, "child"), name)
				if after := h.snap("own-1"); !before.equal(after) {
					t.Fatalf("%s: state moved: %+v -> %+v", name, before, after)
				}
			}
		}
		// The same holder, going straight to tools/call, still cannot spend
		// from it (round 1), so the delegation was the only way in.
		if r := h.order(tok, "BTC", "5"); r.Error == nil {
			t.Fatalf("%s: tools/call spent through an unsealed reference", name)
		}
	}
	if len(h.store.Keys()) != 0 || h.up.calls.Load() != 0 {
		t.Fatalf("counters %v / upstream calls %d", h.store.Keys(), h.up.calls.Load())
	}
	// The untouched sealed parent still delegates.
	if r := h.delegate(parent, 5, "", "child"); r.Error != nil {
		t.Fatalf("the sealed parent was refused: %+v", r.Error)
	}
}

// A parent with an offline-appended budget word and no budget of its own.
func TestDelegateRefusesAnOfflineBudgetWordOnATokenWithNoBudget(t *testing.T) {
	h := newDelegHarness(t)
	svc, tok := mintBudgetToken(t, argTestRoot, bothTools, "tenant-1", "own-3", "50")
	h.svc = svc
	_ = h.proxy.budget.Initialize(context.Background(), "own-3", parentCreds)
	evil := appendCaveat(t, svc, tok, controlWord(t, budgetDoc(budgetB, "1000000", "")))
	before := h.snap("own-3")
	h.wantRefused(h.delegate(evil, 5, "", "child"), "no-budget parent")
	if after := h.snap("own-3"); !before.equal(after) {
		t.Fatalf("state moved: %+v -> %+v", before, after)
	}
	if r := h.order(evil, "BTC", "5"); r.Error == nil || h.up.calls.Load() != 0 {
		t.Fatalf("the token reached the upstream (calls %d)", h.up.calls.Load())
	}
}

// ---- B1, path 3: the parent's sealed budget passes to the child unchanged ---

func TestSealedParentDelegatesAndTheChildSpendsFromTheSameBudget(t *testing.T) {
	h := newDelegHarness(t)
	parent := h.parent("own-1")

	// Narrow to one tool and name it: a plain scope is fine.
	child := childToken(t, h.delegate(parent, 10, "place_crypto_order", "child"))

	key := func() (string, int64) {
		keys := h.store.Keys()
		if len(keys) != 1 || !strings.Contains(keys[0], ":b:"+budgetA+":") {
			t.Fatalf("counters = %v, want exactly the budget's", keys)
		}
		return keys[0], h.store.TotalOf(keys[0])
	}

	// One counter move per call, on the budget's counter, from either token.
	if r := h.order(child, "BTC", "10"); r.Error != nil {
		t.Fatalf("child order refused: %+v", r.Error)
	}
	if k, v := key(); v != 10*units {
		t.Fatalf("after the child's call %s = %d", k, v)
	}
	if r := h.order(parent, "ETH", "15"); r.Error != nil {
		t.Fatalf("parent order refused: %+v", r.Error)
	}
	if k, v := key(); v != 25*units {
		t.Fatalf("after the parent's call %s = %d", k, v)
	}
	if h.up.calls.Load() != 2 || h.up.executed.Load() != 25 {
		t.Fatalf("upstream: %d calls, $%d", h.up.calls.Load(), h.up.executed.Load())
	}
	// The child is held to the same room: $200 - $25 = $175.
	r := h.order(child, "BTC", "180")
	d := errData(t, r)
	if d["error"] != toolcontrols.DenialCodeSpendLimit || d["shared"] != true || d["remaining"] != "175" {
		t.Fatalf("over-budget child call: %s", r.Error.Data)
	}
	if _, v := key(); v != 25*units || h.up.calls.Load() != 2 {
		t.Fatalf("a refused call moved something: %d / %d calls", v, h.up.calls.Load())
	}

	cmac, err := h.svc.Decode(child)
	if err != nil {
		t.Fatal(err)
	}
	pmac, _ := h.svc.Decode(parent)

	// Same reference, same place: the child's first caveats are the parent's.
	for i, c := range pmac.Caveats {
		if cmac.Caveats[i] != c {
			t.Fatalf("caveat %d changed: %q -> %q", i, c, cmac.Caveats[i])
		}
	}

	// The child cannot drop the budget: a token without the control word does
	// not verify (the chain signature covers it).
	var dropped []string
	for _, c := range cmac.Caveats {
		if !strings.HasPrefix(c, "scope = "+"argrules:ctl:") {
			dropped = append(dropped, c)
		}
	}
	if len(dropped) == len(cmac.Caveats) {
		t.Fatal("the child carries no control word")
	}
	forged := *cmac
	forged.Caveats = dropped
	droppedTok := h.svc.Encode(&forged) // keeps the child's signature
	before := h.snap("own-1")
	// A token that fails verification is refused at the door (401) or in the
	// reply; either way nothing is spent.
	if r, err := h.proxy.handleRequest(context.Background(), argCall(9, droppedTok, "place_crypto_order", orderSym("BTC", "1"))); err != nil || r == nil || r.Error == nil {
		t.Fatalf("a child without the budget's control word was accepted: %v %v", r, err)
	}
	// Dropped and re-signed by the holder (who has no root key) fails too.
	forged.Signature = strings.Repeat("0", len(cmac.Signature))
	if r, err := h.proxy.handleRequest(context.Background(), argCall(10, h.svc.Encode(&forged), "place_crypto_order", orderSym("BTC", "1"))); err != nil || r == nil || r.Error == nil {
		t.Fatalf("a re-signed child was accepted: %v %v", r, err)
	}
	if after := h.snap("own-1"); !before.equal(after) {
		t.Fatalf("state moved: %+v -> %+v", before, after)
	}

	// The child cannot swap it: a new budget word it appends is unsealed.
	for name, word := range map[string]string{
		"another budget":   controlWord(t, budgetDoc(budgetB, "1000000", "")),
		"a raised maximum": controlWord(t, budgetDoc(budgetA, "9999999", "")),
	} {
		swapped := appendCaveat(t, h.svc, child, word)
		before := h.snap("own-1")
		if r := h.order(swapped, "BTC", "1"); r.Error == nil {
			t.Fatalf("%s: accepted", name)
		}
		h.wantRefused(h.delegate(swapped, 3, "", "grandchild"), name)
		if after := h.snap("own-1"); !before.equal(after) {
			t.Fatalf("%s: state moved: %+v -> %+v", name, before, after)
		}
	}

	// The child delegates on: the grandchild stays in the budget, and the same
	// refusals apply at every level.
	grand := childToken(t, h.delegate(child, 4, "", "grandchild"))
	if r := h.order(grand, "BTC", "5"); r.Error != nil {
		t.Fatalf("grandchild order refused: %+v", r.Error)
	}
	if _, v := key(); v != 30*units {
		t.Fatalf("after the grandchild's call: %d", v)
	}
	h.wantRefused(h.delegate(child, 1, scopeWordOf(t, budgetDoc(budgetB, "9", "")), "x"), "child with a budget word")
	h.wantRefused(h.delegate(grand, 1, scopeWordOf(t, budgetDoc(budgetB, "9", "")), "x"), "grandchild with a budget word")
}

// A delegation that carves no credits (an observe-only child) still carries the
// parent's budget unchanged and seals nothing new.
func TestZeroCreditDelegationKeepsTheParentsBudget(t *testing.T) {
	h := newDelegHarness(t)
	parent := h.parent("own-1")
	child := childToken(t, h.delegate(parent, 0, "place_crypto_order", "watcher"))
	if r := h.order(child, "BTC", "7"); r.Error != nil {
		t.Fatalf("refused: %+v", r.Error)
	}
	keys := h.store.Keys()
	if len(keys) != 1 || h.store.TotalOf(keys[0]) != 7*units || !strings.Contains(keys[0], ":b:"+budgetA+":") {
		t.Fatalf("counters = %v", keys)
	}
}

// ---- the proof before the seal ---------------------------------------------

// childKeepsOnlyTheParentsSealedBudget is the check that stands between a
// holder-chosen caveat and the seal. It must say no to a child whose prefix
// carries a budget reference the parent's issuer seal does not cover, or whose
// first caveats are not the parent's.
func TestSealProofRefusesAHolderBudgetInTheChildPrefix(t *testing.T) {
	svc, tok := sealedBudgetToken(t, "tenant-1", "own-1", bothTools, budgetDoc(budgetA, "200", ""))
	parent, _ := svc.Decode(tok)
	svc = svc.AcceptingArgumentRules()

	honest, err := svc.Delegate(tok, []string{"parent = x", "budget_id = del-aaaaaaaaaaaaaaaaaaaaaaaa", "budget_limit = 5.00"})
	if err != nil {
		t.Fatal(err)
	}
	if !childKeepsOnlyTheParentsSealedBudget(svc, parent, honest) {
		t.Fatal("an honest child was refused")
	}
	for name, extra := range map[string][]string{
		"a holder budget before the seal":   {"parent = x", controlWord(t, budgetDoc(budgetB, "9", ""))},
		"a holder budget with a raised max": {"parent = x", controlWord(t, budgetDoc(budgetA, "9999999", ""))},
	} {
		child, err := svc.Delegate(tok, extra)
		if err != nil {
			t.Fatal(err)
		}
		if childKeepsOnlyTheParentsSealedBudget(svc, parent, child) {
			t.Fatalf("%s: the proof passed", name)
		}
	}
	// A child that changed one of the parent's own words.
	changed := *honest
	changed.Caveats = append([]string{}, honest.Caveats...)
	for i, c := range changed.Caveats {
		if strings.HasPrefix(c, "scope = argrules:ctl:") {
			changed.Caveats[i] = controlWord(t, budgetDoc(budgetA, "9999999", ""))
		}
	}
	if childKeepsOnlyTheParentsSealedBudget(svc, parent, &changed) {
		t.Fatal("a child that rewrote the parent's budget word passed")
	}
	if childKeepsOnlyTheParentsSealedBudget(svc, nil, honest) || childKeepsOnlyTheParentsSealedBudget(svc, parent, nil) {
		t.Fatal("nil passed")
	}
}

// ---- direct: the refusal happens before any reservation --------------------

func TestDelegateRefusesBeforeAnyReservationDirectly(t *testing.T) {
	svc, tok := sealedBudgetToken(t, "tenant-1", "own-1", bothTools, budgetDoc(budgetA, "200", ""))
	evil := appendCaveat(t, svc, tok, controlWord(t, budgetDoc(budgetB, "1000000", "")))
	mac, err := svc.AcceptingArgumentRules().Verify(evil)
	if err != nil {
		t.Fatal(err)
	}
	info, err := FillTokenInfo(svc.AcceptingArgumentRules(), mac, evil)
	if err != nil {
		t.Fatal(err)
	}
	credits := &spyBudget{InMemoryBudgetEnforcer: NewInMemoryBudgetEnforcer()}
	_ = credits.Initialize(context.Background(), info.BudgetID, 50)
	d := NewDelegator(svc, credits)
	pub := NewChannelPublisher(4)
	d.SetEventPublisher(pub)

	if _, err := d.Delegate(context.Background(), info, &DelegateParams{Budget: 5, Label: "c"}); err == nil || !isDelegationBudgetRefusal(err) {
		t.Fatalf("err = %v", err)
	}
	if credits.spends != 0 || credits.inits != 1 {
		t.Fatalf("the refused delegation touched the credit budget: %d spends, %d inits", credits.spends, credits.inits)
	}
	scope := scopeWordOf(t, budgetDoc(budgetB, "1", ""))
	svc2, tok2 := sealedBudgetToken(t, "tenant-1", "own-2", bothTools, budgetDoc(budgetA, "200", ""))
	mac2, _ := svc2.AcceptingArgumentRules().Verify(tok2)
	info2, _ := FillTokenInfo(svc2.AcceptingArgumentRules(), mac2, tok2)
	_ = credits.Initialize(context.Background(), info2.BudgetID, 50)
	d2 := NewDelegator(svc2, credits)
	if _, err := d2.Delegate(context.Background(), info2, &DelegateParams{Budget: 5, Scope: scope}); err == nil || !isDelegationBudgetRefusal(err) {
		t.Fatalf("err = %v", err)
	}
	if credits.spends != 0 {
		t.Fatalf("a refused delegation reserved credits: %d", credits.spends)
	}
	select {
	case ev := <-pub.Events():
		t.Fatalf("an event was published: %+v", ev)
	default:
	}
	// A hand-built identity cannot skip the check: it reads the token itself.
	forged := *info
	forged.SealedPrefix = len(mac.Caveats) // claims everything is sealed
	if _, err := d.Delegate(context.Background(), &forged, &DelegateParams{Budget: 5}); err == nil || !isDelegationBudgetRefusal(err) {
		t.Fatalf("a TokenInfo that claims a full seal skipped the check: %v", err)
	}
}

type spyBudget struct {
	*InMemoryBudgetEnforcer
	spends, inits int
}

func (s *spyBudget) Spend(ctx context.Context, id, tool string, cost int64, req string) (*BudgetResult, error) {
	s.spends++
	return s.InMemoryBudgetEnforcer.Spend(ctx, id, tool, cost, req)
}

func (s *spyBudget) Initialize(ctx context.Context, id string, credits int64) error {
	s.inits++
	return s.InMemoryBudgetEnforcer.Initialize(ctx, id, credits)
}

// ---- no raw scope in events or logs -----------------------------------------

func TestDelegationNeverRecordsTheRawScope(t *testing.T) {
	h := newDelegHarness(t)
	parent := h.parent("own-1")
	word := scopeWordOf(t, budgetDoc(budgetB, "1000000", ""))
	payload := strings.TrimPrefix(word, "argrules:ctl:")

	// Refused: scope with a budget word and a marker.
	refusedScope := scopeMarker + "," + word
	h.wantRefused(h.delegate(parent, 5, refusedScope, "child"), "refused")
	if evs := h.delegationEvents(); len(evs) != 0 {
		t.Fatalf("a refused call published %d event(s)", len(evs))
	}
	surfaces := h.logs.String()
	for _, bad := range []string{scopeMarker, word, payload, refusedScope} {
		if strings.Contains(surfaces, bad) {
			t.Fatalf("the log carries %q:\n%s", bad, surfaces)
		}
	}
	// The refusal says it logged a hash and a word count.
	sum := sha256.Sum256([]byte(refusedScope))
	if !strings.Contains(surfaces, hex.EncodeToString(sum[:])) || !strings.Contains(surfaces, `"scopeWords":`) {
		t.Fatalf("the refusal log has no hash or word count:\n%s", surfaces)
	}
	if h.up.calls.Load() != 0 {
		t.Fatal("upstream contacted")
	}

	// Accepted: the event carries a hash and a count, never the text.
	h.logs.Reset()
	const accepted = "place_crypto_order"
	childToken(t, h.delegate(parent, 5, accepted, "child"))
	evs := h.delegationEvents()
	if len(evs) != 1 {
		t.Fatalf("delegation events = %d", len(evs))
	}
	data := evs[0].Data
	if _, has := data["scope"]; has {
		t.Fatalf("the event still carries a raw scope: %v", data["scope"])
	}
	want := sha256.Sum256([]byte(accepted))
	if data[EventScopeSHA256] != hex.EncodeToString(want[:]) || data[EventScopeWords] != 1 {
		t.Fatalf("event scope fields = %v / %v", data[EventScopeSHA256], data[EventScopeWords])
	}
	// Everything but the child's own effective tool scope (published for the
	// gateway writer, read from the child token, not from the request).
	delete(data, EventChildScopes)
	raw, _ := json.Marshal(data)
	if strings.Contains(string(raw), accepted) || strings.Contains(h.logs.String(), accepted) {
		t.Fatalf("the raw scope reached the event or the log:\n%s\n%s", raw, h.logs.String())
	}
}

func TestScopeHashAndWordCount(t *testing.T) {
	if scopeWordCount("") != 0 || scopeWordCount("a") != 1 || scopeWordCount("a,b c") != 3 || scopeWordCount(" , ") != 0 {
		t.Fatal("word count")
	}
	sum := sha256.Sum256([]byte("a,b"))
	if scopeSHA256("a,b") != hex.EncodeToString(sum[:]) {
		t.Fatal("hash")
	}
}
