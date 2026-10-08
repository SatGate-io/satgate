package mcpserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const testMetadataURL = "https://mcp.example.test/.well-known/oauth-protected-resource/mcp"

const wantChallenge = `Bearer resource_metadata="` + testMetadataURL + `"`

// Challenge on: tokenless and bad-token initialize get 401 with the RFC 9728
// challenge, a short body, no session, and the presented token is never echoed.
func TestOAuthChallengeStreamableInitialize(t *testing.T) {
	sse, _ := newHeaderModeStreamable(t, WithOAuthChallenge(testMetadataURL))
	srv := httptest.NewServer(sse.Handler())
	defer srv.Close()

	for _, tok := range []string{"", "bad-token-SECRET-123"} {
		status, hdr, body := streamablePost(t, srv.URL+"/mcp", "", tok, initializeBody("2025-03-26"))
		if status != http.StatusUnauthorized {
			t.Fatalf("token %q: status=%d", tok, status)
		}
		if got := hdr.Get("WWW-Authenticate"); got != wantChallenge {
			t.Fatalf("token %q: WWW-Authenticate=%q want %q", tok, got, wantChallenge)
		}
		if strings.Contains(body, "SECRET") || len(body) > 64 {
			t.Fatalf("body must be short and never echo the token: %q", body)
		}
	}
	if n := streamSessionCount(sse); n != 0 {
		t.Fatalf("refused initialize left %d sessions", n)
	}
	status, hdr, _ := streamablePost(t, srv.URL+"/mcp", "", "good-token", initializeBody("2025-03-26"))
	if status != http.StatusOK || hdr.Get("Mcp-Session-Id") == "" {
		t.Fatalf("good token must still initialize: %d", status)
	}
}

// Challenge off (default): the 401 is byte-for-byte the historical one.
func TestOAuthChallengeOffKeepsBare401(t *testing.T) {
	sse, _ := newHeaderModeStreamable(t)
	srv := httptest.NewServer(sse.Handler())
	defer srv.Close()

	for tok, wantBody := range map[string]string{"": "authentication required\n", "bad-token": "authentication failed\n"} {
		status, hdr, body := streamablePost(t, srv.URL+"/mcp", "", tok, initializeBody("2025-03-26"))
		if status != http.StatusUnauthorized || body != wantBody {
			t.Fatalf("token %q: status=%d body=%q", tok, status, body)
		}
		if _, present := hdr["Www-Authenticate"]; present {
			t.Fatalf("token %q: unexpected WWW-Authenticate %q", tok, hdr.Get("WWW-Authenticate"))
		}
	}
}

// An invalid metadata URL (http, userinfo, quote, empty) is ignored: the
// server keeps the bare 401 rather than emitting a broken header.
func TestOAuthChallengeRejectsUnsafeMetadataURL(t *testing.T) {
	for _, bad := range []string{
		"", "http://mcp.example.test/.well-known/oauth-protected-resource",
		"https://u:p@mcp.example.test/x", "https://mcp.example.test/x\"; evil=\"1",
		"https://mcp.example.test/x#frag", "https://mcp.example.test/x?a=b", "/relative", "https://mcp.example.test/a b",
		"https://mcp.example.test/a\r\nX: y",
	} {
		sse, _ := newHeaderModeStreamable(t, WithOAuthChallenge(bad))
		srv := httptest.NewServer(sse.Handler())
		_, hdr, _ := streamablePost(t, srv.URL+"/mcp", "", "", initializeBody("2025-03-26"))
		srv.Close()
		if _, present := hdr["Www-Authenticate"]; present {
			t.Fatalf("url %q must be ignored, got header %q", bad, hdr.Get("WWW-Authenticate"))
		}
	}
}

// Existing session: after the access token expires or is revoked the app must
// get 401 + challenge (so it refreshes), and a freshly refreshed token that
// verifies but is not the session's token gets 404 (start a new session).
func TestOAuthChallengeExistingSessionTokenStates(t *testing.T) {
	sse, _ := newHeaderModeStreamable(t, WithOAuthChallenge(testMetadataURL))
	srv := httptest.NewServer(sse.Handler())
	defer srv.Close()
	sid := mustInitSession(t, srv.URL, "good-token")

	status, hdr, _ := streamablePost(t, srv.URL+"/mcp", sid, "", toolsListBody())
	if status != http.StatusUnauthorized || hdr.Get("WWW-Authenticate") != wantChallenge {
		t.Fatalf("no token on session: %d %q", status, hdr.Get("WWW-Authenticate"))
	}
	status, hdr, _ = streamablePost(t, srv.URL+"/mcp", sid, "expired-token", toolsListBody())
	if status != http.StatusUnauthorized || hdr.Get("WWW-Authenticate") != wantChallenge {
		t.Fatalf("bad token on session: %d %q", status, hdr.Get("WWW-Authenticate"))
	}
	status, _, _ = streamablePost(t, srv.URL+"/mcp", sid, "good-token", toolsListBody())
	if status != http.StatusOK {
		t.Fatalf("same token on session: %d", status)
	}
}

// Challenge off: a mismatched session token is still 403 session_token_mismatch.
func TestOAuthChallengeOffSessionMismatchStill403(t *testing.T) {
	sse, _ := newHeaderModeStreamable(t)
	srv := httptest.NewServer(sse.Handler())
	defer srv.Close()
	sid := mustInitSession(t, srv.URL, "good-token")
	status, _, body := streamablePost(t, srv.URL+"/mcp", sid, "other", toolsListBody())
	if status != http.StatusForbidden || !strings.Contains(body, "session_token_mismatch") {
		t.Fatalf("status=%d body=%q", status, body)
	}
}

// Legacy SSE path: tokenless GET /sse gets the challenge and opens no stream.
func TestOAuthChallengeSSEConnect(t *testing.T) {
	sse, _ := newHeaderModeStreamable(t, WithOAuthChallenge(testMetadataURL))
	srv := httptest.NewServer(sse.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/sse")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized || resp.Header.Get("WWW-Authenticate") != wantChallenge {
		t.Fatalf("tokenless sse: %d %q", resp.StatusCode, resp.Header.Get("WWW-Authenticate"))
	}
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/sse", nil)
	req.Header.Set("Authorization", "Bearer bad-token")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	// fixedTokenAuth errors do not match the hard-failure strings, so the
	// challenge branch is the one that answers.
	if resp.StatusCode != http.StatusUnauthorized || resp.Header.Get("WWW-Authenticate") != wantChallenge {
		t.Fatalf("bad-token sse: %d %q", resp.StatusCode, resp.Header.Get("WWW-Authenticate"))
	}
	if n := len(sse.sessions); n != 0 {
		t.Fatalf("refused sse connect left %d sessions", n)
	}
}

// Legacy SSE path, challenge off: the historical behaviour is untouched
// (tokenless connect opens a stream; deferred verification).
func TestOAuthChallengeOffSSEConnectUnchanged(t *testing.T) {
	sse, _ := newHeaderModeStreamable(t)
	srv := httptest.NewServer(sse.Handler())
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/sse")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK || resp.Header.Get("WWW-Authenticate") != "" {
		t.Fatalf("challenge off must not change /sse: %d %q", resp.StatusCode, resp.Header.Get("WWW-Authenticate"))
	}
}

func toolsListBody() map[string]any {
	return map[string]any{"jsonrpc": "2.0", "id": 7, "method": "tools/list", "params": map[string]any{}}
}
