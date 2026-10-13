package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
)

// Tools that already passed the credential check stay listed while the host
// cannot read a service's credentials in full (tool_names_checked.go).

// hostRead is a host whose read of the sign-ins can be made to fail, per
// service.
type hostRead struct {
	mu      sync.Mutex
	order   []ClaimService
	bad     []string
	all     bool
	failErr error
}

func (h *hostRead) set(bad []string, order ...ClaimService) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.bad, h.all, h.failErr = bad, false, nil
	if len(order) > 0 {
		h.order = order
	}
}

func (h *hostRead) setErr(err error) {
	h.mu.Lock()
	h.failErr = err
	h.mu.Unlock()
}

func (h *hostRead) get(context.Context) (ClaimOrder, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	co := ClaimOrder{Services: append([]ClaimService(nil), h.order...), Unreadable: append([]string(nil), h.bad...), AllUnreadable: h.all}
	return co, h.failErr
}

func keptSetup(t *testing.T) (*UpstreamManager, *hostRead, *fakeService, *fakeService) {
	t.Helper()
	rh, alp := twoBrokers(t)
	h := &hostRead{}
	h.set(nil, claimSvc("robinhood", "Robinhood", rh, false), claimSvc("alpaca", "Alpaca", alp, false))
	cfgs := map[string]UpstreamConfig{"robinhood": rh.cfg("Robinhood"), "alpaca": alp.cfg("Alpaca")}
	m, _ := startCfg(t, func(m *UpstreamManager) { m.SetToolNameStore(newSeenStore()); m.SetClaimOrderSource(h.get); fast(m) }, nil, cfgs)
	return m, h, rh, alp
}

func rawList(t *testing.T, m *UpstreamManager) string {
	t.Helper()
	b, err := json.Marshal(m.AllTools())
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestChecked_AFailedReadKeepsEveryToolThatPassed(t *testing.T) {
	m, h, _, alp := keptSetup(t)
	before := rawList(t, m)
	n := len(listTools(t, m))
	if n != 9 {
		t.Fatalf("setup: %d tools", n)
	}
	// The read fails for both services (a whole-set failure on the host).
	h.set([]string{"robinhood", "alpaca"})
	m.namesVer.Add(1)
	if after := rawList(t, m); after != before {
		t.Fatalf("the list changed when the read failed:\n before %s\n after  %s", before, after)
	}
	// While it still fails, a new tool appears upstream: held, the rest stay.
	alp.setTools(append(alpacaTools(), fakeTool{Name: "brand_new", Description: "New.", Schema: obj(map[string]any{})})...)
	reDiscover(t, m, "alpaca")
	got := listTools(t, m)
	if got["brand_new"].Name != "" || len(got) != n {
		t.Fatalf("new tool listed while the read fails, or others lost: %v", names(got))
	}
	if after := rawList(t, m); after != before {
		t.Fatalf("the list changed while a new tool was held")
	}
	// The read recovers: the new tool appears after its check.
	h.set(nil)
	m.namesVer.Add(1)
	eventually(t, "the new tool after the read recovers", func() bool {
		return listTools(t, m)["alpaca_brand_new"].Name != "" || listTools(t, m)["brand_new"].Name != ""
	})
	if len(listTools(t, m)) != n+1 {
		t.Fatalf("after recovery: %v", names(listTools(t, m)))
	}
}

func TestChecked_AChangedDescriptionWaitsAndTheOldBytesAreNotKept(t *testing.T) {
	m, h, rh, _ := keptSetup(t)
	n := len(listTools(t, m))
	h.set([]string{"robinhood"})
	m.namesVer.Add(1)
	tools := robinhoodTools()
	tools[0].Description = "Get the brokerage account, now with more."
	rh.setTools(tools...)
	reDiscover(t, m, "robinhood")
	got := listTools(t, m)
	if got["get_account"].Name != "" {
		t.Fatalf("a tool whose description changed was listed while the read fails: %v", got["get_account"])
	}
	if len(got) != n-1 {
		t.Fatalf("others were lost: %v", names(got))
	}
	h.set(nil)
	m.namesVer.Add(1)
	eventually(t, "changed tool after recovery", func() bool { return strings.Contains(listTools(t, m)["get_account"].Description, "more") })
}

// Changed in round 2 (tool-names-keep-checked, 1012b): round 1 expected that
// one unreadable sign-in affected only its own service. The rule rounds 4 and
// 5 confirmed is the other one: a sign-in that cannot be read means new tools
// cannot be checked against it, in any service. A cold start with Robinhood's
// sign-in unreadable therefore lists nothing, and the first list after the
// read recovers shows everything.
func TestChecked_OneUnreadableSignInHoldsNewToolsOfEveryService(t *testing.T) {
	rh, alp := twoBrokers(t)
	h := &hostRead{}
	h.set([]string{"robinhood"}, claimSvc("robinhood", "Robinhood", rh, false), claimSvc("alpaca", "Alpaca", alp, false))
	cfgs := map[string]UpstreamConfig{"robinhood": rh.cfg("Robinhood"), "alpaca": alp.cfg("Alpaca")}
	m, _ := startCfg(t, func(m *UpstreamManager) { m.SetToolNameStore(newSeenStore()); m.SetClaimOrderSource(h.get); fast(m) }, nil, cfgs)
	if got := listTools(t, m); len(got) != 0 {
		t.Fatalf("a cold start with one unreadable sign-in listed %v", names(got))
	}
	h.set(nil) // no namesVer bump, no waiting: the next list is the first after recovery
	if got := listTools(t, m); len(got) != 9 {
		t.Fatalf("the first list after recovery has %d of 9: %v", len(got), names(got))
	}
}

func TestChecked_ColdStartWithAFailingReadHoldsTheServiceAndRetries(t *testing.T) {
	rh, _ := twoBrokers(t)
	h := &hostRead{}
	h.set(nil, claimSvc("robinhood", "Robinhood", rh, false))
	h.setErr(fmt.Errorf("database down"))
	h.mu.Lock()
	h.all = true
	h.mu.Unlock()
	m, _ := startCfg(t, func(m *UpstreamManager) { m.SetToolNameStore(newSeenStore()); m.SetClaimOrderSource(h.get); fast(m) }, nil, map[string]UpstreamConfig{"robinhood": rh.cfg("Robinhood")})
	if n := len(m.AllTools()); n != 0 {
		t.Fatalf("a cold start with a failing read listed %d tools", n)
	}
	h.set(nil)
	m.namesVer.Add(1)
	eventually(t, "tools after the read recovers", func() bool { return len(listTools(t, m)) == 4 })
}

func TestChecked_ASecretInADescriptionHidesTheToolWhetherTheReadWorksOrFails(t *testing.T) {
	m, h, rh, _ := keptSetup(t)
	secret := "private-token-value-123456"
	tools := robinhoodTools()
	tools[1].Description = "Positions. Debug: " + secret
	// 1. The read works and the host knows the value: hidden.
	m.mu.Lock()
	cfg := m.config["robinhood"]
	cfg.ExtraSecrets = []string{secret}
	m.config["robinhood"] = cfg
	m.mu.Unlock()
	rh.setTools(tools...)
	reDiscover(t, m, "robinhood")
	if listTools(t, m)["get_positions"].Name != "" {
		t.Fatal("a tool holding a sign-in value was listed while the read works")
	}
	// 2. The read fails: the tool's bytes are new, so it waits as well.
	h.set([]string{"robinhood"})
	m.namesVer.Add(1)
	if listTools(t, m)["get_positions"].Name != "" {
		t.Fatal("a tool holding a sign-in value was listed while the read fails")
	}
}

func TestChecked_AKeptToolIsStillCheckedAgainstTheValuesTheManagerHas(t *testing.T) {
	m, h, _, _ := keptSetup(t)
	if listTools(t, m)["get_account"].Name == "" {
		t.Fatal("setup")
	}
	// The read fails; the config's credentials now include a value that sits in
	// the (unchanged) description of a tool that passed before. The bytes are
	// the same as when it passed, but it holds a credential: hidden.
	h.set([]string{"robinhood"})
	m.mu.Lock()
	cfg := m.config["robinhood"]
	cfg.ExtraSecrets = []string{"brokerage account"}
	m.config["robinhood"] = cfg
	m.mu.Unlock()
	m.namesVer.Add(1)
	if listTools(t, m)["get_account"].Name != "" {
		t.Fatal("a kept tool that holds a known credential was listed")
	}
}

func TestChecked_SetCredentialsUnreadableKeepsToolsWithoutAClaimOrder(t *testing.T) {
	rh, alp := twoBrokers(t)
	cfgs := map[string]UpstreamConfig{"robinhood": rh.cfg("Robinhood"), "alpaca": alp.cfg("Alpaca")}
	m, _ := startCfg(t, func(m *UpstreamManager) {
		m.SetServiceOrder([]string{"robinhood", "alpaca"})
		fast(m)
	}, nil, cfgs)
	before := rawList(t, m)
	m.SetCredentialsUnreadable([]string{"robinhood", "alpaca"})
	if after := rawList(t, m); after != before {
		t.Fatal("the list changed when the host said the sign-ins could not be read")
	}
	rh.setTools(append(robinhoodTools(), fakeTool{Name: "brand_new", Description: "New."})...)
	reDiscover(t, m, "robinhood")
	if listTools(t, m)["brand_new"].Name != "" {
		t.Fatal("a new tool was listed while the sign-ins could not be read")
	}
	m.SetCredentialsUnreadable(nil)
	eventually(t, "new tool", func() bool { return listTools(t, m)["brand_new"].Name != "" })
}

func TestChecked_ARecordSharedByManagersSurvivesARebuild(t *testing.T) {
	rh, _ := twoBrokers(t)
	rec := NewToolCheckRecord()
	cfgs := map[string]UpstreamConfig{"robinhood": rh.cfg("Robinhood")}
	store := newSeenStore()
	h := &hostRead{}
	h.set(nil, claimSvc("robinhood", "Robinhood", rh, false))
	m1, _ := startCfg(t, func(m *UpstreamManager) {
		m.SetToolNameStore(store)
		m.SetToolCheckRecord(rec)
		m.SetClaimOrderSource(h.get)
	}, nil, cfgs)
	if len(listTools(t, m1)) != 4 {
		t.Fatal("setup")
	}
	// A new manager over the same record, built while the read fails.
	h.set([]string{"robinhood"})
	m2, _ := startCfg(t, func(m *UpstreamManager) {
		m.SetToolNameStore(store)
		m.SetToolCheckRecord(rec)
		m.SetClaimOrderSource(h.get)
	}, nil, cfgs)
	if n := len(listTools(t, m2)); n != 4 {
		t.Fatalf("a rebuilt manager with a shared record lists %d of 4", n)
	}
	// A manager with a record of its own has nothing to keep.
	m3, _ := startCfg(t, func(m *UpstreamManager) { m.SetToolNameStore(store); m.SetClaimOrderSource(h.get) }, nil, cfgs)
	if n := len(m3.AllTools()); n != 0 {
		t.Fatalf("a manager that never passed a check listed %d", n)
	}
}

// reDiscover makes the manager read one service's tools again and rebuild.
func reDiscover(t *testing.T, m *UpstreamManager, key string) {
	t.Helper()
	m.mu.Lock()
	c := m.clients[key]
	m.mu.Unlock()
	if c == nil {
		t.Fatalf("no client %q", key)
	}
	if err := m.discoverTools(context.Background(), c); err != nil {
		t.Fatal(err)
	}
	m.namesVer.Add(1)
}
