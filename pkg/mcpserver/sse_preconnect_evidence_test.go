package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

type rejectingAuthenticator struct{ err error }

func (a rejectingAuthenticator) Verify(context.Context, string) (*TokenInfo, error) {
	return nil, a.err
}

type resolvingAuthenticator struct{}

func (resolvingAuthenticator) Verify(context.Context, string) (*TokenInfo, error) {
	time.Sleep(time.Millisecond)
	return &TokenInfo{
		TokenID:  "token-late",
		TenantID: "tenant-late",
		BudgetID: "budget-late",
	}, nil
}

func preConnectRequest(t *testing.T, proxy *Proxy, trustedTenant string) *httptest.ResponseRecorder {
	t.Helper()
	sse := NewSSEServer(proxy, ":0")
	req := httptest.NewRequest(http.MethodGet, "/sse", nil)
	req.Header.Set("Authorization", "Bearer redacted-test-token")
	if trustedTenant != "" {
		req = req.WithContext(context.WithValue(req.Context(), CtxTenantID, trustedTenant))
	}
	rec := httptest.NewRecorder()
	sse.Handler().ServeHTTP(rec, req)
	return rec
}

func decodePreConnectBody(t *testing.T, rec *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v body=%q", err, rec.Body.String())
	}
	return body
}

func TestSSEPreConnectRevocationReturnsTrustedDenialProof(t *testing.T) {
	proxy := newEvidenceTestProxy(t)
	recorder := &fakeEvidenceRecorder{}
	proxy.SetEvidenceRecorder(recorder)
	proxy.SetAuthenticator(rejectingAuthenticator{err: errors.New("token revoked")})

	rec := preConnectRequest(t, proxy, "tenant-trusted")
	body := decodePreConnectBody(t, rec)
	if rec.Code != http.StatusForbidden || body["error"] != "token_revoked" || body["code"] != "TOKEN_REVOKED" {
		t.Fatalf("unexpected denial: status=%d body=%#v", rec.Code, body)
	}
	proof, _ := body["proof"].(map[string]interface{})
	if proof["evidence_url"] == "" || rec.Header().Get("X-SatGate-Evidence-Pack-ID") == "" || !strings.Contains(rec.Header().Get("Link"), "satgate-evidence-pack") {
		t.Fatalf("missing proof metadata: headers=%v body=%#v", rec.Header(), body)
	}
	if len(recorder.decisions) != 1 {
		t.Fatalf("decisions=%d", len(recorder.decisions))
	}
	decision := recorder.decisions[0]
	if decision.TenantID != "tenant-trusted" || decision.Decision != "denied" || decision.DecisionReason != "token_revoked" || !decision.NoVerifiedCapability || decision.TokenID != "" || decision.BudgetID != "" {
		t.Fatalf("unsafe denial decision: %#v", decision)
	}
}

func TestSSEPreConnectRevocationWithoutTrustedTenantReturnsNoProof(t *testing.T) {
	proxy := newEvidenceTestProxy(t)
	recorder := &fakeEvidenceRecorder{}
	proxy.SetEvidenceRecorder(recorder)
	proxy.SetAuthenticator(rejectingAuthenticator{err: errors.New("token revoked")})

	rec := preConnectRequest(t, proxy, "")
	body := decodePreConnectBody(t, rec)
	if rec.Code != http.StatusForbidden || body["error"] != "token_revoked" || body["proof"] != nil {
		t.Fatalf("unexpected denial: status=%d body=%#v", rec.Code, body)
	}
	if len(recorder.decisions) != 0 || rec.Header().Get("X-SatGate-Evidence-Pack-ID") != "" {
		t.Fatal("untrusted request produced proof")
	}
}

func TestSSEPreConnectRevocationRecorderFailureStillDeniesWithoutProof(t *testing.T) {
	proxy := newEvidenceTestProxy(t)
	recorder := &fakeEvidenceRecorder{recordErr: errors.New("archive unavailable")}
	proxy.SetEvidenceRecorder(recorder)
	proxy.SetAuthenticator(rejectingAuthenticator{err: errors.New("token revoked")})

	rec := preConnectRequest(t, proxy, "tenant-trusted")
	body := decodePreConnectBody(t, rec)
	if rec.Code != http.StatusForbidden || body["error"] != "token_revoked" || body["proof"] != nil {
		t.Fatalf("enforcement changed on recorder failure: status=%d body=%#v", rec.Code, body)
	}
}

func TestSSEPreConnectRevocationPreflightFailureStillDeniesWithoutProof(t *testing.T) {
	proxy := newEvidenceTestProxy(t)
	recorder := &fakeEvidenceRecorder{preflightErr: errors.New("signer unavailable")}
	proxy.SetEvidenceRecorder(recorder)
	proxy.SetAuthenticator(rejectingAuthenticator{err: errors.New("token revoked")})

	rec := preConnectRequest(t, proxy, "tenant-trusted")
	body := decodePreConnectBody(t, rec)
	if rec.Code != http.StatusForbidden || body["error"] != "token_revoked" || body["proof"] != nil {
		t.Fatalf("enforcement changed on preflight failure: status=%d body=%#v", rec.Code, body)
	}
	if len(recorder.decisions) != 0 {
		t.Fatal("preflight failure still attempted evidence recording")
	}
}

func TestSSEPreConnectWrongLaneNeverProducesRevocationProof(t *testing.T) {
	proxy := newEvidenceTestProxy(t)
	recorder := &fakeEvidenceRecorder{}
	proxy.SetEvidenceRecorder(recorder)
	proxy.SetAuthenticator(rejectingAuthenticator{err: errors.New("enterprise deployment token rejected")})

	rec := preConnectRequest(t, proxy, "tenant-trusted")
	body := decodePreConnectBody(t, rec)
	if rec.Code != http.StatusForbidden || body["error"] != "wrong_lane" || body["code"] != "WRONG_LANE" || body["proof"] != nil {
		t.Fatalf("unexpected wrong-lane denial: status=%d body=%#v", rec.Code, body)
	}
	if len(recorder.decisions) != 0 {
		t.Fatal("wrong-lane rejection was mislabeled as revocation proof")
	}
}

func TestSSEConcurrentLateIdentityResolutionAndClose(t *testing.T) {
	proxy := newEvidenceTestProxy(t)
	proxy.SetAuthenticator(resolvingAuthenticator{})
	sse := NewSSEServer(proxy, ":0")
	server := httptest.NewServer(sse.Handler())
	defer server.Close()

	sseResp, err := http.Get(server.URL + "/sse")
	if err != nil {
		t.Fatal(err)
	}
	sessionID := readEndpointEvent(t, sseResp.Body)
	if sessionID == "" {
		t.Fatal("missing session ID")
	}

	const requestCount = 32
	errs := make(chan error, requestCount)
	var wg sync.WaitGroup
	for i := 0; i < requestCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req, err := http.NewRequest(http.MethodPost, server.URL+"/message?sessionId="+sessionID, strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"ping"}`))
			if err != nil {
				errs <- err
				return
			}
			req.Header.Set("Authorization", "Bearer late-token")
			req.Header.Set("Content-Type", "application/json")
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				errs <- err
				return
			}
			_, _ = io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusNotFound {
				errs <- errors.New("unexpected message status: " + resp.Status)
			}
		}()
	}

	time.Sleep(time.Millisecond)
	_ = sseResp.Body.Close()
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
}
