package toolcontrols

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"
)

// ReserveStatus is the result of one reservation.
type ReserveStatus int

const (
	// Reserved: the amount is now counted against the window.
	Reserved ReserveStatus = iota
	// OverLimit: counting the amount would pass the limit; nothing changed.
	OverLimit
	// Duplicate: this call id already holds a reservation in this window;
	// nothing changed.
	Duplicate
)

// Reservation is one amount to count against one window of one limit.
type Reservation struct {
	// Key names the counter: tenant, limit and window together.
	Key string
	// CallID names the call, so a retried call counts once.
	CallID string
	// Units is the amount in 10^-8 units, never negative.
	Units int64
	// Max is the limit in the same units.
	Max int64
	// ExpiresAt is when the counter can be dropped (the end of the window,
	// plus a grace period).
	ExpiresAt time.Time
}

// ReserveResult is what a store returns for Reserve.
type ReserveResult struct {
	Status ReserveStatus
	// Total is the window total after this call (Reserved) or the total
	// that was already there (OverLimit, Duplicate), in 10^-8 units.
	Total int64
}

// SpendStore is the durable counter behind spending limits. Both methods must
// be atomic and safe for any number of concurrent callers across processes.
// A store that cannot answer returns an error; the gateway then refuses the
// call (fail closed).
type SpendStore interface {
	// Reserve adds r.Units to the counter at r.Key if call r.CallID has no
	// reservation there and the new total would not pass r.Max.
	Reserve(ctx context.Context, r Reservation) (ReserveResult, error)
	// Release removes the reservation r.CallID holds at r.Key and lowers the
	// counter by what it held. It does nothing when there is none, so it is
	// safe to call twice.
	Release(ctx context.Context, r Reservation) error
}

// ErrStoreUnavailable is what a store wraps when it cannot reach its backend.
var ErrStoreUnavailable = errors.New("spend limit store unavailable")

// MemoryStore is a SpendStore in process memory. It is correct for one
// process, loses its counters on restart, and is for tests and single-process
// use. It takes a clock so a test can move past a window.
type MemoryStore struct {
	mu       sync.Mutex
	now      func() time.Time
	counters map[string]*memCounter
	// Err, when set, is returned by every call (a store that is down).
	Err error
}

type memCounter struct {
	total   int64
	calls   map[string]int64
	expires time.Time
}

// NewMemoryStore returns an empty store. now may be nil for the real clock.
func NewMemoryStore(now func() time.Time) *MemoryStore {
	if now == nil {
		now = time.Now
	}
	return &MemoryStore{now: now, counters: map[string]*memCounter{}}
}

func (m *MemoryStore) sweepLocked() {
	t := m.now()
	for k, c := range m.counters {
		if !t.Before(c.expires) {
			delete(m.counters, k)
		}
	}
}

// Reserve implements SpendStore.
func (m *MemoryStore) Reserve(_ context.Context, r Reservation) (ReserveResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Err != nil {
		return ReserveResult{}, m.Err
	}
	m.sweepLocked()
	c := m.counters[r.Key]
	if c == nil {
		c = &memCounter{calls: map[string]int64{}, expires: r.ExpiresAt}
		m.counters[r.Key] = c
	}
	if _, dup := c.calls[r.CallID]; dup {
		return ReserveResult{Status: Duplicate, Total: c.total}, nil
	}
	if r.Units > r.Max-c.total {
		return ReserveResult{Status: OverLimit, Total: c.total}, nil
	}
	c.calls[r.CallID] = r.Units
	c.total += r.Units
	return ReserveResult{Status: Reserved, Total: c.total}, nil
}

// Release implements SpendStore.
func (m *MemoryStore) Release(_ context.Context, r Reservation) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Err != nil {
		return m.Err
	}
	c := m.counters[r.Key]
	if c == nil {
		return nil
	}
	if units, ok := c.calls[r.CallID]; ok {
		delete(c.calls, r.CallID)
		c.total -= units
	}
	return nil
}

// Total reports the current counter at key (tests).
func (m *MemoryStore) Total(key string) int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	if c := m.counters[key]; c != nil {
		return c.total
	}
	return 0
}

// Keys lists the live counters (tests).
func (m *MemoryStore) Keys() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []string
	for k := range m.counters {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
