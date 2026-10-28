package macaroon

import (
	"bytes"
	"encoding/base64"
	"testing"
	"time"
)

// tokenSpellings returns a JSON and a binary macaroon signed by svc, each
// in the four base64 encodings, and the bytes each one encodes.
func tokenSpellings(t testing.TB, svc *Service) (spellings map[string]string, data map[string][]byte) {
	t.Helper()
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
	spellings, data = map[string]string{}, map[string][]byte{}
	for format, d := range map[string][]byte{"json": jsonData, "binary": binData} {
		for enc, e := range map[string]*base64.Encoding{
			"standard": base64.StdEncoding, "standard raw": base64.RawStdEncoding,
			"url-safe": base64.URLEncoding, "url-safe raw": base64.RawURLEncoding,
		} {
			spellings[format+" "+enc] = e.EncodeToString(d)
			data[format+" "+enc] = d
		}
	}
	return spellings, data
}

// Every spelling of a token that Verify accepts is recognised by
// SignedByThisKey, and TokenBytes reads it as the bytes it encodes: each
// encoding, with a line feed or CR LF (which Verify skips) inserted at
// every position, padding included.
func TestSignedByThisKeyRecognisesEverySpellingVerifyAccepts(t *testing.T) {
	svc, err := NewService("token-bytes-root")
	if err != nil {
		t.Fatal(err)
	}
	spellings, data := tokenSpellings(t, svc)
	accepted := 0
	for name, s := range spellings {
		for i := 0; i <= len(s); i++ {
			for _, brk := range []string{"", "\n", "\r\n"} {
				v := s[:i] + brk + s[i:]
				if _, err := svc.Verify(v); err != nil {
					continue
				}
				accepted++
				if !svc.SignedByThisKey(v) {
					t.Errorf("%s, %q at %d: Verify accepts it, SignedByThisKey does not recognise it", name, brk, i)
				}
				if got, ok := TokenBytes(v); !ok || !bytes.Equal(got, data[name]) {
					t.Errorf("%s, %q at %d: TokenBytes does not read the token's bytes", name, brk, i)
				}
			}
		}
	}
	if accepted == 0 {
		t.Fatal("no spelling was accepted by Verify")
	}
}

// Whatever string Verify accepts, SignedByThisKey recognises.
func FuzzSignedByThisKeyRecognisesWhatVerifyAccepts(f *testing.F) {
	svc, err := NewService("token-bytes-root")
	if err != nil {
		f.Fatal(err)
	}
	spellings, _ := tokenSpellings(f, svc)
	for _, s := range spellings {
		f.Add(s)
		f.Add(s[:10] + "\r\n" + s[10:])
	}
	f.Fuzz(func(t *testing.T, s string) {
		if _, err := svc.Verify(s); err == nil && !svc.SignedByThisKey(s) {
			t.Errorf("Verify accepts a spelling SignedByThisKey does not recognise (length %d)", len(s))
		}
	})
}
