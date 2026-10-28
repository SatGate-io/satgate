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
)

// Tests of the removal boundary for gateway tokens in Authorization on
// public and chargeback routes (upstream_credentials.go): every form the
// gateway accepts, inside one wrapper; two or more wrappers forwarded like
// other client data. The whole-field rule and its cost are tested in
// upstream_whole_field_test.go. Values are fixtures, not credentials.

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
