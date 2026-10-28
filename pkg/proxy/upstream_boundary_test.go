package proxy

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/satgate-io/satgate/pkg/config"
)

// Tests of the removal boundary for gateway tokens in Authorization on
// public and chargeback routes (upstream_credentials.go): every form the
// gateway accepts, inside one wrapper; each credential and parameter judged
// on its own; two or more wrappers forwarded like other client data; work
// linear in the value's length. Values are fixtures, not credentials.

// percentFirst percent-encodes the first character of s, as a client that
// percent-encodes a value does for a character it escapes.
func percentFirst(s string) string {
	return fmt.Sprintf("%%%02X", s[0]) + s[1:]
}

// escapeEvery writes s with a backslash before every character, as a
// quoted string may.
func escapeEvery(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		b.WriteByte('\\')
		b.WriteByte(s[i])
	}
	return b.String()
}

func b64(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }

// sendAuthorizationTrailer sends value as an Authorization trailer to an
// MCP-enabled route (the gateway reads the body, so trailers are
// forwarded) and returns the Authorization trailer values the upstream
// received.
func sendAuthorizationTrailer(t *testing.T, kind string, value func(gw *Gateway) string) []string {
	t.Helper()
	up := newRequestRecorder(t)
	gw, cfg := newTestGatewayWithUpstream(t, up.srv, kind)
	cfg.Routes[0].MCP = &config.MCPConfig{Enabled: true}
	front := httptest.NewServer(gw)
	defer front.Close()
	raw := trailerRequest("/api/data", "", [][2]string{{"Authorization", value(gw)}})
	if status := sendRaw(t, front.URL, raw); status != http.StatusOK {
		t.Fatalf("status %d", status)
	}
	return up.take(t).Trailer.Values("Authorization")
}

// boundaryCase is an Authorization value built from a token the gateway
// issued, and the values the upstream must receive for it.
type boundaryCase struct {
	name  string
	value func(tok string) string
	want  func(tok string) []string
}

func none(string) []string { return nil }

// An unrelated malformed, empty or very long part of an Authorization value
// never stops a gateway token in another part of it from being removed, in
// headers and in trailers, on public and chargeback routes. Foreign
// credentials joined to it are kept.
func TestUpstreamCredentials_UnrelatedMalformedPartDoesNotStopRemoval(t *testing.T) {
	long := strings.Repeat("%Z", 20000)
	cases := []boundaryCase{
		{"percent token, malformed percent in another parameter", func(tok string) string {
			return `Digest token="` + percentFirst(tok) + `", realm="%GG"`
		}, none},
		{"malformed percent before the percent token", func(tok string) string {
			return `Digest realm="%GG", token="` + percentFirst(tok) + `"`
		}, none},
		{"lone percent after the percent token", func(tok string) string {
			return "Bearer " + percentFirst(tok) + " %"
		}, none},
		{"invalid base64 and malformed percent beside the percent token", func(tok string) string {
			return `Digest nonce="` + strings.Repeat("A", 17) + `", token=` + percentFirst(tok) + `, opaque="%zz"`
		}, none},
		{"long malformed parameter (3 KB)", func(tok string) string {
			return `Digest token="` + percentFirst(tok) + `", opaque="` + long[:3000] + `"`
		}, none},
		{"very long malformed parameter (40 KB)", func(tok string) string {
			return `Digest token="` + percentFirst(tok) + `", opaque="` + long + `"`
		}, none},
		{"empty elements", func(tok string) string {
			return `Digest , , token="` + percentFirst(tok) + `", ,realm=""`
		}, none},
		{"every character of the quoted token escaped", func(tok string) string {
			return `Digest token="` + escapeEvery(tok) + `", realm="%GG"`
		}, none},
		{"Basic credentials with a non-UTF-8 user name", func(tok string) string {
			return "Basic " + b64("\xe9t\xe9:"+tok)
		}, none},
		{"Basic credentials with a malformed parameter", func(tok string) string {
			return "Basic " + b64("user:"+tok) + `, realm="%GG"`
		}, none},
		{"unclosed quote after the token: whole value judged", func(tok string) string {
			return "Bearer " + percentFirst(tok) + `, Digest realm="unclosed`
		}, none},
		{"unclosed quote before the token: whole value judged", func(tok string) string {
			return `Digest realm="unclosed, Bearer ` + tok
		}, none},
		{"foreign credential kept beside the percent token", func(tok string) string {
			return `Basic Zm9yZWlnbg==, Digest token="` + percentFirst(tok) + `", realm="%GG"`
		}, func(string) []string { return []string{"Basic Zm9yZWlnbg=="} }},
	}
	for _, kind := range []string{"public", "chargeback"} {
		t.Run(kind+"/headers", func(t *testing.T) {
			up := newRequestRecorder(t)
			gw, _ := newTestGatewayWithUpstream(t, up.srv, kind)
			tok, _ := mintRoot(t, gw)
			front := httptest.NewServer(gw)
			defer front.Close()
			for _, c := range cases {
				got := sendAuthorization(t, front.URL, up, c.value(tok)).Header.Values("Authorization")
				if want := c.want(tok); strings.Join(got, "\n") != strings.Join(want, "\n") {
					t.Errorf("%s: upstream received %d Authorization values, want %d (the foreign ones only)", c.name, len(got), len(want))
				}
			}
		})
		t.Run(kind+"/trailers", func(t *testing.T) {
			for _, c := range cases {
				if strings.Contains(c.name, "40 KB") {
					// A trailer line longer than 4 KB is not forwarded
					// (the gateway answers 502), before this change as
					// after it; the 3 KB case covers the trailer path.
					continue
				}
				t.Run(c.name, func(t *testing.T) {
					var tok string
					got := sendAuthorizationTrailer(t, kind, func(gw *Gateway) string {
						tok, _ = mintRoot(t, gw)
						return c.value(tok)
					})
					if want := c.want(tok); strings.Join(got, "\n") != strings.Join(want, "\n") {
						t.Errorf("upstream received %d Authorization trailer values, want %d (the foreign ones only)", len(got), len(want))
					}
				})
			}
		})
	}
}

// A malformed Authorization value that carries no gateway token reaches the
// upstream byte for byte.
func TestUpstreamCredentials_MalformedForeignValueForwardedUnchanged(t *testing.T) {
	values := []string{
		`Digest realm="%GG", nonce="abc"`,
		`Digest realm="unclosed`,
		"Bearer %",
		"Basic !!!not-base64!!!",
		"Basic " + b64("\xe9t\xe9:pass"),
		`Digest , , realm=""`,
		`Digest opaque="` + strings.Repeat("%Z", 20000) + `"`,
		`Digest token="` + escapeEvery("upstream-own-api-key-0123456789") + `"`,
	}
	for _, kind := range []string{"public", "chargeback"} {
		t.Run(kind, func(t *testing.T) {
			up := newRequestRecorder(t)
			gw, _ := newTestGatewayWithUpstream(t, up.srv, kind)
			front := httptest.NewServer(gw)
			defer front.Close()
			for i, v := range values {
				got := sendAuthorization(t, front.URL, up, v).Header.Values("Authorization")
				if len(got) != 1 || got[0] != v {
					t.Errorf("value %d: foreign value changed or dropped (%d values)", i, len(got))
				}
			}
		})
	}
}

// A token inside exactly one wrapper is removed; a token the client wraps
// in two or more (double base64, double percent, base64 of percent-encoded
// text, percent-encoded Basic credentials) is not decoded and reaches the
// upstream as the client sent it, like a token in any other client data.
func TestUpstreamCredentials_OneWrapperRemovedTwoWrappersForwarded(t *testing.T) {
	for _, kind := range []string{"public", "chargeback"} {
		t.Run(kind, func(t *testing.T) {
			up := newRequestRecorder(t)
			gw, _ := newTestGatewayWithUpstream(t, up.srv, kind)
			tok, _ := mintRoot(t, gw)
			mac, err := gw.macaroonSvc.Verify(tok)
			if err != nil {
				t.Fatal(err)
			}
			bin, err := gw.macaroonSvc.EncodeBinary(mac)
			if err != nil {
				t.Fatal(err)
			}
			front := httptest.NewServer(gw)
			defer front.Close()

			oneLayer := map[string]string{
				"percent":               "Bearer " + percentFirst(tok),
				"percent binary":        "Bearer " + url.QueryEscape(bin),
				"base64 of the token":   "Bearer " + b64(tok),
				"base64url of Bearer":   base64.RawURLEncoding.EncodeToString([]byte("Bearer " + tok)),
				"Basic user:token":      "Basic " + b64("user:"+tok),
				"Basic token:":          "Basic " + b64(tok+":"),
				"quoted parameter":      `Digest token="` + tok + `"`,
				"quoted percent":        `Digest token="` + percentFirst(tok) + `"`,
				"auth-param percent":    "Digest token=" + percentFirst(tok),
				"quoted escaped":        `Digest token="` + escapeEvery(tok) + `"`,
				"Basic binary password": "Basic " + b64("user:"+bin),
			}
			for name, v := range oneLayer {
				if got := sendAuthorization(t, front.URL, up, v).Header.Values("Authorization"); len(got) != 0 {
					t.Errorf("one wrapper, %s: the token reached the upstream", name)
				}
			}

			twoLayers := map[string]string{
				"double base64":       "Basic " + b64("user:"+b64(tok)),
				"double base64, bare": "Bearer " + b64(b64(tok)),
				"double percent":      "Bearer %25" + percentFirst(tok)[1:],
				// A percent-encoded character of the token itself; a
				// binary token's base64 may have no character that
				// url.QueryEscape changes other than its "=" padding, and
				// the token without padding is a form the gateway accepts.
				"base64 of percent binary": "Basic " + b64("user:"+percentFirst(bin)),
				"base64 of percent":        "Basic " + b64("user:"+percentFirst(tok)),
				"percent of Basic":         "Basic " + percentFirst(b64("user:"+tok)),
				"percent of base64":        "Bearer " + percentFirst(b64(tok)),
			}
			for name, v := range twoLayers {
				got := sendAuthorization(t, front.URL, up, v).Header.Values("Authorization")
				if len(got) != 1 || got[0] != v {
					t.Errorf("two wrappers, %s: not forwarded as sent (%d values)", name, len(got))
				}
			}
		})
	}
}

// parserCost is what one removeGatewayCredentials call costs.
type parserCost struct {
	bytes, allocs uint64
	elapsed       time.Duration
}

// measureRemoval runs removeGatewayCredentials on a fresh copy of h several
// times and returns the allocations of one run and the shortest time.
func measureRemoval(gw *Gateway, h http.Header) parserCost {
	var best parserCost
	for i := 0; i < 7; i++ {
		c := h.Clone()
		runtime.GC()
		var before, after runtime.MemStats
		runtime.ReadMemStats(&before)
		start := time.Now()
		gw.removeGatewayCredentials(c, false)
		elapsed := time.Since(start)
		runtime.ReadMemStats(&after)
		if i == 0 || elapsed < best.elapsed {
			best.elapsed = elapsed
		}
		best.bytes = after.TotalAlloc - before.TotalAlloc
		best.allocs = after.Mallocs - before.Mallocs
	}
	return best
}

// Removing gateway tokens from Authorization costs allocations and time
// linear in the input: four times the input (5 KB, 20 KB, 80 KB of
// parameters; 1000, 4000, 16000 values or credentials) costs less than six
// times as much, whether or not a token is removed.
func TestRemoveGatewayCredentials_FourTimesInputCostsUnderSixTimes(t *testing.T) {
	gw, _ := newTestGatewayWithUpstream(t, newHeaderRecorder(t).srv, "public")
	tok, _ := mintRoot(t, gw)
	shapes := map[string]func(n int) []string{
		"many parameters": func(n int) []string {
			return []string{"Digest a=1" + strings.Repeat(", a=1", n)}
		},
		"many parameters, token last": func(n int) []string {
			return []string{"Digest a=1" + strings.Repeat(", a=1", n) + `, token="` + percentFirst(tok) + `"`}
		},
		"many quoted escaped parameters": func(n int) []string {
			return []string{"Digest a=1" + strings.Repeat(`, a="\%"`, n)}
		},
		"many credentials, token last": func(n int) []string {
			return []string{strings.Repeat("Basic Zm9v, ", n) + "Bearer " + tok}
		},
		"many values": func(n int) []string {
			v := make([]string, n)
			for i := range v {
				v[i] = "Basic foreign-fixture"
			}
			return v
		},
		"many values, token last": func(n int) []string {
			v := make([]string, n)
			for i := range v {
				v[i] = "Basic foreign-fixture"
			}
			return append(v, "Bearer "+tok)
		},
	}
	for name, shape := range shapes {
		t.Run(name, func(t *testing.T) {
			var prev parserCost
			for i, n := range []int{1000, 4000, 16000} {
				h := http.Header{"Authorization": shape(n)}
				c := measureRemoval(gw, h)
				t.Logf("n=%d input=%d bytes: allocated %d bytes in %d allocations, %v", n, len(strings.Join(h["Authorization"], "")), c.bytes, c.allocs, c.elapsed)
				if i > 0 {
					if c.bytes > 6*prev.bytes || c.allocs > 6*prev.allocs {
						t.Errorf("n=%d: allocations grew more than 6x for 4x the input (%d -> %d bytes, %d -> %d allocations)", n, prev.bytes, c.bytes, prev.allocs, c.allocs)
					}
					if c.elapsed > 6*prev.elapsed {
						t.Errorf("n=%d: time grew more than 6x for 4x the input (%v -> %v)", n, prev.elapsed, c.elapsed)
					}
				}
				prev = c
			}
		})
	}
}
