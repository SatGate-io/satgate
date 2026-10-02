// Package denial is the single signed-denial entry the HTTP proxy and MCP both call.
// A later change can call RecordSignedDenial from a rate limiter. This package
// takes no lock. The recorder must sign and store without a process-wide mutex,
// so a caller that still holds a limiter lock cannot deadlock on evidence.
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

// ProofRecorder signs and stores one denial. Implementations must not acquire a
// process-wide lock around signing or evidence storage.
type ProofRecorder interface {
	RecordSignedDenial(ctx context.Context, in SignedDenial) (any, error)
}

// ErrNoRecorder is returned when no recorder is configured. Callers that must
// fail closed map this to proof_unavailable. Callers with no evidence recorder
// should not call RecordSignedDenial.
var ErrNoRecorder = errors.New("signed denial recorder is not configured")

type lockFreeKey struct{}
type issuedAtKey struct{}

// SkipsGlobalLock reports whether this context was created by RecordSignedDenial.
// Archive code uses it to write the pack without the process-wide archive mutex.
func SkipsGlobalLock(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	v, _ := ctx.Value(lockFreeKey{}).(bool)
	return v
}

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
// It does not sign, store, or lock. The recorder does that, on a context that
// asks storage to skip any process-wide archive lock.
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
	ctx = context.WithValue(ctx, lockFreeKey{}, true)
	ctx = context.WithValue(ctx, issuedAtKey{}, at.UTC())
	return rec.RecordSignedDenial(ctx, SignedDenial{
		ReasonCode: reasonCode,
		Identity:   identity,
		Target:     target,
		At:         at.UTC(),
	})
}
