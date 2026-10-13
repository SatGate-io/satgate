package mcpserver

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Round 5 of the kept-tools fix (1012e): the view lock can be abandoned. A
// list that stops waiting leaves no goroutine, timer or queued waiter behind.

func settledGoroutines() int {
	// Let goroutines that have already been told to stop finish.
	for i := 0; i < 5; i++ {
		runtime.GC()
		time.Sleep(10 * time.Millisecond)
	}
	return runtime.NumGoroutine()
}

// An abandoned wait leaves nothing behind: not a goroutine, not a queued
// waiter (which would take the lock from the next real waiter, or hold it
// when the owner lets go).
func TestViewLock_AnAbandonedWaitLeavesNothingBehind(t *testing.T) {
	var l viewLock
	l.Lock() // a call that is building
	before := settledGoroutines()

	const n = 200
	var wg sync.WaitGroup
	cancels := make([]context.CancelFunc, 0, n/2)
	for i := 0; i < n; i++ {
		wg.Add(1)
		if i%2 == 0 { // gives up at its deadline
			go func() {
				defer wg.Done()
				if l.LockWithin(context.Background(), time.Now().Add(30*time.Millisecond)) {
					t.Error("took a lock that was held")
				}
			}()
			continue
		}
		ctx, cancel := context.WithCancel(context.Background()) // gives up when ctx ends
		cancels = append(cancels, cancel)
		go func() {
			defer wg.Done()
			if l.LockWithin(ctx, time.Now().Add(time.Hour)) {
				t.Error("took a lock that was held")
			}
		}()
	}
	time.Sleep(60 * time.Millisecond)
	for _, c := range cancels {
		c()
	}
	waited := make(chan struct{})
	go func() { wg.Wait(); close(waited) }()
	select {
	case <-waited:
	case <-time.After(3 * time.Second): // a wait that ignores its deadline or ctx: fail, do not hang
		t.Fatal("200 waits did not end 3 s after their deadlines and cancels: a wait ignores its deadline or its ctx")
	}

	// The owner still holds the lock: nothing of the 200 may be left.
	if after := settledGoroutines(); after > before+2 {
		t.Fatalf("goroutines with the lock still held: %d before the waits, %d after them (%d abandoned waits)", before, after, n)
	}
	// No queued waiter: when the owner lets go the lock is free at once.
	l.Unlock()
	if !l.TryLock() {
		t.Fatal("the owner let go and the lock is not free: an abandoned wait is still queued")
	}
	l.Unlock()
}

// A wait that gets the lock at the moment it gives up does not keep it: it
// answers true and the caller holds it (and lets go), or it answers false and
// holds nothing. Never false while holding.
func TestViewLock_AWaitThatGivesUpAtTheMomentOfTheUnlockDoesNotKeepTheLock(t *testing.T) {
	var l viewLock
	var got, gaveUp int
	for i := 0; i < 1000; i++ {
		l.Lock()
		done := make(chan bool)
		dl := time.Now().Add(300 * time.Microsecond)
		go func() { done <- l.LockWithin(context.Background(), dl) }()
		// The owner lets go about when the waiter's time is up.
		time.Sleep(time.Duration(200+(i%7)*50) * time.Microsecond)
		l.Unlock()
		if <-done {
			got++
			l.Unlock() // what the caller of a true answer does
		} else {
			gaveUp++
		}
		if !l.TryLock() {
			t.Fatalf("wait %d (got=%d gaveUp=%d): the lock is held after the wait and the owner both let go", i, got, gaveUp)
		}
		l.Unlock()
	}
	t.Logf("1000 timed waits racing an unlock: %d got the lock, %d gave up", got, gaveUp)

	// The same with ctx ending at the moment of the unlock.
	got, gaveUp = 0, 0
	for i := 0; i < 1000; i++ {
		l.Lock()
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan bool)
		go func() { done <- l.LockWithin(ctx, time.Now().Add(time.Hour)) }()
		time.Sleep(time.Duration(50+(i%5)*20) * time.Microsecond)
		var wg sync.WaitGroup
		wg.Add(2)
		go func() { defer wg.Done(); cancel() }()
		go func() { defer wg.Done(); l.Unlock() }()
		wg.Wait()
		if <-done {
			got++
			l.Unlock()
		} else {
			gaveUp++
		}
		if !l.TryLock() {
			t.Fatalf("ctx wait %d (got=%d gaveUp=%d): the lock is held after the wait and the owner both let go", i, got, gaveUp)
		}
		l.Unlock()
	}
	t.Logf("1000 ctx waits racing an unlock: %d got the lock, %d gave up", got, gaveUp)
}

// A call's wait (Lock) has no time limit: it outlasts ListBudget and any
// list's deadline, and gets the lock when it is let go. Lists that queued
// behind it in the meantime have gone.
func TestViewLock_ACallsWaitHasNoTimeLimit(t *testing.T) {
	var l viewLock
	l.Lock()
	hold := ListBudget + 400*time.Millisecond

	acquired := make(chan time.Duration, 1)
	start := time.Now()
	go func() {
		l.Lock()
		acquired <- time.Since(start)
		l.Unlock()
	}()
	// A list behind the same owner gives up at its deadline, and the call
	// is still waiting.
	t0 := time.Now()
	if l.LockWithin(context.Background(), t0.Add(200*time.Millisecond)) {
		t.Fatal("a list took a lock that was held")
	}
	if el := time.Since(t0); el > 400*time.Millisecond {
		t.Fatalf("the list waited %s, want its 200ms", el)
	}
	select {
	case el := <-acquired:
		t.Fatalf("the call got a lock that was held, after %s", el)
	case <-time.After(hold - time.Since(start)):
	}
	l.Unlock()
	select {
	case el := <-acquired:
		if el < hold {
			t.Fatalf("the call got the lock after %s, before it was let go at %s", el, hold)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the call did not get the lock after it was let go")
	}
}

// Lists are not starved and a list never answers late: with a list arriving
// every few ms while owners come and go, each LockWithin returns by its own
// deadline (plus scheduling), whether it got the lock or not.
func TestViewLock_ListsArrivingAllTheTimeAnswerByTheirDeadline(t *testing.T) {
	var l viewLock
	stop := make(chan struct{})
	var owners sync.WaitGroup
	owners.Add(1)
	go func() { // builds: take the lock, hold it 40ms, repeat
		defer owners.Done()
		for {
			select {
			case <-stop:
				return
			default:
			}
			l.Lock()
			time.Sleep(40 * time.Millisecond)
			l.Unlock()
		}
	}()

	const wait = 25 * time.Millisecond
	var lists sync.WaitGroup
	var got, gaveUp int64
	var worst atomic.Int64
	for i := 0; i < 300; i++ {
		lists.Add(1)
		go func() {
			defer lists.Done()
			t0 := time.Now()
			ok := l.LockWithin(context.Background(), t0.Add(wait))
			el := time.Since(t0)
			if ok {
				atomic.AddInt64(&got, 1)
				l.Unlock()
			} else {
				atomic.AddInt64(&gaveUp, 1)
			}
			for {
				w := worst.Load()
				if int64(el) <= w || worst.CompareAndSwap(w, int64(el)) {
					break
				}
			}
		}()
		time.Sleep(3 * time.Millisecond)
	}
	lists.Wait()
	close(stop)
	owners.Wait()
	if w := time.Duration(worst.Load()); w > wait+150*time.Millisecond {
		t.Fatalf("a list took %s to answer, its wait was %s", w, wait)
	}
	t.Logf("300 lists, one every 3ms: %d got the lock, %d gave up, slowest %s (wait %s)", got, gaveUp, time.Duration(worst.Load()), wait)
	if !l.TryLock() {
		t.Fatal("the lock is held after every owner and list is done")
	}
	l.Unlock()
}
