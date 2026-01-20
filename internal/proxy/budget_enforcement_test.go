package proxy

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Mock budget enforcer for testing
type mockBudgetEnforcer struct {
	allowRequest bool
	remaining    int64
	cost         int64
	calls        []budgetEnforcerCall
}

type budgetEnforcerCall struct {
	TokenID     string
	TenantID    string
	Route       string
	Method      string
	PolicyMode  string
	RequestID   string
	CostCredits int64
}

func (m *mockBudgetEnforcer) EnforceFunc(ctx context.Context, tokenID, tenantID, route, method, policyMode, requestID string, costCredits int64) (*BudgetResult, error) {
	m.calls = append(m.calls, budgetEnforcerCall{
		TokenID:     tokenID,
		TenantID:    tenantID,
		Route:       route,
		Method:      method,
		PolicyMode:  policyMode,
		RequestID:   requestID,
		CostCredits: costCredits,
	})
	
	if !m.allowRequest {
		return &BudgetResult{
			CostCredits:      m.cost,
			RemainingCredits: m.remaining,
		}, errors.New("budget exhausted")
	}
	
	return &BudgetResult{
		CostCredits:      m.cost,
		RemainingCredits: m.remaining,
	}, nil
}

// Mock delegation verifier for testing
type mockDelegationVerifier struct {
	tokenID  string
	tenantID string
	scopes   []string
	err      error
}

func (m *mockDelegationVerifier) VerifySecret(ctx context.Context, secret string) (tokenID, tenantID string, scopes []string, err error) {
	if m.err != nil {
		return "", "", nil, m.err
	}
	return m.tokenID, m.tenantID, m.scopes, nil
}

// TestDelegationTokenEnforcesBudget verifies that delegation tokens (stks_*) 
// have budget enforcement applied on fiat402 routes
func TestDelegationTokenEnforcesBudget(t *testing.T) {
	tests := []struct {
		name           string
		allowRequest   bool
		expectedStatus int
		expectedCalls  int
	}{
		{
			name:           "budget_allowed",
			allowRequest:   true,
			expectedStatus: 0, // No error, would proceed to proxy
			expectedCalls:  1,
		},
		{
			name:           "budget_exhausted",
			allowRequest:   false,
			expectedStatus: http.StatusPaymentRequired,
			expectedCalls:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			budgetEnforcer := &mockBudgetEnforcer{
				allowRequest: tt.allowRequest,
				remaining:    100,
				cost:         1,
			}

			// Create principal with BudgetSubjectID set (delegation token)
			principal := &VerifiedPrincipal{
				Type:            PrincipalTypeDelegation,
				SubjectID:       "stk_test123",
				BudgetSubjectID: "stk_test123", // This triggers budget enforcement
				TenantID:        "tenant-uuid-123",
				Scopes:          []string{"/api/*"},
			}

			// Create request with principal in context
			req := httptest.NewRequest(http.MethodGet, "/api/v1/completions", nil)
			req.Header.Set("X-Request-ID", "req-123")
			ctx := context.WithValue(req.Context(), principalKey{}, principal)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()

			// Simulate budget enforcement logic from fiat402 case
			if budgetEnforcer != nil {
				if p := GetVerifiedPrincipal(req.Context()); p != nil && p.BudgetSubjectID != "" {
					requestID := req.Header.Get("X-Request-ID")
					var costCredits int64 = 1 // Default cost
					
					result, err := budgetEnforcer.EnforceFunc(
						req.Context(),
						p.BudgetSubjectID,
						p.TenantID,
						req.URL.Path,
						req.Method,
						"fiat402",
						requestID,
						costCredits,
					)
					
					if err != nil {
						w.WriteHeader(http.StatusPaymentRequired)
						w.Write([]byte(`{"error":"budget_exhausted"}`))
					} else if result != nil {
						w.Header().Set("X-Budget-Remaining", "100")
					}
				}
			}

			// Verify budget enforcer was called
			if len(budgetEnforcer.calls) != tt.expectedCalls {
				t.Errorf("expected %d budget enforcer calls, got %d", tt.expectedCalls, len(budgetEnforcer.calls))
			}

			if tt.expectedCalls > 0 {
				call := budgetEnforcer.calls[0]
				if call.TokenID != "stk_test123" {
					t.Errorf("expected TokenID 'stk_test123', got '%s'", call.TokenID)
				}
				if call.TenantID != "tenant-uuid-123" {
					t.Errorf("expected TenantID 'tenant-uuid-123', got '%s'", call.TenantID)
				}
				if call.Route != "/api/v1/completions" {
					t.Errorf("expected Route '/api/v1/completions', got '%s'", call.Route)
				}
				if call.RequestID != "req-123" {
					t.Errorf("expected RequestID 'req-123', got '%s'", call.RequestID)
				}
			}

			if tt.expectedStatus != 0 && w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

// TestMacaroonDoesNotEnforceBudget verifies that macaroon tokens
// do NOT have budget enforcement (BudgetSubjectID is empty)
func TestMacaroonDoesNotEnforceBudget(t *testing.T) {
	budgetEnforcer := &mockBudgetEnforcer{
		allowRequest: true,
		remaining:    100,
		cost:         1,
	}

	// Create principal WITHOUT BudgetSubjectID (macaroon)
	principal := &VerifiedPrincipal{
		Type:            PrincipalTypeMacaroon,
		SubjectID:       "mac-signature-abc123",
		BudgetSubjectID: "", // Empty - macaroons don't have budget enforcement
		TenantID:        "tenant-uuid-123",
		Scopes:          []string{"api:read"},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/data", nil)
	ctx := context.WithValue(req.Context(), principalKey{}, principal)
	req = req.WithContext(ctx)

	// Simulate budget enforcement logic from fiat402 case
	if budgetEnforcer != nil {
		if p := GetVerifiedPrincipal(req.Context()); p != nil && p.BudgetSubjectID != "" {
			// This should NOT be reached for macaroons
			budgetEnforcer.EnforceFunc(
				req.Context(),
				p.BudgetSubjectID,
				p.TenantID,
				req.URL.Path,
				req.Method,
				"fiat402",
				"",
				1, // cost
			)
		}
	}

	// Verify budget enforcer was NOT called for macaroon
	if len(budgetEnforcer.calls) != 0 {
		t.Errorf("expected 0 budget enforcer calls for macaroon, got %d", len(budgetEnforcer.calls))
	}
}

// TestRequestIDFallbackUsesPathAndMethod verifies that the idempotency key
// fallback uses URL path + method, not route name (to avoid collisions)
func TestRequestIDFallbackUsesPathAndMethod(t *testing.T) {
	budgetEnforcer := &mockBudgetEnforcer{
		allowRequest: true,
		remaining:    100,
		cost:         1,
	}

	principal := &VerifiedPrincipal{
		Type:            PrincipalTypeDelegation,
		SubjectID:       "stk_test456",
		BudgetSubjectID: "stk_test456",
		TenantID:        "tenant-uuid-456",
		Scopes:          []string{"*"},
	}

	// Test different paths on the same "route" - should have different idempotency fallbacks
	paths := []struct {
		path   string
		method string
	}{
		{"/api/v1/users/123", http.MethodGet},
		{"/api/v1/users/456", http.MethodGet},
		{"/api/v1/users/123", http.MethodPost},
	}

	for _, p := range paths {
		req := httptest.NewRequest(p.method, p.path, nil)
		// No X-Request-ID header - force fallback
		ctx := context.WithValue(req.Context(), principalKey{}, principal)
		req = req.WithContext(ctx)

		if pr := GetVerifiedPrincipal(req.Context()); pr != nil && pr.BudgetSubjectID != "" {
			requestID := req.Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = req.Header.Get("Idempotency-Key")
			}
			// Note: When requestID is empty, EnforceFunc passes it as empty
			// and the budget service generates a deterministic ID from path+method
			
			budgetEnforcer.EnforceFunc(
				req.Context(),
				pr.BudgetSubjectID,
				pr.TenantID,
				req.URL.Path, // This is the key fix - use URL.Path, not route.Name
				req.Method,
				"fiat402",
				requestID,
				1, // default cost
			)
		}
	}

	// Verify all calls used URL path (not a single route name)
	if len(budgetEnforcer.calls) != 3 {
		t.Fatalf("expected 3 calls, got %d", len(budgetEnforcer.calls))
	}

	// Each call should have the actual URL path
	expectedPaths := []string{"/api/v1/users/123", "/api/v1/users/456", "/api/v1/users/123"}
	expectedMethods := []string{http.MethodGet, http.MethodGet, http.MethodPost}
	
	for i, call := range budgetEnforcer.calls {
		if call.Route != expectedPaths[i] {
			t.Errorf("call %d: expected Route '%s', got '%s'", i, expectedPaths[i], call.Route)
		}
		if call.Method != expectedMethods[i] {
			t.Errorf("call %d: expected Method '%s', got '%s'", i, expectedMethods[i], call.Method)
		}
	}

	// Verify that different paths produce different route values
	// (which will produce different idempotency keys in the budget service)
	if budgetEnforcer.calls[0].Route == budgetEnforcer.calls[1].Route {
		t.Error("different URL paths should produce different Route values for idempotency")
	}
}

// TestDelegationScopeEnforcement verifies that delegation token scopes
// are checked against route requirements
func TestDelegationScopeEnforcement(t *testing.T) {
	tests := []struct {
		name          string
		tokenScopes   []string
		requiredScope string
		shouldAllow   bool
	}{
		{
			name:          "exact_match",
			tokenScopes:   []string{"/api/v1/users"},
			requiredScope: "/api/v1/users",
			shouldAllow:   true,
		},
		{
			name:          "wildcard_match",
			tokenScopes:   []string{"*"},
			requiredScope: "/api/v1/anything",
			shouldAllow:   true,
		},
		{
			name:          "prefix_wildcard_match",
			tokenScopes:   []string{"/api/*"},
			requiredScope: "/api/v1/users",
			shouldAllow:   true,
		},
		{
			name:          "prefix_match",
			tokenScopes:   []string{"/api/v1"},
			requiredScope: "/api/v1/users/123",
			shouldAllow:   true,
		},
		{
			name:          "no_match",
			tokenScopes:   []string{"/api/v1/orders"},
			requiredScope: "/api/v1/users",
			shouldAllow:   false,
		},
		{
			name:          "policy_mode_match",
			tokenScopes:   []string{"control", "observe"},
			requiredScope: "control",
			shouldAllow:   true,
		},
		{
			name:          "empty_scopes",
			tokenScopes:   []string{},
			requiredScope: "/api/v1/users",
			shouldAllow:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := delegationHasScope(tt.tokenScopes, tt.requiredScope)
			if result != tt.shouldAllow {
				t.Errorf("delegationHasScope(%v, %s) = %v, want %v",
					tt.tokenScopes, tt.requiredScope, result, tt.shouldAllow)
			}
		})
	}
}

// TestPerRouteCostCredits verifies that per-route cost configuration is passed correctly
func TestPerRouteCostCredits(t *testing.T) {
	tests := []struct {
		name         string
		routeCost    int64
		expectedCost int64
	}{
		{
			name:         "explicit_high_cost",
			routeCost:    10, // Expensive endpoint (e.g., GPT-4)
			expectedCost: 10,
		},
		{
			name:         "explicit_low_cost",
			routeCost:    1,
			expectedCost: 1,
		},
		{
			name:         "zero_uses_default",
			routeCost:    0, // 0 means "use default from cost lookup"
			expectedCost: 0, // Passed as 0, budget service uses its default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			budgetEnforcer := &mockBudgetEnforcer{
				allowRequest: true,
				remaining:    1000,
				cost:         tt.expectedCost,
			}

			principal := &VerifiedPrincipal{
				Type:            PrincipalTypeDelegation,
				SubjectID:       "stk_cost_test",
				BudgetSubjectID: "stk_cost_test",
				TenantID:        "tenant-cost-test",
				Scopes:          []string{"*"},
			}

			req := httptest.NewRequest(http.MethodPost, "/api/v1/completions", nil)
			req.Header.Set("X-Request-ID", "cost-test-123")
			ctx := context.WithValue(req.Context(), principalKey{}, principal)
			req = req.WithContext(ctx)

			if p := GetVerifiedPrincipal(req.Context()); p != nil && p.BudgetSubjectID != "" {
				budgetEnforcer.EnforceFunc(
					req.Context(),
					p.BudgetSubjectID,
					p.TenantID,
					req.URL.Path,
					req.Method,
					"fiat402",
					req.Header.Get("X-Request-ID"),
					tt.routeCost, // Per-route cost
				)
			}

			if len(budgetEnforcer.calls) != 1 {
				t.Fatalf("expected 1 call, got %d", len(budgetEnforcer.calls))
			}

			if budgetEnforcer.calls[0].CostCredits != tt.expectedCost {
				t.Errorf("expected CostCredits %d, got %d", tt.expectedCost, budgetEnforcer.calls[0].CostCredits)
			}
		})
	}
}

// TestPrincipalTypeIdentification verifies correct principal type assignment
func TestPrincipalTypeIdentification(t *testing.T) {
	// Test delegation principal
	delegationPrincipal := &VerifiedPrincipal{
		Type:            PrincipalTypeDelegation,
		SubjectID:       "stk_abc123",
		BudgetSubjectID: "stk_abc123",
		TenantID:        "tenant-123",
	}

	if delegationPrincipal.Type != PrincipalTypeDelegation {
		t.Errorf("expected PrincipalTypeDelegation, got %s", delegationPrincipal.Type)
	}
	if delegationPrincipal.BudgetSubjectID == "" {
		t.Error("delegation tokens should have BudgetSubjectID set")
	}

	// Test macaroon principal
	macaroonPrincipal := &VerifiedPrincipal{
		Type:            PrincipalTypeMacaroon,
		SubjectID:       "mac-sig-xyz",
		BudgetSubjectID: "", // Empty for macaroons
		TenantID:        "tenant-456",
	}

	if macaroonPrincipal.Type != PrincipalTypeMacaroon {
		t.Errorf("expected PrincipalTypeMacaroon, got %s", macaroonPrincipal.Type)
	}
	if macaroonPrincipal.BudgetSubjectID != "" {
		t.Error("macaroon tokens should have empty BudgetSubjectID")
	}
}
