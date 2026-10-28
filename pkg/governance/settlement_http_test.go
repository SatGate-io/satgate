package governance_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"

	"github.com/satgate-io/satgate/pkg/config"
	"github.com/satgate-io/satgate/pkg/governance"
	"github.com/satgate-io/satgate/pkg/lightning"
	"github.com/satgate-io/satgate/pkg/macaroon"
	"github.com/satgate-io/satgate/pkg/proxy"
)

// The final allow of an HTTP request rests on answers that are fresh when the
// decision holds the cache lock, not when it started waiting for it. These
// tests drive the gateway through an HTTP listener; the cache lock and the
// clock are reached through the hooks in export_test.go.

type httpRig struct {
	ga, gb *governance.Service // two gateways on one Redis; requests go to gb
	mac    *macaroon.Service
	url    string
	hits   *atomic.Int64
}

func newHTTPRig(t *testing.T, clock func() time.Time) *httpRig {
	t.Helper()
	mr := miniredis.RunT(t)
	newGov := func() *governance.Service {
		g, err := governance.NewServiceWithRedis(&governance.RedisConfig{Addr: mr.Addr()})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = g.Close() })
		return g
	}
	ga, gb := newGov(), newGov()
	if clock != nil {
		gb.SetClockForTest(clock)
	}
	return newHTTPRigWith(t, ga, gb)
}

// newHTTPRigWith serves an HTTP gateway whose decisions use gb; ga is another
// gateway on the same store (or nil).
func newHTTPRigWith(t *testing.T, ga, gb *governance.Service) *httpRig {
	t.Helper()
	var hits atomic.Int64
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(up.Close)
	mac, err := macaroon.NewService("settlement-http-test-root")
	if err != nil {
		t.Fatal(err)
	}
	gw, err := proxy.New(proxy.Options{
		Config: &config.Config{
			Server:    config.ServerConfig{Listen: ":8080"},
			Admin:     config.AdminConfig{Token: "admin-test"},
			Cloud:     &config.CloudConfig{SSRF: &config.CloudSSRFConfig{AllowPrivateIPs: true}},
			Upstreams: map[string]config.Upstream{"backend": {URL: up.URL, Timeout: 10 * time.Second}},
			Routes: []config.Route{{
				Name:     "api-route",
				Match:    config.RouteMatch{PathPrefix: "/api/"},
				Upstream: "backend",
				Policy:   config.RoutePolicy{Kind: "capability"},
			}},
		},
		Macaroon:   mac,
		Governance: gb,
		Lightning:  lightning.NewMockProvider(),
	})
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(gw)
	t.Cleanup(srv.Close)
	return &httpRig{ga: ga, gb: gb, mac: mac, url: srv.URL, hits: &hits}
}

func (r *httpRig) get(t *testing.T, token string) int {
	req, _ := http.NewRequest(http.MethodGet, r.url+"/api/x", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Error(err)
		return 0
	}
	_ = resp.Body.Close()
	return resp.StatusCode
}

func (r *httpRig) mint(t *testing.T) (token, sig string, lineage []string) {
	t.Helper()
	m, err := r.mac.Mint("api:*", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	token = r.mac.Encode(m)
	lineage, err = r.mac.Lineage(m)
	if err != nil {
		t.Fatal(err)
	}
	return token, m.Signature, lineage
}

type fakeClock struct {
	mu   sync.Mutex
	base time.Time
	now  time.Time
}

func (c *fakeClock) Now() time.Time      { c.mu.Lock(); defer c.mu.Unlock(); return c.now }
func (c *fakeClock) set(d time.Duration) { c.mu.Lock(); c.now = c.base.Add(d); c.mu.Unlock() }

// Injected clock: the answer is cached at 0, the token banned on the other
// gateway, the request starts at 1.9s and waits for the cache lock until
// 2.1s. It is refused (401) and does not reach the upstream.
func TestSettlementHTTP_AnswerExpiredDuringCacheLockWaitNotUsedToServe(t *testing.T) {
	base := time.Now()
	clk := &fakeClock{base: base, now: base}
	var armed atomic.Bool
	started := make(chan struct{})
	r := newHTTPRig(t, func() time.Time {
		v := clk.Now()
		if armed.CompareAndSwap(true, false) {
			close(started)
		}
		return v
	})
	tok, sig, _ := r.mint(t)
	if code := r.get(t, tok); code != http.StatusOK {
		t.Fatalf("control before the ban: %d", code)
	}
	if err := r.ga.BanContext(context.Background(), sig, "test", "admin"); err != nil {
		t.Fatal(err)
	}
	clk.set(1900 * time.Millisecond)
	armed.Store(true)
	release := r.gb.HoldCacheLockForTest()
	done := make(chan int, 1)
	go func() { done <- r.get(t, tok) }()
	<-started
	clk.set(2100 * time.Millisecond)
	release()
	if code := <-done; code != http.StatusUnauthorized {
		t.Fatalf("banned token after the cached answer expired during the lock wait: want 401, got %d", code)
	}
	if n := r.hits.Load(); n != 1 {
		t.Fatalf("upstream hits: want 1 (the control), got %d", n)
	}
}

// The same with the real clock (the security reviewer's case): the cached
// lineage answer is 1.85s old when the request starts, 2.1s old when the
// decision gets the cache lock.
func TestSettlementHTTP_RealClock_FinalFreshnessAfterCacheLockWait(t *testing.T) {
	var armed atomic.Bool
	started := make(chan struct{})
	r := newHTTPRig(t, func() time.Time {
		v := time.Now()
		if armed.CompareAndSwap(true, false) {
			close(started)
		}
		return v
	})
	tok, sig, lineage := r.mint(t)
	if code := r.get(t, tok); code != http.StatusOK {
		t.Fatalf("control before the ban: %d", code)
	}
	at := r.gb.OldestCachedAnswerForTest(lineage)
	if err := r.ga.BanContext(context.Background(), sig, "test", "admin"); err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Until(at.Add(1850 * time.Millisecond)))
	armed.Store(true)
	release := r.gb.HoldCacheLockForTest()
	done := make(chan int, 1)
	go func() { done <- r.get(t, tok) }()
	<-started
	time.Sleep(time.Until(at.Add(2100 * time.Millisecond)))
	release()
	if code := <-done; code == http.StatusOK || r.hits.Load() != 1 {
		t.Fatalf("banned token served after its cached answer expired: status=%d upstream hits=%d", code, r.hits.Load())
	}
}

// Injected clock: fresh answers, but the request waits for the cache lock
// past its deadline (BanStoreReadTimeout). It is refused with 503 and does
// not reach the upstream.
func TestSettlementHTTP_DeadlinePassedDuringCacheLockWaitAnswers503(t *testing.T) {
	base := time.Now()
	clk := &fakeClock{base: base, now: base}
	var armed atomic.Bool
	started := make(chan struct{})
	r := newHTTPRig(t, func() time.Time {
		v := clk.Now()
		if armed.CompareAndSwap(true, false) {
			close(started)
		}
		return v
	})
	tok, _, _ := r.mint(t)
	if code := r.get(t, tok); code != http.StatusOK {
		t.Fatalf("control: %d", code)
	}
	clk.set(100 * time.Millisecond)
	armed.Store(true)
	release := r.gb.HoldCacheLockForTest()
	done := make(chan int, 1)
	go func() { done <- r.get(t, tok) }()
	<-started
	clk.set(100*time.Millisecond + governance.BanStoreReadTimeout)
	release()
	if code := <-done; code != http.StatusServiceUnavailable {
		t.Fatalf("decision past its deadline: want 503, got %d", code)
	}
	if n := r.hits.Load(); n != 1 {
		t.Fatalf("upstream hits: want 1 (the control), got %d", n)
	}
}
