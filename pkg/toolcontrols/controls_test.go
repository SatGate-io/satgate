package toolcontrols

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/satgate-io/satgate/pkg/argrules"
)

const goodDoc = `{"v":1,"spend_limits":[{"tools":["place_crypto_order"],"field":"dollar_amount","max":"200","window":"day"}]}`

func allowAll(string, string) bool { return true }

func TestParseDefaultsAndCanonical(t *testing.T) {
	d, err := ParseJSON([]byte(goodDoc))
	if err != nil {
		t.Fatal(err)
	}
	l := d.SpendLimits[0]
	if l.TZ != DefaultTimeZone || l.Window != WindowDay || l.MaxUnits() != 200*unitsPerOne {
		t.Fatalf("limit = %+v", l)
	}
	canon, err := Canonical([]byte(goodDoc))
	if err != nil {
		t.Fatal(err)
	}
	want := `{"v":1,"spend_limits":[{"tools":["place_crypto_order"],"field":"dollar_amount","max":"200","window":"day","tz":"America/New_York"}]}`
	if string(canon) != want {
		t.Fatalf("canonical = %s", canon)
	}
	again, err := Canonical(canon)
	if err != nil || string(again) != want {
		t.Fatalf("canonical is not stable: %s %v", again, err)
	}
}

func TestParseRejects(t *testing.T) {
	bad := map[string]string{
		"empty":             ``,
		"not object":        `[]`,
		"no version":        `{"spend_limits":[]}`,
		"wrong version":     `{"v":2,"spend_limits":[{"tools":["a"],"field":"f","max":"1","window":"day"}]}`,
		"unknown doc key":   `{"v":1,"spend_limits":[{"tools":["a"],"field":"f","max":"1","window":"day"}],"extra":1}`,
		"unknown limit key": `{"v":1,"spend_limits":[{"tools":["a"],"field":"f","max":"1","window":"day","rolling":true}]}`,
		"duplicate key":     `{"v":1,"v":1,"spend_limits":[{"tools":["a"],"field":"f","max":"1","window":"day"}]}`,
		"no limits":         `{"v":1}`,
		"empty limits":      `{"v":1,"spend_limits":[]}`,
		"missing tools":     `{"v":1,"spend_limits":[{"field":"f","max":"1","window":"day"}]}`,
		"missing field":     `{"v":1,"spend_limits":[{"tools":["a"],"max":"1","window":"day"}]}`,
		"missing max":       `{"v":1,"spend_limits":[{"tools":["a"],"field":"f","window":"day"}]}`,
		"missing window":    `{"v":1,"spend_limits":[{"tools":["a"],"field":"f","max":"1"}]}`,
		"bad window":        `{"v":1,"spend_limits":[{"tools":["a"],"field":"f","max":"1","window":"month"}]}`,
		"numeric max":       `{"v":1,"spend_limits":[{"tools":["a"],"field":"f","max":1,"window":"day"}]}`,
		"exponent max":      `{"v":1,"spend_limits":[{"tools":["a"],"field":"f","max":"1e3","window":"day"}]}`,
		"zero max":          `{"v":1,"spend_limits":[{"tools":["a"],"field":"f","max":"0","window":"day"}]}`,
		"negative max":      `{"v":1,"spend_limits":[{"tools":["a"],"field":"f","max":"-5","window":"day"}]}`,
		"leading zero":      `{"v":1,"spend_limits":[{"tools":["a"],"field":"f","max":"05","window":"day"}]}`,
		"too many decimals": `{"v":1,"spend_limits":[{"tools":["a"],"field":"f","max":"1.123456789","window":"day"}]}`,
		"huge max":          `{"v":1,"spend_limits":[{"tools":["a"],"field":"f","max":"12345678901","window":"day"}]}`,
		"bad tz":            `{"v":1,"spend_limits":[{"tools":["a"],"field":"f","max":"1","window":"day","tz":"Mars/Base"}]}`,
		"path tz":           `{"v":1,"spend_limits":[{"tools":["a"],"field":"f","max":"1","window":"day","tz":"../etc/localtime"}]}`,
		"local tz":          `{"v":1,"spend_limits":[{"tools":["a"],"field":"f","max":"1","window":"day","tz":"Local"}]}`,
		"empty tools":       `{"v":1,"spend_limits":[{"tools":[],"field":"f","max":"1","window":"day"}]}`,
		"wildcard tool":     `{"v":1,"spend_limits":[{"tools":["*"],"field":"f","max":"1","window":"day"}]}`,
		"dup tool":          `{"v":1,"spend_limits":[{"tools":["a","a"],"field":"f","max":"1","window":"day"}]}`,
		"bad field":         `{"v":1,"spend_limits":[{"tools":["a"],"field":"a b","max":"1","window":"day"}]}`,
	}
	for name, doc := range bad {
		if _, err := ParseJSON([]byte(doc)); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestSeveralLimitsAndSeveralTools(t *testing.T) {
	doc := `{"v":1,"spend_limits":[
	 {"tools":["place_crypto_order","place_equity_order"],"field":"dollar_amount","max":"200.50","window":"day","tz":"America/Los_Angeles"},
	 {"tools":["place_crypto_order"],"field":"dollar_amount","max":"1000","window":"week"}]}`
	d, err := ParseJSON([]byte(doc))
	if err != nil {
		t.Fatal(err)
	}
	if len(d.SpendLimits) != 2 || !d.SpendLimits[0].Covers("place_equity_order") || d.SpendLimits[1].Covers("place_equity_order") {
		t.Fatalf("limits = %+v", d.SpendLimits)
	}
	if d.SpendLimits[0].MaxUnits() != 20050000000 {
		t.Fatalf("units = %d", d.SpendLimits[0].MaxUnits())
	}
}

func mustCaveat(t *testing.T, doc string) string {
	t.Helper()
	d, err := ParseJSON([]byte(doc))
	if err != nil {
		t.Fatal(err)
	}
	c, err := Caveat(d)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestCaveatRoundTripAndWordShape(t *testing.T) {
	c1, c2 := mustCaveat(t, goodDoc), mustCaveat(t, goodDoc)
	if c1 == c2 {
		t.Fatal("two mints share a nonce")
	}
	v := strings.TrimPrefix(c1, "scope = ")
	if !strings.HasPrefix(v, "argrules:ctl:v1:") || !argrules.IsRuleScope(v) || !argrules.IsControlScope(v) {
		t.Fatalf("word = %s", v)
	}
	d, err := ParseScopeValue(v)
	if err != nil || d.SpendLimits[0].ID == "" {
		t.Fatalf("parse: %v %+v", err, d)
	}
	other, _ := ParseScopeValue(strings.TrimPrefix(c2, "scope = "))
	if other.SpendLimits[0].ID == d.SpendLimits[0].ID {
		t.Fatal("two mints share a counter id")
	}
	for _, mangled := range []string{
		strings.Replace(v, "ctl:v1:", "ctl:v2:", 1),
		v[:len(v)-3] + "!!!",
		strings.Replace(v, "v1:", "v1:zz", 1),
	} {
		if _, err := ParseScopeValue(mangled); err == nil {
			t.Errorf("accepted %.40s", mangled)
		}
	}
}

// The argument-rules reader must neither choke on a control word nor count it.
func TestArgrulesIgnoresControlWords(t *testing.T) {
	c := mustCaveat(t, goodDoc)
	rules, err := argrules.Collect([]string{"scope = t:mcp:*", c}, allowAll)
	if err != nil || len(rules) != 0 {
		t.Fatalf("rules = %v err = %v", rules, err)
	}
	sum, err := argrules.DocumentsSHA256([]string{c})
	if err != nil || sum != "" {
		t.Fatalf("sum = %q err = %v", sum, err)
	}
}

func TestCollect(t *testing.T) {
	c := mustCaveat(t, goodDoc)
	limits, err := Collect([]string{"tenant_id = t", "scope = t:mcp:*", c})
	if err != nil || len(limits) != 1 || limits[0].ID == "" {
		t.Fatalf("limits = %v err = %v", limits, err)
	}
	// A limit on a tool the scope does not allow is inert, not an error.
	if l, err := Collect([]string{"scope = t:other", c}); err != nil || len(l) != 1 {
		t.Fatalf("narrowed child lost its limit: %v %v", l, err)
	}
	// A control word that does not parse is an error, never skipped.
	if _, err := Collect([]string{"scope = argrules:ctl:v1:nope"}); err == nil {
		t.Fatal("garbled control word was skipped")
	}
	var many []string
	for i := 0; i <= MaxControlCaveats; i++ {
		many = append(many, mustCaveat(t, goodDoc))
	}
	if _, err := Collect(many); err == nil {
		t.Fatal("too many control caveats accepted")
	}
	if !HasControls([]string{"scope = argrules:ctl:v1:nope"}) || HasControls([]string{"scope = a"}) {
		t.Fatal("HasControls")
	}
}

func TestWindowBounds(t *testing.T) {
	ny, _ := time.LoadLocation("America/New_York")
	day := SpendLimit{Window: WindowDay, loc: ny}
	// 03:30 UTC on 10 Oct is still 9 Oct 23:30 in New York.
	id, start, end := day.WindowBounds(time.Date(2026, 10, 10, 3, 30, 0, 0, time.UTC))
	if id != "2026-10-09" || start.Day() != 9 || end.Sub(start) != 24*time.Hour {
		t.Fatalf("day = %s %v %v", id, start, end)
	}
	id2, _, _ := day.WindowBounds(time.Date(2026, 10, 10, 4, 0, 0, 0, time.UTC))
	if id2 != "2026-10-10" {
		t.Fatalf("rollover at local midnight = %s", id2)
	}
	// A day that has 25 hours (DST ends 1 Nov 2026) still ends at local midnight.
	_, s, e := day.WindowBounds(time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC))
	if e.Sub(s) != 25*time.Hour {
		t.Fatalf("DST day = %v", e.Sub(s))
	}
	week := SpendLimit{Window: WindowWeek, loc: ny}
	// Fri 9 Oct 2026 is in ISO week 41, Monday 5 Oct to Sunday 11 Oct.
	id, start, end = week.WindowBounds(time.Date(2026, 10, 9, 15, 0, 0, 0, time.UTC))
	if id != "2026-W41" || start.Weekday() != time.Monday || start.Day() != 5 || end.Day() != 12 {
		t.Fatalf("week = %s %v %v", id, start, end)
	}
	// Sunday night New York is still the same week; Monday 00:00 is the next.
	idSun, _, _ := week.WindowBounds(time.Date(2026, 10, 12, 3, 59, 0, 0, time.UTC))
	idMon, _, _ := week.WindowBounds(time.Date(2026, 10, 12, 4, 0, 0, 0, time.UTC))
	if idSun != "2026-W41" || idMon != "2026-W42" {
		t.Fatalf("week boundary = %s %s", idSun, idMon)
	}
	// ISO year edge: 31 Dec 2026 is a Thursday in ISO week 53 of 2026.
	idEdge, _, _ := week.WindowBounds(time.Date(2026, 12, 31, 12, 0, 0, 0, time.UTC))
	idNext, _, _ := week.WindowBounds(time.Date(2027, 1, 4, 12, 0, 0, 0, time.UTC))
	if idEdge != "2026-W53" || idNext != "2027-W01" {
		t.Fatalf("iso year edge = %s %s", idEdge, idNext)
	}
}

func TestDescribe(t *testing.T) {
	d, _ := ParseJSON([]byte(goodDoc))
	got := Describe(d.SpendLimits)
	if len(got) != 1 || got[0] != "place_crypto_order: at most $200 a day in total on dollar_amount" {
		t.Fatalf("describe = %v", got)
	}
	w, _ := ParseJSON([]byte(`{"v":1,"spend_limits":[{"tools":["t"],"field":"quantity","max":"5","window":"week","tz":"Europe/London"}]}`))
	if got := Describe(w.SpendLimits)[0]; got != "t: at most 5 a week in total on quantity (days start at midnight Europe/London time)" {
		t.Fatalf("describe = %q", got)
	}
}

func res(key, call string, units, max int64) Reservation {
	return Reservation{Key: key, CallID: call, Units: units, Max: max, ExpiresAt: time.Now().Add(time.Hour)}
}

func TestMemoryStoreReserveReleaseDuplicate(t *testing.T) {
	s := NewMemoryStore(nil)
	ctx := context.Background()
	r, _ := s.Reserve(ctx, res("k", "a", 150, 200))
	if r.Status != Reserved || r.Total != 150 {
		t.Fatalf("first = %+v", r)
	}
	r, _ = s.Reserve(ctx, res("k", "b", 51, 200))
	if r.Status != OverLimit || r.Total != 150 {
		t.Fatalf("over = %+v", r)
	}
	r, _ = s.Reserve(ctx, res("k", "a", 150, 200))
	if r.Status != Duplicate || s.Total("k") != 150 {
		t.Fatalf("dup = %+v total %d", r, s.Total("k"))
	}
	_ = s.Release(ctx, res("k", "a", 150, 200))
	_ = s.Release(ctx, res("k", "a", 150, 200)) // twice is harmless
	if s.Total("k") != 0 {
		t.Fatalf("total after release = %d", s.Total("k"))
	}
	r, _ = s.Reserve(ctx, res("k", "b", 200, 200))
	if r.Status != Reserved {
		t.Fatalf("exactly the limit must fit: %+v", r)
	}
}

func TestMemoryStoreConcurrencyNeverExceedsMax(t *testing.T) {
	s := NewMemoryStore(nil)
	ctx := context.Background()
	const n, each, max = 200, 7, 500
	var ok int64
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			r, err := s.Reserve(ctx, res("k", string(rune('A'+i%50))+string(rune('a'+i/50)), each, max))
			if err == nil && r.Status == Reserved {
				atomic.AddInt64(&ok, each)
			}
		}(i)
	}
	wg.Wait()
	if ok > max || s.Total("k") != ok || ok != (max/each)*each {
		t.Fatalf("reserved %d, counter %d, max %d", ok, s.Total("k"), max)
	}
}

func TestMemoryStoreExpiry(t *testing.T) {
	now := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	s := NewMemoryStore(func() time.Time { return now })
	ctx := context.Background()
	r := Reservation{Key: "k", CallID: "a", Units: 10, Max: 10, ExpiresAt: now.Add(time.Hour)}
	if out, _ := s.Reserve(ctx, r); out.Status != Reserved {
		t.Fatal(out)
	}
	r.CallID = "b"
	if out, _ := s.Reserve(ctx, r); out.Status != OverLimit {
		t.Fatal(out)
	}
	now = now.Add(2 * time.Hour)
	if out, _ := s.Reserve(ctx, r); out.Status != Reserved {
		t.Fatalf("counter should have expired: %+v", out)
	}
}

func TestMemoryStoreDown(t *testing.T) {
	s := NewMemoryStore(nil)
	s.Err = ErrStoreUnavailable
	if _, err := s.Reserve(context.Background(), res("k", "a", 1, 2)); err == nil {
		t.Fatal("down store answered")
	}
}
