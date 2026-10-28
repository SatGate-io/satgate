package macaroon

import (
	"encoding/base64"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	macaroonv2 "gopkg.in/macaroon.v2"
)

// Reviewer cases: through 128 offline delegations, in SatGate's JSON form and
// in the standard binary form, every ancestor's signature is in the lineage;
// an altered chain, a nil macaroon, a foreign root and a forged signature get
// no lineage.
func TestLineage_KeepsEveryAncestorTo128InBothFormsAndRefusesForgeries(t *testing.T) {
	s := newLineageService(t)
	other, err := NewService("lineage-unrelated-root-key")
	if err != nil {
		t.Fatal(err)
	}
	root, err := s.Mint("api:*", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}

	// JSON form.
	tok := s.Encode(root)
	ancestors := []string{root.Signature}
	var last *Macaroon
	for i := 0; i < 128; i++ {
		m, err := s.DelegateWithoutVerify(tok, []string{fmt.Sprintf("label = depth-%d", i)})
		if err != nil {
			t.Fatal(err)
		}
		tok = s.Encode(m)
		v, err := s.Verify(tok)
		if err != nil {
			t.Fatal(err)
		}
		chain, err := s.Lineage(v)
		if err != nil {
			t.Fatal(err)
		}
		for _, a := range ancestors {
			if !slices.Contains(chain, a) {
				t.Fatalf("JSON depth %d: an ancestor is missing from the lineage", i+1)
			}
		}
		ancestors = append(ancestors, m.Signature)
		last = v
	}
	tampered := *last
	tampered.Caveats = append(append([]string{}, last.Caveats[:2]...), last.Caveats[3:]...)
	if _, err := s.Lineage(&tampered); err == nil {
		t.Fatal("a chain with a caveat removed got a lineage")
	}

	// Binary form, attenuated with the standard library.
	bin, err := s.EncodeBinary(root)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := base64.StdEncoding.DecodeString(bin)
	var m macaroonv2.Macaroon
	if err := m.UnmarshalBinary(raw); err != nil {
		t.Fatal(err)
	}
	var binAncestors []string
	for i := 0; i < 128; i++ {
		v, err := s.Verify(bin)
		if err != nil {
			t.Fatal(err)
		}
		binAncestors = append(binAncestors, v.Signature)
		if err := m.AddFirstPartyCaveat([]byte(fmt.Sprintf("label = binary-depth-%d", i))); err != nil {
			t.Fatal(err)
		}
		raw, _ = m.MarshalBinary()
		bin = base64.StdEncoding.EncodeToString(raw)
		v, err = s.Verify(bin)
		if err != nil {
			t.Fatal(err)
		}
		chain, err := s.Lineage(v)
		if err != nil {
			t.Fatal(err)
		}
		for _, a := range binAncestors {
			if !slices.Contains(chain, a) {
				t.Fatalf("binary depth %d: an ancestor is missing from the lineage", i+1)
			}
		}
	}

	foreign, err := other.Mint("api:*", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	for _, candidate := range []*Macaroon{
		nil,
		foreign,
		{Identifier: root.Identifier, Caveats: root.Caveats, Signature: strings.Repeat("0", 64)},
		{Identifier: root.Identifier, Caveats: root.Caveats[1:], Signature: root.Signature},
	} {
		if _, err := s.Lineage(candidate); err == nil {
			t.Fatal("a macaroon that does not follow from the root key got a lineage")
		}
	}
}
