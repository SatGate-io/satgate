package macaroon

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	macaroonv2 "gopkg.in/macaroon.v2"
)

// ErrNotIssuedHere is returned by Lineage when a macaroon's signature does not
// follow from this service's root key, so its ancestors cannot be named.
var ErrNotIssuedHere = errors.New("macaroon signature does not follow from this service's root key")

// Lineage returns the signature of mac and of every token it was delegated
// from, oldest first: element k is the chain signature after the first k
// caveats, and the last element is mac.Signature.
//
// Delegation (Delegate, DelegateWithoutVerify, or a holder appending
// first-party caveats offline) keeps the identifier and every caveat of the
// parent and chains new caveats from the parent's signature. So the signature
// of each ancestor is one of the prefix signatures of the child, and nothing
// else is: a sibling shares only the ancestors both were delegated from, and a
// parent never contains its child's signature. A holder cannot change a prefix
// without changing the final signature, which Verify would then refuse.
//
// The chain is recomputed from the root key, in the format mac verified under
// (JSON tokens use SatGate's chain; binary tokens use the libmacaroon chain).
// It returns ErrNotIssuedHere when neither chain ends at mac.Signature. Call it
// only on a macaroon returned by Verify.
func (s *Service) Lineage(mac *Macaroon) ([]string, error) {
	if mac == nil {
		return nil, fmt.Errorf("%w: nil macaroon", ErrNotIssuedHere)
	}
	if chain := s.jsonChain(mac); hmac.Equal([]byte(chain[len(chain)-1]), []byte(mac.Signature)) {
		return chain, nil
	}
	chain, err := s.binaryChain(mac)
	if err == nil && hmac.Equal([]byte(chain[len(chain)-1]), []byte(mac.Signature)) {
		return chain, nil
	}
	return nil, ErrNotIssuedHere
}

// jsonChain is chainedSignature, keeping every intermediate value.
func (s *Service) jsonChain(mac *Macaroon) []string {
	out := make([]string, 0, len(mac.Caveats)+1)
	h := hmac.New(sha256.New, s.rootKey)
	h.Write([]byte(mac.Identifier))
	sig := h.Sum(nil)
	out = append(out, hex.EncodeToString(sig))
	for _, caveat := range mac.Caveats {
		h = hmac.New(sha256.New, sig)
		h.Write([]byte(caveat))
		sig = h.Sum(nil)
		out = append(out, hex.EncodeToString(sig))
	}
	return out
}

// binaryChain is the libmacaroon signature chain EncodeBinary produces,
// keeping every intermediate value.
func (s *Service) binaryChain(mac *Macaroon) ([]string, error) {
	m, err := macaroonv2.New(s.rootKey, []byte(mac.Identifier), mac.Location, macaroonv2.V2)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(mac.Caveats)+1)
	out = append(out, hex.EncodeToString(m.Signature()))
	for _, caveat := range mac.Caveats {
		if err := m.AddFirstPartyCaveat([]byte(caveat)); err != nil {
			return nil, err
		}
		out = append(out, hex.EncodeToString(m.Signature()))
	}
	return out, nil
}
