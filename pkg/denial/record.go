// Package denial is the single signed-denial entry the HTTP proxy and MCP both call.
// A later change can call RecordSignedDenial from a rate limiter.
//
// Locking: this package takes no lock, and receipt signing takes no lock.
// Durable storage is the recorder's job. The enterprise filesystem archive
// appends every pack to one hash chain (previous_evidence_pack_hash plus
// .archive_head), so that append is serialized by the archive's own mutex.
// A pack written outside that chain is rejected by the archive collector, so
// this package does not offer a way to skip it. Callers must not hold their own
// locks (for example a limiter lock) while calling RecordSignedDenial.
package denial

import (
	"context"
	"errors"
	"time"
)

// SignedDenial is the reusable denial record. Identity is a token or caller id.
// Target is a route or tool. At is the decision time the receipt must stamp.
type SignedDenial struct {
	ReasonCode string
	Identity   string
	Target     string
	At         time.Time
}

// ProofRecorder signs and stores one denial and returns the caller's proof
// handle (a receipt map for HTTP, *MCPEvidence for MCP).
type ProofRecorder interface {
	RecordSignedDenial(ctx context.Context, in SignedDenial) (any, error)
}

// ErrNoRecorder is returned when no recorder is configured. Callers that must
// fail closed map this to proof_unavailable. Callers with no evidence recorder
// should not call RecordSignedDenial.
var ErrNoRecorder = errors.New("signed denial recorder is not configured")

type issuedAtKey struct{}

// IssuedAt is the decision time RecordSignedDenial stored on the context.
// Receipt builders stamp this instead of a second clock read.
func IssuedAt(ctx context.Context) (time.Time, bool) {
	if ctx == nil {
		return time.Time{}, false
	}
	at, ok := ctx.Value(issuedAtKey{}).(time.Time)
	if !ok || at.IsZero() {
		return time.Time{}, false
	}
	return at.UTC(), true
}

// RecordSignedDenial is the one function the HTTP proxy and MCP use to record a
// signed denial. reasonCode, identity, target, and at are the reusable inputs.
// It does not sign, store, or lock. The recorder signs and stores, on a context
// that carries at so the receipt's issued_at equals the decision time.
func RecordSignedDenial(ctx context.Context, rec ProofRecorder, reasonCode, identity, target string, at time.Time) (any, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if at.IsZero() {
		at = time.Now().UTC()
	}
	if rec == nil {
		return nil, ErrNoRecorder
	}
	ctx = context.WithValue(ctx, issuedAtKey{}, at.UTC())
	return rec.RecordSignedDenial(ctx, SignedDenial{
		ReasonCode: reasonCode,
		Identity:   identity,
		Target:     target,
		At:         at.UTC(),
	})
}
