package mcpserver

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"
)

type limiterCall struct {
	tokenID  string
	method   string
	toolName string
}

// scriptedLimiter refuses once `allow` calls have been allowed and records
// every call it sees.
type scriptedLimiter struct {
	mu         sync.Mutex
	allow      int
	retryAfter time.Duration
	calls      []limiterCall
}

func (l *scriptedLimiter) Allow(_ context.Context, info *TokenInfo, method, toolName string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	id := ""
	if info != nil {
		id = info.TokenID
	}
	l.calls = append(l.calls, limiterCall{tokenID: id, method: method, toolName: toolName})
	if l.allow > 0 {
		l.allow--
		return true, 0
	}
	return false, l.retryAfter
}

func (l *scriptedLimiter) seen() []limiterCall {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]limiterCall(nil), l.calls...)
}

type revokeAll struct{}

func (revokeAll) IsRevoked(context.Context, string) bool { return true }

func newLimiterTestProxy(t *testing.T, authMode string) (*Proxy, *ChannelPublisher) {
	t.Helper()
	cfg := &Config{
		Server:      ServerConfig{Transport: "sse", Name: "limiter-test", Version: "test"},
		Auth:        AuthConfig{Mode: authMode, RootKey: "limiter-test-root"},
		Budget:      BudgetConfig{Limit: 10, FailMode: "closed"},
		Tools:       ToolsConfig{DefaultCost: 1},
		Enforcement: EnforcementConfig{Mode: "control"},
		Logging:     LoggingConfig{Level: "error"},
	}
	proxy, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	proxy.SetUpstreamRouter(&countingForwardRouter{})
	events := NewChannelPublisher(64)
	proxy.SetEventPublisher(events)
	return proxy, events
}

func toolCall(id int, name string) *Request {
	return &Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(mustJSONString(id)),
		Method:  MethodToolsCall,
		Params:  json.RawMessage(`{"name":"` + name + `","arguments":{}}`),
	}
}

func mustJSONString(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func drainEvents(events *ChannelPublisher) []Event {
	var out []Event
	for {
		select {
		case e := <-events.Events():
			out = append(out, e)
		default:
			return out
		}
	}
}

func remainingFor(t *testing.T, proxy *Proxy) int64 {
	t.Helper()
	rem, err := proxy.budget.Remaining(context.Background(), proxy.tokenID)
	if err != nil {
		t.Fatal(err)
	}
	return rem
}

// A refused request is answered with -32004 before cost, spend, events or
// the upstream: it costs nothing and records nothing.
func TestRequestLimiterRefusalCostsAndRecordsNothing(t *testing.T) {
	proxy, events := newLimiterTestProxy(t, "none")
	limiter := &scriptedLimiter{allow: 1, retryAfter: 2300 * time.Millisecond}
	proxy.SetRequestLimiter(limiter)
	ctx := context.Background()

	resp, err := proxy.handleRequest(ctx, toolCall(1, "echo"))
	if err != nil || resp == nil || resp.Error != nil {
		t.Fatalf("first call should pass: resp=%+v err=%v", resp, err)
	}
	if got := streamableForwardCount(proxy); got != 1 {
		t.Fatalf("first call forwards=%d, want 1", got)
	}
	if got := remainingFor(t, proxy); got != 9 {
		t.Fatalf("first call remaining=%d, want 9", got)
	}
	_ = drainEvents(events)

	resp, err = proxy.handleRequest(ctx, toolCall(2, "echo"))
	if err != nil || resp == nil || resp.Error == nil {
		t.Fatalf("second call should be refused: resp=%+v err=%v", resp, err)
	}
	if resp.Error.Code != CodeRateLimited {
		t.Fatalf("code=%d, want %d", resp.Error.Code, CodeRateLimited)
	}
	if string(resp.ID) != "2" {
		t.Fatalf("refusal id=%s, want 2", resp.ID)
	}
	var data struct {
		Error             string `json:"error"`
		RetryAfterSeconds int64  `json:"retry_after_seconds"`
	}
	if err := json.Unmarshal(resp.Error.Data, &data); err != nil {
		t.Fatalf("refusal data: %v (%s)", err, resp.Error.Data)
	}
	if data.Error != "rate_limited" || data.RetryAfterSeconds != 3 {
		t.Fatalf("refusal data=%+v, want rate_limited / 3s (2.3s rounded up)", data)
	}
	if got := streamableForwardCount(proxy); got != 1 {
		t.Fatalf("refused call reached upstream: forwards=%d", got)
	}
	if got := remainingFor(t, proxy); got != 9 {
		t.Fatalf("refused call spent: remaining=%d, want 9", got)
	}
	if evs := drainEvents(events); len(evs) != 0 {
		t.Fatalf("refused call published %d events: %+v", len(evs), evs)
	}
}

// The limiter sees the verified token, the method and, for tools/call, the
// tool name. Other authenticated methods are limited too.
func TestRequestLimiterSeesTokenMethodAndTool(t *testing.T) {
	proxy, _ := newLimiterTestProxy(t, "none")
	limiter := &scriptedLimiter{allow: 1, retryAfter: 0}
	proxy.SetRequestLimiter(limiter)
	ctx := context.Background()

	if resp, _ := proxy.handleRequest(ctx, toolCall(1, "echo")); resp == nil || resp.Error != nil {
		t.Fatalf("tools/call should pass: %+v", resp)
	}
	budgetReq := &Request{JSONRPC: "2.0", ID: json.RawMessage(`2`), Method: MethodSatGateBudget, Params: json.RawMessage(`{}`)}
	resp, _ := proxy.handleRequest(ctx, budgetReq)
	if resp == nil || resp.Error == nil || resp.Error.Code != CodeRateLimited {
		t.Fatalf("satgate/budget should be limited: %+v", resp)
	}
	var data struct {
		RetryAfterSeconds int64 `json:"retry_after_seconds"`
	}
	_ = json.Unmarshal(resp.Error.Data, &data)
	if data.RetryAfterSeconds != 1 {
		t.Fatalf("retry_after_seconds=%d, want the 1s floor for a zero wait", data.RetryAfterSeconds)
	}
	seen := limiter.seen()
	want := []limiterCall{
		{tokenID: proxy.tokenID, method: MethodToolsCall, toolName: "echo"},
		{tokenID: proxy.tokenID, method: MethodSatGateBudget, toolName: ""},
	}
	if len(seen) != len(want) {
		t.Fatalf("limiter calls=%+v, want %+v", seen, want)
	}
	for i := range want {
		if seen[i] != want[i] {
			t.Fatalf("limiter call %d=%+v, want %+v", i, seen[i], want[i])
		}
	}
}

// Unauthenticated and revoked requests are refused by those checks first;
// the limiter is never asked, so it cannot be used to probe token state.
func TestRequestLimiterRunsAfterAuthAndRevocation(t *testing.T) {
	t.Run("unauthenticated", func(t *testing.T) {
		proxy, _ := newLimiterTestProxy(t, "header")
		limiter := &scriptedLimiter{}
		proxy.SetRequestLimiter(limiter)
		resp, _ := proxy.handleRequest(context.Background(), toolCall(1, "echo"))
		if resp == nil || resp.Error == nil || resp.Error.Code != CodePolicyDenied {
			t.Fatalf("tokenless call: %+v", resp)
		}
		if n := len(limiter.seen()); n != 0 {
			t.Fatalf("limiter consulted %d times before auth", n)
		}
	})
	t.Run("revoked", func(t *testing.T) {
		proxy, _ := newLimiterTestProxy(t, "none")
		proxy.SetRevocationChecker(revokeAll{})
		limiter := &scriptedLimiter{}
		proxy.SetRequestLimiter(limiter)
		resp, _ := proxy.handleRequest(context.Background(), toolCall(1, "echo"))
		if resp == nil || resp.Error == nil || resp.Error.Message != "token revoked" {
			t.Fatalf("revoked call: %+v", resp)
		}
		if n := len(limiter.seen()); n != 0 {
			t.Fatalf("limiter consulted %d times for a revoked token", n)
		}
	})
}

// With no limiter installed (the OSS default) nothing changes.
func TestNoRequestLimiterLeavesCallsUnlimited(t *testing.T) {
	proxy, _ := newLimiterTestProxy(t, "none")
	for i := 1; i <= 5; i++ {
		if resp, _ := proxy.handleRequest(context.Background(), toolCall(i, "echo")); resp == nil || resp.Error != nil {
			t.Fatalf("call %d refused without a limiter: %+v", i, resp)
		}
	}
	if got := streamableForwardCount(proxy); got != 5 {
		t.Fatalf("forwards=%d, want 5", got)
	}
}
