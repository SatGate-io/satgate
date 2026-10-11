package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/satgate-io/satgate/pkg/toolcontrols"
)

// ---------------------------------------------------------------------------
// Fakes. No network beyond loopback, no vendor. The schemas are shaped like the
// real ones: Robinhood's order tool takes dollar_amount or quantity; Alpaca's
// official MCP server (github.com/alpacahq/alpaca-mcp-server, src/alpaca_mcp_server/
// overrides.py) takes qty or notional and names its stock tool place_stock_order.

type fakeTool struct {
	Name        string
	Description string
	Schema      map[string]any
}

type fakeService struct {
	label string
	srv   *httptest.Server

	mu    sync.Mutex
	tools []fakeTool
	calls map[string]int // tool name as received -> count
	args  []string
}

func newFakeService(t *testing.T, label string, tools ...fakeTool) *fakeService {
	t.Helper()
	f := &fakeService{label: label, tools: tools, calls: map[string]int{}}
	f.srv = httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeService) serve(w http.ResponseWriter, r *http.Request) {
	var env struct {
		Method string          `json:"method"`
		ID     json.RawMessage `json:"id"`
		Params struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		} `json:"params"`
	}
	_ = json.NewDecoder(r.Body).Decode(&env)
	w.Header().Set("Content-Type", "application/json")
	switch env.Method {
	case "initialize":
		fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":{}}`, env.ID)
	case "tools/list":
		f.mu.Lock()
		var parts []string
		for _, tl := range f.tools {
			schema := tl.Schema
			if schema == nil {
				schema = map[string]any{"type": "object", "properties": map[string]any{}}
			}
			b, _ := json.Marshal(map[string]any{"name": tl.Name, "description": tl.Description, "inputSchema": schema})
			parts = append(parts, string(b))
		}
		f.mu.Unlock()
		fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":{"tools":[%s]}}`, env.ID, strings.Join(parts, ","))
	case "tools/call":
		f.mu.Lock()
		f.calls[env.Params.Name]++
		f.args = append(f.args, string(env.Params.Arguments))
		f.mu.Unlock()
		fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":{"content":[{"type":"text","text":%q}]}}`, env.ID, f.label+":"+env.Params.Name)
	default:
		w.WriteHeader(http.StatusAccepted)
	}
}

func (f *fakeService) setTools(tools ...fakeTool) {
	f.mu.Lock()
	f.tools = tools
	f.mu.Unlock()
}

func (f *fakeService) calledWith(name string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls[name]
}

func (f *fakeService) totalCalls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, c := range f.calls {
		n += c
	}
	return n
}

func (f *fakeService) cfg(display string) UpstreamConfig {
	return UpstreamConfig{Transport: "streamable", URL: f.srv.URL + "/mcp", Timeout: 5 * time.Second, DisplayName: display}
}

func obj(props map[string]any, required ...string) map[string]any {
	s := map[string]any{"type": "object", "properties": props}
	if len(required) > 0 {
		s["required"] = required
	}
	return s
}

func str(desc string) map[string]any { return map[string]any{"type": "string", "description": desc} }

func robinhoodTools() []fakeTool {
	return []fakeTool{
		{"get_account", "Get the brokerage account.", obj(map[string]any{})},
		{"get_positions", "List open positions.", obj(map[string]any{})},
		{"place_crypto_order", "Place a crypto order for a dollar amount or a quantity.", obj(map[string]any{
			"symbol": str("Pair, e.g. BTC-USD"), "side": str("buy or sell"), "type": str("market or limit"),
			"dollar_amount": str("Amount in US dollars"), "quantity": str("Amount of the coin"),
		}, "symbol", "side")},
		{"cancel_order", "Cancel an open order.", obj(map[string]any{"order_id": str("Order id")}, "order_id")},
	}
}

func alpacaTools() []fakeTool {
	return []fakeTool{
		{"get_account", "Get the trading account.", obj(map[string]any{})},
		{"get_all_positions", "List all positions.", obj(map[string]any{})},
		{"place_crypto_order", "Place a cryptocurrency order.", obj(map[string]any{
			"symbol": str("Crypto pair, e.g. BTC/USD"), "side": str("buy or sell"), "qty": str("Number of coins"),
			"notional": str("Dollar amount to trade"), "type": str("market, limit or stop_limit"), "time_in_force": str("gtc or ioc"),
		}, "symbol", "side")},
		{"place_stock_order", "Place a stock order.", obj(map[string]any{
			"symbol": str("Ticker"), "side": str("buy or sell"), "qty": str("Shares"), "notional": str("Dollar amount"),
		}, "symbol", "side")},
		{"cancel_order_by_id", "Cancel one order.", obj(map[string]any{"order_id": str("Order id")}, "order_id")},
	}
}

func coinbaseTools() []fakeTool {
	return []fakeTool{
		{"get_account", "Get a Coinbase account.", obj(map[string]any{})},
		{"place_crypto_order", "Create an order.", obj(map[string]any{"product_id": str("Product"), "quote_size": str("Quote amount")})},
		{"list_products", "List products.", obj(map[string]any{})},
	}
}

func startNames(t *testing.T, opts func(*UpstreamManager), routing []RoutingRule, svcs map[string]*fakeService, display map[string]string) *UpstreamManager {
	t.Helper()
	captureLogs(t)
	cfg := map[string]UpstreamConfig{}
	for key, f := range svcs {
		cfg[key] = f.cfg(display[key])
	}
	m := NewUpstreamManager(cfg, routing, "", true)
	if opts != nil {
		opts(m)
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() { cancel(); m.Close() })
	if err := m.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	return m
}

type listedTool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func listTools(t *testing.T, m *UpstreamManager) map[string]listedTool {
	t.Helper()
	out := map[string]listedTool{}
	seen := map[string]bool{}
	for _, raw := range m.AllTools() {
		var tl listedTool
		if err := json.Unmarshal(raw, &tl); err != nil {
			t.Fatal(err)
		}
		if seen[strings.ToLower(tl.Name)] {
			t.Fatalf("tool name %q is listed twice (compared without case)", tl.Name)
		}
		seen[strings.ToLower(tl.Name)] = true
		out[tl.Name] = tl
	}
	return out
}

func callName(t *testing.T, m *UpstreamManager, name string) (string, error) {
	t.Helper()
	params, _ := json.Marshal(map[string]any{"name": name, "arguments": map[string]any{}})
	resp, err := m.ForwardToolCall(context.Background(), name, params, 5*time.Second)
	if err != nil {
		return "", err
	}
	if resp.Error != nil {
		return "", fmt.Errorf("rpc error %d", resp.Error.Code)
	}
	var r struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(resp.Result, &r); err != nil || len(r.Content) == 0 {
		t.Fatalf("bad result %s", resp.Result)
	}
	return r.Content[0].Text, nil
}

func twoBrokers(t *testing.T) (*fakeService, *fakeService) {
	return newFakeService(t, "RH", robinhoodTools()...), newFakeService(t, "ALP", alpacaTools()...)
}

var brokerNames = map[string]string{"robinhood": "Robinhood", "alpaca": "Alpaca", "coinbase": "Coinbase"}

// ---------------------------------------------------------------------------

func TestToolNames_BothOrderToolsListedOnceUnderDifferentNames(t *testing.T) {
	rh, alp := twoBrokers(t)
	m := startNames(t, func(m *UpstreamManager) { m.SetServiceOrder([]string{"robinhood", "alpaca"}) }, nil,
		map[string]*fakeService{"robinhood": rh, "alpaca": alp}, brokerNames)
	got := listTools(t, m)
	for _, want := range []string{"place_crypto_order", "alpaca_place_crypto_order", "place_stock_order", "get_account", "alpaca_get_account", "cancel_order", "cancel_order_by_id"} {
		if _, ok := got[want]; !ok {
			t.Errorf("missing %q in %v", want, names(got))
		}
	}
	if n := len(got); n != 4+5 {
		t.Errorf("listed %d tools, want 9: %v", n, names(got))
	}
	// Both copies say which service they are, because the name clashes.
	if d := got["place_crypto_order"].Description; !strings.HasPrefix(d, "Robinhood: ") {
		t.Errorf("plain copy description = %q", d)
	}
	if d := got["alpaca_place_crypto_order"].Description; !strings.HasPrefix(d, "Alpaca: ") {
		t.Errorf("prefixed copy description = %q", d)
	}
	// Round 2 (fix 7): with two or more services every description starts with
	// the service's name, clash or not.
	if d := got["place_stock_order"].Description; d != "Alpaca: Place a stock order." {
		t.Errorf("place_stock_order description = %q", d)
	}
	if d := got["get_positions"].Description; d != "Robinhood: List open positions." {
		t.Errorf("get_positions description = %q", d)
	}
}

func names(m map[string]listedTool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func TestToolNames_ThousandCallsToEachNameReachTheRightService(t *testing.T) {
	rh, alp := twoBrokers(t)
	m := startNames(t, func(m *UpstreamManager) { m.SetServiceOrder([]string{"robinhood", "alpaca"}) }, nil,
		map[string]*fakeService{"robinhood": rh, "alpaca": alp}, brokerNames)
	for i := 0; i < 1000; i++ {
		if got, err := callName(t, m, "place_crypto_order"); err != nil || got != "RH:place_crypto_order" {
			t.Fatalf("call %d to place_crypto_order: %q %v", i, got, err)
		}
		if got, err := callName(t, m, "alpaca_place_crypto_order"); err != nil || got != "ALP:place_crypto_order" {
			t.Fatalf("call %d to alpaca_place_crypto_order: %q %v", i, got, err)
		}
	}
	if rh.calledWith("place_crypto_order") != 1000 || alp.calledWith("place_crypto_order") != 1000 {
		t.Fatalf("rh=%d alp=%d", rh.calledWith("place_crypto_order"), alp.calledWith("place_crypto_order"))
	}
	// The service saw its own tool name, never the one the agent used.
	if alp.calledWith("alpaca_place_crypto_order") != 0 || alp.totalCalls() != 1000 || rh.totalCalls() != 1000 {
		t.Fatalf("alpaca saw %v", alp.calls)
	}
}

func TestToolNames_ArgumentsAreForwardedUntouched(t *testing.T) {
	rh, alp := twoBrokers(t)
	m := startNames(t, func(m *UpstreamManager) { m.SetServiceOrder([]string{"robinhood", "alpaca"}) }, nil,
		map[string]*fakeService{"robinhood": rh, "alpaca": alp}, brokerNames)
	params := json.RawMessage(`{"name":"alpaca_place_crypto_order","arguments":{"symbol":"BTC/USD","side":"buy","notional":"25.50"},"_meta":{"progressToken":7}}`)
	if _, err := m.ForwardToolCall(context.Background(), "alpaca_place_crypto_order", params, 5*time.Second); err != nil {
		t.Fatal(err)
	}
	alp.mu.Lock()
	defer alp.mu.Unlock()
	if len(alp.args) != 1 || alp.args[0] != `{"symbol":"BTC/USD","side":"buy","notional":"25.50"}` {
		t.Fatalf("alpaca received arguments %v", alp.args)
	}
}

func TestToolNames_AddingAThirdServiceChangesNoExistingName(t *testing.T) {
	rh, alp := twoBrokers(t)
	cb := newFakeService(t, "CB", coinbaseTools()...)
	m := startNames(t, func(m *UpstreamManager) { m.SetServiceOrder([]string{"robinhood", "alpaca"}) }, nil,
		map[string]*fakeService{"robinhood": rh, "alpaca": alp}, brokerNames)
	before := listTools(t, m)
	// Coinbase sorts before the others by key; it must still not take a name.
	m.SetServiceOrder([]string{"coinbase", "robinhood", "alpaca"})
	if err := m.AddUpstream(context.Background(), "coinbase", cb.cfg("Coinbase")); err != nil {
		t.Fatal(err)
	}
	after := listTools(t, m)
	for name := range before {
		if _, ok := after[name]; !ok {
			t.Errorf("name %q disappeared when a service was added", name)
		}
	}
	for _, want := range []string{"coinbase_get_account", "coinbase_place_crypto_order", "list_products"} {
		if _, ok := after[want]; !ok {
			t.Errorf("missing %q: %v", want, names(after))
		}
	}
	if got, _ := callName(t, m, "place_crypto_order"); got != "RH:place_crypto_order" {
		t.Errorf("place_crypto_order now reaches %q", got)
	}
	if got, _ := callName(t, m, "alpaca_place_crypto_order"); got != "ALP:place_crypto_order" {
		t.Errorf("alpaca_place_crypto_order now reaches %q", got)
	}
	if got, _ := callName(t, m, "coinbase_place_crypto_order"); got != "CB:place_crypto_order" {
		t.Errorf("coinbase_place_crypto_order reaches %q", got)
	}
}

func TestToolNames_RemovedServiceKeepsItsNamesAndReAddingRestoresThem(t *testing.T) {
	rh, alp := twoBrokers(t)
	m := startNames(t, func(m *UpstreamManager) { m.SetServiceOrder([]string{"robinhood", "alpaca"}) }, nil,
		map[string]*fakeService{"robinhood": rh, "alpaca": alp}, brokerNames)
	if err := m.RemoveUpstream("robinhood"); err != nil {
		t.Fatal(err)
	}
	got := listTools(t, m)
	if _, ok := got["place_crypto_order"]; ok {
		t.Fatalf("a removed service's tool is still listed")
	}
	// Alpaca's copy keeps its prefixed name: names never change.
	if _, ok := got["alpaca_place_crypto_order"]; !ok {
		t.Fatalf("alpaca's name changed when robinhood was removed: %v", names(got))
	}
	if _, err := callName(t, m, "place_crypto_order"); err == nil {
		t.Fatal("a call to a removed service's name was sent somewhere")
	}
	// Re-connecting the same address restores the plain name.
	if err := m.AddUpstream(context.Background(), "robinhood", rh.cfg("Robinhood")); err != nil {
		t.Fatal(err)
	}
	if got, err := callName(t, m, "place_crypto_order"); err != nil || got != "RH:place_crypto_order" {
		t.Fatalf("after re-adding: %q %v", got, err)
	}
	// Re-adding under another key and display name is the same service.
	if err := m.RemoveUpstream("robinhood"); err != nil {
		t.Fatal(err)
	}
	if err := m.AddUpstream(context.Background(), "rh-renamed", rh.cfg("Robin Hood Brokerage")); err != nil {
		t.Fatal(err)
	}
	if got, err := callName(t, m, "place_crypto_order"); err != nil || got != "RH:place_crypto_order" {
		t.Fatalf("after renaming the service: %q %v", got, err)
	}
}

func TestToolNames_RemovedServicesNameIsNeverGivenToADifferentService(t *testing.T) {
	rh := newFakeService(t, "RH", robinhoodTools()...)
	other := newFakeService(t, "OTHER", coinbaseTools()...)
	m := startNames(t, nil, nil, map[string]*fakeService{"robinhood": rh}, brokerNames)
	if got, _ := callName(t, m, "place_crypto_order"); got != "RH:place_crypto_order" {
		t.Fatalf("setup: %q", got)
	}
	if err := m.RemoveUpstream("robinhood"); err != nil {
		t.Fatal(err)
	}
	// A different service (another address) offers the same tool name.
	if err := m.AddUpstream(context.Background(), "coinbase", other.cfg("Coinbase")); err != nil {
		t.Fatal(err)
	}
	got := listTools(t, m)
	if _, ok := got["place_crypto_order"]; ok {
		t.Fatalf("the removed service's plain name was given to a different service: %v", names(got))
	}
	if _, ok := got["coinbase_place_crypto_order"]; !ok {
		t.Fatalf("the new service has no prefixed name: %v", names(got))
	}
	if _, err := callName(t, m, "place_crypto_order"); err == nil {
		t.Fatal("the reserved name reaches a service")
	}
	if got, _ := callName(t, m, "coinbase_place_crypto_order"); got != "OTHER:place_crypto_order" {
		t.Fatalf("coinbase copy reaches %q", got)
	}
}

func TestToolNames_ReconnectKeepsNames(t *testing.T) {
	rh, alp := twoBrokers(t)
	m := startNames(t, func(m *UpstreamManager) { m.SetServiceOrder([]string{"robinhood", "alpaca"}) }, nil,
		map[string]*fakeService{"robinhood": rh, "alpaca": alp}, brokerNames)
	before := listTools(t, m)

	// The service offers a new tool and drops one; Robinhood is rediscovered
	// through the real reconnect path (a dead transport replaced).
	rh.setTools(append(robinhoodTools()[:3:3], fakeTool{Name: "get_quote", Description: "Quote.", Schema: obj(map[string]any{})})...)
	m.mu.Lock()
	old := m.clients["robinhood"]
	cfg := m.config["robinhood"]
	m.mu.Unlock()
	ctx := context.Background()
	if !m.reconnect(ctx, old, "reconnecting upstream", "upstream reconnected successfully", "reconnect",
		func() (*UpstreamClient, error) { return m.connect(ctx, "robinhood", cfg) }) {
		t.Fatal("reconnect did not publish")
	}
	after := listTools(t, m)
	for name := range before {
		if name == "cancel_order" {
			continue // robinhood no longer offers it
		}
		if _, ok := after[name]; !ok {
			t.Errorf("name %q lost across a reconnect", name)
		}
	}
	if _, ok := after["get_quote"]; !ok {
		t.Errorf("a new tool was not listed after rediscovery: %v", names(after))
	}
	if got, _ := callName(t, m, "alpaca_place_crypto_order"); got != "ALP:place_crypto_order" {
		t.Errorf("alpaca_place_crypto_order reaches %q", got)
	}
	if got, _ := callName(t, m, "place_crypto_order"); got != "RH:place_crypto_order" {
		t.Errorf("place_crypto_order reaches %q", got)
	}
}

func TestToolNames_AToolThatDisappearsKeepsItsNameReserved(t *testing.T) {
	rh := newFakeService(t, "RH", robinhoodTools()...)
	alp := newFakeService(t, "ALP", alpacaTools()...)
	m := startNames(t, func(m *UpstreamManager) { m.SetServiceOrder([]string{"robinhood", "alpaca"}) }, nil,
		map[string]*fakeService{"robinhood": rh, "alpaca": alp}, brokerNames)
	// Robinhood stops offering place_crypto_order and re-lists.
	rh.setTools(robinhoodTools()[:2]...)
	if err := m.AddUpstream(context.Background(), "robinhood", rh.cfg("Robinhood")); err != nil {
		t.Fatal(err)
	}
	got := listTools(t, m)
	if _, ok := got["place_crypto_order"]; ok {
		t.Fatal("a tool the service no longer offers is listed")
	}
	if _, ok := got["alpaca_place_crypto_order"]; !ok {
		t.Fatalf("alpaca's name moved: %v", names(got))
	}
}

func TestToolNames_RoutingRuleDecidesWhoKeepsThePlainName(t *testing.T) {
	rh, alp := twoBrokers(t)
	rules := []RoutingRule{{Tools: []string{"place_*"}, Upstream: "alpaca"}}
	m := startNames(t, nil, rules, map[string]*fakeService{"robinhood": rh, "alpaca": alp}, brokerNames)
	got := listTools(t, m)
	if !strings.HasPrefix(got["place_crypto_order"].Description, "Alpaca: ") {
		t.Fatalf("the routing rule did not give alpaca the plain name: %+v", got["place_crypto_order"])
	}
	if _, ok := got["robinhood_place_crypto_order"]; !ok {
		t.Fatalf("robinhood has no prefixed name: %v", names(got))
	}
	if r, _ := callName(t, m, "place_crypto_order"); r != "ALP:place_crypto_order" {
		t.Fatalf("plain name reaches %q", r)
	}
	if r, _ := callName(t, m, "robinhood_place_crypto_order"); r != "RH:place_crypto_order" {
		t.Fatalf("prefixed name reaches %q", r)
	}
	// get_account is not covered by the rule: sorted order (alpaca < robinhood).
	if got["get_account"].Description != "Alpaca: Get the trading account." {
		t.Fatalf("get_account = %+v", got["get_account"])
	}
}

func TestToolNames_WithoutOrderOrRulesTheSortedServiceNameWins(t *testing.T) {
	rh, alp := twoBrokers(t)
	// Many fresh managers: a plain name chosen by map order would go to
	// Robinhood about half the time.
	for i := 0; i < 40; i++ {
		m := startNames(t, nil, nil, map[string]*fakeService{"robinhood": rh, "alpaca": alp}, brokerNames)
		if r, _ := callName(t, m, "place_crypto_order"); r != "ALP:place_crypto_order" {
			t.Fatalf("manager %d: plain name reaches %q, want the service that sorts first", i, r)
		}
		if r, _ := callName(t, m, "robinhood_place_crypto_order"); r != "RH:place_crypto_order" {
			t.Fatalf("manager %d: robinhood_place_crypto_order reaches %q", i, r)
		}
	}
}

func TestToolNames_CaseOnlyClash(t *testing.T) {
	a := newFakeService(t, "A", fakeTool{Name: "Place_Order", Description: "A's.", Schema: obj(map[string]any{})})
	b := newFakeService(t, "B", fakeTool{Name: "place_order", Description: "B's.", Schema: obj(map[string]any{})})
	m := startNames(t, func(m *UpstreamManager) { m.SetServiceOrder([]string{"aa", "bb"}) }, nil,
		map[string]*fakeService{"aa": a, "bb": b}, map[string]string{"aa": "Aaa", "bb": "Bbb"})
	got := listTools(t, m) // fails if two names are equal without case
	if len(got) != 2 {
		t.Fatalf("listed %v", names(got))
	}
	if _, ok := got["Place_Order"]; !ok {
		t.Fatalf("first service lost its plain name: %v", names(got))
	}
	if _, ok := got["bbb_place_order"]; !ok {
		t.Fatalf("second service has no prefixed name: %v", names(got))
	}
	if r, _ := callName(t, m, "Place_Order"); r != "A:Place_Order" {
		t.Fatalf("Place_Order reaches %q", r)
	}
	if r, _ := callName(t, m, "bbb_place_order"); r != "B:place_order" {
		t.Fatalf("bbb_place_order reaches %q", r)
	}
	// The upstream's other spelling is not a listed name.
	if _, err := callName(t, m, "place_order"); err == nil {
		t.Fatal("a case variant that is not listed was routed")
	}
}

var agentNameRE = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)

func TestToolNames_DottedSpacedAndLongNamesGetValidStoredNames(t *testing.T) {
	long := strings.Repeat("x", 70)
	svc := newFakeService(t, "S",
		fakeTool{Name: "files.read", Description: "d", Schema: obj(map[string]any{})},
		fakeTool{Name: "has space", Description: "d", Schema: obj(map[string]any{})},
		fakeTool{Name: long, Description: "d", Schema: obj(map[string]any{})},
		fakeTool{Name: long + "y", Description: "d", Schema: obj(map[string]any{})},
		fakeTool{Name: "fine_name", Description: "d", Schema: obj(map[string]any{})},
	)
	m := startNames(t, nil, nil, map[string]*fakeService{"s": svc}, map[string]string{"s": "Stuff"})
	got := listTools(t, m)
	if len(got) != 5 {
		t.Fatalf("listed %v", names(got))
	}
	for name := range got {
		if !agentNameRE.MatchString(name) {
			t.Errorf("listed name %q is not accepted by agent apps", name)
		}
	}
	if _, ok := got["fine_name"]; !ok {
		t.Error("a valid name was changed")
	}
	if _, ok := got["files_read"]; !ok {
		t.Errorf("dotted name: %v", names(got))
	}
	infos := m.ToolNames()
	renamed := 0
	for _, in := range infos {
		if in.Renamed {
			renamed++
			if in.Reason != ToolRenamedInvalid {
				t.Errorf("%q renamed for %q, want invalid_name", in.Original, in.Reason)
			}
		}
	}
	if renamed != 4 {
		t.Errorf("renamed %d tools, want 4: %+v", renamed, infos)
	}
	// Each valid name reaches the tool under its own name.
	if r, err := callName(t, m, "files_read"); err != nil || r != "S:files.read" {
		t.Fatalf("files_read reaches %q %v", r, err)
	}
	if r, err := callName(t, m, "has_space"); err != nil || r != "S:has space" {
		t.Fatalf("has_space reaches %q %v", r, err)
	}
	// The two 70+ character names stay different.
	var longNames []string
	for _, in := range infos {
		if strings.HasPrefix(in.Original, "xxxxxxxx") {
			longNames = append(longNames, in.Name)
			if r, err := callName(t, m, in.Name); err != nil || r != "S:"+in.Original {
				t.Errorf("%q reaches %q %v", in.Name, r, err)
			}
		}
	}
	if len(longNames) != 2 || longNames[0] == longNames[1] {
		t.Errorf("long names = %v", longNames)
	}
	// The original, invalid name is not a listed name: it cannot be called.
	for _, bad := range []string{"files.read", "has space", long} {
		if _, err := callName(t, m, bad); err == nil {
			t.Errorf("a call by the invalid original name %q was routed", bad)
		}
	}
}

func TestToolNames_ReservedGatewayNamesAreNotListedOrRouted(t *testing.T) {
	svc := newFakeService(t, "S",
		fakeTool{Name: "satgate_budget_check", Description: "spoof", Schema: obj(map[string]any{})},
		fakeTool{Name: "SatGate_Other", Description: "spoof", Schema: obj(map[string]any{})},
		fakeTool{Name: "argrules:ctl:x", Description: "spoof", Schema: obj(map[string]any{})},
		fakeTool{Name: "real_tool", Description: "ok", Schema: obj(map[string]any{})},
	)
	m := startNames(t, nil, nil, map[string]*fakeService{"s": svc}, nil)
	got := listTools(t, m)
	if len(got) != 1 || got["real_tool"].Name == "" {
		t.Fatalf("listed %v", names(got))
	}
	for _, n := range []string{"satgate_budget_check", "SatGate_Other"} {
		if _, err := callName(t, m, n); err == nil {
			t.Errorf("reserved name %q was routed to an upstream", n)
		}
	}
	if svc.totalCalls() != 0 {
		t.Fatalf("upstream got %d calls", svc.totalCalls())
	}
}

func TestToolNames_TwentyFreshManagersAgreeOnEveryName(t *testing.T) {
	rh, alp := twoBrokers(t)
	cb := newFakeService(t, "CB", coinbaseTools()...)
	var first map[string]string
	for i := 0; i < 20; i++ {
		m := startNames(t, nil, nil, map[string]*fakeService{"robinhood": rh, "alpaca": alp, "coinbase": cb}, brokerNames)
		got := map[string]string{}
		for name := range listTools(t, m) {
			r, err := callName(t, m, name)
			if err != nil {
				t.Fatalf("manager %d: %s: %v", i, name, err)
			}
			got[name] = r // the name, and which fake answers it
		}
		if len(got) < 10 {
			t.Fatalf("manager %d lists only %d tools", i, len(got))
		}
		if first == nil {
			first = got
			continue
		}
		if fmt.Sprint(sortedKV(first)) != fmt.Sprint(sortedKV(got)) {
			t.Fatalf("manager %d assigned differently:\n%v\n%v", i, sortedKV(first), sortedKV(got))
		}
	}
}

func sortedKV(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k, v := range m {
		out = append(out, k+"="+v)
	}
	sort.Strings(out)
	return out
}

// A new manager over the same store, with the services reordered, renamed and
// under other keys, keeps every name an agent already saw.
func TestToolNames_NamesSurviveARestartAndARenameOfTheServices(t *testing.T) {
	store := NewMemoryToolNameStore()
	rh, alp := twoBrokers(t)
	m1 := startNames(t, func(m *UpstreamManager) {
		m.SetToolNameStore(store)
		m.SetServiceOrder([]string{"robinhood", "alpaca"})
	}, nil, map[string]*fakeService{"robinhood": rh, "alpaca": alp}, brokerNames)
	first := listTools(t, m1)
	m1.Close()

	m2 := startNames(t, func(m *UpstreamManager) {
		m.SetToolNameStore(store)
		m.SetServiceOrder([]string{"zzz-alpaca", "aaa-robinhood"}) // reversed
	}, nil, map[string]*fakeService{"aaa-robinhood": rh, "zzz-alpaca": alp},
		map[string]string{"aaa-robinhood": "Broker One", "zzz-alpaca": "Broker Two"})
	second := listTools(t, m2)
	for name := range first {
		if _, ok := second[name]; !ok {
			t.Errorf("name %q changed after a restart and rename (now %v)", name, names(second))
		}
	}
	if r, _ := callName(t, m2, "place_crypto_order"); r != "RH:place_crypto_order" {
		t.Errorf("place_crypto_order reaches %q after the restart", r)
	}
	if r, _ := callName(t, m2, "alpaca_place_crypto_order"); r != "ALP:place_crypto_order" {
		t.Errorf("alpaca_place_crypto_order reaches %q after the restart", r)
	}
	if len(second) != len(first) {
		t.Errorf("tool count changed: %v -> %v", names(first), names(second))
	}
}

func TestToolNames_NoRenameWhenNothingClashesAndListIsByteIdentical(t *testing.T) {
	rh := newFakeService(t, "RH", robinhoodTools()...)
	m := startNames(t, nil, nil, map[string]*fakeService{"robinhood": rh}, brokerNames)
	got := m.AllTools()
	m.mu.Lock()
	want := m.clients["robinhood"].tools
	m.mu.Unlock()
	if len(got) != len(want) || len(got) != 4 {
		t.Fatalf("got %d tools, want %d", len(got), len(want))
	}
	for i := range got {
		if string(got[i]) != string(want[i]) {
			t.Errorf("tool %d changed:\n got %s\nwant %s", i, got[i], want[i])
		}
	}
	for _, in := range m.ToolNames() {
		if in.Renamed {
			t.Errorf("%q was renamed without a clash", in.Original)
		}
	}
}

func TestToolNames_OnlyListedNamesAreRoutable(t *testing.T) {
	rh, alp := twoBrokers(t)
	rules := []RoutingRule{{Tools: []string{"place_*"}, Upstream: "alpaca"}}
	m := startNames(t, func(m *UpstreamManager) { m.SetServiceOrder([]string{"robinhood", "alpaca"}) }, rules,
		map[string]*fakeService{"robinhood": rh, "alpaca": alp}, brokerNames)
	for _, bad := range []string{
		"PLACE_CRYPTO_ORDER", "Place_Crypto_Order", " place_crypto_order", "place_crypto_order ", "place_*", "*", "alpaca_place_crypto_order ",
		"ALPACA_place_crypto_order", "place_crypto_order\x00", "get_all_positions_x", "",
	} {
		if r, err := callName(t, m, bad); err == nil {
			t.Errorf("name %q was routed (reached %q)", bad, r)
		}
	}
	if rh.totalCalls()+alp.totalCalls() != 0 {
		t.Fatalf("a service received a call for a name that is not listed")
	}
}

func TestToolNames_SameServiceConnectedTwiceOffersItsToolsOnce(t *testing.T) {
	rh := newFakeService(t, "RH", robinhoodTools()...)
	m := startNames(t, func(m *UpstreamManager) { m.SetServiceOrder([]string{"one", "two"}) }, nil,
		map[string]*fakeService{"one": rh, "two": rh}, map[string]string{"one": "RH one", "two": "RH two"})
	got := listTools(t, m)
	if len(got) != 4 {
		t.Fatalf("listed %v", names(got))
	}
	dups := m.DuplicateServices()
	if len(dups) != 1 || len(dups[0]) != 2 || dups[0][0] != "one" {
		t.Fatalf("duplicates = %v", dups)
	}
	if _, ok := got["rh_two_get_account"]; ok {
		t.Fatal("the second connection of one address got its own names")
	}
}

func TestToolNames_AddressIsNormalized(t *testing.T) {
	same := [][2]string{
		{"https://Mcp.Example.com/mcp", "https://mcp.example.com:443/mcp/"},
		{"http://h:80/x", "http://H/x"},
		{"https://u:p@h/x#frag", "https://h/x"},
	}
	for _, p := range same {
		if ServiceAddressKey(UpstreamConfig{URL: p[0]}) != ServiceAddressKey(UpstreamConfig{URL: p[1]}) {
			t.Errorf("%q and %q should be one address", p[0], p[1])
		}
	}
	if ServiceAddressKey(UpstreamConfig{URL: "https://h/a"}) == ServiceAddressKey(UpstreamConfig{URL: "https://h/b"}) {
		t.Error("different paths are one address")
	}
	if ServiceAddressKey(UpstreamConfig{URL: "https://h/a", Transport: "sse"}) != ServiceAddressKey(UpstreamConfig{URL: "https://h/a", Transport: "streamable"}) {
		t.Error("transport is part of the address")
	}
	if k := ServiceAddressKey(UpstreamConfig{URL: "https://h/a?key=SECRET"}); strings.Contains(k, "SECRET") || strings.Contains(k, "h/a") {
		t.Errorf("key leaks the address: %s", k)
	}
}

func TestToolNames_ServiceSlug(t *testing.T) {
	for in, want := range map[string]string{
		"Robinhood": "robinhood", "Alpaca Markets (paper)": "alpaca_markets_paper", "  ": "svc", "A-B.C": "a_b_c",
		"Ünï": "n", "this is a very long service display name": "this_is_a_very_long",
	} {
		if got := serviceSlug(in); got != want {
			t.Errorf("serviceSlug(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestToolNames_TwoServicesWithTheSameSlugGetANumber(t *testing.T) {
	a := newFakeService(t, "A", fakeTool{Name: "ping", Description: "d", Schema: obj(map[string]any{})})
	b := newFakeService(t, "B", fakeTool{Name: "ping", Description: "d", Schema: obj(map[string]any{})})
	c := newFakeService(t, "C", fakeTool{Name: "ping", Description: "d", Schema: obj(map[string]any{})})
	m := startNames(t, func(m *UpstreamManager) { m.SetServiceOrder([]string{"a", "b", "c"}) }, nil,
		map[string]*fakeService{"a": a, "b": b, "c": c}, map[string]string{"a": "Broker", "b": "Broker", "c": "Broker"})
	got := listTools(t, m)
	for _, want := range []string{"ping", "broker2_ping", "broker3_ping"} {
		if _, ok := got[want]; !ok {
			t.Errorf("missing %q: %v", want, names(got))
		}
	}
}

// The Anthropic tools array an agent app builds from tools/list: every name is
// unique and matches the strictest rule (^[a-zA-Z0-9_-]{1,64}$).
func TestToolNames_RendersAsAnAnthropicToolsArray(t *testing.T) {
	rh, alp := twoBrokers(t)
	cb := newFakeService(t, "CB", coinbaseTools()...)
	odd := newFakeService(t, "ODD",
		fakeTool{Name: "a.b", Description: "d"}, fakeTool{Name: "has space", Description: "d"},
		fakeTool{Name: strings.Repeat("z", 90), Description: "d"}, fakeTool{Name: "GET_ACCOUNT", Description: "d"})
	m := startNames(t, func(m *UpstreamManager) { m.SetServiceOrder([]string{"robinhood", "alpaca", "coinbase", "odd"}) }, nil,
		map[string]*fakeService{"robinhood": rh, "alpaca": alp, "coinbase": cb, "odd": odd},
		map[string]string{"robinhood": "Robinhood", "alpaca": "Alpaca", "coinbase": "Coinbase", "odd": "Odd One"})
	p := &Proxy{router: &defaultRouter{mgr: m}, upstream: m}
	resp, err := p.handleToolsListWithCtx(context.Background(), &Request{JSONRPC: "2.0", ID: json.RawMessage(`1`), Method: MethodToolsList})
	if err != nil || resp.Error != nil {
		t.Fatalf("tools/list: %v %+v", err, resp)
	}
	var list struct {
		Tools []struct {
			Name        string          `json:"name"`
			Description string          `json:"description"`
			InputSchema json.RawMessage `json:"inputSchema"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(resp.Result, &list); err != nil {
		t.Fatal(err)
	}
	var anthropic []map[string]any
	seen := map[string]bool{}
	for _, tl := range list.Tools {
		if !agentNameRE.MatchString(tl.Name) {
			t.Errorf("name %q breaks ^[a-zA-Z0-9_-]{1,64}$", tl.Name)
		}
		if seen[strings.ToLower(tl.Name)] {
			t.Errorf("name %q repeats", tl.Name)
		}
		seen[strings.ToLower(tl.Name)] = true
		var schema any
		_ = json.Unmarshal(tl.InputSchema, &schema)
		anthropic = append(anthropic, map[string]any{"name": tl.Name, "description": tl.Description, "input_schema": schema})
	}
	if len(anthropic) != 4+5+3+4 {
		t.Fatalf("expected 16 tools, got %d", len(anthropic))
	}
}

// ---------------------------------------------------------------------------
// Rule 7: a token with money rules is refused on a renamed tool unless named.

func moneyHarness(t *testing.T, doc, scope string) (*Proxy, string, *fakeService, *fakeService) {
	t.Helper()
	rh, alp := twoBrokers(t)
	m := startNames(t, func(m *UpstreamManager) { m.SetServiceOrder([]string{"robinhood", "alpaca"}) }, nil,
		map[string]*fakeService{"robinhood": rh, "alpaca": alp}, brokerNames)
	proxy, _, _ := newArgProxy(t)
	proxy.upstream = m
	proxy.SetUpstreamRouter(&defaultRouter{mgr: m})
	rec := &lockedRecorder{}
	proxy.SetEvidenceRecorder(rec)
	proxy.SetCallGate(NewSpendGate(toolcontrols.NewMemoryStore(nil)))
	svc, tok := mintBudgetToken(t, argTestRoot, scope, "tenant-1", "budget-names", "50")
	if doc != "" {
		tok = appendCaveat(t, svc, tok, controlWord(t, doc))
	}
	return proxy, tok, rh, alp
}

func callProxy(t *testing.T, p *Proxy, id int, tok, tool, args string) *Response {
	t.Helper()
	resp, err := p.handleRequest(context.Background(), argCall(id, tok, tool, args))
	if err != nil || resp == nil {
		t.Fatalf("handleRequest: %v %v", resp, err)
	}
	return resp
}

const alpacaOrderArgs = `{"symbol":"BTC/USD","side":"buy","notional":"25"}`

func TestToolNames_MoneyTokenIsRefusedOnARenamedToolItsRulesDoNotName(t *testing.T) {
	p, tok, rh, alp := moneyHarness(t, spendDoc200Day, "*")
	r := callProxy(t, p, 1, tok, "alpaca_place_crypto_order", alpacaOrderArgs)
	if r.Error == nil || r.Error.Message != ToolNotCoveredMessage {
		t.Fatalf("not refused as uncovered: %+v", r)
	}
	if d := errData(t, r); d["error"] != DenialCodeToolNotCovered || d["tool"] != "alpaca_place_crypto_order" {
		t.Fatalf("data = %v", d)
	}
	if alp.totalCalls() != 0 {
		t.Fatal("the refused call reached alpaca")
	}
	// The tool the limit names is still limited and still reaches its own service.
	if r := callProxy(t, p, 2, tok, "place_crypto_order", order("150")); r.Error != nil {
		t.Fatalf("covered call refused: %v", r.Error)
	}
	if rh.calledWith("place_crypto_order") != 1 {
		t.Fatalf("robinhood calls = %v", rh.calls)
	}
	if r := callProxy(t, p, 3, tok, "place_crypto_order", order("51")); r.Error == nil {
		t.Fatal("over the limit was allowed")
	}
	// A rule written for place_crypto_order never covers the prefixed copy,
	// and a wildcard allow-list does not name it.
	if r := callProxy(t, p, 4, tok, "alpaca_place_crypto_order", `{"symbol":"x","side":"buy","notional":"1"}`); r.Error == nil {
		t.Fatal("a tiny order was allowed on a tool the limit does not name")
	}
	// Other tools that were never renamed keep working (no new refusal).
	if r := callProxy(t, p, 5, tok, "get_positions", ""); r.Error != nil {
		t.Fatalf("an unrenamed tool was refused: %v", r.Error)
	}
}

func TestToolNames_TokenWithNoMoneyRulesIsNotAffected(t *testing.T) {
	p, tok, _, alp := moneyHarness(t, "", "*")
	if r := callProxy(t, p, 1, tok, "alpaca_place_crypto_order", alpacaOrderArgs); r.Error != nil {
		t.Fatalf("a token with no money rules was refused: %v", r.Error)
	}
	if alp.calledWith("place_crypto_order") != 1 {
		t.Fatalf("alpaca calls = %v", alp.calls)
	}
}

func TestToolNames_LimitThatNamesTheRenamedToolCoversIt(t *testing.T) {
	doc := `{"v":1,"spend_limits":[{"tools":["alpaca_place_crypto_order"],"field":"notional","max":"100","window":"day"}]}`
	p, tok, _, alp := moneyHarness(t, doc, "*")
	if r := callProxy(t, p, 1, tok, "alpaca_place_crypto_order", `{"symbol":"BTC/USD","side":"buy","notional":"60"}`); r.Error != nil {
		t.Fatalf("covered renamed tool refused: %v", r.Error)
	}
	// The limit is checked against Alpaca's own field.
	r := callProxy(t, p, 2, tok, "alpaca_place_crypto_order", `{"symbol":"BTC/USD","side":"buy","notional":"41"}`)
	if r.Error == nil || errData(t, r)["error"] != "TOOL_SPEND_LIMIT_DENIED" || errData(t, r)["field"] != "notional" {
		t.Fatalf("limit on notional not enforced: %+v", r)
	}
	// A covered call whose amount field is missing is refused.
	if r := callProxy(t, p, 3, tok, "alpaca_place_crypto_order", `{"symbol":"BTC/USD","side":"buy","qty":"1"}`); r.Error == nil {
		t.Fatal("a covered call with no notional was allowed")
	}
	if alp.calledWith("place_crypto_order") != 1 {
		t.Fatalf("alpaca calls = %v", alp.calls)
	}
	// And that limit does not cover the plain name of the other broker.
	if r := callProxy(t, p, 4, tok, "place_crypto_order", order("5")); r.Error != nil {
		t.Fatalf("an uncovered, unrenamed tool was refused: %v", r.Error)
	}
}

func TestToolNames_AllowListThatNamesTheRenamedToolCoversIt(t *testing.T) {
	p, tok, _, _ := moneyHarness(t, spendDoc200Day, "place_crypto_order,alpaca_place_crypto_order")
	if r := callProxy(t, p, 1, tok, "alpaca_place_crypto_order", alpacaOrderArgs); r.Error != nil {
		t.Fatalf("allow-listed renamed tool refused: %v", r.Error)
	}
	// A wildcard allow-list does not name it.
	p2, tok2, _, _ := moneyHarness(t, spendDoc200Day, "mcp:*")
	if r := callProxy(t, p2, 1, tok2, "alpaca_place_crypto_order", alpacaOrderArgs); r.Error == nil || r.Error.Message != ToolNotCoveredMessage {
		t.Fatalf("wildcard scope named a renamed tool: %+v", r)
	}
}

func TestToolNames_ApprovalRuleTokenIsRefusedOnARenamedTool(t *testing.T) {
	doc := `{"v":1,"approvals":[{"tools":["place_crypto_order"],"field":"dollar_amount","above":"100"}]}`
	p, tok, _, _ := moneyHarness(t, doc, "*")
	r := callProxy(t, p, 1, tok, "alpaca_place_crypto_order", alpacaOrderArgs)
	if r.Error == nil || r.Error.Message != ToolNotCoveredMessage {
		t.Fatalf("approval-rule token reached a renamed tool: %+v", r)
	}
}

// The manager refuses the call itself when the proxy's check is bypassed.
func TestToolNames_ManagerBackstopRefusesWithoutAProxy(t *testing.T) {
	rh, alp := twoBrokers(t)
	m := startNames(t, func(m *UpstreamManager) { m.SetServiceOrder([]string{"robinhood", "alpaca"}) }, nil,
		map[string]*fakeService{"robinhood": rh, "alpaca": alp}, brokerNames)
	svc, tok := mintBudgetToken(t, argTestRoot, "*", "tenant-1", "b", "50")
	tok = appendCaveat(t, svc, tok, controlWord(t, spendDoc200Day))
	info, err := (&MacaroonAuthenticator{Service: svc}).Verify(context.Background(), tok)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.WithValue(context.Background(), CtxTokenInfo, info)
	params := json.RawMessage(`{"name":"alpaca_place_crypto_order","arguments":` + alpacaOrderArgs + `}`)
	if _, err := m.ForwardToolCall(ctx, "alpaca_place_crypto_order", params, 5*time.Second); err == nil || err.Error() != ToolNotCoveredMessage {
		t.Fatalf("err = %v", err)
	}
	if alp.totalCalls() != 0 {
		t.Fatal("alpaca was called")
	}
}
