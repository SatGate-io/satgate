package proxy

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/satgate-io/satgate/pkg/config"
	"github.com/satgate-io/satgate/pkg/macaroon"
)

// Tests of the whole-field rule on public and chargeback routes
// (upstream_credentials.go removeGatewayTokens, carriesGatewayToken): when
// any Authorization value carries a token this gateway issued, as written
// or inside one wrapper, the whole Authorization field is not forwarded;
// otherwise every value is forwarded unchanged. Values are fixtures, not
// credentials.

// wireRoute is a public or chargeback route on an MCP-enabled test gateway
// (so trailers are forwarded), with the upstream recorder and front server.
type wireRoute struct {
	kind  string
	gw    *Gateway
	up    *requestRecorder
	front *httptest.Server
}

func newWireRoute(t *testing.T, kind string) *wireRoute {
	t.Helper()
	up := newRequestRecorder(t)
	gw, cfg := newTestGatewayWithUpstream(t, up.srv, kind)
	cfg.Routes[0].MCP = &config.MCPConfig{Enabled: true}
	front := httptest.NewServer(gw)
	t.Cleanup(front.Close)
	return &wireRoute{kind: kind, gw: gw, up: up, front: front}
}

// headers sends values as Authorization header lines and returns the
// Authorization values the upstream received.
func (w *wireRoute) headers(t *testing.T, values ...string) []string {
	t.Helper()
	return sendAuthorization(t, w.front.URL, w.up, values...).Header.Values("Authorization")
}

// trailers sends values as Authorization trailer lines and returns the
// Authorization trailer values the upstream received.
func (w *wireRoute) trailers(t *testing.T, values ...string) []string {
	t.Helper()
	tr := make([][2]string, len(values))
	for i, v := range values {
		tr[i] = [2]string{"Authorization", v}
	}
	if status := sendRaw(t, w.front.URL, trailerRequest("/api/data", "", tr)); status != http.StatusOK {
		t.Fatalf("status %d", status)
	}
	return w.up.take(t).Trailer.Values("Authorization")
}

// both sends values as header lines and then as trailer lines, and calls
// check with what the upstream received each time.
func (w *wireRoute) both(t *testing.T, values []string, check func(where string, got []string)) {
	t.Helper()
	check("headers", w.headers(t, values...))
	check("trailers", w.trailers(t, values...))
}

// withLineBreaks returns tok with CR LF or LF inserted where Go's base64
// decoders, and so Verify, skip them.
func withLineBreaks(tok string) map[string]string {
	out := map[string]string{
		"LF in the middle":          tok[:40] + "\n" + tok[40:],
		"CRLF in the middle":        tok[:40] + "\r\n" + tok[40:],
		"LF after the first char":   tok[:1] + "\n" + tok[1:],
		"CRLF at the end":           tok + "\r\n",
		"LF at the start":           "\n" + tok,
		"LF every 16 chars":         everyN(tok, 16, "\n"),
		"CRLF every 64 chars":       everyN(tok, 64, "\r\n"),
		"LF before the last char":   tok[:len(tok)-1] + "\n" + tok[len(tok)-1:],
		"CRLF and LF in two places": tok[:20] + "\r\n" + tok[20:60] + "\n" + tok[60:],
	}
	if strings.HasSuffix(tok, "==") {
		out["LF inside the padding"] = tok[:len(tok)-1] + "\n="
		out["CRLF inside the padding"] = tok[:len(tok)-1] + "\r\n="
	}
	return out
}

func everyN(s string, n int, sep string) string {
	var b strings.Builder
	for i := 0; i < len(s); i += n {
		if i > 0 {
			b.WriteString(sep)
		}
		b.WriteString(s[i:min(i+n, len(s))])
	}
	return b.String()
}

// verifierSpellings returns every spelling of the gateway's token that
// Verify accepts, for both macaroon formats: the encodings Verify reads
// (JSON in URL-safe unpadded or standard padded base64, binary in standard
// base64 padded or not), each also with line breaks Verify skips. It fails
// the test if Verify refuses any of them.
func verifierSpellings(t *testing.T, gw *Gateway) map[string]string {
	t.Helper()
	tok, _ := mintRoot(t, gw)
	mac, err := gw.macaroonSvc.Verify(tok)
	if err != nil {
		t.Fatal(err)
	}
	bin, err := gw.macaroonSvc.EncodeBinary(mac)
	if err != nil {
		t.Fatal(err)
	}
	jsonData, err := base64.RawURLEncoding.DecodeString(tok)
	if err != nil {
		t.Fatal(err)
	}
	binData, err := base64.StdEncoding.DecodeString(bin)
	if err != nil {
		t.Fatal(err)
	}
	base := map[string]string{
		"json url-safe unpadded": tok,
		"json standard padded":   base64.StdEncoding.EncodeToString(jsonData),
		"binary standard padded": bin,
		"binary standard raw":    base64.RawStdEncoding.EncodeToString(binData),
	}
	out := map[string]string{}
	for name, s := range base {
		out[name] = s
		for brk, v := range withLineBreaks(s) {
			out[name+", "+brk] = v
		}
	}
	for name, s := range out {
		if _, err := gw.macaroonSvc.Verify(s); err != nil {
			t.Fatalf("fixture %q is not a spelling Verify accepts: %v", name, err)
		}
	}
	return out
}

// percentAll percent-encodes every byte of s.
func percentAll(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		fmt.Fprintf(&b, "%%%02X", s[i])
	}
	return b.String()
}

// The cases both reviewers reported against round 3, on the wire, in
// headers and trailers, on public and chargeback routes, each with a foreign
// value on another line: no Authorization value reaches the upstream.
//   - A quoted token (every character escaped) after or before a part whose
//     quoting is broken.
//   - A token spelled with a line break that Verify skips, inside one Basic
//     wrapper, in both macaroon formats; and the same token percent-encoded
//     or base64-encoded once.
func TestUpstreamCredentials_ReviewerRound3CasesDropWholeFieldOnTheWire(t *testing.T) {
	for _, kind := range []string{"public", "chargeback"} {
		t.Run(kind, func(t *testing.T) {
			w := newWireRoute(t, kind)
			tok, _ := mintRoot(t, w.gw)
			mac, err := w.gw.macaroonSvc.Verify(tok)
			if err != nil {
				t.Fatal(err)
			}
			bin, err := w.gw.macaroonSvc.EncodeBinary(mac)
			if err != nil {
				t.Fatal(err)
			}
			quoted := `Digest token="` + escapeEvery(tok) + `"`
			cases := map[string]string{
				"quoted token":                          quoted,
				"unclosed quote and comma before":       `Digest unclosed=" , ` + quoted,
				"unfinished realm before":               `Digest realm="unfinished, ` + quoted,
				"broken realm after":                    quoted + `, Digest realm="broken`,
				"broken realm before, plain token":      `Digest realm="broken, Bearer ` + tok,
				"broken realm after, percent token":     "Bearer " + percentFirst(tok) + `, Digest realm="broken`,
				"malformed percent before, quoted":      `Digest realm="%GG", ` + quoted,
				"malformed percent run after, unclosed": quoted + `, extra="` + strings.Repeat("%Z", 200),
			}
			for format, s := range map[string]string{"json": tok, "binary": bin} {
				for _, brk := range []string{"\n", "\r\n"} {
					v := s[:40] + brk + s[40:]
					if _, err := w.gw.macaroonSvc.Verify(v); err != nil {
						t.Fatalf("fixture: Verify refuses the %s token with a line break: %v", format, err)
					}
					name := fmt.Sprintf("%s token with %q", format, brk)
					cases["Basic user:token, "+name] = "Basic " + b64("user:"+v)
					cases["percent-encoded, "+name] = "Bearer " + percentAll(v)
					cases["base64-encoded, "+name] = "Bearer " + b64(v)
				}
			}
			for name, v := range cases {
				w.both(t, []string{"Basic foreign-fixture", v}, func(where string, got []string) {
					if len(got) != 0 {
						t.Errorf("%s, %s: upstream received %d Authorization values (the token's value as sent among them: %v), want none", name, where, len(got), slices.Contains(got, v))
					}
				})
			}
		})
	}
}

// A token inside one Basic wrapper (user:<token> and <token>:, in either
// base64 alphabet, padded or not) is found in every spelling Verify
// accepts, in both macaroon formats, and the field is not forwarded.
func TestUpstreamCredentials_BasicWrappedTokenInEverySpellingVerifyAcceptsDropsField(t *testing.T) {
	wrappers := map[string]*base64.Encoding{
		"standard": base64.StdEncoding, "standard raw": base64.RawStdEncoding,
		"url-safe": base64.URLEncoding, "url-safe raw": base64.RawURLEncoding,
	}
	for _, kind := range []string{"public", "chargeback"} {
		t.Run(kind, func(t *testing.T) {
			w := newWireRoute(t, kind)
			for spelling, s := range verifierSpellings(t, w.gw) {
				for wname, enc := range wrappers {
					for shape, payload := range map[string]string{"user:token": "user:" + s, "token:": s + ":"} {
						v := "Basic " + enc.EncodeToString([]byte(payload))
						where := []string{"headers"}
						if wname == "standard" {
							where = append(where, "trailers")
						}
						for _, at := range where {
							var got []string
							if at == "headers" {
								got = w.headers(t, v)
							} else {
								got = w.trailers(t, v)
							}
							if len(got) != 0 {
								t.Errorf("%s in Basic %s (%s), %s: forwarded", spelling, shape, wname, at)
							}
						}
					}
				}
			}
		})
	}
}

// surroundings are malformed or unusual text placed around a value: broken
// and unclosed quotes, malformed parameters and percent sequences, empty
// elements, stray backslashes and a long malformed parameter.
var surroundings = map[string]func(v string) string{
	"as is":                       func(v string) string { return v },
	"unclosed quote before":       func(v string) string { return `Digest realm="unclosed, ` + v },
	"unclosed quote after":        func(v string) string { return v + `, Digest realm="unclosed` },
	"lone quote between":          func(v string) string { return `Digest a=" , ` + v + ` ", b="` },
	"escaped quote before":        func(v string) string { return `Digest realm="a\"b, ` + v },
	"backslash at the end":        func(v string) string { return v + `, x="\` },
	"malformed percent before":    func(v string) string { return `Digest realm="%GG", nonce="%zz", ` + v },
	"malformed percent after":     func(v string) string { return v + `, realm="%GG", opaque="%"` },
	"lone percent":                func(v string) string { return "% " + v + " %" },
	"empty elements":              func(v string) string { return ", ,," + v + ", ,," },
	"semicolons and equals":       func(v string) string { return "a=;b==;" + v + ";c=" },
	"invalid base64 before":       func(v string) string { return "Basic !!!not-base64!!!, " + v },
	"parameter without a value":   func(v string) string { return "Digest a=, b, " + v + ", c=" },
	"long malformed parameter":    func(v string) string { return `Digest opaque="` + strings.Repeat("%Z", 1000) + `", ` + v },
	"quoted string around it all": func(v string) string { return `"` + v + `"` },
}

// A gateway token in any form, with broken quotes, malformed parameters or
// percent sequences around it, and a foreign value on another line: the
// whole Authorization field is not forwarded, in headers and trailers.
func TestUpstreamCredentials_BrokenQuotesParametersAndPercentAroundTokenDropWholeField(t *testing.T) {
	for _, kind := range []string{"public", "chargeback"} {
		t.Run(kind, func(t *testing.T) {
			w := newWireRoute(t, kind)
			tok, _ := mintRoot(t, w.gw)
			mac, err := w.gw.macaroonSvc.Verify(tok)
			if err != nil {
				t.Fatal(err)
			}
			bin, err := w.gw.macaroonSvc.EncodeBinary(mac)
			if err != nil {
				t.Fatal(err)
			}
			forms := map[string]string{
				"Bearer":          "Bearer " + tok,
				"bare":            tok,
				"L402 binary":     "L402 " + bin + ":" + strings.Repeat("ab", 32),
				"quoted":          `Digest token="` + tok + `"`,
				"quoted, escaped": `Digest token="` + escapeEvery(tok) + `"`,
				"parameter":       "Digest token=" + tok,
				"percent":         "Bearer " + percentFirst(tok),
				"Basic":           "Basic " + b64("user:"+tok),
				"base64":          "Bearer " + b64(tok),
			}
			for fname, form := range forms {
				for sname, wrap := range surroundings {
					v := wrap(form)
					w.both(t, []string{v, "Bearer upstream-own-api-key"}, func(where string, got []string) {
						if len(got) != 0 {
							t.Errorf("%s, %s, %s: upstream received %d Authorization values (the token's value as sent among them: %v), want none", fname, sname, where, len(got), slices.Contains(got, v))
						}
					})
				}
			}
		})
	}
}

// One value with a gateway token among many foreign values, first, in the
// middle or last: none of the values is forwarded.
func TestUpstreamCredentials_ManyValuesOneTokenDropsEveryValue(t *testing.T) {
	for _, kind := range []string{"public", "chargeback"} {
		t.Run(kind, func(t *testing.T) {
			w := newWireRoute(t, kind)
			tok, _ := mintRoot(t, w.gw)
			for _, n := range []int{20, 2000} {
				for _, at := range []string{"first", "middle", "last"} {
					values := make([]string, n)
					for i := range values {
						values[i] = fmt.Sprintf("Basic foreign-fixture-%d", i)
					}
					switch at {
					case "first":
						values[0] = `Digest realm="broken, Bearer ` + tok
					case "middle":
						values[n/2] = `Digest realm="broken, Bearer ` + tok
					default:
						values[n-1] = `Digest realm="broken, Bearer ` + tok
					}
					if got := w.headers(t, values...); len(got) != 0 {
						t.Errorf("%d values, token %s, headers: upstream received %d values, want none", n, at, len(got))
					}
					if n <= 20 {
						if got := w.trailers(t, values...); len(got) != 0 {
							t.Errorf("%d values, token %s, trailers: upstream received %d values, want none", n, at, len(got))
						}
					}
				}
			}
		})
	}
}

// Foreign values with the same malformations, and foreign macaroons in the
// same forms and wrappers (another key signed them), reach the upstream
// byte for byte and in order, in headers and trailers.
func TestUpstreamCredentials_ForeignOnlyMalformedValuesForwardedUnchanged(t *testing.T) {
	other, err := macaroon.NewService("a-different-root-key-for-tests")
	if err != nil {
		t.Fatal(err)
	}
	fm, err := other.Mint("api:read", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	foreign := other.Encode(fm)
	foreignBin, err := other.EncodeBinary(fm)
	if err != nil {
		t.Fatal(err)
	}
	forms := map[string]string{
		"api key":                    "Bearer upstream-own-api-key-0123456789",
		"foreign macaroon":           "Bearer " + foreign,
		"foreign L402":               "L402 " + foreignBin + ":" + strings.Repeat("cd", 32),
		"foreign quoted, escaped":    `Digest token="` + escapeEvery(foreign) + `"`,
		"foreign percent":            "Bearer " + percentFirst(foreign),
		"foreign Basic":              "Basic " + b64("user:"+foreign),
		"foreign Basic, line break":  "Basic " + b64("user:"+foreign[:40]+"\r\n"+foreign[40:]),
		"foreign percent line break": "Bearer " + percentAll(foreign[:40]+"\n"+foreign[40:]),
		"Basic credentials":          "Basic dXNlcjpwYXNz",
	}
	for _, kind := range []string{"public", "chargeback"} {
		t.Run(kind, func(t *testing.T) {
			w := newWireRoute(t, kind)
			for fname, form := range forms {
				for sname, wrap := range surroundings {
					values := []string{wrap(form), "Basic second-foreign-value"}
					w.both(t, values, func(where string, got []string) {
						if strings.Join(got, "\n") != strings.Join(values, "\n") {
							t.Errorf("%s, %s, %s: foreign values changed or dropped (%d values)", fname, sname, where, len(got))
						}
					})
				}
			}
		})
	}
}

// timeRemovals returns, for each header, the time of one
// removeGatewayCredentials call on it: the median of five timed batches,
// after one warm-up batch. The headers take turns, batch by batch, so a slow
// stretch on a busy machine falls on all of them rather than on one. A batch
// runs reps[i] fresh, separately allocated copies back to back; with reps
// chosen so that every batch reads the same amount of input, each batch lasts
// about as long, and a short run cannot fit between interruptions that a long
// run cannot avoid. The median, unlike the shortest batch, is not set by one
// batch that ran unusually fast or slow.
func timeRemovals(gw *Gateway, hs []http.Header, reps []int) []time.Duration {
	samples := make([][]time.Duration, len(hs))
	for round := 0; round < 6; round++ {
		for s, h := range hs {
			copies := make([]http.Header, reps[s])
			for j := range copies {
				c := make(http.Header, len(h))
				for name, values := range h {
					own := make([]string, len(values))
					for k, v := range values {
						own[k] = strings.Clone(v)
					}
					c[name] = own
				}
				copies[j] = c
			}
			runtime.GC()
			start := time.Now()
			for _, c := range copies {
				gw.removeGatewayCredentials(c, false)
			}
			elapsed := time.Since(start) / time.Duration(reps[s])
			if round > 0 {
				samples[s] = append(samples[s], elapsed)
			}
		}
	}
	median := make([]time.Duration, len(hs))
	for s, d := range samples {
		slices.Sort(d)
		median[s] = d[len(d)/2]
	}
	return median
}

// Finding gateway tokens in Authorization costs allocations and time linear
// in the input: 20 KB and 80 KB of each shape cost less than six times what
// 5 KB and 20 KB cost, whether the field is forwarded or dropped.
func TestRemoveGatewayCredentials_5KB20KB80KBCostStaysLinear(t *testing.T) {
	gw, _ := newTestGatewayWithUpstream(t, newHeaderRecorder(t).srv, "public")
	tok, _ := mintRoot(t, gw)
	fill := func(unit string, size int) string {
		return strings.Repeat(unit, size/len(unit)+1)[:size]
	}
	shapes := map[string]func(size int) []string{
		"parameters": func(n int) []string { return []string{"Digest " + fill("a=1, ", n)} },
		"parameters, token": func(n int) []string {
			return []string{"Digest " + fill("a=1, ", n) + `, token="` + percentFirst(tok) + `"`}
		},
		"broken quotes": func(n int) []string { return []string{fill(`Digest realm="x, `, n)} },
		"broken quotes, token": func(n int) []string {
			return []string{fill(`Digest realm="x, `, n) + `, token="` + escapeEvery(tok) + `"`}
		},
		"escaped quoted pairs": func(n int) []string { return []string{"Digest a=\"" + fill(`\A\%`, n) + `"`} },
		"one base64 run":       func(n int) []string { return []string{"Basic " + fill("QUFB", n)} },
		"base64 of text runs":  func(n int) []string { return []string{"Basic " + b64(fill("user:pass-0123456789abcdef ", n*3/4))} },
		"base64 of line breaks": func(n int) []string {
			return []string{"Basic " + b64(fill("QUFBQUFBQUFBQUFBQUFBQUFB\r\n", n*3/4))}
		},
		"percent sequences": func(n int) []string { return []string{"Bearer " + fill("%41%2", n)} },
		"padding splits":    func(n int) []string { return []string{fill("aaaaaaaaaaaaaaaaaaaa=", n)} },
		"many values": func(n int) []string {
			v := make([]string, n/len("Basic foreign-fixture"))
			for i := range v {
				v[i] = "Basic foreign-fixture"
			}
			return v
		},
		"many values, token last": func(n int) []string {
			v := make([]string, n/len("Basic foreign-fixture"))
			for i := range v {
				v[i] = "Basic foreign-fixture"
			}
			return append(v, "Bearer "+tok)
		},
	}
	for name, shape := range shapes {
		t.Run(name, func(t *testing.T) {
			sizes := []int{5 << 10, 20 << 10, 80 << 10}
			hs := make([]http.Header, len(sizes))
			reps := make([]int, len(sizes))
			for i, size := range sizes {
				hs[i] = http.Header{"Authorization": shape(size)}
				reps[i] = sizes[len(sizes)-1] / size
			}
			elapsed := timeRemovals(gw, hs, reps)
			var prev parserCost
			for i, size := range sizes {
				h := hs[i]
				c := measureRemoval(gw, h)
				c.elapsed = elapsed[i]
				t.Logf("%s, %d KB: allocated %d bytes in %d allocations, %v", name, size>>10, c.bytes, c.allocs, c.elapsed)
				if i > 0 {
					if c.bytes > 6*prev.bytes+4096 || c.allocs > 6*prev.allocs+16 {
						t.Errorf("%d KB: allocations grew more than 6x for 4x the input (%d -> %d bytes, %d -> %d allocations)", size>>10, prev.bytes, c.bytes, prev.allocs, c.allocs)
					}
					if c.elapsed > 6*prev.elapsed+time.Millisecond {
						t.Errorf("%d KB: time grew more than 6x for 4x the input (%v -> %v)", size>>10, prev.elapsed, c.elapsed)
					}
				}
				prev = c
			}
		})
	}
}
