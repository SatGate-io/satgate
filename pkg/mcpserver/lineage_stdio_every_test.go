package mcpserver

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/satgate-io/satgate/pkg/governance"
	"github.com/satgate-io/satgate/pkg/macaroon"
)

// Over stdio with a lineage checker installed, every credential a request
// supplies, the connection's (withTransportToken) and params._meta.token, is
// verified and its lineage checked before anything is signed or forwarded,
// on every branch, with sign-every-action off and on: tools/list, the
// authenticated methods (tools/call, satgate/delegate, satgate/budget,
// budget/check) and the methods passed on to the default upstream. A
// request that supplies no token, and an embedder that installs no checker,
// behave as before.

// stdioAuthenticatedMethods go through dispatchAuthenticated whatever the
// switch.
var stdioAuthenticatedMethods = []string{MethodToolsCall, MethodSatGateDelegate, MethodSatGateBudget, "budget/check"}

// selectivelyUnreadable is a checker that cannot read the ban state of one
// signature (and of every lineage that contains it).
type selectivelyUnreadable struct{ sig string }

func (c selectivelyUnreadable) IsAnyBanned(_ context.Context, sigs []string) (bool, error) {
	for _, s := range sigs {
		if s == c.sig {
			return false, errors.New("ban state unreadable")
		}
	}
	return false, nil
}

// stdioCreds are credentials under the stdio rig's root key: bad is a
// descendant of badRoot; healthy is an unrelated root (other expiry, so a
// different signature).
type stdioCreds struct {
	badRoot      *macaroon.Macaroon
	bad, healthy string
}

func newStdioCreds(t *testing.T) stdioCreds {
	t.Helper()
	svc, err := macaroon.NewService(stdioLineageRoot)
	if err != nil {
		t.Fatal(err)
	}
	root, err := svc.Mint("api:*", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	child, err := svc.Delegate(svc.Encode(root), []string{"label = stdio-supplied"})
	if err != nil {
		t.Fatal(err)
	}
	other, err := svc.Mint("api:*", time.Now().Add(2*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if other.Signature == root.Signature {
		t.Fatal("fixture: healthy and banned roots share a signature")
	}
	return stdioCreds{badRoot: root, bad: svc.Encode(child), healthy: svc.Encode(other)}
}

// stdioRigWith runs newStdioRig with a connection credential conn ("" for
// none), a tenant on the context, and the switch set to sign.
func stdioRigWith(t *testing.T, checker LineageRevocationChecker, conn string, sign bool) *stdioRig {
	t.Helper()
	r := newStdioRig(t, checker, func(ctx context.Context) context.Context {
		ctx = context.WithValue(ctx, CtxTenantID, "stdio-test-tenant")
		if conn != "" {
			ctx = withTransportToken(ctx, conn)
		}
		return ctx
	})
	if sign {
		r.proxy.SetEvidenceRecorder(newEAWindowRecorder(&eaRecorder{}))
		if err := r.proxy.SetSignEveryAction(true); err != nil {
			t.Fatal(err)
		}
	}
	return r
}

func switchName(sign bool) string {
	if sign {
		return "sign_every_action_on"
	}
	return "sign_every_action_off"
}

// Every stdio branch that signs or forwards refuses a request when either
// supplied credential is a descendant of a banned token ("token revoked"),
// has a ban state that cannot be read (revocation-unavailable) or does not
// verify ("authentication failed"), and the other one is healthy; under both
// switch states.
func TestMCPStdio_EverySuppliedCredentialCheckedOnEveryBranch(t *testing.T) {
	methods := append(append([]string{MethodToolsList}, stdioAuthenticatedMethods...), stdioPassedOn...)
	for _, sign := range []bool{false, true} {
		for _, state := range []string{"banned", "unreadable", "does_not_verify"} {
			for _, where := range []string{"connection", "meta"} {
				t.Run(switchName(sign)+"/"+state+"_"+where, func(t *testing.T) {
					c := newStdioCreds(t)
					gov := governance.NewService(nil)
					var checker LineageRevocationChecker = gov
					switch state {
					case "banned":
						gov.Ban(c.badRoot.Signature, "test", "admin")
					case "unreadable":
						checker = selectivelyUnreadable{sig: c.badRoot.Signature}
					case "does_not_verify":
						other, _ := macaroon.NewService("some-other-root")
						mac, _ := other.Mint("api:*", time.Now().Add(time.Hour))
						c.bad = other.Encode(mac)
					}
					conn, meta := c.bad, c.healthy
					if where == "meta" {
						conn, meta = c.healthy, c.bad
					}
					r := stdioRigWith(t, checker, conn, sign)
					if _, err := r.proxy.auth.Verify(context.Background(), c.healthy); err != nil {
						t.Fatalf("fixture: healthy credential refused: %v", err)
					}
					for _, m := range methods {
						resp := r.call(m, meta)
						if resp.Error == nil {
							t.Fatalf("%s served with a %s %s credential", m, state, where)
						}
						if state == "banned" && !strings.Contains(resp.Error.Message, "token revoked") {
							t.Fatalf("%s: want token revoked, got %+v", m, resp.Error)
						}
						if state == "unreadable" && !isRevocationUnavailableResponse(resp) {
							t.Fatalf("%s: want revocation-unavailable, got %+v", m, resp.Error)
						}
						if state == "does_not_verify" && !strings.Contains(resp.Error.Message, "authentication failed") {
							t.Fatalf("%s: want authentication failed, got %+v", m, resp.Error)
						}
					}
				})
			}
		}
	}
}

// Control: with a checker installed and both supplied credentials healthy
// (and different), tools/list is answered and passed-on methods are
// forwarded, under both switch states.
func TestMCPStdio_TwoHealthySuppliedCredentialsServed(t *testing.T) {
	for _, sign := range []bool{false, true} {
		t.Run(switchName(sign), func(t *testing.T) {
			c := newStdioCreds(t)
			gov := governance.NewService(nil)
			gov.Ban("unrelated-signature", "test", "admin")
			svc, _ := macaroon.NewService(stdioLineageRoot)
			conn, _ := svc.Mint("api:*", time.Now().Add(3*time.Hour))
			r := stdioRigWith(t, gov, svc.Encode(conn), sign)
			if resp := r.call(MethodToolsList, c.healthy); !listed(resp) {
				t.Fatalf("tools/list refused: %+v", resp.Error)
			}
			for _, m := range stdioPassedOn {
				if resp := r.call(m, c.healthy); !forwarded(resp) {
					t.Fatalf("%s not passed on: %+v", m, resp.Error)
				}
			}
			if resp := r.call(MethodSatGateBudget, c.healthy); resp.Error != nil {
				t.Fatalf("satgate/budget refused: %+v", resp.Error)
			}
		})
	}
}

// Unchanged: a stdio request that supplies no token, with a checker
// installed, is served on tools/list and passed-on methods with the switch
// off, and refused for having no token on the authenticated path (header
// mode) as before.
func TestMCPStdio_TokenlessRequestsUnchangedUnderBothSwitchStates(t *testing.T) {
	for _, sign := range []bool{false, true} {
		t.Run(switchName(sign), func(t *testing.T) {
			r := stdioRigWith(t, governance.NewService(nil), "", sign)
			if resp := r.call(MethodToolsList, ""); !listed(resp) {
				t.Fatalf("tools/list without a token refused: %+v", resp.Error)
			}
			for _, m := range stdioPassedOn {
				resp := r.call(m, "")
				if !sign && !forwarded(resp) {
					t.Fatalf("%s without a token not passed on: %+v", m, resp.Error)
				}
				if sign && (resp.Error == nil || !strings.Contains(resp.Error.Message, "no token provided")) {
					t.Fatalf("%s without a token, switch on: want the no-token refusal, got %+v", m, resp.Error)
				}
			}
			for _, m := range stdioAuthenticatedMethods {
				if resp := r.call(m, ""); resp.Error == nil || !strings.Contains(resp.Error.Message, "no token provided") {
					t.Fatalf("%s without a token: want the no-token refusal, got %+v", m, resp.Error)
				}
			}
		})
	}
}

// Unchanged: an embedder that installs no checker. A connection credential
// that does not verify, next to a healthy params._meta.token, is not
// checked: passed-on methods are forwarded and tools/list answered, under
// both switch states.
func TestMCPStdio_NoCheckerLeavesSecondCredentialUnchecked(t *testing.T) {
	other, _ := macaroon.NewService("some-other-root")
	mac, _ := other.Mint("api:*", time.Now().Add(time.Hour))
	forged := other.Encode(mac)
	for _, sign := range []bool{false, true} {
		t.Run(switchName(sign), func(t *testing.T) {
			c := newStdioCreds(t)
			r := stdioRigWith(t, nil, forged, sign)
			if resp := r.call(MethodToolsList, c.healthy); !listed(resp) {
				t.Fatalf("tools/list refused without a checker: %+v", resp.Error)
			}
			for _, m := range stdioPassedOn {
				if resp := r.call(m, c.healthy); !forwarded(resp) {
					t.Fatalf("%s not passed on without a checker: %+v", m, resp.Error)
				}
			}
		})
	}
}

// ---- the reviewers' cases ----

// Astra: sign-every-action on, a banned connection credential (descendant of
// a banned root, no tenant on the context) and a healthy params._meta.token:
// a passed-on method is refused, not forwarded.
func TestMCPStdio_SignEveryAction_BannedConnectionCredentialRefusedDespiteHealthyMetaToken(t *testing.T) {
	gov := governance.NewService(nil)
	svc, _ := macaroon.NewService(stdioLineageRoot)
	root, _ := svc.Mint("api:*", time.Now().Add(time.Hour))
	child, _ := svc.Delegate(svc.Encode(root), []string{"label = connection"})
	gov.Ban(root.Signature, "test", "admin")
	r := newStdioRig(t, gov, func(ctx context.Context) context.Context { return withTransportToken(ctx, svc.Encode(child)) })
	r.proxy.SetEvidenceRecorder(newEAWindowRecorder(&eaRecorder{}))
	if err := r.proxy.SetSignEveryAction(true); err != nil {
		t.Fatal(err)
	}
	other, _ := svc.Mint("api:*", time.Now().Add(2*time.Hour))
	resp := r.call("resources/list", svc.Encode(other))
	if forwarded(resp) {
		t.Fatal("passed-on method forwarded despite a supplied banned connection credential")
	}
	if resp.Error == nil || !strings.Contains(resp.Error.Message, "token revoked") {
		t.Fatalf("want token revoked, got %+v", resp.Error)
	}
}

// Sol: both switch states, a banned connection credential or one whose ban
// state cannot be read, a healthy params._meta.token, a tenant on the
// context: resources/list, prompts/list and a vendor method are refused.
func TestMCPStdio_BothSuppliedTokensCheckedOnPassedOnMethods(t *testing.T) {
	for _, sign := range []bool{false, true} {
		for _, outage := range []bool{false, true} {
			name := switchName(sign)
			if outage {
				name += "_unavailable"
			}
			t.Run(name, func(t *testing.T) {
				gov := governance.NewService(nil)
				defer gov.Close()
				svc, _ := macaroon.NewService(stdioLineageRoot)
				conn, _ := svc.Mint("api:*", time.Now().Add(time.Hour))
				gov.Ban(conn.Signature, "test", "admin")
				r := newStdioRig(t, gov, func(ctx context.Context) context.Context {
					return withTransportToken(context.WithValue(ctx, CtxTenantID, "stdio-test-tenant"), svc.Encode(conn))
				})
				healthyMac, _ := r.svc.Mint("mcp:*", time.Now().Add(2*time.Hour))
				healthy := r.svc.Encode(healthyMac)
				if _, err := r.proxy.auth.Verify(context.Background(), healthy); err != nil {
					t.Fatal("healthy credential fixture did not verify")
				}
				if outage {
					r.proxy.SetLineageRevocationChecker(selectivelyUnreadable{sig: conn.Signature})
				}
				if sign {
					r.proxy.SetEvidenceRecorder(newEAWindowRecorder(&eaRecorder{}))
					if err := r.proxy.SetSignEveryAction(true); err != nil {
						t.Fatal(err)
					}
				}
				for _, method := range []string{"resources/list", "prompts/list", "vendor/custom"} {
					if resp := r.call(method, healthy); resp.Error == nil {
						t.Errorf("%s served with a supplied refused connection credential", method)
					}
				}
			})
		}
	}
}
