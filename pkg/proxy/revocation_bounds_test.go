package proxy

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"

	"github.com/satgate-io/satgate/pkg/config"
	"github.com/satgate-io/satgate/pkg/governance"
)

// Tests for the propagation and read-time bounds on the gateway, and for the
// views and the simulation when the shared ban store is involved.

// heldBanStore answers BannedAmong from the real store and then waits for
// release before returning, once.
type heldBanStore struct {
	governance.Store
	once          sync.Once
	read, release chan struct{}
}

func (h *heldBanStore) BannedAmong(ctx context.Context, sigs []string) ([]bool, error) {
	out := make([]bool, len(sigs))
	for i, sig := range sigs {
		b, err := h.Store.IsBanned(ctx, sig)
		if err != nil {
			return nil, err
		}
		out[i] = b
	}
	h.once.Do(func() { close(h.read); <-h.release })
	return out, nil
}

// Two gateways on one Redis. Gateway B starts a lookup for a token, gateway
// A bans the token while B's answer is held for 750ms; at ban+2.1s B refuses
// the token.
func TestRevocation_DelayedLookupDoesNotExtendPropagationToAnotherGateway(t *testing.T) {
	mr := miniredis.RunT(t)
	up, _ := countingUpstream(t)
	aGov := newRedisGovernance(t, mr.Addr())
	t.Cleanup(func() { _ = aGov.Close() })
	store, err := governance.NewRedisStore(&governance.RedisConfig{Addr: mr.Addr()})
	if err != nil {
		t.Fatal(err)
	}
	held := &heldBanStore{Store: store, read: make(chan struct{}), release: make(chan struct{})}
	bGov := governance.NewService(held)
	t.Cleanup(func() { _ = bGov.Close() })
	a := newRevocationGateway(t, up.URL, aGov)
	b := newRevocationGateway(t, up.URL, bGov)
	tok, sig := mintRoot(t, a)

	first := make(chan int)
	go func() { first <- bearerGet(t, b, "/api/x", tok).Code }()
	<-held.read
	banVia(t, a, sig)
	bannedAt := time.Now()
	time.Sleep(750 * time.Millisecond)
	close(held.release)
	if code := <-first; code != http.StatusOK {
		t.Fatalf("the lookup read the store before the ban: want 200, got %d", code)
	}
	time.Sleep(time.Until(bannedAt.Add(2100 * time.Millisecond)))
	if rec := bearerGet(t, b, "/api/x", tok); rec.Code != http.StatusUnauthorized {
		t.Fatalf("%s after the ban another gateway still serves the token: status %d", time.Since(bannedAt), rec.Code)
	}
}

// slowRedis relays TCP to a Redis server and, once slowed, holds every reply
// for delay before passing it on.
type slowRedis struct {
	ln     net.Listener
	target string
	delay  atomic.Int64
	mu     sync.Mutex
	conns  []net.Conn
}

func newSlowRedis(t *testing.T, target string) *slowRedis {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	l := &slowRedis{ln: ln, target: target}
	go l.serve()
	t.Cleanup(func() {
		_ = l.ln.Close()
		l.mu.Lock()
		defer l.mu.Unlock()
		for _, c := range l.conns {
			_ = c.Close()
		}
	})
	return l
}

func (l *slowRedis) serve() {
	for {
		c, err := l.ln.Accept()
		if err != nil {
			return
		}
		s, err := net.Dial("tcp", l.target)
		if err != nil {
			_ = c.Close()
			continue
		}
		l.mu.Lock()
		l.conns = append(l.conns, c, s)
		l.mu.Unlock()
		go func() { _, _ = io.Copy(s, c) }()
		go func() {
			buf := make([]byte, 32<<10)
			for {
				n, err := s.Read(buf)
				if n > 0 {
					if d := time.Duration(l.delay.Load()); d > 0 {
						time.Sleep(d)
					}
					if _, werr := c.Write(buf[:n]); werr != nil {
						return
					}
				}
				if err != nil {
					_ = c.Close()
					return
				}
			}
		}()
	}
}

// Redis answers 3s late: the request is refused with 503 by the 2s read
// deadline and the upstream is not contacted.
func TestRevocation_SlowRedisRefusedWith503WithinReadDeadline(t *testing.T) {
	mr := miniredis.RunT(t)
	link := newSlowRedis(t, mr.Addr())
	up, hits := countingUpstream(t)
	gov := newRedisGovernance(t, link.ln.Addr().String())
	t.Cleanup(func() { _ = gov.Close() })
	gw := newRevocationGateway(t, up.URL, gov)
	tok, _ := mintRoot(t, gw)
	link.delay.Store(int64(3 * time.Second))

	start := time.Now()
	rec := bearerGet(t, gw, "/api/x", tok)
	elapsed := time.Since(start)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("slow Redis: want 503, got %d", rec.Code)
	}
	if elapsed > governance.BanStoreReadTimeout+300*time.Millisecond {
		t.Fatalf("refusal took %s; read deadline is %s", elapsed, governance.BanStoreReadTimeout)
	}
	if hits.Load() != 0 {
		t.Fatal("upstream contacted while the ban state could not be read")
	}
}

// The simulation answers 503 REVOCATION_UNAVAILABLE, like every other API
// endpoint, when the ban state cannot be read; nothing reaches the upstream.
func TestRevocation_AgentSimulateStoreUnavailableAnswers503(t *testing.T) {
	mr := miniredis.RunT(t)
	gov := newRedisGovernance(t, mr.Addr())
	t.Cleanup(func() { _ = gov.Close() })
	up, hits := countingUpstream(t)
	gw := newTestGateway(t, &config.Config{
		Admin: config.AdminConfig{Token: "admin-secret"},
		Cloud: &config.CloudConfig{SSRF: &config.CloudSSRFConfig{AllowPrivateIPs: true}},
	})
	gw.governance = gov
	tok, _ := mintRoot(t, gw)
	staged := agentPost(t, gw, "/api/agent/policy", map[string]string{
		"name": "gated", "kind": "capability", "path_prefix": "/gated", "upstream_url": up.URL, "scope": "api:read",
	})
	var policy agentPolicy
	decode(t, staged, &policy)
	mr.Close()

	rec := agentPost(t, gw, "/api/agent/simulate", map[string]string{"policy_id": policy.ID, "authorization": "Bearer " + tok})
	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if rec.Code != http.StatusServiceUnavailable || body["code"] != "REVOCATION_UNAVAILABLE" {
		t.Fatalf("simulate with store down: want 503 REVOCATION_UNAVAILABLE, got %d %v", rec.Code, body["code"])
	}
	if hits.Load() != 0 {
		t.Fatal("simulation contacted the upstream while the ban state could not be read")
	}
}

func graphVia(t *testing.T, gw *Gateway) (*httptest.ResponseRecorder, governance.GraphResponse) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/governance/graph", nil)
	req.Header.Set("X-Admin-Token", "admin-secret")
	rec := httptest.NewRecorder()
	gw.ServeHTTP(rec, req)
	var graph governance.GraphResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &graph)
	return rec, graph
}

// A token recorded on gateway B and banned on gateway A is shown banned in
// B's graph at once; with the store down B's graph answers 503.
func TestRevocation_GraphShowsBanMadeOnAnotherGateway(t *testing.T) {
	mr := miniredis.RunT(t)
	up, _ := countingUpstream(t)
	aGov := newRedisGovernance(t, mr.Addr())
	bGov := newRedisGovernance(t, mr.Addr())
	t.Cleanup(func() { _ = aGov.Close(); _ = bGov.Close() })
	a := newRevocationGateway(t, up.URL, aGov)
	b := newRevocationGateway(t, up.URL, bGov)
	_, sig := mintRoot(t, b)
	bGov.RegisterMint(sig, "api:*", time.Now().Add(time.Hour))
	banVia(t, a, sig)

	rec, graph := graphVia(t, b)
	if rec.Code != http.StatusOK || len(graph.Nodes) != 1 || graph.Nodes[0].Data.Status != "BANNED" || graph.Stats.Banned != 1 {
		t.Fatalf("graph on another gateway after the ban: %d %+v", rec.Code, graph)
	}

	mr.Close()
	if rec, _ := graphVia(t, b); rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("graph with the store down: want 503, got %d", rec.Code)
	}
}
