package mcpserver

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/satgate-io/satgate/pkg/governance"
)

const currentResourceTestURL = "https://current.example.test/mcp"
const previousResourceTestURL = "https://previous.example.test/mcp"

func TestPreviousResourcesExactMatchAndMixedChains(t *testing.T) {
	const current = currentResourceTestURL
	const previous = previousResourceTestURL
	for _, tc := range []struct {
		name, resource string
		previous       []string
		values         []string
		refused        bool
	}{
		{"default plain", current, nil, nil, false},
		{"default current", current, nil, []string{current}, false},
		{"default previous refused", current, nil, []string{previous}, true},
		{"empty list unchanged", current, []string{}, []string{previous}, true},
		{"previous exact", current, []string{previous}, []string{previous}, false},
		{"current still accepted", current, []string{previous}, []string{current}, false},
		{"plain still accepted", current, []string{previous}, nil, false},
		{"second previous", current, []string{"https://older.example.test/mcp", previous}, []string{previous}, false},
		{"previous repeated", current, []string{previous}, []string{previous, previous}, false},
		{"mixed current previous", current, []string{previous}, []string{current, previous}, true},
		{"mixed previous current", current, []string{previous}, []string{previous, current}, true},
		{"mixed previous URLs", current, []string{previous, "https://older.example.test/mcp"}, []string{previous, "https://older.example.test/mcp"}, true},
		{"unconfigured", current, []string{previous}, []string{"https://other.example.test/mcp"}, true},
		{"slash", current, []string{previous}, []string{previous + "/"}, true},
		{"case", current, []string{previous}, []string{"https://PREVIOUS.example.test/mcp"}, true},
		{"port", current, []string{previous}, []string{"https://previous.example.test:443/mcp"}, true},
		{"query", current, []string{previous}, []string{previous + "?x=1"}, true},
		{"fragment", current, []string{previous}, []string{previous + "#x"}, true},
		{"http", current, []string{previous}, []string{"http://previous.example.test/mcp"}, true},
		{"lookalike", current, []string{previous}, []string{"https://previous.example.test.evil.test/mcp"}, true},
		{"empty value", current, []string{previous, ""}, []string{""}, true},
		{"empty current fails closed", "", []string{previous}, []string{previous}, true},
		{"empty current plain unchanged", "", []string{previous}, nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			caveats := []string{"scope = *"}
			for _, value := range tc.values {
				caveats = append(caveats, "resource = "+value)
			}
			err := CheckResourceCaveats(caveats, tc.resource, tc.previous...)
			if tc.refused && !errors.Is(err, ErrResourceMismatch) || !tc.refused && err != nil {
				t.Fatalf("refused=%v: %v", tc.refused, err)
			}
		})
	}
}

func TestPreviousResourcesTransportAndDefaultAnswers(t *testing.T) {
	for _, configured := range []bool{false, true} {
		name := "default"
		if configured {
			name = "previous configured"
		}
		t.Run(name, func(t *testing.T) {
			sse := newResourceServer(t, currentResourceTestURL, WithOAuthChallenge(testMetadataURL))
			if configured {
				// Use the public server option, not a test-only authenticator.
				WithResourceBinding(currentResourceTestURL, previousResourceTestURL)(sse)
			}
			srv := httptest.NewServer(sse.Handler())
			defer srv.Close()
			for _, tc := range []struct {
				name      string
				resources []string
				accepted  bool
			}{
				{"plain", nil, true},
				{"current", []string{currentResourceTestURL}, true},
				{"previous", []string{previousResourceTestURL}, configured},
				{"repeated previous", []string{previousResourceTestURL, previousResourceTestURL}, configured},
				{"unconfigured", []string{"https://unknown.example.test/mcp"}, false},
				{"previous slash", []string{previousResourceTestURL + "/"}, false},
				{"mixed", []string{previousResourceTestURL, currentResourceTestURL}, false},
			} {
				t.Run(tc.name, func(t *testing.T) {
					var caveats []string
					for _, resource := range tc.resources {
						caveats = append(caveats, "resource = "+resource)
					}
					token := resourceTestToken(t, caveats...)
					status, hdr, body := streamablePost(t, srv.URL+"/mcp", "", token, initializeBody("2025-03-26"))
					if tc.accepted {
						if status != http.StatusOK {
							t.Fatalf("initialize status=%d", status)
						}
						sid := hdr.Get("Mcp-Session-Id")
						status, _, body = streamablePost(t, srv.URL+"/mcp", sid, token, toolsListBody())
						if status != http.StatusOK || !strings.Contains(body, `"result"`) {
							t.Fatalf("tools/list status=%d", status)
						}
						conn := eaOpenSSE(t, srv.URL, token)
						conn.body.Close()
					} else {
						if status != http.StatusUnauthorized || hdr.Get("WWW-Authenticate") != wantChallenge || body != "authentication failed\n" {
							t.Fatalf("refusal status=%d challenge=%q", status, hdr.Get("WWW-Authenticate"))
						}
						req, _ := http.NewRequest(http.MethodGet, srv.URL+"/sse", nil)
						req.Header.Set("Authorization", "Bearer "+token)
						resp, err := http.DefaultClient.Do(req)
						if err != nil {
							t.Fatal(err)
						}
						resp.Body.Close()
						if resp.StatusCode != http.StatusUnauthorized || resp.Header.Get("WWW-Authenticate") != wantChallenge {
							t.Fatalf("SSE refusal status=%d", resp.StatusCode)
						}
					}
				})
			}
		})
	}
}

func TestPreviousResourcesBindingCopiesAndReplacesList(t *testing.T) {
	svc := newResourceServer(t, currentResourceTestURL).proxy.auth.(*ResourceBoundAuthenticator).Inner
	previous := []string{previousResourceTestURL}
	bound := BindResource(svc, currentResourceTestURL, previous...)
	previous[0] = "https://unconfigured.example.test/mcp"
	ctx := context.Background()
	token := resourceTestToken(t, "resource = "+previousResourceTestURL)
	if _, err := bound.Verify(ctx, token); err != nil {
		t.Fatal(err)
	}
	rebound := BindResource(bound, currentResourceTestURL).(*ResourceBoundAuthenticator)
	if rebound.Inner != svc {
		t.Fatal("rebind nested the wrapper")
	}
	if _, err := rebound.Verify(ctx, token); !errors.Is(err, ErrResourceMismatch) {
		t.Fatalf("removed previous URL still accepted: %v", err)
	}
	if BindResource(nil, currentResourceTestURL, previousResourceTestURL) != nil {
		t.Fatal("nil authenticator changed")
	}
}

func TestPreviousResourcesRetainLineageChecksWhenRebound(t *testing.T) {
	for _, replaceAuth := range []bool{false, true} {
		name := "existing authenticator"
		if replaceAuth {
			name = "replacement authenticator"
		}
		t.Run(name, func(t *testing.T) {
			gov := governance.NewService(nil)
			r := newLineageRig(t, gov)
			previous := []string{previousResourceTestURL}
			r.proxy.SetResource(currentResourceTestURL, previous...)
			previous[0] = "https://unconfigured.example.test/mcp"
			if replaceAuth {
				r.proxy.SetAuthenticator(r.proxy.auth.(*ResourceBoundAuthenticator).Inner)
			}
			root, rootSig := r.mint()
			child, _ := r.delegate(root, "resource = "+previousResourceTestURL)
			r.fund(child, 10)
			sid := mustInitSession(t, r.srv.URL, child)
			gov.Ban(rootSig, "dummy ban", "dummy admin")
			status, _, body := streamablePost(t, r.srv.URL+"/mcp", "", child, initializeBody("2025-03-26"))
			if status != http.StatusForbidden || !strings.Contains(body, "TOKEN_REVOKED") {
				t.Fatalf("banned previous-resource initialize status=%d", status)
			}
			_, _, body = streamablePost(t, r.srv.URL+"/mcp", sid, child, toolCallBody(2, "echo"))
			if e := rpcErrorIn(t, body); e == nil || !strings.Contains(e.Message, "token revoked") {
				t.Fatal("banned previous-resource tools/call accepted")
			}
			if streamableForwardCount(r.proxy) != 0 {
				t.Fatal("banned token forwarded")
			}
			// Rebinding without previous URLs drops them without losing the gate.
			r.proxy.SetResource(currentResourceTestURL)
			if _, err := r.proxy.auth.Verify(context.Background(), child); !errors.Is(err, ErrResourceMismatch) {
				t.Fatalf("previous URL survived reset: %v", err)
			}
			r.proxy.SetResource(currentResourceTestURL, previousResourceTestURL)
			if _, err := r.proxy.auth.Verify(context.Background(), child); !errors.Is(err, errTokenRevoked) {
				t.Fatalf("lineage gate lost on rebind: %v", err)
			}
		})
	}
}
