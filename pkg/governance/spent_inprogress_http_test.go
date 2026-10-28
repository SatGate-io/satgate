package governance_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/satgate-io/satgate/pkg/config"
	"github.com/satgate-io/satgate/pkg/governance"
	"github.com/satgate-io/satgate/pkg/lightning"
	"github.com/satgate-io/satgate/pkg/macaroon"
	"github.com/satgate-io/satgate/pkg/proxy"
)

// The HTTP gateway: a decision in progress when the count is spent ends with
// 503 REVOCATION_UNAVAILABLE on every path that checks a token (capability
// route, MCP-enabled route, ping, admin, validate, delegate, an L402 proof,
// mock payment, simulation), with no upstream work, no token minted or
// recorded, and the L402 proof not consumed.

const spentAdminToken = "spent-admin-test"

// sharedReplay is an L402 consumed-proof store that outlives a gateway, so a
// gateway started on it after a restart sees what the earlier one consumed.
type sharedReplay struct {
	mu   sync.Mutex
	used map[string]bool
}

func (s *sharedReplay) Consume(_ context.Context, hash string, _ time.Duration) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.used[hash] {
		return false, nil
	}
	s.used[hash] = true
	return true, nil
}

type spentHTTPRig struct {
	b       *spentBoundary
	mac     *macaroon.Service
	ln      lightning.Provider
	replay  *sharedReplay
	gw      *proxy.Gateway
	hits    *atomic.Int64
	upURL   string
	root    string // a root token; ban rootSig to cover child
	rootSig string
	child   string // delegated from root
}

func newSpentHTTPRig(t *testing.T) *spentHTTPRig {
	t.Helper()
	var hits atomic.Int64
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(up.Close)
	mac, err := macaroon.NewService("spent-inprogress-http-root")
	if err != nil {
		t.Fatal(err)
	}
	r := &spentHTTPRig{b: newSpentBoundaryAtMaximum(t), mac: mac, ln: lightning.NewMockProvider(),
		replay: &sharedReplay{used: map[string]bool{}}, hits: &hits, upURL: up.URL}
	r.gw = r.gateway(t, r.b.g)
	m, err := mac.Mint("api:*", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	r.root, r.rootSig = mac.Encode(m), m.Signature
	c, err := mac.Delegate(r.root, []string{"label = spent-in-progress"})
	if err != nil {
		t.Fatal(err)
	}
	r.child = mac.Encode(c)
	return r
}

// gateway is a gateway on g with the rig's macaroon key, payment provider and
// consumed-proof store.
func (r *spentHTTPRig) gateway(t *testing.T, g *governance.Service) *proxy.Gateway {
	t.Helper()
	gw, err := proxy.New(proxy.Options{
		Config: &config.Config{
			Server:    config.ServerConfig{Listen: ":8080"},
			Admin:     config.AdminConfig{Token: spentAdminToken},
			Cloud:     &config.CloudConfig{SSRF: &config.CloudSSRFConfig{AllowPrivateIPs: true}},
			Upstreams: map[string]config.Upstream{"backend": {URL: r.upURL, Timeout: 10 * time.Second}},
			Routes: []config.Route{
				{Name: "mcp-route", Match: config.RouteMatch{PathPrefix: "/mcp"}, Upstream: "backend",
					Policy: config.RoutePolicy{Kind: "capability"}, MCP: &config.MCPConfig{Enabled: true}},
				{Name: "api-route", Match: config.RouteMatch{PathPrefix: "/api/"}, Upstream: "backend",
					Policy: config.RoutePolicy{Kind: "capability"}},
				{Name: "cheap", Match: config.RouteMatch{PathPrefix: "/cheap/"}, Upstream: "backend",
					Policy: config.RoutePolicy{Kind: "l402", PriceSats: 10}},
			},
		},
		Macaroon:        r.mac,
		Governance:      g,
		Lightning:       r.ln,
		L402ReplayStore: r.replay,
	})
	if err != nil {
		t.Fatal(err)
	}
	return gw
}

// do serves one request on gw. header values are set as given.
func do(gw http.Handler, method, path, body string, header map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range header {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	gw.ServeHTTP(rec, req)
	return rec
}

func bearer(tok string) map[string]string { return map[string]string{"Authorization": "Bearer " + tok} }

var adminHeader = map[string]string{"X-Admin-Token": spentAdminToken}

// challenge asks for an L402 challenge at path and returns its macaroon,
// payment hash and signature.
func (r *spentHTTPRig) challenge(t *testing.T, path string) (mac, hash, sig string) {
	t.Helper()
	rec := do(r.gw, http.MethodGet, path, "", nil)
	if rec.Code != http.StatusPaymentRequired {
		t.Fatalf("challenge: want 402, got %d", rec.Code)
	}
	m := regexp.MustCompile(`macaroon="([^"]+)"`).FindStringSubmatch(rec.Header().Get("WWW-Authenticate"))
	if len(m) != 2 {
		t.Fatal("no macaroon in the challenge")
	}
	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	hash, _ = body["payment_hash"].(string)
	verified, err := r.mac.Verify(m[1])
	if err != nil {
		t.Fatalf("challenge macaroon: %v", err)
	}
	return m[1], hash, verified.Signature
}

// pay pays a challenge through the mock provider and returns the
// Authorization value of the paid proof.
func (r *spentHTTPRig) pay(t *testing.T, mac, hash string) string {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"payment_hash": hash, "macaroon": mac})
	rec := do(r.gw, http.MethodPost, "/api/l402/mock-pay", string(body), adminHeader)
	if rec.Code != http.StatusOK {
		t.Fatalf("mock pay: want 200, got %d", rec.Code)
	}
	var paid map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&paid); err != nil || !strings.HasPrefix(paid["authorization"], "L402 ") {
		t.Fatalf("mock pay: no authorization (%v)", err)
	}
	return paid["authorization"]
}

// wantUnavailable checks the answer of a decision refused as unavailable:
// 503 with the revocation-unavailable body of the path (code
// REVOCATION_UNAVAILABLE on the API endpoints, "Revocation status
// unavailable" on proxied routes), granting nothing.
func wantUnavailable(t *testing.T, what string, rec *httptest.ResponseRecorder) {
	t.Helper()
	body := rec.Body.String()
	marked := strings.Contains(body, "REVOCATION_UNAVAILABLE") || strings.Contains(body, "Revocation status unavailable")
	if rec.Code != http.StatusServiceUnavailable || !marked {
		t.Errorf("%s: status %d, want 503 revocation unavailable", what, rec.Code)
	}
	for _, granted := range []string{`"valid":true`, `"token":`, `"authorization":`, `"evidence_pack_id":`} {
		if strings.Contains(body, granted) {
			t.Errorf("%s: the refusal carries %s", what, granted)
		}
	}
}

func TestSpentHTTP_DecisionInProgressAnswersUnavailableOnEveryPath(t *testing.T) {
	type path struct {
		name string
		// run prepares the request, crosses the boundary with it in
		// progress, and checks what it may not have done.
		run func(t *testing.T, r *spentHTTPRig)
	}
	simple := func(name, method, target string, body func(r *spentHTTPRig) string, header func(r *spentHTTPRig) map[string]string) path {
		return path{name, func(t *testing.T, r *spentHTTPRig) {
			var rec *httptest.ResponseRecorder
			r.b.cross(t, r.rootSig, func() { rec = do(r.gw, method, target, body(r), header(r)) })
			wantUnavailable(t, name+" in progress when the count was spent", rec)
		}}
	}
	none := func(*spentHTTPRig) string { return "" }
	child := func(r *spentHTTPRig) map[string]string { return bearer(r.child) }
	noHeader := func(*spentHTTPRig) map[string]string { return nil }
	paths := []path{
		simple("capability route", http.MethodGet, "/api/x", none, child),
		simple("MCP-enabled route", http.MethodPost, "/mcp", func(*spentHTTPRig) string {
			return `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"echo","arguments":{}}}`
		}, child),
		simple("ping", http.MethodGet, "/api/capability/ping", none, child),
		simple("admin", http.MethodGet, "/api/capability/admin", none, child),
		simple("validate", http.MethodPost, "/api/capability/validate", func(r *spentHTTPRig) string {
			return `{"token":"` + r.child + `"}`
		}, noHeader),
		{"delegate", func(t *testing.T, r *spentHTTPRig) {
			nodes := len(r.b.g.GetGraph().Nodes)
			var rec *httptest.ResponseRecorder
			r.b.cross(t, r.rootSig, func() {
				rec = do(r.gw, http.MethodPost, "/api/capability/delegate",
					`{"parentToken":"`+r.child+`","caveats":["scope = api:read"]}`, nil)
			})
			wantUnavailable(t, "delegate in progress when the count was spent", rec)
			if n := len(r.b.g.GetGraph().Nodes); n != nodes {
				t.Errorf("delegate refused as unavailable recorded a token (%d -> %d nodes)", nodes, n)
			}
		}},
		{"L402 proof", func(t *testing.T, r *spentHTTPRig) {
			mac, hash, sig := r.challenge(t, "/cheap/x")
			auth := r.pay(t, mac, hash)
			var rec *httptest.ResponseRecorder
			r.b.cross(t, sig, func() { rec = do(r.gw, http.MethodGet, "/cheap/x", "", map[string]string{"Authorization": auth}) })
			wantUnavailable(t, "L402 proof in progress when the count was spent", rec)
			if n := r.hits.Load(); n != 0 {
				t.Fatalf("upstream reached %d times", n)
			}
			// After a restart (a new service on the same store) and an
			// unban, the proof is still usable: it was not consumed.
			fresh, err := governance.NewServiceWithRedis(&governance.RedisConfig{Addr: r.b.mr.Addr()})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = fresh.Close() })
			fresh.Unban(sig)
			if rec := do(r.gateway(t, fresh), http.MethodGet, "/cheap/x", "", map[string]string{"Authorization": auth}); rec.Code != http.StatusOK {
				t.Fatalf("L402 proof after the restart: want 200, got %d (consumed while refused?)", rec.Code)
			}
			r.hits.Store(0)
		}},
		{"mock payment", func(t *testing.T, r *spentHTTPRig) {
			mac, hash, sig := r.challenge(t, "/cheap/y")
			body, _ := json.Marshal(map[string]string{"payment_hash": hash, "macaroon": mac})
			var rec *httptest.ResponseRecorder
			r.b.cross(t, sig, func() { rec = do(r.gw, http.MethodPost, "/api/l402/mock-pay", string(body), adminHeader) })
			wantUnavailable(t, "mock payment in progress when the count was spent", rec)
		}},
		{"simulation", func(t *testing.T, r *spentHTTPRig) {
			staged := do(r.gw, http.MethodPost, "/api/agent/policy", fmt.Sprintf(
				`{"name":"spent","kind":"capability","path_prefix":"/spent","upstream_url":%q,"scope":"api:read"}`, r.upURL), adminHeader)
			var policy struct {
				ID string `json:"policy_id"`
			}
			if err := json.NewDecoder(staged.Body).Decode(&policy); err != nil || policy.ID == "" {
				t.Fatalf("stage policy: status %d (%v)", staged.Code, err)
			}
			var rec *httptest.ResponseRecorder
			r.b.cross(t, r.rootSig, func() {
				rec = do(r.gw, http.MethodPost, "/api/agent/simulate",
					`{"policy_id":"`+policy.ID+`","authorization":"Bearer `+r.child+`"}`, adminHeader)
			})
			wantUnavailable(t, "simulation in progress when the count was spent", rec)
		}},
	}
	for _, p := range paths {
		t.Run(strings.ReplaceAll(p.name, " ", "_"), func(t *testing.T) {
			r := newSpentHTTPRig(t)
			p.run(t, r)
			if n := r.hits.Load(); n != 0 {
				t.Errorf("upstream reached %d times", n)
			}
			// A decision that starts now is refused the same way.
			wantUnavailable(t, "a later request", do(r.gw, http.MethodGet, "/api/x", "", bearer(r.child)))
		})
	}
}

// Sol's HTTP case: over an HTTP listener, a request for a delegated token is
// reading the store when its root is banned here with the count at its
// maximum, and the cache turns over; that request and the next are answered
// 503, and the upstream is never reached.
func TestSpentHTTP_RequestInProgressAndNextRequestAnswer503(t *testing.T) {
	b := newSpentBoundaryAtMaximum(t)
	r := newHTTPRigWith(t, nil, b.g)
	root, sig, _ := r.mint(t)
	child, err := r.mac.Delegate(root, []string{"label = crossing"})
	if err != nil {
		t.Fatal(err)
	}
	tok := r.mac.Encode(child)
	var code int
	b.cross(t, sig, func() { code = r.get(t, tok) })
	if code != http.StatusServiceUnavailable {
		t.Errorf("request in progress when the count was spent: status %d, want 503", code)
	}
	if code := r.get(t, tok); code != http.StatusServiceUnavailable {
		t.Errorf("next request: status %d, want 503", code)
	}
	if n := r.hits.Load(); n != 0 {
		t.Errorf("upstream reached %d times", n)
	}
}

// Astra's HTTP case: while a request for a delegated token is reading the
// store, its root is banned here with the count at its maximum and a decision
// for an unrelated lineage longer than the cache bound is made; that decision
// is refused as unavailable, the request in progress is answered 503, and so
// is the next one, with no upstream work.
func TestSpentHTTP_UnrelatedDecisionDuringRequestInProgressAllUnavailable(t *testing.T) {
	b := newSpentBoundaryAtMaximum(t)
	r := newHTTPRigWith(t, nil, b.g)
	root, sig, _ := r.mint(t)
	child, err := r.mac.Delegate(root, []string{"label = eviction"})
	if err != nil {
		t.Fatal(err)
	}
	tok := r.mac.Encode(child)
	b.h.hold.Store(true)
	done := make(chan int, 1)
	go func() { done <- r.get(t, tok) }()
	<-b.h.read
	if err := b.g.BanContext(context.Background(), sig, "test", "admin"); err != nil {
		t.Fatal(err)
	}
	sigs := make([]string, 100001)
	for i := range sigs {
		sigs[i] = fmt.Sprintf("unrelated-%d", i)
	}
	hit, err := b.g.IsAnyBanned(context.Background(), sigs)
	close(b.h.release)
	first := <-done
	if hit || !errorsIsUnavailable(err) {
		t.Errorf("unrelated decision after the count was spent: banned=%v err=%v, want refused as unavailable", hit, err)
	}
	if first != http.StatusServiceUnavailable {
		t.Errorf("request in progress when the count was spent: status %d, want 503", first)
	}
	if code := r.get(t, tok); code != http.StatusServiceUnavailable || r.hits.Load() != 0 {
		t.Errorf("next request: status %d, upstream reached %d times; want 503 and none", code, r.hits.Load())
	}
}
