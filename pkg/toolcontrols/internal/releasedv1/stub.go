package releasedv1

import "encoding/json"

// Approval rules are out of scope for the snapshot: they are not what the
// test exercises. This stub keeps controls.go compiling unchanged.
type ApprovalRule struct {
	Tools []string
	Field string
	Above string
	ID    string
}

func parseApprovalRule(json.RawMessage) (ApprovalRule, error) {
	return ApprovalRule{}, ctlErr("approval rules are not part of this snapshot")
}

func (r *ApprovalRule) validate() error { return nil }
