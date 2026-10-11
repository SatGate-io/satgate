package mcpserver

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/satgate-io/satgate/pkg/macaroon"
)

// Unique tool names across upstreams.
//
// An agent with several services connected (two brokers that both offer
// place_crypto_order) needs one name per tool: Claude's API refuses a tool
// list that repeats a name, and a call must reach the service the list
// described. This file is the one place that decides every name an agent
// sees. tools/list (AllTools) and tools/call (ForwardToolCall,
// ResolveUpstream) both read the same view, built here.
//
// The rules:
//
//  1. Every listed name is unique, compared without regard to letter case, and
//     maps to exactly one (service, tool) pair.
//  2. The service that first offered a name keeps the plain name. Another
//     service's tool with the same name is listed as <service>_<tool>, with a
//     short lowercase slug of the service's name; a number is added when two
//     services share a slug. While a name clashes, both descriptions start with
//     the service's name ("Robinhood: ...").
//  3. A name never changes. Each assignment is kept in a ToolNameStore and is
//     bound to the service's address. A restart, a reconnect, a rename of the
//     service or a new service never changes it. A removed service keeps its
//     names reserved; a different service never inherits them.
//  4. Every listed name matches ^[A-Za-z0-9_-]{1,64}$, the strictest rule of
//     the Anthropic and OpenAI APIs and the MCP specification. A tool whose own
//     name does not is given a valid stored name the same way as a clash.
//  5. Tools whose names start with "satgate_" belong to the gateway and are
//     never taken from an upstream.
//
// Who keeps the plain name is decided when a name is first assigned, in this
// order: an explicit routing rule that names a service which offers the tool;
// then the order of the services (SetServiceOrder; without one, the sorted
// service names). Once assigned, the order no longer matters.

// MaxAgentToolNameLen is the longest tool name every agent app accepts: the
// OpenAI function-name limit (Anthropic allows 128 and the MCP specification
// 128).
const MaxAgentToolNameLen = 64

// GatewayToolPrefix starts the names of tools the gateway answers itself
// (satgate_budget_check). An upstream cannot offer them.
const GatewayToolPrefix = "satgate_"

// Why a tool is listed under a name other than its own.
const (
	// ToolRenamedClash: another tool already held the plain name.
	ToolRenamedClash = "clash"
	// ToolRenamedInvalid: the tool's own name is not accepted by agent apps.
	ToolRenamedInvalid = "invalid_name"
)

// ValidAgentToolName reports whether name is accepted by the Anthropic and
// OpenAI APIs: 1 to 64 characters from A-Z, a-z, 0-9, underscore and hyphen.
func ValidAgentToolName(name string) bool {
	if name == "" || len(name) > MaxAgentToolNameLen {
		return false
	}
	for i := 0; i < len(name); i++ {
		c := name[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '_', c == '-':
		default:
			return false
		}
	}
	return true
}

// IsGatewayToolName reports whether a tool name belongs to the gateway.
func IsGatewayToolName(name string) bool {
	return len(name) >= len(GatewayToolPrefix) && strings.EqualFold(name[:len(GatewayToolPrefix)], GatewayToolPrefix)
}

// ToolNameAssignment is one stored name. Name is what agents see; the pair
// (Address, Original) is the tool it stands for. Both Name (compared without
// regard to case) and (Address, Original) are unique in a store.
type ToolNameAssignment struct {
	// Name is the name agents see.
	Name string
	// Address is ServiceAddressKey of the service: a digest, never a URL.
	Address string
	// Original is the tool's own name at that service.
	Original string
	// Service is the service's display name when the name was assigned. It is
	// for people (the owner's panel) and never decides anything.
	Service string
	// Reason is "" (the plain name), ToolRenamedClash or ToolRenamedInvalid.
	Reason string
	// Peer is the display name of the service that held the plain name when a
	// clash was settled; "" otherwise.
	Peer string
}

// Renamed reports whether the tool is listed under another name than its own.
func (a ToolNameAssignment) Renamed() bool { return a.Name != a.Original }

// ToolNameStore keeps assignments for one namespace (one tenant). The gateway
// of one machine and the gateway of another read and write the same store, so
// a store must make Claim atomic: of two concurrent claims for the same name,
// or the same tool, one wins and the other reads the winner back.
type ToolNameStore interface {
	// Load returns every assignment, including those of services that are no
	// longer connected.
	Load(ctx context.Context) ([]ToolNameAssignment, error)
	// Claim records a. It returns:
	//   (stored, true, nil)  the tool (Address, Original) has a name: a's, or
	//                        the one already stored for it. Use stored.Name.
	//   (zero, false, nil)   a.Name belongs to another tool; propose another.
	Claim(ctx context.Context, a ToolNameAssignment) (ToolNameAssignment, bool, error)
}

// NewMemoryToolNameStore returns a store that lives as long as the process.
// It is what a manager uses when no other store is set (self-hosted gateways
// with static configuration).
func NewMemoryToolNameStore() ToolNameStore {
	return &memoryToolNameStore{byName: map[string]ToolNameAssignment{}, byTool: map[string]ToolNameAssignment{}}
}

type memoryToolNameStore struct {
	mu     sync.Mutex
	byName map[string]ToolNameAssignment // lower(name)
	byTool map[string]ToolNameAssignment // address \x00 original
}

func toolBindingKey(address, original string) string { return address + "\x00" + original }

func (s *memoryToolNameStore) Load(context.Context) ([]ToolNameAssignment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]ToolNameAssignment, 0, len(s.byName))
	for _, a := range s.byName {
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (s *memoryToolNameStore) Claim(_ context.Context, a ToolNameAssignment) (ToolNameAssignment, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if have, ok := s.byTool[toolBindingKey(a.Address, a.Original)]; ok {
		return have, true, nil
	}
	if _, taken := s.byName[strings.ToLower(a.Name)]; taken {
		return ToolNameAssignment{}, false, nil
	}
	s.byName[strings.ToLower(a.Name)] = a
	s.byTool[toolBindingKey(a.Address, a.Original)] = a
	return a, true, nil
}

// ServiceAddress is the identity of a service for naming: where it is, not
// what it is called or how it signs in. For a URL it is the scheme, host (lower
// case, default port removed), path (no trailing slash) and query, without
// credentials or fragment. For a command it is the command line. The transport
// is not part of it. The result can hold a secret that sits in a query string:
// store ServiceAddressKey, never this.
func ServiceAddress(cfg UpstreamConfig) string {
	if strings.TrimSpace(cfg.URL) == "" {
		if len(cfg.Command) == 0 {
			return ""
		}
		return "cmd:" + strings.Join(cfg.Command, "\x00")
	}
	u, err := url.Parse(strings.TrimSpace(cfg.URL))
	if err != nil || u.Host == "" {
		return "url:" + strings.TrimSpace(cfg.URL)
	}
	scheme := strings.ToLower(u.Scheme)
	host := strings.ToLower(u.Hostname())
	port := u.Port()
	if (scheme == "https" && port == "443") || (scheme == "http" && port == "80") {
		port = ""
	}
	if port != "" {
		host += ":" + port
	}
	path := strings.TrimRight(u.EscapedPath(), "/")
	out := scheme + "://" + host + path
	if u.RawQuery != "" {
		out += "?" + u.RawQuery
	}
	return out
}

// ServiceAddressKey is a digest of ServiceAddress, safe to store and compare.
// An empty address gives "".
func ServiceAddressKey(cfg UpstreamConfig) string {
	a := ServiceAddress(cfg)
	if a == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(a))
	return hex.EncodeToString(sum[:])
}

// ToolNameInfo describes one listed tool, for the owner's panel and for tests.
type ToolNameInfo struct {
	Service  string // the service's display name
	Key      string // the service's key in the manager
	Original string // the tool's own name
	Name     string // the name agents see
	Renamed  bool
	Reason   string // "", ToolRenamedClash or ToolRenamedInvalid
	Peer     string // the service that holds the plain name, for a clash
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
// before Start.
func (m *UpstreamManager) SetServiceOrder(keys []string) {
	m.viewMu.Lock()
	m.order = append([]string(nil), keys...)
	m.viewMu.Unlock()
	m.namesVer.Add(1)
}

// ToolNames returns every tool the manager lists, in listing order.
func (m *UpstreamManager) ToolNames() []ToolNameInfo {
	v := m.toolView()
	out := make([]ToolNameInfo, 0, len(v.entries))
	for _, e := range v.entries {
		out = append(out, ToolNameInfo{Service: e.label, Key: e.service, Original: e.original,
			Name: e.exposed, Renamed: e.exposed != e.original, Reason: e.reason, Peer: e.peer})
	}
	return out
}

// DuplicateServices returns groups of service keys that share one address. The
// first of a group is served; the others offer no tools to agents.
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

// ---------------------------------------------------------------------------
// the view

// serviceSnap is one connected service, as read for a view.
type serviceSnap struct {
	key     string
	label   string
	address string // ServiceAddressKey
	client  *UpstreamClient
	tools   []json.RawMessage
}

type toolEntry struct {
	service  string // manager key
	label    string
	address  string
	original string
	exposed  string
	reason   string
	peer     string
	client   *UpstreamClient
	raw      json.RawMessage // as listed
	toolIdx  int
	svcIdx   int
}

type toolView struct {
	ver        uint64
	at         time.Time
	incomplete bool // a name could not be stored; try again soon
	entries    []*toolEntry
	byName     map[string]*toolEntry // exposed, exact
	tools      []json.RawMessage
	dups       [][]string
}

const viewRetryAfter = 2 * time.Second

func (m *UpstreamManager) snapshotServices() []serviceSnap {
	m.mu.Lock()
	defer m.mu.Unlock()
	keys := make([]string, 0, len(m.clients))
	for k := range m.clients {
		keys = append(keys, k)
	}
	snaps := make(map[string]serviceSnap, len(keys))
	for _, k := range keys {
		c := m.clients[k]
		cfg := m.config[k]
		label := strings.TrimSpace(cfg.DisplayName)
		if label == "" {
			label = k
		}
		addr := ServiceAddressKey(cfg)
		if addr == "" {
			// No address to bind a name to (an upstream with neither URL nor
			// command). Bind to the key, so it still gets stable names.
			sum := sha256.Sum256([]byte("key:" + k))
			addr = hex.EncodeToString(sum[:])
		}
		snaps[k] = serviceSnap{key: k, label: label, address: addr, client: c, tools: append([]json.RawMessage(nil), c.tools...)}
	}
	sort.Strings(keys)
	rank := map[string]int{}
	for i, k := range m.order {
		if _, dup := rank[k]; !dup {
			rank[k] = i
		}
	}
	sort.SliceStable(keys, func(i, j int) bool {
		ri, iok := rank[keys[i]]
		rj, jok := rank[keys[j]]
		switch {
		case iok && jok:
			return ri < rj
		case iok:
			return true
		case jok:
			return false
		}
		return keys[i] < keys[j]
	})
	out := make([]serviceSnap, 0, len(keys))
	for _, k := range keys {
		out = append(out, snaps[k])
	}
	return out
}

// toolView returns the current view, building it when services changed since
// the last one.
func (m *UpstreamManager) toolView() *toolView {
	if v := m.viewPtr.Load(); v != nil && v.ver == m.namesVer.Load() && !v.incomplete {
		return v
	}
	m.viewMu.Lock()
	defer m.viewMu.Unlock()
	ver := m.namesVer.Load()
	if v := m.viewPtr.Load(); v != nil && v.ver == ver && (!v.incomplete || time.Since(v.at) < viewRetryAfter) {
		return v
	}
	snap := m.snapshotServices()
	if m.nameStore == nil {
		m.nameStore = NewMemoryToolNameStore()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	nv := m.buildToolView(ctx, snap)
	nv.ver = ver
	nv.at = time.Now()
	m.viewPtr.Store(nv)
	return nv
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
		if prev := m.viewPtr.Load(); prev != nil {
			keep := *prev
			keep.incomplete = true
			return &keep
		}
		v.incomplete = true
		return v
	}
	byTool := map[string]ToolNameAssignment{}
	byLower := map[string]ToolNameAssignment{}
	for _, a := range assigned {
		byTool[toolBindingKey(a.Address, a.Original)] = a
		byLower[strings.ToLower(a.Name)] = a
	}

	// Services in order; a second service at an address already served offers
	// nothing (it is the same service connected twice).
	var services []serviceSnap
	seenAddr := map[string]int{}
	var dupGroups [][]string
	groupAt := map[string]int{}
	for _, s := range snap {
		if _, dup := seenAddr[s.address]; dup {
			gi, ok := groupAt[s.address]
			if !ok {
				first := services[seenAddr[s.address]]
				dupGroups = append(dupGroups, []string{first.key})
				gi = len(dupGroups) - 1
				groupAt[s.address] = gi
			}
			dupGroups[gi] = append(dupGroups[gi], s.key)
			continue
		}
		seenAddr[s.address] = len(services)
		services = append(services, s)
	}
	v.dups = dupGroups

	// Slugs, unique among the connected services.
	slugs := make([]string, len(services))
	usedSlug := map[string]bool{}
	for i, s := range services {
		base := serviceSlug(s.label)
		slug := base
		for n := 2; usedSlug[slug]; n++ {
			slug = fmt.Sprintf("%s%d", base, n)
		}
		usedSlug[slug] = true
		slugs[i] = slug
	}

	// The tools each service offers.
	var all []*toolEntry
	offers := make([]map[string]bool, len(services)) // lower(original) offered by service i
	for i, s := range services {
		offers[i] = map[string]bool{}
		seen := map[string]bool{}
		for ti, raw := range s.tools {
			var probe struct {
				Name json.RawMessage `json:"name"`
			}
			if json.Unmarshal(raw, &probe) != nil {
				continue
			}
			var name string
			if json.Unmarshal(probe.Name, &name) != nil || name == "" {
				continue
			}
			if IsGatewayToolName(name) || macaroon.IsReservedToolName(name) || seen[name] {
				continue
			}
			seen[name] = true
			offers[i][strings.ToLower(name)] = true
			all = append(all, &toolEntry{service: s.key, label: s.label, address: s.address, original: name,
				client: s.client, raw: raw, toolIdx: ti, svcIdx: i})
		}
	}

	taken := map[string]bool{} // lower(name) held by anything
	for l := range byLower {
		taken[l] = true
	}
	// Phase A: tools that already have a name keep it.
	var fresh []*toolEntry
	for _, e := range all {
		if a, ok := byTool[toolBindingKey(e.address, e.original)]; ok {
			e.exposed, e.reason, e.peer = a.Name, a.Reason, a.Peer
			continue
		}
		fresh = append(fresh, e)
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
	failed := map[*toolEntry]bool{}
	assign := func(e *toolEntry, plainOnly bool) {
		valid := ValidAgentToolName(e.original)
		cands := candidateNames(slugs[e.svcIdx], e.original)
		if plainOnly {
			cands = cands[:1]
		}
		for _, cand := range cands {
			l := strings.ToLower(cand)
			if taken[l] {
				continue
			}
			a := ToolNameAssignment{Name: cand, Address: e.address, Original: e.original, Service: e.label}
			switch {
			case !valid:
				a.Reason = ToolRenamedInvalid
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
				v.incomplete = true
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
		if ValidAgentToolName(e.original) {
			assign(e, true)
		}
	}
	for _, e := range fresh {
		if e.exposed == "" && !failed[e] {
			assign(e, false)
		}
		if e.exposed == "" && !failed[e] {
			failed[e] = true
			v.incomplete = true
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
		if e.exposed == "" {
			continue
		}
		prefix := clashCount[strings.ToLower(e.original)] > 1 || e.reason == ToolRenamedClash
		e.raw = renderTool(e.raw, e.exposed, e.exposed != e.original, prefix, e.label)
		v.entries = append(v.entries, e)
		v.byName[e.exposed] = e
		v.tools = append(v.tools, e.raw)
	}
	return v
}

// candidateNames lists, in order of preference, the names a tool may take. A
// tool with a valid own name tries it first. Then come the service-prefixed
// name and numbered forms of it. The list is a pure function of its inputs, so
// every machine proposes the same names.
func candidateNames(slug, original string) []string {
	var out []string
	add := func(s string) {
		if ValidAgentToolName(s) {
			out = append(out, s)
		}
	}
	base := sanitizeToolName(original)
	if ValidAgentToolName(original) {
		add(original)
	} else {
		add(fitToolName(base, original))
	}
	prefixed := fitToolName(slug+"_"+base, original)
	add(prefixed)
	for n := 2; n <= 1000; n++ {
		add(numberedToolName(prefixed, n))
	}
	return out
}

// sanitizeToolName replaces every character agent apps do not accept with an
// underscore.
func sanitizeToolName(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_', r == '-':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	if b.Len() == 0 {
		return "tool"
	}
	return b.String()
}

// fitToolName shortens s to 64 characters, ending in a hash of seed so that
// two long names that share a start stay different.
func fitToolName(s, seed string) string {
	if len(s) <= MaxAgentToolNameLen {
		return s
	}
	sum := sha256.Sum256([]byte(seed))
	return s[:MaxAgentToolNameLen-9] + "_" + hex.EncodeToString(sum[:4])
}

func numberedToolName(s string, n int) string {
	suffix := fmt.Sprintf("_%d", n)
	if len(s)+len(suffix) > MaxAgentToolNameLen {
		s = s[:MaxAgentToolNameLen-len(suffix)]
	}
	return s + suffix
}

// serviceSlug is a short lowercase form of a service's name: a-z, 0-9, _.
func serviceSlug(label string) string {
	var b strings.Builder
	under := false
	for _, r := range strings.ToLower(label) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			under = false
			continue
		}
		if !under && b.Len() > 0 {
			b.WriteByte('_')
			under = true
		}
	}
	s := strings.TrimRight(b.String(), "_")
	if len(s) > 20 {
		s = strings.TrimRight(s[:20], "_")
	}
	if s == "" {
		return "svc"
	}
	return s
}

// renderTool returns the tool as agents see it. A tool that needs no change is
// returned byte for byte as the upstream listed it.
func renderTool(raw json.RawMessage, exposed string, rename, prefix bool, label string) json.RawMessage {
	if !rename && !prefix {
		return raw
	}
	var obj map[string]json.RawMessage
	if json.Unmarshal(raw, &obj) != nil {
		return raw
	}
	if rename {
		obj["name"] = marshalNoEscape(exposed)
	}
	if prefix {
		var desc string
		if d, ok := obj["description"]; ok {
			_ = json.Unmarshal(d, &desc)
		}
		if desc == "" {
			obj["description"] = marshalNoEscape(label)
		} else {
			obj["description"] = marshalNoEscape(label + ": " + desc)
		}
	}
	out, err := marshalRaw(obj)
	if err != nil {
		return raw
	}
	return out
}

func marshalNoEscape(v any) json.RawMessage {
	b, _ := marshalRaw(v)
	return b
}

func marshalRaw(v any) (json.RawMessage, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}

// renameCallParams returns params with the tool name replaced, every other
// field untouched.
func renameCallParams(params json.RawMessage, original string) (json.RawMessage, error) {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(params, &obj); err != nil {
		return nil, fmt.Errorf("tool call could not be read")
	}
	obj["name"] = marshalNoEscape(original)
	return marshalRaw(obj)
}
