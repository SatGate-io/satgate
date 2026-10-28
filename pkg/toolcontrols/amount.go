package toolcontrols

import (
	"errors"

	"github.com/satgate-io/satgate/pkg/argrules"
)

// ErrNotWholeAmount is returned by ParseWholeAmount for every value it refuses.
var ErrNotWholeAmount = errors.New("must be a positive whole number")

// ParseWholeAmount reads a whole-unit amount written in a caveat (for example
// the value of a budget_limit or delegation_budget caveat) exactly, with the
// same plain-decimal grammar a spending limit's max is read with
// (argrules.ParseDecimal). It is never converted to float64 and never rounded.
//
// The value must be a whole number above zero that fits in an int64. Zero, a
// negative value, a value with a fractional part ("0.5", "2.25"), text that is
// not a plain decimal ("", " 5", "+5", "05", "1e3", "0x10", "NaN") and a value
// out of range are refused with ErrNotWholeAmount. A whole value written with
// zeros after the point ("5.00") reads as 5.
func ParseWholeAmount(text string) (int64, error) {
	d, err := argrules.ParseDecimal(text)
	if err != nil {
		return 0, ErrNotWholeAmount
	}
	r := d.Rat()
	if r.Sign() <= 0 || !r.IsInt() || !r.Num().IsInt64() {
		return 0, ErrNotWholeAmount
	}
	return r.Num().Int64(), nil
}
