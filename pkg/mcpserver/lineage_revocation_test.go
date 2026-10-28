package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"

	"github.com/satgate-io/satgate/pkg/governance"
	"github.com/satgate-io/satgate/pkg/macaroon"
)

// Tests for the lineage ban check of the MCP runtime
// (SetLineageRevocationChecker): a ban written to the gateway's governance
// service covers the banned token and every token delegated from it on every
// transport, before anything is admitted, spent, forwarded or minted, and a
// ban state that cannot be read refuses.

const lineageTestRoot = "lineage-revocation-test-root"

type lineageRig struct {
	t      *testing.T
	svc    *macaroon.Service
	proxy  *Proxy
	router *countingForwardRouter
	srv    *httptest.Server
}

func newLineageRig(t *testing.T, checker LineageRevocationChecker) *lineageRig {
	t.Helper()
	cfg := &Config{
		Server:      ServerConfig{Transport: "sse", Name: "lineage-test", Version: "test"},
		Auth:        AuthConfig{Mode: "header", RootKey: lineageTestRoot},
		Budget:      BudgetConfig{Limit: 100, FailMode: "closed"},
		Tools:       ToolsConfig{DefaultCost: 1},
		Enforcement: EnforcementConfig{Mode: "control"},
		Logging:     LoggingConfig{Level: "error"},
	}
	proxy, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	router := &countingForwardRouter{}
	proxy.SetUpstreamRouter(router)
	if checker != nil {
		proxy.SetLineageRevocationChecker(checker)
	}
	srv := httptest.NewServer(NewSSEServer(proxy, ":0").Handler())
	t.Cleanup(srv.Close)
	svc, err := macaroon.NewService(lineageTestRoot)
	if err != nil {
		t.Fatal(err)
	}
	return &lineageRig{t: t, svc: svc, proxy: proxy, router: router, srv: srv}
}

// mint returns a root token and its signature.
func (r *lineageRig) mint() (string, string) {
	r.t.Helper()
	mac, err := r.svc.Mint("api:*", time.Now().Add(time.Hour))
	if err != nil {
		r.t.Fatal(err)
	}
	return r.svc.Encode(mac), mac.Signature
}

// delegate delegates parent through the macaroon service (verifies parent).
func (r *lineageRig) delegate(parent string, caveats ...string) (string, string) {
	r.t.Helper()
	mac, err := r.svc.Delegate(parent, caveats)
	if err != nil {
		r.t.Fatal(err)
	}
	return r.svc.Encode(mac), mac.Signature
}

// attenuate appends caveats the way a holder does offline.
func (r *lineageRig) attenuate(parent string, caveats ...string) string {
	r.t.Helper()
	mac, err := r.svc.DelegateWithoutVerify(parent, caveats)
	if err != nil {
		r.t.Fatal(err)
	}
	return r.svc.Encode(mac)
}

// fund verifies tok without any ban check, gives its budget credits and
// returns the budget id.
func (r *lineageRig) fund(tok string, credits int64) string {
	r.t.Helper()
	info, err := (&MacaroonAuthenticator{Service: r.svc}).Verify(context.Background(), tok)
	if err != nil {
		r.t.Fatal(err)
	}
	if err := r.proxy.budget.Initialize(context.Background(), info.BudgetID, credits); err != nil {
		r.t.Fatal(err)
	}
	return info.BudgetID
}

func (r *lineageRig) remaining(budgetID string) int64 {
	r.t.Helper()
	rem, err := r.proxy.budget.Remaining(context.Background(), budgetID)
	if err != nil {
		r.t.Fatal(err)
	}
	return rem
}

func delegateBody(id int) map[string]any {
	return map[string]any{"jsonrpc": "2.0", "id": id, "method": MethodSatGateDelegate,
		"params": map[string]any{"budget": 0, "label": "helper", "scope": "api:read"}}
}

func helperTokenIn(body string) string {
	var resp struct {
		Result struct {
			Token string `json:"token"`
		} `json:"result"`
	}
	_ = json.Unmarshal([]byte(body), &resp)
	return resp.Result.Token
}

func rpcErrorIn(t *testing.T, body string) *RPCError {
	t.Helper()
	var resp Response
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		t.Fatalf("decode JSON-RPC answer: %v", err)
	}
	return resp.Error
}

// Reviewer case (standalone MCP, Streamable HTTP): a descendant of a banned
// token is not admitted, spends nothing and reaches no upstream.
func TestMCPLineage_BannedDescendantNotAdmittedChargedOrForwarded(t *testing.T) {
	gov := governance.NewService(nil)
	r := newLineageRig(t, gov)
	root, rootSig := r.mint()
	child, _ := r.delegate(root, "scope = api:*")
	budget := r.fund(child, 10)
	gov.Ban(rootSig, "test", "admin")

	code, _, body := streamablePost(t, r.srv.URL+"/mcp", "", child, initializeBody("2025-03-26"))
	if code != http.StatusForbidden || !strings.Contains(body, "TOKEN_REVOKED") {
		t.Fatalf("initialize with a descendant of a banned token: status %d", code)
	}
	if streamableForwardCount(r.proxy) != 0 || r.remaining(budget) != 10 {
		t.Fatalf("banned descendant: forwards %d, remaining %d", streamableForwardCount(r.proxy), r.remaining(budget))
	}
}

// Reviewer case (standalone MCP): a banned root and its descendant serve no
// tools and mint no helpers, on a session opened before the ban too.
func TestMCPLineage_BannedRootAndDescendantServeNoToolsAndMintNoHelpers(t *testing.T) {
	gov := governance.NewService(nil)
	r := newLineageRig(t, gov)
	root, rootSig := r.mint()
	child, _ := r.delegate(root, "label = child")
	offline := r.attenuate(child, "label = offline")
	for _, tok := range []string{root, child, offline} {
		budget := r.fund(tok, 20)
		sid := mustInitSession(t, r.srv.URL, tok) // before the ban
		gov.Ban(rootSig, "test", "admin")

		code, _, body := streamablePost(t, r.srv.URL+"/mcp", sid, tok, toolCallBody(2, "echo"))
		if e := rpcErrorIn(t, body); code != http.StatusOK || e == nil || e.Code != CodePolicyDenied || !strings.Contains(e.Message, "token revoked") {
			t.Fatalf("tools/call with a banned lineage: status %d error %+v", code, e)
		}
		_, _, body = streamablePost(t, r.srv.URL+"/mcp", sid, tok, delegateBody(3))
		if helperTokenIn(body) != "" {
			t.Fatal("satgate/delegate minted a helper for a banned lineage")
		}
		if e := rpcErrorIn(t, body); e == nil || !strings.Contains(e.Message, "token revoked") {
			t.Fatalf("satgate/delegate with a banned lineage: %+v", e)
		}
		if streamableForwardCount(r.proxy) != 0 || r.remaining(budget) != 20 {
			t.Fatalf("banned lineage: forwards %d, remaining %d", streamableForwardCount(r.proxy), r.remaining(budget))
		}
		gov.Unban(rootSig)
	}
}

// Banning a child leaves its parent and its sibling served.
func TestMCPLineage_ChildBanLeavesParentAndSibling(t *testing.T) {
	gov := governance.NewService(nil)
	r := newLineageRig(t, gov)
	root, _ := r.mint()
	child, childSig := r.delegate(root, "label = a")
	sibling, _ := r.delegate(root, "label = b")
	grandchild := r.attenuate(child, "label = c")
	gov.Ban(childSig, "test", "admin")
	for _, tok := range []string{root, sibling} {
		r.fund(tok, 5)
		sid := mustInitSession(t, r.srv.URL, tok)
		if _, _, body := streamablePost(t, r.srv.URL+"/mcp", sid, tok, toolCallBody(2, "echo")); !strings.Contains(body, `"result"`) {
			t.Fatal("a token the ban does not cover was refused")
		}
	}
	for _, tok := range []string{child, grandchild} {
		if code, _, _ := streamablePost(t, r.srv.URL+"/mcp", "", tok, initializeBody("2025-03-26")); code != http.StatusForbidden {
			t.Fatalf("banned child or its descendant: initialize status %d", code)
		}
	}
	if streamableForwardCount(r.proxy) != 2 {
		t.Fatalf("forwards %d, want 2", streamableForwardCount(r.proxy))
	}
}

// SSE: the connection of a banned lineage is refused; on a connection opened
// before the ban, each message is checked again.
func TestMCPLineage_SSEConnectAndEachMessageRefuseBannedToken(t *testing.T) {
	gov := governance.NewService(nil)
	r := newLineageRig(t, gov)
	root, rootSig := r.mint()
	child, _ := r.delegate(root, "label = sse")
	budget := r.fund(child, 10)

	conn := eaOpenSSE(t, r.srv.URL, child) // before the ban
	defer conn.body.Close()
	gov.Ban(rootSig, "test", "admin")

	req, _ := http.NewRequest(http.MethodGet, r.srv.URL+"/sse", nil)
	req.Header.Set("Authorization", "Bearer "+child)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden || !strings.Contains(string(raw), "TOKEN_REVOKED") {
		t.Fatalf("SSE connect with a banned lineage: status %d", resp.StatusCode)
	}

	if status := conn.post(mustJSON(t, toolCallBody(7, "echo"))); status != http.StatusAccepted {
		t.Fatalf("POST /message status %d", status)
	}
	answer := conn.next()
	if answer.Error == nil || answer.Error.Code != CodePolicyDenied || !strings.Contains(answer.Error.Message, "token revoked") {
		t.Fatalf("SSE tools/call with a banned lineage: %+v", answer.Error)
	}
	if streamableForwardCount(r.proxy) != 0 || r.remaining(budget) != 10 {
		t.Fatalf("SSE banned lineage: forwards %d, remaining %d", streamableForwardCount(r.proxy), r.remaining(budget))
	}
}

// Streamable HTTP resume: a session lost to a restart is not resumed for a
// banned lineage; before the ban the same resume works.
func TestMCPLineage_StreamableResumeRefusesBannedToken(t *testing.T) {
	gov := governance.NewService(nil)
	first := newLineageRig(t, gov)
	root, rootSig := first.mint()
	child, _ := first.delegate(root, "label = resume")
	sid := mustInitSession(t, first.srv.URL, child)

	control := newLineageRig(t, gov)
	control.fund(child, 5)
	if code, _, _ := streamablePost(t, control.srv.URL+"/mcp", sid, child, toolCallBody(2, "echo")); code != http.StatusOK {
		t.Fatalf("resume before the ban: status %d", code)
	}

	gov.Ban(rootSig, "test", "admin")
	after := newLineageRig(t, gov)
	budget := after.fund(child, 5)
	code, _, body := streamablePost(t, after.srv.URL+"/mcp", sid, child, toolCallBody(3, "echo"))
	if code != http.StatusForbidden || !strings.Contains(body, "TOKEN_REVOKED") {
		t.Fatalf("resume with a banned lineage: status %d", code)
	}
	if streamableForwardCount(after.proxy) != 0 || after.remaining(budget) != 5 {
		t.Fatal("a refused resume forwarded or spent")
	}
}

// stdio: a request read by Run with a banned lineage is refused.
func TestMCPLineage_StdioDispatchRefusesBannedToken(t *testing.T) {
	gov := governance.NewService(nil)
	r := newLineageRig(t, gov)
	root, rootSig := r.mint()
	child, _ := r.delegate(root, "label = stdio")
	budget := r.fund(child, 10)
	gov.Ban(rootSig, "test", "admin")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	clientRead, serverWrite := io.Pipe()
	serverRead, clientWrite := io.Pipe()
	client := NewStdioTransport(clientRead, clientWrite, nil)
	go func() { _ = r.proxy.Run(ctx, NewStdioTransport(serverRead, serverWrite, nil)) }()

	for i, msg := range []map[string]any{
		{"jsonrpc": "2.0", "id": 1, "method": MethodToolsCall, "params": map[string]any{"name": "echo", "arguments": map[string]any{}, "_meta": map[string]string{"token": child}}},
		{"jsonrpc": "2.0", "id": 2, "method": MethodSatGateDelegate, "params": map[string]any{"budget": 0, "label": "h", "_meta": map[string]string{"token": child}}},
	} {
		if err := client.WriteMessage(ctx, mustJSON(t, msg)); err != nil {
			t.Fatal(err)
		}
		raw, err := client.ReadMessage(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if e := rpcErrorIn(t, string(raw)); e == nil || !strings.Contains(e.Message, "token revoked") {
			t.Fatalf("stdio request %d with a banned lineage: %+v", i, e)
		}
		if helperTokenIn(string(raw)) != "" {
			t.Fatal("stdio delegate minted a helper")
		}
	}
	if streamableForwardCount(r.proxy) != 0 || r.remaining(budget) != 10 {
		t.Fatal("stdio banned lineage forwarded or spent")
	}
}

// forwardCallerToken: the connection credential of a banned lineage is not
// passed on to the upstream.
func TestMCPLineage_ForwardedCredentialWithheldForBannedToken(t *testing.T) {
	gov := governance.NewService(nil)
	p := newTokenProbe(t, true)
	svc, err := macaroon.NewService("hygiene-test-root")
	if err != nil {
		t.Fatal(err)
	}
	p.proxy.SetAuthenticator(&MacaroonAuthenticator{Service: svc})
	p.proxy.SetLineageRevocationChecker(gov)
	root, err := svc.Mint("api:*", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	childMac, err := svc.Delegate(svc.Encode(root), []string{"label = fwd"})
	if err != nil {
		t.Fatal(err)
	}
	child := svc.Encode(childMac)
	sid := mustInitSession(t, p.srv.URL, child)
	raw := `{"jsonrpc":"2.0","id":510,"method":"resources/list","params":{}}`
	wantForwarded(t, "before the ban", receivedParams(t, "resources/list", streamableCall(t, p.srv.URL, sid, child, raw)), child)
	gov.Ban(root.Signature, "test", "admin")
	wantForwarded(t, "after the ban", receivedParams(t, "resources/list", streamableCall(t, p.srv.URL, sid, child, raw)), "")
}

// The delegator checks the parent again before it mints a helper, so a
// caller that reaches Delegate directly is refused too.
func TestMCPLineage_DelegatorRefusesBannedParentBeforeMint(t *testing.T) {
	gov := governance.NewService(nil)
	r := newLineageRig(t, gov)
	root, rootSig := r.mint()
	child, _ := r.delegate(root, "label = parent")
	mac, err := r.svc.Verify(child)
	if err != nil {
		t.Fatal(err)
	}
	info, err := FillTokenInfo(r.svc, mac, child)
	if err != nil {
		t.Fatal(err)
	}
	if res, err := r.proxy.delegator.Delegate(context.Background(), info, &DelegateParams{Label: "before"}); err != nil || res == nil || res.Token == "" {
		t.Fatalf("delegate before the ban: %v", err)
	}
	gov.Ban(rootSig, "test", "admin")
	res, err := r.proxy.delegator.Delegate(context.Background(), info, &DelegateParams{Label: "after"})
	if !errors.Is(err, errTokenRevoked) || res != nil {
		t.Fatalf("delegate of a banned lineage: res %v err %v", res != nil, err)
	}
}

// The ban state cannot be read (shared store down): every transport refuses
// with the revocation-unavailable answer and nothing is admitted, spent,
// forwarded or minted.
func TestMCPLineage_StoreErrorRefusesOnEveryTransport(t *testing.T) {
	mr := miniredis.RunT(t)
	gov, err := governance.NewServiceWithRedis(&governance.RedisConfig{Addr: mr.Addr()})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = gov.Close() })
	r := newLineageRig(t, gov)
	root, _ := r.mint()
	child, _ := r.delegate(root, "label = outage")
	budget := r.fund(child, 10)
	sid := mustInitSession(t, r.srv.URL, child)
	conn := eaOpenSSE(t, r.srv.URL, child)
	defer conn.body.Close()
	mr.Close()
	// Answers read before the outage are reused for the cache TTL; wait
	// them out.
	time.Sleep(governance.BanCacheTTL)

	// Streamable initialize and SSE connect: HTTP 503.
	if code, _, body := streamablePost(t, r.srv.URL+"/mcp", "", child, initializeBody("2025-03-26")); code != http.StatusServiceUnavailable || !strings.Contains(body, "REVOCATION_UNAVAILABLE") {
		t.Fatalf("initialize with the ban store down: status %d", code)
	}
	req, _ := http.NewRequest(http.MethodGet, r.srv.URL+"/sse", nil)
	req.Header.Set("Authorization", "Bearer "+child)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("SSE connect with the ban store down: status %d", resp.StatusCode)
	}
	// Streamable request on an open session: HTTP 503, JSON-RPC refusal.
	for _, msg := range []map[string]any{toolCallBody(2, "echo"), delegateBody(3)} {
		code, _, body := streamablePost(t, r.srv.URL+"/mcp", sid, child, msg)
		if !isRevocationUnavailableResponse(&Response{Error: rpcErrorIn(t, body)}) || code != http.StatusServiceUnavailable {
			t.Fatalf("streamable %v with the ban store down: status %d", msg["method"], code)
		}
		if helperTokenIn(body) != "" {
			t.Fatal("a helper was minted with the ban store down")
		}
	}
	// Streamable resume on another instance: HTTP 503.
	other := newLineageRig(t, gov)
	if code, _, _ := streamablePost(t, other.srv.URL+"/mcp", sid, child, toolCallBody(4, "echo")); code != http.StatusServiceUnavailable {
		t.Fatalf("resume with the ban store down: status %d", code)
	}
	// SSE message: the JSON-RPC refusal on the stream.
	if status := conn.post(mustJSON(t, toolCallBody(5, "echo"))); status != http.StatusAccepted {
		t.Fatalf("POST /message status %d", status)
	}
	if answer := conn.next(); !isRevocationUnavailableResponse(answer) {
		t.Fatalf("SSE tools/call with the ban store down: %+v", answer.Error)
	}
	// Direct dispatch (the path stdio takes).
	direct, err := r.proxy.handleRequest(context.Background(), &Request{JSONRPC: "2.0", ID: json.RawMessage(`6`), Method: MethodToolsCall,
		Params: json.RawMessage(`{"name":"echo","arguments":{},"_meta":{"token":"` + child + `"}}`)})
	if err != nil || !isRevocationUnavailableResponse(direct) {
		t.Fatalf("direct tools/call with the ban store down: %v", err)
	}
	if streamableForwardCount(r.proxy) != 0 || streamableForwardCount(other.proxy) != 0 || r.remaining(budget) != 10 {
		t.Fatal("a request was forwarded or charged with the ban store down")
	}
}

// A ban written through the gateway (its own governance service on the same
// Redis) is honoured by the MCP runtime's separate service.
func TestMCPLineage_BanWrittenByGatewayOnSharedStoreHonoured(t *testing.T) {
	mr := miniredis.RunT(t)
	gateway, err := governance.NewServiceWithRedis(&governance.RedisConfig{Addr: mr.Addr()})
	if err != nil {
		t.Fatal(err)
	}
	store, err := governance.NewRedisStore(&governance.RedisConfig{Addr: mr.Addr()})
	if err != nil {
		t.Fatal(err)
	}
	mcp := governance.NewService(store)
	t.Cleanup(func() { _ = gateway.Close(); _ = mcp.Close() })
	r := newLineageRig(t, mcp)
	root, rootSig := r.mint()
	child := r.attenuate(root, "label = shared")
	if err := gateway.BanContext(context.Background(), rootSig, "test", "admin"); err != nil {
		t.Fatal(err)
	}
	if code, _, _ := streamablePost(t, r.srv.URL+"/mcp", "", child, initializeBody("2025-03-26")); code != http.StatusForbidden {
		t.Fatalf("ban written by the gateway on the shared store: initialize status %d", code)
	}
}

// With sign every action on, the refusal of a banned lineage is signed as
// token_revoked, and nothing is spent.
func TestMCPLineage_SignEveryActionSignsTokenRevokedRefusal(t *testing.T) {
	e := newEAEnv(t, eaOpts{tenantCtx: true, window: true})
	eaSetSwitch(e, true)
	mac, err := e.svc.Verify(e.token)
	if err != nil {
		t.Fatal(err)
	}
	gov := governance.NewService(nil)
	gov.Ban(mac.Signature, "test", "admin")
	e.proxy.SetLineageRevocationChecker(gov)
	r := e.call(1, "paid_read", "")
	if r.Error == nil || r.Error.Code != CodePolicyDenied || !strings.Contains(r.Error.Message, "token revoked") {
		t.Fatalf("answer = %s", eaShape(r))
	}
	found := false
	for _, s := range e.base.signed() {
		found = found || s == "denied/token_revoked/TOKEN_REVOKED"
	}
	if !found {
		t.Fatalf("signed %v, want a token_revoked refusal", e.base.signed())
	}
	if e.remaining() != eaCredits {
		t.Fatalf("remaining %d, want %d", e.remaining(), eaCredits)
	}
}

// With no checker set (the default for an embedding program) nothing
// changes: the runtime does not read any ban list.
func TestMCPLineage_NoCheckerLeavesBehaviourUnchanged(t *testing.T) {
	gov := governance.NewService(nil)
	r := newLineageRig(t, nil)
	root, rootSig := r.mint()
	gov.Ban(rootSig, "test", "admin")
	r.fund(root, 5)
	sid := mustInitSession(t, r.srv.URL, root)
	if _, _, body := streamablePost(t, r.srv.URL+"/mcp", sid, root, toolCallBody(2, "echo")); !strings.Contains(body, `"result"`) {
		t.Fatal("without a checker a token was refused")
	}
}
