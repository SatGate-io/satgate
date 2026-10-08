package argrules

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
