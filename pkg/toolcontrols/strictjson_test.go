package toolcontrols

import (
	"encoding/json"
	"math/rand"
	"reflect"
	"strings"
	"testing"
)

func call(args string) json.RawMessage {
	return json.RawMessage(`{"name":"place","arguments":` + args + `}`)
}

// Astra's two cases: a lone surrogate must never share a hash with U+FFFD or
// with another lone surrogate. They are refused outright.
func TestCallHashRefusesUnpairedSurrogates(t *testing.T) {
	for name, args := range map[string]string{
		"lone high":              `{"symbol":"\ud800"}`,
		"lone high 2":            `{"symbol":"\ud801"}`,
		"lone high, last":        `{"symbol":"\udbff"}`,
		"lone low":               `{"symbol":"\udc00"}`,
		"lone low, last":         `{"symbol":"\udfff"}`,
		"reversed pair":          `{"symbol":"\udc00\ud800"}`,
		"high then non-low":      `{"symbol":"\ud800\u0041"}`,
		"high then high":         `{"symbol":"\ud800\ud800"}`,
		"high at end of string":  `{"symbol":"x\ud83d"}`,
		"high then literal char": `{"symbol":"\ud83dx"}`,
		"in a key":               `{"sym\ud800bol":"BTC"}`,
		"in a nested key":        `{"a":{"b\udc00":1}}`,
		"in a nested string":     `{"a":[{"b":["ok","\ud800"]}]}`,
		"upper-case hex":         `{"symbol":"\uD800"}`,
		"after a valid pair":     `{"symbol":"\ud83d\ude00\ud83d"}`,
	} {
		if h, err := CallHash("tok", "place", call(args)); err == nil {
			t.Errorf("%s: hashed to %s", name, h)
		}
		if CheckCallLossless(call(args)) == nil {
			t.Errorf("%s: CheckCallLossless accepted it", name)
		}
	}
	// The collisions encoding/json produced: U+FFFD against a lone half.
	if _, err := CallHash("tok", "place", call(`{"symbol":"\ud800"}`)); err == nil {
		t.Fatal("\\ud800 accepted")
	}
	if _, err := CallHash("tok", "place", call(`{"symbol":"\ufffd"}`)); err != nil {
		t.Fatalf("a real U+FFFD must stay allowed: %v", err)
	}
	// Valid pairs stay allowed, and spell the same call however they are written.
	pair, err := CallHash("tok", "place", call(`{"symbol":"\ud83d\ude00"}`))
	if err != nil {
		t.Fatalf("valid pair refused: %v", err)
	}
	upper, _ := CallHash("tok", "place", call(`{"symbol":"\uD83D\uDE00"}`))
	lit, _ := CallHash("tok", "place", call(`{"symbol":"😀"}`))
	if pair != upper || pair != lit {
		t.Fatal("a valid pair, its upper-case spelling and the literal character should be one call")
	}
}

func TestCallHashRefusesWhatCannotBeReadWithoutLoss(t *testing.T) {
	for name, p := range map[string]string{
		"invalid utf8 in a value":      "{\"name\":\"place\",\"arguments\":{\"a\":\"\xff\"}}",
		"invalid utf8 in a key":        "{\"name\":\"place\",\"arguments\":{\"\xc3\":1}}",
		"invalid utf8 outside strings": "{\"name\":\"place\",\"arguments\":{},\"x\":1}\xff",
		"truncated utf8":               "{\"name\":\"place\",\"arguments\":{\"a\":\"\xe2\x82\"}}",
		"overlong utf8":                "{\"name\":\"place\",\"arguments\":{\"a\":\"\xc0\xaf\"}}",
		"encoded surrogate in utf8":    "{\"name\":\"place\",\"arguments\":{\"a\":\"\xed\xa0\x80\"}}",
		"dup top key":                  `{"name":"place","arguments":{},"arguments":{}}`,
		"dup arg key":                  `{"name":"place","arguments":{"a":1,"a":2}}`,
		"dup nested key":               `{"name":"place","arguments":{"o":{"k":1,"k":1}}}`,
		"dup deep key":                 `{"name":"place","arguments":{"o":[{"p":{"q":{"k":1,"k":2}}}]}}`,
		"dup key by escape":            `{"name":"place","arguments":{"a":1,"\u0061":2}}`,
		"dup key in _meta":             `{"name":"place","arguments":{},"_meta":{"x":1,"x":2}}`,
		"case-fold reserved key":       `{"name":"place","Arguments":{"a":1},"arguments":{"a":1}}`,
		"case-fold name":               `{"name":"place","NAME":"other","arguments":{}}`,
		"number 1e999":                 `{"name":"place","arguments":{"n":1e999}}`,
		"number -1e999":                `{"name":"place","arguments":{"n":-1e999}}`,
		"number 1e-999":                `{"name":"place","arguments":{"n":1e-999}}`,
		"number huge exponent":         `{"name":"place","arguments":{"n":1e99999999999}}`,
		"number 400 digits":            `{"name":"place","arguments":{"n":` + strings.Repeat("9", 400) + `}}`,
		"number 41 digits":             `{"name":"place","arguments":{"n":` + strings.Repeat("1", 41) + `}}`,
		"number long fraction":         `{"name":"place","arguments":{"n":0.` + strings.Repeat("1", 60) + `}}`,
		"number over 64 chars":         `{"name":"place","arguments":{"n":0.` + strings.Repeat("0", 70) + `1}}`,
		"number leading zero":          `{"name":"place","arguments":{"n":01}}`,
		"number plus sign":             `{"name":"place","arguments":{"n":+1}}`,
		"number bare dot":              `{"name":"place","arguments":{"n":1.}}`,
		"NaN":                          `{"name":"place","arguments":{"n":NaN}}`,
		"Infinity":                     `{"name":"place","arguments":{"n":Infinity}}`,
		"trailing data":                `{"name":"place","arguments":{}} {}`,
		"trailing comma":               `{"name":"place","arguments":{"a":1,}}`,
		"control char in string":       "{\"name\":\"place\",\"arguments\":{\"a\":\"x\ty\"}}",
		"bad escape":                   `{"name":"place","arguments":{"a":"\x"}}`,
		"short unicode escape":         `{"name":"place","arguments":{"a":"\u12"}}`,
		"arguments array":              `{"name":"place","arguments":[1]}`,
		"arguments string":             `{"name":"place","arguments":"x"}`,
		"too deep":                     `{"name":"place","arguments":` + strings.Repeat(`[`, 70) + strings.Repeat(`]`, 70) + `}`,
		"params not an object":         `[1]`,
	} {
		if h, err := CallHash("tok", "place", json.RawMessage(p)); err == nil {
			t.Errorf("%s: hashed to %s", name, h)
		}
	}
	// Numbers that are exact and in range stay allowed, spelled as sent.
	for name, n := range map[string]string{
		"integer": "75", "decimal": "75.50", "zero": "0", "negative zero": "-0", "exponent": "7.5e1",
		"40 digits": strings.Repeat("1", 40), "small": "0.000000000001", "1e308": "1e308", "tiny": "1e-307",
	} {
		if _, err := CallHash("tok", "place", call(`{"n":`+n+`}`)); err != nil {
			t.Errorf("%s: refused: %v", name, err)
		}
	}
}

// What is spelled differently but read the same is one call; what is read
// differently is not. Numbers are text, never converted.
func TestCallHashNumbersAreExactText(t *testing.T) {
	a, _ := CallHash("tok", "place", call(`{"n":9007199254740993}`))
	b, _ := CallHash("tok", "place", call(`{"n":9007199254740992}`))
	c, _ := CallHash("tok", "place", call(`{"n":9007199254740993.0}`))
	if a == "" || a == b || a == c {
		t.Fatal("numbers beyond float64 precision collide")
	}
	x, _ := CallHash("tok", "place", call(`{"n":0.1}`))
	y, _ := CallHash("tok", "place", call(`{"n":0.10}`))
	z, _ := CallHash("tok", "place", call(`{"n":1e-1}`))
	if x == y || x == z || y == z {
		t.Fatal("different spellings of a number share a hash")
	}
}

// strictArgs is the strict reading of the forwarded call, minus the members
// the hash leaves out on purpose (name equals the tool; _meta.token and
// _meta.progressToken do not say what is ordered).
func strictForwarded(t *testing.T, params []byte) interface{} {
	t.Helper()
	v, err := readStrict(params)
	if err != nil {
		t.Fatalf("an accepted call does not decode strictly: %v", err)
	}
	top := v.(objectEntries)
	delete(top, "name")
	if meta, ok := top["_meta"].(objectEntries); ok {
		delete(meta, "token")
		delete(meta, "progressToken")
		if len(meta) == 0 {
			delete(top, "_meta")
		}
	}
	return map[string]interface{}(top)
}

// mutations of a call: some are the same call spelled differently, some are
// not; some cannot be read without loss and must be refused.
var mutationSeeds = []string{
	`{"name":"place","arguments":{"symbol":"BTC-USD","side":"buy","dollar_amount":"75","legs":[{"a":1,"b":[1,2,3]}]}}`,
	`{"name":"place","arguments":{"symbol":"\ufffd","dollar_amount":75}}`,
	`{"name":"place","arguments":{"symbol":"\ud83d\ude00","n":1.50}}`,
	`{"name":"place","arguments":{}}`,
	`{"name":"place"}`,
	`{"name":"place","arguments":null,"_meta":{"token":"abc","progressToken":9,"x":{"y":1}}}`,
}

var spellings = []struct{ from, to string }{
	{`"symbol"`, `"sym\u0062ol"`},
	{`"BTC-USD"`, `"BTC\u002dUSD"`},
	{`"buy"`, `"b\u0075y"`},
	{`"\ufffd"`, `"\ud800"`},
	{`"\ufffd"`, `"\udc00"`},
	{`"\ufffd"`, "\"\xef\xbf\xbd\""},
	{`"\ufffd"`, "\"\xff\""},
	{`"\ud83d\ude00"`, `"\ud83d"`},
	{`"\ud83d\ude00"`, `"\ude00\ud83d"`},
	{`"\ud83d\ude00"`, `"\uD83D\uDE00"`},
	{`"\ud83d\ude00"`, "\"\U0001F600\""},
	{`"75"`, `"75.0"`},
	{`"75"`, `75`},
	{`75`, `75.0`},
	{`75`, `7.5e1`},
	{`1.50`, `1.5`},
	{`"a":1`, `"a":1,"a":2`},
	{`"b":[1,2,3]`, `"b":[3,2,1]`},
	{`"n":1.50`, `"n":1.50,"N":1.50`},
	{`"arguments"`, `"Arguments"`},
	{`{"name":"place"`, `{"name":"place","name":"place"`},
	{`{"name":"place"`, `{ "name" : "place" ,"extra":1`},
	{`,"dollar_amount"`, `,"dollar_amount":"1","dollar_amount"`},
	{`"y":1`, `"y":1e999`},
	{`"y":1`, `"y":1e-999`},
	{`"y":1`, `"y":` + "123456789012345678901234567890123456789012345"},
	{`"token":"abc"`, `"token":"zzz"`},
	{`"progressToken":9`, `"progressToken":10`},
	{`:`, ` : `},
	{`,`, ` , `},
}

func mutate(r *rand.Rand, seed string) string {
	s := seed
	for n := 1 + r.Intn(3); n > 0; n-- {
		m := spellings[r.Intn(len(spellings))]
		if strings.Contains(s, m.from) {
			if r.Intn(2) == 0 {
				s = strings.Replace(s, m.from, m.to, 1)
			} else {
				s = strings.ReplaceAll(s, m.from, m.to)
			}
		}
	}
	if r.Intn(6) == 0 && len(s) > 4 { // a byte-level slip
		i := r.Intn(len(s))
		s = s[:i] + string([]byte{byte(r.Intn(256))}) + s[i+1:]
	}
	return s
}

// Property: of any two calls the hash accepts, equal hashes mean the
// forwarded bytes decode to equal values under the strict decoder. The hash
// may only merge spellings, never meanings.
func TestPropertyEqualHashImpliesEqualStrictDecode(t *testing.T) {
	r := rand.New(rand.NewSource(227))
	byHash := map[string][]string{}
	accepted, refused := 0, 0
	for i := 0; i < 60000; i++ {
		p := mutate(r, mutationSeeds[r.Intn(len(mutationSeeds))])
		h, err := CallHash("tok", "place", json.RawMessage(p))
		if err != nil {
			refused++
			continue
		}
		accepted++
		byHash[h] = append(byHash[h], p)
	}
	merged := 0
	for h, group := range byHash {
		first := strictForwarded(t, []byte(group[0]))
		for _, other := range group[1:] {
			merged++
			if got := strictForwarded(t, []byte(other)); !reflect.DeepEqual(first, got) {
				t.Fatalf("hash %s covers two different calls:\n  %q\n  %q", h[:12], group[0], other)
			}
		}
	}
	// The corpus must exercise both sides, or the property says nothing.
	if accepted < 5000 || refused < 5000 || merged < 2000 || len(byHash) < 20 {
		t.Fatalf("corpus too thin: accepted=%d refused=%d merged=%d distinct=%d", accepted, refused, merged, len(byHash))
	}
	t.Logf("accepted=%d refused=%d distinct hashes=%d same-hash pairs checked=%d", accepted, refused, len(byHash), merged)
}

// The same property, fed by the Go fuzzer when it is run (go test -fuzz). Under
// plain go test it runs the seeds.
func FuzzCallHashIsLossless(f *testing.F) {
	for _, s := range mutationSeeds {
		f.Add(s, s)
	}
	f.Add(`{"name":"place","arguments":{"s":"\ufffd"}}`, `{"name":"place","arguments":{"s":"\ud800"}}`)
	f.Add(`{"name":"place","arguments":{"s":"\ud800"}}`, `{"name":"place","arguments":{"s":"\ud801"}}`)
	f.Fuzz(func(t *testing.T, a, b string) {
		ha, ea := CallHash("tok", "place", json.RawMessage(a))
		hb, eb := CallHash("tok", "place", json.RawMessage(b))
		if ea != nil || eb != nil || ha != hb {
			return
		}
		if !reflect.DeepEqual(strictForwarded(t, []byte(a)), strictForwarded(t, []byte(b))) {
			t.Fatalf("same hash, different calls: %q %q", a, b)
		}
	})
}
