package mcpserver

// Round 4: the in-memory hold store, on its own. A hold is named by its
// identity; commit and release act on that one hold and can be repeated.

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func eaHoldTestStore() (*InMemoryBudgetEnforcer, *eaClock) {
	clk := &eaClock{t: time.Now()}
	b := NewInMemoryBudgetEnforcer()
	b.now = clk.Now
	_ = b.Initialize(context.Background(), "b", 50)
	return b, clk
}

// One identity, one hold. A second Hold under an identity in use is refused
// and changes nothing, whether the first is open, committed, released or
// lapsed.
func TestR4_InMemoryHold_AnIdentityInUseIsRefused(t *testing.T) {
	ctx := context.Background()
	b, clk := eaHoldTestStore()
	if _, err := b.Hold(ctx, "b", "t", 3, "h1", time.Minute); err != nil {
		t.Fatal(err)
	}
	for _, step := range []struct {
		name string
		end  func()
	}{
		{"open", func() {}},
		{"committed", func() { _ = b.CommitHold(ctx, "b", "h1") }},
	} {
		step.end()
		before, _ := b.Remaining(ctx, "b")
		res, err := b.Hold(ctx, "b", "t", 3, "h1", time.Minute)
		if !errors.Is(err, ErrHoldIDInUse) || res != nil {
			t.Fatalf("%s: second hold = %+v, %v; want ErrHoldIDInUse", step.name, res, err)
		}
		if after, _ := b.Remaining(ctx, "b"); after != before {
			t.Fatalf("%s: the refused hold changed the balance %d -> %d", step.name, before, after)
		}
	}
	_, _ = b.Hold(ctx, "b", "t", 3, "h2", time.Minute)
	_ = b.ReleaseHold(ctx, "b", "h2")
	if _, err := b.Hold(ctx, "b", "t", 3, "h2", time.Minute); !errors.Is(err, ErrHoldIDInUse) {
		t.Fatalf("a released identity: %v", err)
	}
	_, _ = b.Hold(ctx, "b", "t", 3, "h3", time.Second)
	clk.advance(2 * time.Second)
	if _, err := b.Hold(ctx, "b", "t", 3, "h3", time.Minute); !errors.Is(err, ErrHoldIDInUse) {
		t.Fatalf("a lapsed identity: %v", err)
	}
	if _, err := b.Hold(ctx, "b", "t", 1, "", time.Minute); err == nil {
		t.Fatal("a hold with no identity must be refused")
	}
	// Another budget may use the same identity: a hold is named by both.
	_ = b.Initialize(ctx, "other", 10)
	if _, err := b.Hold(ctx, "other", "t", 1, "h1", time.Minute); err != nil {
		t.Fatalf("same identity on another budget: %v", err)
	}
	// Memory is bounded: an identity is forgotten long after any retry.
	clk.advance(settledHoldMemory + time.Minute)
	if _, err := b.Hold(ctx, "b", "t", 1, "h1", time.Minute); err != nil {
		t.Fatalf("a forgotten identity: %v", err)
	}
	if len(b.settled) > 1 {
		t.Fatalf("settled holds were not forgotten: %d", len(b.settled))
	}
}

// Commit and release are idempotent for their hold: a repeat reports what
// happened and changes nothing; neither reaches another hold.
func TestR4_InMemoryHold_RepeatsFindTheSameHold(t *testing.T) {
	ctx := context.Background()
	b, clk := eaHoldTestStore()
	_, _ = b.Hold(ctx, "b", "t", 3, "a", time.Minute)
	_, _ = b.Hold(ctx, "b", "t", 4, "b", time.Minute)
	if rem, _ := b.Remaining(ctx, "b"); rem != 43 {
		t.Fatalf("two holds open: %d", rem)
	}
	// Commit "a" three times: one debit; "b" is untouched.
	for i := 0; i < 3; i++ {
		if err := b.CommitHold(ctx, "b", "a"); err != nil {
			t.Fatalf("commit %d: %v", i, err)
		}
	}
	if n := len(b.SpendLog()); n != 1 {
		t.Fatalf("%d spend records, want 1", n)
	}
	if rem, _ := b.Remaining(ctx, "b"); rem != 43 {
		t.Fatalf("after the repeats: %d, want 43 (a charged 3, b still held 4)", rem)
	}
	// Release "b" twice: the first gives it back, the second changes nothing.
	for i := 0; i < 2; i++ {
		if err := b.ReleaseHold(ctx, "b", "b"); err != nil {
			t.Fatalf("release %d: %v", i, err)
		}
	}
	if rem, _ := b.Remaining(ctx, "b"); rem != 47 {
		t.Fatalf("after releasing b: %d, want 47", rem)
	}
	// What already happened is reported: release after commit says the debit
	// stands; commit after release says the hold is gone. Neither changes a
	// balance.
	if err := b.ReleaseHold(ctx, "b", "a"); !errors.Is(err, ErrHoldCommitted) {
		t.Fatalf("release after commit = %v", err)
	}
	if err := b.CommitHold(ctx, "b", "b"); !errors.Is(err, ErrHoldGone) {
		t.Fatalf("commit after release = %v", err)
	}
	if rem, _ := b.Remaining(ctx, "b"); rem != 47 {
		t.Fatalf("after the late calls: %d, want 47", rem)
	}
	// Lapse: commit says gone, release is fine, the amount came back once.
	_, _ = b.Hold(ctx, "b", "t", 5, "c", time.Second)
	clk.advance(2 * time.Second)
	if err := b.CommitHold(ctx, "b", "c"); !errors.Is(err, ErrHoldGone) {
		t.Fatalf("commit after the lapse = %v", err)
	}
	if err := b.ReleaseHold(ctx, "b", "c"); err != nil {
		t.Fatalf("release after the lapse = %v", err)
	}
	if rem, _ := b.Remaining(ctx, "b"); rem != 47 {
		t.Fatalf("after the lapse: %d, want 47", rem)
	}
	// An identity the store has never seen: commit is gone, release is fine.
	if err := b.CommitHold(ctx, "b", "never"); !errors.Is(err, ErrHoldGone) {
		t.Fatalf("commit of an unknown hold = %v", err)
	}
	if err := b.ReleaseHold(ctx, "b", "never"); err != nil {
		t.Fatalf("release of an unknown hold = %v", err)
	}
	// A debit that SettleCarve took after the hold lapsed is reported as
	// committed to a late repeat of the commit.
	_, _ = b.Hold(ctx, "b", "_delegation", 5, "k", time.Second)
	clk.advance(2 * time.Second)
	if _, err := b.SettleCarve(ctx, "b", "kid", "k", 5); err != nil {
		t.Fatal(err)
	}
	if err := b.CommitHold(ctx, "b", "k"); err != nil {
		t.Fatalf("commit after the settle charged it = %v", err)
	}
	if err := b.ReleaseHold(ctx, "b", "k"); !errors.Is(err, ErrHoldCommitted) {
		t.Fatalf("release after the settle charged it = %v", err)
	}
	if rem, _ := b.Remaining(ctx, "b"); rem != 42 {
		t.Fatalf("after the settle: %d, want 42", rem)
	}
}

// Concurrent repeats of one commit take the debit once.
func TestR4_InMemoryHold_ConcurrentRepeatsCommitOnce(t *testing.T) {
	ctx := context.Background()
	b, _ := eaHoldTestStore()
	_, _ = b.Hold(ctx, "b", "t", 3, "a", time.Minute)
	_, _ = b.Hold(ctx, "b", "t", 3, "other", time.Minute)
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := b.CommitHold(ctx, "b", "a"); err != nil {
				t.Errorf("commit: %v", err)
			}
		}()
	}
	wg.Wait()
	if n := len(b.SpendLog()); n != 1 {
		t.Fatalf("%d spend records, want 1", n)
	}
	if rem, _ := b.Remaining(ctx, "b"); rem != 44 {
		t.Fatalf("remaining %d, want 44", rem)
	}
}

// retryUndo's assumption (every_action.go): a repeat is a no-op. A lost answer
// is repeated and succeeds without a second effect; an answer that says what
// became of the hold is final and is not repeated.
func TestR4_RetryUndo_RepeatIsANoOpAndAFinalAnswerIsNotRepeated(t *testing.T) {
	ctx := context.Background()
	b, _ := eaHoldTestStore()
	_, _ = b.Hold(ctx, "b", "t", 3, "a", time.Minute)
	tries := 0
	err := retryUndo(ctx, time.Second, func(c context.Context) error {
		tries++
		if cerr := b.CommitHold(c, "b", "a"); cerr != nil {
			return cerr
		}
		if tries == 1 {
			return errors.New("answer lost")
		}
		return nil
	})
	if err != nil || tries != 2 {
		t.Fatalf("err %v after %d tries", err, tries)
	}
	if n := len(b.SpendLog()); n != 1 {
		t.Fatalf("%d spend records, want 1", n)
	}
	for _, final := range []error{ErrHoldGone, ErrHoldCommitted, ErrHoldIDInUse} {
		tries = 0
		got := retryUndo(ctx, time.Second, func(context.Context) error { tries++; return final })
		if !errors.Is(got, final) || tries != 1 {
			t.Fatalf("%v: got %v after %d tries", final, got, tries)
		}
	}
}
