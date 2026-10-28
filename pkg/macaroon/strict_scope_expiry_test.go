package macaroon

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

type expirySpelling struct {
	value string
	want  time.Time // zero: refused
}

func utc(s string) time.Time {
	ts, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		panic(err)
	}
	return ts
}

// expirySpellings is the table of RFC 3339 spellings an expires caveat may
// and may not use. Every accepted value is in the future, so a refusal can
// only come from the spelling. The same table is used on the HTTP gateway
// (pkg/proxy) and on MCP (pkg/mcpserver).
func expirySpellings() []expirySpelling {
	return []expirySpelling{
		// Accepted: the RFC 3339 date-time grammar, upper-case T and Z.
		{"2099-01-01T00:00:00Z", utc("2099-01-01T00:00:00Z")},
		{"2099-01-01T00:00:00.5Z", utc("2099-01-01T00:00:00.5Z")},
		{"2099-01-01T00:00:00.123456789Z", utc("2099-01-01T00:00:00.123456789Z")},
		{"2099-01-01T01:30:00+01:30", utc("2099-01-01T00:00:00Z")},
		{"2099-01-01T00:00:00-05:00", utc("2099-01-01T05:00:00Z")},
		{"2099-12-31T23:59:59+23:59", utc("2099-12-31T00:00:59Z")},
		{"2096-02-29T00:00:00Z", utc("2096-02-29T00:00:00Z")},
		{" 2099-01-01T00:00:00Z\t", utc("2099-01-01T00:00:00Z")},
		// Unix milliseconds: unchanged.
		{"4102444800000", utc("2100-01-01T00:00:00Z")},

		// Refused: a field with the wrong number of digits.
		{"2099-01-01T0:00:00Z", time.Time{}},
		{"2099-01-01T00:0:00Z", time.Time{}},
		{"2099-01-01T00:00:0Z", time.Time{}},
		{"2099-1-01T00:00:00Z", time.Time{}},
		{"2099-01-1T00:00:00Z", time.Time{}},
		{"20990-01-01T00:00:00Z", time.Time{}},
		{"+2099-01-01T00:00:00Z", time.Time{}},
		// Fraction: only "." and one or more digits.
		{"2099-01-01T00:00:00,5Z", time.Time{}},
		{"2099-01-01T00:00:00.Z", time.Time{}},
		{"2099-01-01T00:00:00.5.5Z", time.Time{}},
		// Offset: +hh:mm or -hh:mm in range.
		{"2099-01-01T00:00:00+24:00", time.Time{}},
		{"2099-01-01T00:00:00+00:60", time.Time{}},
		{"2099-01-01T00:00:00+0100", time.Time{}},
		{"2099-01-01T00:00:00+01", time.Time{}},
		{"2099-01-01T00:00:00", time.Time{}},
		// Upper-case T and Z only; no space for T.
		{"2099-01-01t00:00:00Z", time.Time{}},
		{"2099-01-01T00:00:00z", time.Time{}},
		{"2099-01-01 00:00:00Z", time.Time{}},
		// Fields out of range.
		{"2099-01-01T24:00:00Z", time.Time{}},
		{"2099-01-01T00:60:00Z", time.Time{}},
		{"2099-01-01T00:00:60Z", time.Time{}},
		{"2099-13-01T00:00:00Z", time.Time{}},
		{"2099-00-01T00:00:00Z", time.Time{}},
		{"2099-01-00T00:00:00Z", time.Time{}},
		{"2099-01-32T00:00:00Z", time.Time{}},
		{"2099-02-30T00:00:00Z", time.Time{}},
		{"2097-02-29T00:00:00Z", time.Time{}},
		// Anything else.
		{"2099-01-01T00:00:00ZZ", time.Time{}},
		{"2099-01-01T00:00:00Z junk", time.Time{}},
		{"2099-01-01T00:00Z", time.Time{}},
		{"2099-01-01", time.Time{}},
		{"Thu, 01 Jan 2099 00:00:00 GMT", time.Time{}},
		{"24h", time.Time{}},
	}
}

// ParseExpires accepts an RFC 3339 value only in the exact RFC 3339 spelling.
func TestParseExpires_RFC3339Spelling(t *testing.T) {
	for _, tc := range expirySpellings() {
		got, err := ParseExpires(tc.value)
		if tc.want.IsZero() {
			if err == nil || !errors.Is(err, ErrExpiresNotAbsolute) {
				t.Errorf("ParseExpires(%q) = %v, %v; want ErrExpiresNotAbsolute", tc.value, got, err)
			}
			continue
		}
		if err != nil || !got.Equal(tc.want) {
			t.Errorf("ParseExpires(%q) = %v, %v; want %v", tc.value, got, err, tc.want)
		}
	}
}

// Every reader in this package goes through ParseExpires: Verify (JSON and
// binary), GetExpiry, Delegate (an added caveat and a parent carrying it) and
// CheckL402Request.
func TestExpiresRFC3339Spelling_EveryVerifier(t *testing.T) {
	svc, err := NewService("strict-expiry-test-secret")
	if err != nil {
		t.Fatal(err)
	}
	parent := expiresToken(t, svc, false)
	for _, tc := range expirySpellings() {
		ok := !tc.want.IsZero()
		for _, binary := range []bool{false, true} {
			t.Run(fmt.Sprintf("%q/binary=%v", tc.value, binary), func(t *testing.T) {
				tok := expiresToken(t, svc, binary, "expires = "+tc.value)
				mac, verr := svc.Verify(tok)
				if (verr == nil) != ok {
					t.Errorf("Verify accepted=%v, want %v (%v)", verr == nil, ok, verr)
				}
				if !ok && verr != nil && !errors.Is(verr, ErrExpiresNotAbsolute) {
					t.Errorf("Verify error %v does not wrap ErrExpiresNotAbsolute", verr)
				}
				decoded, derr := svc.Decode(tok)
				if binary {
					decoded, derr = svc.DecodeBinary(tok)
				}
				if derr != nil {
					t.Fatal(derr)
				}
				if exp, got := decoded.GetExpiry(); got != ok {
					t.Errorf("GetExpiry ok=%v, want %v", got, ok)
				} else if ok && exp.After(tc.want) {
					t.Errorf("GetExpiry = %v, want at most %v", exp, tc.want)
				}
				if !binary {
					if _, err := svc.Delegate(parent, []string{"expires = " + tc.value}); (err == nil) != ok {
						t.Errorf("Delegate adding the caveat accepted=%v, want %v", err == nil, ok)
					}
					if _, err := svc.Delegate(tok, nil); (err == nil) != ok {
						t.Errorf("Delegate of a parent carrying it accepted=%v, want %v", err == nil, ok)
					}
				}
				if ok && mac == nil {
					t.Fatal("Verify returned no macaroon")
				}
			})
		}
		t.Run(fmt.Sprintf("%q/L402", tc.value), func(t *testing.T) {
			mac, err := svc.MintL402(L402Binding{PaymentHash: "ph", Route: "r", PriceSats: 1}, time.Now().Add(time.Hour))
			if err != nil {
				t.Fatal(err)
			}
			mac.Caveats = append(mac.Caveats, "expires = "+tc.value)
			mac.Signature = svc.RecalculateSignature(mac)
			if err := mac.CheckL402Request(L402Request{Method: "GET", Path: "/x", Route: "r"}); (err == nil) != ok {
				t.Errorf("CheckL402Request accepted=%v, want %v (%v)", err == nil, ok, err)
			}
		})
	}
}
