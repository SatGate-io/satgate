package toolcontrols

import (
	"context"
	"errors"
	"sync"
	"time"
)

// ErrApprovalNotFound is returned by owner operations on an approval that does
// not exist for the tenant. A tenant never learns whether another tenant's
// approval exists: both cases are this error.
var ErrApprovalNotFound = errors.New("approval not found")

// ErrApprovalNotPending is returned when an approval is no longer waiting
// (already decided, used or expired).
var ErrApprovalNotPending = errors.New("approval is not waiting")

// Approval states.
const (
	StatePending  = "pending"
	StateApproved = "approved"
	StateDenied   = "denied"
	StateUsed     = "used"
	StateExpired  = "expired"
)

// MemoryApprovalStore is an ApprovalStore in process memory, with the owner
// side (Approve, Deny, Pending) a dashboard would call. It is correct for one
// process and loses its approvals on restart; it is for tests and for
// single-process use.
type MemoryApprovalStore struct {
	mu   sync.Mutex
	rows []*MemoryApproval
	seq  int
	// Err, when set, is returned by every call (a store that is down).
	Err error
}

// MemoryApproval is one row of the memory store.
type MemoryApproval struct {
	ID, TenantID, TokenID, Tool, CallHash, Field, Above string
	Summary                                             []SummaryItem
	State                                               string
	CreatedAt, ExpiresAt                                time.Time
	NotifiedAt                                          time.Time
	DecidedBy                                           string
}

// NewMemoryApprovalStore returns an empty store.
func NewMemoryApprovalStore() *MemoryApprovalStore { return &MemoryApprovalStore{} }

// Decide implements ApprovalStore.
func (m *MemoryApprovalStore) Decide(_ context.Context, req ApprovalRequest) (ApprovalDecision, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Err != nil {
		return ApprovalDecision{}, m.Err
	}
	var newest *MemoryApproval
	for _, r := range m.rows {
		if r.TenantID == req.TenantID && r.TokenID == req.TokenID && r.CallHash == req.CallHash &&
			(r.State == StatePending || r.State == StateApproved || r.State == StateDenied) {
			if newest == nil || r.CreatedAt.After(newest.CreatedAt) || (r.CreatedAt.Equal(newest.CreatedAt) && r.ID > newest.ID) {
				newest = r
			}
		}
	}
	if newest != nil {
		live := req.Now.Before(newest.ExpiresAt)
		switch {
		case newest.State == StateApproved && live:
			newest.State = StateUsed
			return ApprovalDecision{Outcome: ApprovalConsumed, ApprovalID: newest.ID, ExpiresAt: newest.ExpiresAt}, nil
		case newest.State == StatePending && live:
			return ApprovalDecision{Outcome: ApprovalHeld, ApprovalID: newest.ID, ExpiresAt: newest.ExpiresAt, NeedsNotice: newest.NotifiedAt.IsZero()}, nil
		case newest.State == StateDenied && live:
			return ApprovalDecision{Outcome: ApprovalDenied, ApprovalID: newest.ID, ExpiresAt: newest.ExpiresAt}, nil
		case newest.State == StatePending || newest.State == StateApproved:
			newest.State = StateExpired
			return ApprovalDecision{Outcome: ApprovalExpired, ApprovalID: newest.ID, ExpiresAt: newest.ExpiresAt}, nil
		}
		// A denied approval that ran out: ask again below.
	}
	waiting := 0
	for _, r := range m.rows {
		if r.TenantID == req.TenantID && r.TokenID == req.TokenID && r.State == StatePending && req.Now.Before(r.ExpiresAt) {
			waiting++
		}
	}
	if waiting >= MaxPendingApprovals {
		return ApprovalDecision{Outcome: ApprovalQueueFull}, nil
	}
	id, err := NewApprovalID()
	if err != nil {
		return ApprovalDecision{}, err
	}
	ttl := req.TTL
	if ttl <= 0 {
		ttl = ApprovalTTL
	}
	row := &MemoryApproval{
		ID: id, TenantID: req.TenantID, TokenID: req.TokenID, Tool: req.Tool, CallHash: req.CallHash,
		Field: req.Field, Above: req.Above, Summary: append([]SummaryItem(nil), req.Summary...),
		State: StatePending, CreatedAt: req.Now, ExpiresAt: req.Now.Add(ttl),
	}
	m.rows = append(m.rows, row)
	return ApprovalDecision{Outcome: ApprovalHeld, ApprovalID: id, ExpiresAt: row.ExpiresAt, NeedsNotice: true}, nil
}

// MarkNotified implements ApprovalStore.
func (m *MemoryApprovalStore) MarkNotified(_ context.Context, tenantID, approvalID string, at time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Err != nil {
		return m.Err
	}
	if r := m.find(tenantID, approvalID); r != nil {
		r.NotifiedAt = at
	}
	return nil
}

func (m *MemoryApprovalStore) find(tenantID, id string) *MemoryApproval {
	for _, r := range m.rows {
		if r.ID == id && r.TenantID == tenantID {
			return r
		}
	}
	return nil
}

// Approve is the owner saying yes. The tenant is the signed-in owner's; an id
// from another tenant is ErrApprovalNotFound.
func (m *MemoryApprovalStore) Approve(tenantID, approvalID, by string, now time.Time) error {
	return m.decide(tenantID, approvalID, by, now, StateApproved)
}

// Deny is the owner saying no.
func (m *MemoryApprovalStore) Deny(tenantID, approvalID, by string, now time.Time) error {
	return m.decide(tenantID, approvalID, by, now, StateDenied)
}

func (m *MemoryApprovalStore) decide(tenantID, id, by string, now time.Time, to string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	r := m.find(tenantID, id)
	if r == nil {
		return ErrApprovalNotFound
	}
	if r.State != StatePending || !now.Before(r.ExpiresAt) {
		return ErrApprovalNotPending
	}
	r.State, r.DecidedBy = to, by
	return nil
}

// Row returns a copy of one approval (tests).
func (m *MemoryApprovalStore) Row(approvalID string) (MemoryApproval, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, r := range m.rows {
		if r.ID == approvalID {
			return *r, true
		}
	}
	return MemoryApproval{}, false
}

// Rows returns copies of every approval (tests).
func (m *MemoryApprovalStore) Rows() []MemoryApproval {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]MemoryApproval, 0, len(m.rows))
	for _, r := range m.rows {
		out = append(out, *r)
	}
	return out
}
