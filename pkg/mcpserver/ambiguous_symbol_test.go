package mcpserver

// Round 2, blocker B3: a BTCUSD refusal says how much is left.
//
// After $40 of BTC against a $50 cap, an order for "BTCUSD" (no hyphen, so it
// could be the coin or another pair) is refused and asked to use "BTC". The
// refusal names BTC, gives that cap's read-only remaining, and ends with the
// same words the other cap refusals use. An unsealed budget reference never
// reads a counter on this path.

import (
	"context"
	"encoding/json"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/satgate-io/satgate/pkg/toolcontrols"
)

// countingStore counts every call that reaches the store.
type countingStore struct {
	toolcontrols.SpendStore
	reserves, totals, releases atomic.Int64
}

func (c *countingStore) Reserve(ctx context.Context, r toolcontrols.Reservation) (toolcontrols.ReserveResult, error) {
	c.reserves.Add(1)
	return c.SpendStore.Reserve(ctx, r)
}
func (c *countingStore) Total(ctx context.Context, k string) (int64, error) {
	c.totals.Add(1)
	return c.SpendStore.Total(ctx, k)
}
func (c *countingStore) Release(ctx context.Context, r toolcontrols.Reservation) error {
	c.releases.Add(1)
	return c.SpendStore.Release(ctx, r)
}

func (c *countingStore) touched() int64 {
	return c.reserves.Load() + c.totals.Load() + c.releases.Load()
}

func TestBTCUSDRefusalOverHTTPNamesBTCAndWhatIsLeft(t *testing.T) {
	h := newDelegHarness(t)
	_, tok := sealedBudgetToken(t, "tenant-1", "own-1", bothTools, budgetDoc(budgetA, "200", "50"))
	rec := &lockedRecorder{}
	h.proxy.SetEvidenceRecorder(rec)

	if r := h.order(tok, "BTC", "40"); r.Error != nil {
		t.Fatalf("the $40 order was refused: %+v", r.Error)
	}
	before := h.snap("own-1")

	r := h.order(tok, "BTCUSD", "20")
	if r.Error == nil {
		t.Fatalf("BTCUSD was not refused: %s", r.Result)
	}
	if r.Error.Code != CodePolicyDenied {
		t.Fatalf("code = %d", r.Error.Code)
	}
	var d map[string]any
	if err := json.Unmarshal(r.Error.Data, &d); err != nil {
		t.Fatal(err)
	}
	if d["error"] != toolcontrols.DenialCodeSpendLimit || d["shared"] != true || d["symbol"] != "BTC" || d["remaining"] != "10" {
		t.Fatalf("data = %s", r.Error.Data)
	}
	if d["limit"] != "50" || d["field"] != "dollar_amount" || d["window"] != "day" {
		t.Fatalf("data = %s", r.Error.Data)
	}
	for _, want := range []string{`send the symbol as "BTC"`, "$10 is left today for BTC in the budget shared by your agents"} {
		if !strings.Contains(r.Error.Message, want) {
			t.Fatalf("message = %q, missing %q", r.Error.Message, want)
		}
	}
	// Same closing words as the ordinary cap refusal.
	over := h.order(tok, "BTC", "20")
	if over.Error == nil || !strings.HasSuffix(over.Error.Message, "$10 is left today for BTC in the budget shared by your agents") ||
		!strings.HasSuffix(r.Error.Message, "$10 is left today for BTC in the budget shared by your agents") {
		t.Fatalf("closing words differ:\n%q\n%q", r.Error.Message, over.Error.Message)
	}
	// Reading is read only: nothing moved, and the total the gate took first was given back.
	if after := h.snap("own-1"); !before.equal(after) {
		t.Fatalf("a refused call moved something: %+v -> %+v", before, after)
	}
	// Neither id nor name appears.
	if strings.Contains(string(r.Error.Data), budgetA) || strings.Contains(r.Error.Message, budgetA) {
		t.Fatal("the budget id reached the agent")
	}
}

func TestBTCUSDRefusalReceiptCarriesSymbolAndSharedButNotRemaining(t *testing.T) {
	h := newBudgetHarness(t)
	_, tok := sealedBudgetToken(t, "tenant-1", "own-1", bothTools, budgetDoc(budgetA, "200", "50"))
	h.call(t, tok, 1, "place_crypto_order", orderSym("BTC", "40"))
	r := h.call(t, tok, 2, "place_crypto_order", orderSym("BTCUSD", "20"))
	if r.Error == nil || errData(t, r)["remaining"] != "10" {
		t.Fatalf("response = %+v", r)
	}
	if len(h.rec.decisions) != 2 {
		t.Fatalf("decisions = %d", len(h.rec.decisions))
	}
	d := h.rec.decisions[1]
	if d.Decision != "denied" || d.DenialCode != toolcontrols.DenialCodeSpendLimit {
		t.Fatalf("decision = %+v", d)
	}
	if d.ControlSymbol != "BTC" || !d.ControlShared {
		t.Fatalf("receipt projection: symbol %q shared %v", d.ControlSymbol, d.ControlShared)
	}
	if d.ControlLimit != "50" || d.ControlWindow != "day" || d.ArgumentField != "dollar_amount" {
		t.Fatalf("receipt projection: %+v", d)
	}
	// remaining stays response-only: the receipt input has no spend-balance
	// key. (remaining_credits and remaining_before_credits are the older
	// credit-budget fields, not the spend room.)
	raw, _ := json.Marshal(d)
	var keys map[string]json.RawMessage
	_ = json.Unmarshal(raw, &keys)
	for k := range keys {
		if strings.Contains(k, "remaining") && k != "remaining_credits" && k != "remaining_before_credits" {
			t.Fatalf("a spend balance reached the receipt input as %q: %s", k, raw)
		}
	}
	// The same fields as the plain cap refusal's receipt.
	h.call(t, tok, 3, "place_crypto_order", orderSym("BTC", "90"))
	plain := h.rec.decisions[2]
	if plain.ControlSymbol != d.ControlSymbol || plain.ControlShared != d.ControlShared || plain.ControlLimit != d.ControlLimit {
		t.Fatalf("receipts differ: %+v vs %+v", plain, d)
	}
}

// A cap with no budget behind it names its symbol and what is left too.
func TestBTCUSDRefusalForACapWithNoBudget(t *testing.T) {
	h := newBudgetHarness(t)
	doc := `{"v":1,"spend_limits":[{"tools":["place_crypto_order"],"field":"dollar_amount","max":"50","window":"day","symbols":["BTC"]}]}`
	_, tok := sealedBudgetToken(t, "tenant-1", "own-1", bothTools, doc)
	h.call(t, tok, 1, "place_crypto_order", orderSym("BTC", "40"))
	r := h.call(t, tok, 2, "place_crypto_order", orderSym("BTCUSD", "20"))
	d := errData(t, r)
	if d["symbol"] != "BTC" || d["remaining"] != "10" || d["shared"] != nil {
		t.Fatalf("data = %s", r.Error.Data)
	}
	if !strings.HasSuffix(r.Error.Message, "$10 is left today for BTC") {
		t.Fatalf("message = %q", r.Error.Message)
	}
}

// An unsealed reference never reads a counter, BTCUSD or not.
func TestBTCUSDOnAnUnsealedReferenceNeverReadsACounter(t *testing.T) {
	proxy, _, _ := newArgProxy(t)
	rr := &scriptedRouter{}
	proxy.SetUpstreamRouter(rr)
	proxy.SetEvidenceRecorder(&lockedRecorder{})
	mem := toolcontrols.NewMemoryStore(nil)
	cs := &countingStore{SpendStore: mem}
	proxy.SetCallGate(newTestSpendGate(cs))
	h := &budgetHarness{proxy: proxy, rr: rr, store: mem, rec: &lockedRecorder{}}

	svc, sealed := sealedBudgetToken(t, "tenant-1", "own-1", bothTools, budgetDoc(budgetA, "200", "50"))
	// The holder appends a budget reference offline: not sealed.
	evil := appendCaveat(t, svc, sealed, controlWord(t, budgetDoc(budgetB, "1000000", "50")))
	for i, sym := range []string{"BTCUSD", "btcusd", " BTCUSD ", "BTC", "BTC-USD", "ETH"} {
		r := h.call(t, evil, 10+i, "place_crypto_order", orderSym(sym, "5"))
		if r.Error == nil {
			t.Fatalf("%q: an unsealed reference was honoured", sym)
		}
		d := errData(t, r)
		if _, has := d["remaining"]; has {
			t.Fatalf("%q: an unsealed reference's refusal carries a remaining: %s", sym, r.Error.Data)
		}
		if _, has := d["symbol"]; has {
			t.Fatalf("%q: an unsealed reference's refusal carries a symbol: %s", sym, r.Error.Data)
		}
	}
	if cs.touched() != 0 {
		t.Fatalf("an unsealed reference reached the store: %d reserves, %d reads, %d releases", cs.reserves.Load(), cs.totals.Load(), cs.releases.Load())
	}
	if rr.forwarded() != 0 {
		t.Fatal("forwarded")
	}
	// And the sealed token on the same store does read (the counter is live).
	h.call(t, sealed, 30, "place_crypto_order", orderSym("BTC", "40"))
	r := h.call(t, sealed, 31, "place_crypto_order", orderSym("BTCUSD", "5"))
	if errData(t, r)["remaining"] != "10" || cs.totals.Load() == 0 {
		t.Fatalf("sealed BTCUSD: %s (reads %d)", r.Error.Data, cs.totals.Load())
	}
}

// If the counter cannot be read the refusal stands, without a made-up balance.
func TestBTCUSDRefusalStandsWhenTheCounterCannotBeRead(t *testing.T) {
	h := newBudgetHarness(t)
	_, tok := sealedBudgetToken(t, "tenant-1", "own-1", bothTools, budgetDoc(budgetA, "200", "50"))
	h.call(t, tok, 1, "place_crypto_order", orderSym("BTC", "40"))
	h.store.Err = nil
	failing := &readFailsStore{SpendStore: h.store}
	h.proxy.SetCallGate(newTestSpendGate(failing))
	r := h.call(t, tok, 2, "place_crypto_order", orderSym("BTCUSD", "5"))
	d := errData(t, r)
	if d["error"] != toolcontrols.DenialCodeSpendLimit || d["symbol"] != "BTC" {
		t.Fatalf("data = %s", r.Error.Data)
	}
	if _, has := d["remaining"]; has || strings.Contains(r.Error.Message, "is left") {
		t.Fatalf("a balance was invented: %s / %s", r.Error.Data, r.Error.Message)
	}
}

type readFailsStore struct{ toolcontrols.SpendStore }

func (readFailsStore) Total(context.Context, string) (int64, error) {
	return 0, toolcontrols.ErrStoreUnavailable
}
