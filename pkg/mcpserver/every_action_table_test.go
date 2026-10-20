package mcpserver

// The master table for "sign every action".
//
// Each row is one request an agent key can send, written so that it runs the
// same on this branch and on the main branch it was cut from: it uses only
// calls that exist on both. The branch-only parts (the switch, the window
// recorder) live in every_action_test.go.
//
// One table, three uses:
//   - switch on, signer up: an included row yields exactly the receipts it
//     lists and an excluded row yields none;
//   - switch on, signer down: nothing included is passed on, the agent gets
//     proof_unavailable and nothing is debited;
//   - switch off: every answer, in both signer states, equals the golden file
//     made by running this same table on the main branch.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/satgate-io/satgate/pkg/macaroon"
	"github.com/satgate-io/satgate/pkg/toolcontrols"
)

// eaRecorder is an evidence recorder that can be told to fail. It records every
// decision it was asked to sign, and the ones it signed.
type eaRecorder struct {
	mu            sync.Mutex
	preflights    int
	attempts      []MCPDecision
	decisions     []MCPDecision // signed
	failPreflight bool
	failRecord    bool
}

func (r *eaRecorder) Preflight(context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.preflights++
	if r.failPreflight {
		return errors.New("signer down")
	}
	return nil
}

func (r *eaRecorder) RecordMCPDecision(_ context.Context, d MCPDecision) (*MCPEvidence, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.attempts = append(r.attempts, d)
	if r.failRecord {
		return nil, errors.New("signer down")
	}
	r.decisions = append(r.decisions, d)
	n := len(r.decisions)
	return &MCPEvidence{
		ReceiptID:      fmt.Sprintf("rcpt_%d", n),
		ReceiptHash:    fmt.Sprintf("sha256:%d", n),
		EvidencePackID: fmt.Sprintf("ep_%d", n),
		EvidenceURL:    fmt.Sprintf("https://issuer.example/v1/evidence/evid_%d", n),
	}, nil
}

func (r *eaRecorder) setDown(down bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.failPreflight, r.failRecord = down, down
}

// signed lists what was signed as "decision/reason[/code]".
func (r *eaRecorder) signed() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []string
	for _, d := range r.decisions {
		out = append(out, eaLabel(d))
	}
	return out
}

func eaLabel(d MCPDecision) string {
	s := d.Decision + "/" + d.DecisionReason
	if d.DenialCode != "" {
		s += "/" + d.DenialCode
	}
	return s
}

// eaEnv is one row's proxy and everything a row can look at afterwards.
type eaEnv struct {
	t         *testing.T
	proxy     *Proxy
	rr        *scriptedRouter
	rec       EvidenceRecorder
	base      *eaRecorder
	store     *toolcontrols.MemoryStore
	approvals *toolcontrols.MemoryApprovalStore
	clock     time.Time
	svc       *macaroon.Service
	token     string
	ctx       context.Context
	budgetID  string
	credits   int64
	window    bool
	// sender, when set, carries a request to the proxy by another road (a real
	// HTTP transport) and brings the answer back. Nil means a direct call.
	sender func(*Request) *Response
}

type eaOpts struct {
	observe   bool
	scope     string
	noBudget  bool
	approvals bool // chain an approval gate behind the spending-limit gate
	gate      bool // install the spending-limit gate
	caveats   []func(t *testing.T) string
	expired   bool
	tenantCtx bool
	revoked   bool
	limited   bool
	badPrice  bool
	brokenBud bool
	garbage   bool
	noKey     bool
	defaultUp bool
	failOpen  bool // the budget store's fail mode is "open"
	charge    bool // the proxy decides under charge mode (l402)
	renamed   bool // paid_read is a renamed tool of the tenant
	window    bool // set by the runner: sign refusals that repeat through a window recorder
}

const (
	eaTenant  = "tenant-ea"
	eaBudget  = "budget-ea"
	eaCredits = int64(50)
	eaScope   = "paid_read,free_read,big_order,place_crypto_order,place_equity_order"
)

// eaUpstreamTransport answers every request to the default service with an ok
// result, the way a connected service would.
type eaUpstreamTransport struct {
	client *UpstreamClient
	mu     sync.Mutex
	writes int
}

func (u *eaUpstreamTransport) ReadMessage(ctx context.Context) (json.RawMessage, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}
func (u *eaUpstreamTransport) Close() error { return nil }
func (u *eaUpstreamTransport) WriteMessage(_ context.Context, msg json.RawMessage) error {
	var req Request
	if err := json.Unmarshal(msg, &req); err != nil {
		return err
	}
	u.mu.Lock()
	u.writes++
	u.mu.Unlock()
	if ch, ok := u.client.pending.Load(string(req.ID)); ok {
		ch.(chan *Response) <- &Response{JSONRPC: "2.0", ID: req.ID, Result: json.RawMessage(`{"resources":[]}`)}
	}
	return nil
}

func (u *eaUpstreamTransport) count() int {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.writes
}

// eaRenamedRouter is the scripted router for a tenant whose paid_read tool is
// listed under a name other than its own.
type eaRenamedRouter struct{ *scriptedRouter }

func (eaRenamedRouter) IsRenamedToolForTenant(_ context.Context, _, tool string) bool {
	return tool == "paid_read"
}

func newEAEnv(t *testing.T, o eaOpts) *eaEnv {
	t.Helper()
	cfg := &Config{
		Server:      ServerConfig{Transport: "sse", Name: "every-action", Version: "test"},
		Auth:        AuthConfig{Mode: "header", RootKey: argTestRoot},
		Budget:      BudgetConfig{Limit: 100, FailMode: "closed"},
		Tools:       ToolsConfig{DefaultCost: 1, Costs: map[string]int64{"free_read": 0, "big_order": 60}},
		Enforcement: EnforcementConfig{Mode: "control"},
		Logging:     LoggingConfig{Level: "error"},
	}
	if o.observe {
		cfg.Enforcement.Mode = "observe"
	}
	if o.failOpen {
		cfg.Budget.FailMode = "open"
	}
	proxy, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	rr := &scriptedRouter{}
	if o.renamed {
		proxy.SetUpstreamRouter(eaRenamedRouter{rr})
	} else {
		proxy.SetUpstreamRouter(rr)
	}
	base := &eaRecorder{}
	e := &eaEnv{t: t, proxy: proxy, rr: rr, base: base, rec: base, ctx: context.Background(), budgetID: eaBudget, credits: eaCredits, window: o.window}
	if o.tenantCtx {
		e.ctx = context.WithValue(e.ctx, CtxTenantID, eaTenant)
	}
	scope := o.scope
	if scope == "" {
		scope = eaScope
	}
	budget := eaBudget
	if o.noBudget {
		budget = ""
	}
	limit := ""
	if !o.noBudget {
		limit = "50"
	}
	e.svc, e.token = mintBudgetToken(t, argTestRoot, scope, eaTenant, budget, limit)
	for _, c := range o.caveats {
		e.token = appendCaveat(t, e.svc, e.token, c(t))
	}
	if o.expired {
		mac, err := e.svc.Mint(scope, time.Now().Add(-time.Hour))
		if err != nil {
			t.Fatal(err)
		}
		e.token = e.svc.Encode(mac)
	}
	if o.garbage {
		e.token = "not-a-token"
	}
	if o.noKey {
		e.token = ""
	}
	if o.noBudget {
		// A key with no budget caveat: its budget is its own token id.
		if info, err := proxy.auth.Verify(context.Background(), e.token); err == nil {
			e.budgetID = info.BudgetID
			_ = proxy.budget.Initialize(context.Background(), e.budgetID, eaCredits)
		}
	} else {
		_ = proxy.budget.Initialize(context.Background(), eaBudget, eaCredits)
	}
	proxy.SetEvidenceRecorder(base)
	if o.gate || o.approvals {
		e.store = toolcontrols.NewMemoryStore(nil)
		spend := newTestSpendGate(e.store)
		if o.approvals {
			e.approvals = toolcontrols.NewMemoryApprovalStore()
			e.clock = time.Now().UTC()
			ag := NewApprovalGate(e.approvals, &fakeNotifier{})
			ag.Synchronous = true
			ag.now = func() time.Time { return e.clock }
			proxy.SetCallGate(ChainGates(spend, ag))
		} else {
			proxy.SetCallGate(spend)
		}
	}
	if o.revoked {
		proxy.SetRevocationChecker(revokeAll{})
	}
	if o.limited {
		proxy.SetRequestLimiter(&scriptedLimiter{retryAfter: 3 * time.Second})
	}
	if o.badPrice {
		proxy.SetTenantCostResolver(&checkedResolver{plainResolver: plainResolver{price: 1}, err: errors.New("price store down")})
	}
	if o.brokenBud {
		proxy.SetBudgetEnforcer(failingBudget{proxy.budget.(*InMemoryBudgetEnforcer)})
	}
	if o.charge {
		proxy.SetEnforcementMode("l402")
	}
	if o.defaultUp {
		tr := &eaUpstreamTransport{}
		client := &UpstreamClient{name: "default", transport: tr}
		tr.client = client
		proxy.upstream.mu.Lock()
		proxy.upstream.defaultUp = "default"
		if proxy.upstream.clients == nil {
			proxy.upstream.clients = map[string]*UpstreamClient{}
		}
		proxy.upstream.clients["default"] = client
		proxy.upstream.mu.Unlock()
	}
	return e
}

// raw builds a request whose params carry the token (params._meta.token) and
// the given fields.
func (e *eaEnv) raw(id int, method, fields string) *Request {
	params := `{`
	if fields != "" {
		params += fields + `,`
	}
	params += `"_meta":{"token":"` + e.token + `"}}`
	return &Request{JSONRPC: "2.0", ID: json.RawMessage(mustJSONString(id)), Method: method, Params: json.RawMessage(params)}
}

func (e *eaEnv) send(req *Request) *Response {
	e.t.Helper()
	if e.sender != nil {
		return e.sender(req)
	}
	resp, err := e.proxy.handleRequest(e.ctx, req)
	if err != nil {
		// The transports turn a handler error into an internal error reply.
		return NewErrorResponse(req.ID, CodeInternalError, err.Error())
	}
	return resp
}

func (e *eaEnv) call(id int, tool, args string) *Response {
	e.t.Helper()
	return e.send(argCall(id, e.token, tool, args))
}

func (e *eaEnv) remaining() int64 {
	rem, err := e.proxy.budget.Remaining(context.Background(), e.budgetID)
	if err != nil {
		return -1
	}
	return rem
}

func (e *eaEnv) counter() int64 {
	if e.store == nil {
		return 0
	}
	var total int64
	for _, k := range e.store.Keys() {
		total += e.store.TotalOf(k)
	}
	return total
}

// eaRow is one row of the table.
type eaRow struct {
	id       string
	what     string
	included bool
	why      string // excluded rows: the reason a customer would accept
	// observeOpen marks the Observe rows. Today an Observe call goes on when
	// its receipt cannot be made, and this PR keeps that (the owner has not
	// chosen yet), so with the signer down they are the only included rows
	// still passed on.
	observeOpen bool
	// window marks the rows signed through the window recorder (a refusal that
	// repeats): the agent's answer does not change when the signer is down.
	window bool
	// transports marks the rows that are also run through a real Streamable
	// HTTP POST /mcp and an SSE POST /message (every_action_transport_test.go).
	// A row that needs no key, an expired or a revoked key, or a request context
	// the transports do not carry cannot be sent that way.
	transports bool
	opts       eaOpts
	run        func(e *eaEnv) []*Response
	// want is what signing yields with the switch on and the signer up, as
	// "decision/reason[/code]". For a window row it is the opener.
	want []string
}

func eaToolCall(id int, tool, args string) func(e *eaEnv) []*Response {
	return func(e *eaEnv) []*Response { return []*Response{e.call(id, tool, args)} }
}

func eaMethod(method, fields string) func(e *eaEnv) []*Response {
	return func(e *eaEnv) []*Response { return []*Response{e.send(e.raw(1, method, fields))} }
}

func eaArgRule(doc string) func(t *testing.T) string {
	return func(t *testing.T) string { return ruleScopeWord(t, doc) }
}

func eaControl(doc string) func(t *testing.T) string {
	return func(t *testing.T) string { return controlWord(t, doc) }
}

const eaNoUpstream = "denied/policy_denied/NO_DEFAULT_UPSTREAM"

// eaTable is every row of the map of requests an agent key can send, with what
// each yields.
func eaTable() []eaRow {
	allowedFree := []string{"allowed/policy_allowed"}
	spent := []string{"allowed/budget_authorized"}
	return []eaRow{
		// ---- excluded: not an agent's action ----
		{id: "E01", what: "initialize", why: "handshake: no service reached, nothing decided", run: eaMethod(MethodInitialize, "")},
		{id: "E02", what: "ping", why: "handshake", run: eaMethod(MethodPing, "")},
		{id: "E03", what: "notifications/initialized", why: "notification: no reply, nothing decided", run: eaMethod(MethodInitialized, "")},
		{id: "E04", what: "notifications/cancelled", why: "notification", run: eaMethod(MethodCancelled, "")},
		{id: "E05", what: "notifications/progress", why: "notification", run: eaMethod("notifications/progress", "")},
		{id: "E06", what: "tools/list with a key", why: "answered from SatGate's own table: no service reached, nothing spent or changed", run: eaMethod(MethodToolsList, "")},
		{id: "E07", what: "tools/list with no key", why: "no key, and answered from SatGate's own table", opts: eaOpts{noKey: true}, run: eaMethod(MethodToolsList, "")},
		{id: "E08", what: "resources/read with no key", why: "no key: nothing says whose request it is", opts: eaOpts{noKey: true}, run: eaMethod("resources/read", `"uri":"x"`)},
		{id: "E09", what: "tools/call with no key", why: "no key", opts: eaOpts{noKey: true}, run: eaToolCall(1, "paid_read", "")},
		{id: "E10", what: "tools/call with a key that does not verify", why: "a key SatGate did not issue: nothing says whose it is", opts: eaOpts{garbage: true}, run: eaToolCall(1, "paid_read", "")},
		{id: "E11", what: "tools/call with an expired key, no tenant on the request", why: "the account is not proven: no trusted tenant on the request", opts: eaOpts{expired: true}, run: eaToolCall(1, "paid_read", "")},
		{id: "E12", what: "tools/call whose price could not be read", why: "SatGate could not decide", opts: eaOpts{badPrice: true}, run: eaToolCall(1, "paid_read", "")},
		{id: "E13", what: "tools/call on a key with a limit and no gate installed", why: "SatGate could not decide", opts: eaOpts{caveats: []func(*testing.T) string{eaControl(spendDoc200Day)}}, run: eaToolCall(1, "place_crypto_order", order("10"))},
		{id: "E14", what: "tools/call when the budget store does not answer", why: "SatGate could not decide", opts: eaOpts{brokenBud: true}, run: eaToolCall(1, "paid_read", "")},

		// ---- included: tools/call ----
		{id: "T01", what: "allowed, priced, Control", included: true, transports: true, want: spent, run: eaToolCall(1, "paid_read", "")},
		{id: "T02", what: "allowed, free tool, Control", included: true, transports: true, want: allowedFree, run: eaToolCall(1, "free_read", "")},
		{id: "T03", what: "allowed, priced, key with no budget caveat", included: true, transports: true, want: spent, opts: eaOpts{noBudget: true}, run: eaToolCall(1, "paid_read", "")},
		{id: "T04", what: "allowed, priced, Observe, inside the budget", included: true, transports: true, observeOpen: true, want: spent, opts: eaOpts{observe: true}, run: eaToolCall(1, "paid_read", "")},
		{id: "T05", what: "allowed over budget, Observe", included: true, transports: true, observeOpen: true, want: allowedFree, opts: eaOpts{observe: true}, run: eaToolCall(1, "big_order", "")},
		{id: "T06", what: "refused: over budget, Control", included: true, transports: true, want: []string{"denied/budget_exhausted"}, run: eaToolCall(1, "big_order", "")},
		{id: "T07", what: "refused: tool not in the key's scope", included: true, transports: true, want: []string{"denied/policy_denied"}, opts: eaOpts{scope: "free_read"}, run: eaToolCall(1, "paid_read", "")},
		{id: "T08", what: "refused: argument rule", included: true, transports: true, want: []string{"denied/policy_denied/TOOL_ARGUMENT_DENIED"},
			opts: eaOpts{caveats: []func(*testing.T) string{eaArgRule(robinhoodCapDoc)}},
			run:  eaToolCall(1, "place_equity_order", `{"type":"limit","quantity":"1","dollar_amount":"5000"}`)},
		{id: "T14", what: "refused: call that cannot be read", included: true, transports: true, want: []string{"denied/policy_denied/INVALID_PARAMS"},
			run: func(e *eaEnv) []*Response {
				return []*Response{e.send(e.raw(1, MethodToolsCall, `"name":123`))}
			}},
		{id: "T15", what: "refused: tool name with the reserved prefix", included: true, transports: true, want: []string{"denied/policy_denied/RESERVED_TOOL_NAME"}, run: eaToolCall(1, "argrules:v1:abc", "")},
		{id: "T16", what: "allowed, priced, service unreachable", included: true, transports: true, want: spent, run: func(e *eaEnv) []*Response {
			e.rr.replies = []func() (*Response, error){forwardFails}
			return []*Response{e.call(1, "paid_read", "")}
		}},
		{id: "T17", what: "allowed, free, service unreachable", included: true, transports: true, want: allowedFree, run: func(e *eaEnv) []*Response {
			e.rr.replies = []func() (*Response, error){forwardFails}
			return []*Response{e.call(1, "free_read", "")}
		}},
		{id: "T18", what: "allowed, free, service answers unknown tool", included: true, transports: true, want: allowedFree, run: func(e *eaEnv) []*Response {
			e.rr.replies = []func() (*Response, error){rpcErrorCode(-32601)}
			return []*Response{e.call(1, "free_read", "")}
		}},
		{id: "T19", what: "allowed, free, service answers isError", included: true, transports: true, want: allowedFree, run: func(e *eaEnv) []*Response {
			e.rr.replies = []func() (*Response, error){isErrorReply}
			return []*Response{e.call(1, "free_read", "")}
		}},
		{id: "T20", what: "refused: spending limit", included: true, transports: true, want: []string{"denied/policy_denied/TOOL_SPEND_LIMIT_DENIED"},
			opts: eaOpts{gate: true, caveats: []func(*testing.T) string{eaControl(spendDoc200Day)}},
			run:  eaToolCall(1, "place_crypto_order", order("300"))},
		{id: "T21", what: "allowed inside a spending limit, priced", included: true, transports: true, want: spent,
			opts: eaOpts{gate: true, caveats: []func(*testing.T) string{eaControl(spendDoc200Day)}},
			run:  eaToolCall(1, "place_crypto_order", order("100"))},
		{id: "T22", what: "order held for the owner's approval", included: true, transports: true, want: []string{"denied/policy_denied/APPROVAL_REQUIRED"},
			opts: eaOpts{approvals: true, caveats: []func(*testing.T) string{eaControl(askAbove50)}},
			run:  eaToolCall(1, "place_crypto_order", order("75"))},
		{id: "T23", what: "order below the approval threshold", included: true, transports: true, want: spent,
			opts: eaOpts{approvals: true, caveats: []func(*testing.T) string{eaControl(askAbove50)}},
			run:  eaToolCall(1, "place_crypto_order", order("10"))},
		{id: "T24", what: "held order, owner approves, agent sends it again", included: true, transports: true,
			want: []string{"denied/policy_denied/APPROVAL_REQUIRED", "allowed/budget_authorized"},
			opts: eaOpts{approvals: true, caveats: []func(*testing.T) string{eaControl(askAbove50)}},
			run: func(e *eaEnv) []*Response {
				first := e.call(1, "place_crypto_order", order("75"))
				for _, row := range e.approvals.Rows() {
					if err := e.approvals.Approve(eaTenant, row.ID, "owner@example.com", e.clock); err != nil {
						e.t.Fatal(err)
					}
				}
				return []*Response{first, e.call(2, "place_crypto_order", order("75"))}
			}},

		// ---- included: every other outcome that lets a call through, or refuses it ----
		{id: "T25", what: "allowed, Control, budget store down and its fail mode is open", included: true, transports: true, want: allowedFree, opts: eaOpts{brokenBud: true, failOpen: true}, run: eaToolCall(1, "paid_read", "")},
		{id: "T26", what: "allowed, Observe, price could not be read (default price)", included: true, transports: true, observeOpen: true, want: spent, opts: eaOpts{observe: true, badPrice: true}, run: eaToolCall(1, "paid_read", "")},
		{id: "T27", what: "allowed, Observe, budget store down", included: true, transports: true, observeOpen: true, want: allowedFree, opts: eaOpts{observe: true, brokenBud: true}, run: eaToolCall(1, "paid_read", "")},
		{id: "T28", what: "refused: renamed tool on a key with money rules", included: true, transports: true, want: []string{"denied/policy_denied/TOOL_NOT_COVERED_BY_LIMITS"},
			opts: eaOpts{renamed: true, gate: true, scope: "*", caveats: []func(*testing.T) string{eaControl(spendDoc200Day)}}, run: eaToolCall(1, "paid_read", "")},
		{id: "T29", what: "allowed, charge mode (nothing debited here)", included: true, transports: true, want: allowedFree, opts: eaOpts{charge: true}, run: eaToolCall(1, "paid_read", "")},

		// ---- included: the other methods ----
		{id: "M08", what: "resources/list, key valid, no default service", included: true, transports: true, want: []string{eaNoUpstream}, run: eaMethod("resources/list", "")},
		{id: "M09", what: "resources/read, key valid, no default service", included: true, transports: true, want: []string{eaNoUpstream}, run: eaMethod("resources/read", `"uri":"x"`)},
		{id: "M10", what: "prompts/list, key valid, no default service", included: true, transports: true, want: []string{eaNoUpstream}, run: eaMethod("prompts/list", "")},
		{id: "M11", what: "prompts/get, key valid, no default service", included: true, transports: true, want: []string{eaNoUpstream}, run: eaMethod("prompts/get", `"name":"x"`)},
		{id: "M13", what: "unknown method, key valid, no default service", included: true, transports: true, want: []string{eaNoUpstream}, run: eaMethod("foo/bar", "")},
		{id: "M13b", what: "unknown method, key valid, default service connected", included: true, transports: true, want: allowedFree, opts: eaOpts{defaultUp: true}, run: eaMethod("resources/list", "")},
		{id: "M14", what: "satgate/budget", included: true, transports: true, want: allowedFree, run: eaMethod(MethodSatGateBudget, "")},
		{id: "M15", what: "budget/check", included: true, transports: true, want: allowedFree, run: eaMethod("budget/check", "")},
		{id: "M16", what: "satgate/delegate, carved", included: true, transports: true, want: allowedFree, run: eaMethod(MethodSatGateDelegate, `"budget":5,"scope":"free_read","label":"helper"`)},
		{id: "M16b", what: "satgate/delegate, refused: parent cannot afford it", included: true, transports: true, want: []string{"denied/policy_denied/DELEGATION_REFUSED"}, run: eaMethod(MethodSatGateDelegate, `"budget":5000,"label":"helper"`)},
		{id: "M16c", what: "satgate/delegate, refused: params cannot be read", included: true, transports: true, want: []string{"denied/policy_denied/INVALID_PARAMS"}, run: eaMethod(MethodSatGateDelegate, `"budget":"many"`)},
		{id: "M17", what: "any authenticated method with a revoked key", included: true, window: true, want: []string{"denied/token_revoked/TOKEN_REVOKED"}, opts: eaOpts{revoked: true}, run: eaToolCall(1, "paid_read", "")},
		{id: "M18", what: "any authenticated method when the limiter says no", included: true, transports: true, window: true, want: []string{"denied/policy_denied/RATE_LIMITED"}, opts: eaOpts{limited: true}, run: eaToolCall(1, "paid_read", "")},
		{id: "M19", what: "expired key, signature verifies, tenant on the request", included: true, window: true, want: []string{"denied/capability_expired/CREDENTIAL_EXPIRED"}, opts: eaOpts{expired: true, tenantCtx: true}, run: eaToolCall(1, "paid_read", "")},
	}
}

// ---- running a row ----

type eaResult struct {
	responses  []*Response
	signed     []string
	attempts   int
	preflights int
	forwarded  int
	debit      int64
	counter    int64
}

// eaRun runs a row once. on is the switch (nil setSwitch means the code has
// none: the main branch). down makes the signer fail before the row starts.
func eaRun(t *testing.T, row eaRow, on, down bool) eaResult {
	t.Helper()
	return eaRunVia(t, row, on, down, nil)
}

// eaRunVia is eaRun with the requests carried by another road: via, when not
// nil, installs a sender on the environment (a real HTTP transport) and returns
// what to run when the row is done.
func eaRunVia(t *testing.T, row eaRow, on, down bool, via func(*testing.T, *eaEnv) func()) eaResult {
	t.Helper()
	opts := row.opts
	opts.window = row.window
	e := newEAEnv(t, opts)
	if on {
		if eaSetSwitch == nil {
			t.Fatal("this build has no switch")
		}
		eaSetSwitch(e, true)
	}
	if via != nil {
		defer via(t, e)()
	}
	e.base.setDown(down)
	before := e.remaining()
	resps := row.run(e)
	e.base.mu.Lock()
	attempts, preflights := len(e.base.attempts), e.base.preflights
	e.base.mu.Unlock()
	forwarded := e.rr.forwarded()
	if row.opts.defaultUp {
		e.proxy.upstream.mu.Lock()
		if c := e.proxy.upstream.clients["default"]; c != nil {
			forwarded += c.transport.(*eaUpstreamTransport).count()
		}
		e.proxy.upstream.mu.Unlock()
	}
	return eaResult{
		responses:  resps,
		signed:     e.base.signed(),
		attempts:   attempts,
		preflights: preflights,
		forwarded:  forwarded,
		debit:      before - e.remaining(),
		counter:    e.counter(),
	}
}

// eaSetSwitch turns the switch on in an environment. It is nil on the main
// branch, which has no switch, and set by every_action_test.go on this one.
var eaSetSwitch func(e *eaEnv, on bool)

// ---- what a response looks like, for comparing ----

var (
	eaTimeRE = regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?Z`)
	eaIDRE   = regexp.MustCompile(`(apr_|del-|sg_)?[0-9a-f]{12,}`)
	eaTokRE  = regexp.MustCompile(`[A-Za-z0-9_\-]{60,}`)
)

// eaShape is a response as bytes with what changes from run to run (clock
// times, random ids, minted keys) replaced by a fixed word.
func eaShape(r *Response) string {
	if r == nil {
		return "<nil>"
	}
	b, _ := json.Marshal(r)
	s := string(b)
	s = eaTimeRE.ReplaceAllString(s, "<time>")
	s = eaTokRE.ReplaceAllString(s, "<token>")
	s = eaIDRE.ReplaceAllString(s, "<id>")
	return s
}

// eaDump is the text the switch-off golden file holds for one run.
func eaDump(row eaRow, res eaResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "  responses:\n")
	for _, r := range res.responses {
		fmt.Fprintf(&b, "    %s\n", eaShape(r))
	}
	signed := append([]string(nil), res.signed...)
	sort.Strings(signed)
	fmt.Fprintf(&b, "  signed: %v\n  asked: %d\n  preflights: %d\n  forwarded: %d\n  debit: %d\n  window_total: %d\n",
		signed, res.attempts, res.preflights, res.forwarded, res.debit, res.counter)
	return b.String()
}

// eaGolden renders the whole table, in both signer states, with the switch off.
func eaGolden(t *testing.T) string {
	t.Helper()
	var b strings.Builder
	for _, row := range eaTable() {
		for _, down := range []bool{false, true} {
			state := "signer up"
			if down {
				state = "signer down"
			}
			fmt.Fprintf(&b, "%s %s (%s)\n%s", row.id, row.what, state, eaDump(row, eaRun(t, row, false, down)))
		}
	}
	return b.String()
}

// TestEveryActionWriteGolden writes the switch-off golden file. It is how the
// file was made, on the main branch, from this table. It does nothing unless
// EA_WRITE_GOLDEN names the file to write.
func TestEveryActionWriteGolden(t *testing.T) {
	path := os.Getenv("EA_WRITE_GOLDEN")
	if path == "" {
		t.Skip("EA_WRITE_GOLDEN not set")
	}
	if err := os.WriteFile(path, []byte(eaGolden(t)), 0o644); err != nil {
		t.Fatal(err)
	}
}
