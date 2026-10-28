package proxy

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	macaroonv2 "gopkg.in/macaroon.v2"

	"github.com/satgate-io/satgate/pkg/governance"
)

// Reviewer cases for the HTTP gateway that held at round 1, kept as tests.

// Through 128 holder attenuations before the ban and 32 after, in the JSON and
// the binary form, every descendant of a banned token is refused and reaches
// no upstream; banning a child then leaves the root and a sibling served and
// refuses the child's descendant.
func TestRevocation_DeepJSONAndBinaryDescendantsRefusedBeforeAndAfterBan(t *testing.T) {
	for _, binary := range []bool{false, true} {
		t.Run(fmt.Sprintf("binary=%v", binary), func(t *testing.T) {
			up, hits := countingUpstream(t)
			gw := newRevocationGateway(t, up.URL, governance.NewService(nil))
			root, err := gw.macaroonSvc.Mint("api:*", time.Now().Add(time.Hour))
			if err != nil {
				t.Fatal(err)
			}
			tok := gw.macaroonSvc.Encode(root)
			if binary {
				if tok, err = gw.macaroonSvc.EncodeBinary(root); err != nil {
					t.Fatal(err)
				}
			}
			parent, err := gw.macaroonSvc.Verify(tok)
			if err != nil {
				t.Fatal(err)
			}
			attenuate := func(tok, caveat string) string {
				t.Helper()
				if binary {
					raw, err := base64.StdEncoding.DecodeString(tok)
					if err != nil {
						t.Fatal(err)
					}
					var m macaroonv2.Macaroon
					if err := m.UnmarshalBinary(raw); err != nil {
						t.Fatal(err)
					}
					if err := m.AddFirstPartyCaveat([]byte(caveat)); err != nil {
						t.Fatal(err)
					}
					raw, _ = m.MarshalBinary()
					return base64.StdEncoding.EncodeToString(raw)
				}
				m, err := gw.macaroonSvc.DelegateWithoutVerify(tok, []string{caveat})
				if err != nil {
					t.Fatal(err)
				}
				return gw.macaroonSvc.Encode(m)
			}
			rootTok := tok
			all := []string{tok}
			for depth := 1; depth <= 128; depth++ {
				tok = attenuate(tok, fmt.Sprintf("label = before-%d", depth))
				all = append(all, tok)
			}
			if rec := bearerGet(t, gw, "/api/x", tok); rec.Code != http.StatusOK {
				t.Fatalf("depth 128 before the ban: want 200, got %d", rec.Code)
			}
			banVia(t, gw, parent.Signature)
			served := hits.Load()
			for i, tk := range all {
				rec := bearerGet(t, gw, "/api/x", tk)
				if rec.Code != http.StatusUnauthorized || strings.Contains(rec.Body.String(), tk) {
					t.Fatalf("depth %d delegated before the ban: status %d", i, rec.Code)
				}
			}
			for depth := 129; depth <= 160; depth++ {
				tok = attenuate(tok, fmt.Sprintf("label = after-%d", depth))
				if rec := bearerGet(t, gw, "/api/x", tok); rec.Code != http.StatusUnauthorized {
					t.Fatalf("depth %d delegated after the ban: status %d", depth, rec.Code)
				}
			}
			if hits.Load() != served {
				t.Fatal("a refused descendant reached the upstream")
			}

			gw.governance.Unban(parent.Signature)
			childTok := attenuate(rootTok, "label = banned-child")
			siblingTok := attenuate(rootTok, "label = sibling")
			child, err := gw.macaroonSvc.Verify(childTok)
			if err != nil {
				t.Fatal(err)
			}
			banVia(t, gw, child.Signature)
			if bearerGet(t, gw, "/api/x", rootTok).Code != http.StatusOK || bearerGet(t, gw, "/api/x", siblingTok).Code != http.StatusOK {
				t.Fatal("a child ban refused the root or a sibling")
			}
			if bearerGet(t, gw, "/api/x", attenuate(childTok, "label = grandchild")).Code != http.StatusUnauthorized {
				t.Fatal("a child ban did not cover the child's descendant")
			}
		})
	}
}

// The refusals of a banned token carry the documented status and code and
// never echo the token.
func TestRevocation_DenialBodiesCarryCodeAndDoNotEchoToken(t *testing.T) {
	up, _ := countingUpstream(t)
	gw := newRevocationGateway(t, up.URL, governance.NewService(nil))
	tok, sig := mintRoot(t, gw)
	banVia(t, gw, sig)
	if rec := bearerGet(t, gw, "/api/x", tok); rec.Code != http.StatusUnauthorized || strings.Contains(rec.Body.String(), tok) {
		t.Fatalf("route: status %d", rec.Code)
	}
	for _, path := range []string{"/api/capability/ping", "/api/capability/admin"} {
		rec := bearerGet(t, gw, path, tok)
		if rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), `"code":"TOKEN_BANNED"`) || strings.Contains(rec.Body.String(), tok) {
			t.Fatalf("%s: status %d", path, rec.Code)
		}
	}
	rec := postJSON(t, gw, "/api/capability/validate", map[string]string{"token": tok}, false)
	if body := rec.Body.String(); rec.Code != http.StatusUnauthorized || !strings.Contains(body, `"valid":false`) || !strings.Contains(body, `"code":"TOKEN_BANNED"`) || strings.Contains(body, tok) {
		t.Fatalf("validate: status %d", rec.Code)
	}
	rec = postJSON(t, gw, "/api/capability/delegate", map[string]any{"parentToken": tok, "caveats": []string{"scope = api:read"}}, false)
	if body := rec.Body.String(); rec.Code != http.StatusForbidden || !strings.Contains(body, `"code":"TOKEN_BANNED"`) || strings.Contains(body, tok) {
		t.Fatalf("delegate: status %d", rec.Code)
	}
}

// Two gateways on one Redis: with the other gateway's answer cached, a ban
// and then an unban are each honoured there once the 2-second TTL has passed.
func TestRevocation_BanAndUnbanReachAnotherGatewayAfterTheTTL(t *testing.T) {
	mr := miniredis.RunT(t)
	up, _ := countingUpstream(t)
	aGov := newRedisGovernance(t, mr.Addr())
	bGov := newRedisGovernance(t, mr.Addr())
	t.Cleanup(func() { _ = aGov.Close(); _ = bGov.Close() })
	a := newRevocationGateway(t, up.URL, aGov)
	b := newRevocationGateway(t, up.URL, bGov)
	tok, sig := mintRoot(t, a)
	if rec := bearerGet(t, b, "/api/x", tok); rec.Code != http.StatusOK {
		t.Fatalf("before the ban: %d", rec.Code)
	}
	banVia(t, a, sig)
	time.Sleep(governance.BanCacheTTL + 20*time.Millisecond)
	if rec := bearerGet(t, b, "/api/x", tok); rec.Code != http.StatusUnauthorized {
		t.Fatalf("ban after the TTL: %d", rec.Code)
	}
	if !aGov.Unban(sig) {
		t.Fatal("unban reported no ban")
	}
	time.Sleep(governance.BanCacheTTL + 20*time.Millisecond)
	if rec := bearerGet(t, b, "/api/x", tok); rec.Code != http.StatusOK {
		t.Fatalf("unban after the TTL: %d", rec.Code)
	}
}
