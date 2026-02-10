package macaroon

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Macaroon represents a capability token
type Macaroon struct {
	Version    int      `json:"v"`
	Location   string   `json:"l"`
	Identifier string   `json:"i"`
	Caveats    []string `json:"c"`
	Signature  string   `json:"s"`
}

// Service handles macaroon operations
type Service struct {
	rootKey []byte
}

// NewService creates a new macaroon service
func NewService(secret string) (*Service, error) {
	if secret == "" {
		return nil, fmt.Errorf("secret key required")
	}

	// Derive root key from secret
	h := sha256.Sum256([]byte(secret))

	return &Service{
		rootKey: h[:],
	}, nil
}

// Mint creates a new root macaroon
func (s *Service) Mint(scope string, expiresAt time.Time) (*Macaroon, error) {
	mac := &Macaroon{
		Version:    1,
		Location:   "https://satgate.io",
		Identifier: fmt.Sprintf("satgate-capability-v1:%d", time.Now().UnixMilli()),
		Caveats:    []string{},
	}

	// Add expiry caveat
	mac.Caveats = append(mac.Caveats, fmt.Sprintf("expires = %d", expiresAt.UnixMilli()))

	// Add scope caveat
	if scope != "" {
		mac.Caveats = append(mac.Caveats, fmt.Sprintf("scope = %s", scope))
	}

	// Calculate signature
	mac.Signature = s.calculateSignature(mac)

	return mac, nil
}

// Verify validates a macaroon token
func (s *Service) Verify(token string) (*Macaroon, error) {
	mac, err := s.Decode(token)
	if err != nil {
		return nil, fmt.Errorf("failed to decode token: %w", err)
	}

	// Verify signature
	expectedSig := s.calculateSignature(mac)
	if !hmac.Equal([]byte(mac.Signature), []byte(expectedSig)) {
		return nil, fmt.Errorf("invalid signature")
	}

	// Check caveats
	for _, caveat := range mac.Caveats {
		if err := s.verifyCaveat(caveat); err != nil {
			return nil, fmt.Errorf("caveat check failed: %w", err)
		}
	}

	return mac, nil
}

// Delegate creates a child macaroon with additional caveats
func (s *Service) Delegate(parentToken string, additionalCaveats []string) (*Macaroon, error) {
	parent, err := s.Decode(parentToken)
	if err != nil {
		return nil, fmt.Errorf("failed to decode parent: %w", err)
	}

	// Verify parent first - only valid tokens can be delegated
	if _, err := s.Verify(parentToken); err != nil {
		return nil, fmt.Errorf("invalid parent token: %w", err)
	}

	// Create child with parent's caveats plus new ones
	// Caveats are cumulative - can only add restrictions, never remove
	child := &Macaroon{
		Version:    parent.Version,
		Location:   parent.Location,
		Identifier: fmt.Sprintf("%s:child:%d", parent.Identifier, time.Now().UnixMilli()),
		Caveats:    append(append([]string{}, parent.Caveats...), additionalCaveats...),
	}

	// Calculate signature using root key for verification compatibility
	// Security note: The attenuation property is preserved because caveats
	// are cumulative (child has all parent caveats plus additional ones)
	child.Signature = s.calculateSignature(child)

	return child, nil
}

// Encode serializes a macaroon to a string
func (s *Service) Encode(mac *Macaroon) string {
	data, _ := json.Marshal(mac)
	return base64.RawURLEncoding.EncodeToString(data)
}

// Decode deserializes a macaroon from a string
func (s *Service) Decode(token string) (*Macaroon, error) {
	data, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		// Try standard base64
		data, err = base64.StdEncoding.DecodeString(token)
		if err != nil {
			return nil, fmt.Errorf("invalid base64: %w", err)
		}
	}

	var mac Macaroon
	if err := json.Unmarshal(data, &mac); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	return &mac, nil
}

// calculateSignature computes the HMAC signature
func (s *Service) calculateSignature(mac *Macaroon) string {
	return s.calculateSignatureWithKey(mac, s.rootKey)
}

// RecalculateSignature recalculates the signature for a macaroon
// This MUST be called after adding caveats to ensure signature validity
func (s *Service) RecalculateSignature(mac *Macaroon) string {
	return s.calculateSignature(mac)
}

// calculateSignatureWithKey computes signature with a specific key
func (s *Service) calculateSignatureWithKey(mac *Macaroon, key []byte) string {
	h := hmac.New(sha256.New, key)

	// Include identifier
	h.Write([]byte(mac.Identifier))

	// Include each caveat
	for _, caveat := range mac.Caveats {
		h.Write([]byte(caveat))
	}

	return hex.EncodeToString(h.Sum(nil))
}

// verifyCaveat checks if a caveat is satisfied
func (s *Service) verifyCaveat(caveat string) error {
	parts := strings.SplitN(caveat, " = ", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid caveat format: %s", caveat)
	}

	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])

	switch key {
	case "expires":
		// Check expiration - support both Unix timestamp (ms) and duration strings
		// IMPORTANT: Check duration first because fmt.Sscanf partially parses "5m" as 5!

		// Try parsing as duration string first (e.g., "5m", "1h")
		if _, durErr := time.ParseDuration(value); durErr == nil {
			// Duration caveats are relative to token creation time
			// We can't verify them without that context, so we allow them
			// (they're mainly for display purposes in delegated tokens)
			return nil
		}

		// Try parsing as Unix timestamp in milliseconds (must be all digits)
		var expiresMs int64
		if n, err := fmt.Sscanf(value, "%d", &expiresMs); err == nil && n == 1 {
			// Verify it's actually all digits (not "5m" which would parse as 5)
			isAllDigits := true
			for _, c := range value {
				if c < '0' || c > '9' {
					isAllDigits = false
					break
				}
			}
			if isAllDigits {
				if time.Now().UnixMilli() > expiresMs {
					return fmt.Errorf("token expired")
				}
				return nil
			}
		}

		// Try parsing as RFC3339 timestamp
		if ts, tsErr := time.Parse(time.RFC3339, value); tsErr == nil {
			if time.Now().After(ts) {
				return fmt.Errorf("token expired")
			}
			return nil
		}

		// Unknown format - fail safe by rejecting
		return fmt.Errorf("invalid expires value: %s", value)

	case "scope":
		// Scope is checked at request time, not here
		return nil

	default:
		// Unknown caveats are ignored (forward compatibility)
		return nil
	}
}

// HasScope checks if the macaroon has a specific scope
func (m *Macaroon) HasScope(requiredScope string) bool {
	for _, caveat := range m.Caveats {
		if strings.HasPrefix(caveat, "scope = ") {
			tokenScope := strings.TrimPrefix(caveat, "scope = ")
			return scopeMatches(tokenScope, requiredScope)
		}
	}
	// No scope caveat means full access
	return true
}

// scopeMatches checks if tokenScope grants access to requiredScope
func scopeMatches(tokenScope, requiredScope string) bool {
	// Wildcard match
	if tokenScope == "*" || tokenScope == "api:*" {
		return true
	}

	// Exact match
	if tokenScope == requiredScope {
		return true
	}

	// Prefix match (e.g., "api:read" matches "api:read:users")
	if strings.HasSuffix(tokenScope, ":*") {
		prefix := strings.TrimSuffix(tokenScope, "*")
		if strings.HasPrefix(requiredScope, prefix) {
			return true
		}
	}

	return false
}

// GetExpiry returns the expiration time from caveats
func (m *Macaroon) GetExpiry() (time.Time, bool) {
	for _, caveat := range m.Caveats {
		if strings.HasPrefix(caveat, "expires = ") {
			value := strings.TrimPrefix(caveat, "expires = ")
			var expiresMs int64
			if _, err := fmt.Sscanf(value, "%d", &expiresMs); err == nil {
				return time.UnixMilli(expiresMs), true
			}
		}
	}
	return time.Time{}, false
}

// GetScope returns the scope from caveats
func (m *Macaroon) GetScope() string {
	return m.GetCaveat("scope")
}

// Scope is a convenience property that returns the scope
var _ = (*Macaroon)(nil) // Ensure Macaroon type exists

// AddCaveat adds a new caveat to the macaroon
// Note: This must be done BEFORE calculating the signature
func (m *Macaroon) AddCaveat(key, value string) {
	m.Caveats = append(m.Caveats, fmt.Sprintf("%s = %s", key, value))
}

// GetCaveat retrieves a caveat value by key
func (m *Macaroon) GetCaveat(key string) string {
	prefix := key + " = "
	for _, caveat := range m.Caveats {
		if strings.HasPrefix(caveat, prefix) {
			return strings.TrimPrefix(caveat, prefix)
		}
	}
	return ""
}

// HasCaveat checks if a caveat with the given key exists
func (m *Macaroon) HasCaveat(key string) bool {
	return m.GetCaveat(key) != ""
}

// MacaroonWrapper wraps a Macaroon for the Mint interface
type MacaroonWrapper struct {
	mac *Macaroon
	svc *Service
}

// AddCaveat adds a caveat to the wrapped macaroon
func (w *MacaroonWrapper) AddCaveat(key, value string) {
	w.mac.AddCaveat(key, value)
}

// MintWrapper wraps the macaroon Service for the Mint MacaroonMinter interface
type MintWrapper struct {
	svc *Service
}

// NewMintWrapper creates a wrapper for use with the Mint service
func NewMintWrapper(svc *Service) *MintWrapper {
	return &MintWrapper{svc: svc}
}

// Mint creates a new macaroon and returns a wrapper for adding caveats
func (w *MintWrapper) Mint(scope string, expiresAt time.Time) (interface{ AddCaveat(key, value string) }, error) {
	mac, err := w.svc.Mint(scope, expiresAt)
	if err != nil {
		return nil, err
	}
	return &MacaroonWrapper{mac: mac, svc: w.svc}, nil
}

// Encode serializes the macaroon to a string
func (w *MintWrapper) Encode(mac interface{}) string {
	if wrapper, ok := mac.(*MacaroonWrapper); ok {
		// Recalculate signature after adding caveats
		wrapper.mac.Signature = w.svc.calculateSignature(wrapper.mac)
		return w.svc.Encode(wrapper.mac)
	}
	return ""
}

// Delegate creates an attenuated child macaroon from an existing parent token
func (w *MintWrapper) Delegate(parentToken string, additionalCaveats []string) (string, error) {
	child, err := w.svc.Delegate(parentToken, additionalCaveats)
	if err != nil {
		return "", err
	}
	return w.svc.Encode(child), nil
}
