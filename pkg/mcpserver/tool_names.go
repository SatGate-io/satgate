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
//     services share a slug. When a tenant has two or more services, every
//     description starts with the service's name ("Robinhood: ...").
//  3. A name never changes. Each assignment is kept in a ToolNameStore and is
//     bound to the service's identity (its address, or the claim key the
//     caller gives it). A restart, a reconnect, a rename of the service or a
//     new service never changes it. A removed service keeps its names
//     reserved; a different service never inherits them.
//  4. Every listed name matches ^[A-Za-z0-9_-]{1,64}$, the strictest rule of
//     the Anthropic and OpenAI APIs and the MCP specification. A tool whose own
//     name does not is given a valid stored name the same way as a clash.
//  5. Names that start with "satgate_" (any letter case) belong to the gateway
//     and are never taken from an upstream, whether the upstream offers them
//     as they are or a sanitized or prefixed form of them becomes one.
//  6. Every page of a service's tools/list is read before any name is chosen
//     (at most 50 pages; a repeated cursor stops the walk). A service whose
//     list is incomplete keeps its last complete list; a service that never
//     had one gets no new names and its tools are "not available yet".
//  7. A tool whose name or description holds a credential of its own service
//     (a URL query value, a header value, an OAuth token) is not named, not
//     stored and not listed.
//  8. A new plain name that starts with another service's slug and an
//     underscore is treated as a clash: it is listed under its own service's
//     prefix, so one service cannot pass as another.
//
// Who keeps the plain name is decided when a name is first assigned, in this
// order: an explicit routing rule that names a service which offers the tool;
// then the order of the services (SetServiceOrder; without one, the sorted
// service names). Once assigned, the order no longer matters.
//
// Who may take a plain name before an earlier service has answered:
//
//   - Static configuration (no durable store): while a service that comes
//     earlier in that order (routing rules first, then sorted keys) has no
//     complete tool list in this process, the tools of later services are not
//     listed, and a call to one gets ToolsNotAvailableYetMessage. No
//     temporary name is ever offered. The same configuration and the same
//     tool lists therefore give the same names whatever was up at start. The
//     cost: one earlier service that is down hides later services' tools
//     until it answers (turn it off, or change the order). Known limit:
//     without a database, if a tool list changes between restarts so that an
//     earlier service now offers a tool a later one already had, the earlier
//     service takes the plain name at the next start.
//   - Durable store with a claim order (SetClaimOrderSource): a service claims
//     a new name only after every earlier service has been seen (it is in this
//     view, or an earlier complete discovery was recorded). Until then its
//     tools that need a new name are not listed; tools with stored names keep
//     working. If the order cannot be read, nothing new is claimed.

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
	// ToolRenamedPending: an earlier service had not answered when the name
	// was chosen, so the plain name was left for it. No new name gets this
	// reason any more (round 3: a name that a later start could take back is
	// never listed); it stays so that names stored by an earlier version are
	// still understood.
	ToolRenamedPending = "pending"
	// ToolRenamedLookalike: the plain name started with another service's
	// slug, so it could pass as that service's tool.
	ToolRenamedLookalike = "lookalike"
)

// ToolsNotAvailableYetMessage is what an agent gets for a tool that is not
// listed while some service's tools are not all in yet.
const ToolsNotAvailableYetMessage = "this service's tools aren't available yet, try again in a minute"

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

// IsReservedAgentToolName reports whether a candidate name may never be given
// to an upstream's tool: the gateway's own names (satgate_..., any letter
// case) and the argument-rules prefix.
func IsReservedAgentToolName(name string) bool {
	return IsGatewayToolName(name) || macaroon.IsReservedToolName(name)
}

// ToolNameAssignment is one stored name. Name is what agents see; the pair
// (Address, Original) is the tool it stands for. Both Name (compared without
// regard to case) and (Address, Original) are unique in a store.
type ToolNameAssignment struct {
	// Name is the name agents see.
	Name string
	// Address is the service's claim key (ServiceKey): a digest, never a URL.
	// It is ServiceAddressKey of the service unless the caller gave the
	// service a key of its own (two rows at one address).
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
	if IsReservedAgentToolName(a.Name) {
		return ToolNameAssignment{}, false, nil
	}
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

// ToolSeenStore is an optional part of a ToolNameStore that records which
// services have been fully discovered at least once, by claim key. A durable
// claim order needs it (see SetClaimOrderSource).
type ToolSeenStore interface {
	// LoadSeen returns the claim keys that were recorded as seen.
	LoadSeen(ctx context.Context) (map[string]bool, error)
	// MarkSeen records that the service with this claim key had a complete
	// discovery. It is idempotent.
	MarkSeen(ctx context.Context, claimKey string) error
}

// ClaimService is one enabled service in connection order.
type ClaimService struct {
	// Key is the service's key in the manager.
	Key string
	// Label is its display name, for the wait message when the manager does
	// not hold the service.
	Label string
	// ClaimKey is the key its names are stored under (ServiceKey).
	ClaimKey string
	// Free marks a service that the database shows its owner never finished
	// setting up (it is set to sign in and has no sign-in on file). It never
	// makes a later service wait. Nothing else sets it: not a stored
	// needs_signin, not a health status, not one machine's refusal at load,
	// not a read error. Every machine reads it from the database the same way.
	Free bool
	// Secrets are the service's current credential values the host read in
	// this cycle (stored sign-in tokens), added to the ones in its config for
	// the check on names and descriptions.
	Secrets []string
}

// ClaimOrder lists a tenant's enabled services in the order they were
// connected.
type ClaimOrder struct {
	Services []ClaimService
	// Unreadable lists the keys (in the manager) of the services whose stored
	// sign-ins the host could not read in full in this read, and AllUnreadable
	// says that none could be read. They are honoured even when the read
	// returns an error. When the read succeeds they are the host's answer for
	// this build and replace what it said earlier with
	// SetCredentialsUnreadable; on an error they add to it. For such a service
	// only tools that passed the credential check before, with the same name
	// and description, are listed; the rest wait (tool_names_checked.go).
	Unreadable    []string
	AllUnreadable bool
}

// ServiceKey is the key a service's names are stored under: the claim key the
// caller gave it, else the digest of its address.
func ServiceKey(cfg UpstreamConfig) string {
	if cfg.NameKey != "" {
		return cfg.NameKey
	}
	return ServiceAddressKey(cfg)
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

// ServiceSlug is the short lowercase form of a service's name used in prefixed
// tool names. The owner's panel uses it to spot names that look like another
// service's.
func ServiceSlug(label string) string { return serviceSlug(label) }

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
