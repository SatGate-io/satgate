package mcpserver

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

// What a call to a dead upstream gets depends on which connection it is
// written on. These tests force both cases instead of leaving it to the
// timing of a closed keep-alive connection.
//
//	(a) no pooled connection exists: the call can only dial, the dial is
//	    refused, and not a byte (not even a connection) was obtained. The call
//	    was never sent, so the reservation is released.
//	(b) a pooled idle connection exists and the upstream side of it is dead,
//	    but the client has not noticed: the call is written on it. The
//	    connection was obtained, so the call may have left. Whatever the
//	    upstream did is unknown, and the reservation is kept.
//
// The rule is in dispatch_record.go: traceDispatch marks "sent" at GotConn,
// and classify returns endNotRun only for attempted && !sent.

// connProbe sits in front of the upstream HTTP transport and counts what the
// transport did for each request: dials started and failed, connections
// obtained, and how many of those were reused from the pool.
type connProbe struct {
	inner http.RoundTripper
	idle  chan struct{} // one send per connection put back in the pool

	dialStarts atomic.Int64
	dialFailed atomic.Int64
	gotConn    atomic.Int64
	reused     atomic.Int64
}

func (p *connProbe) RoundTrip(req *http.Request) (*http.Response, error) {
	trace := &httptrace.ClientTrace{
		ConnectStart: func(string, string) { p.dialStarts.Add(1) },
		ConnectDone: func(_, _ string, err error) {
			if err != nil {
				p.dialFailed.Add(1)
			}
		},
		GotConn: func(i httptrace.GotConnInfo) {
			p.gotConn.Add(1)
			if i.Reused {
				p.reused.Add(1)
			}
		},
		PutIdleConn: func(err error) {
			if err == nil {
				select {
				case p.idle <- struct{}{}:
				default:
				}
			}
		},
	}
	return p.inner.RoundTrip(req.WithContext(httptrace.WithClientTrace(req.Context(), trace)))
}

func (p *connProbe) reset() {
	p.dialStarts.Store(0)
	p.dialFailed.Store(0)
	p.gotConn.Store(0)
	p.reused.Store(0)
}

// deadPooledConn is a client-side connection whose far end the client has not
// noticed is gone. Until arm is called it passes everything through. After
// arm, a read that would report the peer's close (EOF or a reset) waits until
// the first write or Close, so net/http keeps the connection in its idle pool
// and hands it to the next call; once that call has written, the real result
// is reported. failWrite, when set, makes the first write after arm fail
// without sending a byte.
type deadPooledConn struct {
	net.Conn
	gate      chan struct{}
	once      sync.Once
	hidden    atomic.Bool
	failWrite error
	wrote     atomic.Int64
}

func newDeadPooledConn(c net.Conn, failWrite error) *deadPooledConn {
	return &deadPooledConn{Conn: c, gate: make(chan struct{}), failWrite: failWrite}
}

func (c *deadPooledConn) open() { c.once.Do(func() { close(c.gate) }) }

// arm starts hiding the peer's close. Call it before the upstream is closed.
func (c *deadPooledConn) arm() { c.hidden.Store(true) }

func (c *deadPooledConn) Read(p []byte) (int, error) {
	n, err := c.Conn.Read(p)
	if err != nil && n == 0 && c.hidden.Load() {
		<-c.gate
	}
	return n, err
}

func (c *deadPooledConn) Write(p []byte) (int, error) {
	if !c.hidden.Load() {
		n, err := c.Conn.Write(p)
		c.wrote.Add(int64(n))
		return n, err
	}
	defer c.open()
	if c.failWrite != nil {
		return 0, c.failWrite
	}
	n, err := c.Conn.Write(p)
	c.wrote.Add(int64(n))
	return n, err
}

func (c *deadPooledConn) Close() error {
	c.open()
	return c.Conn.Close()
}

// harnessStreamable is the Streamable HTTP transport the harness's proxy sends
// upstream "u" calls through.
func harnessStreamable(t *testing.T, h *spendHarness) *StreamableHTTPTransport {
	t.Helper()
	m := h.proxy.upstream
	m.mu.Lock()
	c := m.clients["u"]
	m.mu.Unlock()
	if c == nil {
		t.Fatal("harness has no upstream client \"u\"")
	}
	tr, ok := c.transport.(*StreamableHTTPTransport)
	if !ok {
		t.Fatalf("upstream transport is %T, want *StreamableHTTPTransport", c.transport)
	}
	return tr
}

// freshUpstreamClient replaces the harness's upstream HTTP client with a new
// one, built the way the product builds it (newUpstreamHTTPClient), whose pool
// is empty. The old client's idle connections are closed. A call through the
// new client can only dial. wrapDial, if set, wraps each connection the new
// client dials.
func freshUpstreamClient(t *testing.T, h *spendHarness, wrapDial func(net.Conn) net.Conn) *connProbe {
	t.Helper()
	tr := harnessStreamable(t, h)
	if old, ok := tr.httpClient.Transport.(*http.Transport); ok {
		old.CloseIdleConnections()
	}
	client := newUpstreamHTTPClient(false, true)
	ht := client.Transport.(*http.Transport)
	if wrapDial != nil {
		dial := ht.DialContext
		ht.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			c, err := dial(ctx, network, addr)
			if err != nil {
				return nil, err
			}
			return wrapDial(c), nil
		}
	}
	t.Cleanup(ht.CloseIdleConnections)
	probe := &connProbe{inner: ht, idle: make(chan struct{}, 8)}
	client.Transport = probe
	tr.httpClient = client
	return probe
}

// poolOneConn sends a request that nothing upstream acts on (a GET the fake
// answers 405) through the probe's transport and waits until its connection is
// back in the idle pool, so the next call finds it there.
func poolOneConn(t *testing.T, up *orderUpstream, probe *connProbe) {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, up.srv.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := probe.RoundTrip(req)
	if err != nil {
		t.Fatalf("priming request: %v", err)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	select {
	case <-probe.idle:
	case <-time.After(10 * time.Second):
		t.Fatal("the primed connection never went back to the idle pool")
	}
	probe.reset()
}

// A call written on a pooled connection whose upstream side is dead: the
// connection was obtained and the request was handed to it, so the outcome is
// unknown and the reservation stays counted. This is what a dial-failure test
// gets by accident when the client still holds an idle keep-alive connection
// to the upstream it has just closed.
func TestSpendKeptWhenCallIsWrittenOnDeadPooledConnection(t *testing.T) {
	cases := []struct {
		name      string
		failWrite error
		wantWrote bool
	}{
		{"request written, no reply", nil, true},
		{"write fails on the dead connection", syscall.EPIPE, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			up := newOrderUpstream(t)
			h := httpSpendHarness(t, up, 2*time.Second)

			var conn atomic.Pointer[deadPooledConn]
			probe := freshUpstreamClient(t, h, func(c net.Conn) net.Conn {
				d := newDeadPooledConn(c, tc.failWrite)
				conn.Store(d)
				return d
			})
			poolOneConn(t, up, probe)
			conn.Load().arm()
			wroteBefore := conn.Load().wrote.Load()
			up.srv.Close() // the far end of the pooled connection is gone

			r := h.call(t, 1, "place_crypto_order", order("200"))
			if r.Error == nil {
				t.Fatalf("call to a dead upstream succeeded: %s", r.Result)
			}
			d := conn.Load()
			if d == nil {
				t.Fatal("no connection was dialed")
			}
			if probe.reused.Load() != 1 || probe.gotConn.Load() != 1 {
				t.Fatalf("the call used %d connections, %d of them pooled; want 1 pooled", probe.gotConn.Load(), probe.reused.Load())
			}
			if tc.wantWrote && d.wrote.Load() == wroteBefore {
				t.Fatal("the call was not written on the pooled connection")
			}
			if h.counter(t) != 200*units {
				t.Fatalf("counter after a call written on a dead pooled connection = %d, want 200 kept (the outcome is unknown)", h.counter(t)/units)
			}
			if up.executed.Load() != 0 {
				t.Fatalf("executed %d", up.executed.Load())
			}
			// The kept reservation is the point: nothing ran, but SatGate
			// could not know, so the allowance is not whole again.
			if r2 := h.call(t, 2, "place_crypto_order", order("200")); r2.Error == nil {
				t.Fatal("a second order went through while the first one's outcome was unknown")
			}
		})
	}
}
