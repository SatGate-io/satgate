package mcpserver

import (
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// Round 2 of the kept-tools fix (1012b): every tool is checked against every
// sign-in of the account; an incomplete read is never cached as good; kept
// tools have a time limit; the first list after recovery shows what was held.

// clockAt is a clock the test moves.
type clockAt struct{ ns atomic.Int64 }

func newClock() *clockAt {
	c := &clockAt{}
	c.ns.Store(time.Now().UnixNano())
	return c
}
func (c *clockAt) now() time.Time          { return time.Unix(0, c.ns.Load()).UTC() }
func (c *clockAt) advance(d time.Duration) { c.ns.Add(int64(d)) }

// withSecrets sets the sign-in values the host read for one service.
func (h *hostRead) withSecrets(key string, vals ...string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for i := range h.order {
		if h.order[i].Key == key {
			h.order[i].Secrets = vals
		}
	}
}

// B1: the check covers every sign-in of the account, whichever service the
// value belongs to.
func TestChecked_AToolHoldingAnotherServicesSignInValueIsHiddenWhetherTheReadWorksOrFails(t *testing.T) {
	const aRefresh = "synthetic-robinhood-refresh-4455"
	for _, mode := range []string{"works", "a row cannot be read", "the read errors"} {
		t.Run(mode, func(t *testing.T) {
			rh, alp := twoBrokers(t)
			h := &hostRead{}
			h.set(nil, claimSvc("robinhood", "Robinhood", rh, false), claimSvc("alpaca", "Alpaca", alp, false))
			h.withSecrets("robinhood", "synthetic-robinhood-access-4455", aRefresh)
			// Alpaca's tool describes Robinhood's refresh value.
			tools := alpacaTools()
			tools[2].Description = "Cross service: " + aRefresh
			alp.setTools(tools...)
			cfgs := map[string]UpstreamConfig{"robinhood": rh.cfg("Robinhood"), "alpaca": alp.cfg("Alpaca")}
			m, _ := startCfg(t, func(m *UpstreamManager) { m.SetToolNameStore(newSeenStore()); m.SetClaimOrderSource(h.get); fast(m) }, nil, cfgs)
			switch mode {
			case "a row cannot be read":
				h.set([]string{"robinhood"})
				h.withSecrets("robinhood") // Robinhood's values are unknown
			case "the read errors":
				h.down()
			}
			m.namesVer.Add(1)
			if strings.Contains(rawList(t, m), aRefresh) {
				t.Fatalf("%s: another service's refresh value was listed", mode)
			}
		})
	}
}

// down makes the host's read fail as a whole, as the gateway's does when the
// database does not answer: an error, and no sign-in known.
func (h *hostRead) down() {
	h.mu.Lock()
	h.failErr, h.all = errString("database down"), true
	h.mu.Unlock()
}

type errString string

func (e errString) Error() string { return string(e) }

// A value the manager knows from the last complete read is checked against
// every service's kept tools while the read fails: Robinhood's sign-in value
// appears in Alpaca's unchanged description (the upstream was changed by the
// time the value rotated; the bytes are the same as when it passed).
func TestChecked_AKeptToolIsCheckedAgainstTheValuesOfTheLastCompleteReadOfEveryService(t *testing.T) {
	rh, alp := twoBrokers(t)
	h := &hostRead{}
	h.set(nil, claimSvc("robinhood", "Robinhood", rh, false), claimSvc("alpaca", "Alpaca", alp, false))
	cfgs := map[string]UpstreamConfig{"robinhood": rh.cfg("Robinhood"), "alpaca": alp.cfg("Alpaca")}
	m, _ := startCfg(t, func(m *UpstreamManager) { m.SetToolNameStore(newSeenStore()); m.SetClaimOrderSource(h.get); fast(m) }, nil, cfgs)
	if n := len(m.AllTools()); n != 9 {
		t.Fatalf("setup: %d", n)
	}
	// Robinhood's sign-in is now "Get the trading account." (a value the
	// config carries as ExtraSecrets), and its read fails from here on.
	m.mu.Lock()
	cfg := m.config["robinhood"]
	cfg.ExtraSecrets = []string{"Get the trading account."}
	m.config["robinhood"] = cfg
	m.mu.Unlock()
	h.down()
	m.Recheck()
	for name, tl := range listTools(t, m) {
		if strings.Contains(tl.Description, "trading account") {
			t.Fatalf("a kept tool %q of another service holding a sign-in value of the account was listed", name)
		}
	}
	if n := len(m.AllTools()); n != 8 {
		t.Fatalf("only that tool should be hidden: %d listed", n)
	}
}

// B2: nothing incomplete is cached as good, and no wait is needed.
func TestChecked_TheFirstListAfterRecoveryShowsWhatWasHeld(t *testing.T) {
	t.Run("cold start", func(t *testing.T) {
		rh, alp := twoBrokers(t)
		h := &hostRead{}
		h.set(nil, claimSvc("robinhood", "Robinhood", rh, false), claimSvc("alpaca", "Alpaca", alp, false))
		h.down()
		cfgs := map[string]UpstreamConfig{"robinhood": rh.cfg("Robinhood"), "alpaca": alp.cfg("Alpaca")}
		m, _ := startCfg(t, func(m *UpstreamManager) { m.SetToolNameStore(newSeenStore()); m.SetClaimOrderSource(h.get) }, nil, cfgs)
		if n := len(m.AllTools()); n != 0 {
			t.Fatalf("cold start with a failing read lists %d", n)
		}
		if n := len(m.AllTools()); n != 0 {
			t.Fatalf("second list while it still fails lists %d", n)
		}
		h.set(nil) // the store is back; nothing else happens
		if n := len(m.AllTools()); n != 9 {
			t.Fatalf("the first list after recovery has %d of 9", n)
		}
	})
	t.Run("warm with a waiting new tool", func(t *testing.T) {
		m, h, _, alp := keptSetup(t)
		// As the gateway's router does: the order the services were connected
		// in is the fallback when the order cannot be read.
		m.SetServiceOrder([]string{"robinhood", "alpaca"})
		before := rawList(t, m)
		h.down()
		m.namesVer.Add(1)
		alp.setTools(append(alpacaTools(), fakeTool{Name: "brand_new", Description: "New.", Schema: obj(map[string]any{})})...)
		reDiscover(t, m, "alpaca")
		if got := rawList(t, m); got != before {
			t.Fatalf("the list changed while the read fails:\n%s\n%s", before, got)
		}
		h.set(nil)
		if listTools(t, m)["brand_new"].Name == "" {
			t.Fatal("the first list after recovery did not show the waiting tool")
		}
	})
}

// A view built without a complete read is not cached as good even when nothing
// is held: all its tools are kept. The first list after the read recovers
// checks the kept tools against the complete read; a kept tool that holds a
// value that read shows is gone from that list, with no other trigger.
func TestChecked_AViewBuiltWithoutACompleteReadIsBuiltAgainByTheFirstListAfterRecovery(t *testing.T) {
	m, h, _, _ := keptSetup(t)
	if n := len(m.AllTools()); n != 9 {
		t.Fatalf("setup: %d", n)
	}
	h.down()
	m.Recheck()
	if n := len(m.AllTools()); n != 9 {
		t.Fatalf("while the read fails all 9 are kept: %d", n)
	}
	// The store is back, and its complete read now holds a value that sits in
	// the unchanged description of Alpaca's get_account.
	h.set(nil)
	h.withSecrets("robinhood", "Get the trading account.")
	for name, tl := range listTools(t, m) {
		if strings.Contains(tl.Description, "trading account") {
			t.Fatalf("the first list after recovery still lists %q, which holds a sign-in value", name)
		}
	}
}

// B3: the time limit, on an injected clock; a failed read does not extend it.
func TestChecked_KeptToolsHaveATimeLimitAFailedReadDoesNotExtend(t *testing.T) {
	rh, alp := twoBrokers(t)
	h := &hostRead{}
	h.set(nil, claimSvc("robinhood", "Robinhood", rh, false), claimSvc("alpaca", "Alpaca", alp, false))
	clk := newClock()
	cfgs := map[string]UpstreamConfig{"robinhood": rh.cfg("Robinhood"), "alpaca": alp.cfg("Alpaca")}
	m, _ := startCfg(t, func(m *UpstreamManager) {
		m.SetClock(clk.now)
		m.SetToolNameStore(newSeenStore())
		m.SetClaimOrderSource(h.get)
	}, nil, cfgs)
	if n := len(m.AllTools()); n != 9 {
		t.Fatalf("setup: %d", n)
	}
	h.down()
	m.namesVer.Add(1)

	clk.advance(KeptToolsMaxAge - time.Second)
	if n := len(m.AllTools()); n != 9 {
		t.Fatalf("at the limit minus 1 s: %d of 9 listed", n)
	}
	// Failed reads in between (each list reads and fails) do not extend it.
	for i := 0; i < 3; i++ {
		clk.advance(300 * time.Millisecond)
		_ = m.AllTools()
	}
	clk.advance(time.Second) // now at the limit plus 1 s - 0.1 s... move well past
	clk.advance(time.Second)
	if n := len(m.AllTools()); n != 0 {
		t.Fatalf("at the limit plus 1 s: %d listed, want 0", n)
	}
	// A complete read works: listed again.
	h.set(nil)
	if n := len(m.AllTools()); n != 9 {
		t.Fatalf("after a complete read: %d of 9", n)
	}
}

// A view cached while the read was fine does not outlive the limit once the
// host has said the read is failing.
func TestChecked_AViewCachedBeforeTheFailureStopsAtTheLimit(t *testing.T) {
	m, h, _, _ := keptSetup(t)
	clk := newClock()
	m.SetClock(clk.now)
	if n := len(m.AllTools()); n != 9 {
		t.Fatalf("setup: %d", n)
	}
	h.set([]string{"robinhood"})
	m.SetCredentialsUnreadable([]string{"robinhood"})
	clk.advance(KeptToolsMaxAge - time.Second)
	if n := len(m.AllTools()); n != 9 {
		t.Fatalf("before the limit: %d", n)
	}
	clk.advance(2 * time.Second)
	if n := len(m.AllTools()); n != 0 {
		t.Fatalf("after the limit: %d", n)
	}
}

// Forget empties the record: kept tools wait, as on a cold start, and a read
// that began before the Forget cannot bring anything back.
func TestChecked_ForgetMakesKeptToolsWaitAndDropsAReadThatBeganBefore(t *testing.T) {
	m, h, _, _ := keptSetup(t)
	if n := len(m.AllTools()); n != 9 {
		t.Fatalf("setup: %d", n)
	}
	rec := m.checked
	epoch := rec.Epoch()
	rec.Forget()
	if rec.NoteCompleteRead(epoch, time.Now(), []string{"x-old-value-123"}) {
		t.Fatal("a read that began before Forget was recorded")
	}
	h.down()
	m.namesVer.Add(1)
	if n := len(m.AllTools()); n != 0 {
		t.Fatalf("after Forget with a failing read: %d listed", n)
	}
	h.set(nil)
	if n := len(m.AllTools()); n != 9 {
		t.Fatalf("after a complete read: %d", n)
	}
}

// A sign-in written on this machine drops the kept values and the time of the
// last complete read: tools kept only by the time limit wait for the next
// complete read, and a list builds from a read that works at once.
func TestChecked_ForgetSignInsMakesKeptToolsWaitForTheNextCompleteRead(t *testing.T) {
	m, h, _, _ := keptSetup(t)
	if n := len(m.AllTools()); n != 9 {
		t.Fatalf("setup: %d", n)
	}
	h.down()
	m.checked.ForgetSignIns()
	m.Recheck()
	if n := len(m.AllTools()); n != 0 {
		t.Fatalf("with the read failing after a sign-in was written: %d listed", n)
	}
	h.set(nil)
	if n := len(m.AllTools()); n != 9 {
		t.Fatalf("after a complete read: %d", n)
	}
}
