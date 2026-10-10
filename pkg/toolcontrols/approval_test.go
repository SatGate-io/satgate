package toolcontrols

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const askDoc = `{"v":1,"approvals":[{"tools":["place_crypto_order"],"field":"dollar_amount","above":"50"}]}`
const bothDoc = `{"v":1,"spend_limits":[{"tools":["place_crypto_order"],"field":"dollar_amount","max":"200","window":"day"}],"approvals":[{"tools":["place_crypto_order"],"field":"dollar_amount","above":"50"}]}`

func TestApprovalRuleParseAndCanonical(t *testing.T) {
	d, err := ParseJSON([]byte(askDoc))
	if err != nil {
		t.Fatal(err)
	}
	if len(d.SpendLimits) != 0 || len(d.Approvals) != 1 {
		t.Fatalf("doc = %+v", d)
	}
	r := d.Approvals[0]
	if r.Field != "dollar_amount" || r.Above != "50" || !r.Covers("place_crypto_order") || r.Covers("get_quote") {
		t.Fatalf("rule = %+v", r)
	}
	canon, err := Canonical([]byte(askDoc))
	if err != nil || string(canon) != askDoc {
		t.Fatalf("canonical = %s %v", canon, err)
	}
	// Both kinds in one document keep their fixed order.
	canon, err = Canonical([]byte(bothDoc))
	if err != nil {
		t.Fatal(err)
	}
	want := `{"v":1,"spend_limits":[{"tools":["place_crypto_order"],"field":"dollar_amount","max":"200","window":"day","tz":"America/New_York"}],"approvals":[{"tools":["place_crypto_order"],"field":"dollar_amount","above":"50"}]}`
	if string(canon) != want {
		t.Fatalf("canonical = %s", canon)
	}
	again, err := Canonical(canon)
	if err != nil || string(again) != want {
		t.Fatalf("canonical is not stable: %s %v", again, err)
	}
}

func TestApprovalRuleParseRejects(t *testing.T) {
	bad := map[string]string{
		"unknown key":     `{"v":1,"approvals":[{"tools":["a"],"field":"f","above":"1","who":"me"}]}`,
		"missing above":   `{"v":1,"approvals":[{"tools":["a"],"field":"f"}]}`,
		"missing field":   `{"v":1,"approvals":[{"tools":["a"],"above":"1"}]}`,
		"missing tools":   `{"v":1,"approvals":[{"field":"f","above":"1"}]}`,
		"numeric above":   `{"v":1,"approvals":[{"tools":["a"],"field":"f","above":1}]}`,
		"exponent":        `{"v":1,"approvals":[{"tools":["a"],"field":"f","above":"1e3"}]}`,
		"negative":        `{"v":1,"approvals":[{"tools":["a"],"field":"f","above":"-1"}]}`,
		"empty list":      `{"v":1,"approvals":[]}`,
		"not a list":      `{"v":1,"approvals":{}}`,
		"dup tool":        `{"v":1,"approvals":[{"tools":["a","a"],"field":"f","above":"1"}]}`,
		"bad tool":        `{"v":1,"approvals":[{"tools":["bad name"],"field":"f","above":"1"}]}`,
		"dup key":         `{"v":1,"approvals":[{"tools":["a"],"field":"f","above":"1","above":"2"}]}`,
		"too many digits": `{"v":1,"approvals":[{"tools":["a"],"field":"f","above":"1.123456789"}]}`,
	}
	for name, doc := range bad {
		if _, err := ParseJSON([]byte(doc)); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	many := make([]string, MaxApprovalRules+1)
	for i := range many {
		many[i] = `{"tools":["a"],"field":"f","above":"1"}`
	}
	if _, err := ParseJSON([]byte(`{"v":1,"approvals":[` + strings.Join(many, ",") + `]}`)); err == nil {
		t.Error("too many approval rules accepted")
	}
	// Zero is a real threshold: every call is held.
	if _, err := ParseJSON([]byte(`{"v":1,"approvals":[{"tools":["a"],"field":"f","above":"0"}]}`)); err != nil {
		t.Errorf("above 0 refused: %v", err)
	}
}

func TestCollectApprovalsAndSpendLimitsTogether(t *testing.T) {
	d, err := ParseJSON([]byte(bothDoc))
	if err != nil {
		t.Fatal(err)
	}
	caveat, err := Caveat(d)
	if err != nil {
		t.Fatal(err)
	}
	rules, err := CollectApprovals([]string{"scope = place_crypto_order", caveat})
	if err != nil || len(rules) != 1 || rules[0].ID == "" {
		t.Fatalf("rules = %+v %v", rules, err)
	}
	limits, err := Collect([]string{"scope = place_crypto_order", caveat})
	if err != nil || len(limits) != 1 || limits[0].ID == "" || limits[0].ID == rules[0].ID {
		t.Fatalf("limits = %+v %v", limits, err)
	}
	if got := DescribeApprovals(rules); len(got) != 1 || got[0] != "place_crypto_order: calls with dollar_amount above $50 wait for your approval" {
		t.Fatalf("describe = %v", got)
	}
	// A word that does not parse is an error, never skipped.
	if _, err := CollectApprovals([]string{"scope = argrules:ctl:v1:zz:zz"}); err == nil {
		t.Fatal("bad word skipped")
	}
}

func TestCallHash(t *testing.T) {
	h := func(token, tool, params string) string {
		t.Helper()
		out, err := CallHash(token, tool, json.RawMessage(params))
		if err != nil {
			t.Fatalf("%s: %v", params, err)
		}
		return out
	}
	base := h("tok", "place", `{"name":"place","arguments":{"symbol":"BTC","dollar_amount":"60","side":"buy"}}`)
	same := []string{
		// key order, whitespace, _meta and extra top-level keys are not the call
		`{"name":"place","arguments":{"side":"buy","dollar_amount":"60","symbol":"BTC"}}`,
		"{ \"name\" : \"place\",\n \"arguments\" : { \"symbol\":\"BTC\", \"dollar_amount\":\"60\", \"side\":\"buy\" } }",
		`{"name":"place","arguments":{"symbol":"BTC","dollar_amount":"60","side":"buy"},"_meta":{"token":"abc","progressToken":7}}`,
	}
	for _, p := range same {
		if h("tok", "place", p) != base {
			t.Errorf("same call hashes differently: %s", p)
		}
	}
	different := map[string]string{
		"amount":      `{"name":"place","arguments":{"symbol":"BTC","dollar_amount":"61","side":"buy"}}`,
		"amount type": `{"name":"place","arguments":{"symbol":"BTC","dollar_amount":60,"side":"buy"}}`,
		"number text": `{"name":"place","arguments":{"symbol":"BTC","dollar_amount":"60.0","side":"buy"}}`,
		"side":        `{"name":"place","arguments":{"symbol":"BTC","dollar_amount":"60","side":"sell"}}`,
		"extra arg":   `{"name":"place","arguments":{"symbol":"BTC","dollar_amount":"60","side":"buy","x":1}}`,
		"less arg":    `{"name":"place","arguments":{"symbol":"BTC","dollar_amount":"60"}}`,
		"nested":      `{"name":"place","arguments":{"symbol":"BTC","dollar_amount":"60","side":"buy","legs":[{"a":1}]}}`,
		"case":        `{"name":"place","arguments":{"symbol":"btc","dollar_amount":"60","side":"buy"}}`,
	}
	seen := map[string]string{base: "base"}
	for name, p := range different {
		got := h("tok", "place", p)
		if prev, dup := seen[got]; dup {
			t.Errorf("%s hashes like %s", name, prev)
		}
		seen[got] = name
	}
	if h("tok2", "place", `{"name":"place","arguments":{"symbol":"BTC","dollar_amount":"60","side":"buy"}}`) == base {
		t.Error("another token hashes the same")
	}
	// Nested key order is also not part of the call.
	a := h("tok", "p", `{"name":"p","arguments":{"legs":[{"a":1,"b":2}]}}`)
	b := h("tok", "p", `{"name":"p","arguments":{"legs":[{"b":2,"a":1}]}}`)
	if a != b {
		t.Error("nested key order changes the hash")
	}
	// Array order is part of the call.
	if h("tok", "p", `{"name":"p","arguments":{"l":[1,2]}}`) == h("tok", "p", `{"name":"p","arguments":{"l":[2,1]}}`) {
		t.Error("array order ignored")
	}
	// Absent, null and empty arguments are three different byte strings. They
	// are three calls: the hash does not decide that a consumer reads them alike.
	abs, null, empty := h("tok", "p", `{"name":"p"}`), h("tok", "p", `{"name":"p","arguments":null}`), h("tok", "p", `{"name":"p","arguments":{}}`)
	if abs == null || abs == empty || null == empty {
		t.Error("absent, null and empty arguments share a hash")
	}
}

func TestCallHashRefusesUnreadableCalls(t *testing.T) {
	for name, p := range map[string]string{
		"dup arg key": `{"name":"p","arguments":{"a":1,"a":2}}`,
		"dup top key": `{"name":"p","name":"p","arguments":{}}`,
		"wrong name":  `{"name":"other","arguments":{}}`,
		"args array":  `{"name":"p","arguments":[1]}`,
		"not json":    `nope`,
		"bad utf8":    "{\"name\":\"p\",\"arguments\":{\"a\":\"\xff\"}}",
		"too deep":    `{"name":"p","arguments":` + strings.Repeat(`{"a":`, 80) + `1` + strings.Repeat(`}`, 80) + `}`,
	} {
		if _, err := CallHash("tok", "p", json.RawMessage(p)); err == nil {
			t.Errorf("%s: hashed", name)
		}
	}
	if _, err := CallHash("", "p", json.RawMessage(`{"name":"p"}`)); err == nil {
		t.Error("empty token id hashed")
	}
}

func TestOwnerSummaryShowsValuesToTheOwner(t *testing.T) {
	s := OwnerSummary(json.RawMessage(`{"name":"place","arguments":{"symbol":"BTC-USD","side":"buy","dollar_amount":"60","type":"market"}}`), "dollar_amount")
	want := []SummaryItem{{"dollar_amount", "60"}, {"side", "buy"}, {"symbol", "BTC-USD"}, {"type", "market"}}
	if len(s) != len(want) {
		t.Fatalf("summary = %+v", s)
	}
	for i := range want {
		if s[i] != want[i] {
			t.Fatalf("summary[%d] = %+v, want %+v", i, s[i], want[i])
		}
	}
	long := strings.Repeat("x", 500)
	s = OwnerSummary(json.RawMessage(`{"name":"p","arguments":{"memo":"`+long+`"}}`), "f")
	if len(s) != 1 || len(s[0].Value) > maxSummaryValue+20 || !strings.HasSuffix(s[0].Value, "(cut)") {
		t.Fatalf("long value not cut: %+v", s)
	}
	if OwnerSummary(json.RawMessage(`nope`), "f") != nil {
		t.Fatal("unreadable params gave a summary")
	}
}

func approvalReq(token, hash string, now time.Time) ApprovalRequest {
	return ApprovalRequest{TenantID: "tenant-a", TokenID: token, Tool: "place", CallHash: hash, Field: "dollar_amount", Above: "50", Now: now}
}

func TestMemoryApprovalStoreLifecycle(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryApprovalStore()
	now := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	d, err := s.Decide(ctx, approvalReq("tok", "h1", now))
	if err != nil || d.Outcome != ApprovalHeld || !d.NeedsNotice || !ValidApprovalID(d.ApprovalID) {
		t.Fatalf("first = %+v %v", d, err)
	}
	if !d.ExpiresAt.Equal(now.Add(ApprovalTTL)) {
		t.Fatalf("expires = %v", d.ExpiresAt)
	}
	id := d.ApprovalID
	// A repeat waits on the same row.
	d2, _ := s.Decide(ctx, approvalReq("tok", "h1", now.Add(time.Minute)))
	if d2.Outcome != ApprovalHeld || d2.ApprovalID != id {
		t.Fatalf("repeat = %+v", d2)
	}
	_ = s.MarkNotified(ctx, "tenant-a", id, now)
	d3, _ := s.Decide(ctx, approvalReq("tok", "h1", now.Add(2*time.Minute)))
	if d3.NeedsNotice {
		t.Fatal("owner told twice")
	}
	// Another tenant cannot approve it; the owner can.
	if err := s.Approve("tenant-b", id, "mallory", now); err != ErrApprovalNotFound {
		t.Fatalf("cross-tenant approve = %v", err)
	}
	if err := s.Approve("tenant-a", id, "owner", now.Add(3*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := s.Approve("tenant-a", id, "owner", now.Add(3*time.Minute)); err != ErrApprovalNotPending {
		t.Fatalf("second approve = %v", err)
	}
	// Another token, another call, another tenant: no match.
	for _, r := range []ApprovalRequest{
		approvalReq("tok2", "h1", now.Add(4*time.Minute)),
		approvalReq("tok", "h2", now.Add(4*time.Minute)),
	} {
		o, _ := s.Decide(ctx, r)
		if o.Outcome != ApprovalHeld || o.ApprovalID == id {
			t.Fatalf("mismatch matched: %+v", o)
		}
	}
	other := approvalReq("tok", "h1", now.Add(4*time.Minute))
	other.TenantID = "tenant-b"
	if o, _ := s.Decide(ctx, other); o.Outcome != ApprovalHeld || o.ApprovalID == id {
		t.Fatalf("another tenant matched: %+v", o)
	}
	// The match is used once.
	c, _ := s.Decide(ctx, approvalReq("tok", "h1", now.Add(5*time.Minute)))
	if c.Outcome != ApprovalConsumed || c.ApprovalID != id {
		t.Fatalf("consume = %+v", c)
	}
	again, _ := s.Decide(ctx, approvalReq("tok", "h1", now.Add(5*time.Minute)))
	if again.Outcome != ApprovalHeld || again.ApprovalID == id {
		t.Fatalf("second use = %+v", again)
	}
}

func TestMemoryApprovalStoreDeniedAndExpired(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryApprovalStore()
	now := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	d, _ := s.Decide(ctx, approvalReq("tok", "h", now))
	if err := s.Deny("tenant-a", d.ApprovalID, "owner", now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if o, _ := s.Decide(ctx, approvalReq("tok", "h", now.Add(2*time.Minute))); o.Outcome != ApprovalDenied || o.ApprovalID != d.ApprovalID {
		t.Fatalf("denied = %+v", o)
	}
	// After the denial runs out, the same call can ask again.
	if o, _ := s.Decide(ctx, approvalReq("tok", "h", now.Add(ApprovalTTL+time.Second))); o.Outcome != ApprovalHeld || o.ApprovalID == d.ApprovalID {
		t.Fatalf("after denial = %+v", o)
	}

	// An approval that is not used in time expires, once.
	e, _ := s.Decide(ctx, approvalReq("tok", "h2", now))
	if err := s.Approve("tenant-a", e.ApprovalID, "owner", now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	late := now.Add(ApprovalTTL + time.Second)
	if o, _ := s.Decide(ctx, approvalReq("tok", "h2", late)); o.Outcome != ApprovalExpired || o.ApprovalID != e.ApprovalID {
		t.Fatalf("expired = %+v", o)
	}
	if o, _ := s.Decide(ctx, approvalReq("tok", "h2", late)); o.Outcome != ApprovalHeld || o.ApprovalID == e.ApprovalID {
		t.Fatalf("after expiry = %+v", o)
	}
	// A late owner cannot approve a row that ran out.
	f, _ := s.Decide(ctx, approvalReq("tok", "h3", now))
	if err := s.Approve("tenant-a", f.ApprovalID, "owner", now.Add(ApprovalTTL)); err != ErrApprovalNotPending {
		t.Fatalf("late approve = %v", err)
	}
}

func TestMemoryApprovalStoreQueueLimit(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryApprovalStore()
	now := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	for i := 0; i < MaxPendingApprovals; i++ {
		d, err := s.Decide(ctx, approvalReq("tok", "h"+string(rune('a'+i)), now))
		if err != nil || d.Outcome != ApprovalHeld {
			t.Fatalf("%d: %+v %v", i, d, err)
		}
	}
	if d, _ := s.Decide(ctx, approvalReq("tok", "overflow", now)); d.Outcome != ApprovalQueueFull {
		t.Fatalf("overflow = %+v", d)
	}
	// Another token has its own queue; the full one still answers its waiting calls.
	if d, _ := s.Decide(ctx, approvalReq("tok2", "overflow", now)); d.Outcome != ApprovalHeld {
		t.Fatalf("other token = %+v", d)
	}
	if d, _ := s.Decide(ctx, approvalReq("tok", "ha", now)); d.Outcome != ApprovalHeld {
		t.Fatalf("waiting call = %+v", d)
	}
}

// Of many concurrent retries of one approved call, exactly one is let through.
func TestMemoryApprovalStoreConsumesOnceUnderConcurrency(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryApprovalStore()
	now := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	d, _ := s.Decide(ctx, approvalReq("tok", "h", now))
	if err := s.Approve("tenant-a", d.ApprovalID, "owner", now); err != nil {
		t.Fatal(err)
	}
	var consumed int64
	var wg sync.WaitGroup
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			o, err := s.Decide(ctx, approvalReq("tok", "h", now.Add(time.Second)))
			if err == nil && o.Outcome == ApprovalConsumed {
				atomic.AddInt64(&consumed, 1)
			}
		}()
	}
	wg.Wait()
	if consumed != 1 {
		t.Fatalf("consumed %d times", consumed)
	}
}

func TestApprovalIDShape(t *testing.T) {
	id, err := NewApprovalID()
	if err != nil || !ValidApprovalID(id) {
		t.Fatalf("id = %q %v", id, err)
	}
	for _, bad := range []string{"", "apr_", "apr_zz", "apr_0123456789abcdef0123456", "apr_0123456789ABCDEF01234567", "xyz_0123456789abcdef01234567", "apr_0123456789abcdef012345678"} {
		if ValidApprovalID(bad) {
			t.Errorf("%q accepted", bad)
		}
	}
}

func TestOwnerSummaryHasNoNULForTheDatabase(t *testing.T) {
	s := OwnerSummary(json.RawMessage(`{"name":"p","arguments":{"memo":"a\u0000b","k\u0000":1}}`), "f")
	for _, it := range s {
		if strings.ContainsRune(it.Name, 0) || strings.ContainsRune(it.Value, 0) {
			t.Fatalf("NUL in summary: %+v", it)
		}
	}
	b, _ := json.Marshal(s)
	if strings.Contains(string(b), `\u0000`) {
		t.Fatalf("summary JSON holds \\u0000: %s", b)
	}
}
