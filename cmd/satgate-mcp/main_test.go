package main

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"

	"github.com/satgate-io/satgate/pkg/governance"
	"github.com/satgate-io/satgate/pkg/mcpserver"
)

// With revocation.redis.addr set, satgate-mcp reads the bans every gateway on
// that Redis writes.
func TestNewRevocationService_SharedStoreWhenConfigured(t *testing.T) {
	mr := miniredis.RunT(t)
	cfg := &mcpserver.Config{Revocation: mcpserver.RevocationConfig{Redis: mcpserver.RevocationRedisConfig{Addr: mr.Addr()}}}
	svc, err := newRevocationService(cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = svc.Close() })
	gateway, err := governance.NewServiceWithRedis(&governance.RedisConfig{Addr: mr.Addr()})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = gateway.Close() })
	if err := gateway.BanContext(context.Background(), "sig-gw", "test", "admin"); err != nil {
		t.Fatal(err)
	}
	if banned, err := svc.IsAnyBanned(context.Background(), []string{"sig-gw"}); err != nil || !banned {
		t.Fatalf("ban written by a gateway: banned=%v err=%v", banned, err)
	}
}

// With no shared store, bans are kept in this process's memory.
func TestNewRevocationService_InMemoryWhenNoneConfigured(t *testing.T) {
	svc, err := newRevocationService(&mcpserver.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := svc.GetStore().(*governance.MemoryStore); !ok {
		t.Fatalf("store %T, want *governance.MemoryStore", svc.GetStore())
	}
}

// A configured store that cannot be reached is an error, never a silent
// fallback to memory.
func TestNewRevocationService_UnreachableStoreIsAnError(t *testing.T) {
	mr := miniredis.RunT(t)
	addr := mr.Addr()
	mr.Close()
	cfg := &mcpserver.Config{Revocation: mcpserver.RevocationConfig{Redis: mcpserver.RevocationRedisConfig{Addr: addr}}}
	if svc, err := newRevocationService(cfg); err == nil {
		_ = svc.Close()
		t.Fatal("an unreachable shared store was accepted")
	}
}
