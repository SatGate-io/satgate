package mcpserver

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// The view: the one list of names agents see. See tool_names.go for the rules.

// ToolNameInfo describes one listed tool, for the owner's panel and for tests.
type ToolNameInfo struct {
	Service  string // the service's display name
	Key      string // the service's key in the manager
	Original string // the tool's own name
	Name     string // the name agents see
	Renamed  bool
	Reason   string // "", ToolRenamedClash, ToolRenamedInvalid, ToolRenamedPending or ToolRenamedLookalike
	Peer     string // the service that holds the plain name, for a clash
	// LooksLike is the display name of another connected service whose slug
	// this name starts with ("robinhood_..."), for names stored before that
	// was a clash. Empty for every name chosen now.
	LooksLike string
}

// ToolWait says one service's new tools are waiting for an earlier one.
type ToolWait struct {
	Service    string // the service whose new tools wait
	WaitingFor string // the earlier service that has not answered
}

// SetToolNameStore sets where assignments are kept. Call it before Start.
// Without one the manager keeps them in memory for the life of the process.
func (m *UpstreamManager) SetToolNameStore(s ToolNameStore) {
	m.viewMu.Lock()
	m.nameStore = s
	m.viewMu.Unlock()
	m.namesVer.Add(1)
}

// SetServiceOrder says which service came first: it keeps a plain name that
// two services offer, when the name has not been assigned yet. Keys not listed
// follow in sorted order. Without it all services are sorted by key. Call it
// before Start. A manager with a claim order (SetClaimOrderSource) ignores it.
func (m *UpstreamManager) SetServiceOrder(keys []string) {
	m.viewMu.Lock()
	m.order = append([]string(nil), keys...)
	m.viewMu.Unlock()
	m.namesVer.Add(1)
}

// SetClaimOrderSource makes the manager take its order from the durable
// connection order of the services (see tool_names.go, "Durable store"). fn is
// called whenever the view is built. While it fails, nothing new is named.
// Call it before Start.
func (m *UpstreamManager) SetClaimOrderSource(fn func(ctx context.Context) (ClaimOrder, error)) {
	m.viewMu.Lock()
	m.claimOrder = fn
	m.viewMu.Unlock()
	m.namesVer.Add(1)
}

// ToolNames returns every tool the manager lists, in listing order.
func (m *UpstreamManager) ToolNames() []ToolNameInfo {
	v := m.toolView()
	out := make([]ToolNameInfo, 0, len(v.entries))
	for _, e := range v.entries {
		out = append(out, ToolNameInfo{Service: e.label, Key: e.service, Original: e.original,
			Name: e.exposed, Renamed: e.exposed != e.original, Reason: e.reason, Peer: e.peer, LooksLike: e.looksLike})
	}
	return out
}

// Waiting returns the services whose new tools wait for an earlier service.
func (m *UpstreamManager) Waiting() []ToolWait {
	v := m.toolView()
	return append([]ToolWait(nil), v.waiting...)
}

// DuplicateServices returns groups of service keys that share one claim key.
// The first of a group is served; the others offer no tools to agents.
func (m *UpstreamManager) DuplicateServices() [][]string {
	v := m.toolView()
	return append([][]string(nil), v.dups...)
}

// IsRenamedTool reports whether name is a listed tool whose name is not its
// own (a clash or an invalid original). Names that are not listed give false.
func (m *UpstreamManager) IsRenamedTool(name string) bool {
	e := m.toolView().byName[name]
	return e != nil && e.exposed != e.original
}

// ToolOrigin says which service and which of its own tools a name agents see
// stands for, read from the same view that lists and routes the names. A name
// that is not listed while some service's tools are still missing (or the
// stored names could not be read) cannot be answered: Err is set.
func (m *UpstreamManager) ToolOrigin(name string) ToolOrigin {
	v := m.toolView()
	if e := v.byName[name]; e != nil {
		return ToolOrigin{Listed: true, Renamed: e.exposed != e.original, Service: e.service, Original: e.original}
	}
	if v.unavailable || v.incomplete {
		return ToolOrigin{Err: ErrToolTableUnavailable}
	}
	return ToolOrigin{}
}

// ---------------------------------------------------------------------------

// serviceSnap is one configured service, as read for a view.
type serviceSnap struct {
	key      string
	label    string
	claimKey string
	client   *UpstreamClient // nil: configured, not connected
	tools    []json.RawMessage
	complete bool
	withheld bool // the host could not read the service's credentials
	// unreadable: the host could not read the service's complete credential
	// set just now. Tools that passed the check before, with the same bytes,
	// stay; the rest are held back (tool_names_checked.go).
	unreadable bool
	secrets    []string
	// signIns are the OAuth sign-in values (access and refresh) the host
	// passed with the service's config (ExtraSecrets). They are screened
	// against every service's tools, not only this one's.
	signIns []string
}

type toolEntry struct {
	service  string // manager key
	label    string
	claimKey string
	original string
	exposed  string
	reason   string
	peer     string
	client   *UpstreamClient
	raw      json.RawMessage // as listed
	toolIdx  int
	svcIdx   int
	blocked  bool
	// looksLike: the display name of another connected service whose slug the
	// listed name starts with.
	looksLike string
}

type toolView struct {
	ver        uint64
	at         time.Time
	retryAfter time.Duration
	incomplete bool // something could not be settled; build again soon
	fastRetry  bool // the cause is a store or order read: retry in 2s, not 10s
	// unavailable: some service's tools are not all in yet, so an unlisted
	// name may be one of them.
	unavailable bool
	missing     bool // a service has no complete list: retry its discovery
	// held: some tool waits for a complete read of the account's sign-ins
	// (the host could not read all of them in full, or the time limit on kept
	// tools has passed). Every list builds the view again, one build at a
	// time, until a complete read works.
	held bool
	// keptUntil: the time (on the manager's clock) this view stops being
	// served: KeptToolsMaxAge after the complete read it stands on (a view
	// built from a complete read), or after the last complete read whose
	// values screened its kept tools (a view built while the read fails).
	// Zero: it stands on no read (a hold, or a view of nothing). A view past
	// it is never returned: it is built again, and with no complete read to
	// stand on that build holds every tool.
	keptUntil time.Time
	// rec, recEpoch: the record this view's check used and its generation
	// when the host's read came back. A view is served only while the record
	// is still in that generation (Forget, ForgetSignIns, InvalidateTenant
	// and a sign-in written on this machine all move it).
	rec      *ToolCheckRecord
	recEpoch uint64
	// stale: the generation moved while the view was being built, so it was
	// screened with values the host has since replaced. Never published.
	stale   bool
	entries []*toolEntry
	byName  map[string]*toolEntry // exposed, exact
	tools   []json.RawMessage
	dups    [][]string
	waiting []ToolWait
}

const (
	viewRetryAfter   = 2 * time.Second
	viewWaitingRetry = 10 * time.Second
	// defaultRetryEvery is how often a service that has not answered is tried
	// again.
	defaultRetryEvery = 15 * time.Second
)

func (m *UpstreamManager) snapshotServices() []serviceSnap {
	m.mu.Lock()
	defer m.mu.Unlock()
	keys := map[string]bool{}
	for k := range m.clients {
		keys[k] = true
	}
	for k := range m.config {
		keys[k] = true
	}
	out := make([]serviceSnap, 0, len(keys))
	for k := range keys {
		cfg := m.config[k]
		label := strings.TrimSpace(cfg.DisplayName)
		if label == "" {
			label = k
		}
		claim := ServiceKey(cfg)
		if claim == "" {
			// No address to bind a name to (an upstream with neither URL nor
			// command). Bind to the key, so it still gets stable names.
			sum := sha256.Sum256([]byte("key:" + k))
			claim = hex.EncodeToString(sum[:])
		}
		s := serviceSnap{key: k, label: label, claimKey: claim, secrets: credentialValues(cfg), unreadable: m.unreadable[k]}
		for _, v := range cfg.ExtraSecrets {
			if v = strings.TrimSpace(v); len(v) >= minSecretLen {
				s.signIns = append(s.signIns, v)
			}
		}
		if c := m.clients[k]; c != nil {
			s.client = c
			s.tools, s.complete = c.listState()
		}
		if cfg.WithholdTools {
			// The host could not read this service's complete credential
			// set, so nothing it lists can be checked: none of it is shown.
			s.tools, s.complete, s.withheld = nil, false, true
		}
		out = append(out, s)
	}
	return out
}

// orderServices sorts snap in the order names are claimed: the durable claim
// order when there is one, else the routing/explicit order, else sorted keys.
func (m *UpstreamManager) orderServices(snap []serviceSnap, co *ClaimOrder) {
	pos := map[string]int{}
	if co != nil {
		for i, s := range co.Services {
			if _, dup := pos[s.Key]; !dup {
				pos[s.Key] = i
			}
		}
	} else {
		for i, k := range m.order {
			if _, dup := pos[k]; !dup {
				pos[k] = i
			}
		}
	}
	sort.SliceStable(snap, func(i, j int) bool {
		pi, iok := pos[snap[i].key]
		pj, jok := pos[snap[j].key]
		switch {
		case iok && jok:
			return pi < pj
		case iok:
			return true
		case jok:
			return false
		}
		return snap[i].key < snap[j].key
	})
}

// toolView returns the current view, building it when services changed since
// the last one, when its retry time has passed, or when the tools it keeps
// have reached the time limit (KeptToolsMaxAge). It is for calls and for the
// accessors: it waits for the view lock and for a build as long as that takes
// (a build is bounded by 10 s), exactly as before the list budget.
func (m *UpstreamManager) toolView() *toolView { return m.viewFor(nil, false) }

// listView is toolView for a list of tools, under one time budget: ListBudget
// from now, or the deadline of ctx if that is sooner. The budget
// covers the wait for the view lock and every build, including each build's
// read of the sign-ins and of the stored names. When it is used up the list
// answers at once with what is safe: a complete view that is still valid, or
// the held view (no tools).
//
// A view built without a complete read of the account's sign-ins is built
// again by every list, one build at a time: a list that arrives while a
// build runs uses that build's result. The first list after the store
// recovers therefore shows the held tools.
func (m *UpstreamManager) listView(ctx context.Context) *toolView { return m.viewFor(ctx, true) }

// valid says whether a built view may still be served at now: it is within
// KeptToolsMaxAge of the complete read it stands on, the record it was
// checked against has not been forgotten since, and it was not built from a
// read that a sign-in write had already made stale. Every path that serves a
// view (the cached view, a shared build, the name-store fallback) asks it.
func (v *toolView) valid(now time.Time) bool {
	if v.stale {
		return false
	}
	if !v.keptUntil.IsZero() && !now.Before(v.keptUntil) {
		return false
	}
	return v.rec == nil || v.rec.Current(v.recEpoch)
}

// heldView is what is served when no view could be built from a read that is
// still current: no tool is listed or routed, and every list tries again.
func heldView() *toolView {
	return &toolView{byName: map[string]*toolEntry{}, tools: []json.RawMessage{},
		held: true, incomplete: true, unavailable: true, fastRetry: true}
}

const (
	// A view whose read was made stale by a sign-in write while it was being
	// built is built again from the new values, at most viewBuildAttempts
	// times (a cap, not three allowances: every build ends with the list's
	// budget) and not after viewBuildBudget; then nothing is listed until the
	// next list builds again.
	viewBuildAttempts = 3
	viewBuildBudget   = 2 * time.Second

	// ListBudget is the time one list has, from the moment it reaches
	// AllToolsContext (or the earlier deadline of the caller's context). The
	// enterprise router starts the same clock when the list reaches
	// AllToolsForTenant and passes what is left.
	//
	// Why 2.5 s: the 2 s read deadline of the sign-in read and the 250 ms a
	// list gives a config load to wind down already come to 2.25 s, the
	// longest single wait measured at round 3 (2.25 s inside the limit, 2.76 s
	// past it). The budget ends there, 0.5 s under the 3 s bar. The 0.5 s is
	// for what the budget cannot bound: the scheduler waking the goroutines
	// that notice the deadline (measured in tens of milliseconds under -race
	// with the machine loaded), the answer being marshalled, and a slower CI
	// runner. A budget of 2.8 s or more would leave under 200 ms for that.
	ListBudget = 2500 * time.Millisecond

	// viewBuildLimit bounds one build for a call (not a list), as before.
	viewBuildLimit = 10 * time.Second
)

// listBudget is ListBudget; tests in this package shorten it.
var listBudget = ListBudget

// lockViewWithin takes viewMu, giving up at dl or when ctx ends. sync.Mutex
// has no timed lock: the wait is made by a helper goroutine, in the mutex's
// own queue, and a list that gives up leaves the helper to release the lock
// the moment it gets it (a build holds it for at most its own budget, so the
// helper does not outlive that). The wait of a list is never longer than its
// budget, whoever holds the lock.
func (m *UpstreamManager) lockViewWithin(ctx context.Context, dl time.Time) bool {
	if m.viewMu.TryLock() {
		return true
	}
	got := make(chan struct{})
	go func() {
		m.viewMu.Lock()
		close(got)
	}()
	t := time.NewTimer(time.Until(dl))
	defer t.Stop()
	select {
	case <-got:
		return true
	case <-t.C:
	case <-ctx.Done():
	}
	select {
	case <-got: // got it at the last moment
		return true
	default:
	}
	go func() {
		<-got
		m.viewMu.Unlock()
	}()
	return false
}

// outOfBudget is the answer of a list that has no time left: the cached view
// if it is complete, valid and current, otherwise the held view. It never
// takes the view lock and never builds.
func (m *UpstreamManager) outOfBudget() *toolView {
	if v := m.viewPtr.Load(); v != nil && v.ver == m.namesVer.Load() && !v.incomplete && v.valid(m.now()) {
		return v
	}
	return heldView()
}

func (m *UpstreamManager) viewFor(ctx context.Context, list bool) *toolView {
	arrived := time.Now()
	if v := m.viewPtr.Load(); v != nil && v.ver == m.namesVer.Load() && !v.incomplete && v.valid(m.now()) {
		return v
	}
	// A list has one budget from here: the wait for the lock, then every
	// build. A call (list == false) waits and builds as before.
	var dl time.Time
	if list {
		if ctx == nil {
			ctx = context.Background()
		}
		// The caller's deadline if it is sooner than ListBudget from now.
		dl = arrived.Add(listBudget)
		if cd, ok := ctx.Deadline(); ok && cd.Before(dl) {
			dl = cd
		}
		if !m.lockViewWithin(ctx, dl) {
			return m.outOfBudget()
		}
	} else {
		m.viewMu.Lock()
	}
	defer m.viewMu.Unlock()
	ver := m.namesVer.Load()
	if v := m.viewPtr.Load(); v != nil && v.valid(m.now()) {
		switch {
		case v.ver != ver:
			// Out of date. The one thing a queued list shares anyway is the
			// held result of a build that finished after it arrived and
			// lists nothing: the sign-in writes that made the build give up
			// moved the version, and another build would meet the same
			// writes. A view with tools and an old version is never shared.
			if list && v.held && len(v.tools) == 0 && v.at.After(arrived) {
				return v
			}
		case !v.incomplete:
			return v
		case list && v.held && v.at.After(arrived):
			// A build that finished after this list arrived: lists queued
			// behind it share its result instead of each paying for a read.
			return v
		case list && v.held:
			// Another build is due: this list starts it.
		case time.Since(v.at) < v.retryAfter:
			return v
		}
	}
	if list && !time.Now().Before(dl) {
		// No build starts once the budget is used up.
		return heldView()
	}
	if m.nameStore == nil {
		m.nameStore = NewMemoryToolNameStore()
	}
	// Every build of this list ends with its budget (dl); a call's builds
	// end after viewBuildLimit.
	end := dl
	if !list {
		end = time.Now().Add(viewBuildLimit)
	}
	ctx, cancel := context.WithDeadline(context.Background(), end)
	defer cancel()
	// A view is published only if the record is still in the generation its
	// read came back in. If a sign-in write moved it during the build, the
	// view is built again from the new values (a bounded number of times,
	// inside viewBuildBudget and the list's budget), then held.
	started := time.Now()
	var nv *toolView
	bctx := ctx
	for attempt := 1; ; attempt++ {
		nv = m.buildToolView(bctx, m.snapshotServices())
		if nv.valid(m.now()) {
			break
		}
		left := viewBuildBudget - time.Since(started)
		if attempt >= viewBuildAttempts || left <= 0 || (list && !time.Now().Before(dl)) {
			nv = heldView()
			break
		}
		// The build again gets what is left of viewBuildBudget (and never
		// more than what is left of the list's budget: bctx is a child of
		// the context that ends with it).
		var stop context.CancelFunc
		bctx, stop = context.WithTimeout(ctx, left)
		defer stop()
	}
	nv.ver = ver
	nv.at = time.Now()
	nv.retryAfter = viewRetryAfter
	if !nv.fastRetry && (nv.missing || nv.held || len(nv.waiting) > 0) {
		nv.retryAfter = viewWaitingRetry
		if m.waitRetry > 0 {
			nv.retryAfter = m.waitRetry
		}
	}
	m.viewPtr.Store(nv)
	if nv.missing {
		m.kickRetry()
	}
	return nv
}

// kickRetry tries, in the background and at most every retryEvery, the
// services that have no complete tool list: a service that is not connected
// is connected again; one whose list was cut short is listed again.
func (m *UpstreamManager) kickRetry() {
	ctx := m.runCtx
	if ctx == nil || m.closed.Load() || ctx.Err() != nil {
		return
	}
	every := m.retryEvery
	if every == 0 {
		every = defaultRetryEvery
	}
	if last := m.lastRetry.Load(); last != 0 && time.Since(time.Unix(0, last)) < every {
		return
	}
	// A retry started after Close would outlive the manager: refuse it, and
	// let Close wait for the one that is running.
	m.retryMu.Lock()
	if m.closed.Load() || !m.retrying.CompareAndSwap(false, true) {
		m.retryMu.Unlock()
		return
	}
	m.retryWG.Add(1)
	m.retryMu.Unlock()
	m.lastRetry.Store(time.Now().UnixNano())
	go func() {
		defer m.retryWG.Done()
		defer m.retrying.Store(false)
		type job struct {
			key    string
			cfg    UpstreamConfig
			client *UpstreamClient
		}
		var jobs []job
		m.mu.Lock()
		for k, cfg := range m.config {
			c := m.clients[k]
			if c == nil {
				jobs = append(jobs, job{key: k, cfg: cfg})
				continue
			}
			if _, ok := c.listState(); !ok {
				jobs = append(jobs, job{key: k, cfg: cfg, client: c})
			}
		}
		m.mu.Unlock()
		sort.Slice(jobs, func(i, j int) bool { return jobs[i].key < jobs[j].key })
		for _, j := range jobs {
			if m.closed.Load() || ctx.Err() != nil {
				return
			}
			if j.client == nil {
				// The new client's read loop lives as long as ctx: do not
				// give it a context that ends with this attempt.
				if err := m.addUpstream(ctx, j.key, j.cfg, true); err == nil {
					log.Info().Str("upstream", j.key).Msg("upstream answered after an earlier failure")
				}
				continue
			}
			rctx, cancel := context.WithTimeout(ctx, 45*time.Second)
			if err := m.discoverTools(rctx, j.client); err == nil {
				if _, ok := j.client.listState(); ok {
					m.namesVer.Add(1)
				}
			}
			cancel()
		}
	}()
}

// minSecretLen is the shortest value treated as a credential in a tool name or
// description: the shortest credential the gateway protects in replies.
const minSecretLen = 8

// protocolHeaderNames are header names whose values are not secrets.
var protocolHeaderNames = map[string]bool{
	"accept": true, "accept-charset": true, "accept-encoding": true, "accept-language": true,
	"cache-control": true, "connection": true, "content-encoding": true, "content-language": true,
	"content-length": true, "content-type": true, "dnt": true, "host": true,
	"mcp-protocol-version": true, "mcp-session-id": true, "origin": true, "pragma": true,
	"referer": true, "user-agent": true, "x-requested-with": true, "x-forwarded-proto": true,
	"x-satgate-request-kind": true,
}

// credentialValues lists the credential values of one service: every query
// value in its URL, every header value (and the part after a scheme word such
// as "Bearer"), the OAuth tokens the host passes in ExtraSecrets.
func credentialValues(cfg UpstreamConfig) []string {
	var out []string
	add := func(s string) {
		if s = strings.TrimSpace(s); len(s) >= minSecretLen {
			out = append(out, s)
		}
	}
	if u, err := url.Parse(strings.TrimSpace(cfg.URL)); err == nil && u.Host != "" {
		for _, vs := range u.Query() {
			for _, v := range vs {
				add(v)
			}
		}
		for _, part := range strings.Split(u.RawQuery, "&") {
			if i := strings.IndexByte(part, '='); i >= 0 {
				add(part[i+1:])
			}
		}
		if u.User != nil {
			add(u.User.Username())
			if p, ok := u.User.Password(); ok {
				add(p)
			}
		}
	}
	for name, v := range cfg.Headers {
		if protocolHeaderNames[strings.ToLower(strings.TrimSpace(name))] {
			continue
		}
		add(v)
		if i := strings.IndexByte(strings.TrimSpace(v), ' '); i > 0 && i <= 16 {
			add(strings.TrimSpace(v)[i+1:])
		}
	}
	for _, v := range cfg.ExtraSecrets {
		add(v)
	}
	return out
}

func holdsSecret(secrets []string, texts ...string) bool {
	for _, t := range texts {
		lt := strings.ToLower(t)
		for _, s := range secrets {
			if strings.Contains(lt, strings.ToLower(s)) {
				return true
			}
		}
	}
	return false
}

// assignSlugs gives every service a slug that is unique, does not start with
// the gateway's reserved prefix, and is not "_"-prefixed by another slug (so a
// name that starts with one service's slug is never another's own prefix).
func assignSlugs(services []serviceSnap) []string {
	slugs := make([]string, len(services))
	used := map[string]bool{}
	related := func(a, b string) bool {
		return strings.HasPrefix(a, b+"_") || strings.HasPrefix(b, a+"_")
	}
	for i, s := range services {
		base := serviceSlug(s.label)
		if strings.HasPrefix(base, "satgate") {
			base = "svc_" + base
		}
		clash := func(c string) bool {
			if used[c] {
				return true
			}
			for u := range used {
				if related(c, u) {
					return true
				}
			}
			return false
		}
		slug := base
		if clash(slug) {
			// Try the same slug without separators first, then numbered.
			if flat := strings.ReplaceAll(base, "_", ""); flat != "" && !clash(flat) {
				slug = flat
			}
		}
		for n := 2; clash(slug); n++ {
			slug = fmt.Sprintf("%s%d", strings.ReplaceAll(base, "_", ""), n)
			if strings.ReplaceAll(base, "_", "") == "" {
				slug = fmt.Sprintf("svc%d", n)
			}
		}
		used[slug] = true
		slugs[i] = slug
	}
	return slugs
}

// ServiceSlugs returns the slug each service gets, given the display names of
// the distinct services in claim order. The owner's panel uses it to spot
// names that start with another service's slug; it is the function the
// manager itself uses.
func ServiceSlugs(labels []string) []string {
	snaps := make([]serviceSnap, len(labels))
	for i, l := range labels {
		snaps[i] = serviceSnap{label: l}
	}
	return assignSlugs(snaps)
}

type nameCandidates struct {
	plain      string   // the tool's own name, or its sanitized form; "" if not allowed
	lookalike  bool     // plain was dropped because it starts with another service's slug
	invalid    bool     // the tool's own name is not accepted by agent apps
	prefixed   []string // <slug>_<tool> and its numbered forms
	skipReason string
}

// candidateNames lists, in order of preference, the names a tool may take. A
// tool with a valid own name tries it first. Then come the service-prefixed
// name and numbered forms of it. The list is a pure function of its inputs, so
// every machine proposes the same names. Every candidate is checked against
// the reserved names, whatever path made it.
func candidateNames(slug, original string, otherSlugs []string) nameCandidates {
	var c nameCandidates
	base := sanitizeToolName(original)
	c.invalid = !ValidAgentToolName(original)
	plain := original
	if c.invalid {
		plain = fitToolName(base, original)
	}
	switch {
	case !ValidAgentToolName(plain) || IsReservedAgentToolName(plain):
	case startsWithOtherSlug(plain, otherSlugs):
		c.lookalike = true
	default:
		c.plain = plain
	}
	add := func(s string) {
		if ValidAgentToolName(s) && !IsReservedAgentToolName(s) {
			c.prefixed = append(c.prefixed, s)
		}
	}
	prefixed := fitToolName(slug+"_"+base, original)
	add(prefixed)
	for n := 2; n <= 1000; n++ {
		add(numberedToolName(prefixed, n))
	}
	return c
}

func startsWithOtherSlug(name string, others []string) bool {
	l := strings.ToLower(name)
	for _, s := range others {
		if s != "" && strings.HasPrefix(l, s+"_") {
			return true
		}
	}
	return false
}

// buildToolView assigns a name to every tool and renders the list.
func (m *UpstreamManager) buildToolView(ctx context.Context, snap []serviceSnap) *toolView {
	v := &toolView{byName: map[string]*toolEntry{}, tools: []json.RawMessage{}}
	store := m.nameStore

	assigned, err := store.Load(ctx)
	if err != nil {
		// Names never change, so the last view is still right about every name
		// it has; keep serving it and try again soon.
		log.Warn().Msg("tool names: stored names could not be read; the last list is kept until they can")
		// The last view is reused only while it is still valid: within
		// KeptToolsMaxAge of the complete read it stands on and in the
		// record's current generation (Forget and InvalidateTenant end it).
		// Otherwise nothing is listed until the names can be read.
		if prev := m.viewPtr.Load(); prev != nil && prev.valid(m.now()) && (prev.rec == nil || prev.rec == m.checked) {
			keep := *prev
			keep.incomplete = true
			keep.fastRetry = true
			return &keep
		}
		// Nothing can be listed, and every list tries the store again (held),
		// so the first list after it recovers shows the tools.
		v.held = true
		v.unavailable = true
		v.incomplete = true
		v.fastRetry = true
		return v
	}
	byTool := map[string]ToolNameAssignment{}
	byLower := map[string]ToolNameAssignment{}
	for _, a := range assigned {
		byTool[toolBindingKey(a.Address, a.Original)] = a
		byLower[strings.ToLower(a.Name)] = a
	}

	// The order new names are claimed in. A durable claim order that cannot
	// be read means nothing new is named this cycle.
	var co *ClaimOrder
	durable := m.claimOrder != nil
	claimsOK := true
	seenSet := map[string]bool{}
	seenStore, hasSeen := store.(ToolSeenStore)
	if m.checked == nil {
		m.checked = NewToolCheckRecord()
	}
	// Is the read of the account's sign-ins complete? One sign-in that cannot
	// be read makes it incomplete for every service: a new or changed tool is
	// checked against all of the account's sign-ins or not at all.
	epoch := m.checked.Epoch()
	readAt := m.now()
	readComplete := true
	var addAcctEarly []string
	if durable {
		c, err := m.claimOrder(ctx)
		if !c.ReadAt.IsZero() {
			epoch, readAt = c.ReadEpoch, c.ReadAt
		} else {
			readAt = m.now()
		}
		// What the host says about the sign-ins decides whether the read is
		// complete, with or without an error: an order that cannot be read
		// (err) is not by itself a sign-in that cannot (the host says
		// AllUnreadable when it knows no sign-in). The values the host did
		// read count either way.
		if c.AllUnreadable || len(c.Unreadable) > 0 {
			readComplete = false
		}
		for _, cs := range c.Services {
			addAcctEarly = append(addAcctEarly, cs.Secrets...)
		}
		if err != nil {
			claimsOK = false
			log.Warn().Msg("tool names: the connection order could not be read; no new names are chosen until it can")
		} else {
			co = &c
			if hasSeen {
				s, err := seenStore.LoadSeen(ctx)
				if err != nil {
					claimsOK = false
					log.Warn().Msg("tool names: the list of services already answered could not be read; no new names are chosen until it can")
				} else {
					seenSet = s
				}
			}
		}
	} else {
		m.mu.Lock()
		readComplete = len(m.unreadable) == 0
		m.mu.Unlock()
	}
	for i := range snap {
		snap[i].unreadable = !readComplete
	}
	// The generation this view is bound to: the record's, now that the host's
	// read is back. A sign-in write after this point moves it and ends the
	// view (valid). One before it is caught by NoteCompleteRead below, which
	// compares the generation the read began in (epoch).
	v.rec, v.recEpoch = m.checked, m.checked.Epoch()
	m.orderServices(snap, co)
	// Every sign-in value the account has: the values the host read in this
	// build, and the values its services' configs carry (for a service that
	// could not be read, the values of the last complete read). The check of a
	// tool covers all of them, whichever service the tool belongs to.
	var acct []string
	addAcct := func(vs ...string) {
		for _, v := range vs {
			if v = strings.TrimSpace(v); len(v) >= minSecretLen {
				acct = append(acct, v)
			}
		}
	}
	addAcct(addAcctEarly...)
	if co != nil {
		extra := map[string][]string{}
		for _, cs := range co.Services {
			if len(cs.Secrets) > 0 {
				extra[cs.ClaimKey] = append(extra[cs.ClaimKey], cs.Secrets...)
			}
		}
		for i := range snap {
			for _, sec := range extra[snap[i].claimKey] {
				if len(strings.TrimSpace(sec)) >= minSecretLen {
					snap[i].secrets = append(snap[i].secrets, strings.TrimSpace(sec))
				}
			}
		}
	}
	for i := range snap {
		addAcct(snap[i].signIns...)
	}
	keptOK := false
	if readComplete {
		// Was the record forgotten after this read began (a sign-in was
		// written, or InvalidateTenant ran)? Then the values just read may be
		// older than the stored ones: nothing is named, stored or listed from
		// them, and the caller builds again (viewFor). NoteCompleteRead says
		// it for a read that is recorded; a build whose order could not be
		// read records nothing and asks the record directly.
		var fresh bool
		if claimsOK {
			fresh = m.checked.NoteCompleteRead(epoch, readAt, acct)
		} else {
			fresh = m.checked.Current(epoch)
		}
		if !fresh {
			v.stale = true
			return v
		}
		// The view stands on this complete read, and no longer than
		// KeptToolsMaxAge after it.
		v.keptUntil = readAt.Add(KeptToolsMaxAge)
	} else {
		// The read failed: the values of the last complete read stand in for
		// it, for at most KeptToolsMaxAge after that read. A failed read does
		// not move that time.
		var vals []string
		var until time.Time
		vals, until, keptOK = m.checked.kept(m.now())
		addAcct(vals...)
		if keptOK {
			v.keptUntil = until
		}
	}
	if !claimsOK {
		v.fastRetry = true
	}
	if !readComplete {
		// Not every sign-in was read: whatever is listed is listed on the
		// strength of an earlier read. Every list reads again until one is
		// complete (viewFor), which also moves the time of the last complete
		// read forward as soon as the store is back.
		v.held = true
		v.incomplete = true
	}

	// Services in order; a second service with the same claim key is the same
	// service connected twice and offers nothing.
	var services []serviceSnap
	seenKey := map[string]int{}
	var dupGroups [][]string
	groupAt := map[string]int{}
	for _, s := range snap {
		if _, dup := seenKey[s.claimKey]; dup {
			gi, ok := groupAt[s.claimKey]
			if !ok {
				first := services[seenKey[s.claimKey]]
				dupGroups = append(dupGroups, []string{first.key})
				gi = len(dupGroups) - 1
				groupAt[s.claimKey] = gi
			}
			dupGroups[gi] = append(dupGroups[gi], s.key)
			continue
		}
		seenKey[s.claimKey] = len(services)
		services = append(services, s)
	}
	v.dups = dupGroups
	slugs := assignSlugs(services)
	multi := len(services) >= 2

	inView := map[string]bool{} // claim key of services with a complete list now
	labelOf := map[string]string{}
	heldNames := map[string]map[string]bool{} // claim key -> names its held tools could take
	// What each service offers, screened once: the tools that hold none of its
	// credentials and, for a service whose credentials could not be read in
	// full, only those that passed the check before with the same bytes.
	screens := make([]toolScreen, len(services))
	for i, s := range services {
		labelOf[s.claimKey] = s.label
		if s.complete {
			screens[i] = m.screenTools(s, acct, readComplete, keptOK, claimsOK, epoch)
		}
		switch {
		case s.complete && screens[i].held == 0:
			inView[s.claimKey] = true
			if m.answered == nil {
				m.answered = map[string]bool{}
			}
			m.answered[s.claimKey] = true
		case s.complete:
			// Tools of this service are held back until the host can read its
			// credentials in full. The host says when that changes
			// (SetCredentialsUnreadable), which builds the view again; an
			// unlisted name may be one of the held tools. The service is not
			// recorded as answered (not all its tools are named), and a later
			// service's new tool waits only if it wants a plain name that a
			// held tool could take.
			heldNames[s.claimKey] = screens[i].heldNames
			v.held = true
			v.unavailable = true
			v.incomplete = true
		default:
			v.missing = true
			v.unavailable = true
			v.incomplete = true
		}
	}
	// The tools each service offers.
	var all []*toolEntry
	offers := make([]map[string]bool, len(services))
	for i, s := range services {
		offers[i] = map[string]bool{}
		if !s.complete {
			continue
		}
		for _, it := range screens[i].items {
			offers[i][strings.ToLower(it.name)] = true
			all = append(all, &toolEntry{service: s.key, label: s.label, claimKey: s.claimKey, original: it.name,
				client: s.client, raw: it.raw, toolIdx: it.idx, svcIdx: i})
		}
		m.noteRefused(s.claimKey, s.label, screens[i].refused)
	}

	taken := map[string]bool{} // lower(name) held by anything
	for l := range byLower {
		taken[l] = true
	}
	// Phase A: tools that already have a name keep it.
	var fresh []*toolEntry
	for _, e := range all {
		if a, ok := byTool[toolBindingKey(e.claimKey, e.original)]; ok {
			e.exposed, e.reason, e.peer = a.Name, a.Reason, a.Peer
			continue
		}
		fresh = append(fresh, e)
	}

	// Who must answer before this tool may take a new name.
	waitSeen := map[ToolWait]bool{}
	wait := func(e *toolEntry, forLabel string) {
		if forLabel == "" {
			return
		}
		w := ToolWait{Service: e.label, WaitingFor: forLabel}
		if !waitSeen[w] {
			waitSeen[w] = true
			v.waiting = append(v.waiting, w)
			m.noteWaiting(w, durable)
		}
	}
	// durableBlocker: in a durable order, the first earlier service (that can
	// answer) not yet seen.
	durableBlocker := func(e *toolEntry) (blocked bool, label string) {
		if !claimsOK || co == nil {
			return true, ""
		}
		pos := -1
		for i, s := range co.Services {
			if s.ClaimKey == e.claimKey {
				pos = i
				break
			}
		}
		if pos < 0 {
			return true, "" // not in the order: it appears next cycle
		}
		for _, s := range co.Services[:pos] {
			if s.Free {
				continue
			}
			if hn, held := heldNames[s.ClaimKey]; held {
				// Held tools are new or changed ones: even for a service seen
				// before, a later service does not take a plain name they
				// could take. It does not wait for anything else.
				if !wantsHeldName(hn, e.original) {
					continue
				}
			} else if inView[s.ClaimKey] || seenSet[s.ClaimKey] {
				continue
			}
			l := labelOf[s.ClaimKey]
			if l == "" {
				l = s.Label
			}
			if l == "" {
				l = s.Key
			}
			return true, l
		}
		return false, ""
	}
	// staticBlocker: with no durable order, an earlier service (or the one a
	// routing rule favours) that has not answered in this process.
	unanswered := func(s serviceSnap) bool { return !m.answered[s.claimKey] }
	staticBlocker := func(e *toolEntry) (blocked bool, label string) {
		for _, rule := range m.routing {
			for _, p := range rule.Tools {
				if !matchToolPattern(p, e.original) {
					continue
				}
				for _, s := range services {
					if s.key != rule.Upstream {
						continue
					}
					if s.key == e.service {
						return false, "" // the rule favours this service
					}
					if unanswered(s) {
						return true, s.label
					}
				}
			}
		}
		for _, s := range services[:e.svcIdx] {
			if hn, held := heldNames[s.claimKey]; held {
				if wantsHeldName(hn, e.original) {
					return true, s.label
				}
				continue
			}
			if unanswered(s) {
				return true, s.label
			}
		}
		return false, ""
	}

	// A sign-in written since the screen began: do not name or store anything
	// from it (the caller builds again).
	if !m.checked.Current(v.recEpoch) {
		v.stale = true
		return v
	}

	// Phase B: new tools, the ones a routing rule favours first, then by
	// service order and listing order.
	rank := func(e *toolEntry) int {
		for _, rule := range m.routing {
			for _, p := range rule.Tools {
				if !matchToolPattern(p, e.original) {
					continue
				}
				for i, s := range services {
					if s.key == rule.Upstream && offers[i][strings.ToLower(e.original)] {
						if e.svcIdx == i {
							return 0
						}
						return 1
					}
				}
			}
		}
		return 1
	}
	ranks := make(map[*toolEntry]int, len(fresh))
	for _, e := range fresh {
		ranks[e] = rank(e)
	}
	sort.SliceStable(fresh, func(i, j int) bool {
		a, b := fresh[i], fresh[j]
		if ranks[a] != ranks[b] {
			return ranks[a] < ranks[b]
		}
		if a.svcIdx != b.svcIdx {
			return a.svcIdx < b.svcIdx
		}
		return a.toolIdx < b.toolIdx
	})
	for _, e := range fresh {
		var blocked bool
		var who string
		if durable {
			blocked, who = durableBlocker(e)
		} else {
			blocked, who = staticBlocker(e)
		}
		if !blocked {
			continue
		}
		// Not listed until the earlier service has answered. No temporary
		// name is offered: a name that a later start could take back is
		// never listed.
		e.blocked = true
		wait(e, who)
		v.unavailable = true
		v.incomplete = true
		if who == "" {
			v.fastRetry = true // the order could not be read: try again soon
		}
	}

	otherSlugs := func(i int) []string {
		var out []string
		for j, s := range slugs {
			if j != i {
				out = append(out, s)
			}
		}
		return out
	}
	failed := map[*toolEntry]bool{}
	claimErr := false // a store error stops all claiming this cycle
	assign := func(e *toolEntry, plainPass bool) {
		if claimErr {
			return
		}
		c := candidateNames(slugs[e.svcIdx], e.original, otherSlugs(e.svcIdx))
		var cands []string
		if plainPass {
			if c.plain == "" {
				return
			}
			cands = []string{c.plain}
		} else {
			cands = c.prefixed
		}
		for _, cand := range cands {
			l := strings.ToLower(cand)
			if taken[l] {
				continue
			}
			a := ToolNameAssignment{Name: cand, Address: e.claimKey, Original: e.original, Service: e.label}
			switch {
			case c.invalid:
				a.Reason = ToolRenamedInvalid
			case cand != e.original && c.lookalike:
				a.Reason = ToolRenamedLookalike
			case cand != e.original:
				a.Reason = ToolRenamedClash
				if holder, ok := byLower[strings.ToLower(e.original)]; ok {
					a.Peer = holder.Service
				}
			}
			got, ok, err := store.Claim(ctx, a)
			if err != nil {
				log.Warn().Msg("tool names: a name could not be stored; the tool is not listed until it can")
				failed[e] = true
				claimErr = true
				v.incomplete = true
				v.fastRetry = true
				return
			}
			if !ok {
				taken[l] = true // another machine took it first
				continue
			}
			e.exposed, e.reason, e.peer = got.Name, got.Reason, got.Peer
			taken[strings.ToLower(got.Name)] = true
			byLower[strings.ToLower(got.Name)] = got
			return
		}
	}
	// Plain names first, so that no prefixed name can take a plain name that
	// a later tool is entitled to.
	for _, e := range fresh {
		if !e.blocked {
			assign(e, true)
		}
	}
	for _, e := range fresh {
		if e.blocked {
			continue
		}
		if e.exposed == "" && !failed[e] {
			assign(e, false)
		}
		if e.exposed == "" && !failed[e] && !claimErr {
			failed[e] = true
			v.incomplete = true
		}
	}

	// A service whose tools all have names is recorded as seen, durably when
	// the store can. A zero-tool service counts once it has been discovered.
	if durable && hasSeen && !claimErr {
		unnamed := map[string]bool{}
		for _, e := range all {
			if e.exposed == "" {
				unnamed[e.claimKey] = true
			}
		}
		for key := range inView {
			if unnamed[key] || m.seenLocal[key] {
				continue
			}
			if err := seenStore.MarkSeen(ctx, key); err != nil {
				log.Warn().Msg("tool names: an answered service could not be recorded; it is tried again soon")
				v.incomplete = true
				v.fastRetry = true
				continue
			}
			if m.seenLocal == nil {
				m.seenLocal = map[string]bool{}
			}
			m.seenLocal[key] = true
		}
	}

	// Render in service order, then each service's own order.
	sort.SliceStable(all, func(i, j int) bool {
		if all[i].svcIdx != all[j].svcIdx {
			return all[i].svcIdx < all[j].svcIdx
		}
		return all[i].toolIdx < all[j].toolIdx
	})
	clashCount := map[string]int{}
	for _, e := range all {
		if e.exposed != "" {
			clashCount[strings.ToLower(e.original)]++
		}
	}
	for _, e := range all {
		// View admission: a stored name that is reserved is never listed.
		if e.exposed == "" || IsReservedAgentToolName(e.exposed) {
			continue
		}
		for j, sl := range slugs {
			if j != e.svcIdx && strings.HasPrefix(strings.ToLower(e.exposed), sl+"_") {
				e.looksLike = services[j].label
				break
			}
		}
		prefix := multi || clashCount[strings.ToLower(e.original)] > 1 || e.reason == ToolRenamedClash
		e.raw = renderTool(e.raw, e.exposed, e.exposed != e.original, prefix, e.label)
		v.entries = append(v.entries, e)
		v.byName[e.exposed] = e
		v.tools = append(v.tools, e.raw)
	}
	return v
}

// wantsHeldName reports whether a tool named original could take a plain name
// that one of an earlier service's held tools could take.
func wantsHeldName(held map[string]bool, original string) bool {
	return held[strings.ToLower(original)] || held[strings.ToLower(sanitizeToolName(original))]
}

// toolScreen is the result of screening one service's tools.
type toolScreen struct {
	items   []screenedTool
	refused int // hold one of the service's credentials
	held    int // not checkable now: the credential set was not read in full and the tool is new or changed
	// heldNames are the lower-case plain names the held tools could take (their
	// own name and its sanitized form). A later service's new tool that wants
	// one of these waits: the plain name belongs to the earlier service.
	heldNames map[string]bool
}

type screenedTool struct {
	raw  json.RawMessage
	idx  int
	name string
}

// screenTools applies the credential check to one service's complete list. A
// tool that holds one of the service's credentials, or any sign-in value of
// the account (acct), is never listed, whichever service issued the value.
// When the host could not read every sign-in of the account in full
// (readComplete false), a tool is listed only if keptOK says the time limit
// has not passed and it passed before with the same name and description;
// the rest are held. When the read is complete (and record says the build's
// other reads succeeded), every tool that passes is recorded under epoch.
func (m *UpstreamManager) screenTools(s serviceSnap, acct []string, readComplete, keptOK, record bool, epoch uint64) toolScreen {
	var out toolScreen
	seen := map[string]bool{}
	var passed []toolDigest
	secrets := append(append([]string(nil), s.secrets...), acct...)
	for ti, raw := range s.tools {
		var probe struct {
			Name        json.RawMessage `json:"name"`
			Description json.RawMessage `json:"description"`
		}
		if json.Unmarshal(raw, &probe) != nil {
			continue
		}
		var name string
		if json.Unmarshal(probe.Name, &name) != nil || name == "" {
			continue
		}
		if IsReservedAgentToolName(name) || seen[name] {
			continue
		}
		seen[name] = true
		var desc string
		_ = json.Unmarshal(probe.Description, &desc)
		if holdsSecret(secrets, name, desc) {
			out.refused++
			continue
		}
		d := digestOfTool(name, desc)
		if !readComplete {
			if !keptOK || !m.checked.has(s.claimKey, d) {
				out.held++
				if out.heldNames == nil {
					out.heldNames = map[string]bool{}
				}
				out.heldNames[strings.ToLower(name)] = true
				out.heldNames[strings.ToLower(sanitizeToolName(name))] = true
				continue
			}
		} else {
			passed = append(passed, d)
		}
		out.items = append(out.items, screenedTool{raw: raw, idx: ti, name: name})
	}
	// Record only what passed against a read of the whole account's sign-ins.
	if readComplete && record {
		m.checked.replace(epoch, s.claimKey, passed)
	}
	return out
}

// noteWaiting logs once per pair that one service's tools are not listed until
// an earlier one answers, and what the owner can do about it. Display names
// only.
func (m *UpstreamManager) noteWaiting(w ToolWait, durable bool) {
	if durable {
		return // the host (the owner's panel) says it
	}
	if m.waitLogged == nil {
		m.waitLogged = map[ToolWait]bool{}
	}
	if m.waitLogged[w] {
		return
	}
	m.waitLogged[w] = true
	log.Warn().Str("service", w.Service).Str("waiting_for", w.WaitingFor).
		Msg("tool names: this service's tools are not listed until an earlier service answers; to stop waiting, turn the earlier service off or change the order")
}

// noteRefused logs, once per change of count, that tools were left out because
// they hold a credential of their own service. The line holds the service's
// display name and the count and nothing else.
func (m *UpstreamManager) noteRefused(claimKey, label string, n int) {
	if m.refusedLogged == nil {
		m.refusedLogged = map[string]int{}
	}
	if m.refusedLogged[claimKey] == n {
		return
	}
	m.refusedLogged[claimKey] = n
	if n > 0 {
		log.Warn().Str("service", label).Int("tools", n).
			Msg("tool names: tools left out because their name or description holds one of the service's credentials")
	}
}

// addressDigest8 is the first 8 hex characters of the digest of a service's
// address, for logs that must not hold the address.
func addressDigest8(rawURL string) string {
	k := ServiceAddressKey(UpstreamConfig{URL: rawURL})
	if len(k) > 8 {
		return k[:8]
	}
	return k
}
