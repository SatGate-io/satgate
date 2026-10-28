package governance

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// These tests use only API that existed before the read bounds were
// enforced (IsAnyBanned, BanContext, NewRedisStore, the views without a
// context), so they also run against the earlier code.

// heldStore answers BannedAmong from the real store, then waits for release
// before returning, once. read is closed when the store has been read.
type heldStore struct {
	Store
	once          sync.Once
	read, release chan struct{}
}

func newHeldStore(s Store) *heldStore {
	return &heldStore{Store: s, read: make(chan struct{}), release: make(chan struct{})}
}

func (h *heldStore) BannedAmong(ctx context.Context, sigs []string) ([]bool, error) {
	out := make([]bool, len(sigs))
	for i, sig := range sigs {
		b, err := h.Store.IsBanned(ctx, sig)
		if err != nil {
			return nil, err
		}
		out[i] = b
	}
	h.once.Do(func() { close(h.read); <-h.release })
	return out, nil
}

func newMiniRedisStore(t *testing.T, addr string) *RedisStore {
	t.Helper()
	st, err := NewRedisStore(&RedisConfig{Addr: addr})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

// A lookup that read the store before a ban and answered late must not let
// the banned token through for more than BanCacheTTL after the ban (clock
// controlled: lookup at 0, ban during it, answer at 1s, check at 2.1s).
func TestSharedStore_DelayedAnswerDoesNotExtendPropagationBound(t *testing.T) {
	mr := miniredis.RunT(t)
	a := newRedisService(t, mr.Addr())
	held := newHeldStore(newMiniRedisStore(t, mr.Addr()))
	b := NewService(held)
	var mu sync.Mutex
	base := time.Now()
	now := base
	b.now = func() time.Time { mu.Lock(); defer mu.Unlock(); return now }
	set := func(d time.Duration) { mu.Lock(); now = base.Add(d); mu.Unlock() }

	done := make(chan bool)
	go func() {
		hit, err := b.IsAnyBanned(context.Background(), []string{"sig-late"})
		done <- hit || err != nil
	}()
	<-held.read
	if err := a.BanContext(context.Background(), "sig-late", "test", "admin"); err != nil {
		t.Fatal(err)
	}
	set(time.Second)
	close(held.release)
	if <-done {
		t.Fatal("the lookup read the store before the ban; it answers not banned")
	}
	set(BanCacheTTL + 100*time.Millisecond)
	if hit, err := b.IsAnyBanned(context.Background(), []string{"sig-late"}); err != nil || !hit {
		t.Fatalf("2.1s after the ban the token is still allowed: banned=%v err=%v", hit, err)
	}
}

// The same case in real time against Redis: ban during a lookup held for
// 800ms; at ban+2.1s the ban is honoured.
func TestSharedStore_BanDuringSlowLookupHonouredWithinTTLOfBan(t *testing.T) {
	mr := miniredis.RunT(t)
	a := newRedisService(t, mr.Addr())
	held := newHeldStore(newMiniRedisStore(t, mr.Addr()))
	b := NewService(held)

	done := make(chan struct{})
	go func() { defer close(done); _, _ = b.IsAnyBanned(context.Background(), []string{"sig-slow"}) }()
	<-held.read
	if err := a.BanContext(context.Background(), "sig-slow", "test", "admin"); err != nil {
		t.Fatal(err)
	}
	bannedAt := time.Now()
	time.Sleep(800 * time.Millisecond)
	close(held.release)
	<-done
	time.Sleep(time.Until(bannedAt.Add(BanCacheTTL + 100*time.Millisecond)))
	if hit, err := b.IsAnyBanned(context.Background(), []string{"sig-slow"}); err != nil || !hit {
		t.Fatalf("ban not honoured %s after it was written: banned=%v err=%v", time.Since(bannedAt), hit, err)
	}
}

// An answer that arrives when its lookup started BanCacheTTL ago or more is
// refused, not used to allow, and not cached.
func TestSharedStore_AnswerOlderThanTTLIsRefused(t *testing.T) {
	mr := miniredis.RunT(t)
	held := newHeldStore(newMiniRedisStore(t, mr.Addr()))
	b := NewService(held)
	var mu sync.Mutex
	base := time.Now()
	now := base
	b.now = func() time.Time { mu.Lock(); defer mu.Unlock(); return now }

	type res struct {
		hit bool
		err error
	}
	done := make(chan res)
	go func() {
		hit, err := b.IsAnyBanned(context.Background(), []string{"sig-old"})
		done <- res{hit, err}
	}()
	<-held.read
	mu.Lock()
	now = base.Add(BanCacheTTL)
	mu.Unlock()
	close(held.release)
	r := <-done
	if r.hit || !errors.Is(r.err, ErrRevocationUnavailable) {
		t.Fatalf("answer %s old: want refusal with ErrRevocationUnavailable, got banned=%v err=%v", BanCacheTTL, r.hit, r.err)
	}
	b.cacheMu.Lock()
	_, cached := b.banCache["sig-old"]
	b.cacheMu.Unlock()
	if cached {
		t.Fatal("an answer older than the TTL was cached")
	}
}

// A ban made on this gateway while a lookup is in flight wins over the
// lookup's older "not banned" answer, now and afterwards.
func TestSharedStore_LocalBanDuringLookupStillWins(t *testing.T) {
	mr := miniredis.RunT(t)
	held := newHeldStore(newMiniRedisStore(t, mr.Addr()))
	b := NewService(held)

	done := make(chan bool)
	go func() {
		hit, err := b.IsAnyBanned(context.Background(), []string{"sig-local"})
		done <- hit && err == nil
	}()
	<-held.read
	if err := b.BanContext(context.Background(), "sig-local", "test", "admin"); err != nil {
		t.Fatal(err)
	}
	close(held.release)
	if !<-done {
		t.Fatal("lookup in flight during a local ban answered not banned")
	}
	if hit, err := b.IsAnyBanned(context.Background(), []string{"sig-local"}); err != nil || !hit {
		t.Fatalf("local ban lost after the lookup: banned=%v err=%v", hit, err)
	}
}

// hungStore never answers BannedAmong until its context ends, and then
// still waits a further hang (a store that ignores deadlines).
type hungStore struct {
	Store
	hang time.Duration
}

func (h *hungStore) BannedAmong(ctx context.Context, sigs []string) ([]bool, error) {
	<-ctx.Done()
	time.Sleep(h.hang)
	return make([]bool, len(sigs)), nil
}

// A store that does not answer is abandoned at BanStoreReadTimeout and the
// check refuses, even if the store ignores its context.
func TestSharedStore_UnansweredReadRefusedAtReadDeadline(t *testing.T) {
	mr := miniredis.RunT(t)
	b := NewService(&hungStore{Store: newMiniRedisStore(t, mr.Addr()), hang: 3 * time.Second})
	start := time.Now()
	hit, err := b.IsAnyBanned(context.Background(), []string{"sig-hung"})
	elapsed := time.Since(start)
	if hit || !errors.Is(err, ErrRevocationUnavailable) {
		t.Fatalf("want refusal with ErrRevocationUnavailable, got banned=%v err=%v", hit, err)
	}
	if elapsed > banStoreTimeout+300*time.Millisecond {
		t.Fatalf("refusal took %s; read deadline is %s", elapsed, banStoreTimeout)
	}
}

// slowLink is a TCP relay to a Redis server that, once slowed, holds every
// reply from the server for delay before passing it on.
type slowLink struct {
	ln     net.Listener
	target string
	delay  atomic.Int64
	mu     sync.Mutex
	conns  []net.Conn
}

func newSlowLink(t *testing.T, target string) *slowLink {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	l := &slowLink{ln: ln, target: target}
	go l.serve()
	t.Cleanup(l.close)
	return l
}

func (l *slowLink) addr() string { return l.ln.Addr().String() }

func (l *slowLink) slow(d time.Duration) { l.delay.Store(int64(d)) }

func (l *slowLink) track(c net.Conn) {
	l.mu.Lock()
	l.conns = append(l.conns, c)
	l.mu.Unlock()
}

func (l *slowLink) close() {
	_ = l.ln.Close()
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, c := range l.conns {
		_ = c.Close()
	}
}

func (l *slowLink) serve() {
	for {
		c, err := l.ln.Accept()
		if err != nil {
			return
		}
		s, err := net.Dial("tcp", l.target)
		if err != nil {
			_ = c.Close()
			continue
		}
		l.track(c)
		l.track(s)
		go func() { _, _ = io.Copy(s, c) }()
		go func() {
			buf := make([]byte, 32<<10)
			for {
				n, err := s.Read(buf)
				if n > 0 {
					if d := time.Duration(l.delay.Load()); d > 0 {
						time.Sleep(d)
					}
					if _, werr := c.Write(buf[:n]); werr != nil {
						return
					}
				}
				if err != nil {
					_ = c.Close()
					return
				}
			}
		}()
	}
}

// Redis answers every command 3s late: an access check is refused with
// ErrRevocationUnavailable by the 2s read deadline, not answered at 3s.
func TestRedisStore_DelayedReplyRefusedAtReadDeadline(t *testing.T) {
	mr := miniredis.RunT(t)
	link := newSlowLink(t, mr.Addr())
	svc, err := NewServiceWithRedis(&RedisConfig{Addr: link.addr()})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = svc.Close() })
	link.slow(3 * time.Second)

	start := time.Now()
	hit, err := svc.IsAnyBanned(context.Background(), []string{"sig-delayed"})
	elapsed := time.Since(start)
	if hit || !errors.Is(err, ErrRevocationUnavailable) {
		t.Fatalf("want refusal with ErrRevocationUnavailable, got banned=%v err=%v", hit, err)
	}
	if elapsed > banStoreTimeout+300*time.Millisecond {
		t.Fatalf("refusal took %s; read deadline is %s", elapsed, banStoreTimeout)
	}
}

// The Redis client applies a command's context deadline to the socket: with
// replies held 3s, a command with a 300ms deadline returns within about
// 300ms, not after the client's own socket timeout.
func TestRedisStore_SocketHonoursContextDeadline(t *testing.T) {
	mr := miniredis.RunT(t)
	link := newSlowLink(t, mr.Addr())
	st := newMiniRedisStore(t, link.addr())
	link.slow(3 * time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := st.BannedAmong(ctx, []string{"sig-ctx"})
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("a reply held past the deadline was returned as an answer")
	}
	if elapsed > time.Second {
		t.Fatalf("command with a 300ms deadline returned after %s", elapsed)
	}
}

// Optional: against a real Redis (SATGATE_TEST_REDIS_ADDR), a server that
// stops answering (CLIENT PAUSE 2600ms) gets the check refused within the
// read deadline, not answered after it.
func TestRedisStore_PausedServerRefusedWithinReadDeadline(t *testing.T) {
	addr := os.Getenv("SATGATE_TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("SATGATE_TEST_REDIS_ADDR not set")
	}
	svc, err := NewServiceWithRedis(&RedisConfig{Addr: addr})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = svc.Close() })
	admin := redis.NewClient(&redis.Options{Addr: addr})
	t.Cleanup(func() { _ = admin.Close() })
	if err := admin.Do(context.Background(), "CLIENT", "PAUSE", 2600, "ALL").Err(); err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	hit, err := svc.IsAnyBanned(context.Background(), []string{"sig-paused"})
	elapsed := time.Since(start)
	if hit || !errors.Is(err, ErrRevocationUnavailable) {
		t.Fatalf("want refusal with ErrRevocationUnavailable, got banned=%v err=%v after %s", hit, err, elapsed)
	}
	if elapsed > banStoreTimeout+300*time.Millisecond {
		t.Fatalf("refusal took %s; read deadline is %s", elapsed, banStoreTimeout)
	}
	time.Sleep(time.Until(start.Add(2700 * time.Millisecond)))
}

// A ban written by another gateway on the same store shows in token info,
// the graph, the stats and the ban list at once, and an unban there clears
// them, without waiting for any cache.
func TestViews_ShowBanAndUnbanMadeOnAnotherGateway(t *testing.T) {
	mr := miniredis.RunT(t)
	a := newRedisService(t, mr.Addr())
	b := newRedisService(t, mr.Addr())
	exp := time.Now().Add(time.Hour)
	b.RegisterMint("root-v", "api:*", exp)
	b.RegisterDelegation("child-v", "api:read", exp, "root-v")
	if info := b.GetTokenInfoBySignature("child-v"); info == nil || info.Status != "active" {
		t.Fatalf("before the ban: %+v", info)
	}

	if err := a.BanContext(context.Background(), "root-v", "test", "admin"); err != nil {
		t.Fatal(err)
	}
	for _, sig := range []string{"root-v", "child-v"} {
		if info := b.GetTokenInfoBySignature(sig); info == nil || info.Status != "banned" {
			t.Fatalf("token info of %s after a ban on another gateway: %+v", sig, info)
		}
	}
	graphBanned := 0
	for _, n := range b.GetGraph().Nodes {
		if n.Data.Status == "BANNED" {
			graphBanned++
		}
	}
	if graphBanned != 2 {
		t.Fatalf("graph: want 2 banned nodes, got %d", graphBanned)
	}
	for _, ti := range b.GetAllTokenInfo() {
		if ti.Status != "banned" {
			t.Fatalf("all token info: %s is %s", ti.Signature, ti.Status)
		}
	}
	if got := b.GetStats()["active"]; got != 0 {
		t.Fatalf("stats active after the ban: %v", got)
	}
	if len(b.GetBanList()) != 1 {
		t.Fatalf("ban list: %+v", b.GetBanList())
	}

	if !a.Unban("root-v") {
		t.Fatal("unban reported no ban")
	}
	if info := b.GetTokenInfoBySignature("child-v"); info == nil || info.Status != "active" {
		t.Fatalf("token info after an unban on another gateway: %+v", info)
	}
	if len(b.GetBanList()) != 0 {
		t.Fatalf("ban list after unban: %+v", b.GetBanList())
	}
}

// With the shared store unreadable, the views show no token as active.
func TestViews_StoreUnavailableShowsNoTokenActive(t *testing.T) {
	mr := miniredis.RunT(t)
	b := newRedisService(t, mr.Addr())
	b.RegisterMint("root-u", "api:*", time.Now().Add(time.Hour))
	mr.Close()
	if info := b.GetTokenInfoBySignature("root-u"); info == nil || info.Status == "active" {
		t.Fatalf("token info with the store down: %+v", info)
	}
	for _, n := range b.GetGraph().Nodes {
		if n.Data.Status == "ACTIVE" {
			t.Fatalf("graph node %s ACTIVE with the store down", n.Data.ID)
		}
	}
	if got := b.GetStats()["active"]; got != 0 {
		t.Fatalf("stats active with the store down: %v", got)
	}
}

// Reviewer case: the ban cache never holds more than maxBanCacheEntries
// answers.
func TestSharedStore_BanCacheStaysBounded(t *testing.T) {
	mr := miniredis.RunT(t)
	s := NewService(newMiniRedisStore(t, mr.Addr()))
	sigs := make([]string, 0, maxBanCacheEntries+100)
	for i := 0; i < maxBanCacheEntries+100; i++ {
		sigs = append(sigs, "sig-"+strconv.Itoa(i))
	}
	for i := 0; i < len(sigs); i += 5000 {
		end := i + 5000
		if end > len(sigs) {
			end = len(sigs)
		}
		if _, err := s.IsAnyBanned(context.Background(), sigs[i:end]); err != nil {
			t.Fatal(err)
		}
		s.cacheMu.Lock()
		n := len(s.banCache)
		s.cacheMu.Unlock()
		if n > maxBanCacheEntries {
			t.Fatalf("ban cache holds %d answers, bound is %d", n, maxBanCacheEntries)
		}
	}
}
