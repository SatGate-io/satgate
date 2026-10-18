package mcpserver

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"os"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/satgate-io/satgate/pkg/toolcontrols"
)

// What a call to a dead upstream gets depends on which connection it is
// written on. These tests force both cases instead of leaving it to the
// timing of a closed keep-alive connection or to which process holds a
// localhost port.
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
// transport did for each request: connections obtained and how many of those
// were reused from the pool. Dials are counted where the transport makes them
// (see freshUpstreamClient).
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

// deadPooledConn is a client-side connection whose far end is gone, but which
// the client has not noticed. Until arm is called it passes everything through.
// After arm:
//
//   - a read that would report the peer's close (EOF or a reset) waits until
//     the first write or Close, so net/http keeps the connection in its idle
//     pool and hands it to the next call; once that call has written, the read
//     reports EOF;
//   - the first write is not sent to the real socket. The real socket is closed
//     instead, so no server, this test's or any other process's, can receive
//     the call and the outcome does not depend on how fast the real server
//     side closes. The write then reports what a dead peer gives: failWrite
//     (EPIPE) when set, otherwise it is swallowed (success) and the next read
//     is EOF.
type deadPooledConn struct {
	net.Conn
	gate      chan struct{}
	once      sync.Once
	dead      atomic.Bool
	failWrite error
	wrote     atomic.Int64 // bytes the transport handed to the connection
}

func newDeadPooledConn(c net.Conn, failWrite error) *deadPooledConn {
	return &deadPooledConn{Conn: c, gate: make(chan struct{}), failWrite: failWrite}
}

func (c *deadPooledConn) open() { c.once.Do(func() { close(c.gate) }) }

// arm marks the far end dead. Call it before the upstream is closed.
func (c *deadPooledConn) arm() { c.dead.Store(true) }

func (c *deadPooledConn) Read(p []byte) (int, error) {
	n, err := c.Conn.Read(p)
	if err != nil && n == 0 && c.dead.Load() {
		<-c.gate
		return 0, io.EOF
	}
	return n, err
}

func (c *deadPooledConn) Write(p []byte) (int, error) {
	if !c.dead.Load() {
		n, err := c.Conn.Write(p)
		c.wrote.Add(int64(n))
		return n, err
	}
	defer c.open()
	_ = c.Conn.Close() // nothing is sent; a read blocked on the real socket ends
	if c.failWrite != nil {
		return 0, c.failWrite
	}
	c.wrote.Add(int64(len(p)))
	return len(p), nil
}

func (c *deadPooledConn) Close() error {
	c.open()
	return c.Conn.Close()
}

// dialFunc is the dial function of an http.Transport.
type dialFunc func(ctx context.Context, network, addr string) (net.Conn, error)

// refusedDial replaces the dial with one that fails the way a closed port does
// (connect: connection refused) without opening a socket, so there is nothing
// another process could have bound, accepted or answered.
func refusedDial(dialFunc) dialFunc {
	return func(_ context.Context, network, addr string) (net.Conn, error) {
		op := &net.OpError{Op: "dial", Net: network, Err: os.NewSyscallError("connect", syscall.ECONNREFUSED)}
		if a, err := net.ResolveTCPAddr(network, addr); err == nil {
			op.Addr = a
		}
		return nil, op
	}
}

// wrapConn makes each connection the product's dial returns a deadPooledConn
// kept in *got.
func wrapConn(got *atomic.Pointer[deadPooledConn], failWrite error) func(dialFunc) dialFunc {
	return func(next dialFunc) dialFunc {
		return func(ctx context.Context, network, addr string) (net.Conn, error) {
			c, err := next(ctx, network, addr)
			if err != nil {
				return nil, err
			}
			d := newDeadPooledConn(c, failWrite)
			got.Store(d)
			return d, nil
		}
	}
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
// new client can only dial. wrapDial, if set, wraps the new client's dial
// function. Every dial the client makes is counted in the returned probe
// before it reaches wrapDial's function, whether it succeeds or not.
func freshUpstreamClient(t *testing.T, h *spendHarness, wrapDial func(dialFunc) dialFunc) *connProbe {
	t.Helper()
	tr := harnessStreamable(t, h)
	if old, ok := tr.httpClient.Transport.(*http.Transport); ok {
		old.CloseIdleConnections()
	}
	client := newUpstreamHTTPClient(false, true)
	ht := client.Transport.(*http.Transport)
	probe := &connProbe{idle: make(chan struct{}, 8)}
	dial := dialFunc(ht.DialContext)
	if wrapDial != nil {
		dial = wrapDial(dial)
	}
	ht.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		probe.dialStarts.Add(1)
		c, err := dial(ctx, network, addr)
		if err != nil {
			probe.dialFailed.Add(1)
		}
		return c, err
	}
	t.Cleanup(ht.CloseIdleConnections)
	probe.inner = ht
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
//
// Nothing in the fixture can decide the outcome but the product. The far end of
// the pooled connection is dead before the call is written: once armed,
// deadPooledConn does not forward the write to the real socket (it closes that
// socket), so the request cannot reach this test's upstream, or a server in
// another process that has taken the port. The call neither dials (a dial could
// reach a stranger) nor can it be answered. The fake upstream also carries out
// only requests that bear this fixture's token (requireToken), so a request
// from some other test that was aimed at a port this one now holds is not
// counted as an order.
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
			up.requireToken(t)
			h := httpSpendHarness(t, up, 2*time.Second)

			var conn atomic.Pointer[deadPooledConn]
			probe := freshUpstreamClient(t, h, wrapConn(&conn, tc.failWrite))
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
			if probe.dialStarts.Load() != 0 {
				t.Fatalf("the call dialed %d times; it must be written on the pooled connection without a dial", probe.dialStarts.Load())
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
			r2 := h.call(t, 2, "place_crypto_order", order("200"))
			if got := errData(t, r2)["error"]; got != toolcontrols.DenialCodeSpendLimit {
				t.Fatalf("second order: error = %v, want a spend-limit refusal while the first one's outcome was unknown", got)
			}
		})
	}
}
