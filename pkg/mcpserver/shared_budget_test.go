package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/satgate-io/satgate/pkg/macaroon"
	"github.com/satgate-io/satgate/pkg/toolcontrols"
)

const (
	budgetA = "aaaaaaaaaaaaaaaaaaaaaaaa"
	budgetB = "bbbbbbbbbbbbbbbbbbbbbbbb"
	// A name an owner might give a budget. It must never reach an agent.
	budgetMarker = "FAMILY-SECRET-MARKER-7731"
)

// budgetDoc200 is the document a server writes for a budget of $200 a day over
// place_crypto_order, with a $50 cap for BTC inside it.
func budgetDoc(id, total string, capBTC string) string {
	lim := func(max, extra string) string {
		return `{"tools":["place_crypto_order"],"field":"dollar_amount","max":"` + max + `","window":"day","tz":"America/New_York","budget":"` + id + `"` + extra + `}`
	}
	s := lim(total, "")
	if capBTC != "" {
		s += "," + lim(capBTC, `,"symbols":["BTC"]`)
	}
	return `{"v":1,"spend_limits":[` + s + `]}`
}

// sealedBudgetToken mints a token the way the server does: control words
// first, then the budget_id caveat and the budget_bind seal over all of it.
func sealedBudgetToken(t *testing.T, tenant, ownBudget, scope string, docs ...string) (*macaroon.Service, string) {
	t.Helper()
	svc, err := macaroon.NewService(argTestRoot)
	if err != nil {
		t.Fatal(err)
	}
	mac, err := svc.Mint(scope, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	mac.AddCaveat("tenant_id", tenant)
	mac.AddCaveat("budget_id", ownBudget)
	mac.AddCaveat("budget_limit", "50")
	for _, d := range docs {
		mac.Caveats = append(mac.Caveats, controlWord(t, d))
	}
	mac.Signature = svc.RecalculateSignature(mac)
	sealed, err := svc.AppendBudgetBind(svc.Encode(mac))
	if err != nil {
		t.Fatal(err)
	}
	return svc, svc.Encode(sealed)
}

type budgetHarness struct {
	proxy *Proxy
	rr    *scriptedRouter
	store *toolcontrols.MemoryStore
	rec   *lockedRecorder
	svc   *macaroon.Service
}

func newBudgetHarness(t *testing.T) *budgetHarness {
	t.Helper()
	proxy, _, _ := newArgProxy(t)
	rr := &scriptedRouter{}
	proxy.SetUpstreamRouter(rr)
	rec := &lockedRecorder{}
	proxy.SetEvidenceRecorder(rec)
	store := toolcontrols.NewMemoryStore(nil)
	proxy.SetCallGate(NewSpendGate(store))
	return &budgetHarness{proxy: proxy, rr: rr, store: store, rec: rec}
}

func (h *budgetHarness) call(t *testing.T, token string, id int, tool, args string) *Response {
	t.Helper()
	resp, err := h.proxy.handleRequest(context.Background(), argCall(id, token, tool, args))
	if err != nil || resp == nil {
		t.Fatalf("handleRequest: %v %v", resp, err)
	}
	return resp
}

func (h *budgetHarness) totals() map[string]int64 {
	out := map[string]int64{}
	for _, k := range h.store.Keys() {
		out[k] = h.store.TotalOf(k)
	}
	return out
}

func (h *budgetHarness) sum() int64 {
	var n int64
	for _, v := range h.totals() {
		n += v
	}
	return n
}

func orderSym(sym, amount string) string {
	return `{"side":"buy","type":"market","symbol":` + mustJSONString(sym) + `,"dollar_amount":"` + amount + `"}`
}

const bothTools = "place_crypto_order,place_equity_order"

// Two tokens minted separately, in one budget, spend from one counter.
func TestSharedBudgetIsOneCounterAcrossSeparatelyMintedTokens(t *testing.T) {
	h := newBudgetHarness(t)
	_, tokA := sealedBudgetToken(t, "tenant-1", "own-1", bothTools, budgetDoc(budgetA, "200", ""))
	_, tokB := sealedBudgetToken(t, "tenant-1", "own-2", bothTools, budgetDoc(budgetA, "200", ""))
	if tokA == tokB {
		t.Fatal("tokens are the same")
	}
	if r := h.call(t, tokA, 1, "place_crypto_order", order("120")); r.Error != nil {
		t.Fatalf("agent A refused: %v", r.Error)
	}
	if r := h.call(t, tokB, 1, "place_crypto_order", order("70")); r.Error != nil {
		t.Fatalf("agent B refused: %v", r.Error)
	}
	if len(h.totals()) != 1 || h.sum() != 190*units {
		t.Fatalf("counters = %v, want one holding 190", h.totals())
	}
	r := h.call(t, tokA, 2, "place_crypto_order", order("11"))
	if r.Error == nil {
		t.Fatal("agent A went past the shared total")
	}
	d := errData(t, r)
	if d["error"] != "TOOL_SPEND_LIMIT_DENIED" || d["shared"] != true || d["remaining"] != "10" || d["limit"] != "200" {
		t.Fatalf("data = %v", d)
	}
	if _, has := d["symbol"]; has {
		t.Fatalf("a total-only refusal carries a symbol: %v", d)
	}
	if want := "place_crypto_order: the limit shared by your agents is $200 a day on dollar_amount; this order would go over it; $10 is left today in the budget shared by your agents"; r.Error.Message != want {
		t.Fatalf("message = %q", r.Error.Message)
	}
	// Exactly the room left is allowed, from the other agent.
	if r := h.call(t, tokB, 2, "place_crypto_order", order("10")); r.Error != nil {
		t.Fatalf("the last $10 refused: %v", r.Error)
	}
	if r := h.call(t, tokB, 3, "place_crypto_order", order("0.01")); r.Error == nil {
		t.Fatal("went past the total")
	}
	// remaining never goes below zero and reads 0 when full.
	if d := errData(t, h.call(t, tokA, 4, "place_crypto_order", order("1"))); d["remaining"] != "0" {
		t.Fatalf("remaining = %v", d["remaining"])
	}
}

// Without a budget nothing changes: two tokens minted with the same document
// have separate counters.
func TestWithoutABudgetEachMintHasItsOwnCounter(t *testing.T) {
	h := newBudgetHarness(t)
	doc := `{"v":1,"spend_limits":[{"tools":["place_crypto_order"],"field":"dollar_amount","max":"200","window":"day"}]}`
	_, a := sealedBudgetToken(t, "tenant-1", "own-1", bothTools, doc)
	_, b := sealedBudgetToken(t, "tenant-1", "own-2", bothTools, doc)
	h.call(t, a, 1, "place_crypto_order", order("150"))
	h.call(t, b, 1, "place_crypto_order", order("150"))
	if len(h.totals()) != 2 {
		t.Fatalf("counters = %v, want two", h.totals())
	}
	if d := errData(t, h.call(t, a, 2, "place_crypto_order", order("100"))); d["error"] != "TOOL_SPEND_LIMIT_DENIED" {
		t.Fatalf("data = %v", d)
	} else if _, has := d["shared"]; has {
		t.Fatalf("a plain refusal says shared: %v", d)
	} else if _, has := d["remaining"]; has {
		t.Fatalf("a plain refusal says remaining: %v", d)
	}
}

// The counter is named b:<id>, with the tenant in the key.
func TestBudgetCounterKeyHasTenantAndBudget(t *testing.T) {
	h := newBudgetHarness(t)
	_, one := sealedBudgetToken(t, "tenant-1", "own-1", bothTools, budgetDoc(budgetA, "200", ""))
	_, two := sealedBudgetToken(t, "tenant-2", "own-2", bothTools, budgetDoc(budgetA, "200", ""))
	h.call(t, one, 1, "place_crypto_order", order("150"))
	h.call(t, two, 1, "place_crypto_order", order("150"))
	keys := h.store.Keys()
	if len(keys) != 2 {
		t.Fatalf("the same id under two tenants shares a counter: %v", keys)
	}
	day := time.Now().In(mustZone(t, "America/New_York")).Format("2006-01-02")
	want := []string{
		"satgate:spendlimit:tenant-1:b:" + budgetA + ":" + day,
		"satgate:spendlimit:tenant-2:b:" + budgetA + ":" + day,
	}
	for i := range want {
		if keys[i] != want[i] && !strings.HasPrefix(keys[i], want[i][:len(want[i])-10]) {
			t.Fatalf("key %d = %q, want %q", i, keys[i], want[i])
		}
	}
	// And tenant 2's spending does not use up tenant 1's room.
	if r := h.call(t, one, 2, "place_crypto_order", order("50")); r.Error != nil {
		t.Fatalf("tenant 1 refused after tenant 2 spent: %v", r.Error)
	}
}

func mustZone(t *testing.T, name string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(name)
	if err != nil {
		t.Fatal(err)
	}
	return loc
}

// BTC, btc, " BTC ", BTC-USD and btc-usd are one symbol; BTCUSD is refused.
func TestSymbolSpellingsHitOneCounter(t *testing.T) {
	h := newBudgetHarness(t)
	_, tok := sealedBudgetToken(t, "tenant-1", "own-1", bothTools, budgetDoc(budgetA, "200", "50"))
	spellings := []string{"BTC", "btc", " BTC ", "BTC-USD", "btc-usd"}
	for i, s := range spellings {
		if r := h.call(t, tok, i+1, "place_crypto_order", orderSym(s, "10")); r.Error != nil {
			t.Fatalf("%q refused: %v", s, r.Error)
		}
	}
	// Five orders of $10 in five spellings: one symbol counter holds all 50, the
	// total holds all 50.
	var symCounter, totalCounter int64
	for k, v := range h.totals() {
		if strings.Contains(k, ":s:") {
			symCounter += v
		} else {
			totalCounter += v
		}
	}
	if symCounter != 50*units || totalCounter != 50*units {
		t.Fatalf("symbol counter %d, total %d: %v", symCounter, totalCounter, h.totals())
	}
	for i, s := range spellings {
		r := h.call(t, tok, 10+i, "place_crypto_order", orderSym(s, "0.01"))
		d := errData(t, r)
		if d["error"] != "TOOL_SPEND_LIMIT_DENIED" || d["symbol"] != "BTC" || d["remaining"] != "0" || d["limit"] != "50" {
			t.Fatalf("%q after the cap: %v", s, d)
		}
		if _, shared := d["shared"]; !shared {
			t.Fatalf("a symbol cap inside a budget should say shared: %v", d)
		}
		if want := "$0 is left today for BTC in the budget shared by your agents"; !strings.HasSuffix(r.Error.Message, want) {
			t.Fatalf("message = %q", r.Error.Message)
		}
	}
	// ETH is not under the BTC cap; only the $200 total applies to it.
	if r := h.call(t, tok, 30, "place_crypto_order", orderSym("ETH-USD", "100")); r.Error != nil {
		t.Fatalf("ETH refused: %v", r.Error)
	}
	// BTCUSD: refused, asked for BTC.
	r := h.call(t, tok, 31, "place_crypto_order", orderSym("BTCUSD", "1"))
	if r.Error == nil {
		t.Fatal("BTCUSD was allowed")
	}
	if !strings.Contains(r.Error.Message, `send the symbol as "BTC"`) {
		t.Fatalf("message = %q", r.Error.Message)
	}
	if errData(t, r)["error"] != "TOOL_SPEND_LIMIT_DENIED" {
		t.Fatalf("data = %v", errData(t, r))
	}
	// ETHUSD is not ambiguous (ETH is not capped); it is its own symbol, ETHUSD.
	if r := h.call(t, tok, 32, "place_crypto_order", orderSym("ETHUSD", "1")); r.Error != nil {
		t.Fatalf("ETHUSD refused: %v", r.Error)
	}
}

// A covered call with a missing, non-string or non-matching symbol is refused.
func TestSymbolCapFailsClosed(t *testing.T) {
	h := newBudgetHarness(t)
	_, tok := sealedBudgetToken(t, "tenant-1", "own-1", bothTools, budgetDoc(budgetA, "200", "50"))
	for i, args := range []string{
		`{"side":"buy","dollar_amount":"5"}`,
		`{"side":"buy","symbol":null,"dollar_amount":"5"}`,
		`{"side":"buy","symbol":5,"dollar_amount":"5"}`,
		`{"side":"buy","symbol":["BTC"],"dollar_amount":"5"}`,
		`{"side":"buy","symbol":"","dollar_amount":"5"}`,
		`{"side":"buy","symbol":"BTC/USD","dollar_amount":"5"}`,
		`{"side":"buy","symbol":"BTC-USD-USD","dollar_amount":"5"}`,
		`{"side":"buy","symbol":"BT\u0421","dollar_amount":"5"}`,
		`{"side":"buy","symbol":"\u00a0BTC","dollar_amount":"5"}`,
		`{"side":"buy","Symbol":"ETH","dollar_amount":"5"}`,
		`{"side":"buy","symbol":"ETH","symbol":"BTC","dollar_amount":"5"}`,
		`{"side":"buy","symbol":"ETH","Symbol":"BTC","dollar_amount":"5"}`,
	} {
		r := h.call(t, tok, i+1, "place_crypto_order", args)
		if r.Error == nil {
			t.Errorf("accepted %s", args)
			continue
		}
		if errData(t, r)["error"] != "TOOL_SPEND_LIMIT_DENIED" {
			t.Errorf("%s: data = %s", args, r.Error.Data)
		}
	}
	if h.sum() != 0 || h.rr.forwarded() != 0 {
		t.Fatalf("a refused call counted or was forwarded: %v forwarded %d", h.totals(), h.rr.forwarded())
	}
}

// A symbol cap alone (no budget) counts per token and is also canonical.
func TestSymbolCapWithoutABudget(t *testing.T) {
	h := newBudgetHarness(t)
	doc := `{"v":1,"spend_limits":[{"tools":["place_crypto_order"],"field":"dollar_amount","max":"50","window":"day","symbols":["BTC"]}]}`
	_, tok := sealedBudgetToken(t, "tenant-1", "own-1", bothTools, doc)
	h.call(t, tok, 1, "place_crypto_order", orderSym("btc-usd", "30"))
	r := h.call(t, tok, 2, "place_crypto_order", orderSym("BTC", "30"))
	d := errData(t, r)
	if d["symbol"] != "BTC" || d["remaining"] != "20" {
		t.Fatalf("data = %v", d)
	}
	if _, has := d["shared"]; has {
		t.Fatalf("a plain symbol cap says shared: %v", d)
	}
	if want := "; $20 is left today for BTC"; !strings.HasSuffix(r.Error.Message, want) {
		t.Fatalf("message = %q", r.Error.Message)
	}
	if r := h.call(t, tok, 3, "place_crypto_order", orderSym("ETH", "500")); r.Error != nil {
		t.Fatalf("ETH touched by the BTC cap: %v", r.Error)
	}
}

// A symbol-cap refusal gives back what the total had already taken.
func TestSymbolCapRefusalGivesBackTheTotal(t *testing.T) {
	h := newBudgetHarness(t)
	_, tok := sealedBudgetToken(t, "tenant-1", "own-1", bothTools, budgetDoc(budgetA, "200", "50"))
	h.call(t, tok, 1, "place_crypto_order", orderSym("BTC", "40"))
	if h.sum() != 80*units {
		t.Fatalf("after the first order: %v", h.totals())
	}
	// 20 more BTC: the total (60 of 200) accepts, the BTC cap (60 of 50) refuses.
	r := h.call(t, tok, 2, "place_crypto_order", orderSym("BTC", "20"))
	if d := errData(t, r); d["symbol"] != "BTC" || d["remaining"] != "10" {
		t.Fatalf("data = %v", d)
	}
	var total int64
	for k, v := range h.totals() {
		if !strings.Contains(k, ":s:") {
			total = v
		}
	}
	if total != 40*units {
		t.Fatalf("the total kept the refused order: %d (%v)", total, h.totals())
	}
	// And the room is really there for another symbol.
	if r := h.call(t, tok, 3, "place_crypto_order", orderSym("ETH", "160")); r.Error != nil {
		t.Fatalf("room not given back: %v", r.Error)
	}
	if h.rr.forwarded() != 2 {
		t.Fatalf("forwarded %d", h.rr.forwarded())
	}
}

// The reverse order: the total refuses after the symbol cap took its share.
func TestTotalRefusalGivesBackTheSymbolCap(t *testing.T) {
	h := newBudgetHarness(t)
	doc := budgetDocOrdered(budgetA)
	_, tok := sealedBudgetToken(t, "tenant-1", "own-1", bothTools, doc)
	h.call(t, tok, 1, "place_crypto_order", orderSym("ETH", "90"))
	r := h.call(t, tok, 2, "place_crypto_order", orderSym("BTC", "20"))
	if r.Error == nil {
		t.Fatal("went past the total")
	}
	for k, v := range h.totals() {
		if strings.Contains(k, ":s:") && v != 0 {
			t.Fatalf("the symbol counter kept a refused order: %s=%d", k, v)
		}
	}
}

// budgetDocOrdered lists the symbol cap first and the total second.
func budgetDocOrdered(id string) string {
	lim := func(max, extra string) string {
		return `{"tools":["place_crypto_order"],"field":"dollar_amount","max":"` + max + `","window":"day","tz":"America/New_York","budget":"` + id + `"` + extra + `}`
	}
	return `{"v":1,"spend_limits":[` + lim("50", `,"symbols":["BTC"]`) + `,` + lim("100", "") + `]}`
}

// A holder can append caveats offline. An unsealed budget reference refuses
// every call it covers and never spends from or drains any budget.
func TestUnsealedBudgetReferenceIsRefusedAndNeverCounts(t *testing.T) {
	h := newBudgetHarness(t)
	svc, sealed := sealedBudgetToken(t, "tenant-1", "own-1", bothTools, budgetDoc(budgetA, "200", ""))

	// Offline: the holder appends a control caveat naming another budget.
	evil := appendCaveat(t, svc, sealed, controlWord(t, budgetDoc(budgetB, "1000000", "")))
	r := h.call(t, evil, 1, "place_crypto_order", order("5"))
	if r.Error == nil {
		t.Fatal("a token with an appended budget reference got through")
	}
	if errData(t, r)["error"] != "TOOL_SPEND_LIMIT_DENIED" {
		t.Fatalf("data = %s", r.Error.Data)
	}
	if h.sum() != 0 || len(h.totals()) != 0 || h.rr.forwarded() != 0 {
		t.Fatalf("an unsealed reference touched a counter or forwarded: %v / %d", h.totals(), h.rr.forwarded())
	}
	// It did not even take from the budget the token is properly in.
	for k := range h.totals() {
		if strings.Contains(k, budgetB) || strings.Contains(k, budgetA) {
			t.Fatalf("counter %s exists", k)
		}
	}
	// The sealed token itself still works.
	if r := h.call(t, sealed, 2, "place_crypto_order", order("5")); r.Error != nil {
		t.Fatalf("the sealed token was refused: %v", r.Error)
	}
	if h.sum() != 5*units {
		t.Fatalf("totals = %v", h.totals())
	}
	// A reference to the SAME budget, appended offline with a huge max, is
	// also refused: it cannot raise the room.
	big := appendCaveat(t, svc, sealed, controlWord(t, budgetDoc(budgetA, "9999999", "")))
	if r := h.call(t, big, 3, "place_crypto_order", order("1")); r.Error == nil {
		t.Fatal("an appended reference to the same budget got through")
	}
	if h.sum() != 5*units {
		t.Fatalf("totals = %v", h.totals())
	}
	// A call for a tool the appended limit does not cover is not refused by it.
	if r := h.call(t, evil, 4, "place_equity_order", order("5")); r.Error != nil {
		t.Fatalf("the appended reference refused a tool it does not cover: %v", r.Error)
	}
}

// A token minted with no seal at all, and a hand-built identity, cannot use a
// budget either.
func TestBudgetReferenceWithNoSealIsRefused(t *testing.T) {
	h := newBudgetHarness(t)
	svc, err := macaroon.NewService(argTestRoot)
	if err != nil {
		t.Fatal(err)
	}
	mac, _ := svc.Mint(bothTools, time.Now().Add(time.Hour))
	mac.AddCaveat("tenant_id", "tenant-1")
	mac.AddCaveat("budget_id", "own-1")
	mac.AddCaveat("budget_limit", "50")
	mac.Caveats = append(mac.Caveats, controlWord(t, budgetDoc(budgetA, "200", "")))
	mac.Signature = svc.RecalculateSignature(mac)
	if r := h.call(t, svc.Encode(mac), 1, "place_crypto_order", order("5")); r.Error == nil {
		t.Fatal("an unsealed token spent from a budget")
	}
	if len(h.totals()) != 0 {
		t.Fatalf("totals = %v", h.totals())
	}

	// A hand-built TokenInfo (no caveats read, no seal) is refused by the gate.
	d, _ := toolcontrols.ParseJSON([]byte(budgetDoc(budgetA, "200", "")))
	lim := d.SpendLimits
	lim[0].ID = "x"
	g := NewSpendGate(h.store)
	adm, err := g.Admit(context.Background(), GateCall{
		Token:  &TokenInfo{TenantID: "tenant-1", SpendLimits: lim},
		Tool:   "place_crypto_order",
		Params: json.RawMessage(`{"name":"place_crypto_order","arguments":` + order("5") + `}`),
	})
	if err != nil || adm == nil || adm.Refusal == nil {
		t.Fatalf("hand-built identity: %+v %v", adm, err)
	}
	if len(h.totals()) != 0 {
		t.Fatalf("totals = %v", h.totals())
	}
}

// The tenant is part of the key: a budget id under no tenant is refused.
func TestBudgetReferenceNeedsATenant(t *testing.T) {
	h := newBudgetHarness(t)
	_, tok := sealedBudgetToken(t, "", "own-1", bothTools, budgetDoc(budgetA, "200", ""))
	if r := h.call(t, tok, 1, "place_crypto_order", order("5")); r.Error == nil {
		t.Fatal("a budget without a tenant was honored")
	}
	if len(h.totals()) != 0 {
		t.Fatalf("totals = %v", h.totals())
	}
}

// Nothing an agent sees names the budget: not the id, not the owner's name.
func TestAgentSurfacesNeverNameTheBudget(t *testing.T) {
	var logs bytes.Buffer
	prev := log.Logger
	log.Logger = zerolog.New(&logs).Level(zerolog.DebugLevel)
	defer func() { log.Logger = prev }()

	h := newBudgetHarness(t)
	h.rr.replies = []func() (*Response, error){lostAfterSend}
	_, tok := sealedBudgetToken(t, "tenant-1", "own-1", bothTools, budgetDoc(budgetA, "200", "50"))
	var seen []string
	collect := func(r *Response) {
		b, _ := json.Marshal(r)
		seen = append(seen, string(b))
	}
	collect(h.call(t, tok, 1, "place_crypto_order", orderSym("BTC", "40")))  // outcome unknown: logs a warning
	collect(h.call(t, tok, 2, "place_crypto_order", orderSym("BTC", "40")))  // symbol cap
	collect(h.call(t, tok, 3, "place_crypto_order", orderSym("ETH", "400"))) // total
	collect(h.call(t, tok, 4, "place_crypto_order", orderSym("BTCUSD", "1")))
	collect(h.call(t, tok, 5, "place_crypto_order", `{"side":"buy","quantity":"1","symbol":"ETH"}`))
	for _, d := range h.rec.decisions {
		b, _ := json.Marshal(d)
		seen = append(seen, string(b))
	}
	seen = append(seen, logs.String())
	h.store.Err = fmt.Errorf("store down")
	collect(h.call(t, tok, 6, "place_crypto_order", orderSym("ETH", "1")))
	h.store.Err = nil
	// A release on a failed call logs through logCounter; force one.
	seen = append(seen, logs.String())
	for i, s := range seen {
		for _, bad := range []string{budgetA, budgetMarker, "b:" + budgetA, "satgate:spendlimit"} {
			if strings.Contains(s, bad) {
				t.Errorf("surface %d carries %q: %s", i, bad, s)
			}
		}
	}
	if len(h.rec.decisions) == 0 {
		t.Fatal("no receipt input was recorded")
	}
}

// Receipt input: the new fields are on the receipts of covered calls only.
func TestReceiptDetailOnlyForBudgetAndSymbolCalls(t *testing.T) {
	h := newBudgetHarness(t)
	_, shared := sealedBudgetToken(t, "tenant-1", "own-1", bothTools, budgetDoc(budgetA, "200", "50"))
	plainDoc := `{"v":1,"spend_limits":[{"tools":["place_crypto_order"],"field":"dollar_amount","max":"200","window":"day"}]}`
	_, plain := sealedBudgetToken(t, "tenant-1", "own-2", bothTools, plainDoc)

	h.call(t, shared, 1, "place_crypto_order", orderSym("BTC", "10")) // allowed, shared + symbol
	h.call(t, shared, 2, "place_crypto_order", orderSym("ETH", "10")) // allowed, shared only
	h.call(t, shared, 3, "place_crypto_order", orderSym("BTC", "90")) // refused by the symbol cap
	h.call(t, plain, 4, "place_crypto_order", orderSym("BTC", "10"))  // allowed, plain
	h.call(t, plain, 5, "place_crypto_order", orderSym("BTC", "999")) // refused, plain
	if len(h.rec.decisions) != 5 {
		t.Fatalf("decisions = %d", len(h.rec.decisions))
	}
	d := h.rec.decisions
	type want struct {
		decision, symbol, remaining string
		shared                      bool
	}
	for i, w := range []want{
		{"allowed", "BTC", "", true},
		{"allowed", "", "", true},
		{"denied", "BTC", "40", true},
		{"allowed", "", "", false},
		{"denied", "", "", false},
	} {
		if d[i].Decision != w.decision || d[i].ControlSymbol != w.symbol || d[i].ControlShared != w.shared {
			t.Errorf("decision %d = %+v", i, d[i])
		}
	}
	// The plain receipts' JSON carries none of the new keys at all.
	for _, i := range []int{3, 4} {
		b, _ := json.Marshal(d[i])
		for _, k := range []string{"control_shared", "control_symbol"} {
			if strings.Contains(string(b), k) {
				t.Errorf("plain decision %d carries %s: %s", i, k, b)
			}
		}
	}
}

// Two agents at once: many goroutines on two tokens in one budget never push
// the total past the maximum.
func TestRaceTwoAgentsOneBudgetNeverPassTheMax(t *testing.T) {
	store := toolcontrols.NewMemoryStore(nil)
	raceTwoAgentsOneBudget(t, store, func() int64 {
		var n int64
		for _, k := range store.Keys() {
			n += store.TotalOf(k)
		}
		return n
	})
}

// raceTwoAgentsOneBudget drives SpendGate with two tokens minted separately
// into one $200 budget from many goroutines, each asking for $3, and checks
// the total never passes $200, that exactly 66 orders ($198) are admitted, and
// that the total reads back the same through store.Total.
func raceTwoAgentsOneBudget(t *testing.T, store toolcontrols.SpendStore, total func() int64) {
	t.Helper()
	svcA, tokA := sealedBudgetToken(t, "tenant-1", "own-1", bothTools, budgetDoc(budgetA, "200", ""))
	_, tokB := sealedBudgetToken(t, "tenant-1", "own-2", bothTools, budgetDoc(budgetA, "200", ""))
	infos := make([]*TokenInfo, 0, 2)
	for _, tok := range []string{tokA, tokB} {
		mac, err := svcA.Decode(tok)
		if err != nil {
			t.Fatal(err)
		}
		info, err := FillTokenInfo(svcA, mac, tok)
		if err != nil {
			t.Fatal(err)
		}
		infos = append(infos, info)
	}
	g := NewSpendGate(store)
	var admitted, refused int64
	var wg sync.WaitGroup
	const workers, each = 16, 10
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < each; i++ {
				call := GateCall{
					Token:     infos[w%2],
					Tool:      "place_crypto_order",
					Params:    json.RawMessage(`{"name":"place_crypto_order","arguments":` + order("3") + `}`),
					RequestID: fmt.Sprintf("w%d-c%d", w, i),
					Now:       time.Now().UTC(),
				}
				adm, err := g.Admit(context.Background(), call)
				switch {
				case err != nil:
					t.Errorf("admit: %v", err)
				case adm != nil && adm.Refusal != nil:
					atomic.AddInt64(&refused, 1)
				default:
					atomic.AddInt64(&admitted, 1)
					if got := total(); got > 200*units {
						t.Errorf("total %d is past the maximum", got)
					}
				}
			}
		}(w)
	}
	wg.Wait()
	if admitted+refused != workers*each {
		t.Fatalf("admitted %d + refused %d", admitted, refused)
	}
	if admitted != 66 {
		t.Fatalf("admitted %d orders of $3 into a $200 budget, want 66", admitted)
	}
	if got := total(); got != 198*units {
		t.Fatalf("total = %d, want %d", got, 198*units)
	}
	// The read-only Total agrees with the counter.
	day := time.Now().UTC()
	l := infos[0].SpendLimits[0]
	id, _, _ := l.WindowBounds(day)
	read, err := store.Total(context.Background(), counterKey("tenant-1", counterName(l), id))
	if err != nil || read != 198*units {
		t.Fatalf("Total = %d %v", read, err)
	}
}

// An ask-me approval is checked when the call is forwarded: an approved call
// is refused if other agents used the room in the meantime.
func TestApprovedCallRefusedWhenOtherAgentsUsedTheRoom(t *testing.T) {
	const doc = `{"v":1,"approvals":[{"tools":["place_crypto_order"],"field":"dollar_amount","above":"50"}],"spend_limits":[{"tools":["place_crypto_order"],"field":"dollar_amount","max":"200","window":"day","tz":"America/New_York","budget":"` + budgetA + `"}]}`
	h := newBudgetHarness(t)
	approvals := toolcontrols.NewMemoryApprovalStore()
	gate := NewApprovalGate(approvals, &fakeNotifier{})
	gate.Synchronous = true
	clock := time.Now().UTC()
	gate.now = func() time.Time { return clock }
	h.proxy.SetCallGate(ChainGates(NewSpendGate(h.store), gate))

	_, asker := sealedBudgetToken(t, "tenant-1", "own-1", bothTools, doc)
	_, other := sealedBudgetToken(t, "tenant-1", "own-2", bothTools, doc)

	// 150 fits the room now, so it is held for the owner.
	r := h.call(t, asker, 1, "place_crypto_order", order("150"))
	id := errData(t, r)["approval_id"].(string)
	if err := approvals.Approve("tenant-1", id, "owner@example.com", clock); err != nil {
		t.Fatal(err)
	}
	// Meanwhile another agent in the same budget spends 100 (under the ask-me line).
	if r := h.call(t, other, 1, "place_crypto_order", order("50")); r.Error != nil {
		t.Fatal(r.Error)
	}
	if r := h.call(t, other, 2, "place_crypto_order", order("50")); r.Error != nil {
		t.Fatal(r.Error)
	}
	// The approved call is now over the budget: refused at forward time.
	r = h.call(t, asker, 2, "place_crypto_order", order("150"))
	d := errData(t, r)
	if d["error"] != "TOOL_SPEND_LIMIT_DENIED" || d["shared"] != true || d["remaining"] != "100" {
		t.Fatalf("data = %v", d)
	}
	if h.rr.forwarded() != 2 || h.sum() != 100*units {
		t.Fatalf("forwarded %d, counters %v", h.rr.forwarded(), h.totals())
	}
	// The owner's yes was not used up by the refusal; the room can come back.
	if row, _ := approvals.Row(id); row.State != toolcontrols.StateApproved {
		t.Fatalf("approval state = %s", row.State)
	}
}

// A covered call that sends a quantity instead of dollar_amount is refused,
// and the message says what to send.
func TestQuantityOrderIsRefusedAndToldToSendDollarAmount(t *testing.T) {
	h := newBudgetHarness(t)
	_, tok := sealedBudgetToken(t, "tenant-1", "own-1", bothTools, budgetDoc(budgetA, "200", "50"))
	r := h.call(t, tok, 1, "place_crypto_order", `{"side":"buy","type":"market","symbol":"BTC","quantity":"0.001"}`)
	if r.Error == nil {
		t.Fatal("a quantity order was allowed")
	}
	if !strings.Contains(r.Error.Message, "send dollar_amount instead of quantity") {
		t.Fatalf("message = %q", r.Error.Message)
	}
	if h.sum() != 0 || h.rr.forwarded() != 0 {
		t.Fatal("a refused order counted")
	}
	// Same text on a plain (non-budget) limit, which was refused before too.
	_, plain := sealedBudgetToken(t, "tenant-1", "own-2", bothTools, spendDoc200Day)
	r = h.call(t, plain, 2, "place_crypto_order", `{"side":"buy","symbol":"BTC","quantity":"1"}`)
	if r.Error == nil || !strings.Contains(r.Error.Message, "send dollar_amount instead of quantity") {
		t.Fatalf("plain limit: %+v", r.Error)
	}
	// A present but unreadable amount is not a quantity order: no hint.
	r = h.call(t, plain, 3, "place_crypto_order", `{"side":"buy","symbol":"BTC","dollar_amount":"abc"}`)
	if r.Error == nil || strings.Contains(r.Error.Message, "instead of quantity") {
		t.Fatalf("bad amount: %+v", r.Error)
	}
}

// The seal helper: a prefix covers exactly what came before the seal.
func TestBudgetSealedPrefix(t *testing.T) {
	svc, tok := sealedBudgetToken(t, "tenant-1", "own-1", bothTools, budgetDoc(budgetA, "200", ""))
	mac, _ := svc.Decode(tok)
	n := svc.BudgetSealedPrefix(mac)
	if n == 0 || n >= len(mac.Caveats) || !strings.HasPrefix(mac.Caveats[n], "budget_bind = ") {
		t.Fatalf("prefix = %d of %d", n, len(mac.Caveats))
	}
	appended := appendCaveat(t, svc, tok, controlWord(t, budgetDoc(budgetB, "9", "")))
	mac2, _ := svc.Decode(appended)
	if svc.BudgetSealedPrefix(mac2) != n {
		t.Fatalf("an appended caveat moved the sealed prefix to %d", svc.BudgetSealedPrefix(mac2))
	}
	// A token with no seal has none.
	plainSvc, plain := mintBudgetToken(t, argTestRoot, "x", "t", "", "")
	m3, _ := plainSvc.Decode(plain)
	if plainSvc.BudgetSealedPrefix(m3) != 0 {
		t.Fatal("unsealed token reports a prefix")
	}
}
