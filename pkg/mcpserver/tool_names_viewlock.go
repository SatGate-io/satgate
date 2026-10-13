package mcpserver

import (
	"context"
	"sync"
	"time"
)

// viewLock is the view lock: a one-slot channel. Lock sends into the slot,
// Unlock receives from it, TryLock is a send with a default, and a list waits
// with LockWithin, a select on the send, a timer and ctx.Done().
//
// Why a channel and not a sync.Mutex: a sync.Mutex cannot be waited for with a
// time limit. The round 4 version waited in a helper goroutine, and a list
// that gave up left the helper queued on the mutex (and a second goroutine to
// release the lock when the helper got it) until the holder, which can be a
// call building for up to viewBuildLimit, let go. A select that is left by
// another case is taken out of the channel's wait queue by the runtime before
// select returns, so a list that gives up leaves no goroutine, no queued
// waiter and (the timer is stopped) no timer behind.
//
// Order: the runtime queues blocked senders first in, first out, and Unlock
// (a receive from a full one-slot channel with senders waiting) moves the
// head sender's value into the slot itself. The lock is therefore handed to
// the longest-waiting waiter and a TryLock that arrives in between does not
// get in front (it sees the slot full). Lists and calls are served in
// arrival order. That order is not needed for the list's time promise (every
// list leaves at its deadline whatever the order), and it does no harm to
// calls (a list ahead of a call holds the lock for at most its own budget).
//
// The zero value is an unlocked lock, as with sync.Mutex. It must not be
// copied after first use.
type viewLock struct {
	once sync.Once
	c    chan struct{}
}

func (l *viewLock) slot() chan struct{} {
	l.once.Do(func() { l.c = make(chan struct{}, 1) })
	return l.c
}

// Lock waits for the lock as long as it takes (a call's wait).
func (l *viewLock) Lock() { l.slot() <- struct{}{} }

// TryLock takes the lock if it is free.
func (l *viewLock) TryLock() bool {
	select {
	case l.slot() <- struct{}{}:
		return true
	default:
		return false
	}
}

// Unlock releases the lock. Like sync.Mutex, it is a fatal mistake to unlock
// a lock that is not locked.
func (l *viewLock) Unlock() {
	select {
	case <-l.slot():
	default:
		panic("mcpserver: unlock of unlocked view lock")
	}
}

// LockWithin takes the lock, giving up at dl or when ctx ends, whichever is
// first. It reports whether the caller holds the lock. If it reports false it
// holds nothing and has left nothing behind; if it reports true the caller
// holds the lock and must Unlock it. The decision is made by one select, so a
// wait cannot both give up and keep the lock.
func (l *viewLock) LockWithin(ctx context.Context, dl time.Time) bool {
	if l.TryLock() {
		return true
	}
	if ctx.Err() != nil {
		return false
	}
	d := time.Until(dl)
	if d <= 0 {
		return false
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case l.slot() <- struct{}{}:
		return true
	case <-t.C:
		return false
	case <-ctx.Done():
		return false
	}
}
