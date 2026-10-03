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
	return svc, svc.Encode(mac)
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
