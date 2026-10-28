package mcpserver

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/satgate-io/satgate/pkg/macaroon"
)

const maxBudgetText = "9223372036854775807" // math.MaxInt64

// mintChildAndVerify delegates amount from a parent whose budget_limit is
// math.MaxInt64, then verifies the child with the same authenticator a tool
// call uses. It returns the child's caveats and what the verifier read.
func mintChildAndVerify(t *testing.T, amount int64) ([]string, *TokenInfo, error) {
	t.Helper()
	ctx := context.Background()
	svc, token := mintBudgetToken(t, "root", "mcp:*", "", "budget-parent", maxBudgetText)
	auth := &MacaroonAuthenticator{Service: svc}
	parent, err := auth.Verify(ctx, token)
	if err != nil {
		t.Fatal(err)
	}
	budget := NewInMemoryBudgetEnforcer()
	if err := budget.Initialize(ctx, parent.BudgetID, math.MaxInt64); err != nil {
		t.Fatal(err)
	}
	result, err := NewDelegator(svc, budget).Delegate(ctx, parent, &DelegateParams{Budget: amount, Scope: "api:read"})
	if err != nil {
		t.Fatalf("budget %d: delegation refused: %v", amount, err)
	}
	mac, err := svc.Decode(result.Token)
	if err != nil {
		t.Fatal(err)
	}
	info, verr := auth.Verify(ctx, result.Token)
	return mac.Caveats, info, verr
}

func budgetLimitCaveats(caveats []string) []string {
	var out []string
	for _, c := range caveats {
		if strings.HasPrefix(c, "budget_limit") {
			out = append(out, c)
		}
	}
	return out
}

// Every budget the delegator accepts is written as the exact integer and read
// back by the verifier as the same number: 1, typical values, values above
// 2^53 (where a float64 loses whole numbers) and math.MaxInt64.
func TestDelegate_MintedBudgetLimitVerifiesExactlyAcrossRange(t *testing.T) {
	for _, amount := range []int64{
		1, 5, 10, 100, 1000, 25000, 1000000,
		1 << 53, 1<<53 + 1, 1<<62 + 1,
		math.MaxInt64 - 511, math.MaxInt64 - 1, math.MaxInt64,
	} {
		t.Run(strconv.FormatInt(amount, 10), func(t *testing.T) {
			caveats, info, err := mintChildAndVerify(t, amount)
			if err != nil {
				t.Fatalf("the minted child is refused at verify: %v", err)
			}
			want := "budget_limit = " + strconv.FormatInt(amount, 10)
			// The parent's own budget_limit comes first; the child's is last.
			got := budgetLimitCaveats(caveats)
			if len(got) == 0 || got[len(got)-1] != want {
				t.Fatalf("child budget_limit caveats %q, want the last to be %q", got, want)
			}
			if info.BudgetLimit != amount {
				t.Fatalf("verified budget %d, want exactly %d", info.BudgetLimit, amount)
			}
		})
	}
}

// The largest budget the delegator accepts is math.MaxInt64; the child minted
// with it verifies.
func TestDelegate_LargestWholeBudgetChildVerifies(t *testing.T) {
	_, info, err := mintChildAndVerify(t, math.MaxInt64)
	if err != nil {
		t.Fatalf("the child minted with budget %d is refused at verify: %v", int64(math.MaxInt64), err)
	}
	if info.BudgetLimit != math.MaxInt64 {
		t.Fatalf("verified budget %d, want %d", info.BudgetLimit, int64(math.MaxInt64))
	}
}

// A budget above 2^53 is not rounded on the way into the child: 2^53+1 is
// read back as 2^53+1, not as 2^53.
func TestDelegate_UpperRangeBudgetNotRoundedThroughFloat(t *testing.T) {
	const amount = int64(9007199254740993) // 2^53 + 1
	_, info, err := mintChildAndVerify(t, amount)
	if err != nil {
		t.Fatalf("child refused at verify: %v", err)
	}
	if info.BudgetLimit != amount {
		t.Fatalf("verified budget %d, want exactly %d", info.BudgetLimit, amount)
	}
}

// A zero-credit delegation writes no budget_limit and verifies with no limit
// of its own, as before.
func TestDelegate_ZeroBudgetChildCarriesNoBudgetLimit(t *testing.T) {
	ctx := context.Background()
	svc, token := mintBudgetToken(t, "root", "mcp:*", "", "budget-parent", "100")
	auth := &MacaroonAuthenticator{Service: svc}
	parent, err := auth.Verify(ctx, token)
	if err != nil {
		t.Fatal(err)
	}
	result, err := NewDelegator(svc, NewInMemoryBudgetEnforcer()).Delegate(ctx, parent, &DelegateParams{Budget: 0, Scope: "api:read"})
	if err != nil {
		t.Fatal(err)
	}
	mac, err := svc.Decode(result.Token)
	if err != nil {
		t.Fatal(err)
	}
	if got := budgetLimitCaveats(mac.Caveats); len(got) != 1 || got[0] != "budget_limit = 100" {
		t.Fatalf("a zero-credit child must carry only the parent's budget_limit, got %q", got)
	}
	if _, err := auth.Verify(ctx, result.Token); err != nil {
		t.Fatalf("zero-credit child refused: %v", err)
	}
}

// A budget the delegator cannot hold exactly (above math.MaxInt64, or not a
// whole number) is refused when the request is read, with an error that says
// what is allowed. Nothing is minted and the parent is not charged.
func TestHandleDelegate_BudgetAboveLargestRefusedAtMint(t *testing.T) {
	ctx := context.Background()
	for name, raw := range map[string]string{
		"one above the largest": `{"budget":9223372036854775808,"scope":"api:read"}`,
		"far above the largest": `{"budget":99999999999999999999,"scope":"api:read"}`,
		"a fraction":            `{"budget":1.5,"scope":"api:read"}`,
		"exponent form":         `{"budget":1e3,"scope":"api:read"}`,
	} {
		t.Run(name, func(t *testing.T) {
			svc, token := mintBudgetToken(t, "root", "mcp:*", "", "budget-parent", maxBudgetText)
			parent, err := (&MacaroonAuthenticator{Service: svc}).Verify(ctx, token)
			if err != nil {
				t.Fatal(err)
			}
			budget := NewInMemoryBudgetEnforcer()
			if err := budget.Initialize(ctx, parent.BudgetID, math.MaxInt64); err != nil {
				t.Fatal(err)
			}
			resp, err := NewDelegator(svc, budget).HandleDelegate(ctx, &Request{JSONRPC: "2.0", ID: json.RawMessage(`1`), Method: MethodSatGateDelegate, Params: json.RawMessage(raw)}, parent)
			if err != nil {
				t.Fatal(err)
			}
			if resp == nil || resp.Error == nil || resp.Result != nil {
				t.Fatalf("want a refusal and no token, got %+v", resp)
			}
			if resp.Error.Code != CodeInvalidParams {
				t.Fatalf("code %d, want %d", resp.Error.Code, CodeInvalidParams)
			}
			if want := "budget must be a whole number of credits from 0 to 9223372036854775807"; !strings.Contains(resp.Error.Message, want) {
				t.Fatalf("message %q does not say %q", resp.Error.Message, want)
			}
			if left, _ := budget.Remaining(ctx, parent.BudgetID); left != math.MaxInt64 {
				t.Fatalf("parent charged: %d left", left)
			}
		})
	}
}

// The same largest-budget delegation made over the MCP HTTP transport returns
// a child that verifies.
func TestDelegate_LargestWholeBudgetOverHTTPVerifies(t *testing.T) {
	svc, tok := mintBudgetToken(t, "root", "mcp:*", "", "budget-parent", maxBudgetText)
	p := newEvidenceTestProxy(t)
	p.config.Auth.Mode = "header"
	p.SetAuthenticator(&MacaroonAuthenticator{Service: svc})
	p.delegator = NewDelegator(svc, p.budget)
	srv := httptest.NewServer(NewSSEServer(p, ":0").Handler())
	defer srv.Close()
	sid := mustInitSession(t, srv.URL, tok)
	body := map[string]any{"jsonrpc": "2.0", "id": 2, "method": MethodSatGateDelegate, "params": map[string]any{"budget": int64(math.MaxInt64), "scope": "api:read"}}
	status, _, raw := streamablePost(t, srv.URL+"/mcp", sid, tok, body)
	if status != http.StatusOK {
		t.Fatalf("delegation HTTP status %d", status)
	}
	var resp Response
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatal("unreadable delegation response")
	}
	if resp.Error != nil {
		t.Fatalf("delegation refused: %+v", resp.Error)
	}
	var result DelegateResult
	if err := json.Unmarshal(resp.Result, &result); err != nil || result.Token == "" {
		t.Fatal("no child token in the response")
	}
	info, err := (&MacaroonAuthenticator{Service: svc}).Verify(context.Background(), result.Token)
	if err != nil {
		t.Fatalf("the child issued over HTTP is refused at verify: %v", err)
	}
	if info.BudgetLimit != math.MaxInt64 {
		t.Fatalf("verified budget %d, want %d", info.BudgetLimit, int64(math.MaxInt64))
	}
}

// Children written by earlier releases as "budget_limit = N.00" still verify,
// and read as N. The largest such value an earlier release could write
// exactly is the largest float64 below 2^63.
func TestFillTokenInfo_LegacyTwoDecimalBudgetLimitStillVerifies(t *testing.T) {
	ctx := context.Background()
	for text, want := range map[string]int64{
		"1.00":                   1,
		"5.00":                   5,
		"100.00":                 100,
		"1000000.00":             1000000,
		"9007199254740992.00":    9007199254740992,
		"9223372036854774784.00": 9223372036854774784,
	} {
		t.Run(text, func(t *testing.T) {
			svc, token := mintBudgetToken(t, "root", "mcp:*", "", "budget-parent", maxBudgetText)
			child := legacyChild(t, svc, token, "budget_limit = "+text)
			info, err := (&MacaroonAuthenticator{Service: svc}).Verify(ctx, child)
			if err != nil {
				t.Fatalf("budget_limit = %s refused: %v", text, err)
			}
			if info.BudgetLimit != want {
				t.Fatalf("budget_limit = %s read as %d, want %d", text, info.BudgetLimit, want)
			}
		})
	}
}

// Earlier releases wrote budgets within 512 of math.MaxInt64 as
// "9223372036854775808.00" (the float64 nearest to them, 2^63). That value is
// outside the int64 range and is refused; this delegator no longer writes it.
func TestFillTokenInfo_LegacyBudgetLimitAboveInt64IsRefused(t *testing.T) {
	svc, token := mintBudgetToken(t, "root", "mcp:*", "", "budget-parent", maxBudgetText)
	child := legacyChild(t, svc, token, "budget_limit = 9223372036854775808.00")
	if _, err := (&MacaroonAuthenticator{Service: svc}).Verify(context.Background(), child); err == nil || !strings.Contains(err.Error(), "invalid budget_limit caveat") {
		t.Fatalf("budget_limit = 9223372036854775808.00 must be refused, got %v", err)
	}
}

// legacyChild writes a child the way earlier releases' delegator did: the
// parent link, a budget_id, the given budget_limit text, then the seal.
func legacyChild(t *testing.T, svc *macaroon.Service, parentToken, budgetLimit string) string {
	t.Helper()
	child, err := svc.Delegate(parentToken, []string{"parent = legacy", "budget_id = del-aaaaaaaaaaaaaaaaaaaaaaaa", budgetLimit})
	if err != nil {
		t.Fatal(err)
	}
	sealed, err := svc.AppendBudgetBind(svc.Encode(child))
	if err != nil {
		t.Fatal(err)
	}
	return svc.Encode(sealed)
}
