// Package l402 provides the L402 payment protocol implementation.
package l402

import (
	"context"
	"net/http"

	"github.com/redis/go-redis/v9"
	"github.com/satgate-io/satgate/internal/lightning"
	"github.com/satgate-io/satgate/internal/macaroon"
)

// Store interface for L402 token storage.
type Store interface {
	SaveToken(ctx context.Context, token *Token) error
	GetToken(ctx context.Context, paymentHash string) (*Token, error)
}

// Token represents an L402 token.
type Token struct {
	PaymentHash string
	Macaroon    string
	Preimage    string
	Paid        bool
}

// RedisStore stores L402 tokens in Redis.
type RedisStore struct {
	client *redis.Client
}

// NewRedisStore creates a new Redis-backed L402 store.
func NewRedisStore(client *redis.Client) *RedisStore {
	return &RedisStore{client: client}
}

// Service handles L402 payment challenges and verification.
type Service struct {
	macaroonSvc *macaroon.Service
	lightning   lightning.Provider
	store       Store
}

// NewService creates a new L402 service.
func NewService(macaroonSvc *macaroon.Service, lightning lightning.Provider) *Service {
	return &Service{
		macaroonSvc: macaroonSvc,
		lightning:   lightning,
	}
}

// NewServiceWithStore creates a new L402 service with a token store.
func NewServiceWithStore(macaroonSvc *macaroon.Service, lightning lightning.Provider, store Store) *Service {
	return &Service{
		macaroonSvc: macaroonSvc,
		lightning:   lightning,
		store:       store,
	}
}

// Challenge represents an L402 challenge response.
type Challenge struct {
	Macaroon       string
	Invoice        string
	PaymentHash    string
	AmountSats     int64
	ExpiresAt      int64
}

// CreateChallenge creates an L402 payment challenge.
func (s *Service) CreateChallenge(ctx context.Context, priceSats int64, scope string) (*Challenge, error) {
	// Create macaroon with payment hash caveat
	// Create lightning invoice
	// Return challenge
	return &Challenge{
		AmountSats: priceSats,
	}, nil
}

// ValidateToken validates an L402 token (macaroon + preimage).
func (s *Service) ValidateToken(ctx context.Context, token string) (*macaroon.VerifiedMacaroon, error) {
	// Parse L402 token format: macaroon:preimage
	// Verify macaroon signature
	// Verify preimage matches payment hash caveat
	return nil, nil
}

// Challenge issues an L402 payment challenge (HTTP response).
func (s *Service) Challenge(w http.ResponseWriter, r *http.Request, priceSats int64) {
	w.Header().Set("WWW-Authenticate", `L402 error="payment required"`)
	w.WriteHeader(http.StatusPaymentRequired)
}

// Verify verifies an L402 payment proof.
func (s *Service) Verify(ctx context.Context, token string) (bool, error) {
	return false, nil
}
