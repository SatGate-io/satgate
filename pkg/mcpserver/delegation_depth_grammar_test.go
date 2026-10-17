package mcpserver

import (
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
)

// Round 5: satgate/delegate answers every delegation_depth value the way
// /v1/mint/delegate does. That route reads a depth caveat with
// fmt.Sscanf("delegation_depth = %d") (mint.go) and its validator
// (macaroon.go validateDelegationDepthCaveats) refuses the token when the
// value does not read, reads below zero, or (in the parent's chain) reads as
// zero. Every other value is a cap, read as the leading integer.
//
// The expectations below were taken from that route (they are also run against
// the real route in the enterprise tests), not from this package's reading.

type depthGrammarCase struct {
	name string
	// caveats are the depth caveat values, in the order the chain carries them.
	values []string
	// refused: both placements are refused before any read, mint or carve.
	refused bool
	// cap: for an accepted case, the handoff count the smallest value allows.
	cap int
}

var depthGrammarCases = []depthGrammarCase{
	{"bogus", []string{"bogus"}, true, 0},
	{"nope", []string{"nope"}, true, 0},
	{"empty value", []string{""}, true, 0},
	{"spaces only", []string{"   "}, true, 0},
	{"-1", []string{"-1"}, true, 0},
	{"-0", []string{"-0"}, true, 0},
	{"0", []string{"0"}, true, 0},
	{"+0", []string{"+0"}, true, 0},
	{"00", []string{"00"}, true, 0},
	{"0x10", []string{"0x10"}, true, 0},
	{"abc3", []string{"abc3"}, true, 0},
	{"non-ASCII digit", []string{"\u0663"}, true, 0},
	{"overflow", []string{"99999999999999999999"}, true, 0},
	{"0 then 3", []string{"0", "3"}, true, 0},
	{"3 then 0", []string{"3", "0"}, true, 0},
	{"bogus then 3", []string{"bogus", "3"}, true, 0},
	{"3 then bogus", []string{"3", "bogus"}, true, 0},

	{"2.5 reads 2", []string{"2.5"}, false, 2},
	{"3abc reads 3", []string{"3abc"}, false, 3},
	{"1e3 reads 1", []string{"1e3"}, false, 1},
	{"+2", []string{"+2"}, false, 2},
	{"2", []string{"2"}, false, 2},
	{"leading and trailing spaces", []string{"  2  "}, false, 2},
	{"2 then 9 keeps 2", []string{"2", "9"}, false, 2},
	{"9 then 2 narrows to 2", []string{"9", "2"}, false, 2},
}

// placements: who wrote the caveats. "issuer" writes them all before the
// signature; "holder" appends them all after it; "split" has the issuer write
// the first and the holder append the rest.
func depthGrammarToken(r *zeroDepthRig, values []string, placement string) *TokenInfo {
	var all []string
	for _, v := range values {
		all = append(all, "delegation_depth = "+v)
	}
	switch placement {
	case "issuer":
		return r.token(all, nil, nil)
	case "holder":
		return r.token(nil, all, nil)
	}
	return r.token(all[:1], all[1:], nil)
}

// DEFECT TEST (Sol B1, Astra B1): an unparseable depth is refused by
// /v1/mint/delegate and handed a child here. Every value in the corpus gets
// the route's answer, with and without a stored limit, whoever wrote it.
func TestDelegateDepthValuesGetTheMintRoutesAnswer(t *testing.T) {
	for _, withStore := range []bool{false, true} {
		for _, placement := range []string{"issuer", "holder", "split"} {
			for _, tc := range depthGrammarCases {
				if placement == "split" && len(tc.values) < 2 {
					continue
				}
				name := fmt.Sprintf("store=%v/%s/%s", withStore, placement, tc.name)
				t.Run(name, func(t *testing.T) {
					if tc.refused {
						for _, budget := range []int64{0, 25} {
							r := newZeroDepthRig(t, withStore)
							parent := depthGrammarToken(r, tc.values, placement)
							resp := r.handoff(parent, budget)
							r.wantRefusedAndUntouched(parent, resp, wantNoHandoff, name)
							if r.store != nil && atomic.LoadInt32(&r.store.reads) != 0 {
								t.Fatalf("%s: the stored row was read %d times before the value was refused", name, r.store.reads)
							}
						}
						return
					}
					// Accepted: tc.cap handoffs succeed, the next stops at the cap.
					r := newZeroDepthRig(t, withStore)
					token := depthGrammarToken(r, tc.values, placement)
					for i := 0; i < tc.cap; i++ {
						token = r.childOf(token)
					}
					resp := r.handoff(token, 5)
					want := fmt.Sprintf("delegation depth limit reached (%d)", tc.cap)
					if resp.Error == nil || resp.Error.Message != want {
						t.Fatalf("after %d handoffs: %+v, want %q", tc.cap, resp.Error, want)
					}
				})
			}
		}
	}
}

// DEFECT TEST: the refusal logs the same one line as the zero refusal, with no
// value, token or tenant in it.
func TestDelegateUnreadableDepthRefusalLogsNoValue(t *testing.T) {
	buf := captureLogs(t)
	r := newZeroDepthRig(t, false)
	parent := r.token(nil, []string{"delegation_depth = depth-canary-value"}, nil)
	r.handoff(parent, 25)
	out := buf.String()
	if n := strings.Count(out, "allows no further handoffs"); n != 1 {
		t.Fatalf("want one refusal line, got %d in:\n%s", n, out)
	}
	if strings.Contains(out, "depth-canary-value") {
		t.Fatalf("the refusal log carries the caveat value:\n%s", out)
	}
}
