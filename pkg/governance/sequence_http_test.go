package governance_test

import (
	"context"
	"errors"
	"math"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/alicebob/miniredis/v2"

	"github.com/satgate-io/satgate/pkg/governance"
)

// heldStore is a Redis store whose next BannedAmong, once armed, answers and
// then waits for release before returning (an answer in flight).
type heldStore struct {
	*governance.RedisStore
	hold          atomic.Bool
	read, release chan struct{}
}

func (h *heldStore) BannedAmong(ctx context.Context, sigs []string) ([]bool, error) {
	v, err := h.RedisStore.BannedAmong(ctx, sigs)
	if h.hold.CompareAndSwap(true, false) {
		close(h.read)
		<-h.release
	}
	return v, err
}

// Sol's HTTP case: a request for a delegated token is reading the store when
// its root is banned on this gateway with the count of ended ban writes at
// its maximum, and the cache then turns over. The request in flight is
// refused 401; the next request is refused 503; the upstream is never
// reached; a decision for an unrelated token is refused as unavailable. With
// an ordinary count the next request is refused 401 and the unrelated token
// allowed (the control).
func TestSettlementHTTP_LocalAncestorBanAtSequenceMaximumNeverServes(t *testing.T) {
	for _, tc := range []struct {
		name string
		seed uint64
		want int
	}{{"ordinary", 17, http.StatusUnauthorized}, {"maximum", math.MaxUint64, http.StatusServiceUnavailable}} {
		t.Run(tc.name, func(t *testing.T) {
			mr := miniredis.RunT(t)
			st, err := governance.NewRedisStore(&governance.RedisConfig{Addr: mr.Addr()})
			if err != nil {
				t.Fatal(err)
			}
			h := &heldStore{RedisStore: st, read: make(chan struct{}), release: make(chan struct{})}
			g := governance.NewService(h)
			t.Cleanup(func() { _ = g.Close() })
			g.SetBanSequenceForTest(tc.seed)
			r := newHTTPRigWith(t, nil, g)

			root, sig, _ := r.mint(t)
			child, err := r.mac.Delegate(root, []string{"label = sequence"})
			if err != nil {
				t.Fatal(err)
			}
			tok := r.mac.Encode(child)

			h.hold.Store(true)
			done := make(chan int, 1)
			go func() { done <- r.get(t, tok) }()
			<-h.read
			if err := g.BanContext(context.Background(), sig, "test", "admin"); err != nil {
				t.Fatal(err)
			}
			g.TurnOverBanCacheForTest()
			close(h.release)
			if code := <-done; code != http.StatusUnauthorized {
				t.Fatalf("request in flight: want 401, got %d", code)
			}
			if code := r.get(t, tok); code != tc.want {
				t.Fatalf("request after the completed local ancestor ban: want %d, got %d", tc.want, code)
			}
			if n := r.hits.Load(); n != 0 {
				t.Fatalf("upstream reached %d times", n)
			}
			hit, err := g.IsAnyBanned(context.Background(), []string{"unrelated"})
			if tc.seed == math.MaxUint64 && (hit || !errors.Is(err, governance.ErrRevocationUnavailable)) {
				t.Fatalf("unrelated decision after the ban at the maximum: banned=%v err=%v, want refused as unavailable", hit, err)
			}
			if tc.seed != math.MaxUint64 && (hit || err != nil) {
				t.Fatalf("unrelated control: banned=%v err=%v", hit, err)
			}
		})
	}
}
