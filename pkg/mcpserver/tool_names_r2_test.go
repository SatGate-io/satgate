package mcpserver

// Round 2 of the tool-names work: every page, first-connected names that fail
// closed, reserved names from any path, no credentials in names, a service
// cannot pass as another, two rows at one address. Fakes only, loopback only.
//
// Cited probes (adapted): Astra r1 reviewer_public_test.go
// (TestReviewerLaterPageCall, TestReviewerNormalizedReservedName,
// TestReviewerForeignSlug) and Grok r1 reviewer_r1_http_probe_test.go
// (TestReviewerR1_LaterPageIsUnreachable, TestReviewerR1_SlugSpoofKeepsPlainName).

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// A fake service whose tools/list can be paged, endless, repeating, failing.

type pager struct {
	f *fakeService

	mu        sync.Mutex
	pages     [][]fakeTool // pages[0] is the first; cursor "p2" is pages[1]
	mode      string       // "", "endless", "repeat", "failpage2", "down"
	listCalls int
	auth      map[string]string // tool name as received -> Authorization of that call
}

func newPagedService(t *testing.T, label string, pages ...[]fakeTool) (*fakeService, *pager) {
	t.Helper()
	f := &fakeService{label: label, calls: map[string]int{}}
	pg := &pager{f: f, pages: pages, auth: map[string]string{}}
	f.srv = httptest.NewServer(http.HandlerFunc(pg.serve))
	t.Cleanup(f.srv.Close)
	return f, pg
}

func (p *pager) set(mode string) {
	p.mu.Lock()
	p.mode = mode
	p.mu.Unlock()
}

func (p *pager) setPages(pages ...[]fakeTool) {
	p.mu.Lock()
	p.pages = pages
	p.mu.Unlock()
}

func (p *pager) lists() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.listCalls
}

func (p *pager) authOf(tool string) string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.auth[tool]
}

func toolJSON(tl fakeTool) json.RawMessage {
	schema := tl.Schema
	if schema == nil {
		schema = map[string]any{"type": "object", "properties": map[string]any{}}
	}
	b, _ := json.Marshal(map[string]any{"name": tl.Name, "description": tl.Description, "inputSchema": schema})
	return b
}

func (p *pager) serve(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	var env struct {
		Method string          `json:"method"`
		ID     json.RawMessage `json:"id"`
		Params struct {
			Name   string `json:"name"`
			Cursor string `json:"cursor"`
		} `json:"params"`
	}
	_ = json.Unmarshal(body, &env)
	p.mu.Lock()
	mode := p.mode
	p.mu.Unlock()
	if mode == "down" {
		http.Error(w, "down", http.StatusInternalServerError)
		return
	}
	if env.Method == "tools/call" {
		p.mu.Lock()
		p.auth[env.Params.Name] = r.Header.Get("Authorization")
		p.mu.Unlock()
	}
	if env.Method != "tools/list" {
		r.Body = io.NopCloser(bytes.NewReader(body))
		p.f.serve(w, r)
		return
	}
	p.mu.Lock()
	p.listCalls++
	pages := p.pages
	p.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	reply := func(tools []json.RawMessage, next string) {
		res := map[string]any{"tools": tools}
		if next != "" {
			res["nextCursor"] = next
		}
		b, _ := json.Marshal(res)
		fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":%s}`, env.ID, b)
	}
	switch mode {
	case "endless":
		n := 0
		fmt.Sscanf(env.Params.Cursor, "c%d", &n)
		reply([]json.RawMessage{toolJSON(fakeTool{Name: fmt.Sprintf("endless_%d", n), Description: "d"})}, fmt.Sprintf("c%d", n+1))
		return
	case "repeat":
		reply([]json.RawMessage{toolJSON(fakeTool{Name: "first_page_tool", Description: "d"})}, "same")
		return
	case "failpage2":
		if env.Params.Cursor != "" {
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32603,"message":"page two broke"}}`, env.ID)
			return
		}
	}
	idx := 0
	if env.Params.Cursor != "" {
		fmt.Sscanf(env.Params.Cursor, "p%d", &idx)
		idx--
	}
	if idx < 0 || idx >= len(pages) {
		reply(nil, "")
		return
	}
	var tools []json.RawMessage
	for _, tl := range pages[idx] {
		tools = append(tools, toolJSON(tl))
	}
	next := ""
	if idx+1 < len(pages) {
		next = fmt.Sprintf("p%d", idx+2)
	}
	reply(tools, next)
}

// startCfg is startNames over explicit configs.
func startCfg(t *testing.T, opts func(*UpstreamManager), routing []RoutingRule, cfgs map[string]UpstreamConfig) (*UpstreamManager, *syncBuffer) {
	t.Helper()
	buf := captureLogs(t)
	m := NewUpstreamManager(cfgs, routing, "", true)
	if opts != nil {
		opts(m)
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() { cancel(); m.Close() })
	if err := m.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	return m, buf
}

func fast(m *UpstreamManager) {
	m.retryEvery = 20 * time.Millisecond
	m.waitRetry = 20 * time.Millisecond
}

func eventually(t *testing.T, what string, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		if ok() {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for: %s", what)
}

// ---------------------------------------------------------------------------
// Fix 1: every page.

func TestR2Pages_LaterPageToolIsListedAndReachable(t *testing.T) {
	// Astra TestReviewerLaterPageCall / Grok TestReviewerR1_LaterPageIsUnreachable, adapted:
	// the later-page tool is listed and reachable, and the first page is untouched.
	f, pg := newPagedService(t, "ALP", []fakeTool{{"get_account", "Account.", nil}}, []fakeTool{{"place_stock_order", "Order.", nil}})
	m, _ := startCfg(t, nil, nil, map[string]UpstreamConfig{"alpaca": f.cfg("Alpaca")})
	got := listTools(t, m)
	if len(got) != 2 || got["place_stock_order"].Name == "" || got["get_account"].Name == "" {
		t.Fatalf("listed %v", names(got))
	}
	if r, err := callName(t, m, "place_stock_order"); err != nil || r != "ALP:place_stock_order" {
		t.Fatalf("later-page call: %q %v", r, err)
	}
	if pg.lists() != 2 {
		t.Fatalf("tools/list calls = %d, want 2", pg.lists())
	}
	// One service: the bytes of both pages, in page order.
	all := m.AllTools()
	if string(all[0]) != string(toolJSON(fakeTool{"get_account", "Account.", nil})) || string(all[1]) != string(toolJSON(fakeTool{"place_stock_order", "Order.", nil})) {
		t.Fatalf("single-service list is not the upstream's bytes: %s", all)
	}
}

func TestR2Pages_CapsAtFiftyAndLeavesNoNames(t *testing.T) {
	f, pg := newPagedService(t, "ALP")
	pg.set("endless")
	m, _ := startCfg(t, func(m *UpstreamManager) { m.retryEvery = time.Hour }, nil, map[string]UpstreamConfig{"alpaca": f.cfg("Alpaca")})
	if n := pg.lists(); n != maxToolListPages {
		t.Fatalf("tools/list calls = %d, want the cap %d", n, maxToolListPages)
	}
	if len(m.AllTools()) != 0 {
		t.Fatalf("an incomplete list was published: %v", names(listTools(t, m)))
	}
	_, err := callName(t, m, "endless_0")
	if err == nil || err.Error() != ToolsNotAvailableYetMessage {
		t.Fatalf("err = %v, want the 'not available yet' message", err)
	}
	if f.totalCalls() != 0 {
		t.Fatal("a call reached the service")
	}
}

func TestR2Pages_RepeatedCursorStops(t *testing.T) {
	f, pg := newPagedService(t, "ALP")
	pg.set("repeat")
	m, _ := startCfg(t, func(m *UpstreamManager) { m.retryEvery = time.Hour }, nil, map[string]UpstreamConfig{"alpaca": f.cfg("Alpaca")})
	if n := pg.lists(); n != 2 {
		t.Fatalf("tools/list calls = %d, want 2 (the repeated cursor stops the walk)", n)
	}
	if len(m.AllTools()) != 0 {
		t.Fatal("an incomplete list was published")
	}
}

func TestR2Pages_ALaterPageErrorKeepsTheLastCompleteList(t *testing.T) {
	f, pg := newPagedService(t, "ALP", []fakeTool{{"get_account", "Account.", nil}}, []fakeTool{{"place_stock_order", "Order.", nil}})
	m, _ := startCfg(t, func(m *UpstreamManager) { m.retryEvery = time.Hour }, nil, map[string]UpstreamConfig{"alpaca": f.cfg("Alpaca")})
	before := listTools(t, m)
	// The service changes its list and breaks on page two; the service is reconnected.
	pg.setPages([]fakeTool{{"get_account", "Account.", nil}, {"brand_new", "New.", nil}}, []fakeTool{{"place_stock_order", "Order.", nil}})
	pg.set("failpage2")
	if err := m.AddUpstream(context.Background(), "alpaca", f.cfg("Alpaca")); err != nil {
		t.Fatal(err)
	}
	got := listTools(t, m)
	if len(got) != len(before) || got["brand_new"].Name != "" {
		t.Fatalf("a cut-short list changed the view: %v", names(got))
	}
	if r, err := callName(t, m, "place_stock_order"); err != nil || r != "ALP:place_stock_order" {
		t.Fatalf("a tool of the last complete list stopped working: %q %v", r, err)
	}
	// Healthy again: the new tool appears, the old names are unchanged.
	pg.set("")
	if err := m.AddUpstream(context.Background(), "alpaca", f.cfg("Alpaca")); err != nil {
		t.Fatal(err)
	}
	got = listTools(t, m)
	if got["brand_new"].Name == "" || got["place_stock_order"].Name == "" {
		t.Fatalf("after recovery: %v", names(got))
	}
}

func TestR2Pages_AServiceWithNoCompleteListGetsNoNamesAndOthersAreUnharmed(t *testing.T) {
	rh := newFakeService(t, "RH", robinhoodTools()...)
	f, pg := newPagedService(t, "ALP")
	pg.set("endless")
	m, _ := startCfg(t, func(m *UpstreamManager) { m.SetServiceOrder([]string{"robinhood", "alpaca"}); m.retryEvery = time.Hour }, nil,
		map[string]UpstreamConfig{"robinhood": rh.cfg("Robinhood"), "alpaca": f.cfg("Alpaca")})
	got := listTools(t, m)
	if got["place_crypto_order"].Name == "" || len(got) != 4 {
		t.Fatalf("robinhood's tools are missing: %v", names(got))
	}
	for _, in := range m.ToolNames() {
		if in.Service == "Alpaca" {
			t.Fatalf("a name was assigned to a service with no complete list: %+v", in)
		}
	}
	if _, err := callName(t, m, "alpaca_endless_0"); err == nil || err.Error() != ToolsNotAvailableYetMessage {
		t.Fatalf("err = %v", err)
	}
	if _, err := callName(t, m, "get_account"); err != nil {
		t.Fatalf("a listed tool of the other service failed: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Fix 2, public: degraded start names.

func TestR2Static_AnEarlierServiceThatIsDownNeverLosesItsPlainNames(t *testing.T) {
	rh, rhp := newPagedService(t, "RH", robinhoodTools())
	alp := newFakeService(t, "ALP", alpacaTools()...)
	rhp.set("down")
	store := NewMemoryToolNameStore()
	m, _ := startCfg(t, func(m *UpstreamManager) {
		m.SetToolNameStore(store)
		m.SetServiceOrder([]string{"robinhood", "alpaca"})
		fast(m)
	}, nil,
		map[string]UpstreamConfig{"robinhood": rh.cfg("Robinhood"), "alpaca": alp.cfg("Alpaca")})
	got := listTools(t, m)
	// Alpaca's tools are all listed, prefixed, never plain: one service being
	// down does not hide the other.
	for _, want := range []string{"alpaca_place_crypto_order", "alpaca_get_account", "alpaca_place_stock_order", "alpaca_cancel_order_by_id", "alpaca_get_all_positions"} {
		if got[want].Name == "" {
			t.Errorf("missing %q: %v", want, names(got))
		}
	}
	for _, plain := range []string{"place_crypto_order", "get_account", "place_stock_order"} {
		if got[plain].Name != "" {
			t.Errorf("the later service took the plain name %q while the earlier one is down", plain)
		}
	}
	if w := m.Waiting(); len(w) != 1 || w[0] != (ToolWait{Service: "Alpaca", WaitingFor: "Robinhood"}) {
		t.Fatalf("waiting = %+v", w)
	}
	if r, err := callName(t, m, "alpaca_place_crypto_order"); err != nil || r != "ALP:place_crypto_order" {
		t.Fatalf("alpaca's prefixed name: %q %v", r, err)
	}
	if _, err := callName(t, m, "place_crypto_order"); err == nil {
		t.Fatal("the plain name routed while its owner is down")
	}
	// Robinhood answers: it takes its plain names; Alpaca's names do not move.
	rhp.set("")
	eventually(t, "robinhood to take its plain names", func() bool { return listTools(t, m)["place_crypto_order"].Name != "" })
	if r, err := callName(t, m, "place_crypto_order"); r != "RH:place_crypto_order" {
		t.Fatalf("plain name reaches %q (%v)", r, err)
	}
	after := listTools(t, m)
	for name := range got {
		if _, ok := after[name]; !ok {
			t.Errorf("name %q was lost when robinhood answered", name)
		}
	}
	if r, _ := callName(t, m, "alpaca_place_crypto_order"); r != "ALP:place_crypto_order" {
		t.Fatalf("alpaca_place_crypto_order moved to %q", r)
	}
	if len(m.Waiting()) != 0 {
		t.Fatalf("still waiting: %+v", m.Waiting())
	}
	// A fresh manager with every service up gives the documented healthy names.
	fresh, _ := startCfg(t, func(m *UpstreamManager) { m.SetServiceOrder([]string{"robinhood", "alpaca"}) }, nil,
		map[string]UpstreamConfig{"robinhood": rh.cfg("Robinhood"), "alpaca": alp.cfg("Alpaca")})
	healthy := listTools(t, fresh)
	if healthy["place_stock_order"].Name == "" || healthy["place_crypto_order"].Name == "" {
		t.Fatalf("healthy names: %v", names(healthy))
	}
	// Names the degraded process and the healthy one share name the same service.
	for name := range healthy {
		if _, ok := after[name]; !ok {
			continue
		}
		a, _ := callName(t, m, name)
		b, _ := callName(t, fresh, name)
		if a != b {
			t.Errorf("name %q reaches %q in one process and %q in the other", name, a, b)
		}
	}
}

func TestR2Static_TwoManagersThatSeeDifferentAvailabilityClaimNothingThatDisagrees(t *testing.T) {
	rh, rhp := newPagedService(t, "RH", robinhoodTools())
	alp := newFakeService(t, "ALP", alpacaTools()...)
	store := NewMemoryToolNameStore() // the two "machines" share one store
	order := func(m *UpstreamManager) {
		m.SetToolNameStore(store)
		m.SetServiceOrder([]string{"robinhood", "alpaca"})
		fast(m)
	}
	cfgs := map[string]UpstreamConfig{"robinhood": rh.cfg("Robinhood"), "alpaca": alp.cfg("Alpaca")}

	rhp.set("down")
	m1, _ := startCfg(t, order, nil, cfgs)
	l1 := listTools(t, m1)
	rhp.set("")
	m2, _ := startCfg(t, order, nil, cfgs)
	l2 := listTools(t, m2)
	for _, m := range []*UpstreamManager{m1, m2} {
		eventually(t, "both machines to list robinhood's plain name", func() bool { return listTools(t, m)["place_crypto_order"].Name != "" })
	}
	l1, l2 = listTools(t, m1), listTools(t, m2)
	for name := range l1 {
		a, _ := callName(t, m1, name)
		b, err := callName(t, m2, name)
		if err != nil {
			continue // a name only one machine lists: it must not disagree where both do
		}
		if a != b {
			t.Errorf("name %q reaches %q on one machine and %q on the other", name, a, b)
		}
	}
	if r, _ := callName(t, m1, "place_crypto_order"); r != "RH:place_crypto_order" {
		t.Fatalf("m1: plain name reaches %q", r)
	}
	if r, _ := callName(t, m2, "alpaca_place_crypto_order"); r != "ALP:place_crypto_order" {
		t.Fatalf("m2: prefixed name reaches %q", r)
	}
	_ = l2
}

// ---------------------------------------------------------------------------
// Fix 2, durable order (what the enterprise router wires).

type seenMemStore struct {
	ToolNameStore
	mu      sync.Mutex
	seen    map[string]bool
	failSee atomic.Bool
}

func newSeenStore() *seenMemStore {
	return &seenMemStore{ToolNameStore: NewMemoryToolNameStore(), seen: map[string]bool{}}
}

func (s *seenMemStore) LoadSeen(context.Context) (map[string]bool, error) {
	if s.failSee.Load() {
		return nil, fmt.Errorf("down")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := map[string]bool{}
	for k, v := range s.seen {
		out[k] = v
	}
	return out, nil
}

func (s *seenMemStore) MarkSeen(_ context.Context, k string) error {
	s.mu.Lock()
	s.seen[k] = true
	s.mu.Unlock()
	return nil
}

type orderSource struct {
	mu   sync.Mutex
	err  error
	list []ClaimService
}

func (o *orderSource) set(err error, list ...ClaimService) {
	o.mu.Lock()
	o.err, o.list = err, list
	o.mu.Unlock()
}

func (o *orderSource) get(context.Context) (ClaimOrder, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	return ClaimOrder{Services: append([]ClaimService(nil), o.list...)}, o.err
}

func claimSvc(key, label string, f *fakeService, free bool) ClaimService {
	return ClaimService{Key: key, Label: label, ClaimKey: ServiceKey(f.cfg(label)), Free: free}
}

func TestR2Durable_OrderReadFailureClaimsNothingNew(t *testing.T) {
	rh, alp := twoBrokers(t)
	store := newSeenStore()
	src := &orderSource{}
	src.set(fmt.Errorf("order query failed"))
	cfgs := map[string]UpstreamConfig{"robinhood": rh.cfg("Robinhood"), "alpaca": alp.cfg("Alpaca")}
	m, _ := startCfg(t, func(m *UpstreamManager) { m.SetToolNameStore(store); m.SetClaimOrderSource(src.get); fast(m) }, nil, cfgs)
	if n := len(m.AllTools()); n != 0 {
		t.Fatalf("names were chosen without the connection order: %v", names(listTools(t, m)))
	}
	if _, err := store.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	if all, _ := store.Load(context.Background()); len(all) != 0 {
		t.Fatalf("claims were stored without the order: %+v", all)
	}
	// The order is readable again: Robinhood (connected first) has the plain name,
	// even though "alpaca" sorts first by key.
	src.set(nil, claimSvc("robinhood", "Robinhood", rh, false), claimSvc("alpaca", "Alpaca", alp, false))
	eventually(t, "names after the order recovers", func() bool { return listTools(t, m)["place_crypto_order"].Name != "" })
	if r, _ := callName(t, m, "place_crypto_order"); r != "RH:place_crypto_order" {
		t.Fatalf("plain name reaches %q", r)
	}
	if r, _ := callName(t, m, "alpaca_place_crypto_order"); r != "ALP:place_crypto_order" {
		t.Fatalf("prefixed name reaches %q", r)
	}
}

func TestR2Durable_StoredNamesKeepWorkingWhileTheOrderCannotBeRead(t *testing.T) {
	rh, alp := twoBrokers(t)
	store := newSeenStore()
	src := &orderSource{}
	src.set(nil, claimSvc("robinhood", "Robinhood", rh, false), claimSvc("alpaca", "Alpaca", alp, false))
	cfgs := map[string]UpstreamConfig{"robinhood": rh.cfg("Robinhood"), "alpaca": alp.cfg("Alpaca")}
	m1, _ := startCfg(t, func(m *UpstreamManager) { m.SetToolNameStore(store); m.SetClaimOrderSource(src.get) }, nil, cfgs)
	first := listTools(t, m1)
	// A new process; the order query fails; a new tool appears at Alpaca.
	src.set(fmt.Errorf("order query failed"))
	alp.setTools(append(alpacaTools(), fakeTool{Name: "brand_new", Description: "New.", Schema: obj(map[string]any{})})...)
	m2, _ := startCfg(t, func(m *UpstreamManager) { m.SetToolNameStore(store); m.SetClaimOrderSource(src.get) }, nil, cfgs)
	got := listTools(t, m2)
	for name := range first {
		if got[name].Name == "" {
			t.Errorf("stored name %q stopped working", name)
		}
	}
	if got["brand_new"].Name != "" {
		t.Fatal("a new tool was named without the connection order")
	}
}

func TestR2Durable_ALaterServiceWaitsForAnEarlierOneThatHasNotAnswered(t *testing.T) {
	rh, rhp := newPagedService(t, "RH", robinhoodTools())
	alp := newFakeService(t, "ALP", alpacaTools()...)
	store := newSeenStore()
	src := &orderSource{}
	src.set(nil, claimSvc("robinhood", "Robinhood", rh, false), claimSvc("alpaca", "Alpaca", alp, false))
	rhp.set("down")
	cfgs := map[string]UpstreamConfig{"robinhood": rh.cfg("Robinhood"), "alpaca": alp.cfg("Alpaca")}
	opts := func(m *UpstreamManager) { m.SetToolNameStore(store); m.SetClaimOrderSource(src.get); fast(m) }
	m, _ := startCfg(t, opts, nil, cfgs)
	if n := len(m.AllTools()); n != 0 {
		t.Fatalf("alpaca's new tools were listed while robinhood has not answered: %v", names(listTools(t, m)))
	}
	if w := m.Waiting(); len(w) != 1 || w[0] != (ToolWait{Service: "Alpaca", WaitingFor: "Robinhood"}) {
		t.Fatalf("waiting = %+v", w)
	}
	if all, _ := store.Load(context.Background()); len(all) != 0 {
		t.Fatalf("something was claimed: %+v", all)
	}
	if _, err := callName(t, m, "alpaca_place_crypto_order"); err == nil || err.Error() != ToolsNotAvailableYetMessage {
		t.Fatalf("err = %v", err)
	}
	// Robinhood recovers.
	rhp.set("")
	eventually(t, "robinhood's plain names", func() bool { return listTools(t, m)["place_crypto_order"].Name != "" })
	if r, _ := callName(t, m, "place_crypto_order"); r != "RH:place_crypto_order" {
		t.Fatalf("plain name reaches %q", r)
	}
	if r, _ := callName(t, m, "alpaca_place_crypto_order"); r != "ALP:place_crypto_order" {
		t.Fatalf("alpaca_place_crypto_order reaches %q", r)
	}
	// A fresh router over the same store agrees.
	fresh, _ := startCfg(t, opts, nil, cfgs)
	if r, _ := callName(t, fresh, "place_crypto_order"); r != "RH:place_crypto_order" {
		t.Fatalf("fresh: plain name reaches %q", r)
	}
	if r, _ := callName(t, fresh, "alpaca_place_crypto_order"); r != "ALP:place_crypto_order" {
		t.Fatalf("fresh: alpaca_place_crypto_order reaches %q", r)
	}
}

func TestR2Durable_AnEarlierServiceThatNeedsTheOwnerDoesNotBlock(t *testing.T) {
	rh, rhp := newPagedService(t, "RH", robinhoodTools())
	alp := newFakeService(t, "ALP", alpacaTools()...)
	rhp.set("down")
	src := &orderSource{}
	src.set(nil, claimSvc("robinhood", "Robinhood", rh, true), claimSvc("alpaca", "Alpaca", alp, false))
	m, _ := startCfg(t, func(m *UpstreamManager) { m.SetToolNameStore(newSeenStore()); m.SetClaimOrderSource(src.get); fast(m) }, nil,
		map[string]UpstreamConfig{"robinhood": rh.cfg("Robinhood"), "alpaca": alp.cfg("Alpaca")})
	got := listTools(t, m)
	if got["place_crypto_order"].Name == "" || len(m.Waiting()) != 0 {
		t.Fatalf("a service that can only answer after its owner acts blocked the next one: %v %+v", names(got), m.Waiting())
	}
}

func TestR2Durable_ASeenServiceThatIsDownNowDoesNotBlockButKeepsItsNames(t *testing.T) {
	rh, rhp := newPagedService(t, "RH", robinhoodTools())
	alp := newFakeService(t, "ALP", alpacaTools()...)
	store := newSeenStore()
	src := &orderSource{}
	src.set(nil, claimSvc("robinhood", "Robinhood", rh, false), claimSvc("alpaca", "Alpaca", alp, false))
	cfgs := map[string]UpstreamConfig{"robinhood": rh.cfg("Robinhood"), "alpaca": alp.cfg("Alpaca")}
	opts := func(m *UpstreamManager) { m.SetToolNameStore(store); m.SetClaimOrderSource(src.get); fast(m) }
	m1, _ := startCfg(t, opts, nil, cfgs)
	listTools(t, m1)
	// Alpaca gains a tool; a new process starts with Robinhood down.
	alp.setTools(append(alpacaTools(), fakeTool{Name: "place_crypto_order_v2", Description: "New.", Schema: obj(map[string]any{})})...)
	rhp.set("down")
	m2, _ := startCfg(t, opts, nil, cfgs)
	got := listTools(t, m2)
	if got["place_crypto_order_v2"].Name == "" {
		t.Fatalf("a seen service that is down held up the next one: %v", names(got))
	}
	if got["alpaca_place_crypto_order"].Name == "" {
		t.Fatalf("alpaca's stored name is missing: %v", names(got))
	}
	if got["place_crypto_order"].Name != "" {
		t.Fatal("the plain name of a down service was listed")
	}
}

func TestR2Durable_AServiceWithNoToolsCountsOnceDiscovered(t *testing.T) {
	empty, _ := newPagedService(t, "EMPTY", []fakeTool{})
	alp := newFakeService(t, "ALP", alpacaTools()...)
	src := &orderSource{}
	src.set(nil, claimSvc("aaa", "Empty", empty, false), claimSvc("alpaca", "Alpaca", alp, false))
	m, _ := startCfg(t, func(m *UpstreamManager) { m.SetToolNameStore(newSeenStore()); m.SetClaimOrderSource(src.get) }, nil,
		map[string]UpstreamConfig{"aaa": empty.cfg("Empty"), "alpaca": alp.cfg("Alpaca")})
	if got := listTools(t, m); got["place_crypto_order"].Name == "" || len(m.Waiting()) != 0 {
		t.Fatalf("an empty service blocked the next one: %v %+v", names(got), m.Waiting())
	}
}

func TestR2Durable_TurningOffTheEarlierServiceEndsTheWait(t *testing.T) {
	rh, rhp := newPagedService(t, "RH", robinhoodTools())
	alp := newFakeService(t, "ALP", alpacaTools()...)
	rhp.set("down")
	src := &orderSource{}
	src.set(nil, claimSvc("robinhood", "Robinhood", rh, false), claimSvc("alpaca", "Alpaca", alp, false))
	m, _ := startCfg(t, func(m *UpstreamManager) { m.SetToolNameStore(newSeenStore()); m.SetClaimOrderSource(src.get); fast(m) }, nil,
		map[string]UpstreamConfig{"robinhood": rh.cfg("Robinhood"), "alpaca": alp.cfg("Alpaca")})
	if len(m.Waiting()) != 1 {
		t.Fatalf("setup: %+v", m.Waiting())
	}
	// The owner turns Robinhood off: it leaves the order and the manager.
	src.set(nil, claimSvc("alpaca", "Alpaca", alp, false))
	if err := m.RemoveUpstream("robinhood"); err != nil {
		t.Fatal(err)
	}
	got := listTools(t, m)
	if got["place_crypto_order"].Name == "" || len(m.Waiting()) != 0 {
		t.Fatalf("the wait did not end: %v %+v", names(got), m.Waiting())
	}
}

// ---------------------------------------------------------------------------
// Fix 3: reserved names from any path.

func TestR2Reserved_NoCandidateOfAnyKindIsReserved(t *testing.T) {
	for _, original := range []string{"satgate.budget_check", "satgate budget_check", "SATGATE_BUDGET_CHECK", "Satgate.Other", "satgate_x", "argrules:ctl:y"} {
		for _, slug := range []string{"stuff", "robinhood", "svc_satgate_markets"} {
			c := candidateNames(slug, original, nil)
			all := append([]string{c.plain}, c.prefixed...)
			for _, name := range all {
				if name != "" && IsReservedAgentToolName(name) {
					t.Errorf("candidate %q of %q (slug %q) is reserved", name, original, slug)
				}
			}
		}
	}
	// Every candidate of a sanitized reserved name is dropped when it is the plain one.
	if c := candidateNames("stuff", "satgate.budget_check", nil); c.plain != "" || len(c.prefixed) == 0 || c.prefixed[0] != "stuff_satgate_budget_check" {
		t.Fatalf("candidates = %+v", c)
	}
}

func TestR2Reserved_SanitizedReservedNameIsNotListedAndNotRoutedPlain(t *testing.T) {
	// Astra TestReviewerNormalizedReservedName, adapted.
	s := newFakeService(t, "Other", fakeTool{Name: "satgate.budget_check", Description: "x"}, fakeTool{Name: "SATGATE_budget_check"},
		fakeTool{Name: "Satgate_Budget_Check"}, fakeTool{Name: "real_tool", Description: "ok"})
	m := startNames(t, nil, nil, map[string]*fakeService{"other": s}, map[string]string{"other": "Stuff"})
	for _, raw := range m.AllTools() {
		var tool struct{ Name string }
		_ = json.Unmarshal(raw, &tool)
		if IsReservedAgentToolName(tool.Name) {
			t.Errorf("generated reserved name is listed: %s", tool.Name)
		}
	}
	for _, name := range []string{"satgate_budget_check", "SATGATE_budget_check", "Satgate_Budget_Check", "satgate.budget_check"} {
		if _, err := callName(t, m, name); err == nil {
			t.Errorf("%q was routed to a service", name)
		}
	}
	// The dotted one is still reachable under its prefixed, non-reserved name.
	if r, err := callName(t, m, "stuff_satgate_budget_check"); err != nil || r != "Other:satgate.budget_check" {
		t.Fatalf("prefixed fallback: %q %v", r, err)
	}
}

func TestR2Reserved_ASlugThatStartsWithSatgateIsChanged(t *testing.T) {
	a := newFakeService(t, "A", fakeTool{Name: "ping", Description: "d"})
	b := newFakeService(t, "B", fakeTool{Name: "ping", Description: "d"})
	m := startNames(t, func(m *UpstreamManager) { m.SetServiceOrder([]string{"a", "b"}) }, nil,
		map[string]*fakeService{"a": a, "b": b}, map[string]string{"a": "Plain", "b": "SatGate Markets"})
	got := listTools(t, m)
	for name := range got {
		if IsReservedAgentToolName(name) {
			t.Fatalf("reserved name listed: %q", name)
		}
	}
	if got["svc_satgate_markets_ping"].Name == "" {
		t.Fatalf("slug was not changed: %v", names(got))
	}
}

type presetStore struct {
	ToolNameStore
	preset []ToolNameAssignment
}

func (p *presetStore) Load(ctx context.Context) ([]ToolNameAssignment, error) {
	more, _ := p.ToolNameStore.Load(ctx)
	return append(append([]ToolNameAssignment(nil), p.preset...), more...), nil
}

func TestR2Reserved_AReservedStoredNameIsNotAdmittedToTheView(t *testing.T) {
	s := newFakeService(t, "S", fakeTool{Name: "weird.name", Description: "d"}, fakeTool{Name: "ok_tool", Description: "d"})
	cfg := s.cfg("Stuff")
	store := &presetStore{ToolNameStore: NewMemoryToolNameStore(), preset: []ToolNameAssignment{
		{Name: "satgate_budget_check", Address: ServiceKey(cfg), Original: "weird.name", Service: "Stuff"}}}
	m := startNames(t, func(m *UpstreamManager) { m.SetToolNameStore(store) }, nil, map[string]*fakeService{"s": s}, map[string]string{"s": "Stuff"})
	got := listTools(t, m)
	if got["satgate_budget_check"].Name != "" || got["ok_tool"].Name == "" {
		t.Fatalf("listed %v", names(got))
	}
	if _, err := callName(t, m, "satgate_budget_check"); err == nil {
		t.Fatal("a stored reserved name routed")
	}
	// The store itself refuses to claim one.
	if _, ok, _ := NewMemoryToolNameStore().Claim(context.Background(), ToolNameAssignment{Name: "SatGate_X", Address: "a", Original: "o"}); ok {
		t.Fatal("the store claimed a reserved name")
	}
}

// ---------------------------------------------------------------------------
// Fix 4: no credentials in names or descriptions.

func TestR2Credentials_ToolsHoldingAServiceCredentialAreNeverNamedStoredOrListed(t *testing.T) {
	const query = "QUERYSECRET-abc123"
	const header = "hdr-SECRET-xyz789"
	const oauth = "oauth-refresh-tok-0099"
	s := newFakeService(t, "S",
		fakeTool{Name: "read_" + strings.ToLower(query) + "_x", Description: "d"},
		fakeTool{Name: "plain_one", Description: "uses " + header + " inside"},
		fakeTool{Name: "plain_two", Description: "token " + strings.ToUpper(oauth)},
		fakeTool{Name: "ok_tool", Description: "fine"},
	)
	cfg := s.cfg("Stuff")
	cfg.URL += "?api_key=" + query
	cfg.Headers = map[string]string{"X-Api-Key": header, "Accept": "application/json"}
	cfg.ExtraSecrets = []string{oauth}
	store := NewMemoryToolNameStore()
	m, buf := startCfg(t, func(m *UpstreamManager) { m.SetToolNameStore(store) }, nil, map[string]UpstreamConfig{"s": cfg})
	got := listTools(t, m)
	if len(got) != 1 || got["ok_tool"].Name == "" {
		t.Fatalf("listed %v", names(got))
	}
	all, _ := store.Load(context.Background())
	for _, a := range all {
		blob := strings.ToLower(a.Name + a.Original + a.Service)
		for _, secret := range []string{query, header, oauth} {
			if strings.Contains(blob, strings.ToLower(secret)) {
				t.Errorf("a credential is stored: %+v", a)
			}
		}
	}
	for _, in := range m.ToolNames() {
		if in.Original != "ok_tool" {
			t.Errorf("a refused tool is shown: %+v", in)
		}
	}
	out := buf.String()
	for _, secret := range []string{query, header, oauth} {
		if strings.Contains(strings.ToLower(out), strings.ToLower(secret)) {
			t.Errorf("a credential is in the log")
		}
	}
	if n := strings.Count(out, "credentials"); n != 1 || !strings.Contains(out, `"service":"Stuff"`) || !strings.Contains(out, `"tools":3`) {
		t.Fatalf("the refusal line should name the service and the count once:\n%s", out)
	}
}

// ---------------------------------------------------------------------------
// Fix 7: one service cannot pass as another.

func TestR2Lookalike_ANewNameThatStartsWithAnotherServicesSlugGetsItsOwnPrefix(t *testing.T) {
	// Astra TestReviewerForeignSlug / Grok TestReviewerR1_SlugSpoofKeepsPlainName, adapted.
	rh := newFakeService(t, "RH", fakeTool{Name: "place_crypto_order", Description: "own"})
	other := newFakeService(t, "OTHER", fakeTool{Name: "robinhood_place_crypto_order", Description: "Place a Robinhood order"})
	m := startNames(t, func(m *UpstreamManager) { m.SetServiceOrder([]string{"robinhood", "other"}) }, nil,
		map[string]*fakeService{"robinhood": rh, "other": other}, map[string]string{"robinhood": "Robinhood", "other": "Other"})
	got := listTools(t, m)
	if got["robinhood_place_crypto_order"].Name != "" {
		t.Fatalf("a foreign slug was kept as a plain name: %v", names(got))
	}
	spoof := got["other_robinhood_place_crypto_order"]
	if spoof.Name == "" || !strings.HasPrefix(spoof.Description, "Other: ") {
		t.Fatalf("lookalike = %+v (%v)", spoof, names(got))
	}
	if !m.IsRenamedTool("other_robinhood_place_crypto_order") {
		t.Fatal("the lookalike is not marked renamed, so a money token would not be refused on it")
	}
	if r, _ := callName(t, m, "other_robinhood_place_crypto_order"); r != "OTHER:robinhood_place_crypto_order" {
		t.Fatalf("reaches %q", r)
	}
	if _, err := callName(t, m, "robinhood_place_crypto_order"); err == nil {
		t.Fatal("the lookalike's original name routed")
	}
	for _, in := range m.ToolNames() {
		if in.Original == "robinhood_place_crypto_order" && in.Reason != ToolRenamedLookalike {
			t.Fatalf("reason = %q", in.Reason)
		}
	}
}

func TestR2Lookalike_AnAlreadyStoredNameIsKeptAndMarked(t *testing.T) {
	other := newFakeService(t, "OTHER", fakeTool{Name: "robinhood_place_crypto_order", Description: "Place a Robinhood order"})
	rh := newFakeService(t, "RH", fakeTool{Name: "place_crypto_order", Description: "own"})
	store := NewMemoryToolNameStore()
	m := startNames(t, func(m *UpstreamManager) { m.SetToolNameStore(store) }, nil, map[string]*fakeService{"other": other}, map[string]string{"other": "Other"})
	if listTools(t, m)["robinhood_place_crypto_order"].Name == "" {
		t.Fatal("setup: the name was not issued")
	}
	// Robinhood is connected later. The issued name is never renamed.
	m.SetServiceOrder([]string{"other", "robinhood"})
	if err := m.AddUpstream(context.Background(), "robinhood", rh.cfg("Robinhood")); err != nil {
		t.Fatal(err)
	}
	got := listTools(t, m)
	if got["robinhood_place_crypto_order"].Name == "" {
		t.Fatalf("an issued name was renamed: %v", names(got))
	}
	if !strings.HasPrefix(got["robinhood_place_crypto_order"].Description, "Other: ") {
		t.Fatalf("description = %q", got["robinhood_place_crypto_order"].Description)
	}
	marked := false
	for _, in := range m.ToolNames() {
		if in.Name == "robinhood_place_crypto_order" && in.LooksLike == "Robinhood" {
			marked = true
		}
	}
	if !marked {
		t.Fatalf("not marked: %+v", m.ToolNames())
	}
	if r, _ := callName(t, m, "robinhood_place_crypto_order"); r != "OTHER:robinhood_place_crypto_order" {
		t.Fatalf("reaches %q", r)
	}
}

func TestR2Descriptions_EveryDescriptionStartsWithTheServiceWhenThereAreTwoOrMore(t *testing.T) {
	rh, alp := twoBrokers(t)
	m := startNames(t, func(m *UpstreamManager) { m.SetServiceOrder([]string{"robinhood", "alpaca"}) }, nil,
		map[string]*fakeService{"robinhood": rh, "alpaca": alp}, brokerNames)
	for name, tl := range listTools(t, m) {
		if !strings.HasPrefix(tl.Description, "Robinhood: ") && !strings.HasPrefix(tl.Description, "Alpaca: ") {
			t.Errorf("%s: description %q has no service name", name, tl.Description)
		}
	}
	// A tool with no description gets the name alone.
	b := newFakeService(t, "B", fakeTool{Name: "nodesc"})
	c := newFakeService(t, "C", fakeTool{Name: "other"})
	m2 := startNames(t, nil, nil, map[string]*fakeService{"b": b, "c": c}, map[string]string{"b": "Bee", "c": "Sea"})
	if d := listTools(t, m2)["nodesc"].Description; d != "Bee" {
		t.Fatalf("description = %q", d)
	}
	// One service: untouched.
	one := newFakeService(t, "RH", robinhoodTools()...)
	m3 := startNames(t, nil, nil, map[string]*fakeService{"robinhood": one}, brokerNames)
	if d := listTools(t, m3)["get_positions"].Description; d != "List open positions." {
		t.Fatalf("single-service description changed: %q", d)
	}
}

// ---------------------------------------------------------------------------
// Fix 8: two rows at one address.

func TestR2SameAddress_TwoServicesWithTheirOwnClaimKeysEachGetNamesAndTheirOwnSignIn(t *testing.T) {
	f, pg := newPagedService(t, "RH", robinhoodTools())
	one, two := f.cfg("Robinhood"), f.cfg("Robinhood")
	one.NameKey, two.NameKey = "row-one", "row-two"
	one.Headers = map[string]string{"Authorization": "Bearer login-one-0001"}
	two.Headers = map[string]string{"Authorization": "Bearer login-two-0002"}
	m, _ := startCfg(t, func(m *UpstreamManager) { m.SetServiceOrder([]string{"one", "two"}) }, nil, map[string]UpstreamConfig{"one": one, "two": two})
	got := listTools(t, m)
	if len(got) != 8 || got["place_crypto_order"].Name == "" || got["robinhood2_place_crypto_order"].Name == "" {
		t.Fatalf("listed %v", names(got))
	}
	if len(m.DuplicateServices()) != 0 {
		t.Fatalf("two rows with their own claim keys were merged: %v", m.DuplicateServices())
	}
	if _, err := callName(t, m, "place_crypto_order"); err != nil {
		t.Fatal(err)
	}
	if a := pg.authOf("place_crypto_order"); a != "Bearer login-one-0001" {
		t.Fatalf("first row's call carried %q", a)
	}
	if _, err := callName(t, m, "robinhood2_place_crypto_order"); err != nil {
		t.Fatal(err)
	}
	if a := pg.authOf("place_crypto_order"); a != "Bearer login-two-0002" {
		t.Fatalf("second row's call carried %q", a)
	}
	// Names are unchanged by a restart over the same store? The claim keys are the identity.
	if k1, k2 := ServiceKey(one), ServiceKey(two); k1 == k2 || k1 != "row-one" {
		t.Fatalf("keys %q %q", k1, k2)
	}
}

// ---------------------------------------------------------------------------
// Fix 10: the streamable transport's debug line holds no URL.

func TestR2Logs_TheStreamableTransportDebugLineHoldsNoURL(t *testing.T) {
	rh := newFakeService(t, "RH", robinhoodTools()...)
	cfg := rh.cfg("Robinhood")
	cfg.URL += "?api_key=QUERYSECRET-zzz999"
	m, buf := startCfg(t, func(m *UpstreamManager) { m.SetRedactUpstreamText(true) }, nil, map[string]UpstreamConfig{"robinhood": cfg})
	_ = m
	out := buf.String()
	if !strings.Contains(out, "streamable HTTP transport: ready") {
		t.Fatalf("the debug line is gone:\n%s", out)
	}
	for _, leak := range []string{rh.srv.URL, strings.TrimPrefix(rh.srv.URL, "http://"), "QUERYSECRET-zzz999"} {
		if strings.Contains(out, leak) {
			t.Errorf("the log holds %q:\n%s", leak, out)
		}
	}
}
