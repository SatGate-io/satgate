package macaroon

import (
	"encoding/base64"
	"testing"
	"time"
)

// SignedByThisKey tells a token this service signed (in either format,
// expired or attenuated) from any other value, and checks no caveat.
func TestSignedByThisKey(t *testing.T) {
	svc, err := NewService("signed-by-this-key-root")
	if err != nil {
		t.Fatal(err)
	}
	other, err := NewService("a-different-root-key")
	if err != nil {
		t.Fatal(err)
	}
	live, err := svc.Mint("api:read", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	expired, err := svc.Mint("api:read", time.Now().Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	child, err := svc.Delegate(svc.Encode(live), []string{"scope = api:read:narrow"})
	if err != nil {
		t.Fatal(err)
	}
	binary, err := svc.EncodeBinary(expired)
	if err != nil {
		t.Fatal(err)
	}
	foreign, err := other.Mint("api:read", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	foreignBinary, err := other.EncodeBinary(foreign)
	if err != nil {
		t.Fatal(err)
	}
	tampered := *live
	tampered.Caveats = append(append([]string{}, live.Caveats...), "scope = api:*")

	for name, tok := range map[string]string{
		"live": svc.Encode(live), "expired": svc.Encode(expired), "delegated": svc.Encode(child),
		"binary": binary, "padded with space": "  " + svc.Encode(live) + " ",
	} {
		if !svc.SignedByThisKey(tok) {
			t.Errorf("%s: not recognised as signed by this key", name)
		}
	}
	for name, tok := range map[string]string{
		"foreign": other.Encode(foreign), "foreign binary": foreignBinary, "tampered": svc.Encode(&tampered),
		"empty": "", "api key": "upstream-own-api-key", "jwt": "eyJhbGciOiJFZERTQSJ9.eyJzdWIiOiJ4In0.c2ln",
	} {
		if svc.SignedByThisKey(tok) {
			t.Errorf("%s: recognised as signed by this key", name)
		}
	}
}

// SignedByThisKey recognises both formats in either base64 alphabet, padded
// or not, including encodings Verify does not accept.
func TestSignedByThisKeyEveryEncoding(t *testing.T) {
	svc, err := NewService("signed-by-this-key-root")
	if err != nil {
		t.Fatal(err)
	}
	m, err := svc.Mint("api:read", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	jsonData, err := base64.RawURLEncoding.DecodeString(svc.Encode(m))
	if err != nil {
		t.Fatal(err)
	}
	bin, err := svc.EncodeBinary(m)
	if err != nil {
		t.Fatal(err)
	}
	binData, err := base64.StdEncoding.DecodeString(bin)
	if err != nil {
		t.Fatal(err)
	}
	for _, enc := range []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding} {
		for format, data := range map[string][]byte{"json": jsonData, "binary": binData} {
			if !svc.SignedByThisKey(enc.EncodeToString(data)) {
				t.Errorf("%s in %v not recognised", format, enc)
			}
		}
	}
}
