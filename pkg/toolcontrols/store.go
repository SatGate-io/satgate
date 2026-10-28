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
	// EventsFull: an event stake counter already holds MaxEventsPerStake
	// events and this is a new one; nothing changed. The totals must be
	// reset before another event can be counted.
	EventsFull
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

// SpendStore is the durable counter behind spending limits. Reserve and
// Release must be atomic and safe for any number of concurrent callers across processes.
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
	// Total reads the counter at key, in 10^-8 units, without changing it. A
	// key with no counter (never used, or past its window) reads 0.
	Total(ctx context.Context, key string) (int64, error)
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
	stakes   map[string]*memStake
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

// Total implements SpendStore.
func (m *MemoryStore) Total(_ context.Context, key string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Err != nil {
		return 0, m.Err
	}
	m.sweepLocked()
	if c := m.counters[key]; c != nil {
		return c.total, nil
	}
	return 0, nil
}

// TotalOf is Total for tests: the counter at key, 0 when there is none.
func (m *MemoryStore) TotalOf(key string) int64 {
	n, _ := m.Total(context.Background(), key)
	return n
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

// ---- Per-event totals ---------------------------------------------------------

// MaxEventsPerStake bounds how many events one stake counter holds. A new
// event past it is refused (EventsFull) until the totals are reset.
const MaxEventsPerStake = 10000

// EventReservation is one amount to count against one event of one stake
// limit.
type EventReservation struct {
	// Key names the stake counter: tenant and limit together. Every event
	// of the limit is under one key, so a reset is one step.
	Key string
	// Event is the event id the market belongs to, as the host's market
	// lookup answered it.
	Event string
	// CallID names the call, so a retried call counts once.
	CallID string
	// Units is the amount in 10^-8 units, never negative.
	Units int64
	// Max is the limit in the same units.
	Max int64
}

// EventStakeStore is the durable counter behind per-event stake limits. A
// store a gate is given may implement it; a gate whose store does not refuses
// every call a stake limit covers. Every method must be atomic and safe for
// any number of concurrent callers. A store that cannot answer returns an
// error; the call is then refused (fail closed).
type EventStakeStore interface {
	// ReserveEvent adds r.Units to the event's total at r.Key when r.CallID
	// holds nothing there and the new total would not pass r.Max.
	ReserveEvent(ctx context.Context, r EventReservation) (ReserveResult, error)
	// ReleaseEvent removes what r.CallID holds at r.Key and lowers its
	// event's total by that. It does nothing when there is none.
	ReleaseEvent(ctx context.Context, r EventReservation) error
	// EventTotal reads one event's total at key, 0 when there is none.
	EventTotal(ctx context.Context, key, event string) (int64, error)
	// ResetEvents drops every event total at key in one step. A reservation
	// made before it is gone with it; one made after it counts in full.
	ResetEvents(ctx context.Context, key string) error
}

type memStake struct {
	events map[string]int64
	calls  map[string]memStakeCall
}

type memStakeCall struct {
	event string
	units int64
}

func (m *MemoryStore) stakeLocked(key string, create bool) *memStake {
	if m.stakes == nil {
		m.stakes = map[string]*memStake{}
	}
	st := m.stakes[key]
	if st == nil && create {
		st = &memStake{events: map[string]int64{}, calls: map[string]memStakeCall{}}
		m.stakes[key] = st
	}
	return st
}

// ReserveEvent implements EventStakeStore.
func (m *MemoryStore) ReserveEvent(_ context.Context, r EventReservation) (ReserveResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Err != nil {
		return ReserveResult{}, m.Err
	}
	st := m.stakeLocked(r.Key, true)
	total := st.events[r.Event]
	if _, dup := st.calls[r.CallID]; dup {
		return ReserveResult{Status: Duplicate, Total: total}, nil
	}
	if _, known := st.events[r.Event]; !known && len(st.events) >= MaxEventsPerStake {
		return ReserveResult{Status: EventsFull, Total: 0}, nil
	}
	if r.Units < 0 || r.Units > r.Max-total {
		return ReserveResult{Status: OverLimit, Total: total}, nil
	}
	st.calls[r.CallID] = memStakeCall{event: r.Event, units: r.Units}
	st.events[r.Event] = total + r.Units
	return ReserveResult{Status: Reserved, Total: total + r.Units}, nil
}

// ReleaseEvent implements EventStakeStore.
func (m *MemoryStore) ReleaseEvent(_ context.Context, r EventReservation) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Err != nil {
		return m.Err
	}
	st := m.stakeLocked(r.Key, false)
	if st == nil {
		return nil
	}
	c, ok := st.calls[r.CallID]
	if !ok || c.event != r.Event {
		return nil
	}
	delete(st.calls, r.CallID)
	st.events[c.event] -= c.units
	return nil
}

// EventTotal implements EventStakeStore.
func (m *MemoryStore) EventTotal(_ context.Context, key, event string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Err != nil {
		return 0, m.Err
	}
	if st := m.stakeLocked(key, false); st != nil {
		return st.events[event], nil
	}
	return 0, nil
}

// ResetEvents implements EventStakeStore.
func (m *MemoryStore) ResetEvents(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Err != nil {
		return m.Err
	}
	if m.stakes != nil {
		delete(m.stakes, key)
	}
	return nil
}
