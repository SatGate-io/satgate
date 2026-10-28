package governance

import (
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
)

// These tests use only the API that predates lineage revocation (Ban,
// IsBanned, Unban, NewServiceWithRedis, RegisterMint*, GetTokenInfo*) so they
// also run against older code.

func newRedisService(t *testing.T, addr string) *Service {
	t.Helper()
	svc, err := NewServiceWithRedis(&RedisConfig{Addr: addr})
	if err != nil {
		t.Fatalf("redis service: %v", err)
	}
	t.Cleanup(func() { _ = svc.Close() })
	return svc
}

func TestSharedStore_BanOnOneServiceHonouredOnAnother(t *testing.T) {
	mr := miniredis.RunT(t)
	a := newRedisService(t, mr.Addr())
	b := newRedisService(t, mr.Addr()) // started before the ban

	a.Ban("sig-banned-elsewhere", "test", "admin")
	if !b.IsBanned("sig-banned-elsewhere") {
		t.Fatal("ban written by another service on the same store is not honoured")
	}
	if b.IsBanned("sig-not-banned") {
		t.Fatal("unbanned signature reported banned")
	}
}

func TestSharedStore_StoreErrorFailsClosed(t *testing.T) {
	mr := miniredis.RunT(t)
	svc := newRedisService(t, mr.Addr())
	mr.Close()
	if !svc.IsBanned("sig-unknown") {
		t.Fatal("IsBanned allowed a token whose ban state could not be read")
	}
}

func TestTokenInfo_BanCoversRecordedDescendants(t *testing.T) {
	svc := NewService(nil)
	exp := time.Now().Add(time.Hour)
	svc.RegisterMint("root", "api:*", exp)
	svc.RegisterDelegation("child", "api:read", exp, "root")
	svc.RegisterDelegation("grandchild", "api:read", exp, "child")
	svc.RegisterDelegation("sibling", "api:write", exp, "root")

	svc.Ban("child", "test", "admin")
	for _, sig := range []string{"child", "grandchild"} {
		if info := svc.GetTokenInfoBySignature(sig); info == nil || info.Status != "banned" {
			t.Fatalf("%s: want banned, got %+v", sig, info)
		}
	}
	for _, sig := range []string{"root", "sibling"} {
		if info := svc.GetTokenInfoBySignature(sig); info == nil || info.Status != "active" {
			t.Fatalf("%s: want active, got %+v", sig, info)
		}
	}
	banned := 0
	for _, n := range svc.GetGraph().Nodes {
		if n.Data.Status == "BANNED" {
			banned++
		}
	}
	if banned != 2 {
		t.Fatalf("graph: want 2 banned nodes, got %d", banned)
	}
}
