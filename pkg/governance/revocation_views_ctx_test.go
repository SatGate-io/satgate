package governance

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
)

// The views with a context read the shared ban state and fail, rather than
// answer, when it cannot be read.
func TestViewsContext_ReadSharedBansAndFailWhenUnavailable(t *testing.T) {
	mr := miniredis.RunT(t)
	a := newRedisService(t, mr.Addr())
	b := newRedisService(t, mr.Addr())
	ctx := context.Background()
	b.RegisterMint("root-c", "api:*", time.Now().Add(time.Hour))
	if err := a.BanContext(ctx, "root-c", "test", "admin"); err != nil {
		t.Fatal(err)
	}

	info, err := b.GetTokenInfoBySignatureContext(ctx, "root-c")
	if err != nil || info == nil || info.Status != "banned" {
		t.Fatalf("token info: %+v, %v", info, err)
	}
	graph, err := b.GetGraphContext(ctx)
	if err != nil || len(graph.Nodes) != 1 || graph.Nodes[0].Data.Status != "BANNED" || graph.Stats.Banned != 1 {
		t.Fatalf("graph: %+v, %v", graph, err)
	}
	all, err := b.GetAllTokenInfoContext(ctx)
	if err != nil || len(all) != 1 || all[0].Status != "banned" {
		t.Fatalf("all token info: %+v, %v", all, err)
	}
	stats, err := b.GetStatsContext(ctx)
	if err != nil || stats["banned"] != 1 || stats["active"] != 0 {
		t.Fatalf("stats: %+v, %v", stats, err)
	}
	bans, err := b.GetBanListContext(ctx)
	if err != nil || len(bans) != 1 {
		t.Fatalf("ban list: %+v, %v", bans, err)
	}

	mr.Close()
	if _, err := b.GetTokenInfoBySignatureContext(ctx, "root-c"); !errors.Is(err, ErrRevocationUnavailable) {
		t.Fatalf("token info with the store down: %v", err)
	}
	if _, err := b.GetGraphContext(ctx); !errors.Is(err, ErrRevocationUnavailable) {
		t.Fatalf("graph with the store down: %v", err)
	}
	if _, err := b.GetAllTokenInfoContext(ctx); !errors.Is(err, ErrRevocationUnavailable) {
		t.Fatalf("all token info with the store down: %v", err)
	}
	if _, err := b.GetStatsContext(ctx); !errors.Is(err, ErrRevocationUnavailable) {
		t.Fatalf("stats with the store down: %v", err)
	}
	if _, err := b.GetBanListContext(ctx); !errors.Is(err, ErrRevocationUnavailable) {
		t.Fatalf("ban list with the store down: %v", err)
	}
}

// With the in-memory store the views read this process's ban list, as before.
func TestViewsContext_MemoryStoreUnchanged(t *testing.T) {
	svc := NewService(nil)
	svc.RegisterMint("root-m", "api:*", time.Now().Add(time.Hour))
	svc.Ban("root-m", "test", "admin")
	graph, err := svc.GetGraphContext(context.Background())
	if err != nil || graph.Nodes[0].Data.Status != "BANNED" {
		t.Fatalf("graph: %+v, %v", graph, err)
	}
	if info := svc.GetTokenInfoBySignature("root-m"); info == nil || info.Status != "banned" {
		t.Fatalf("token info: %+v", info)
	}
}
