package toolcontrols

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/satgate-io/satgate/pkg/toolcontrols/internal/releasedv1"
)

const testBudget = "abcdefghij0123456789abcd"

func limitDoc(extra string) string {
	return `{"v":1,"spend_limits":[{"tools":["place_crypto_order"],"field":"dollar_amount","max":"200","window":"day"` + extra + `}]}`
}

func TestBudgetKeyParsesAndRoundTrips(t *testing.T) {
	d, err := ParseJSON([]byte(limitDoc(`,"budget":"` + testBudget + `"`)))
	if err != nil {
		t.Fatal(err)
	}
	if d.SpendLimits[0].Budget != testBudget || d.SpendLimits[0].Sealed {
		t.Fatalf("limit = %+v", d.SpendLimits[0])
	}
	out, err := Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"v":1,"spend_limits":[{"tools":["place_crypto_order"],"field":"dollar_amount","max":"200","window":"day","tz":"America/New_York","budget":"` + testBudget + `"}]}`
	if string(out) != want {
		t.Fatalf("canonical = %s", out)
	}
	// Canonical form is a fixed point.
	again, err := Canonical(out)
	if err != nil || string(again) != want {
		t.Fatalf("not a fixed point: %s %v", again, err)
	}
}

func TestBudgetIDShapeIsEnforced(t *testing.T) {
	for _, id := range []string{
		"", "short", testBudget + "x", strings.ToUpper(testBudget), "abcdefghij0123456789abc-",
		"abcdefghij0123456789abc ", "abcdefghij0123456789abc\u00e9",
	} {
		if _, err := ParseJSON([]byte(limitDoc(`,"budget":"` + id + `"`))); err == nil {
			t.Errorf("budget %q accepted", id)
		}
	}
	if _, err := ParseJSON([]byte(limitDoc(`,"budget":7`))); err == nil {
		t.Error("a number was accepted as a budget id")
	}
	if _, err := ParseJSON([]byte(limitDoc(`,"budget":"` + testBudget + `","budget":"` + testBudget + `"`))); err == nil {
		t.Error("a repeated budget key was accepted")
	}
}

func TestDocumentsWithoutNewKeysAreByteIdentical(t *testing.T) {
	d, err := ParseJSON([]byte(limitDoc(``)))
	if err != nil {
		t.Fatal(err)
	}
	out, _ := Marshal(d)
	want := `{"v":1,"spend_limits":[{"tools":["place_crypto_order"],"field":"dollar_amount","max":"200","window":"day","tz":"America/New_York"}]}`
	if string(out) != want {
		t.Fatalf("canonical = %s", out)
	}
}

func TestSymbolListRules(t *testing.T) {
	ok := []string{`["BTC"]`, `["BTC","ETH"]`, `["ETH","BTC"]`, `["DOGE1"]`}
	for _, s := range ok {
		if _, err := ParseJSON([]byte(limitDoc(`,"symbols":` + s))); err != nil {
			t.Errorf("symbols %s refused: %v", s, err)
		}
	}
	var many []string
	for i := 0; i < 17; i++ {
		many = append(many, `"A`+string(rune('A'+i))+`"`)
	}
	bad := map[string]string{
		"empty":            `[]`,
		"null":             `null`,
		"not a list":       `"BTC"`,
		"not text":         `[1]`,
		"lower case":       `["btc"]`,
		"padded":           `[" BTC"]`,
		"hyphen form":      `["BTC-USD"]`,
		"too long":         `["ABCDEFGHIJKLM"]`,
		"punctuation":      `["B/C"]`,
		"unicode":          `["BT\u0421"]`,
		"empty entry":      `[""]`,
		"duplicate":        `["BTC","BTC"]`,
		"17 entries":       `[` + strings.Join(many, ",") + `]`,
		"stem plus USD":    `["BTC-USD"]`,
		"a nested list":    `[["BTC"]]`,
		"object":           `{"BTC":1}`,
		"boolean":          `true`,
		"number":           `5`,
		"trailing newline": `["BTC\n"]`,
	}
	for name, s := range bad {
		if _, err := ParseJSON([]byte(limitDoc(`,"symbols":` + s))); err == nil {
			t.Errorf("%s: symbols %s accepted", name, s)
		}
	}
	// 16 is the most.
	if _, err := ParseJSON([]byte(limitDoc(`,"symbols":[` + strings.Join(many[:16], ",") + `]`))); err != nil {
		t.Errorf("16 symbols refused: %v", err)
	}
}

func TestSymbolListIsSortedInTheCanonicalDocument(t *testing.T) {
	out, err := Canonical([]byte(limitDoc(`,"symbols":["ETH","BTC"]`)))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), `"symbols":["BTC","ETH"]`) {
		t.Fatalf("canonical = %s", out)
	}
}

func TestSymbolCapOnAnotherToolIsRefusedAtParseTime(t *testing.T) {
	for _, tool := range []string{"place_options_order", "get_quote", "transfer_funds", "other_service_buy"} {
		doc := `{"v":1,"spend_limits":[{"tools":["` + tool + `"],"field":"dollar_amount","max":"5","window":"day","symbols":["BTC"]}]}`
		if _, err := ParseJSON([]byte(doc)); err == nil || !strings.Contains(err.Error(), "symbol cap cannot name tool") {
			t.Errorf("%s: err = %v", tool, err)
		}
	}
	// One unlisted tool among listed ones is enough to refuse.
	doc := `{"v":1,"spend_limits":[{"tools":["place_crypto_order","get_quote"],"field":"dollar_amount","max":"5","window":"day","symbols":["BTC"]}]}`
	if _, err := ParseJSON([]byte(doc)); err == nil {
		t.Error("a symbol cap naming an unlisted tool next to a listed one was accepted")
	}
	// Crypto and equity are different namespaces: one cap does not cover both.
	doc = `{"v":1,"spend_limits":[{"tools":["place_crypto_order","place_equity_order"],"field":"dollar_amount","max":"5","window":"day","symbols":["BTC"]}]}`
	if _, err := ParseJSON([]byte(doc)); err == nil {
		t.Error("a symbol cap over crypto and equity together was accepted")
	}
	// A limit with no symbols still covers any tool.
	doc = `{"v":1,"spend_limits":[{"tools":["get_quote"],"field":"dollar_amount","max":"5","window":"day"}]}`
	if _, err := ParseJSON([]byte(doc)); err != nil {
		t.Errorf("plain limit on another tool refused: %v", err)
	}
}

func TestEquitySymbolCap(t *testing.T) {
	doc := `{"v":1,"spend_limits":[{"tools":["place_equity_order"],"field":"dollar_amount","max":"5","window":"day","symbols":["AAPL","BRK.B"]}]}`
	if _, err := ParseJSON([]byte(doc)); err != nil {
		t.Fatal(err)
	}
	doc = strings.Replace(doc, `"AAPL"`, `"1AAPL"`, 1)
	if _, err := ParseJSON([]byte(doc)); err == nil {
		t.Error("an equity symbol starting with a digit was accepted")
	}
}

func TestCanonicalSymbolTable(t *testing.T) {
	type c struct {
		tool, in, want string
		ok             bool
	}
	for _, tc := range []c{
		{"place_crypto_order", "BTC", "BTC", true},
		{"place_crypto_order", "btc", "BTC", true},
		{"place_crypto_order", " BTC ", "BTC", true},
		{"place_crypto_order", "\tBtc\n", "BTC", true},
		{"place_crypto_order", "BTC-USD", "BTC", true},
		{"place_crypto_order", "btc-usd", "BTC", true},
		{"place_crypto_order", " Btc-Usd ", "BTC", true},
		{"Place_Crypto_Order", "eth-usd", "ETH", true},
		{"place_crypto_order", "1INCH", "1INCH", true},
		{"place_crypto_order", "BTC-USD-USD", "", false}, // only one -USD is stripped
		{"place_crypto_order", "-USD", "", false},
		{"place_crypto_order", "", "", false},
		{"place_crypto_order", "BTC-EUR", "", false},
		{"place_crypto_order", "BTC/USD", "", false},
		{"place_crypto_order", "BTC USD", "", false},
		{"place_crypto_order", "ABCDEFGHIJKLM", "", false},
		{"place_crypto_order", "BT\u0421", "", false},  // Cyrillic S
		{"place_crypto_order", "\u00a0BTC", "", false}, // no-break space is not trimmed
		{"place_crypto_order", "bt\u0131c", "", false}, // dotless i
		{"place_crypto_order", "\u212aTC", "", false},  // Kelvin sign
		{"place_equity_order", "aapl", "AAPL", true},
		{"place_equity_order", " brk.b ", "BRK.B", true},
		{"place_equity_order", "AAPL-USD", "", false},
		{"place_equity_order", "1AAPL", "", false},
		{"place_equity_order", ".A", "", false},
		{"place_equity_order", "ABCDEFGHIJK", "", false},
		{"get_quote", "BTC", "", false},
	} {
		got, ok := CanonicalSymbol(tc.tool, tc.in)
		if ok != tc.ok || (ok && got != tc.want) {
			t.Errorf("%s %q = %q %v, want %q %v", tc.tool, tc.in, got, ok, tc.want, tc.ok)
		}
	}
}

func TestCheckSymbolVerdicts(t *testing.T) {
	d, err := ParseJSON([]byte(limitDoc(`,"symbols":["BTC"]`)))
	if err != nil {
		t.Fatal(err)
	}
	l := d.SpendLimits[0]
	for _, in := range []string{"BTC", "btc", " BTC ", "BTC-USD", "btc-usd"} {
		if v, s := l.CheckSymbol("place_crypto_order", in, true); v != SymbolCounts || s != "BTC" {
			t.Errorf("%q = %v %q", in, v, s)
		}
	}
	if v, _ := l.CheckSymbol("place_crypto_order", "ETH", true); v != SymbolOther {
		t.Errorf("ETH = %v", v)
	}
	if v, s := l.CheckSymbol("place_crypto_order", "BTCUSD", true); v != SymbolAmbiguous || s != "BTC" {
		t.Errorf("BTCUSD = %v %q", v, s)
	}
	if v, _ := l.CheckSymbol("place_crypto_order", "btcusd", true); v != SymbolAmbiguous {
		t.Errorf("btcusd = %v", v)
	}
	// Not ambiguous when the stem is not capped.
	if v, _ := l.CheckSymbol("place_crypto_order", "ETHUSD", true); v != SymbolOther {
		t.Errorf("ETHUSD = %v", v)
	}
	for _, in := range []string{"", "BTC/USD", "B T C"} {
		if v, _ := l.CheckSymbol("place_crypto_order", in, true); v != SymbolUnreadable {
			t.Errorf("%q = %v", in, v)
		}
	}
	if v, _ := l.CheckSymbol("place_crypto_order", "", false); v != SymbolUnreadable {
		t.Errorf("absent = %v", v)
	}
}

func TestOldParserRefusesTheNewKeys(t *testing.T) {
	newer := map[string]string{
		"budget":  limitDoc(`,"budget":"` + testBudget + `"`),
		"symbols": limitDoc(`,"symbols":["BTC"]`),
		"both":    limitDoc(`,"budget":"` + testBudget + `","symbols":["BTC"]`),
	}
	for name, doc := range newer {
		if _, err := ParseJSON([]byte(doc)); err != nil {
			t.Fatalf("%s: the current parser refused it: %v", name, err)
		}
		_, err := releasedv1.ParseJSON([]byte(doc))
		if err == nil || !strings.Contains(err.Error(), "unknown spending limit key") {
			t.Errorf("%s: the released parser (84998dbd) did not refuse it: %v", name, err)
		}
	}
	// The same parser still takes a document without them, and the current
	// parser reads it the same way.
	plain := limitDoc(``)
	if _, err := releasedv1.ParseJSON([]byte(plain)); err != nil {
		t.Fatalf("released parser refused a plain document: %v", err)
	}
	a, _ := releasedv1.Canonical([]byte(plain))
	b, _ := Canonical([]byte(plain))
	if string(a) != string(b) {
		t.Fatalf("canonical forms differ:\n%s\n%s", a, b)
	}
	// And a whole control word built from a newer document is refused as a word.
	d, _ := ParseJSON([]byte(newer["both"]))
	word, err := ScopeValue(d)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := releasedv1.ParseScopeValue(word); err == nil {
		t.Error("the released parser read a control word with the new keys")
	}
}

func TestCollectSealedMarksOnlyTheSealedPrefix(t *testing.T) {
	d, _ := ParseJSON([]byte(limitDoc(`,"budget":"` + testBudget + `"`)))
	w1, _ := Caveat(d)
	w2, _ := Caveat(d)
	caveats := []string{"scope = tool_a", w1, "budget_bind = x", w2}
	for _, tc := range []struct {
		prefix int
		want   []bool
	}{{0, []bool{false, false}}, {2, []bool{true, false}}, {3, []bool{true, false}}, {4, []bool{true, true}}} {
		got, err := CollectSealed(caveats, tc.prefix)
		if err != nil || len(got) != 2 {
			t.Fatal(err, len(got))
		}
		for i := range got {
			if got[i].Sealed != tc.want[i] {
				t.Errorf("prefix %d limit %d sealed = %v", tc.prefix, i, got[i].Sealed)
			}
		}
	}
	plain, _ := Collect(caveats)
	for _, l := range plain {
		if l.Sealed {
			t.Error("Collect marks a limit sealed")
		}
	}
}

func TestDescribeNamesNeitherBudgetNorOwner(t *testing.T) {
	d, _ := ParseJSON([]byte(limitDoc(`,"budget":"` + testBudget + `","symbols":["BTC"]`)))
	d2, _ := ParseJSON([]byte(limitDoc(`,"budget":"` + testBudget + `"`)))
	var lines []string
	lines = append(lines, Describe(d.SpendLimits)...)
	lines = append(lines, Describe(d2.SpendLimits)...)
	for _, ln := range lines {
		if strings.Contains(ln, testBudget) {
			t.Errorf("line names the budget: %s", ln)
		}
	}
	if !strings.Contains(lines[0], "for BTC only") || !strings.Contains(lines[1], "shared with your other agents") {
		t.Errorf("lines = %q", lines)
	}
}

func TestMemoryStoreTotalIsReadOnly(t *testing.T) {
	s := NewMemoryStore(nil)
	ctx := context.Background()
	if n, err := s.Total(ctx, "k"); err != nil || n != 0 {
		t.Fatalf("empty total = %d %v", n, err)
	}
	r := Reservation{Key: "k", CallID: "a", Units: 7, Max: 10, ExpiresAt: time.Now().Add(time.Hour)}
	if _, err := s.Reserve(ctx, r); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if n, _ := s.Total(ctx, "k"); n != 7 {
			t.Fatalf("total = %d", n)
		}
	}
	if len(s.Keys()) != 1 {
		t.Fatalf("keys = %v", s.Keys())
	}
	s.Err = ErrStoreUnavailable
	if _, err := s.Total(ctx, "k"); err == nil {
		t.Fatal("Total hid a store error")
	}
}
