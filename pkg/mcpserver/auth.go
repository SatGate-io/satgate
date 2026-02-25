package mcpserver

import (
	"context"
	"crypto/subtle"
	"fmt"
	"strconv"
	"strings"

	"github.com/satgate-io/satgate/pkg/macaroon"
)

// TokenInfo holds the verified identity from a request's auth token.
type TokenInfo struct {
	// TokenID is a stable identifier for the token (hash of identifier + caveats).
	TokenID string

	// BudgetID is the budget subject — either from a "budget_id" caveat or derived from TokenID.
	BudgetID string
	// BudgetLimit is the token's budget cap (from "budget_limit" caveat), 0 if not set.
	BudgetLimit int64

	// TenantID from the "tenant_id" caveat (for multi-tenant routing).
	TenantID string

	// Scope from the "scope" caveat.
	Scope string

	// DelegationDepth is the max delegation depth allowed (from "delegation_depth" caveat).
	DelegationDepth int
	// DelegationBudget is the max budget per delegation (from "delegation_budget" caveat).
	DelegationBudget int64
	// Depth is the current token's depth in the delegation chain.
	Depth int

	// ParentTokenID from "parent" caveat (for delegated tokens).
	ParentTokenID string

	// Raw macaroon (for delegation).
	Raw *macaroon.Macaroon

	// RawToken is the encoded token string (for re-delegation).
	RawToken string
}

// RevocationChecker checks if a token has been revoked.
type RevocationChecker interface {
	IsRevoked(ctx context.Context, budgetID string) bool
}

// Authenticator verifies tokens on incoming requests.
type Authenticator interface {
	// Verify checks a token and returns the identity, or error if invalid.
	Verify(ctx context.Context, token string) (*TokenInfo, error)
}

// --- No-auth (for local/dev use) ---

// NoAuthAuthenticator always succeeds with a default identity.
type NoAuthAuthenticator struct {
	DefaultTokenID string
}

func (a *NoAuthAuthenticator) Verify(_ context.Context, _ string) (*TokenInfo, error) {
	tid := a.DefaultTokenID
	if tid == "" {
		tid = "default"
	}
	return &TokenInfo{
		TokenID:  tid,
		BudgetID: tid,
		Scope:    "*",
	}, nil
}

// --- Static token auth (config file) ---

// StaticTokenAuthenticator verifies against a single configured token.
type StaticTokenAuthenticator struct {
	Token   string
	TokenID string
}

func (a *StaticTokenAuthenticator) Verify(_ context.Context, token string) (*TokenInfo, error) {
	if token == "" {
		return nil, fmt.Errorf("token required")
	}
	if subtle.ConstantTimeCompare([]byte(token), []byte(a.Token)) != 1 {
		return nil, fmt.Errorf("invalid token")
	}
	return &TokenInfo{
		TokenID:  a.TokenID,
		BudgetID: a.TokenID,
		Scope:    "*",
	}, nil
}

// --- Macaroon auth (per-request verification) ---

// MacaroonAuthenticator verifies macaroon tokens using the SatGate macaroon service.
type MacaroonAuthenticator struct {
	Service *macaroon.Service
}

func (a *MacaroonAuthenticator) Verify(_ context.Context, token string) (*TokenInfo, error) {
	if token == "" {
		return nil, fmt.Errorf("token required")
	}

	// Strip "Bearer " prefix if present
	token = strings.TrimPrefix(token, "Bearer ")

	mac, err := a.Service.Verify(token)
	if err != nil {
		return nil, fmt.Errorf("invalid macaroon: %w", err)
	}

	// TokenID must be unique per token — use identifier + signature
	// (delegated tokens share the same identifier but have different signatures)
	tokenID := hashToken(mac.Identifier + mac.Signature)

	info := &TokenInfo{
		TokenID:  tokenID,
		Scope:    mac.GetScope(),
		Raw:      mac,
		RawToken: token,
	}

	// Check for budget_id caveat.
	// If present, use it. Otherwise fall back to tokenID for backward compat
	// (OSS delegation doesn't add budget_id caveats).
	if budgetID := mac.GetCaveat("budget_id"); budgetID != "" {
		info.BudgetID = budgetID
	} else {
		info.BudgetID = info.TokenID
	}
	if bl := mac.GetCaveat("budget_limit"); bl != "" {
		if v, err := strconv.ParseFloat(bl, 64); err == nil {
			info.BudgetLimit = int64(v)
		}
	}

	// Check for tenant_id caveat (multi-tenant routing)
	if tenantID := mac.GetCaveat("tenant_id"); tenantID != "" {
		info.TenantID = tenantID
	}

	// Delegation constraints
	if dd := mac.GetCaveat("delegation_depth"); dd != "" {
		if v, err := strconv.Atoi(dd); err == nil {
			info.DelegationDepth = v
		}
	}
	if db := mac.GetCaveat("delegation_budget"); db != "" {
		if v, err := strconv.ParseFloat(db, 64); err == nil {
			info.DelegationBudget = int64(v)
		}
	}

	// Count depth by counting "parent" caveats in the chain
	for _, c := range mac.Caveats {
		if strings.HasPrefix(c, "parent = ") {
			info.Depth++
		}
	}

	// Check for parent caveat
	if parent := mac.GetCaveat("parent"); parent != "" {
		info.ParentTokenID = parent
	}

	return info, nil
}

// NewAuthenticator creates the appropriate authenticator from config.
func NewAuthenticator(cfg AuthConfig) (Authenticator, error) {
	switch cfg.Mode {
	case "none", "":
		return &NoAuthAuthenticator{}, nil

	case "config":
		if cfg.Token == "" {
			return nil, fmt.Errorf("auth.token required for mode=config")
		}
		return &StaticTokenAuthenticator{
			Token:   cfg.Token,
			TokenID: hashToken(cfg.Token),
		}, nil

	case "header":
		if cfg.RootKey == "" {
			return nil, fmt.Errorf("auth.rootKey required for mode=header")
		}
		svc, err := macaroon.NewService(cfg.RootKey)
		if err != nil {
			return nil, fmt.Errorf("macaroon service: %w", err)
		}
		return &MacaroonAuthenticator{Service: svc}, nil

	default:
		return nil, fmt.Errorf("unknown auth mode: %q", cfg.Mode)
	}
}
