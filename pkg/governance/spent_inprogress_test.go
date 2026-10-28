package governance_test

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"

	"github.com/satgate-io/satgate/pkg/governance"
)

// Once the count of ended ban store writes made on a gateway is spent (a ban
// made when it is at its maximum), every access decision on that gateway is
// refused as unavailable, including one already in progress when that ban's
// write ends. The tests in spent_inprogress_*_test.go hold one decision's
// store read on each path, make a ban covering the decision's lineage at the
// maximum (spending the count), turn the cache over and release the read:
// the decision must end with the revocation-unavailable answer of its path
// (HTTP 503 REVOCATION_UNAVAILABLE, or the JSON-RPC error with data.error
// "revocation_unavailable"), never the ordinary revoked answer and never an
// allow, with no upstream work, spending, minting or consumed L402 proof.

// spentBoundary is a gateway service on a miniredis store, with its ban count
// at seed, whose next ban read, once armed by cross, is held.
type spentBoundary struct {
	g    *governance.Service
	h    *heldStore
	mr   *miniredis.Miniredis
	seed uint64
}

func newSpentBoundary(t *testing.T, seed uint64) *spentBoundary {
	t.Helper()
	mr := miniredis.RunT(t)
	st, err := governance.NewRedisStore(&governance.RedisConfig{Addr: mr.Addr()})
	if err != nil {
		t.Fatal(err)
	}
	h := &heldStore{RedisStore: st, read: make(chan struct{}), release: make(chan struct{})}
	g := governance.NewService(h)
	t.Cleanup(func() { _ = g.Close() })
	g.SetBanSequenceForTest(seed)
	return &spentBoundary{g: g, h: h, mr: mr, seed: seed}
}

// newSpentBoundaryAtMaximum is newSpentBoundary with the count at its
// maximum: the next ban made here spends it.
func newSpentBoundaryAtMaximum(t *testing.T) *spentBoundary {
	t.Helper()
	return newSpentBoundary(t, math.MaxUint64)
}

// cross runs decide, which must make one access decision that reads the
// store, and holds that read; meanwhile it bans sig here (with the count at
// its maximum this spends it) and turns the cache over, then releases the
// read and waits for decide to return. It fails the test if decide never
// reads the store. Call it once per spentBoundary.
func (b *spentBoundary) cross(t *testing.T, sig string, decide func()) {
	t.Helper()
	b.crossWith(t, sig, true, decide)
}

// crossWith is cross; with turnOver false the cache is not turned over after
// the ban.
func (b *spentBoundary) crossWith(t *testing.T, sig string, turnOver bool, decide func()) {
	t.Helper()
	b.g.TurnOverBanCacheForTest() // no earlier answer is used: the decision reads
	b.h.hold.Store(true)
	done := make(chan struct{})
	go func() { defer close(done); decide() }()
	select {
	case <-b.h.read:
	case <-done:
		t.Fatal("the decision did not read the ban store")
	case <-time.After(10 * time.Second):
		t.Fatal("the decision did not read the ban store in time")
	}
	if err := b.g.BanContext(context.Background(), sig, "test", "admin"); err != nil {
		t.Fatal(err)
	}
	if turnOver {
		b.g.TurnOverBanCacheForTest()
	}
	close(b.h.release)
	select {
	case <-done:
	case <-time.After(20 * time.Second):
		t.Fatal("the decision did not end")
	}
	if _, spent := b.g.BanSequenceForTest(); spent != (b.seed == math.MaxUint64) {
		t.Fatalf("fixture: count spent=%v after the ban from seed %d", spent, b.seed)
	}
}

// errorsIsUnavailable reports whether err refuses a decision as unavailable.
func errorsIsUnavailable(err error) bool {
	return errors.Is(err, governance.ErrRevocationUnavailable)
}
