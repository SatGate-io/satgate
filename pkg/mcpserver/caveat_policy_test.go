package mcpserver

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/satgate-io/satgate/pkg/macaroon"
)

func appendCaveat(t *testing.T, svc *macaroon.Service, token string, caveats ...string) string {
	t.Helper()
	child, err := svc.DelegateWithoutVerify(token, caveats)
	if err != nil {
		t.Fatal(err)
	}
	return svc.Encode(child)
}

func mintBudgetToken(t *testing.T, root, scope, tenantID, budgetID, limit string) (*macaroon.Service, string) {
	t.Helper()
	svc, err := macaroon.NewService(root)
	if err != nil {
		t.Fatal(err)
	}
	mac, err := svc.Mint(scope, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if tenantID != "" {
		mac.AddCaveat("tenant_id", tenantID)
	}
	if budgetID != "" {
		mac.AddCaveat("budget_id", budgetID)
	}
	if limit != "" {
		mac.AddCaveat("budget_limit", limit)
	}
	mac.Signature = svc.RecalculateSignature(mac)
	token := svc.Encode(mac)
	if budgetID != "" {
		sealed, err := svc.AppendBudgetBind(token)
		if err != nil {
			t.Fatal(err)
		}
		token = svc.Encode(sealed)
	}
	return svc, token
}

func TestMacaroonAuthenticator_ConflictingTenantIDRejected(t *testing.T) {
	const tenantA = "00000000-0000-4000-8000-0000000000a1"
	const tenantB = "00000000-0000-4000-8000-0000000000b2"
	svc, token := mintBudgetToken(t, "root", "gpt4_chat", tenantA, "budget-a", "2")
	widened := appendCaveat(t, svc, token, "tenant_id = "+tenantB)
	auth := &MacaroonAuthenticator{Service: svc}
	if _, err := auth.Verify(context.Background(), widened); err == nil || !strings.Contains(err.Error(), "conflicting tenant_id") {
		t.Fatalf("conflicting tenant_id must be rejected, got %v", err)
	}
}

func TestMacaroonAuthenticator_RepeatedTenantIDAllowed(t *testing.T) {
	const tenantA = "00000000-0000-4000-8000-0000000000a1"
	svc, token := mintBudgetToken(t, "root", "gpt4_chat", tenantA, "budget-a", "2")
	repeated := appendCaveat(t, svc, token, "tenant_id = "+tenantA)
	auth := &MacaroonAuthenticator{Service: svc}
	info, err := auth.Verify(context.Background(), repeated)
	if err != nil {
		t.Fatal(err)
	}
	if info.TenantID != tenantA {
		t.Fatalf("tenant = %s, want %s", info.TenantID, tenantA)
	}
}

func TestMacaroonAuthenticator_AppendedWildcardScopeRejected(t *testing.T) {
	svc, token := mintBudgetToken(t, "root", "gpt4_chat", "", "budget-a", "2")
	widened := appendCaveat(t, svc, token, "scope = *")
	auth := &MacaroonAuthenticator{Service: svc}
	if _, err := auth.Verify(context.Background(), widened); err == nil || !strings.Contains(err.Error(), "widens") {
		t.Fatalf("appended universal scope must be rejected, got %v", err)
	}
}

func TestToolsCall_EveryScopeCaveatMustAllowTool(t *testing.T) {
	svc, token := mintBudgetToken(t, "root", "gpt4_chat", "", "budget-a", "2")
	// A later list that names an extra tool is not a universal widen, so
	// verification still succeeds. The request path must still refuse the extra tool.
	widened := appendCaveat(t, svc, token, "scope = gpt4_chat,other_tool")
	auth := &MacaroonAuthenticator{Service: svc}
	info, err := auth.Verify(context.Background(), widened)
	if err != nil {
		t.Fatal(err)
	}
	if info.AllowsTool("other_tool") {
		t.Fatal("a later scope caveat must not add other_tool")
	}
	if !info.AllowsTool("gpt4_chat") {
		t.Fatal("gpt4_chat should remain allowed")
	}

	proxy := newEvidenceTestProxy(t)
	router := &fakeMCPRouter{}
	proxy.router = router
	// Scope is what last-caveat matching used to publish. The request path
	// must not treat that string as sufficient.
	info.Scope = "*"
	req := &Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  MethodToolsCall,
		Params:  json.RawMessage(`{"name":"other_tool","arguments":{}}`),
	}
	resp, err := proxy.handleToolsCall(context.Background(), req, info)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Error == nil || resp.Error.Code != CodePolicyDenied {
		t.Fatalf("tool call must be denied, got %+v", resp.Error)
	}
	if router.called {
		t.Fatal("denied tool must not be forwarded")
	}
}

func TestMacaroonAuthenticator_AppendedBudgetIDRejected(t *testing.T) {
	svc, token := mintBudgetToken(t, "root", "gpt4_chat", "", "budget-a", "2")
	retargeted := appendCaveat(t, svc, token, "budget_id = budget-other")
	auth := &MacaroonAuthenticator{Service: svc}
	if _, err := auth.Verify(context.Background(), retargeted); err == nil || !strings.Contains(err.Error(), "not server-issued") {
		t.Fatalf("unbound budget_id must be rejected, got %v", err)
	}
}

func TestMacaroonAuthenticator_AppendedBudgetLimitClamped(t *testing.T) {
	svc, token := mintBudgetToken(t, "root", "gpt4_chat", "", "budget-a", "2")
	raised := appendCaveat(t, svc, token, "budget_limit = 999999")
	auth := &MacaroonAuthenticator{Service: svc}
	info, err := auth.Verify(context.Background(), raised)
	if err != nil {
		t.Fatal(err)
	}
	if info.BudgetLimit != 2 {
		t.Fatalf("budget limit = %d, want 2 (minimum, not the appended cap)", info.BudgetLimit)
	}
	lowered := appendCaveat(t, svc, token, "budget_limit = 1")
	info, err = auth.Verify(context.Background(), lowered)
	if err != nil {
		t.Fatal(err)
	}
	if info.BudgetLimit != 1 {
		t.Fatalf("budget limit = %d, want 1", info.BudgetLimit)
	}
}

func TestDelegator_ChildBudgetIDStillVerifies(t *testing.T) {
	ctx := context.Background()
	svc, parentToken := mintBudgetToken(t, "root", "api:*", "tenant-a", "budget-parent", "100")
	auth := &MacaroonAuthenticator{Service: svc}
	parent, err := auth.Verify(ctx, parentToken)
	if err != nil {
		t.Fatal(err)
	}
	budget := NewInMemoryBudgetEnforcer()
	if err := budget.Initialize(ctx, parent.BudgetID, 100); err != nil {
		t.Fatal(err)
	}
	result, err := NewDelegator(svc, budget).Delegate(ctx, parent, &DelegateParams{
		Budget: 30,
		Label:  "research-agent",
		Scope:  "api:read",
	})
	if err != nil {
		t.Fatal(err)
	}
	child, err := auth.Verify(ctx, result.Token)
	if err != nil {
		t.Fatalf("delegated child must still verify: %v", err)
	}
	if child.BudgetID != result.BudgetID {
		t.Fatalf("child budget = %s, want %s", child.BudgetID, result.BudgetID)
	}
	if child.BudgetID == parent.BudgetID {
		t.Fatal("child must not spend the parent budget")
	}
	if !child.AllowsTool("api:read") || child.AllowsTool("api:write") {
		t.Fatalf("child scope should be api:read only, scope=%q", child.Scope)
	}
	if child.BudgetLimit != 30 {
		t.Fatalf("child limit = %d, want 30", child.BudgetLimit)
	}
}

func TestMacaroonAuthenticator_FirstAppendedBudgetIDRejected(t *testing.T) {
	svc, token := mintBudgetToken(t, "root", "gpt4_chat", "", "", "")
	appended := appendCaveat(t, svc, token, "budget_id = budget-appended")
	auth := &MacaroonAuthenticator{Service: svc}
	if _, err := auth.Verify(context.Background(), appended); err == nil || !strings.Contains(err.Error(), "not server-issued") {
		t.Fatalf("first appended budget_id must be rejected, got %v", err)
	}

	proxy := newSpendProxy(t, "root", svc)
	req := &Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  MethodToolsCall,
		Params:  json.RawMessage(`{"name":"gpt4_chat","arguments":{},"_meta":{"token":"` + appended + `"}}`),
	}
	resp, err := proxy.handleRequest(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Error == nil || !strings.Contains(resp.Error.Message, "not server-issued") {
		t.Fatalf("spend path must reject the appended id, got %+v", resp.Error)
	}
	for _, rec := range proxy.budget.(*InMemoryBudgetEnforcer).SpendLog() {
		if rec.TokenID == "budget-appended" {
			t.Fatalf("spend used the appended id: %+v", rec)
		}
	}
}

func TestMacaroonAuthenticator_RepeatedSealedBudgetIDStillSpendsIssuedID(t *testing.T) {
	svc, token := mintBudgetToken(t, "root", "gpt4_chat", "", "budget-issued", "20")
	repeated := appendCaveat(t, svc, token, "budget_id = budget-issued")
	auth := &MacaroonAuthenticator{Service: svc}
	info, err := auth.Verify(context.Background(), repeated)
	if err != nil {
		t.Fatal(err)
	}
	if info.BudgetID != "budget-issued" {
		t.Fatalf("budget id = %s, want budget-issued", info.BudgetID)
	}

	proxy := newSpendProxy(t, "root", svc)
	if err := proxy.budget.Initialize(context.Background(), "budget-issued", 20); err != nil {
		t.Fatal(err)
	}
	req := &Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  MethodToolsCall,
		Params:  json.RawMessage(`{"name":"gpt4_chat","arguments":{},"_meta":{"token":"` + repeated + `"}}`),
	}
	resp, err := proxy.handleRequest(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Error != nil {
		t.Fatalf("repeated sealed id must still spend, got %+v", resp.Error)
	}
	log := proxy.budget.(*InMemoryBudgetEnforcer).SpendLog()
	if len(log) != 1 || log[0].TokenID != "budget-issued" {
		t.Fatalf("spend log = %+v, want the issued id", log)
	}
}

func TestMacaroonAuthenticator_CopiedBudgetSealRejected(t *testing.T) {
	svcA, tokenA := mintBudgetToken(t, "root-a", "gpt4_chat", "", "budget-a", "2")
	macA, err := svcA.Decode(tokenA)
	if err != nil {
		t.Fatal(err)
	}
	var copied string
	for _, caveat := range macA.Caveats {
		if strings.HasPrefix(caveat, "budget_bind = ") {
			copied = caveat
		}
	}
	if copied == "" {
		t.Fatal("issued token missing budget_bind")
	}
	svcB, tokenB := mintBudgetToken(t, "root-b", "gpt4_chat", "", "", "")
	stolen := appendCaveat(t, svcB, tokenB, "budget_id = budget-stolen", copied)
	auth := &MacaroonAuthenticator{Service: svcB}
	if _, err := auth.Verify(context.Background(), stolen); err == nil || !strings.Contains(err.Error(), "not server-issued") {
		t.Fatalf("seal copied from another chain must be rejected, got %v", err)
	}
}

func TestDelegator_ProductionChildResolvesSealedBudget(t *testing.T) {
	ctx := context.Background()
	svc, parentToken := mintBudgetToken(t, "root", "api:*", "", "budget-parent", "40")
	auth := &MacaroonAuthenticator{Service: svc}
	parent, err := auth.Verify(ctx, parentToken)
	if err != nil {
		t.Fatal(err)
	}
	budget := NewInMemoryBudgetEnforcer()
	if err := budget.Initialize(ctx, parent.BudgetID, 40); err != nil {
		t.Fatal(err)
	}
	result, err := NewDelegator(svc, budget).Delegate(ctx, parent, &DelegateParams{Budget: 5, Scope: "api:read"})
	if err != nil {
		t.Fatal(err)
	}
	mac, err := svc.Decode(result.Token)
	if err != nil {
		t.Fatal(err)
	}
	got, err := svc.ResolveIssuedBudgetID(mac)
	if err != nil {
		t.Fatalf("production delegator child must resolve: %v", err)
	}
	if got != result.BudgetID {
		t.Fatalf("resolved budget = %s, want %s", got, result.BudgetID)
	}
}

func newSpendProxy(t *testing.T, root string, svc *macaroon.Service) *Proxy {
	t.Helper()
	cfg := &Config{
		Server:      ServerConfig{Transport: "stdio", Name: "test", Version: "1.0"},
		Auth:        AuthConfig{Mode: "header", RootKey: root},
		Upstreams:   map[string]UpstreamConfig{"mock": {Transport: "stdio", Command: []string{"python3", "-c", "import sys; sys.exit(0)"}}},
		Budget:      BudgetConfig{Backend: "memory", FailMode: "closed"},
		Tools:       ToolsConfig{DefaultCost: 10, Costs: map[string]int64{"gpt4_chat": 10}},
		Enforcement: EnforcementConfig{Mode: "control"},
		Logging:     LoggingConfig{Level: "error"},
	}
	cfg.applyDefaults()
	proxy, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	proxy.auth = &MacaroonAuthenticator{Service: svc}
	proxy.router = &fakeMCPRouter{}
	return proxy
}

func TestDelegate_DepthCapIgnoresLaterWiden(t *testing.T) {
	for _, extra := range []string{"delegation_depth = 9", "delegation_depth = nope", "delegation_depth = 0"} {
		t.Run(extra, func(t *testing.T) {
			assertDepthStopsAt(t, "2", extra, 2)
		})
	}
}

func TestDelegate_AppendedDepthCapNarrowsUnlimited(t *testing.T) {
	assertDepthStopsAt(t, "", "delegation_depth = 2", 2)
}

func TestDelegate_BudgetCapIgnoresLaterWiden(t *testing.T) {
	for _, extra := range []string{"delegation_budget = 90", "delegation_budget = nope", "delegation_budget = 0"} {
		t.Run(extra, func(t *testing.T) {
			assertBudgetCap(t, "10", extra, 10)
		})
	}
}

func TestDelegate_AppendedBudgetCapNarrowsUnlimited(t *testing.T) {
	assertBudgetCap(t, "", "delegation_budget = 10", 10)
}

func TestDelegate_LaterSmallerBudgetCapKept(t *testing.T) {
	assertBudgetCap(t, "100", "delegation_budget = 10", 10)
}

func assertDepthStopsAt(t *testing.T, issued, extra string, stop int) {
	t.Helper()
	ctx := context.Background()
	svc, err := macaroon.NewService("root")
	if err != nil {
		t.Fatal(err)
	}
	mac, err := svc.Mint("api:*", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if issued != "" {
		mac.AddCaveat("delegation_depth", issued)
	}
	mac.Signature = svc.RecalculateSignature(mac)
	token := svc.Encode(mac)
	if extra != "" {
		token = appendCaveat(t, svc, token, extra)
	}
	auth := &MacaroonAuthenticator{Service: svc}
	delegator := NewDelegator(svc, NewInMemoryBudgetEnforcer())
	for i := 0; i < stop; i++ {
		parent, err := auth.Verify(ctx, token)
		if err != nil {
			t.Fatal(err)
		}
		result, err := delegator.Delegate(ctx, parent, &DelegateParams{Budget: 0, Scope: "api:read"})
		if err != nil {
			t.Fatalf("delegation %d should succeed under cap %d: %v", i+1, stop, err)
		}
		token = result.Token
	}
	parent, err := auth.Verify(ctx, token)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := delegator.Delegate(ctx, parent, &DelegateParams{Budget: 0, Scope: "api:read"}); err == nil || !strings.Contains(err.Error(), "depth limit") {
		t.Fatalf("delegation past %d must stop, got %v", stop, err)
	}
}

func assertBudgetCap(t *testing.T, issued, extra string, cap int64) {
	t.Helper()
	ctx := context.Background()
	svc, token := mintBudgetToken(t, "root", "api:*", "", "budget-parent", "100")
	mac, err := svc.Decode(token)
	if err != nil {
		t.Fatal(err)
	}
	if issued != "" {
		mac.AddCaveat("delegation_budget", issued)
		mac.Signature = svc.RecalculateSignature(mac)
		sealed, err := svc.AppendBudgetBind(svc.Encode(mac))
		if err != nil {
			t.Fatal(err)
		}
		token = svc.Encode(sealed)
	}
	if extra != "" {
		token = appendCaveat(t, svc, token, extra)
	}
	auth := &MacaroonAuthenticator{Service: svc}
	parent, err := auth.Verify(ctx, token)
	if err != nil {
		t.Fatal(err)
	}
	if parent.DelegationBudget != cap {
		t.Fatalf("delegation budget = %d, want %d", parent.DelegationBudget, cap)
	}
	budget := NewInMemoryBudgetEnforcer()
	if err := budget.Initialize(ctx, parent.BudgetID, 100); err != nil {
		t.Fatal(err)
	}
	delegator := NewDelegator(svc, budget)
	if _, err := delegator.Delegate(ctx, parent, &DelegateParams{Budget: cap + 1, Scope: "api:read"}); err == nil || !strings.Contains(err.Error(), "exceeds cap") {
		t.Fatalf("carve above %d must fail, got %v", cap, err)
	}
	if _, err := delegator.Delegate(ctx, parent, &DelegateParams{Budget: cap, Scope: "api:read"}); err != nil {
		t.Fatalf("carve at the narrowed cap must succeed: %v", err)
	}
}
