package denial

import (
	"context"
	"testing"
	"time"
)

type captureRecorder struct {
	in  SignedDenial
	ctx context.Context
}

func (c *captureRecorder) RecordSignedDenial(ctx context.Context, in SignedDenial) (any, error) {
	c.ctx = ctx
	c.in = in
	return "proof", nil
}

func TestRecordSignedDenialPassesFieldsAndSkipsGlobalLock(t *testing.T) {
	at := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	rec := &captureRecorder{}
	proof, err := RecordSignedDenial(context.Background(), rec, "policy_denied", "token-1", "mcp:tools/call:search", at)
	if err != nil {
		t.Fatal(err)
	}
	if proof != "proof" {
		t.Fatalf("proof = %v", proof)
	}
	if rec.in.ReasonCode != "policy_denied" || rec.in.Identity != "token-1" || rec.in.Target != "mcp:tools/call:search" || !rec.in.At.Equal(at) {
		t.Fatalf("recorded %+v", rec.in)
	}
	if !SkipsGlobalLock(rec.ctx) {
		t.Fatal("recorder context still takes the global archive lock")
	}
	stamped, ok := IssuedAt(rec.ctx)
	if !ok || !stamped.Equal(at) {
		t.Fatalf("issued at = %v ok=%v", stamped, ok)
	}
}

func TestRecordSignedDenialRejectsNilRecorder(t *testing.T) {
	_, err := RecordSignedDenial(context.Background(), nil, "policy_denied", "token-1", "route", time.Now())
	if err != ErrNoRecorder {
		t.Fatalf("err = %v", err)
	}
}
