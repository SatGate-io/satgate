package proxy

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

// gatewayExpirySpellings is the date table of pkg/macaroon
// TestParseExpires_RFC3339Spelling (every accepted spelling is in the
// future).
func gatewayExpirySpellings() []struct {
	value string
	ok    bool
} {
	return []struct {
		value string
		ok    bool
	}{
		{"2099-01-01T00:00:00Z", true},
		{"2099-01-01T00:00:00.5Z", true},
		{"2099-01-01T00:00:00.123456789Z", true},
		{"2099-01-01T01:30:00+01:30", true},
		{"2099-01-01T00:00:00-05:00", true},
		{"2099-12-31T23:59:59+23:59", true},
		{"2096-02-29T00:00:00Z", true},
		{" 2099-01-01T00:00:00Z\t", true},
		{"4102444800000", true},
		{"2099-01-01T0:00:00Z", false},
		{"2099-01-01T00:0:00Z", false},
		{"2099-01-01T00:00:0Z", false},
		{"2099-1-01T00:00:00Z", false},
		{"2099-01-1T00:00:00Z", false},
		{"20990-01-01T00:00:00Z", false},
		{"+2099-01-01T00:00:00Z", false},
		{"2099-01-01T00:00:00,5Z", false},
		{"2099-01-01T00:00:00.Z", false},
		{"2099-01-01T00:00:00.5.5Z", false},
		{"2099-01-01T00:00:00+24:00", false},
		{"2099-01-01T00:00:00+00:60", false},
		{"2099-01-01T00:00:00+0100", false},
		{"2099-01-01T00:00:00+01", false},
		{"2099-01-01T00:00:00", false},
		{"2099-01-01t00:00:00Z", false},
		{"2099-01-01T00:00:00z", false},
		{"2099-01-01 00:00:00Z", false},
		{"2099-01-01T24:00:00Z", false},
		{"2099-01-01T00:60:00Z", false},
		{"2099-01-01T00:00:60Z", false},
		{"2099-13-01T00:00:00Z", false},
		{"2099-00-01T00:00:00Z", false},
		{"2099-01-00T00:00:00Z", false},
		{"2099-01-32T00:00:00Z", false},
		{"2099-02-30T00:00:00Z", false},
		{"2097-02-29T00:00:00Z", false},
		{"2099-01-01T00:00:00ZZ", false},
		{"2099-01-01T00:00:00Z junk", false},
		{"2099-01-01T00:00Z", false},
		{"2099-01-01", false},
		{"Thu, 01 Jan 2099 00:00:00 GMT", false},
		{"24h", false},
	}
}

// D4 on the HTTP gateway: a capability route, the capability admin endpoint,
// /api/capability/validate and /api/capability/delegate read an expires
// caveat with macaroon.ParseExpires, JSON and binary tokens.
func TestGateway_ExpiresRFC3339Spelling(t *testing.T) {
	for _, tc := range gatewayExpirySpellings() {
		for _, binary := range []bool{false, true} {
			t.Run(fmt.Sprintf("%q/binary=%v", tc.value, binary), func(t *testing.T) {
				// A gateway per case: the admin endpoint rate-limits
				// refused requests from one client.
				gw := scopeExpiryGateway(t)
				parent := scopedToken(t, gw, "scope = api:*")
				mac, err := gw.macaroonSvc.Mint("api:*", time.Now().Add(time.Hour))
				if err != nil {
					t.Fatal(err)
				}
				mac.Caveats = append(mac.Caveats, "expires = "+tc.value)
				mac.Signature = gw.macaroonSvc.RecalculateSignature(mac)
				tok := gw.macaroonSvc.Encode(mac)
				if binary {
					if tok, err = gw.macaroonSvc.EncodeBinary(mac); err != nil {
						t.Fatal(err)
					}
				}
				want := http.StatusUnauthorized
				if tc.ok {
					want = http.StatusOK
				}
				for _, path := range []string{"/api/x", "/api/capability/admin"} {
					if rec := bearerGet(t, gw, path, tok); rec.Code != want {
						t.Errorf("GET %s status %d, want %d", path, rec.Code, want)
					}
				}
				if rec := postJSON(t, gw, "/api/capability/validate", map[string]any{"token": tok}, false); (rec.Code == http.StatusOK) != tc.ok {
					t.Errorf("validate status %d, want accepted=%v", rec.Code, tc.ok)
				}
				if !binary {
					if rec := postJSON(t, gw, "/api/capability/delegate", map[string]any{"parentToken": tok, "caveats": []string{"label = child"}}, false); (rec.Code == http.StatusOK) != tc.ok {
						t.Errorf("delegate from this parent status %d, want accepted=%v", rec.Code, tc.ok)
					}
					if rec := postJSON(t, gw, "/api/capability/delegate", map[string]any{"parentToken": parent, "caveats": []string{"expires = " + tc.value}}, false); (rec.Code == http.StatusOK) != tc.ok {
						t.Errorf("delegate adding this expiry status %d, want accepted=%v", rec.Code, tc.ok)
					}
				}
			})
		}
	}
}

// D4 on L402: a paid proof carrying an expires caveat in a spelling outside
// RFC 3339 gets a fresh challenge, JSON and binary.
func TestGateway_L402ExpiresRFC3339Spelling(t *testing.T) {
	for _, tc := range gatewayExpirySpellings() {
		for _, binary := range []bool{false, true} {
			t.Run(fmt.Sprintf("%q/binary=%v", tc.value, binary), func(t *testing.T) {
				gw := newTestGateway(t, newTwoTierL402Config(okUpstream(t).URL))
				paid := payL402At(t, gw, "/cheap/x")
				mac, err := gw.macaroonSvc.DecodeBinary(paid.macaroon)
				if err != nil {
					t.Fatal(err)
				}
				mac.Caveats = append(mac.Caveats, "expires = "+tc.value)
				mac.Signature = gw.macaroonSvc.RecalculateSignature(mac)
				tok := gw.macaroonSvc.Encode(mac)
				if binary {
					if tok, err = gw.macaroonSvc.EncodeBinary(mac); err != nil {
						t.Fatal(err)
					}
				}
				rec := useL402Method(t, gw, "GET", "/cheap/x", "L402 "+tok+":"+paid.preimage)
				if tc.ok {
					if rec.Code != http.StatusOK {
						t.Errorf("proof status %d, want 200", rec.Code)
					}
					return
				}
				wantFreshChallenge(t, rec, "proof with expires "+tc.value)
			})
		}
	}
}
