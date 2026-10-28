package proxy

import (
	"net/http"
	"strings"
	"testing"
)

// SatGate's own cookies are removed from the Cookie header; every other
// cookie is kept exactly as written.
func TestRemoveGatewayCookiesKeepsOthersAsWritten(t *testing.T) {
	cases := []struct {
		in   []string
		want []string
	}{
		{[]string{"a=1; b=2"}, []string{"a=1; b=2"}},
		{[]string{"a=1;b=2"}, []string{"a=1;b=2"}},
		{[]string{"cloud_session=x; a=1; SatGate_x=2; b=\"q\""}, []string{"a=1; b=\"q\""}},
		{[]string{"__Host-cloud_session=x"}, nil},
		{[]string{"impersonation=x", "a=1"}, []string{"a=1"}},
		{[]string{"a=1", "satgate=1; b=2"}, []string{"a=1", "b=2"}},
		{[]string{"cloud_sessions=1"}, []string{"cloud_sessions=1"}},
	}
	for _, c := range cases {
		h := http.Header{"Cookie": append([]string(nil), c.in...)}
		removeGatewayCookies(h)
		if strings.Join(h.Values("Cookie"), "|") != strings.Join(c.want, "|") {
			t.Errorf("removeGatewayCookies(%q) = %q, want %q", c.in, h.Values("Cookie"), c.want)
		}
	}
}
