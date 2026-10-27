package macaroon

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	macaroonv2 "gopkg.in/macaroon.v2"
)

// ErrInvalidSignature is wrapped by Verify when a token's signature chain does
// not verify under this service's root key. Test for it with errors.Is.
var ErrInvalidSignature = errors.New("invalid signature")

// ErrRepeatedCaveat is wrapped when a single-valued caveat key appears more
// than once in a token. Test for it with errors.Is.
var ErrRepeatedCaveat = errors.New("single-valued caveat appears more than once")

// ErrReservedCaveat is wrapped when delegation is asked to add a caveat key
// that only the gateway writes. Test for it with errors.Is.
var ErrReservedCaveat = errors.New("caveat key is written only by the gateway")

// Caveat keys the gateway writes into an L402 challenge macaroon. Each is
// single-valued: Verify refuses a token that carries one of them twice, and
// Delegate refuses to add any of them.
const (
	CaveatPaymentHash       = "payment_hash"
	CaveatRoute             = "route"
	CaveatPriceSats         = "price_sats"
	CaveatInvoiceAmountSats = "invoice_amount_sats"
)

// singleValuedCaveats lists every caveat key the gateway reads as one value.
// Keys that attenuation may legitimately repeat (expires, scope, parent) are
// not listed; their readers check or count every occurrence.
var singleValuedCaveats = map[string]bool{
	CaveatPaymentHash:       true,
	CaveatRoute:             true,
	CaveatPriceSats:         true,
	CaveatInvoiceAmountSats: true,
}

// IsSingleValuedCaveat reports whether key may appear at most once in a token.
func IsSingleValuedCaveat(key string) bool {
	return singleValuedCaveats[key]
}

// L402IdentifierPrefix starts the identifier of every macaroon the gateway
// issues in an L402 payment challenge. The identifier is the first input to
// the signature chain, so only a holder of the root key can produce one.
const L402IdentifierPrefix = "satgate-l402-v2:"

// L402Binding is what an L402 challenge macaroon is issued for.
type L402Binding struct {
	PaymentHash string
	Route       string
	PriceSats   int64
	// InvoiceAmountSats is the amount of the invoice, when the provider
	// reports it. Zero leaves the caveat out.
	InvoiceAmountSats int64
}

// parseCaveat splits a "key = value" caveat the way verifyCaveat does. The key
// is trimmed so that spacing variants of a key count as the same key.
func parseCaveat(caveat string) (key, value string, ok bool) {
	parts := strings.SplitN(caveat, " = ", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return strings.TrimSpace(parts[0]), parts[1], true
}

// checkSingleValued refuses a token in which a single-valued key appears more
// than once.
func (m *Macaroon) checkSingleValued() error {
	seen := make(map[string]bool)
	for _, caveat := range m.Caveats {
		key, _, ok := parseCaveat(caveat)
		if !ok || !singleValuedCaveats[key] {
			continue
		}
		if seen[key] {
			return fmt.Errorf("%w: %s", ErrRepeatedCaveat, key)
		}
		seen[key] = true
	}
	return nil
}

// checkDelegatedCaveats refuses caveats that only the gateway may write.
func checkDelegatedCaveats(caveats []string) error {
	for _, caveat := range caveats {
		if key, _, ok := parseCaveat(caveat); ok && singleValuedCaveats[key] {
			return fmt.Errorf("%w: %s", ErrReservedCaveat, key)
		}
	}
	return nil
}

// Macaroon represents a capability token using chained HMAC signatures.
//
// Signature chaining works as follows:
//
//	sig₀ = HMAC(rootKey, identifier)
//	sig₁ = HMAC(sig₀, caveat₁)
//	sig₂ = HMAC(sig₁, caveat₂)
//	...
//	sigₙ = final signature
//
// This enables third-party attenuation: anyone holding a macaroon can add
// caveats by chaining from the current signature without knowing the root key.
// The verifier reconstructs the chain from the root key to verify.
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

	// acceptArgumentRules is set only on the copy returned by
	// AcceptingArgumentRules. A default Service refuses a token that carries
	// argument rules (see ArgumentRulesScopePrefix), because a verifier that
	// does not enforce the rules must not honour the token.
	acceptArgumentRules bool
}

// ArgumentRulesScopePrefix starts every scope word that carries tool argument
// rules, for example "scope = argrules:v1:<base64url JSON>". The rules are not
// a new caveat name on purpose. Unknown caveat names are ignored by every
// verifier built before rules existed, which would drop the limit silently.
// Those verifiers match a scope word to a tool name by exact equality of the
// whole word, so the word denies every real tool. It would allow a tool named
// exactly like the whole word; the word carries a fresh 128-bit nonce (see
// pkg/argrules), so no upstream can advertise that name for a token it never
// saw. New runtimes refuse every tool whose name starts with this prefix
// (IsReservedToolName), in tools/list and in tools/call.
const ArgumentRulesScopePrefix = "argrules:"

// IsReservedToolName reports whether a tool name starts with the reserved
// argument-rules prefix. An upstream tool with such a name is hidden from
// tools/list and refused on tools/call by runtimes that know about rules.
func IsReservedToolName(name string) bool {
	return len(name) >= len(ArgumentRulesScopePrefix) &&
		strings.EqualFold(name[:len(ArgumentRulesScopePrefix)], ArgumentRulesScopePrefix)
}

// IsArgumentRulesScope reports whether any word of a scope caveat value uses
// the reserved argument-rules prefix. A word counts when it appears in a
// comma or space separated list too, so rules cannot hide inside a list.
func IsArgumentRulesScope(value string) bool {
	for _, part := range strings.Split(value, ",") {
		for _, word := range strings.Fields(part) {
			if strings.HasPrefix(word, ArgumentRulesScopePrefix) {
				return true
			}
		}
	}
	return false
}

// AcceptingArgumentRules returns a copy of the service that verifies tokens
// carrying argument rules. Only a caller that goes on to enforce the rules
// (the MCP proxy) may use it. The default Service refuses such tokens, so an
// HTTP route or any other verifier fails closed instead of ignoring the rules.
func (s *Service) AcceptingArgumentRules() *Service {
	if s == nil {
		return nil
	}
	c := *s
	c.acceptArgumentRules = true
	return &c
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

	// Calculate chained signature
	mac.Signature = s.chainedSignature(mac)

	return mac, nil
}

// MintL402 creates the macaroon of an L402 payment challenge. Its identifier
// starts with L402IdentifierPrefix, and it carries the server-written caveats
// payment_hash, route, price_sats and, when known, invoice_amount_sats.
func (s *Service) MintL402(b L402Binding, expiresAt time.Time) (*Macaroon, error) {
	if b.PaymentHash == "" || b.Route == "" || b.PriceSats <= 0 {
		return nil, fmt.Errorf("L402 binding needs a payment hash, a route and a positive price")
	}
	if strings.ContainsAny(b.Route, "\r\n") {
		return nil, fmt.Errorf("L402 route name contains a line break")
	}
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("generate identifier nonce: %w", err)
	}
	mac := &Macaroon{
		Version:    1,
		Location:   "https://satgate.io",
		Identifier: fmt.Sprintf("%s%d:%s", L402IdentifierPrefix, time.Now().UnixMilli(), hex.EncodeToString(nonce)),
		Caveats: []string{
			fmt.Sprintf("expires = %d", expiresAt.UnixMilli()),
			"scope = l402",
		},
	}
	mac.AddCaveat(CaveatPaymentHash, b.PaymentHash)
	mac.AddCaveat(CaveatRoute, b.Route)
	mac.AddCaveat(CaveatPriceSats, strconv.FormatInt(b.PriceSats, 10))
	if b.InvoiceAmountSats > 0 {
		mac.AddCaveat(CaveatInvoiceAmountSats, strconv.FormatInt(b.InvoiceAmountSats, 10))
	}
	mac.Signature = s.chainedSignature(mac)
	return mac, nil
}

// IsL402Challenge reports whether the gateway issued this macaroon in an L402
// payment challenge. Only meaningful on a macaroon returned by Verify.
func (m *Macaroon) IsL402Challenge() bool {
	return strings.HasPrefix(m.Identifier, L402IdentifierPrefix)
}

// Verify validates a macaroon token by reconstructing the HMAC chain
// from the root key and comparing the final signature.
// Accepts both binary (libmacaroon V2) and JSON (custom) formats.
//
// A token in which a single-valued caveat key (see IsSingleValuedCaveat)
// appears more than once is refused.
func (s *Service) Verify(token string) (*Macaroon, error) {
	// Binary L402 macaroons use libmacaroon's canonical signature algorithm;
	// verify them with the library before converting into SatGate's struct shape.
	if mac, err := s.verifyBinary(token); err == nil {
		if err := mac.checkSingleValued(); err != nil {
			return nil, fmt.Errorf("caveat check failed: %w", err)
		}
		return mac, nil
	}

	mac, err := s.Decode(token)
	if err != nil {
		return nil, fmt.Errorf("failed to decode token: %w", err)
	}

	// Reconstruct the chained signature from root key
	expectedSig := s.chainedSignature(mac)
	if !hmac.Equal([]byte(mac.Signature), []byte(expectedSig)) {
		return nil, fmt.Errorf("%w: token was signed with a different root key. Check that the gateway and the token issuer use the same capability root key", ErrInvalidSignature)
	}

	// Check caveats
	for _, caveat := range mac.Caveats {
		if err := s.verifyCaveat(caveat); err != nil {
			return nil, fmt.Errorf("caveat check failed: %w", err)
		}
	}
	if err := mac.checkSingleValued(); err != nil {
		return nil, fmt.Errorf("caveat check failed: %w", err)
	}

	return mac, nil
}

func (s *Service) verifyBinary(token string) (*Macaroon, error) {
	data, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		data, err = base64.RawStdEncoding.DecodeString(token)
		if err != nil {
			return nil, fmt.Errorf("invalid base64: %w", err)
		}
	}

	var v2mac macaroonv2.Macaroon
	if err := v2mac.UnmarshalBinary(data); err != nil {
		return nil, fmt.Errorf("invalid binary macaroon: %w", err)
	}
	if err := v2mac.Verify(s.rootKey, s.verifyCaveat, nil); err != nil {
		return nil, err
	}

	return s.decodeVerifiedBinaryMacaroon(&v2mac), nil
}

// Delegate creates a child macaroon with additional caveats using proper
// HMAC chaining. The new caveats are chained from the parent's signature,
// meaning the delegator does NOT need the root key — only the current token.
//
// This is the core macaroon attenuation property: permissions can only be
// narrowed, never widened, because caveats are cumulative and signatures
// are cryptographically chained.
//
// Caveat keys that only the gateway writes (see IsSingleValuedCaveat) are
// refused in additionalCaveats.
func (s *Service) Delegate(parentToken string, additionalCaveats []string) (*Macaroon, error) {
	if err := checkDelegatedCaveats(additionalCaveats); err != nil {
		return nil, err
	}
	parent, err := s.Decode(parentToken)
	if err != nil {
		return nil, fmt.Errorf("failed to decode parent: %w", err)
	}

	// Verify parent first — only valid tokens can be delegated
	if _, err := s.Verify(parentToken); err != nil {
		return nil, fmt.Errorf("invalid parent token: %w", err)
	}

	// Create child with parent's identifier, all parent caveats, plus new ones
	child := &Macaroon{
		Version:    parent.Version,
		Location:   parent.Location,
		Identifier: parent.Identifier,
		Caveats:    append(append([]string{}, parent.Caveats...), additionalCaveats...),
	}

	// Chain signature: start from parent's signature and chain through new caveats.
	// This is equivalent to reconstructing from root key through ALL caveats,
	// but demonstrates that delegation only needs the parent signature.
	sig, err := hex.DecodeString(parent.Signature)
	if err != nil {
		return nil, fmt.Errorf("invalid parent signature: %w", err)
	}
	for _, caveat := range additionalCaveats {
		mac := hmac.New(sha256.New, sig)
		mac.Write([]byte(caveat))
		sig = mac.Sum(nil)
	}
	child.Signature = hex.EncodeToString(sig)

	return child, nil
}

// DelegateWithoutVerify creates an attenuated child macaroon without
// requiring the root key for verification. This enables true third-party
// delegation: any token holder can add caveats by chaining from their
// token's signature.
//
// WARNING: The caller is responsible for ensuring the parent token is valid.
// Use Delegate() when the service has the root key available.
//
// This mirrors what any holder can do without the root key, so it does not
// refuse caveat keys. A token that ends up repeating a single-valued key is
// refused by Verify, and the L402 path accepts only macaroons the gateway
// issued as challenges, which already carry every server-written L402 key.
func (s *Service) DelegateWithoutVerify(parentToken string, additionalCaveats []string) (*Macaroon, error) {
	parent, err := s.Decode(parentToken)
	if err != nil {
		return nil, fmt.Errorf("failed to decode parent: %w", err)
	}

	child := &Macaroon{
		Version:    parent.Version,
		Location:   parent.Location,
		Identifier: parent.Identifier,
		Caveats:    append(append([]string{}, parent.Caveats...), additionalCaveats...),
	}

	// Chain from parent signature through new caveats
	sig, err := hex.DecodeString(parent.Signature)
	if err != nil {
		return nil, fmt.Errorf("invalid parent signature: %w", err)
	}
	for _, caveat := range additionalCaveats {
		mac := hmac.New(sha256.New, sig)
		mac.Write([]byte(caveat))
		sig = mac.Sum(nil)
	}
	child.Signature = hex.EncodeToString(sig)

	return child, nil
}

// EncodeBinary converts our Macaroon to a gopkg.in/macaroon.v2 binary-serialized
// base64 string (standard libmacaroon V2 format). This is the format expected by
// lnget, Aperture, and other standard L402 clients.
func (s *Service) EncodeBinary(mac *Macaroon) (string, error) {
	// Create a new v2 macaroon using our root key — this computes sig₀ = HMAC(rootKey, id)
	v2mac, err := macaroonv2.New(s.rootKey, []byte(mac.Identifier), mac.Location, macaroonv2.V2)
	if err != nil {
		return "", fmt.Errorf("failed to create v2 macaroon: %w", err)
	}

	// Add each caveat — this chains the HMAC signature identically to our implementation
	for _, caveat := range mac.Caveats {
		if err := v2mac.AddFirstPartyCaveat([]byte(caveat)); err != nil {
			return "", fmt.Errorf("failed to add caveat: %w", err)
		}
	}

	// Binary serialize
	data, err := v2mac.MarshalBinary()
	if err != nil {
		return "", fmt.Errorf("failed to marshal binary: %w", err)
	}

	return base64.StdEncoding.EncodeToString(data), nil
}

// DecodeBinary deserializes a base64 standard-format binary macaroon into our Macaroon struct.
func (s *Service) DecodeBinary(token string) (*Macaroon, error) {
	data, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		// Try raw standard base64
		data, err = base64.RawStdEncoding.DecodeString(token)
		if err != nil {
			return nil, fmt.Errorf("invalid base64: %w", err)
		}
	}

	var v2mac macaroonv2.Macaroon
	if err := v2mac.UnmarshalBinary(data); err != nil {
		return nil, fmt.Errorf("invalid binary macaroon: %w", err)
	}

	return s.decodeVerifiedBinaryMacaroon(&v2mac), nil
}

func (s *Service) decodeVerifiedBinaryMacaroon(v2mac *macaroonv2.Macaroon) *Macaroon {
	mac := &Macaroon{
		Version:    2,
		Location:   v2mac.Location(),
		Identifier: string(v2mac.Id()),
		Caveats:    make([]string, 0),
		Signature:  hex.EncodeToString(v2mac.Signature()),
	}

	for _, cav := range v2mac.Caveats() {
		mac.Caveats = append(mac.Caveats, string(cav.Id))
	}

	return mac
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

// chainedSignature computes the chained HMAC signature for a macaroon.
//
//	sig₀ = HMAC(rootKey, identifier)
//	sigₙ = HMAC(sigₙ₋₁, caveatₙ)
func (s *Service) chainedSignature(mac *Macaroon) string {
	// sig₀ = HMAC(rootKey, identifier)
	h := hmac.New(sha256.New, s.rootKey)
	h.Write([]byte(mac.Identifier))
	sig := h.Sum(nil)

	// sigₙ = HMAC(sigₙ₋₁, caveatₙ)
	for _, caveat := range mac.Caveats {
		h = hmac.New(sha256.New, sig)
		h.Write([]byte(caveat))
		sig = h.Sum(nil)
	}

	return hex.EncodeToString(sig)
}

// RecalculateSignature recalculates the chained signature for a macaroon.
// This MUST be called after adding caveats to ensure signature validity.
func (s *Service) RecalculateSignature(mac *Macaroon) string {
	return s.chainedSignature(mac)
}

// verifyCaveat checks if a caveat is satisfied
func (s *Service) verifyCaveat(caveat string) error {
	// Handle "time < N" caveats (legacy format from enterprise Mint)
	if strings.HasPrefix(caveat, "time < ") {
		val := strings.TrimPrefix(caveat, "time < ")
		var ts int64
		if _, err := fmt.Sscanf(val, "%d", &ts); err == nil {
			if time.Now().Unix() > ts {
				return fmt.Errorf("token expired")
			}
			return nil
		}
		return fmt.Errorf("invalid time caveat: %s", caveat)
	}

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
		// Scope is checked at request time, not here. Argument rules ride in
		// a scope word, and only an enforcing verifier may accept them.
		if !s.acceptArgumentRules && IsArgumentRulesScope(value) {
			return fmt.Errorf("token carries tool argument rules this verifier does not enforce")
		}
		return nil

	default:
		// Unknown caveats are ignored (forward compatibility)
		return nil
	}
}

// HasScope reports whether the macaroon grants requiredScope.
//
// Every scope caveat must grant it. An additional scope caveat can only narrow
// access. Each caveat is compared as one string. A token with no scope caveat
// keeps full access.
func (m *Macaroon) HasScope(requiredScope string) bool {
	for _, caveat := range m.Caveats {
		if !strings.HasPrefix(caveat, "scope = ") {
			continue
		}
		if !scopeMatches(strings.TrimPrefix(caveat, "scope = "), requiredScope) {
			return false
		}
	}
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

// GetScope returns the last scope caveat, for display. Scope caveats repeat
// under attenuation; an access decision must use HasScope, which checks every
// one of them.
func (m *Macaroon) GetScope() string {
	return m.GetCaveat("scope")
}

// Scope is a convenience property that returns the scope
var _ = (*Macaroon)(nil) // Ensure Macaroon type exists

// AddCaveat adds a new caveat to the macaroon
// Note: After adding caveats, call RecalculateSignature to update the signature.
func (m *Macaroon) AddCaveat(key, value string) {
	m.Caveats = append(m.Caveats, fmt.Sprintf("%s = %s", key, value))
}

// SingleCaveat returns the value of a caveat key that must appear at most
// once. found is false when the key is absent. err wraps ErrRepeatedCaveat
// when the key appears more than once, in any spacing variant.
func (m *Macaroon) SingleCaveat(key string) (value string, found bool, err error) {
	for _, caveat := range m.Caveats {
		k, v, ok := parseCaveat(caveat)
		if !ok || k != key {
			continue
		}
		if found {
			return "", false, fmt.Errorf("%w: %s", ErrRepeatedCaveat, key)
		}
		value, found = v, true
	}
	return value, found, nil
}

// GetCaveat retrieves a caveat value by key.
//
// For a single-valued key (see IsSingleValuedCaveat) it returns the one value,
// or "" when the key is absent or appears more than once. For any other key it
// returns the last match: for "parent" that is the nearest delegation hop.
// Spend does not use this for budget_id; see ResolveIssuedBudgetID.
// Delegation caps use the narrowest positive value.
func (m *Macaroon) GetCaveat(key string) string {
	if singleValuedCaveats[key] {
		value, _, err := m.SingleCaveat(key)
		if err != nil {
			return ""
		}
		return value
	}
	prefix := key + " = "
	result := ""
	for _, caveat := range m.Caveats {
		if strings.HasPrefix(caveat, prefix) {
			result = strings.TrimPrefix(caveat, prefix)
		}
	}
	return result
}

// HasCaveat reports whether a caveat with the given key and a non-empty value
// is present.
func (m *Macaroon) HasCaveat(key string) bool {
	for _, caveat := range m.Caveats {
		if k, v, ok := parseCaveat(caveat); ok && k == key && v != "" {
			return true
		}
	}
	return false
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
		// Recalculate chained signature after adding caveats
		wrapper.mac.Signature = w.svc.chainedSignature(wrapper.mac)
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
