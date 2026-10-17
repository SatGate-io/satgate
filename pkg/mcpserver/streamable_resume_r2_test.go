package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Round 2 (Astra B1/B2, Grok B2 on satgate-enterprise#522): a session ended
// with DELETE stays ended on the process that ended it, whatever the load.

// Fill the tombstone map to its cap with unexpired entries and end one more
// session: no ended id is resumed. When the earliest tombstone expires, only
// its slot is freed; the id that found the map full stays refused until its
// own TTL is over.
func TestStreamableResumeEndedSessionsStayEndedWhenTombstonesFull(t *testing.T) {
	const max = 4
	auth := newResumeTokens()
	in := newResumeInstance(t, auth, WithMaxStreamableSessions(max))
	endOne := func() string {
		t.Helper()
		sid := resumeInit(t, in, "token-alice")
		if status, _, body := resumeDo(t, http.MethodDelete, in.url(), sid, "token-alice", "2025-06-18", nil); status != http.StatusOK {
			t.Fatalf("DELETE status=%d body=%s", status, body)
		}
		return sid
	}
	resume := func(sid string) (int, string) {
		t.Helper()
		status, _, body := resumePost(t, in.url(), sid, "token-alice", toolCallBody(9, "echo"))
		return status, body
	}
	refused := func(what, sid string) {
		t.Helper()
		if status, body := resume(sid); status != http.StatusNotFound || body != "session not found\n" {
			t.Fatalf("%s: status=%d body=%q, want the 404", what, status, body)
		}
	}

	// ended[0..3] fill the map one minute apart; ended[4] finds it full.
	var ended []string
	for i := 0; i < max; i++ {
		ended = append(ended, endOne())
		advanceStreamTombstones(in.sse, time.Minute)
	}
	overflow := endOne()
	ended = append(ended, overflow)
	for i, sid := range ended {
		refused(fmt.Sprintf("ended[%d] with the tombstone map full", i), sid)
	}
	tombs, fullUntil := streamTombstones(in.sse)
	if len(tombs) != max {
		t.Fatalf("tombstones=%d, want %d (none dropped, none added over the cap)", len(tombs), max)
	}
	for i := 0; i < max; i++ {
		if _, ok := tombs[ended[i]]; !ok {
			t.Fatalf("unexpired tombstone of ended[%d] was dropped", i)
		}
	}
	// Every resume on this process is refused, even of an id it never ended.
	foreign := resumeInit(t, newResumeInstance(t, auth), "token-alice")
	refused("an id this process never ended", foreign)
	// initialize and live sessions are not affected.
	live := resumeInit(t, in, "token-alice")
	status, _, body := resumePost(t, in.url(), live, "token-alice", toolCallBody(3, "echo"))
	requireToolResult(t, "live session with the tombstone map full", status, body)
	if n := streamSessionCount(in.sse); n != 1 {
		t.Fatalf("sessions=%d, want only the live one", n)
	}

	// The earliest tombstone (ended[0]) expires; ended[1..3] and the window
	// of the unrecorded ended[4] have not.
	advanceStreamTombstones(in.sse, defaultStreamableSessionTTL-4*time.Minute+30*time.Second)
	for i, sid := range ended {
		refused(fmt.Sprintf("ended[%d] after the earliest tombstone expired", i), sid)
	}
	// Its slot is free and only that slot: the next DELETE is recorded in
	// it, the other tombstones and the refusal window are unchanged.
	again := endOne()
	tombs2, fullUntil2 := streamTombstones(in.sse)
	if len(tombs2) != max {
		t.Fatalf("tombstones=%d, want %d", len(tombs2), max)
	}
	if _, ok := tombs2[again]; !ok {
		t.Fatal("the DELETE after the earliest tombstone expired was not recorded in the freed slot")
	}
	if _, ok := tombs2[ended[0]]; ok {
		t.Fatal("the expired tombstone is still held")
	}
	for i := 1; i < max; i++ {
		if _, ok := tombs2[ended[i]]; !ok {
			t.Fatalf("tombstone of ended[%d] was dropped", i)
		}
	}
	if !fullUntil2.Equal(fullUntil.Add(-(defaultStreamableSessionTTL - 4*time.Minute + 30*time.Second))) {
		t.Fatalf("refusal window moved: %v -> %v", fullUntil, fullUntil2)
	}
	refused("the id ended in the freed slot", again)

	// Past the TTL of every id ended before the map filled up and of the
	// one that found it full: those resume; the later one stays ended.
	advanceStreamTombstones(in.sse, 4*time.Minute)
	refused("the id ended in the freed slot, within its TTL", again)
	for _, i := range []int{4, 0, 1} {
		status, body := resume(ended[i])
		requireToolResult(t, fmt.Sprintf("ended[%d] after its TTL", i), status, body)
	}
}

// Astra's adverse test (TestAstraDeletedTombstoneCapacity), kept: with the
// session cap at 1, ending a second session must not revive the first.
func TestAstraDeletedTombstoneCapacity(t *testing.T) {
	in := newResumeInstance(t, newResumeTokens(), WithMaxStreamableSessions(1))
	first := resumeInit(t, in, "token-alice")
	if code, _, _ := resumeDo(t, http.MethodDelete, in.url(), first, "token-alice", "", nil); code != 200 {
		t.Fatal(code)
	}
	second := resumeInit(t, in, "token-alice")
	if code, _, _ := resumeDo(t, http.MethodDelete, in.url(), second, "token-alice", "", nil); code != 200 {
		t.Fatal(code)
	}
	for _, sid := range []string{first, second} {
		if code, _, _ := resumePost(t, in.url(), sid, "token-alice", toolCallBody(10, "echo")); code != 404 {
			t.Fatalf("deleted session revived under capacity pressure: got %d want 404", code)
		}
	}
}

type astraBlockedAuth struct {
	base             Authenticator
	once             atomic.Bool
	entered, release chan struct{}
}

func (a *astraBlockedAuth) Verify(ctx context.Context, token string) (*TokenInfo, error) {
	if a.once.Swap(false) {
		close(a.entered)
		<-a.release
	}
	return a.base.Verify(ctx, token)
}

// Astra's adverse test (TestAstraDeleteRacesResume), kept: a resume paused in
// Verify while another resume and a DELETE complete must not bring the id
// back when it continues.
func TestAstraDeleteRacesResume(t *testing.T) {
	a := &astraBlockedAuth{base: newResumeTokens(), entered: make(chan struct{}), release: make(chan struct{})}
	sid := resumeInit(t, newResumeInstance(t, a), "token-alice")
	after := newResumeInstance(t, a)
	a.once.Store(true)
	done := make(chan int, 1)
	go func() {
		code, _, _ := resumePost(t, after.url(), sid, "token-alice", toolCallBody(2, "echo"))
		done <- code
	}()
	<-a.entered
	code, _, _ := resumePost(t, after.url(), sid, "token-alice", toolCallBody(3, "echo"))
	if code != 200 {
		close(a.release)
		t.Fatal(code)
	}
	code, _, _ = resumeDo(t, http.MethodDelete, after.url(), sid, "token-alice", "", nil)
	if code != 200 {
		close(a.release)
		t.Fatal(code)
	}
	close(a.release)
	if code := <-done; code != http.StatusNotFound {
		t.Fatalf("resume paused across the DELETE: status=%d want 404", code)
	}
	code, _, _ = resumePost(t, after.url(), sid, "token-alice", toolCallBody(4, "echo"))
	if code != 404 {
		t.Fatalf("completed DELETE raced with resume: next POST=%d want 404", code)
	}
	if n := streamSessionCount(after.sse); n != 0 {
		t.Fatalf("sessions=%d after the DELETE, want 0", n)
	}
}

// jitterAuth widens the window between the resume checks and the insert.
type jitterAuth struct{ base Authenticator }

func (a jitterAuth) Verify(ctx context.Context, token string) (*TokenInfo, error) {
	runtime.Gosched()
	time.Sleep(time.Duration(rand.Intn(300)) * time.Microsecond)
	return a.base.Verify(ctx, token)
}

func rawStreamPost(client *http.Client, method, url, sid, token string, msg any) (int, error) {
	var body io.Reader = http.NoBody
	if msg != nil {
		raw, err := json.Marshal(msg)
		if err != nil {
			return 0, err
		}
		body = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return 0, err
	}
	if msg != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("Mcp-Session-Id", sid)
	req.Header.Set("MCP-Protocol-Version", "2025-06-18")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	return resp.StatusCode, nil
}

// Concurrent resumes and a DELETE of the same id on a restarted process,
// many times: once the DELETE has returned, the id is never served again on
// that process, and no session for it is left in the map.
func TestStreamableResumeConcurrentDeleteNeverRevives(t *testing.T) {
	sid := resumeInit(t, newResumeInstance(t, newResumeTokens()), "token-alice")
	iterations := 150
	if testing.Short() {
		iterations = 30
	}
	client := &http.Client{Timeout: 10 * time.Second}
	deletedRuns := 0
	for it := 0; it < iterations; it++ {
		after := newResumeInstance(t, jitterAuth{base: newResumeTokens()})
		var (
			deleted   atomic.Bool
			wg        sync.WaitGroup
			mu        sync.Mutex
			failures  []string
			lateServe atomic.Int64
		)
		fail := func(format string, args ...any) {
			mu.Lock()
			defer mu.Unlock()
			failures = append(failures, fmt.Sprintf(format, args...))
		}
		for g := 0; g < 6; g++ {
			wg.Add(1)
			go func(g int) {
				defer wg.Done()
				for k := 0; k < 4; k++ {
					startedAfterDelete := deleted.Load()
					status, err := rawStreamPost(client, http.MethodPost, after.url(), sid, "token-alice",
						map[string]any{"jsonrpc": "2.0", "id": 100 + g*10 + k, "method": "tools/list"})
					if err != nil {
						fail("resume: %v", err)
						return
					}
					if startedAfterDelete && status == http.StatusOK {
						lateServe.Add(1)
					}
				}
			}(g)
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			for try := 0; try < 400; try++ {
				status, err := rawStreamPost(client, http.MethodDelete, after.url(), sid, "token-alice", nil)
				if err != nil {
					fail("DELETE: %v", err)
					return
				}
				if status == http.StatusOK {
					deleted.Store(true)
					return
				}
				runtime.Gosched()
			}
		}()
		wg.Wait()
		if len(failures) > 0 {
			t.Fatalf("iteration %d: %v", it, failures)
		}
		if !deleted.Load() {
			continue
		}
		deletedRuns++
		if n := lateServe.Load(); n != 0 {
			t.Fatalf("iteration %d: %d requests sent after the DELETE returned were served", it, n)
		}
		if n := streamSessionCount(after.sse); n != 0 {
			t.Fatalf("iteration %d: the id was resumed after its DELETE (sessions=%d)", it, n)
		}
		if status, err := rawStreamPost(client, http.MethodPost, after.url(), sid, "token-alice", toolCallBody(7, "echo")); err != nil || status != http.StatusNotFound {
			t.Fatalf("iteration %d: POST after the DELETE: status=%d err=%v", it, status, err)
		}
		after.srv.Close()
	}
	if deletedRuns < iterations/2 {
		t.Fatalf("only %d of %d iterations got the DELETE in; the race was not exercised", deletedRuns, iterations)
	}
}
