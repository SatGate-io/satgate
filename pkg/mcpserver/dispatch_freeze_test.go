package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

const (
	dispatchAlias = "alpaca_place_crypto_order"
	dispatchName  = "place_crypto_order"
	dispatchToken = "dummy-dispatch-caller-token"
	dispatchArgs  = `{"symbol":"DUMMY/USD","side":"buy","notional":"25.50"}`
	dispatchInput = `{"name":"` + dispatchAlias + `","arguments":` + dispatchArgs + `,"_meta":{"progressToken":7,"token":"` + dispatchToken + `"}}`
	dispatchWant  = `{"_meta":{"progressToken":7},"arguments":` + dispatchArgs + `,"name":"` + dispatchName + `"}`
)

type dispatchRecorderFunc func(context.Context, []byte) error

func (f dispatchRecorderFunc) BeforeRelease(ctx context.Context, msg []byte) error {
	return f(ctx, msg)
}

type dispatchWriteProbe struct {
	Transport
	writes []json.RawMessage
}

func (p *dispatchWriteProbe) WriteMessage(ctx context.Context, msg json.RawMessage) error {
	p.writes = append(p.writes, bytes.Clone(msg))
	return p.Transport.WriteMessage(ctx, msg)
}

func dispatchFixture(t *testing.T, setup func(*UpstreamManager)) (*UpstreamManager, *dispatchWriteProbe, <-chan []byte) {
	t.Helper()
	rh, alp := twoBrokers(t)
	posts := make(chan []byte, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		var req Request
		if r.Method == http.MethodPost && json.Unmarshal(body, &req) == nil && req.Method == MethodToolsCall {
			select {
			case posts <- body:
			default:
				t.Error("unexpected additional tools/call POST")
			}
		}
		r.Body = io.NopCloser(bytes.NewReader(body))
		alp.serve(w, r)
	}))
	t.Cleanup(srv.Close)
	alc := alp.cfg("Alpaca")
	alc.URL = srv.URL + "/mcp"
	cfg := map[string]UpstreamConfig{"robinhood": rh.cfg("Robinhood"), "alpaca": alc}
	m := NewUpstreamManager(cfg, nil, "", true)
	m.SetServiceOrder([]string{"robinhood", "alpaca"})
	if setup != nil {
		setup(m)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(func() { cancel(); m.Close() })
	var probe *dispatchWriteProbe
	for _, name := range []string{"robinhood", "alpaca"} {
		client, err := m.connectStreamable(ctx, name, cfg[name])
		if err != nil {
			t.Fatal(err)
		}
		if name == "alpaca" {
			probe = &dispatchWriteProbe{Transport: client.transport}
			client.transport = probe
		}
		m.mu.Lock()
		m.clients[name] = client
		m.mu.Unlock()
		go m.readLoop(ctx, client)
		if err := m.initializeUpstream(ctx, client); err != nil {
			t.Fatal(err)
		}
		if err := m.discoverTools(ctx, client); err != nil {
			t.Fatal(err)
		}
	}
	return m, probe, posts
}

func dispatchWrittenCall(t *testing.T, p *dispatchWriteProbe, posts <-chan []byte) []byte {
	t.Helper()
	if len(p.writes) != 4 {
		t.Fatalf("WriteMessage calls = %d, want 3 startup writes and 1 tool call", len(p.writes))
	}
	written := p.writes[3]
	select {
	case body := <-posts:
		if !bytes.Equal(body, written) {
			t.Fatalf("POST body %s != WriteMessage bytes %s", body, written)
		}
	default:
		t.Fatal("missing tools/call POST")
	}
	want := `{"jsonrpc":"2.0","id":5,"method":"tools/call","params":` + dispatchWant + `}`
	if string(written) != want {
		t.Fatalf("WriteMessage got %s, want %s", written, want)
	}
	return written
}

func TestDispatchFreeze_LateMutationDiffersFromCapturedBytes(t *testing.T) {
	m, probe, posts := dispatchFixture(t, nil)
	var captured, mutated []byte
	m.SetDispatchRecorder(dispatchRecorderFunc(func(_ context.Context, msg []byte) error {
		captured = bytes.Clone(msg)
		i := bytes.Index(msg, []byte(dispatchName))
		if i < 0 {
			t.Fatal("missing tool name in recorder bytes")
		}
		msg[i] = 'X'
		mutated = msg
		return nil
	}))
	if _, err := m.ForwardToolCall(context.Background(), dispatchAlias, json.RawMessage(dispatchInput), time.Second); err != nil {
		t.Fatal(err)
	}
	written := dispatchWrittenCall(t, probe, posts)
	if !bytes.Equal(written, captured) || bytes.Equal(written, mutated) {
		t.Fatalf("written = %s, captured = %s, mutated = %s", written, captured, mutated)
	}
}

func TestDispatchFreeze_CapturedBytesEqualSentBytes(t *testing.T) {
	var captured []byte
	calls := 0
	m, probe, posts := dispatchFixture(t, func(m *UpstreamManager) {
		p := &Proxy{upstream: m}
		p.SetDispatchRecorder(dispatchRecorderFunc(func(_ context.Context, msg []byte) error {
			calls++
			captured = bytes.Clone(msg)
			return nil
		}))
	})
	if calls != 0 {
		t.Fatalf("recorder calls during initialize, notification, and tools/list = %d, want 0", calls)
	}
	if _, err := m.ForwardToolCall(context.Background(), dispatchAlias, json.RawMessage(dispatchInput), time.Second); err != nil {
		t.Fatal(err)
	}
	written := dispatchWrittenCall(t, probe, posts)
	if calls != 1 || !bytes.Equal(captured, written) {
		t.Fatalf("recorder calls = %d, captured = %s, written = %s", calls, captured, written)
	}
	if bytes.Contains(written, []byte(dispatchToken)) {
		t.Fatal("caller token reached WriteMessage")
	}
}

func TestDispatchFreeze_RecorderErrorDoesNotSend(t *testing.T) {
	m, probe, posts := dispatchFixture(t, nil)
	want := errors.New("test recorder error")
	ctx, record := withDispatchRecord(context.Background())
	m.SetDispatchRecorder(dispatchRecorderFunc(func(ctx context.Context, _ []byte) error {
		if dispatchFrom(ctx) != record {
			t.Fatal("recorder did not receive the dispatch record")
		}
		return want
	}))
	resp, err := m.ForwardToolCall(ctx, dispatchAlias, json.RawMessage(dispatchInput), time.Second)
	if resp != nil || err != want {
		t.Fatalf("response = %v, error = %v, want nil response and the recorder error", resp, err)
	}
	if len(probe.writes) != 3 || len(posts) != 0 {
		t.Fatalf("recorder error reached the transport: %d writes (including 3 startup writes), %d tool POSTs", len(probe.writes), len(posts))
	}
	record.recordForward(resp, err)
	if !record.attempted.Load() || record.sent.Load() || record.classify() != endNotRun {
		t.Fatalf("attempted = %v, sent = %v, classification = %v", record.attempted.Load(), record.sent.Load(), record.classify())
	}
}

func TestDispatchFreeze_NilRecorder(t *testing.T) {
	m, probe, posts := dispatchFixture(t, nil)
	if _, err := m.ForwardToolCall(context.Background(), dispatchAlias, json.RawMessage(dispatchInput), time.Second); err != nil {
		t.Fatal(err)
	}
	dispatchWrittenCall(t, probe, posts)
}
