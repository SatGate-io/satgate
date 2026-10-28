package proxy

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/rs/zerolog/log"

	"github.com/satgate-io/satgate/pkg/macaroon"
)

// errTokenRevoked is returned when a ban covers the presented token: a ban of
// the token itself or of any token it was delegated from.
var errTokenRevoked = errors.New("token revoked")

// errRevocationUnavailable is returned when the shared ban store cannot be
// read. The request is refused with 503, never served.
var errRevocationUnavailable = errors.New("revocation state unavailable")

// checkRevoked returns nil when no ban covers mac, errTokenRevoked when one
// does, and an error wrapping errRevocationUnavailable when the ban store
// cannot be read. mac must have been returned by g.macaroonSvc.Verify.
//
// A ban covers a token and every token delegated from it, at any depth,
// whether it was delegated before or after the ban. The check uses the
// token's lineage (macaroon.Service.Lineage): the signature chain recomputed
// from the root key, whose prefixes are exactly the signatures of the tokens
// it was delegated from.
func (g *Gateway) checkRevoked(ctx context.Context, mac *macaroon.Macaroon) error {
	if g.governance == nil {
		return nil
	}
	lineage, err := g.macaroonSvc.Lineage(mac)
	if err != nil {
		// Verify accepted mac, so this does not happen; refuse rather than
		// serve a token whose ancestors cannot be named.
		log.Error().Err(err).Msg("Revocation: token lineage could not be computed; refusing")
		return fmt.Errorf("%w: %v", errTokenRevoked, err)
	}
	banned, err := g.governance.IsAnyBanned(ctx, lineage)
	if err != nil {
		// Any error reading the ban state (governance.ErrRevocationUnavailable
		// or otherwise) refuses the request.
		return fmt.Errorf("%w: %v", errRevocationUnavailable, err)
	}
	if banned {
		return errTokenRevoked
	}
	return nil
}

// writeRevocationUnavailableJSON answers an API request whose token's
// revocation state could not be read.
func writeRevocationUnavailableJSON(w http.ResponseWriter) {
	writeJSON(w, http.StatusServiceUnavailable, map[string]string{
		"error": "Revocation status unavailable",
		"code":  "REVOCATION_UNAVAILABLE",
	})
}

// writeTokenBannedJSON answers an API request made with a revoked token.
func writeTokenBannedJSON(w http.ResponseWriter, status int) {
	writeJSON(w, status, map[string]interface{}{
		"error":  "Token Revoked",
		"reason": "This token has been banned by an administrator",
		"code":   "TOKEN_BANNED",
	})
}
