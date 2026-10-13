package mcpserver

import (
	"crypto/sha256"
	"sync"
)

// The passed-check record (tool names: keep tools that already passed).
//
// A tool is listed only after its name and description were checked against
// the service's credentials (tool_names_view.go, holdsSecret). When the host
// cannot read a service's complete credential set at some moment (a slow or
// failed read of stored sign-ins), the check cannot be done for that service
// then. What was already checked does not need doing again: a tool whose name
// and description are, byte for byte, what they were when it passed stays
// listed. Every other tool of that service (new, or changed since) is held
// back until a later view is built with the complete credential set.
//
// The record holds a digest of each passed tool (name and description), never
// the text. It lives in memory, per host process: a process that never had a
// complete credential set for a service has nothing to keep for it and holds
// back all of that service's tools.

// toolDigest identifies a tool by the bytes the check looked at.
type toolDigest [32]byte

func digestOfTool(name, description string) toolDigest {
	h := sha256.New()
	h.Write([]byte(name))
	h.Write([]byte{0})
	h.Write([]byte(description))
	var d toolDigest
	copy(d[:], h.Sum(nil))
	return d
}

// ToolCheckRecord remembers which tools of each service passed the credential
// check. A host that rebuilds managers (a tenant's configuration changes) can
// give each new manager the record of the one before it (SetToolCheckRecord),
// so a rebuild does not lose what was checked. The zero value is not usable:
// use NewToolCheckRecord. Safe for concurrent use.
type ToolCheckRecord struct {
	mu     sync.Mutex
	passed map[string]map[toolDigest]struct{} // service claim key -> digests
}

// NewToolCheckRecord returns an empty record.
func NewToolCheckRecord() *ToolCheckRecord {
	return &ToolCheckRecord{passed: map[string]map[toolDigest]struct{}{}}
}

func (r *ToolCheckRecord) has(claimKey string, d toolDigest) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.passed[claimKey][d]
	return ok
}

// replace sets the service's passed tools to exactly ds: the tools that passed
// the check in the latest view built with a complete credential set. A tool
// that left the service's list is forgotten, so a tool that comes back later
// is a new tool and is checked.
func (r *ToolCheckRecord) replace(claimKey string, ds []toolDigest) {
	set := make(map[toolDigest]struct{}, len(ds))
	for _, d := range ds {
		set[d] = struct{}{}
	}
	r.mu.Lock()
	r.passed[claimKey] = set
	r.mu.Unlock()
}

// Count says how many tools of the service are on record (for tests and logs).
func (r *ToolCheckRecord) Count(claimKey string) int {
	if r == nil {
		return 0
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.passed[claimKey])
}

// SetToolCheckRecord gives the manager the record to read and write. Without
// one the manager keeps its own, for the life of the manager. Call it before
// Start.
func (m *UpstreamManager) SetToolCheckRecord(r *ToolCheckRecord) {
	m.viewMu.Lock()
	m.checked = r
	m.viewMu.Unlock()
	m.namesVer.Add(1)
}

// SetCredentialsUnreadable says which services (by their key in the manager)
// had a credential set the host could not read completely just now. It
// replaces the earlier answer; an empty list says every service is readable.
// The services' configuration is not touched, so nothing reconnects. A claim
// order (SetClaimOrderSource) that is present in a build answers for that
// build instead (ClaimOrder.Unreadable, ClaimOrder.AllUnreadable).
//
// For a listed service, a tool is listed only if it passed the check before
// with the same name and description (ToolCheckRecord) and still holds none of
// the credential values the manager has. Every other tool of the service is
// held back: not named, not stored, not listed, not routed. A change of the
// answer makes the next list be built again.
func (m *UpstreamManager) SetCredentialsUnreadable(keys []string) {
	next := make(map[string]bool, len(keys))
	for _, k := range keys {
		next[k] = true
	}
	m.mu.Lock()
	same := len(next) == len(m.unreadable)
	if same {
		for k := range next {
			if !m.unreadable[k] {
				same = false
				break
			}
		}
	}
	if !same {
		m.unreadable = next
	}
	m.mu.Unlock()
	if !same {
		m.namesVer.Add(1)
	}
}
