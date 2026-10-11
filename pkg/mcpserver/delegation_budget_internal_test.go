package mcpserver

// Round 2, B1: tests of the delegation helpers themselves.

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

// ---- the proof before the seal ---------------------------------------------

// childKeepsOnlyTheParentsSealedBudget is the check that stands between a
// holder-chosen caveat and the seal. It must say no to a child whose prefix
// carries a budget reference the parent's issuer seal does not cover, or whose
// first caveats are not the parent's.
func TestSealProofRefusesAHolderBudgetInTheChildPrefix(t *testing.T) {
	svc, tok := sealedBudgetToken(t, "tenant-1", "own-1", bothTools, budgetDoc(budgetA, "200", ""))
	parent, _ := svc.Decode(tok)
	svc = svc.AcceptingArgumentRules()

	honest, err := svc.Delegate(tok, []string{"parent = x", "budget_id = del-aaaaaaaaaaaaaaaaaaaaaaaa", "budget_limit = 5.00"})
	if err != nil {
		t.Fatal(err)
	}
	if !childKeepsOnlyTheParentsSealedBudget(svc, parent, honest) {
		t.Fatal("an honest child was refused")
	}
	for name, extra := range map[string][]string{
		"a holder budget before the seal":   {"parent = x", controlWord(t, budgetDoc(budgetB, "9", ""))},
		"a holder budget with a raised max": {"parent = x", controlWord(t, budgetDoc(budgetA, "9999999", ""))},
	} {
		child, err := svc.Delegate(tok, extra)
		if err != nil {
			t.Fatal(err)
		}
		if childKeepsOnlyTheParentsSealedBudget(svc, parent, child) {
			t.Fatalf("%s: the proof passed", name)
		}
	}
	// A child that changed one of the parent's own words.
	changed := *honest
	changed.Caveats = append([]string{}, honest.Caveats...)
	for i, c := range changed.Caveats {
		if strings.HasPrefix(c, "scope = argrules:ctl:") {
			changed.Caveats[i] = controlWord(t, budgetDoc(budgetA, "9999999", ""))
		}
	}
	if childKeepsOnlyTheParentsSealedBudget(svc, parent, &changed) {
		t.Fatal("a child that rewrote the parent's budget word passed")
	}
	if childKeepsOnlyTheParentsSealedBudget(svc, nil, honest) || childKeepsOnlyTheParentsSealedBudget(svc, parent, nil) {
		t.Fatal("nil passed")
	}
}

func TestScopeHashAndWordCount(t *testing.T) {
	if scopeWordCount("") != 0 || scopeWordCount("a") != 1 || scopeWordCount("a,b c") != 3 || scopeWordCount(" , ") != 0 {
		t.Fatal("word count")
	}
	sum := sha256.Sum256([]byte("a,b"))
	if scopeSHA256("a,b") != hex.EncodeToString(sum[:]) {
		t.Fatal("hash")
	}
}
