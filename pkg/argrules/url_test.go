package argrules

import (
	"strings"
	"testing"
)

func urlRule(t *testing.T, cond string) []Rule {
	t.Helper()
	return rulesFor(t, `{"v":1,"rules":[{"tool":"t","shapes":[{"url":{"url":`+cond+`}}]}]}`)
}

func urlPasses(rules []Rule, u string) bool {
	b, _ := jsonMarshalString(u)
	return Check(rules, "t", call("t", `{"url":`+b+`}`)) == nil
}

func TestURLTable(t *testing.T) {
	base := urlRule(t, `{"hosts":["www.wixapis.com"]}`)
	suffix := urlRule(t, `{"hosts":["www.wixapis.com"],"path_suffixes":["/query"]}`)
	prefix := urlRule(t, `{"hosts":["www.wixapis.com"],"path_prefixes":["/stores/v1"]}`)
	both := urlRule(t, `{"hosts":["www.wixapis.com"],"path_prefixes":["/stores"],"path_suffixes":["/query"]}`)
	ports := urlRule(t, `{"hosts":["www.wixapis.com"],"ports":[8443]}`)
	multi := urlRule(t, `{"hosts":["www.wixapis.com","wixapis.com"]}`)

	type tc struct {
		name string
		rule []Rule
		url  string
		pass bool
	}
	cases := []tc{
		// scheme
		{"https", base, "https://www.wixapis.com/x", true},
		{"https upper scheme", base, "HTTPS://www.wixapis.com/x", true},
		{"http", base, "http://www.wixapis.com/x", false},
		{"relative", base, "/x", false},
		{"scheme relative", base, "//www.wixapis.com/x", false},
		{"no scheme", base, "www.wixapis.com/x", false},
		{"ftp", base, "ftp://www.wixapis.com/x", false},
		{"javascript", base, "javascript:alert(1)", false},
		{"empty", base, "", false},
		{"spaces", base, " https://www.wixapis.com/x", false},
		{"trailing space", base, "https://www.wixapis.com/x ", false},
		{"newline", base, "https://www.wixapis.com/x\n", false},
		{"tab in host", base, "https://www.wixapis.com\t/x", false},
		// host case and shape
		{"host upper", base, "https://WWW.WIXAPIS.COM/x", true},
		{"host mixed", base, "https://Www.WixApis.com/x", true},
		{"no path", base, "https://www.wixapis.com", true},
		{"root path", base, "https://www.wixapis.com/", true},
		{"query only", base, "https://www.wixapis.com?x=1", true},
		{"other host", base, "https://evil.example/x", false},
		{"suffix host", base, "https://www.wixapis.com.evil.example/x", false},
		{"prefix host", base, "https://evilwww.wixapis.com/x", false},
		{"subdomain", base, "https://a.www.wixapis.com/x", false},
		{"parent domain", base, "https://wixapis.com/x", false},
		{"multi host b", multi, "https://wixapis.com/x", true},
		{"trailing dot", base, "https://www.wixapis.com./x", false},
		{"double dot host", base, "https://www..wixapis.com/x", false},
		{"leading dot host", base, "https://.www.wixapis.com/x", false},
		{"empty host", base, "https:///x", false},
		// userinfo, ip, port
		{"userinfo", base, "https://user@www.wixapis.com/x", false},
		{"userinfo pass", base, "https://user:pw@www.wixapis.com/x", false},
		{"userinfo trick", base, "https://www.wixapis.com@evil.example/x", false},
		{"userinfo trick 2", base, "https://evil.example@www.wixapis.com/x", false},
		{"ipv4", urlRule(t, `{"hosts":["example.com"]}`), "https://93.184.216.34/x", false},
		{"ipv4 decimal", base, "https://2130706433/x", false},
		{"ipv4 hex", base, "https://0x7f000001/x", false},
		{"ipv6", base, "https://[::1]/x", false},
		{"ipv6 port", base, "https://[::1]:443/x", false},
		{"port 443", base, "https://www.wixapis.com:443/x", false},
		{"port 8443 unlisted", base, "https://www.wixapis.com:8443/x", false},
		{"port 8443 listed", ports, "https://www.wixapis.com:8443/x", true},
		{"port 443 not listed even with others", ports, "https://www.wixapis.com:443/x", false},
		{"no port when ports listed", ports, "https://www.wixapis.com/x", true},
		{"port 8444", ports, "https://www.wixapis.com:8444/x", false},
		{"port empty", base, "https://www.wixapis.com:/x", false},
		{"port leading zero", ports, "https://www.wixapis.com:08443/x", false},
		{"port letters", ports, "https://www.wixapis.com:84a3/x", false},
		{"port huge", ports, "https://www.wixapis.com:99999999/x", false},
		{"port twice", ports, "https://www.wixapis.com:8443:8443/x", false},
		{"port plus", ports, "https://www.wixapis.com:+8443/x", false},
		// IDN and punycode
		{"idn host", base, "https://www.wixapis.cöm/x", false},
		{"idn lookalike", base, "https://www.wіxapis.com/x", false}, // Cyrillic i
		{"punycode other", base, "https://xn--wixapis-9ya.com/x", false},
		{"fullwidth dot", base, "https://www.wixapis。com/x", false},
		{"percent host", base, "https://www.wixapis%2Ecom/x", false},
		{"percent host 2", base, "https://www.wix%61pis.com/x", false},
		{"punycode rule matches punycode", urlRule(t, `{"hosts":["xn--bcher-kva.example"]}`), "https://xn--bcher-kva.example/x", true},
		{"unicode never matches punycode rule", urlRule(t, `{"hosts":["xn--bcher-kva.example"]}`), "https://bücher.example/x", false},
		// suffix
		{"suffix ok", suffix, "https://www.wixapis.com/stores/v1/products/query", true},
		{"suffix with query string", suffix, "https://www.wixapis.com/stores/v1/products/query?x=1", true},
		{"suffix with fragment", suffix, "https://www.wixapis.com/stores/v1/products/query#frag", true},
		{"suffix at root of path", suffix, "https://www.wixapis.com/query", true},
		{"suffix wrong", suffix, "https://www.wixapis.com/stores/v1/products/delete", false},
		{"suffix partial segment", suffix, "https://www.wixapis.com/stores/v1/products/myquery", false},
		{"suffix extra", suffix, "https://www.wixapis.com/stores/v1/products/query/more", false},
		{"suffix query in query string only", suffix, "https://www.wixapis.com/stores/v1/delete?x=/query", false},
		{"suffix in fragment only", suffix, "https://www.wixapis.com/stores/v1/delete#/query", false},
		{"suffix case sensitive", suffix, "https://www.wixapis.com/stores/v1/products/QUERY", false},
		{"suffix trailing slash", suffix, "https://www.wixapis.com/stores/v1/products/query/", false},
		{"suffix no path", suffix, "https://www.wixapis.com", false},
		// prefix
		{"prefix ok", prefix, "https://www.wixapis.com/stores/v1/products", true},
		{"prefix exact", prefix, "https://www.wixapis.com/stores/v1", true},
		{"prefix partial segment", prefix, "https://www.wixapis.com/stores/v10/products", false},
		{"prefix partial 2", prefix, "https://www.wixapis.com/stores/v1x", false},
		{"prefix other", prefix, "https://www.wixapis.com/stores/v2/products", false},
		{"prefix root", prefix, "https://www.wixapis.com/", false},
		{"both ok", both, "https://www.wixapis.com/stores/v1/query", true},
		{"both prefix only", both, "https://www.wixapis.com/stores/v1/delete", false},
		{"both suffix only", both, "https://www.wixapis.com/other/query", false},
		// dot, empty segments
		{"dot segment", suffix, "https://www.wixapis.com/a/./query", false},
		{"dotdot segment", suffix, "https://www.wixapis.com/a/../query", false},
		{"dotdot to escape prefix", prefix, "https://www.wixapis.com/stores/v1/../../admin", false},
		{"dotdot at end", prefix, "https://www.wixapis.com/stores/v1/..", false},
		{"dot at end", prefix, "https://www.wixapis.com/stores/v1/.", false},
		{"empty segment", suffix, "https://www.wixapis.com/a//query", false},
		{"double slash start", suffix, "https://www.wixapis.com//query", false},
		{"trailing slash base", base, "https://www.wixapis.com/a/", false},
		{"three dots ok", suffix, "https://www.wixapis.com/a/.../query", true},
		{"dot inside segment ok", suffix, "https://www.wixapis.com/a.b/query", true},
		// backslash
		{"backslash path", suffix, `https://www.wixapis.com/a\query`, false},
		{"backslash before path", base, `https://www.wixapis.com\x`, false},
		{"backslash in host", base, `https://www.wixapis.com\@evil.example/x`, false},
		{"backslash query", base, `https://www.wixapis.com/x?a=\b`, false},
		// percent encodings
		{"%2F", suffix, "https://www.wixapis.com/a%2Fquery", false},
		{"%2f", suffix, "https://www.wixapis.com/a%2fquery", false},
		{"%2F segment", suffix, "https://www.wixapis.com/a/b%2F/query", false},
		{"%2E", suffix, "https://www.wixapis.com/a/%2E%2E/query", false},
		{"%2e", suffix, "https://www.wixapis.com/a/%2e/query", false},
		{"%2e%2e", suffix, "https://www.wixapis.com/a/%2e%2e/query", false},
		{"%5C", suffix, "https://www.wixapis.com/a%5Cquery", false},
		{"%5c", suffix, "https://www.wixapis.com/a%5cquery", false},
		{"%25", suffix, "https://www.wixapis.com/a%252Fquery", false},
		{"%25 alone", suffix, "https://www.wixapis.com/100%25/query", false},
		{"bare percent", suffix, "https://www.wixapis.com/a%/query", false},
		{"short percent", suffix, "https://www.wixapis.com/a%2", false},
		{"bad hex", suffix, "https://www.wixapis.com/a%zz/query", false},
		{"%20 ok", suffix, "https://www.wixapis.com/a%20b/query", true},
		{"%41 not decoded", prefix, "https://www.wixapis.com/stores/v%31/products", true}, // %31 is "1", kept as written: "v%31" != "v1"
		{"%2F in query ok", suffix, "https://www.wixapis.com/a/query?next=%2Fdelete", true},
		{"%5C in query ok", suffix, "https://www.wixapis.com/a/query?x=%5C", true},
		{"%2F in fragment ok", suffix, "https://www.wixapis.com/a/query#%2F", true},
		// query and fragment are allowed and unmatched
		{"query", suffix, "https://www.wixapis.com/a/query?x=1&y=2", true},
		{"empty query", suffix, "https://www.wixapis.com/a/query?", true},
		{"fragment", suffix, "https://www.wixapis.com/a/query#x", true},
		{"query then fragment", suffix, "https://www.wixapis.com/a/query?x=1#y", true},
		{"query with slashes", base, "https://www.wixapis.com?/../x", true},
		// control and non-ascii
		{"nul", base, "https://www.wixapis.com/\x00", false},
		{"del", base, "https://www.wixapis.com/\x7f", false},
		{"non-ascii path", suffix, "https://www.wixapis.com/é/query", false},
		{"non-ascii query ok", suffix, "https://www.wixapis.com/a/query?q=é", true},
		{"too long", base, "https://www.wixapis.com/" + strings.Repeat("a", 3000), false},
	}
	for _, c := range cases {
		got := urlPasses(c.rule, c.url)
		want := c.pass
		if c.name == "%41 not decoded" {
			// "v%31" is a different segment from "v1": the prefix does not match.
			want = false
		}
		if got != want {
			t.Errorf("%s: %q pass=%v, want %v", c.name, c.url, got, want)
		}
	}
}

func TestURLFieldTypes(t *testing.T) {
	r := urlRule(t, `{"hosts":["www.wixapis.com"]}`)
	for _, v := range []string{`null`, `1`, `true`, `["https://www.wixapis.com/x"]`, `{"u":"https://www.wixapis.com/x"}`} {
		if Check(r, "t", call("t", `{"url":`+v+`}`)) == nil {
			t.Errorf("non-string url %s passed", v)
		}
	}
	if Check(r, "t", call("t", `{}`)) == nil {
		t.Error("missing url passed")
	}
	// A JSON escape that spells a backslash or a control byte is judged on the decoded string.
	if Check(r, "t", call("t", `{"url":"https://www.wixapis.com/\u005cx"}`)) == nil {
		t.Error("escaped backslash passed")
	}
	if Check(r, "t", call("t", `{"url":"https://www.wixapis.com/\u0000"}`)) == nil {
		t.Error("escaped NUL passed")
	}
	if Check(r, "t", call("t", `{"url":"https://www.wixapis.com/\u002e\u002e/x"}`)) == nil {
		t.Error("escaped dot-dot passed")
	}
}
