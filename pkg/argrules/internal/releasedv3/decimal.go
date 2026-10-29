// Code in this directory is a frozen copy, kept only so tests can show what a
// verifier built before the event contract keys does with a document that
// carries them. It is a record of public commit e22202ba491c (the last main
// before "event_order", "schema", "event_markets" and "event_stakes"
// existed), with only the package clause changed. Do not edit it, and do not
// import it from non-test code.

package releasedv3

import (
	"fmt"
	"math/big"
)

// MaxDecimalDigits is the most digits (before and after the point, counted
// together) a limit or an argument may have.
const MaxDecimalDigits = 38

// Decimal is an exact plain decimal. It is never converted to float64.
//
// Accepted grammar: an optional "-", then "0" or a digit run with no leading
// zero, then optionally "." and one or more digits. Refused: empty text,
// whitespace, a leading "+", exponents, hex, NaN, Inf, a bare ".", a trailing
// ".", leading zeros ("01"), and more than 38 digits.
type Decimal struct {
	text string
	rat  *big.Rat
}

// ParseDecimal parses text under the grammar above.
func ParseDecimal(text string) (*Decimal, error) {
	if err := checkPlainDecimal(text); err != nil {
		return nil, err
	}
	r, ok := new(big.Rat).SetString(text)
	if !ok {
		return nil, fmt.Errorf("not a decimal")
	}
	return &Decimal{text: text, rat: r}, nil
}

func checkPlainDecimal(s string) error {
	if s == "" {
		return fmt.Errorf("empty")
	}
	i := 0
	if s[0] == '-' {
		i++
	}
	intStart := i
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	intLen := i - intStart
	if intLen == 0 {
		return fmt.Errorf("no digits")
	}
	if intLen > 1 && s[intStart] == '0' {
		return fmt.Errorf("leading zero")
	}
	fracLen := 0
	if i < len(s) && s[i] == '.' {
		i++
		fracStart := i
		for i < len(s) && s[i] >= '0' && s[i] <= '9' {
			i++
		}
		fracLen = i - fracStart
		if fracLen == 0 {
			return fmt.Errorf("no digits after the point")
		}
	}
	if i != len(s) {
		return fmt.Errorf("not a plain decimal")
	}
	if intLen+fracLen > MaxDecimalDigits {
		return fmt.Errorf("more than %d digits", MaxDecimalDigits)
	}
	return nil
}

// Cmp compares exactly: -1 if d < o, 0 if equal, +1 if d > o.
func (d *Decimal) Cmp(o *Decimal) int { return d.rat.Cmp(o.rat) }

// String is the text the decimal was parsed from.
func (d *Decimal) String() string { return d.text }

// Times is the exact product d x o. Nothing is rounded.
func (d *Decimal) Times(o *Decimal) *big.Rat { return new(big.Rat).Mul(d.rat, o.rat) }

// UnitsPerOne is how many amount units make 1: amounts are counted in whole
// 10^-8 units (the scale a spending limit keeps; see toolcontrols.MaxScale).
const UnitsPerOne int64 = 100000000

// UnitsCeil converts a non-negative exact value to whole 10^-8 units,
// rounding UP, so an amount is never undercounted. It reports false for a
// negative value and for one that does not fit an int64. A value above zero
// never comes back as 0 units.
//
// It is the same rounding spend_gate.go's amountUnits and toolcontrols'
// unitsOf (with ceil) apply; tests hold the three equal.
func UnitsCeil(r *big.Rat) (int64, bool) {
	if r.Sign() < 0 {
		return 0, false
	}
	scaled := new(big.Rat).Mul(r, new(big.Rat).SetInt64(UnitsPerOne))
	q, rem := new(big.Int).QuoRem(scaled.Num(), scaled.Denom(), new(big.Int))
	if rem.Sign() != 0 {
		q.Add(q, big.NewInt(1))
	}
	if !q.IsInt64() {
		return 0, false
	}
	return q.Int64(), true
}
