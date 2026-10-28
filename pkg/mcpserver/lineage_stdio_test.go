package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/satgate-io/satgate/pkg/governance"
	"github.com/satgate-io/satgate/pkg/macaroon"
)

// Tests for stdio (Proxy.Run) with a lineage revocation checker installed:
// tools/list and every method passed on to the default upstream verify a
// token the request supplies (params._meta.token, or the connection's) and
// check its lineage before listing or forwarding. A request that supplies no
// token keeps the stdio behaviour, and with no checker installed nothing
// changes.

const stdioLineageRoot = "stdio-lineage-test-root"

// unreadableBanState is a checker whose ban state cannot be read.
type unreadableBanState struct{}

func (unreadableBanState) IsAnyBanned(context.Context, []string) (bool, error) {
	return false, errors.New("ban store unreachable")
}

type stdioRig struct {
	t      *testing.T
	svc    *macaroon.Service
	proxy  *Proxy
	client *StdioTransport
	ctx    context.Context
	nextID int
}

// newStdioRig runs a header-mode proxy with a real default upstream (the
// helper process, which answers any passed-on method with {"echo": params})
// on stdio. checker may be nil. wrap, if set, wraps the context Run gets
// (to carry a connection credential).
func newStdioRig(t *testing.T, checker LineageRevocationChecker, wrap func(context.Context) context.Context) *stdioRig {
	t.Helper()
	cfg := &Config{
		Auth:            AuthConfig{Mode: "header", RootKey: stdioLineageRoot},
		Tools:           ToolsConfig{DefaultCost: 1},
		Budget:          BudgetConfig{Limit: 100, FailMode: "closed"},
		Enforcement:     EnforcementConfig{Mode: "control"},
		Logging:         LoggingConfig{Level: "error"},
		Upstreams:       map[string]UpstreamConfig{"fixture": helperUpstream(nil, nil)},
		DefaultUpstream: "fixture",
	}
	proxy, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if checker != nil {
		proxy.SetLineageRevocationChecker(checker)
	}
	svc, err := macaroon.NewService(stdioLineageRoot)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	clientRead, serverWrite := io.Pipe()
	serverRead, clientWrite := io.Pipe()
	runCtx := ctx
	if wrap != nil {
		runCtx = wrap(ctx)
	}
	done := make(chan struct{})
	go func() { defer close(done); _ = proxy.Run(runCtx, NewStdioTransport(serverRead, serverWrite, nil)) }()
	t.Cleanup(func() {
		cancel()
		_ = clientWrite.Close()
		_ = clientRead.Close()
		_ = serverRead.Close()
		_ = serverWrite.Close()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Error("stdio loop did not stop")
		}
	})
	return &stdioRig{t: t, svc: svc, proxy: proxy, client: NewStdioTransport(clientRead, clientWrite, nil), ctx: ctx}
}

func (r *stdioRig) mint() (string, string) {
	r.t.Helper()
	mac, err := r.svc.Mint("mcp:*", time.Now().Add(time.Hour))
	if err != nil {
		r.t.Fatal(err)
	}
	return r.svc.Encode(mac), mac.Signature
}

func (r *stdioRig) delegate(parent string) string {
	r.t.Helper()
	mac, err := r.svc.Delegate(parent, []string{"label = stdio-child"})
	if err != nil {
		r.t.Fatal(err)
	}
	return r.svc.Encode(mac)
}

// call sends one request over stdio; token "" sends none.
func (r *stdioRig) call(method, token string) *Response {
	r.t.Helper()
	r.nextID++
	params := map[string]any{"name": "echo", "arguments": map[string]any{}, "budget": 0, "label": "h", "uri": "file:///x"}
	if token != "" {
		params["_meta"] = map[string]string{"token": token}
	}
	msg := map[string]any{"jsonrpc": "2.0", "id": r.nextID, "method": method, "params": params}
	if err := r.client.WriteMessage(r.ctx, mustJSON(r.t, msg)); err != nil {
		r.t.Fatal(err)
	}
	raw, err := r.client.ReadMessage(r.ctx)
	if err != nil {
		r.t.Fatal(err)
	}
	var resp Response
	if err := json.Unmarshal(raw, &resp); err != nil {
		r.t.Fatal(err)
	}
	return &resp
}

// forwarded reports whether resp is the default upstream's answer.
func forwarded(resp *Response) bool {
	var v map[string]any
	return resp.Error == nil && json.Unmarshal(resp.Result, &v) == nil && v["echo"] != nil
}

// listed reports whether resp is a tool list.
func listed(resp *Response) bool {
	var v struct {
		Tools []json.RawMessage `json:"tools"`
	}
	return resp.Error == nil && json.Unmarshal(resp.Result, &v) == nil && v.Tools != nil
}

// stdioPassedOn are methods the proxy passes on to the default upstream.
var stdioPassedOn = []string{"resources/list", "resources/read", "prompts/list", "prompts/get", "completion/complete", "vendor/custom"}

// Both reviewers' cases: over stdio, with a checker installed, a supplied
// descendant of a banned token gets no tool list and no passed-on method
// (and, as before, no tools/call or helper); when its ban state cannot be
// read every one of them answers revocation-unavailable.
func TestMCPStdio_SuppliedTokenBannedOrUnreadableRefusedOnEveryMethod(t *testing.T) {
	methods := append([]string{MethodToolsList, MethodToolsCall, MethodSatGateDelegate}, stdioPassedOn...)
	for _, state := range []string{"banned", "unreadable"} {
		t.Run(state, func(t *testing.T) {
			gov := governance.NewService(nil)
			var checker LineageRevocationChecker = gov
			if state == "unreadable" {
				checker = unreadableBanState{}
			}
			r := newStdioRig(t, checker, nil)
			root, rootSig := r.mint()
			child := r.delegate(root)
			gov.Ban(rootSig, "test", "admin")
			for _, tok := range []string{root, child} {
				for _, m := range methods {
					resp := r.call(m, tok)
					if resp.Error == nil {
						t.Fatalf("%s served over stdio with a %s lineage", m, state)
					}
					if state == "banned" && !strings.Contains(resp.Error.Message, "token revoked") {
						t.Fatalf("%s: want token revoked, got %+v", m, resp.Error)
					}
					if state == "unreadable" && !isRevocationUnavailableResponse(resp) {
						t.Fatalf("%s: want revocation-unavailable, got %+v", m, resp.Error)
					}
				}
			}
		})
	}
}

// A token supplied on the connection (not in params) is checked the same way.
func TestMCPStdio_ConnectionTokenBannedRefusedOnToolsListAndPassedOn(t *testing.T) {
	gov := governance.NewService(nil)
	svc, _ := macaroon.NewService(stdioLineageRoot)
	mac, _ := svc.Mint("mcp:*", time.Now().Add(time.Hour))
	childMac, err := svc.Delegate(svc.Encode(mac), []string{"label = conn"})
	if err != nil {
		t.Fatal(err)
	}
	child := svc.Encode(childMac)
	gov.Ban(mac.Signature, "test", "admin")
	r := newStdioRig(t, gov, func(ctx context.Context) context.Context { return withTransportToken(ctx, child) })
	for _, m := range append([]string{MethodToolsList}, stdioPassedOn...) {
		if resp := r.call(m, ""); resp.Error == nil || !strings.Contains(resp.Error.Message, "token revoked") {
			t.Fatalf("%s with a banned connection token: %+v", m, resp.Error)
		}
	}
}

// A supplied token that does not verify is refused on tools/list and passed-on
// methods when a checker is installed.
func TestMCPStdio_SuppliedTokenThatDoesNotVerifyRefused(t *testing.T) {
	r := newStdioRig(t, governance.NewService(nil), nil)
	other, _ := macaroon.NewService("some-other-root")
	mac, _ := other.Mint("mcp:*", time.Now().Add(time.Hour))
	forged := other.Encode(mac)
	for _, m := range append([]string{MethodToolsList}, stdioPassedOn...) {
		if resp := r.call(m, forged); resp.Error == nil || !strings.Contains(resp.Error.Message, "authentication failed") {
			t.Fatalf("%s with a token that does not verify: %+v", m, resp.Error)
		}
	}
}

// With a checker installed: a token no ban covers is served; a request with
// no token keeps the stdio behaviour (served, even with bans in place); and a
// banned child leaves its parent served.
func TestMCPStdio_UnbannedAndTokenlessRequestsServed(t *testing.T) {
	gov := governance.NewService(nil)
	r := newStdioRig(t, gov, nil)
	root, _ := r.mint()
	childMac, err := r.svc.Delegate(root, []string{"label = banned-child"})
	if err != nil {
		t.Fatal(err)
	}
	gov.Ban(childMac.Signature, "test", "admin")
	for _, tok := range []string{root, ""} {
		if resp := r.call(MethodToolsList, tok); !listed(resp) {
			t.Fatalf("tools/list (token supplied: %v) refused: %+v", tok != "", resp.Error)
		}
		for _, m := range stdioPassedOn {
			if resp := r.call(m, tok); !forwarded(resp) {
				t.Fatalf("%s (token supplied: %v) not passed on: %+v", m, tok != "", resp.Error)
			}
		}
	}
}

// With no checker installed (an embedding program that sets none), stdio is
// unchanged: tools/list and passed-on methods do not check a supplied token.
func TestMCPStdio_NoCheckerLeavesToolsListAndPassedOnUnchanged(t *testing.T) {
	r := newStdioRig(t, nil, nil)
	other, _ := macaroon.NewService("some-other-root")
	mac, _ := other.Mint("mcp:*", time.Now().Add(time.Hour))
	for _, tok := range []string{other.Encode(mac), ""} {
		if resp := r.call(MethodToolsList, tok); !listed(resp) {
			t.Fatalf("tools/list without a checker refused: %+v", resp.Error)
		}
		for _, m := range stdioPassedOn {
			if resp := r.call(m, tok); !forwarded(resp) {
				t.Fatalf("%s without a checker not passed on: %+v", m, resp.Error)
			}
		}
	}
}
