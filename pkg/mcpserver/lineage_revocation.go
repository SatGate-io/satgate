package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"

	"github.com/rs/zerolog/log"

	"github.com/satgate-io/satgate/pkg/macaroon"
)

// LineageRevocationChecker reports whether a ban covers a token: whether any
// of signatures, the token's lineage (macaroon.Service.Lineage: its own
// signature and the signature of every token it was delegated from), is
// banned. The gateway's *governance.Service satisfies it, so the MCP runtime
// honours the bans written through the gateway's ban API.
//
// An error means the ban state could not be read; the token is then refused
// (fail closed) with the revocation-unavailable answer.
type LineageRevocationChecker interface {
	IsAnyBanned(ctx context.Context, signatures []string) (bool, error)
}

// RevocationConfig says where satgate-mcp reads token bans from (see
// Proxy.SetLineageRevocationChecker).
type RevocationConfig struct {
	// Redis is the shared ban store: the Redis a SatGate gateway with
	// redis.enabled writes its bans to. Bans written by any gateway on it
	// are then honoured here within 2 seconds. With no address, bans are
	// kept in this process's memory and apply to this process only.
	Redis RevocationRedisConfig `yaml:"redis,omitempty"`
}

// RevocationRedisConfig is the address of the shared ban store.
type RevocationRedisConfig struct {
	Addr     string `yaml:"addr,omitempty"`     // host:port; empty means no shared store
	Password string `yaml:"password,omitempty"` // empty for no auth
	DB       int    `yaml:"db,omitempty"`
}

// ErrRevocationUnavailable is wrapped by the error a token check returns when
// the ban state could not be read. The request is refused, never served.
var ErrRevocationUnavailable = errors.New("revocation state unavailable")

// errTokenRevoked is the refusal of a token a ban covers. Its text is the one
// the transports already treat as a definitive refusal ("token revoked").
var errTokenRevoked = errors.New("token revoked")

// revocationUnavailableError wraps a ban-state read failure.
type revocationUnavailableError struct{ cause error }

func (e *revocationUnavailableError) Error() string {
	return ErrRevocationUnavailable.Error() + ": " + e.cause.Error()
}

func (e *revocationUnavailableError) Is(target error) bool {
	return target == ErrRevocationUnavailable
}

func (e *revocationUnavailableError) Unwrap() error { return e.cause }

// lineageGate holds the proxy's LineageRevocationChecker. One gate is shared
// by the proxy's authenticator and its delegator, so a checker set at any
// time applies to every token they accept from then on.
type lineageGate struct {
	mu      sync.RWMutex
	checker LineageRevocationChecker
	// svc computes the lineage when the verifier's own service is not
	// known (the header-mode service built by New).
	svc *macaroon.Service
}

func (g *lineageGate) set(checker LineageRevocationChecker) {
	g.mu.Lock()
	g.checker = checker
	g.mu.Unlock()
}

// check returns nil when no ban covers the verified token info, an error
// wrapping errTokenRevoked when one does, and an error wrapping
// ErrRevocationUnavailable when the ban state cannot be read. verifier is the
// macaroon service that verified the token, when known. A token that is not
// a macaroon (static token, no auth) is not a ban subject and passes.
func (g *lineageGate) check(ctx context.Context, info *TokenInfo, verifier *macaroon.Service) error {
	if g == nil || info == nil || info.Raw == nil {
		return nil
	}
	g.mu.RLock()
	checker, svc := g.checker, g.svc
	g.mu.RUnlock()
	if checker == nil {
		return nil
	}
	if verifier != nil {
		svc = verifier
	}
	if svc == nil {
		log.Error().Msg("MCP revocation: no root key to compute the token's lineage; refusing")
		return fmt.Errorf("%w: lineage cannot be computed", errTokenRevoked)
	}
	lineage, err := svc.Lineage(info.Raw)
	if err != nil {
		// The token verified, so this does not happen with the verifier's
		// own key; refuse rather than serve a token whose ancestors cannot
		// be named.
		log.Error().Err(err).Msg("MCP revocation: token lineage could not be computed; refusing")
		return fmt.Errorf("%w: lineage cannot be computed", errTokenRevoked)
	}
	banned, err := checker.IsAnyBanned(ctx, lineage)
	if err != nil {
		log.Error().Err(err).Msg("MCP revocation: ban state unavailable; refusing")
		return &revocationUnavailableError{cause: err}
	}
	if banned {
		return errTokenRevoked
	}
	return nil
}

// macaroonServiceOf is the macaroon service an authenticator verifies with,
// or nil when it is not a MacaroonAuthenticator.
func macaroonServiceOf(a Authenticator) *macaroon.Service {
	if m, ok := a.(*MacaroonAuthenticator); ok && m != nil {
		return m.Service
	}
	return nil
}

// SetLineageRevocationChecker makes the proxy refuse every macaroon a ban
// covers: the banned token and every token delegated from it, at any depth,
// whichever way it was delegated (satgate/delegate, the gateway, or a holder
// adding caveats offline). The check runs inside the proxy's authenticator,
// after the signature and resource checks, so every place the runtime
// accepts a token runs it before anything is admitted, spent, forwarded or
// minted: SSE connect and each message, Streamable HTTP initialize, each
// request and resume, stdio, the credential passed to an upstream with
// forwardCallerToken, and satgate/delegate (checked again before the child is
// minted). Over stdio, tools/list and the methods passed on to the default
// upstream answer a request that supplies no token as before; one that
// supplies a token (params._meta.token, or the connection's) is verified and
// checked first.
//
// A covered token is refused like a revoked one ("token revoked"). If the
// ban state cannot be read the request is refused with the
// revocation-unavailable answer: HTTP 503 where the refusal is an HTTP
// answer, else the JSON-RPC error with data.error "revocation_unavailable".
//
// Pass the gateway's governance service; with Redis it is shared with every
// gateway on that Redis. Nil turns the check off (the default). It applies to
// macaroons only: a static token or the no-auth identity is not a ban
// subject. The lineage is recomputed with the root key of the
// MacaroonAuthenticator that verified the token (or auth.rootKey); a token
// accepted by another authenticator whose chain does not follow from that key
// is refused. Call it before the proxy serves requests.
func (p *Proxy) SetLineageRevocationChecker(rc LineageRevocationChecker) {
	p.lineageGate().set(rc)
}

// lineageCheckerInstalled reports whether a LineageRevocationChecker is set.
func (p *Proxy) lineageCheckerInstalled() bool {
	g := p.lineage
	if g == nil {
		return false
	}
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.checker != nil
}

// lineageGate returns the proxy's gate, creating it for a Proxy not built by
// New.
func (p *Proxy) lineageGate() *lineageGate {
	if p.lineage == nil {
		p.lineage = &lineageGate{}
		if p.delegator != nil {
			p.delegator.lineage = p.lineage
		}
		if rb, ok := p.auth.(*ResourceBoundAuthenticator); ok {
			rb.lineage = p.lineage
		}
	}
	return p.lineage
}

// bindAuth binds a to the proxy's resource and its lineage gate.
func (p *Proxy) bindAuth(a Authenticator) Authenticator {
	bound := BindResource(a, p.resource)
	if rb, ok := bound.(*ResourceBoundAuthenticator); ok && p.lineage != nil {
		rb.lineage = p.lineage
	}
	return bound
}

// revocationUnavailableData is data.error of the JSON-RPC refusal of a
// request whose token's ban state could not be read.
const revocationUnavailableData = "revocation_unavailable"

// revocationUnavailableResponse is the JSON-RPC refusal of a request whose
// token's ban state could not be read.
func revocationUnavailableResponse(id json.RawMessage) *Response {
	return NewErrorResponseWithData(id, CodeInternalError, "Revocation status unavailable", map[string]interface{}{
		"error": revocationUnavailableData,
		"code":  "REVOCATION_UNAVAILABLE",
	})
}

// isRevocationUnavailableResponse reports whether resp is that refusal.
func isRevocationUnavailableResponse(resp *Response) bool {
	if resp == nil || resp.Error == nil || len(resp.Error.Data) == 0 {
		return false
	}
	var data struct {
		Error string `json:"error"`
	}
	return json.Unmarshal(resp.Error.Data, &data) == nil && data.Error == revocationUnavailableData
}

// writeRevocationUnavailable answers an HTTP request whose token's ban state
// could not be read: 503, as the HTTP gateway does.
func writeRevocationUnavailable(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusServiceUnavailable)
	_, _ = w.Write([]byte(`{"error":"revocation_unavailable","code":"REVOCATION_UNAVAILABLE","message":"Revocation status unavailable"}` + "\n"))
}

// isRevocationUnavailable reports whether err says the token's ban state
// could not be read.
func isRevocationUnavailable(err error) bool {
	return errors.Is(err, ErrRevocationUnavailable)
}
