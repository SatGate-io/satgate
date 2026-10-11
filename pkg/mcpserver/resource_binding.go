package mcpserver

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/satgate-io/satgate/pkg/macaroon"
)

// ResourceCaveatKey is the caveat a hosted sign-in token carries to name the
// one MCP resource it was issued for: "resource = <exact resource URL>".
const ResourceCaveatKey = "resource"

// ErrResourceMismatch is returned when a token names a resource other than
// this server's own, or when this server has no resource configured. The
// transports answer it with the normal 401 (and the challenge when one is
// configured), on every method.
var ErrResourceMismatch = errors.New("token was issued for a different resource")

// CheckResourceCaveats refuses a token whose resource caveats do not all equal
// resource exactly (string equality, no normalisation). A token with no
// resource caveat passes unchanged. With resource empty, a token that carries
// the caveat is refused: a server with no resource configured fails closed.
// The caveat chain is append-only, so a child token keeps its parent's caveat,
// and a second, different one can only make the token fail here.
func CheckResourceCaveats(caveats []string, resource string) error {
	prefix := ResourceCaveatKey + " = "
	for _, c := range caveats {
		if !strings.HasPrefix(c, prefix) {
			continue
		}
		if resource == "" || strings.TrimPrefix(c, prefix) != resource {
			return ErrResourceMismatch
		}
	}
	return nil
}

// ResourceBoundAuthenticator wraps an Authenticator and enforces
// CheckResourceCaveats on every token the inner authenticator accepts. A token
// without a resource caveat is returned exactly as the inner one returned it.
type ResourceBoundAuthenticator struct {
	Inner    Authenticator
	Resource string
}

// Verify implements Authenticator.
func (a *ResourceBoundAuthenticator) Verify(ctx context.Context, token string) (*TokenInfo, error) {
	if a == nil || a.Inner == nil {
		return nil, fmt.Errorf("authenticator unavailable")
	}
	info, err := a.Inner.Verify(ctx, token)
	if err != nil {
		return nil, err
	}
	if info != nil && info.Raw != nil {
		if err := CheckResourceCaveats(info.Raw.Caveats, a.Resource); err != nil {
			return nil, err
		}
	}
	return info, nil
}

// BindResource returns inner bound to resource. A nil inner stays nil. An
// authenticator that is already bound is rebound, never wrapped twice.
func BindResource(inner Authenticator, resource string) Authenticator {
	if inner == nil {
		return nil
	}
	if b, ok := inner.(*ResourceBoundAuthenticator); ok {
		inner = b.Inner
	}
	return &ResourceBoundAuthenticator{Inner: inner, Resource: resource}
}

// SetResource tells the proxy which MCP resource URL it serves. Every
// authenticator the proxy holds, now or set later, refuses a token that names
// another resource. Until it is called the resource is empty and a token that
// names any resource is refused.
func (p *Proxy) SetResource(resource string) {
	p.resource = resource
	p.auth = BindResource(p.auth, resource)
}

// WithResourceBinding is the server option that carries the resource to the
// proxy. A server built without it keeps the empty resource and fails closed.
func WithResourceBinding(resource string) SSEOption {
	return func(s *SSEServer) {
		if s.proxy != nil {
			s.proxy.SetResource(resource)
		}
	}
}

// tokenResourceCaveats reads the resource caveats of a token without trusting
// it. It is only used to pick the status code before the real verification;
// removing the caveat breaks the signature, so a token cannot shed it.
func tokenResourceCaveats(token string) []string {
	token = strings.TrimPrefix(token, "Bearer ")
	svc, err := macaroon.NewService("decode-only")
	if err != nil {
		return nil
	}
	mac, err := svc.Decode(token)
	if err != nil {
		mac, err = svc.DecodeBinary(token)
		if err != nil {
			return nil
		}
	}
	var out []string
	for _, c := range mac.Caveats {
		if strings.HasPrefix(c, ResourceCaveatKey+" = ") {
			out = append(out, c)
		}
	}
	return out
}

// refuseWrongResource writes the 401 (with the challenge when configured) and
// reports true when the token in the request names a resource that is not
// this server's. It runs on every Streamable HTTP request and on the SSE
// connect and message endpoints, so no method is reachable with such a token.
func (s *SSEServer) refuseWrongResource(w http.ResponseWriter, token string) bool {
	if token == "" || s.proxy == nil {
		return false
	}
	caveats := tokenResourceCaveats(token)
	if len(caveats) == 0 {
		return false
	}
	if CheckResourceCaveats(caveats, s.proxy.resource) == nil {
		return false
	}
	s.writeAuthChallenge(w, "authentication failed")
	return true
}
