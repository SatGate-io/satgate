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

// CheckResourceCaveats refuses a token whose resource caveats do not all name
// the same configured resource (string equality, no normalisation). Optional
// previous resources allow tokens issued before a resource URL change. With
// no previous resources, only resource itself is accepted. A token with no
// resource caveat passes unchanged. With resource empty, a token that carries
// the caveat is refused: a server with no resource configured fails closed.
// The caveat chain is append-only, so a child token keeps its parent's caveat,
// and a second, different one can only make the token fail here.
func CheckResourceCaveats(caveats []string, resource string, previous ...string) error {
	prefix := ResourceCaveatKey + " = "
	selected := ""
	for _, c := range caveats {
		if !strings.HasPrefix(c, prefix) {
			continue
		}
		value := strings.TrimPrefix(c, prefix)
		allowed := resource != "" && value == resource
		for _, old := range previous {
			allowed = allowed || (resource != "" && old != "" && value == old)
		}
		if !allowed || (selected != "" && selected != value) {
			return ErrResourceMismatch
		}
		selected = value
	}
	return nil
}

// ResourceBoundAuthenticator wraps an Authenticator and enforces
// CheckResourceCaveats on every token the inner authenticator accepts. A token
// without a resource caveat is returned exactly as the inner one returned it.
type ResourceBoundAuthenticator struct {
	Inner    Authenticator
	Resource string
	// Previous holds optional exact resource URLs accepted alongside Resource.
	// It has no effect when Resource is empty.
	Previous []string

	// lineage, when the proxy has a LineageRevocationChecker, refuses a
	// token a ban covers (see Proxy.SetLineageRevocationChecker).
	lineage *lineageGate
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
		if err := CheckResourceCaveats(info.Raw.Caveats, a.Resource, a.Previous...); err != nil {
			return nil, err
		}
	}
	if err := a.lineage.check(ctx, info, macaroonServiceOf(a.Inner)); err != nil {
		return nil, err
	}
	return info, nil
}

// BindResource returns inner bound to resource and optional previous resource
// URLs. It copies previous. A nil inner stays nil. An
// authenticator that is already bound is rebound, never wrapped twice.
func BindResource(inner Authenticator, resource string, previous ...string) Authenticator {
	if inner == nil {
		return nil
	}
	var gate *lineageGate
	if b, ok := inner.(*ResourceBoundAuthenticator); ok {
		inner = b.Inner
		gate = b.lineage
	}
	return &ResourceBoundAuthenticator{Inner: inner, Resource: resource, Previous: append([]string(nil), previous...), lineage: gate}
}

// SetResource tells the proxy which MCP resource URL it serves. Every
// authenticator the proxy holds, now or set later, refuses a token that names
// an unconfigured resource. Optional previous URLs accept tokens issued before
// a resource URL change, but mixed-resource caveat chains still fail. Until
// it is called the resource is empty and a token that
// names any resource is refused.
func (p *Proxy) SetResource(resource string, previous ...string) {
	p.resource = resource
	p.previousResources = append([]string(nil), previous...)
	p.auth = p.bindAuth(p.auth)
}

// WithResourceBinding is the server option that carries the resource and
// optional previous resource URLs to the
// proxy. A server built without it keeps the empty resource and fails closed.
func WithResourceBinding(resource string, previous ...string) SSEOption {
	return func(s *SSEServer) {
		if s.proxy != nil {
			s.proxy.SetResource(resource, previous...)
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
	if CheckResourceCaveats(caveats, s.proxy.resource, s.proxy.previousResources...) == nil {
		return false
	}
	s.writeAuthChallenge(w, "authentication failed")
	return true
}
