package proxy

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/satgate-io/satgate/pkg/config"
	"github.com/satgate-io/satgate/pkg/governance"
	"github.com/satgate-io/satgate/pkg/lightning"
)

// Tests that an L402 payment proof identifies exactly one route: route names
// are unique wherever routes are loaded, and a proof is never accepted while
// the served route's name is shared.

// countingReplayStore records Consume calls and otherwise behaves like the
// memory store.
type countingReplayStore struct {
	mu    sync.Mutex
	calls int
	inner L402ReplayStore
}

func newCountingReplayStore() *countingReplayStore {
	return &countingReplayStore{inner: newMemoryReplayStore(time.Minute)}
}

func (s *countingReplayStore) Consume(ctx context.Context, hash string, ttl time.Duration) (bool, error) {
	s.mu.Lock()
	s.calls++
	s.mu.Unlock()
	return s.inner.Consume(ctx, hash, ttl)
}

func (s *countingReplayStore) Calls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

// sameNameSamePriceConfig is two L402 routes at the same price that differ
// only in path. They are given distinct names; tests make them share one.
func sameNameSamePriceConfig(upstreamURL string) *config.Config {
	cfg := newTwoTierL402Config(upstreamURL)
	cfg.Routes[1].Policy.PriceSats = cfg.Routes[0].Policy.PriceSats
	return cfg
}

// Reviewer case (both reviews): two validated routes sharing a name.
func TestL402_DuplicateRouteNamesRefusedByConfigValidation(t *testing.T) {
	cfg := sameNameSamePriceConfig("https://example.com")
	cfg.Routes[1].Name = cfg.Routes[0].Name
	err := cfg.Validate()
	if err == nil {
		t.Fatal("config with two routes named \"cheap\" passed validation")
	}
	if !strings.Contains(err.Error(), `route name "cheap"`) || !strings.Contains(err.Error(), "unique") {
		t.Fatalf("validation error does not name the duplicate: %v", err)
	}

	yaml := `
upstreams:
  backend:
    url: https://example.com
routes:
  - name: cheap
    match: {pathPrefix: /cheap/}
    upstream: backend
    policy: {kind: l402, priceSats: 10}
  - name: cheap
    match: {pathPrefix: /dear/}
    upstream: backend
    policy: {kind: l402, priceSats: 10}
`
	if _, err := config.ParseYAML([]byte(yaml)); err == nil || !strings.Contains(err.Error(), "unique") {
		t.Fatalf("loading a config file with a duplicate route name: want a uniqueness error, got %v", err)
	}
}

func TestL402_DuplicateRouteNamesRefusedByGatewayConstruction(t *testing.T) {
	cfg := sameNameSamePriceConfig(okUpstream(t).URL)
	cfg.Routes[1].Name = cfg.Routes[0].Name
	_, err := New(Options{
		Config:     cfg,
		Macaroon:   newTestMacaroonService(t),
		Governance: governance.NewService(nil),
		Lightning:  lightning.NewMockProvider(),
	})
	if err == nil {
		t.Fatal("gateway was built from routes that share a name")
	}
	if !strings.Contains(err.Error(), `route name "cheap"`) {
		t.Fatalf("construction error does not name the duplicate: %v", err)
	}
}

// Reviewer case (both reviews), over HTTP: a proof paid on one route is not
// accepted on another route that shares its name and price, and the refusal
// does not consume it. The route table is changed after construction, the
// way a path that skips validation would.
func TestL402_ProofRefusedOnOtherRouteWithSameNameAndPrice(t *testing.T) {
	var mu sync.Mutex
	hits := map[string]int{}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		hits[r.URL.Path]++
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(upstream.Close)

	cfg := sameNameSamePriceConfig(upstream.URL)
	gw := newTestGateway(t, cfg)
	store := newCountingReplayStore()
	gw.l402Seen = store
	paid := payL402At(t, gw, "/cheap/x")

	cfg.Routes[1].Name = cfg.Routes[0].Name // both routes now "cheap", both 10 sats
	if code := useL402(t, gw, "/dear/x", paid.authorization()); code == http.StatusOK {
		t.Fatalf("proof paid on /cheap/ was accepted on /dear/, which shares its name and price")
	}
	if code := useL402(t, gw, "/cheap/x", paid.authorization()); code == http.StatusOK {
		t.Fatalf("proof was accepted while its route name was shared")
	}
	// No payment is taken for a route whose name is shared.
	if code := useL402(t, gw, "/dear/x", ""); code == http.StatusPaymentRequired {
		t.Fatalf("a challenge was issued for a route whose name is shared")
	}
	if store.Calls() != 0 {
		t.Fatalf("refused proof reached the consumed-proof store (%d calls)", store.Calls())
	}
	mu.Lock()
	served := hits["/dear/x"] + hits["/cheap/x"]
	mu.Unlock()
	if served != 0 {
		t.Fatalf("upstream served %d refused requests", served)
	}

	cfg.Routes[1].Name = "dear"
	if code := useL402(t, gw, "/cheap/x", paid.authorization()); code != http.StatusOK {
		t.Fatalf("unconsumed proof on its own route once names are unique: want 200, got %d", code)
	}
}

func TestL402_AgentPromotionRefusesRouteNameInUse(t *testing.T) {
	upstream := okUpstream(t)
	cfg := newTwoTierL402Config(upstream.URL)
	gw := newTestGateway(t, cfg)

	staged := agentPost(t, gw, "/api/agent/policy", map[string]string{
		"name": "cheap", "kind": "observe", "path_prefix": "/other", "upstream_url": upstream.URL,
	})
	var policy agentPolicy
	decode(t, staged, &policy)
	if sim := agentPost(t, gw, "/api/agent/simulate", map[string]string{"policy_id": policy.ID}); sim.Code != http.StatusOK {
		t.Fatalf("simulate %d %s", sim.Code, sim.Body.String())
	}
	promoted := agentPost(t, gw, "/api/agent/promote", map[string]string{"policy_id": policy.ID})
	if promoted.Code != http.StatusConflict || !strings.Contains(promoted.Body.String(), "route_name_in_use") {
		t.Fatalf("promoting a route named like an existing route: want 409 route_name_in_use, got %d %s", promoted.Code, promoted.Body.String())
	}
	if n := countRoutesNamed(gw, "cheap"); n != 1 {
		t.Fatalf("route table has %d routes named \"cheap\"", n)
	}
	if live := agentGet(t, gw, "/other"); live.Code != http.StatusNotFound {
		t.Fatalf("refused agent route went live: %d", live.Code)
	}
}

func TestL402_AgentRestoreSkipsRouteNameInUse(t *testing.T) {
	t.Setenv("SATGATE_AGENT_PACK_DIR", t.TempDir())
	upstream := okUpstream(t)
	first := newTestGateway(t, &config.Config{
		Admin: config.AdminConfig{Token: "admin-secret"},
		Cloud: &config.CloudConfig{SSRF: &config.CloudSSRFConfig{AllowPrivateIPs: true}},
	})
	staged := agentPost(t, first, "/api/agent/policy", map[string]string{
		"name": "cheap", "kind": "observe", "path_prefix": "/other", "upstream_url": upstream.URL,
	})
	var policy agentPolicy
	decode(t, staged, &policy)
	agentPost(t, first, "/api/agent/simulate", map[string]string{"policy_id": policy.ID})
	if promoted := agentPost(t, first, "/api/agent/promote", map[string]string{"policy_id": policy.ID}); promoted.Code != http.StatusOK {
		t.Fatalf("promote on a gateway without a \"cheap\" route: %d %s", promoted.Code, promoted.Body.String())
	}

	// A new process whose configuration already has a route named "cheap"
	// does not restore the saved agent route of the same name.
	next := newTestGateway(t, newTwoTierL402Config(upstream.URL))
	if live := agentGet(t, next, "/other"); live.Code != http.StatusNotFound {
		t.Fatalf("saved agent route with a name in use was restored: %d", live.Code)
	}
	if n := countRoutesNamed(next, "cheap"); n != 1 {
		t.Fatalf("route table has %d routes named \"cheap\" after restore", n)
	}
}

func countRoutesNamed(gw *Gateway, name string) int {
	n := 0
	for _, r := range gw.config.Routes {
		if r.Name == name {
			n++
		}
	}
	return n
}
