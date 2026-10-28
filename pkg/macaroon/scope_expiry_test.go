package macaroon

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"
)

// HasScope: an "api:*" caveat grants only scopes in the "api:" namespace.
func TestHasScope_WildcardStaysInNamespace(t *testing.T) {
	for _, tc := range []struct {
		caveats  []string
		required string
		want     bool
	}{
		{[]string{"scope = api:*"}, "api:read", true},
		{[]string{"scope = api:*"}, "api:capability:admin", true},
		{[]string{"scope = api:*"}, "mcp:search", false},
		{[]string{"scope = api:*"}, "l402", false},
		{[]string{"scope = api:*"}, "read", false},
		{[]string{"scope = mcp:*"}, "api:read", false},
		{[]string{"scope = *"}, "read", true},
		{[]string{"scope = *", "scope = api:read"}, "api:write", false},
		{[]string{"scope = api:*", "scope = api:read"}, "api:read", true},
		{[]string{"scope = "}, "api:read", false},
		{[]string{"scope = api:read,api:write"}, "api:read", false},
		{nil, "api:read", true},
	} {
		m := &Macaroon{Caveats: tc.caveats}
		if got := m.HasScope(tc.required); got != tc.want {
			t.Errorf("caveats %q HasScope(%q) = %v, want %v", tc.caveats, tc.required, got, tc.want)
		}
	}
}

func TestVerifiedMacaroon_HasScopeWildcardStaysInNamespace(t *testing.T) {
	for _, tc := range []struct {
		scope, required string
		want            bool
	}{
		{"", "api:read", true},
		{"*", "api:read", true},
		{"api:*", "api:read", true},
		{"api:*", "mcp:search", false},
		{"api*", "apix", false},
		{"api:read", "api:read", true},
	} {
		v := &VerifiedMacaroon{Scope: tc.scope}
		if got := v.HasScope(tc.required); got != tc.want {
			t.Errorf("VerifiedMacaroon{Scope:%q}.HasScope(%q) = %v, want %v", tc.scope, tc.required, got, tc.want)
		}
	}
}

// expiresToken mints a token, adds the extra caveats and encodes it in the
// requested format.
func expiresToken(t *testing.T, svc *Service, binary bool, caveats ...string) string {
	t.Helper()
	mac, err := svc.Mint("api:read", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	mac.Caveats = append(mac.Caveats, caveats...)
	mac.Signature = svc.RecalculateSignature(mac)
	if binary {
		tok, err := svc.EncodeBinary(mac)
		if err != nil {
			t.Fatal(err)
		}
		return tok
	}
	return svc.Encode(mac)
}

// Verify refuses an expires caveat that is not an absolute time, in both
// token formats, and enforces absolute times in both accepted forms.
func TestVerify_ExpiresMustBeAbsolute(t *testing.T) {
	svc, err := NewService("expires-test-secret")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	ms := func(d time.Duration) string { return strconv.FormatInt(now.Add(d).UnixMilli(), 10) }
	rfc := func(d time.Duration) string { return now.Add(d).UTC().Format(time.RFC3339) }
	const (
		accept  = "accept"
		expired = "expired"
		refuse  = "not absolute"
	)
	for _, tc := range []struct {
		value string
		want  string
	}{
		{rfc(time.Hour), accept},
		{ms(time.Hour), accept},
		{rfc(-time.Hour), expired},
		{ms(-time.Hour), expired},
		{"0", expired},
		{"24h", refuse},
		{"5m", refuse},
		{"30m", refuse},
		{"-1h", refuse},
		{"", refuse},
		{"expired", refuse},
		{"12.5", refuse},
		{"-5", refuse},
		{"2026-11-01", refuse},
		{"2026-11-01T00:00:00", refuse},
	} {
		for _, binary := range []bool{false, true} {
			name := fmt.Sprintf("expires=%q binary=%v", tc.value, binary)
			tok := expiresToken(t, svc, binary, "expires = "+tc.value)
			_, err := svc.Verify(tok)
			switch tc.want {
			case accept:
				if err != nil {
					t.Errorf("%s: want accepted, got %v", name, err)
				}
			case expired:
				if err == nil || !strings.Contains(err.Error(), "token expired") {
					t.Errorf("%s: want token expired, got %v", name, err)
				}
			case refuse:
				if err == nil || !strings.Contains(err.Error(), "must be an absolute time") {
					t.Errorf("%s: want an absolute-time refusal, got %v", name, err)
				}
			}
		}
	}
}

// A "time < N" caveat must be decimal Unix seconds, nothing trailing.
func TestVerify_TimeBeforeIsStrict(t *testing.T) {
	svc, err := NewService("expires-test-secret")
	if err != nil {
		t.Fatal(err)
	}
	future := strconv.FormatInt(time.Now().Add(time.Hour).Unix(), 10)
	past := strconv.FormatInt(time.Now().Add(-time.Hour).Unix(), 10)
	if _, err := svc.Verify(expiresToken(t, svc, false, "time < "+future)); err != nil {
		t.Errorf("future time < caveat refused: %v", err)
	}
	if _, err := svc.Verify(expiresToken(t, svc, false, "time < "+past)); err == nil || !strings.Contains(err.Error(), "token expired") {
		t.Errorf("past time < caveat: want token expired, got %v", err)
	}
	for _, v := range []string{future + "abc", future + "h", "-" + future, "soon", ""} {
		if _, err := svc.Verify(expiresToken(t, svc, false, "time < "+v)); err == nil {
			t.Errorf("time < %q accepted", v)
		}
	}
}

func TestGetExpiry_ReadsOnlyWhatVerifyAccepts(t *testing.T) {
	soon := time.Now().Add(10 * time.Minute).Truncate(time.Second)
	later := time.Now().Add(time.Hour).Truncate(time.Second)
	for _, tc := range []struct {
		name    string
		caveats []string
		want    time.Time
		ok      bool
	}{
		{"none", []string{"scope = api:read"}, time.Time{}, false},
		{"unix ms", []string{"expires = " + strconv.FormatInt(later.UnixMilli(), 10)}, later, true},
		{"rfc3339", []string{"expires = " + later.UTC().Format(time.RFC3339)}, later, true},
		{"time <", []string{"time < " + strconv.FormatInt(later.Unix(), 10)}, later, true},
		{"earliest wins", []string{
			"expires = " + strconv.FormatInt(later.UnixMilli(), 10),
			"expires = " + soon.UTC().Format(time.RFC3339),
		}, soon, true},
		{"duration", []string{"expires = 5m"}, time.Time{}, false},
		{"duration after absolute", []string{
			"expires = " + strconv.FormatInt(later.UnixMilli(), 10),
			"expires = 24h",
		}, time.Time{}, false},
		{"malformed", []string{"expires = 12abc"}, time.Time{}, false},
	} {
		got, ok := (&Macaroon{Caveats: tc.caveats}).GetExpiry()
		if ok != tc.ok || !got.Equal(tc.want) {
			t.Errorf("%s: GetExpiry() = %v, %v; want %v, %v", tc.name, got, ok, tc.want, tc.ok)
		}
	}
}

// Delegate refuses to add an expires caveat Verify would refuse.
func TestDelegate_RefusesNonAbsoluteExpires(t *testing.T) {
	svc, err := NewService("expires-test-secret")
	if err != nil {
		t.Fatal(err)
	}
	root, err := svc.Mint("api:*", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	parent := svc.Encode(root)
	for _, v := range []string{"24h", "5m", "soon", ""} {
		if _, err := svc.Delegate(parent, []string{"expires = " + v}); err == nil || !strings.Contains(err.Error(), "must be an absolute time") {
			t.Errorf("Delegate with expires = %q: want an absolute-time refusal, got %v", v, err)
		}
	}
	child, err := svc.Delegate(parent, []string{"expires = " + time.Now().Add(time.Minute).UTC().Format(time.RFC3339)})
	if err != nil {
		t.Fatalf("Delegate with an absolute expiry: %v", err)
	}
	if _, err := svc.Verify(svc.Encode(child)); err != nil {
		t.Fatalf("child with an absolute expiry does not verify: %v", err)
	}
}

// Every minting path in this package writes an absolute expiry.
func TestMint_WritesAbsoluteExpiry(t *testing.T) {
	svc, err := NewService("expires-test-secret")
	if err != nil {
		t.Fatal(err)
	}
	at := time.Now().Add(time.Hour)
	m1, err := svc.Mint("api:read", at)
	if err != nil {
		t.Fatal(err)
	}
	m2, err := svc.MintL402(L402Binding{PaymentHash: "ph", Route: "r", PriceSats: 1}, at)
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range []*Macaroon{m1, m2} {
		got, ok := m.GetExpiry()
		if !ok || got.UnixMilli() != at.UnixMilli() {
			t.Errorf("minted expiry = %v, %v; want %v", got, ok, at)
		}
	}
}
