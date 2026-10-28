package proxy

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"

	"github.com/satgate-io/satgate/pkg/governance"
)

// listenerGet sends GET /api/x with token to an HTTP listener and returns the
// status.
func listenerGet(t *testing.T, url, token string) int {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url+"/api/x", nil)
	if err != nil {
		t.Error(err)
		return 0
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Error(err)
		return 0
	}
	_ = resp.Body.Close()
	return resp.StatusCode
}

// fillerFastStore is heldLineageStore that answers "not banned" for the
// filler signatures of turnOverBanCache without a store round trip, so the
// cache turns over quickly while a request's read is held.
type fillerFastStore struct{ *heldLineageStore }

func (s *fillerFastStore) BannedAmong(ctx context.Context, sigs []string) ([]bool, error) {
	if len(sigs) > 0 && strings.HasPrefix(sigs[0], "filler-") {
		return make([]bool, len(sigs)), nil
	}
	return s.heldLineageStore.BannedAmong(ctx, sigs)
}

// turnOverBanCache makes the gateway's ban cache drop what it holds: decisions
// over more signatures, in all, than the cache keeps (100000).
func turnOverBanCache(t *testing.T, g *governance.Service) {
	t.Helper()
	const batch = 10000
	for b := 0; b*batch <= 100000; b++ {
		sigs := make([]string, batch)
		for i := range sigs {
			sigs[i] = fmt.Sprintf("filler-%d-%d", b, i)
		}
		if hit, err := g.IsAnyBanned(context.Background(), sigs); hit || err != nil {
			t.Fatalf("filler decision: banned=%v err=%v", hit, err)
		}
	}
}

// A ban of the root made on this gateway while a request with a descendant
// is reading the lineage from the store, then the ban cache turns over
// before that request settles. Through an HTTP listener: the request in
// flight is refused, and so are the later requests with the descendant
// (the "not banned" answer read before the ban is never cached or used);
// nothing reaches the upstream. Runs on miniredis, and also on a real Redis
// when SATGATE_TEST_REDIS_ADDR is set.
func TestRevocation_CompletedLocalAncestorBanHonouredAfterCacheTurnover(t *testing.T) {
	addrs := map[string]string{"miniredis": ""}
	if addr := os.Getenv("SATGATE_TEST_REDIS_ADDR"); addr != "" {
		addrs["redis"] = addr
	}
	for name, addr := range addrs {
		t.Run(name, func(t *testing.T) {
			if addr == "" {
				addr = miniredis.RunT(t).Addr()
			}
			st, err := governance.NewRedisStore(&governance.RedisConfig{Addr: addr})
			if err != nil {
				t.Fatal(err)
			}
			held := &heldLineageStore{RedisStore: st, read: make(chan struct{}), release: make(chan struct{})}
			g := governance.NewService(&fillerFastStore{held})
			defer g.Close()
			up, hits := countingUpstream(t)
			gw := newRevocationGateway(t, up.URL, g)
			srv := httptest.NewServer(gw)
			defer srv.Close()

			root, sig := mintRoot(t, gw)
			defer g.Unban(sig)
			child, err := gw.macaroonSvc.Delegate(root, []string{"label = turnover"})
			if err != nil {
				t.Fatal(err)
			}
			tok := gw.macaroonSvc.Encode(child)

			held.armed.Store(true)
			done := make(chan int, 1)
			go func() { done <- listenerGet(t, srv.URL, tok) }()
			<-held.read
			banVia(t, gw, sig)
			turnOverBanCache(t, g)
			close(held.release)
			if code := <-done; code != http.StatusUnauthorized {
				t.Fatalf("request in flight during the ban: want 401, got %d", code)
			}
			for i := 0; i < 2; i++ {
				if code := listenerGet(t, srv.URL, tok); code == http.StatusOK {
					t.Fatalf("later request %d served despite the completed local ancestor ban", i+1)
				}
			}
			if n := hits.Load(); n != 0 {
				t.Fatalf("refused requests reached the upstream: hits=%d", n)
			}
		})
	}
}
