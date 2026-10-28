package proxy

import (
	"net/http"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"

	"github.com/satgate-io/satgate/pkg/governance"
)

func TestRevocation_BanReachesAnotherGatewayWithinTheCacheTTL(t *testing.T) {
	mr := miniredis.RunT(t)
	up, _ := countingUpstream(t)
	gwA := newRevocationGateway(t, up.URL, newRedisGovernance(t, mr.Addr()))
	gwB := newRevocationGateway(t, up.URL, newRedisGovernance(t, mr.Addr()))
	root, rootSig := mintRoot(t, gwA)

	// gwB caches "not banned" for this lineage.
	if rec := bearerGet(t, gwB, "/api/x", root); rec.Code != http.StatusOK {
		t.Fatalf("before ban: want 200, got %d", rec.Code)
	}
	banVia(t, gwA, rootSig)
	deadline := time.Now().Add(governance.BanCacheTTL + time.Second)
	for {
		rec := bearerGet(t, gwB, "/api/x", root)
		if rec.Code == http.StatusUnauthorized {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("ban not honoured on the other gateway within %s (last status %d)", governance.BanCacheTTL, rec.Code)
		}
		time.Sleep(100 * time.Millisecond)
	}
}
