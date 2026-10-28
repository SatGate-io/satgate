package proxy

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"

	"github.com/satgate-io/satgate/pkg/governance"
)

// heldLineageStore answers BannedAmong from Redis and, once armed, waits for
// release before answering (read is closed when the store has been read).
type heldLineageStore struct {
	*governance.RedisStore
	armed         atomic.Bool
	read, release chan struct{}
}

func (s *heldLineageStore) BannedAmong(ctx context.Context, sigs []string) ([]bool, error) {
	out, err := s.RedisStore.BannedAmong(ctx, sigs)
	if err == nil && s.armed.CompareAndSwap(true, false) {
		close(s.read)
		select {
		case <-s.release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return out, err
}

// A request with a new descendant of a token whose answer is cached: the
// descendant is read from the store while the ancestor's cached answer ages.
// remote: the ancestor is banned on another gateway just after it was cached,
// and the read ends 2.45s after the ancestor was looked up; the request must
// not be served on that answer. local: the ancestor is banned on this
// gateway while the descendant is read; the request is refused.
func TestRevocation_MixedCachedAndReadLineageHonoursAncestorBan(t *testing.T) {
	for _, tc := range []struct {
		name  string
		local bool
	}{{"remote_ban_cached_answer_aged_past_ttl", false}, {"local_ban_during_read", true}} {
		t.Run(tc.name, func(t *testing.T) {
			mr := miniredis.RunT(t)
			up, hits := countingUpstream(t)
			ga := newRedisGovernance(t, mr.Addr())
			defer ga.Close()
			st, err := governance.NewRedisStore(&governance.RedisConfig{Addr: mr.Addr()})
			if err != nil {
				t.Fatal(err)
			}
			held := &heldLineageStore{RedisStore: st, read: make(chan struct{}), release: make(chan struct{})}
			gb := governance.NewService(held)
			defer gb.Close()
			a := newRevocationGateway(t, up.URL, ga)
			b := newRevocationGateway(t, up.URL, gb)

			root, sig := mintRoot(t, a)
			child, err := a.macaroonSvc.Delegate(root, []string{"label = new-child"})
			if err != nil {
				t.Fatal(err)
			}
			tok := a.macaroonSvc.Encode(child)

			start := time.Now()
			if code := bearerGet(t, b, "/api/x", root).Code; code != 200 {
				t.Fatalf("control: root before the ban: %d", code)
			}
			held.armed.Store(true)
			if !tc.local {
				banVia(t, a, sig)
				time.Sleep(time.Until(start.Add(1450 * time.Millisecond)))
			}
			done := make(chan int, 1)
			go func() { done <- bearerGet(t, b, "/api/x", tok).Code }()
			<-held.read
			if tc.local {
				banVia(t, b, sig)
			} else {
				time.Sleep(time.Until(start.Add(2450 * time.Millisecond)))
			}
			close(held.release)
			code := <-done
			if code == 200 {
				t.Fatalf("descendant of a banned token served (cached ancestor answer %.2fs old)", time.Since(start).Seconds())
			}
			if code != 401 {
				t.Fatalf("want 401 (token revoked), got %d", code)
			}
			if hits.Load() != 1 {
				t.Fatalf("refusal must not reach the upstream: hits=%d", hits.Load())
			}
		})
	}
}
