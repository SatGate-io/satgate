package governance

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
)

func TestIsAnyBanned_MemoryStore(t *testing.T) {
	svc := NewService(nil)
	svc.Ban("ancestor", "test", "admin")
	if banned, err := svc.IsAnyBanned(context.Background(), []string{"ancestor", "self"}); err != nil || !banned {
		t.Fatalf("ban of an ancestor in the lineage: got %v, %v", banned, err)
	}
	if banned, err := svc.IsAnyBanned(context.Background(), []string{"other", "self"}); err != nil || banned {
		t.Fatalf("lineage with no ban: got %v, %v", banned, err)
	}
}

func TestSharedStore_CachedAnswerExpiresWithinTTL(t *testing.T) {
	mr := miniredis.RunT(t)
	a := newRedisService(t, mr.Addr())
	b := newRedisService(t, mr.Addr())
	now := time.Now()
	b.now = func() time.Time { return now }
	ctx := context.Background()

	if banned, err := b.IsAnyBanned(ctx, []string{"root", "child"}); err != nil || banned {
		t.Fatalf("before ban: %v, %v", banned, err)
	}
	a.Ban("root", "test", "admin")

	now = now.Add(BanCacheTTL - time.Millisecond)
	if banned, _ := b.IsAnyBanned(ctx, []string{"root", "child"}); banned {
		t.Fatal("cached answer was not reused within the TTL (test assumption)")
	}
	now = now.Add(time.Millisecond)
	if banned, err := b.IsAnyBanned(ctx, []string{"root", "child"}); err != nil || !banned {
		t.Fatalf("ban made on another service not honoured after the TTL: %v, %v", banned, err)
	}
}

func TestSharedStore_ReadErrorReturnsRevocationUnavailable(t *testing.T) {
	mr := miniredis.RunT(t)
	svc := newRedisService(t, mr.Addr())
	mr.Close()
	_, err := svc.IsAnyBanned(context.Background(), []string{"a", "b"})
	if !errors.Is(err, ErrRevocationUnavailable) {
		t.Fatalf("want ErrRevocationUnavailable, got %v", err)
	}
}

func TestSharedStore_BanWriteFailureStillHonouredHere(t *testing.T) {
	mr := miniredis.RunT(t)
	svc := newRedisService(t, mr.Addr())
	mr.Close()
	if err := svc.BanContext(context.Background(), "sig-y", "test", "admin"); !errors.Is(err, ErrRevocationUnavailable) {
		t.Fatalf("ban with store down: want ErrRevocationUnavailable, got %v", err)
	}
	if banned, err := svc.IsAnyBanned(context.Background(), []string{"sig-y"}); err != nil || !banned {
		t.Fatalf("ban that could not be written is not honoured here: %v, %v", banned, err)
	}
}

func TestSharedStore_UnbanReachesOtherServicesWithinTTL(t *testing.T) {
	mr := miniredis.RunT(t)
	a := newRedisService(t, mr.Addr())
	b := newRedisService(t, mr.Addr())
	now := time.Now()
	a.now = func() time.Time { return now }
	a.Ban("sig-x", "test", "admin")
	if !b.IsBanned("sig-x") {
		t.Fatal("ban not honoured on b")
	}
	if !b.Unban("sig-x") {
		t.Fatal("unban on b of a ban made on a reported no ban")
	}
	now = now.Add(BanCacheTTL)
	if a.IsBanned("sig-x") {
		t.Fatal("unban on b not honoured on a after the TTL")
	}
}
