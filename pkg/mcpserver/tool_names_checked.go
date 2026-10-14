package mcpserver

import (
	"crypto/sha256"
	"sync"
	"sync/atomic"
	"time"
)

// The passed-check record (tool names: keep tools that already passed).
//
// A tool is listed only after its name and description were checked against
// every sign-in value of the account (tool_names_view.go, holdsSecret). When
// the host cannot read all of the account's sign-ins at some moment (a slow or
// failed read, or one sign-in it cannot decrypt), that check cannot be done
// for new tools. What was already checked does not need doing again, for a
// while: a tool whose name and description are, byte for byte, what they were
// when it passed stays listed, for at most KeptToolsMaxAge after the account's
// last complete read, and is still screened on every build against the values
// of that last complete read. Every other tool of every service (new, or
// changed since) is held back until a view is built from a complete read.
//
// The record holds a digest of each passed tool (name and description), never
// the text, and the sign-in values of the last complete read, in memory. It
// lives in memory, per host process: a process that never had a complete read
// has nothing to keep and holds back all tools.

// KeptToolsMaxAge is how long after the account's last complete read of its
// sign-ins a tool that passed the check may stay listed while the sign-ins
// cannot be read in full. A failed read never extends it. After it, those
// tools are held until a complete read works.
const KeptToolsMaxAge = 10 * time.Minute

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

// ToolCheckRecord remembers, for one account, which tools of each service
// passed the credential check and the sign-in values (and time) of the last
// complete read. A host that rebuilds managers (a tenant's configuration
// changes) can give each new manager the record of the one before it
// (SetToolCheckRecord), so a rebuild does not lose what was checked. The zero
// value is not usable: use NewToolCheckRecord. Safe for concurrent use.
//
// Forget empties it and moves its epoch; a writer that started before the
// Forget (a read in flight, a build in flight) holds the old epoch and its
// write is dropped, so nothing old comes back.
type ToolCheckRecord struct {
	mu     sync.Mutex
	passed map[string]map[toolDigest]struct{} // service claim key -> digests
	epoch  atomic.Uint64                      // moved under mu; read without it
	values []string                           // sign-in values of the last complete read
	readAt time.Time                          // when that read finished; zero: none
}

// NewToolCheckRecord returns an empty record.
func NewToolCheckRecord() *ToolCheckRecord {
	return &ToolCheckRecord{passed: map[string]map[toolDigest]struct{}{}}
}

// Epoch says which generation of the record a reader or builder started in.
func (r *ToolCheckRecord) Epoch() uint64 { return r.epoch.Load() }

// Current says whether the record is still in generation epoch: nothing has
// forgotten it (Forget, ForgetSignIns) since. A view built in generation
// epoch is served only while this is true (toolView.valid).
func (r *ToolCheckRecord) Current(epoch uint64) bool { return r.epoch.Load() == epoch }

// Forget empties the record: passed tools, kept sign-in values and the time
// of the last complete read. The host calls it when the account's sign-ins
// or services changed under it (InvalidateTenant); tools then wait for a
// complete read, as on a cold start.
func (r *ToolCheckRecord) Forget() {
	r.mu.Lock()
	r.epoch.Add(1)
	r.passed = map[string]map[toolDigest]struct{}{}
	r.values, r.readAt = nil, time.Time{}
	r.mu.Unlock()
}

// ForgetSignIns drops the sign-in values and the time of the last complete
// read, and keeps which tools passed. The host calls it when a sign-in was
// written on this machine (a refreshed token): the values kept are then out of
// date, so tools kept only by the time limit wait for the next complete read.
// Tools are not held while reads work: the next list reads again. It moves
// the epoch like Forget does, so a view or a read that began earlier is not
// served or recorded.
func (r *ToolCheckRecord) ForgetSignIns() {
	r.mu.Lock()
	r.epoch.Add(1)
	r.values, r.readAt = nil, time.Time{}
	r.mu.Unlock()
}

// NoteCompleteRead records that the host read every sign-in of the account in
// full, ending at at; values are the access and refresh values of all
// services. epoch is the record's epoch when the
// read began; if the record was forgotten since, nothing is recorded and false
// is returned. A read that ended earlier than the one on record is ignored.
// A failed or partial read must never be noted: it does not extend
// KeptToolsMaxAge.
func (r *ToolCheckRecord) NoteCompleteRead(epoch uint64, at time.Time, values []string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if epoch != r.epoch.Load() {
		return false
	}
	if at.Before(r.readAt) {
		return true
	}
	r.values = append([]string(nil), values...)
	r.readAt = at
	return true
}

// kept returns the sign-in values of the last complete read and the time
// after which tools kept on its account are held; ok is false when there is
// no such read or it is KeptToolsMaxAge old or older at now.
func (r *ToolCheckRecord) kept(now time.Time) (values []string, until time.Time, ok bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.readAt.IsZero() {
		return nil, time.Time{}, false
	}
	until = r.readAt.Add(KeptToolsMaxAge)
	if !now.Before(until) {
		return nil, until, false
	}
	return append([]string(nil), r.values...), until, true
}

func (r *ToolCheckRecord) has(claimKey string, d toolDigest) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.passed[claimKey][d]
	return ok
}

// replace sets the service's passed tools to exactly ds: the tools that passed
// the check in the latest view built with a complete read. A tool that left
// the service's list is forgotten, so a tool that comes back later is a new
// tool and is checked. Dropped when the record was forgotten since epoch.
func (r *ToolCheckRecord) replace(epoch uint64, claimKey string, ds []toolDigest) {
	set := make(map[toolDigest]struct{}, len(ds))
	for _, d := range ds {
		set[d] = struct{}{}
	}
	r.mu.Lock()
	if epoch == r.epoch.Load() {
		r.passed[claimKey] = set
	}
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

// SetClock sets the clock the kept-tools time limit is measured with. The
// host's own times (NoteCompleteRead) must come from the same clock. Tests
// use it; the default is time.Now. Call it before Start.
func (m *UpstreamManager) SetClock(now func() time.Time) {
	if now == nil {
		m.clock.Store(nil)
	} else {
		m.clock.Store(&now)
	}
	m.namesVer.Add(1)
}

func (m *UpstreamManager) now() time.Time {
	if f := m.clock.Load(); f != nil {
		return (*f)()
	}
	return time.Now()
}

// SetCredentialsUnreadable says which services (by their key in the manager)
// had a credential set the host could not read completely just now. It
// replaces the earlier answer; an empty list says every sign-in is readable.
// The services' configuration is not touched, so nothing reconnects. A claim
// order (SetClaimOrderSource) that is present in a build answers for that
// build instead (ClaimOrder.Unreadable, ClaimOrder.AllUnreadable).
//
// One unreadable sign-in counts for the whole account: no tool of any
// service is checked against it, so a tool is listed only if it passed the
// check before with the same name and description (ToolCheckRecord), within
// KeptToolsMaxAge of the last complete read, and still holds none of the
// sign-in values known. Every other tool is held back: not named, not
// stored, not listed, not routed. A change of the answer makes the next list
// be built again.
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

// Recheck makes the next list be built again, so every listed tool is checked
// once more against the values the manager has now.
func (m *UpstreamManager) Recheck() { m.namesVer.Add(1) }
