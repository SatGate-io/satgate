package proxy

import (
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/satgate-io/satgate/pkg/config"
	"github.com/satgate-io/satgate/pkg/governance"
	"github.com/satgate-io/satgate/pkg/lightning"
)

func scopeExpiryGateway(t *testing.T) *Gateway {
	t.Helper()
	up, _ := countingUpstream(t)
	gw, err := New(Options{
		Config: &config.Config{
			Server:    config.ServerConfig{Listen: ":8080"},
			Admin:     config.AdminConfig{Token: "admin-secret"},
			Cloud:     &config.CloudConfig{SSRF: &config.CloudSSRFConfig{AllowPrivateIPs: true}},
			Upstreams: map[string]config.Upstream{"backend": {URL: up.URL, Timeout: 10 * time.Second}},
			Routes: []config.Route{
				{Name: "api-read", Match: config.RouteMatch{PathPrefix: "/api/"}, Upstream: "backend",
					Policy: config.RoutePolicy{Kind: "capability", Scope: "api:read"}},
				{Name: "plain-read", Match: config.RouteMatch{PathPrefix: "/plain/"}, Upstream: "backend",
					Policy: config.RoutePolicy{Kind: "capability", Scope: "read"}},
				{Name: "tools", Match: config.RouteMatch{PathPrefix: "/tools/"}, Upstream: "backend",
					Policy: config.RoutePolicy{Kind: "capability", Scope: "mcp:search"}},
				{Name: "open", Match: config.RouteMatch{PathPrefix: "/open/"}, Upstream: "backend",
					Policy: config.RoutePolicy{Kind: "capability"}},
			},
		},
		Macaroon:   newTestMacaroonService(t),
		Governance: governance.NewService(nil),
		Lightning:  lightning.NewMockProvider(),
	})
	if err != nil {
		t.Fatalf("create gateway: %v", err)
	}
	return gw
}

func scopedToken(t *testing.T, gw *Gateway, caveats ...string) string {
	t.Helper()
	mac, err := gw.macaroonSvc.Mint("", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	mac.Caveats = append(mac.Caveats, caveats...)
	mac.Signature = gw.macaroonSvc.RecalculateSignature(mac)
	return gw.macaroonSvc.Encode(mac)
}

// HTTP gateway: a route's required scope is granted by the same scope, by a
// wildcard of its own namespace, or by "*".
func TestGateway_ScopeWildcardStaysInNamespace(t *testing.T) {
	gw := scopeExpiryGateway(t)
	for _, tc := range []struct {
		scope string
		path  string
		want  int
	}{
		{"api:read", "/api/x", http.StatusOK},
		{"api:*", "/api/x", http.StatusOK},
		{"*", "/api/x", http.StatusOK},
		{"mcp:*", "/api/x", http.StatusForbidden},
		{"api:write", "/api/x", http.StatusForbidden},
		{"api:read:*", "/api/x", http.StatusForbidden},
		{"api:*", "/plain/x", http.StatusForbidden},
		{"read", "/plain/x", http.StatusOK},
		{"*", "/plain/x", http.StatusOK},
		{"api:*", "/tools/x", http.StatusForbidden},
		{"mcp:*", "/tools/x", http.StatusOK},
		{"mcp:search", "/tools/x", http.StatusOK},
		{"api:*", "/open/x", http.StatusOK},
	} {
		tok := scopedToken(t, gw, "scope = "+tc.scope)
		if rec := bearerGet(t, gw, tc.path, tok); rec.Code != tc.want {
			t.Errorf("scope %q on %s: status %d, want %d", tc.scope, tc.path, rec.Code, tc.want)
		}
	}
}

// The capability admin endpoint needs "api:capability:admin" from every scope
// caveat, matched like every other scope.
func TestGateway_CapabilityAdminScope(t *testing.T) {
	gw := scopeExpiryGateway(t)
	for _, tc := range []struct {
		caveats []string
		want    int
	}{
		{[]string{"scope = api:capability:admin"}, http.StatusOK},
		{[]string{"scope = api:capability:*"}, http.StatusOK},
		{[]string{"scope = api:*"}, http.StatusOK},
		{[]string{"scope = *"}, http.StatusOK},
		{[]string{"scope = mcp:*"}, http.StatusForbidden},
		{[]string{"scope = api:read"}, http.StatusForbidden},
		{[]string{"scope = api:*", "scope = api:read"}, http.StatusForbidden},
		{[]string{"scope = api:*", " scope = mcp:*"}, http.StatusForbidden},
		{[]string{"scope = "}, http.StatusForbidden},
		{nil, http.StatusForbidden},
	} {
		tok := scopedToken(t, gw, tc.caveats...)
		if rec := bearerGet(t, gw, "/api/capability/admin", tok); rec.Code != tc.want {
			t.Errorf("caveats %q: admin status %d, want %d", tc.caveats, rec.Code, tc.want)
		}
	}
}

// HTTP gateway: every expires caveat must be an absolute time in the future.
func TestGateway_ExpiresMustBeAbsolute(t *testing.T) {
	gw := scopeExpiryGateway(t)
	now := time.Now()
	for _, tc := range []struct {
		value string
		want  int
	}{
		{now.Add(time.Hour).UTC().Format(time.RFC3339), http.StatusOK},
		{strconv.FormatInt(now.Add(time.Hour).UnixMilli(), 10), http.StatusOK},
		{now.Add(-time.Hour).UTC().Format(time.RFC3339), http.StatusUnauthorized},
		{strconv.FormatInt(now.Add(-time.Hour).UnixMilli(), 10), http.StatusUnauthorized},
		{"24h", http.StatusUnauthorized},
		{"5m", http.StatusUnauthorized},
		{"soon", http.StatusUnauthorized},
		{"12abc", http.StatusUnauthorized},
		{"2026-11-01", http.StatusUnauthorized},
	} {
		tok := scopedToken(t, gw, "scope = api:read", "expires = "+tc.value)
		if rec := bearerGet(t, gw, "/api/x", tok); rec.Code != tc.want {
			t.Errorf("expires = %q: status %d, want %d", tc.value, rec.Code, tc.want)
		}
	}
}

// The delegation endpoint refuses an expires caveat that is not an absolute
// time, and accepts an absolute one.
func TestGateway_DelegateRefusesNonAbsoluteExpires(t *testing.T) {
	gw := scopeExpiryGateway(t)
	parent := scopedToken(t, gw, "scope = api:*")
	for _, v := range []string{"24h", "30m", "soon"} {
		rec := postJSON(t, gw, "/api/capability/delegate", map[string]any{"parentToken": parent, "caveats": []string{"expires = " + v}}, false)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("delegate with expires = %q: status %d, want 400", v, rec.Code)
		}
	}
	child := delegateVia(t, gw, parent, "expires = "+time.Now().Add(time.Minute).UTC().Format(time.RFC3339))
	if rec := bearerGet(t, gw, "/api/x", child.Token); rec.Code != http.StatusOK {
		t.Fatalf("child with an absolute expiry: status %d, want 200", rec.Code)
	}
}
