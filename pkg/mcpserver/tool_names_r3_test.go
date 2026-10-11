package mcpserver

// Round 3 of the tool-names work, public side: no temporary names, a copy of
// the config map, no tool names in the connection log, a service whose
// credentials could not be read lists nothing. Fakes only, loopback only.
//
// Cited probes (adapted): Astra r2 reviewer_r2_restart_test.go
// (TestReviewerR2RestartMustNotDropPreviouslyEnforcedMoneyRule,
// TestReviewerR2RefusedDescriptorMustNotAppearInConnectLog),
// reviewer_r2_http_restart_test.go (TestReviewerR2HTTPHealthyRestartMoney),
// reviewer_r2_public_test.go (TestReviewerR2HealthyRestartLosesPrefixedNonclashingName)
// and Grok r2 reviewer_r2_money_probe_test.go (race on the shared config map).

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/satgate-io/satgate/pkg/toolcontrols"
)

// ---------------------------------------------------------------------------
// Fix 3: no temporary names in public.

func TestR3Static_NoTemporaryNameIsEverOfferedWhileAnEarlierServiceIsDown(t *testing.T) {
	early, pg := newPagedService(t, "Early", []fakeTool{{Name: "account"}})
	pg.set("down")
	alp := newFakeService(t, "ALP", fakeTool{Name: "place_stock_order"}, fakeTool{Name: "account"})
	cfg := map[string]UpstreamConfig{"a": early.cfg("Early"), "b": alp.cfg("Alpaca")}
	m, buf := startCfg(t, fast, nil, cfg)
	if got := listTools(t, m); len(got) != 0 {
		t.Fatalf("listed while an earlier service is down: %v", names(got))
	}
	for _, n := range m.ToolNames() {
		if n.Reason == ToolRenamedPending {
			t.Fatalf("a temporary (pending) name was offered: %+v", n)
		}
	}
	for _, name := range []string{"place_stock_order", "alpaca_place_stock_order", "account", "alpaca_account"} {
		if _, err := callName(t, m, name); err == nil || !strings.Contains(err.Error(), ToolsNotAvailableYetMessage) {
			t.Fatalf("%s: %v", name, err)
		}
	}
	if alp.totalCalls() != 0 {
		t.Fatal("a call reached the later service while its names were undecided")
	}
	// One startup log line says why and what to do.
	out := buf.String()
	if n := strings.Count(out, "not listed until an earlier service answers"); n != 1 {
		t.Fatalf("want one startup waiting line, got %d:\n%s", n, out)
	}
	if !strings.Contains(out, "turn the earlier service off or change the order") {
		t.Fatalf("the line has no remedy:\n%s", out)
	}
	if strings.Contains(out, "place_stock_order") {
		t.Fatalf("a tool name is in the log:\n%s", out)
	}
	pg.set("")
	eventually(t, "the earlier service to answer", func() bool { return len(listTools(t, m)) == 3 })
	got := listTools(t, m)
	if _, err := callName(t, m, "place_stock_order"); err != nil {
		t.Fatalf("place_stock_order: %v (%v)", err, names(got))
	}
	// "account" was offered by both: the earlier service has the plain name.
	if r, _ := callName(t, m, "account"); r != "Early:account" {
		t.Fatalf("account reaches %q", r)
	}
	if r, _ := callName(t, m, "alpaca_account"); r != "ALP:account" {
		t.Fatalf("alpaca_account reaches %q", r)
	}
}

func TestR3Static_TheSameConfigAndToolListsGiveTheSameNamesWhateverWasUpAtStart(t *testing.T) {
	early, pg := newPagedService(t, "Early", []fakeTool{{Name: "account"}, {Name: "place_stock_order"}})
	alp := newFakeService(t, "ALP", fakeTool{Name: "place_stock_order"}, fakeTool{Name: "get_positions"})
	cfg := map[string]UpstreamConfig{"a": early.cfg("Early"), "b": alp.cfg("Alpaca")}
	healthy, _ := startCfg(t, nil, nil, cfg)
	want := names(listTools(t, healthy))
	pg.set("down")
	degraded, _ := startCfg(t, fast, nil, cfg)
	if got := listTools(t, degraded); len(got) != 0 {
		t.Fatalf("degraded start listed %v", names(got))
	}
	pg.set("")
	eventually(t, "the earlier service to answer", func() bool { return len(listTools(t, degraded)) == len(want) })
	if got := names(listTools(t, degraded)); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("degraded-then-healthy %v, healthy %v", got, want)
	}
	for _, n := range want {
		a, _ := callName(t, healthy, n)
		b, _ := callName(t, degraded, n)
		if a != b {
			t.Errorf("%s reaches %q on the healthy start and %q after the degraded one", n, a, b)
		}
	}
}

func TestR3Static_ARoutingRuleMovesTheWaitToTheServiceItFavours(t *testing.T) {
	a := newFakeService(t, "A", fakeTool{Name: "ping"})
	b, bp := newPagedService(t, "B", []fakeTool{{Name: "ping"}})
	bp.set("down")
	routing := []RoutingRule{{Tools: []string{"ping"}, Upstream: "b"}}
	m, _ := startCfg(t, fast, routing, map[string]UpstreamConfig{"a": a.cfg("A"), "b": b.cfg("B")})
	// "a" is first in sorted order but a rule favours b for ping: a's ping
	// waits for b; nothing is listed under a temporary name.
	if got := listTools(t, m); len(got) != 0 {
		t.Fatalf("listed %v", names(got))
	}
	bp.set("")
	eventually(t, "b to answer", func() bool { return len(listTools(t, m)) == 2 })
	if r, _ := callName(t, m, "ping"); r != "B:ping" {
		t.Fatalf("ping reaches %q, want the favoured service", r)
	}
}

// A limit enforced under a name that was listed is enforced under the same
// name after a restart, and a call to a tool that is not listed yet is refused
// before it reaches any service (so no limit can be skipped by a spelling).
func TestR3Static_ALimitEnforcedBeforeARestartIsEnforcedAfterItUnderTheSameName(t *testing.T) {
	early, pg := newPagedService(t, "Early", []fakeTool{{Name: "account"}, {Name: "place_stock_order"}})
	alp := newFakeService(t, "Alpaca", fakeTool{Name: "place_stock_order"})
	cfg := map[string]UpstreamConfig{"a": early.cfg("Early"), "b": alp.cfg("Alpaca")}
	doc := `{"v":1,"spend_limits":[{"tools":["alpaca_place_stock_order"],"field":"notional","max":"1","window":"day"}]}`
	run := func(m *UpstreamManager, tool string) (*Response, *Proxy) {
		p, _, _ := newArgProxy(t)
		p.upstream = m
		p.SetUpstreamRouter(&defaultRouter{mgr: m})
		p.SetCallGate(NewSpendGate(toolcontrols.NewMemoryStore(nil)))
		svc, tok := mintBudgetToken(t, argTestRoot, "*", "tenant-1", "r3-budget", "50")
		tok = appendCaveat(t, svc, tok, controlWord(t, doc))
		return callProxy(t, p, 1, tok, tool, `{"notional":"2"}`), p
	}
	// Healthy start: Early has the plain name, Alpaca's is prefixed.
	healthy, _ := startCfg(t, nil, nil, cfg)
	if r, _ := run(healthy, "alpaca_place_stock_order"); r.Error == nil || alp.totalCalls() != 0 {
		t.Fatalf("limit did not deny before the restart: %+v calls=%d", r, alp.totalCalls())
	}
	healthy.Close()
	// Degraded start: the name is not offered; the call is refused outright.
	pg.set("down")
	degraded, _ := startCfg(t, fast, nil, cfg)
	for _, tool := range []string{"alpaca_place_stock_order", "place_stock_order"} {
		r, _ := run(degraded, tool)
		if r.Error == nil {
			t.Fatalf("%s was allowed at a degraded start: %+v", tool, r)
		}
	}
	if alp.totalCalls() != 0 {
		t.Fatal("a call reached Alpaca at a degraded start")
	}
	degraded.Close()
	// Healthy restart: the same name, the same limit, the same denial.
	pg.set("")
	again, _ := startCfg(t, nil, nil, cfg)
	if r, _ := run(again, "alpaca_place_stock_order"); r.Error == nil || alp.totalCalls() != 0 {
		t.Fatalf("limit not enforced after the restart: %+v calls=%d", r, alp.totalCalls())
	}
	// And the plain spelling is the other service's tool, not an alias.
	if r, _ := callName(t, again, "place_stock_order"); r != "Early:place_stock_order" {
		t.Fatalf("place_stock_order reaches %q", r)
	}
}

func TestR3Static_HTTPDegradedThenHealthyStartListsEveryToolUnderTheSameName(t *testing.T) {
	early, pg := newPagedService(t, "Early", []fakeTool{{Name: "account"}})
	alp := newFakeService(t, "Alpaca", fakeTool{Name: "place_stock_order"})
	cfg := map[string]UpstreamConfig{"a": early.cfg("Early"), "b": alp.cfg("Alpaca")}
	svc, tok := mintBudgetToken(t, argTestRoot, "*", "tenant-1", "r3-http", "50")
	_ = svc
	list := func(healthy bool) []string {
		if !healthy {
			pg.set("down")
		} else {
			pg.set("")
		}
		m, _ := startCfg(t, fast, nil, cfg)
		defer m.Close()
		p, _, _ := newArgProxy(t)
		p.upstream = m
		p.SetUpstreamRouter(&defaultRouter{mgr: m})
		s := httptest.NewServer(NewSSEServer(p, ":0").Handler())
		defer s.Close()
		sid := mustInitSession(t, s.URL, tok)
		status, _, body := streamablePost(t, s.URL+"/mcp", sid, tok, map[string]any{"jsonrpc": "2.0", "id": 9, "method": "tools/list"})
		if status != 200 {
			t.Fatalf("HTTP %d %s", status, body)
		}
		var resp struct {
			Result struct {
				Tools []struct {
					Name string `json:"name"`
				} `json:"tools"`
			} `json:"result"`
		}
		if err := json.Unmarshal([]byte(body), &resp); err != nil {
			t.Fatal(err)
		}
		var out []string
		for _, tl := range resp.Result.Tools {
			out = append(out, tl.Name)
		}
		return out
	}
	deg := list(false)
	for _, n := range deg {
		if n == "alpaca_place_stock_order" || n == "place_stock_order" {
			t.Fatalf("the later service's tool is listed at a degraded start: %v", deg)
		}
	}
	healthy := list(true)
	if strings.Join(healthy, ",") != "account,place_stock_order" {
		t.Fatalf("healthy list %v", healthy)
	}
}

// ---------------------------------------------------------------------------
// Fix 4: the manager owns a copy of its config.

func TestR3Config_TheManagerCopiesTheMapAndEverythingInsideIt(t *testing.T) {
	f := newFakeService(t, "F", fakeTool{Name: "t1"})
	g := newFakeService(t, "G", fakeTool{Name: "t2"})
	c := f.cfg("F")
	c.Headers = map[string]string{"X-Key": "header-value-1"}
	c.Env = map[string]string{"A": "1"}
	c.Command = []string{"x", "y"}
	c.ExtraSecrets = []string{"extra-secret-1"}
	in := map[string]UpstreamConfig{"f": c}
	routing := []RoutingRule{{Tools: []string{"t1"}, Upstream: "f"}}
	m := NewUpstreamManager(in, routing, "", true)
	// The caller changes everything it passed in.
	in["g"] = g.cfg("G")
	c.Headers["X-Key"] = "changed-by-the-caller"
	c.Env["A"] = "changed"
	c.Command[0] = "changed"
	c.ExtraSecrets[0] = "changed"
	routing[0].Tools[0] = "changed"
	routing[0].Upstream = "changed"
	snap := m.configSnapshot()
	if len(snap) != 1 {
		t.Fatalf("the manager saw the caller's new key: %d", len(snap))
	}
	got := snap["f"]
	if got.Headers["X-Key"] != "header-value-1" || got.Env["A"] != "1" || got.Command[0] != "x" || got.ExtraSecrets[0] != "extra-secret-1" {
		t.Fatalf("a map or slice is shared with the caller: %+v", got)
	}
	if m.routing[0].Tools[0] != "t1" || m.routing[0].Upstream != "f" {
		t.Fatalf("routing is shared: %+v", m.routing)
	}
	// The manager writes its own map; the caller's is untouched.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	t.Cleanup(func() { m.Close() })
	if err := m.AddUpstream(ctx, "h", g.cfg("G")); err != nil {
		t.Fatal(err)
	}
	if _, ok := in["h"]; ok {
		t.Fatal("AddUpstream wrote the caller's map")
	}
	if err := m.RemoveUpstream("h"); err != nil {
		t.Fatal(err)
	}
}

// Start reads the config while the retry and live changes write it: run with
// -race.
func TestR3Config_StartAndLiveChangesNeverRaceOnTheConfig(t *testing.T) {
	a := newFakeService(t, "A", fakeTool{Name: "ta"})
	b := newFakeService(t, "B", fakeTool{Name: "tb"})
	c := newFakeService(t, "C", fakeTool{Name: "tc"})
	in := map[string]UpstreamConfig{"a": a.cfg("A"), "b": b.cfg("B")}
	m := NewUpstreamManager(in, nil, "", true)
	fast(m)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	t.Cleanup(func() { m.Close() })
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 5; i++ {
			_ = m.AddUpstream(ctx, "c", c.cfg("C"))
			_ = m.RemoveUpstream("c")
		}
	}()
	if err := m.Start(ctx); err != nil {
		t.Fatal(err)
	}
	wg.Wait()
	_ = m.AllTools()
	if _, ok := in["c"]; ok {
		t.Fatal("the caller's map was written")
	}
}

// ---------------------------------------------------------------------------
// Fix 5: a service whose credentials could not be read lists nothing.

func TestR3Credentials_AServiceWhoseCredentialsCouldNotBeReadPublishesNothing(t *testing.T) {
	f := newFakeService(t, "F", fakeTool{Name: "stored_tool", Description: "uses refresh-token-private-123"})
	cfg := f.cfg("F")
	cfg.WithholdTools = true
	cfg.ExtraSecrets = nil // the token could not be read: the check has nothing to match
	store := newPermissiveStore(ToolNameAssignment{Name: "stored_tool", Address: ServiceKey(cfg), Original: "stored_tool", Service: "F"})
	m, _ := startCfg(t, func(m *UpstreamManager) { m.SetToolNameStore(store); fast(m) }, nil, map[string]UpstreamConfig{"f": cfg})
	if got := listTools(t, m); len(got) != 0 {
		raw, _ := json.Marshal(m.AllTools())
		t.Fatalf("a tool with a stored name was published without a complete credential set: %s", raw)
	}
	if _, err := callName(t, m, "stored_tool"); err == nil {
		t.Fatal("a call to a withheld service's tool was routed")
	}
	if f.totalCalls() != 0 {
		t.Fatal("a call reached the service")
	}
	// The host reads the credentials again: the tool is checked and listed
	// only if it holds none of them.
	secret := "refresh-token-private-123"
	cfg2 := f.cfg("F")
	cfg2.ExtraSecrets = []string{secret}
	ok := NewUpstreamManager(map[string]UpstreamConfig{"f": cfg2}, nil, "", true)
	ok.SetToolNameStore(store)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	t.Cleanup(func() { ok.Close() })
	if err := ok.Start(ctx); err != nil {
		t.Fatal(err)
	}
	if got := listTools(t, ok); len(got) != 0 {
		t.Fatalf("a tool holding the credential was listed: %v", names(got))
	}
	f.setTools(fakeTool{Name: "stored_tool", Description: "fine"})
	fresh := NewUpstreamManager(map[string]UpstreamConfig{"f": cfg2}, nil, "", true)
	fresh.SetToolNameStore(store)
	t.Cleanup(func() { fresh.Close() })
	if err := fresh.Start(ctx); err != nil {
		t.Fatal(err)
	}
	if got := listTools(t, fresh); got["stored_tool"].Name == "" {
		t.Fatalf("a clean tool is not listed once the credentials are readable: %v", names(got))
	}
}

// ---------------------------------------------------------------------------
// Fix 6: no tool names in the connection log.

func TestR3Logs_TheConnectionLogHoldsTheServiceNameAndCountsOnly(t *testing.T) {
	f := newFakeService(t, "Service",
		fakeTool{Name: "REFUSED_DESCRIPTOR", Description: "synthetic-credential-123"},
		fakeTool{Name: "visible_tool_name", Description: "a visible description"})
	cfg := UpstreamConfig{Transport: "streamable", URL: f.srv.URL, Headers: map[string]string{"Authorization": "Bearer synthetic-credential-123"}, DisplayName: "Robinhood"}
	m, buf := startCfg(t, nil, nil, map[string]UpstreamConfig{"only": cfg})
	if got := listTools(t, m); len(got) != 1 || got["visible_tool_name"].Name == "" {
		t.Fatalf("listed %v", names(got))
	}
	out := buf.String()
	for _, leak := range []string{"REFUSED_DESCRIPTOR", "visible_tool_name", "a visible description", "synthetic-credential-123"} {
		if strings.Contains(out, leak) {
			t.Fatalf("the log holds %q:\n%s", leak, out)
		}
	}
	line := ""
	for _, l := range strings.Split(out, "\n") {
		if strings.Contains(l, "upstream connected") {
			line = l
		}
	}
	for _, want := range []string{`"service":"Robinhood"`, `"listed":1`, `"refused":1`} {
		if !strings.Contains(line, want) {
			t.Fatalf("connection line %q lacks %s", line, want)
		}
	}
	// A live add and a reconnect log the same way.
	g := newFakeService(t, "G", fakeTool{Name: "LIVE_ADDED_TOOL"})
	if err := m.AddUpstream(context.Background(), "g", g.cfg("Alpaca")); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), "LIVE_ADDED_TOOL") {
		t.Fatalf("the live-add line holds a tool name:\n%s", buf.String())
	}
}

func TestR3Logs_TheSSEReconnectLineHoldsNoURL(t *testing.T) {
	// The reconnect line of the SSE transport prints the address digest.
	if d := addressDigest8("https://example.com/mcp?key=SECRETVALUE1"); len(d) != 8 || strings.Contains(d, "SECRET") {
		t.Fatalf("digest %q", d)
	}
}

var _ = time.Second
