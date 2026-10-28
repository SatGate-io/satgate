package toolcontrols

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/satgate-io/satgate/pkg/argrules"
	"github.com/satgate-io/satgate/pkg/toolcontrols/internal/releasedv1"
	"github.com/satgate-io/satgate/pkg/toolcontrols/internal/releasedv2"
	"github.com/satgate-io/satgate/pkg/toolcontrols/internal/releasedv3"
)

// A made-up value schema for tests only: {"market","count","price"}, a buy at
// an optional price.
const ctlSchema = "ctltest_v1"

func init() {
	err := argrules.RegisterEventSchema(argrules.EventSchema{
		Name:   ctlSchema,
		Fields: []string{"market", "count", "price"},
		Read: func(args map[string]json.RawMessage) (argrules.EventOrder, argrules.OrderCostWhy) {
			var o argrules.EventOrder
			o.Market, _ = argrules.ReadArgString(args["market"])
			c, ok := argrules.ReadArgDecimal(args["count"])
			if !ok {
				return o, argrules.WhyEventBadCount
			}
			o.Contracts = c
			if raw, ok := args["price"]; ok {
				p, ok := argrules.ReadArgDecimal(raw)
				if !ok {
					return o, argrules.WhyBadNumber
				}
				o.Price = p
			}
			return o, argrules.OrderCostValued
		},
	})
	if err != nil {
		panic(err)
	}
}

const (
	evSpendDoc    = `{"v":1,"spend_limits":[{"tools":["place_event_order"],"field":"dollar_amount","max":"50","window":"day","tz":"America/New_York","value":"event_max_cost","schema":"ctltest_v1"}]}`
	evApprovalDoc = `{"v":1,"approvals":[{"tools":["place_event_order"],"field":"dollar_amount","above":"20","value":"event_max_cost","schema":"ctltest_v1"}]}`
	evMarketsDoc  = `{"v":1,"event_markets":[{"tools":["place_event_order"],"schema":"ctltest_v1","categories":["FINANCIALS","ECONOMICS"],"series":["SER-B","SER-A"]}]}`
	evStakesDoc   = `{"v":1,"event_stakes":[{"tools":["place_event_order"],"schema":"ctltest_v1","max":"40","reset":"manual"}]}`
)

// Each new document parses, has one canonical form, and round-trips.
func TestEventControlsParseAndCanonical(t *testing.T) {
	all := `{"v":1,"event_stakes":[{"reset":"manual","max":"40","schema":"ctltest_v1","tools":["place_event_order"]}],` +
		`"event_markets":[{"series":["SER-B","SER-A"],"tools":["z_tool","place_event_order"],"schema":"ctltest_v1","categories":["FINANCIALS","ECONOMICS"]}],` +
		`"approvals":[{"schema":"ctltest_v1","value":"event_max_cost","above":"20","field":"dollar_amount","tools":["place_event_order"]}],` +
		`"spend_limits":[{"schema":"ctltest_v1","value":"event_max_cost","window":"day","max":"50","field":"dollar_amount","tools":["place_event_order"]}]}`
	want := `{"v":1,"spend_limits":[{"tools":["place_event_order"],"field":"dollar_amount","max":"50","window":"day","tz":"America/New_York","value":"event_max_cost","schema":"ctltest_v1"}],` +
		`"approvals":[{"tools":["place_event_order"],"field":"dollar_amount","above":"20","value":"event_max_cost","schema":"ctltest_v1"}],` +
		`"event_markets":[{"tools":["place_event_order","z_tool"],"schema":"ctltest_v1","categories":["ECONOMICS","FINANCIALS"],"series":["SER-A","SER-B"]}],` +
		`"event_stakes":[{"tools":["place_event_order"],"schema":"ctltest_v1","max":"40","reset":"manual"}]}`
	got, err := Canonical([]byte(all))
	if err != nil || string(got) != want {
		t.Fatalf("canonical:\n got %s (%v)\nwant %s", got, err, want)
	}
	again, err := Canonical(got)
	if err != nil || string(again) != want {
		t.Fatalf("round trip: %s %v", again, err)
	}
	d, _ := ParseJSON(got)
	word, err := ScopeValue(d)
	if err != nil {
		t.Fatal(err)
	}
	pd, err := ParseScopeValue(word)
	if err != nil {
		t.Fatal(err)
	}
	if pd.EventStakes[0].ID == "" || pd.EventStakes[0].ID == pd.SpendLimits[0].ID || pd.EventStakes[0].MaxUnits() != 40*unitsPerOne {
		t.Fatalf("stake id/units: %+v", pd.EventStakes[0])
	}
	caveats := []string{"scope = place_event_order", "scope = " + word}
	if m, err := CollectEventMarkets(caveats); err != nil || len(m) != 1 {
		t.Fatalf("markets %v %v", m, err)
	}
	if s, err := CollectEventStakes(caveats); err != nil || len(s) != 1 || s[0].ID != pd.EventStakes[0].ID {
		t.Fatalf("stakes %v %v", s, err)
	}
	if l, err := Collect(caveats); err != nil || len(l) != 1 || l[0].Schema != ctlSchema {
		t.Fatalf("limits %v %v", l, err)
	}
	if a, err := CollectApprovals(caveats); err != nil || len(a) != 1 || a[0].Schema != ctlSchema {
		t.Fatalf("approvals %v %v", a, err)
	}
	// A document with only event controls is a document with controls.
	for _, doc := range []string{evMarketsDoc, evStakesDoc} {
		if _, err := ParseJSON([]byte(doc)); err != nil {
			t.Errorf("%s: %v", doc, err)
		}
	}
}

// A control naming a schema that is not registered refuses at parse, and so
// does every malformed event control.
func TestEventControlsRefusedAtParse(t *testing.T) {
	spend := func(extra string) string {
		return `{"v":1,"spend_limits":[{"tools":["place_event_order"],"field":"dollar_amount","max":"50","window":"day"` + extra + `}]}`
	}
	appr := func(extra string) string {
		return `{"v":1,"approvals":[{"tools":["place_event_order"],"field":"dollar_amount","above":"5"` + extra + `}]}`
	}
	mk := func(body string) string { return `{"v":1,"event_markets":[` + body + `]}` }
	st := func(body string) string { return `{"v":1,"event_stakes":[` + body + `]}` }
	for name, doc := range map[string]string{
		"spend: unregistered schema":     spend(`,"value":"event_max_cost","schema":"not_registered_v1"`),
		"spend: no schema":               spend(`,"value":"event_max_cost"`),
		"spend: empty schema":            spend(`,"value":"event_max_cost","schema":""`),
		"spend: schema without value":    spend(`,"schema":"ctltest_v1"`),
		"spend: schema with order value": `{"v":1,"spend_limits":[{"tools":["place_equity_order"],"field":"dollar_amount","max":"50","window":"day","value":"order_max_cost","schema":"ctltest_v1"}]}`,
		"spend: other field":             `{"v":1,"spend_limits":[{"tools":["place_event_order"],"field":"count","max":"50","window":"day","value":"event_max_cost","schema":"ctltest_v1"}]}`,
		"spend: symbols":                 `{"v":1,"spend_limits":[{"tools":["place_crypto_order"],"field":"dollar_amount","max":"50","window":"day","symbols":["BTC"],"value":"event_max_cost","schema":"ctltest_v1"}]}`,
		"spend: unknown value":           spend(`,"value":"event_cost"`),
		"approval: unregistered":         appr(`,"value":"event_max_cost","schema":"not_registered_v1"`),
		"approval: no schema":            appr(`,"value":"event_max_cost"`),
		"approval: schema w/o value":     appr(`,"schema":"ctltest_v1"`),
		"markets: unregistered":          mk(`{"tools":["t"],"schema":"not_registered_v1","series":["A"]}`),
		"markets: no schema":             mk(`{"tools":["t"],"series":["A"]}`),
		"markets: no tools":              mk(`{"schema":"ctltest_v1","series":["A"]}`),
		"markets: neither list":          mk(`{"tools":["t"],"schema":"ctltest_v1"}`),
		"markets: empty lists":           mk(`{"tools":["t"],"schema":"ctltest_v1","series":[],"categories":[]}`),
		"markets: null list":             mk(`{"tools":["t"],"schema":"ctltest_v1","series":null}`),
		"markets: lower-case category":   mk(`{"tools":["t"],"schema":"ctltest_v1","categories":["Economics"]}`),
		"markets: spaced category":       mk(`{"tools":["t"],"schema":"ctltest_v1","categories":[" ECONOMICS"]}`),
		"markets: spaced series":         mk(`{"tools":["t"],"schema":"ctltest_v1","series":["A "]}`),
		"markets: non-ASCII":             mk(`{"tools":["t"],"schema":"ctltest_v1","series":["S\u00c9R"]}`),
		"markets: repeated":              mk(`{"tools":["t"],"schema":"ctltest_v1","series":["A","A"]}`),
		"markets: too long":              mk(`{"tools":["t"],"schema":"ctltest_v1","series":["` + strings.Repeat("A", 65) + `"]}`),
		"markets: too many":              mk(`{"tools":["t"],"schema":"ctltest_v1","series":[` + strings.TrimSuffix(strings.Repeat(`"A",`, 0)+manyIDs(33), ",") + `]}`),
		"markets: unknown key":           mk(`{"tools":["t"],"schema":"ctltest_v1","series":["A"],"block":["B"]}`),
		"markets: empty":                 `{"v":1,"event_markets":[]}`,
		"stakes: unregistered":           st(`{"tools":["t"],"schema":"not_registered_v1","max":"40","reset":"manual"}`),
		"stakes: day reset":              st(`{"tools":["t"],"schema":"ctltest_v1","max":"40","reset":"day"}`),
		"stakes: no reset":               st(`{"tools":["t"],"schema":"ctltest_v1","max":"40"}`),
		"stakes: zero max":               st(`{"tools":["t"],"schema":"ctltest_v1","max":"0","reset":"manual"}`),
		"stakes: 9 decimals":             st(`{"tools":["t"],"schema":"ctltest_v1","max":"1.123456789","reset":"manual"}`),
		"stakes: 11 digits":              st(`{"tools":["t"],"schema":"ctltest_v1","max":"12345678901","reset":"manual"}`),
		"stakes: window key":             st(`{"tools":["t"],"schema":"ctltest_v1","max":"40","reset":"manual","window":"day"}`),
		"stakes: empty":                  `{"v":1,"event_stakes":[]}`,
	} {
		if _, err := ParseJSON([]byte(doc)); err == nil {
			t.Errorf("%s: parsed %s", name, doc)
		}
	}
	// Nine entries are over the limit.
	nine := `{"v":1,"event_stakes":[` + strings.TrimSuffix(strings.Repeat(`{"tools":["t"],"schema":"ctltest_v1","max":"1","reset":"manual"},`, 9), ",") + `]}`
	if _, err := ParseJSON([]byte(nine)); err == nil {
		t.Error("nine stakes parsed")
	}
}

func manyIDs(n int) string {
	var b strings.Builder
	for i := 0; i < n; i++ {
		fmt.Fprintf(&b, `"S%d",`, i)
	}
	return b.String()
}

// Series compare exactly; categories after trimming and ASCII upper-casing.
func TestEventMarketRuleAllows(t *testing.T) {
	d, err := ParseJSON([]byte(evMarketsDoc))
	if err != nil {
		t.Fatal(err)
	}
	r := d.EventMarkets[0]
	for _, tc := range []struct {
		series, category string
		ok               bool
	}{
		{"SER-A", "SPORTS", true},
		{"ser-a", "SPORTS", false},
		{"OTHER", "ECONOMICS", true},
		{"OTHER", " economics\t", true},
		{"OTHER", "Financials", true},
		{"OTHER", "SPORTS", false},
		{"", "", false},
		{"OTHER", "ECONOMICS\u00a0", false},
		{"OTHER", "ECON\x00OMICS", false},
	} {
		if got := r.Allows(tc.series, tc.category); got != tc.ok {
			t.Errorf("%q/%q: %v", tc.series, tc.category, got)
		}
	}
	if !r.Covers("PLACE_EVENT_ORDER") || r.Covers("other") {
		t.Fatal("covers")
	}
}

// The owner and agent lines name no category or series.
func TestDescribeEventControlsNamesNoList(t *testing.T) {
	m, _ := ParseJSON([]byte(evMarketsDoc))
	s, _ := ParseJSON([]byte(evStakesDoc))
	lines := strings.Join(DescribeEventControls(m.EventMarkets, s.EventStakes), "\n")
	for _, v := range []string{"SER-A", "ECONOMICS", "FINANCIALS"} {
		if strings.Contains(lines, v) {
			t.Fatalf("%q in %s", v, lines)
		}
	}
	if !strings.Contains(lines, "at most $40 in total on any one event") {
		t.Fatal(lines)
	}
	l, _ := ParseJSON([]byte(evSpendDoc))
	if got := Describe(l.SpendLimits)[0]; !strings.Contains(got, argrules.EventOrderCountedAs) {
		t.Fatal(got)
	}
	a, _ := ParseJSON([]byte(evApprovalDoc))
	if got := DescribeApprovals(a.Approvals)[0]; !strings.Contains(got, argrules.EventOrderCountedAs) {
		t.Fatal(got)
	}
}

// rawWord builds a control word around a document without parsing it, the
// way an older or a foreign minter would.
func rawWord(doc string) string {
	return scopePrefix + strings.Repeat("ab", 16) + ":" + base64.RawURLEncoding.EncodeToString([]byte(doc))
}

// Every release that predates the event keys refuses a document carrying one,
// and a word made from it: a token with an event control is denied by an
// older verifier, never read without the control.
func TestOldVerifiersRefuseEventDocuments(t *testing.T) {
	docs := map[string]string{
		"spend limit":   evSpendDoc,
		"approval":      evApprovalDoc,
		"event markets": evMarketsDoc,
		"event stakes":  evStakesDoc,
		"budget limit":  `{"v":1,"spend_limits":[{"tools":["place_event_order"],"field":"dollar_amount","max":"50","window":"day","budget":"` + testBudget + `","value":"event_max_cost","schema":"ctltest_v1"}]}`,
		"beside plain":  `{"v":1,"spend_limits":[{"tools":["place_crypto_order"],"field":"dollar_amount","max":"50","window":"day"}],"event_stakes":[{"tools":["t"],"schema":"ctltest_v1","max":"40","reset":"manual"}]}`,
	}
	for name, doc := range docs {
		d, err := ParseJSON([]byte(doc))
		if err != nil {
			t.Fatalf("%s: the current parser refused it: %v", name, err)
		}
		_, e3 := releasedv3.ParseJSON([]byte(doc))
		if e3 == nil || !(strings.Contains(e3.Error(), `unknown spending limit key "schema"`) ||
			strings.Contains(e3.Error(), `unknown approval rule key "schema"`) ||
			strings.Contains(e3.Error(), `unknown document key`)) {
			t.Errorf("%s: the parser at e22202ba: %v", name, e3)
		}
		if _, err := releasedv2.ParseJSON([]byte(doc)); err == nil {
			t.Errorf("%s: the parser at 0e60d5f8 read it", name)
		}
		if _, err := releasedv1.ParseJSON([]byte(doc)); err == nil {
			t.Errorf("%s: the parser at 84998dbd read it", name)
		}
		word, err := ScopeValue(d)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := releasedv3.ParseScopeValue(word); err == nil {
			t.Errorf("%s: the parser at e22202ba read the word", name)
		}
		if _, err := releasedv3.CollectSealed([]string{"scope = place_event_order", "scope = " + word}, 0); err == nil {
			t.Errorf("%s: the old CollectSealed read the word", name)
		}
		if _, err := releasedv3.CollectApprovals([]string{"scope = " + word}); err == nil {
			t.Errorf("%s: the old CollectApprovals read the word", name)
		}
		if _, err := releasedv2.ParseScopeValue(word); err == nil {
			t.Errorf("%s: the parser at 0e60d5f8 read the word", name)
		}
		if _, err := releasedv1.ParseScopeValue(word); err == nil {
			t.Errorf("%s: the parser at 84998dbd read the word", name)
		}
	}
	// A spend limit with value event_max_cost and no schema is refused by the
	// old parsers by its value, and by the current one for the missing schema.
	noSchema := `{"v":1,"spend_limits":[{"tools":["t"],"field":"dollar_amount","max":"5","window":"day","value":"event_max_cost"}]}`
	if _, err := releasedv3.ParseJSON([]byte(noSchema)); err == nil {
		t.Error("e22202ba read value event_max_cost")
	}
	if _, err := ParseJSON([]byte(noSchema)); err == nil {
		t.Error("the current parser read event_max_cost with no schema")
	}
	// A word naming a schema this process does not register is refused by the
	// current parser too (fail closed).
	unreg := strings.ReplaceAll(evStakesDoc, ctlSchema, "not_registered_v1")
	if _, err := ParseScopeValue(rawWord(unreg)); err == nil {
		t.Error("a word with an unregistered schema parsed")
	}
}

// Every existing document keeps its canonical bytes: the current code and the
// snapshot of the release before the event keys write the same bytes.
func TestExistingControlDocumentsAreByteIdentical(t *testing.T) {
	for _, doc := range []string{
		`{"v":1,"approvals":[{"above":"50","field":"dollar_amount","tools":["place_equity_order","place_crypto_order"]}],"spend_limits":[{"window":"day","max":"300","field":"dollar_amount","tools":["place_equity_order","place_crypto_order"]}]}`,
		`{"v":1,"spend_limits":[{"tools":["place_equity_order"],"field":"dollar_amount","max":"200","window":"day","tz":"America/New_York","budget":"` + testBudget + `","value":"order_max_cost"},{"tools":["place_equity_order"],"field":"dollar_amount","max":"50","window":"day","budget":"` + testBudget + `","symbols":["AAPL"],"value":"order_max_cost"}]}`,
		`{"v":1,"spend_limits":[{"tools":["transfer"],"field":"amount","max":"1000.25","window":"week","tz":"Europe/London"}]}`,
		`{"v":1,"approvals":[{"tools":["place_crypto_order"],"field":"dollar_amount","above":"5","value":"order_max_cost"}]}`,
	} {
		now, err := Canonical([]byte(doc))
		if err != nil {
			t.Fatal(err)
		}
		then, err := releasedv3.Canonical([]byte(doc))
		if err != nil || string(now) != string(then) {
			t.Errorf("%s:\n now %s\nthen %s (%v)", doc, now, then, err)
		}
		// The same word gives the same limits and the same counter ids.
		word := rawWord(string(now))
		a, err1 := ParseScopeValue(word)
		b, err2 := releasedv3.ParseScopeValue(word)
		if err1 != nil || err2 != nil || len(a.SpendLimits) != len(b.SpendLimits) || len(a.Approvals) != len(b.Approvals) {
			t.Fatalf("%v %v", err1, err2)
		}
		for i := range a.SpendLimits {
			if a.SpendLimits[i].ID != b.SpendLimits[i].ID || a.SpendLimits[i].MaxUnits() != b.SpendLimits[i].MaxUnits() || a.SpendLimits[i].Value != b.SpendLimits[i].Value {
				t.Errorf("limit %d differs", i)
			}
		}
		for i := range a.Approvals {
			if a.Approvals[i].ID != b.Approvals[i].ID || a.Approvals[i].Above != b.Approvals[i].Above {
				t.Errorf("approval %d differs", i)
			}
		}
	}
}

func TestMemoryStoreEventStake(t *testing.T) {
	ctx := context.Background()
	m := NewMemoryStore(nil)
	r := func(event, call string, units int64) ReserveResult {
		t.Helper()
		out, err := m.ReserveEvent(ctx, EventReservation{Key: "k", Event: event, CallID: call, Units: units, Max: 10})
		if err != nil {
			t.Fatal(err)
		}
		return out
	}
	if out := r("E1", "c1", 6); out.Status != Reserved || out.Total != 6 {
		t.Fatalf("%+v", out)
	}
	if out := r("E1", "c2", 5); out.Status != OverLimit || out.Total != 6 {
		t.Fatalf("%+v", out)
	}
	if out := r("E1", "c3", 4); out.Status != Reserved || out.Total != 10 {
		t.Fatalf("%+v", out)
	}
	if out := r("E2", "c4", 10); out.Status != Reserved || out.Total != 10 {
		t.Fatalf("another event is unaffected: %+v", out)
	}
	if out := r("E1", "c1", 1); out.Status != Duplicate {
		t.Fatalf("%+v", out)
	}
	if err := m.ReleaseEvent(ctx, EventReservation{Key: "k", Event: "E1", CallID: "c3"}); err != nil {
		t.Fatal(err)
	}
	if n, _ := m.EventTotal(ctx, "k", "E1"); n != 6 {
		t.Fatalf("after release %d", n)
	}
	_ = m.ReleaseEvent(ctx, EventReservation{Key: "k", Event: "E1", CallID: "c3"}) // twice is a no-op
	if n, _ := m.EventTotal(ctx, "k", "E1"); n != 6 {
		t.Fatalf("after second release %d", n)
	}
	if err := m.ResetEvents(ctx, "k"); err != nil {
		t.Fatal(err)
	}
	if n, _ := m.EventTotal(ctx, "k", "E1"); n != 0 {
		t.Fatalf("after reset %d", n)
	}
	_ = m.ReleaseEvent(ctx, EventReservation{Key: "k", Event: "E1", CallID: "c1"}) // from before the reset
	if out := r("E1", "c5", 10); out.Status != Reserved || out.Total != 10 {
		t.Fatalf("after reset: %+v", out)
	}
	m.Err = ErrStoreUnavailable
	if _, err := m.ReserveEvent(ctx, EventReservation{Key: "k", Event: "E1", CallID: "c9", Units: 1, Max: 10}); err == nil {
		t.Fatal("a store that is down answered")
	}
}

// A counter holds at most MaxEventsPerStake events; a new one past that is
// refused until a reset, and a known one still counts.
func TestMemoryStoreEventStakeIsBounded(t *testing.T) {
	ctx := context.Background()
	m := NewMemoryStore(nil)
	for i := 0; i < MaxEventsPerStake; i++ {
		out, _ := m.ReserveEvent(ctx, EventReservation{Key: "k", Event: fmt.Sprintf("E%d", i), CallID: fmt.Sprintf("c%d", i), Units: 1, Max: 5})
		if out.Status != Reserved {
			t.Fatalf("%d: %+v", i, out)
		}
	}
	if out, _ := m.ReserveEvent(ctx, EventReservation{Key: "k", Event: "NEW", CallID: "x", Units: 1, Max: 5}); out.Status != EventsFull {
		t.Fatalf("%+v", out)
	}
	if out, _ := m.ReserveEvent(ctx, EventReservation{Key: "k", Event: "E1", CallID: "y", Units: 1, Max: 5}); out.Status != Reserved {
		t.Fatalf("a known event: %+v", out)
	}
	_ = m.ResetEvents(ctx, "k")
	if out, _ := m.ReserveEvent(ctx, EventReservation{Key: "k", Event: "NEW", CallID: "x", Units: 1, Max: 5}); out.Status != Reserved {
		t.Fatalf("after reset: %+v", out)
	}
}

func TestEventStakeConcurrentCallsNeverExceedMax(t *testing.T) {
	ctx := context.Background()
	m := NewMemoryStore(nil)
	const n = 200
	var ok int64
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			out, err := m.ReserveEvent(ctx, EventReservation{Key: "k", Event: "E", CallID: fmt.Sprint(i), Units: 3, Max: 100})
			if err == nil && out.Status == Reserved {
				atomic.AddInt64(&ok, 1)
			}
		}(i)
	}
	wg.Wait()
	total, _ := m.EventTotal(ctx, "k", "E")
	if ok != 33 || total != 99 {
		t.Fatalf("admitted %d, total %d", ok, total)
	}
}

// A reset running while calls reserve and release never leaves a call counted
// in neither or both generations: the total always equals what the live
// calls hold.
func TestEventStakeResetIsAtomic(t *testing.T) {
	ctx := context.Background()
	m := NewMemoryStore(nil)
	var wg sync.WaitGroup
	for w := 0; w < 8; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < 300; i++ {
				id := fmt.Sprintf("%d-%d", w, i)
				res := EventReservation{Key: "k", Event: fmt.Sprintf("E%d", i%3), CallID: id, Units: int64(1 + i%5), Max: 1 << 40}
				_, _ = m.ReserveEvent(ctx, res)
				if i%4 == 0 {
					_ = m.ReleaseEvent(ctx, res)
				}
				if w == 0 && i%25 == 0 {
					_ = m.ResetEvents(ctx, "k")
				}
			}
		}(w)
	}
	wg.Wait()
	m.mu.Lock()
	defer m.mu.Unlock()
	st := m.stakes["k"]
	if st == nil {
		return
	}
	held := map[string]int64{}
	for _, c := range st.calls {
		held[c.event] += c.units
	}
	for e, total := range st.events {
		if total != held[e] || total < 0 {
			t.Fatalf("event %s: total %d, calls hold %d", e, total, held[e])
		}
	}
}

// The canonical category is the comparison form.
func TestCanonicalEventCategory(t *testing.T) {
	for in, want := range map[string]string{" economics ": "ECONOMICS", "Sci-Tech": "SCI-TECH", "caf\u00e9": "", "": ""} {
		if got := CanonicalEventCategory(in); got != want {
			t.Errorf("%q: %q", in, got)
		}
	}
}
