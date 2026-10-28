package main

import (
	"testing"

	"github.com/alicebob/miniredis/v2"

	"github.com/satgate-io/satgate/pkg/config"
)

func TestNewGovernanceService_RedisSharesBansAcrossGateways(t *testing.T) {
	mr := miniredis.RunT(t)
	cfg := &config.Config{Redis: &config.RedisConfig{Enabled: true, Addr: mr.Addr()}}
	a, err := newGovernanceService(cfg)
	if err != nil {
		t.Fatal(err)
	}
	b, err := newGovernanceService(cfg)
	if err != nil {
		t.Fatal(err)
	}
	a.Ban("sig-shared", "test", "admin")
	if !b.IsBanned("sig-shared") {
		t.Fatal("gateway on the same Redis does not honour the ban")
	}
}

func TestNewGovernanceService_WithoutRedisIsInMemory(t *testing.T) {
	a, err := newGovernanceService(&config.Config{})
	if err != nil {
		t.Fatal(err)
	}
	b, _ := newGovernanceService(&config.Config{})
	a.Ban("sig-local", "test", "admin")
	if !a.IsBanned("sig-local") || b.IsBanned("sig-local") {
		t.Fatal("in-memory governance is not per process")
	}
}

func TestNewGovernanceService_UnreachableRedisIsAnError(t *testing.T) {
	mr := miniredis.RunT(t)
	addr := mr.Addr()
	mr.Close()
	if _, err := newGovernanceService(&config.Config{Redis: &config.RedisConfig{Enabled: true, Addr: addr}}); err == nil {
		t.Fatal("unreachable Redis did not fail startup")
	}
}
