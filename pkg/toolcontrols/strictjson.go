package toolcontrols

import (
	"fmt"
	"unicode/utf8"
)

// A strict, lossless reading of a tools/call body.
//
// An approval is only as good as its call hash: what is hashed must identify
// exactly what is forwarded. The forwarded call keeps its raw bytes, so the
// reading that feeds the hash must not merge two different byte strings into
// one value. encoding/json does exactly that in a few places (an unpaired
// \ud800 and \ud801 both become U+FFFD; a repeated key keeps the last value;
// a key that differs only by letter case fills a struct field; a number goes
// through float64 if the caller is careless). This reader is the one place
// that decides what a call means, and it refuses anything it cannot read
// without loss:
//
//   - invalid UTF-8 anywhere in the body;
//   - an escaped UTF-16 surrogate that is not half of a correct pair (a lone
//     high or low half, a reversed pair), in any key or string at any depth;
//     a correct pair is allowed;
//   - a repeated object key at any depth (keys compared after unescaping);
//   - a number whose text is not a plain JSON number, is longer than
//     maxNumberText, has more than maxNumberDigits significant digits, or is
//     too large or too small for a binary64 (non-finite or underflowing in
//     any consumer that uses floats);
//   - nesting deeper than maxCallNesting, and anything after the value.
//
// Numbers are never converted: they are kept as their own text.

const (
	maxNumberText   = 64
	maxNumberDigits = 40
	// maxDecimalExp bounds the decimal exponent of a number's leading digit:
	// 1e308 is the largest finite binary64 magnitude, 2.2e-308 the smallest
	// normal one. A number outside that range is infinite or subnormal/zero
	// for a consumer that reads floats, so it is not exact for everyone.
	maxDecimalExp = 308
	minDecimalExp = -307
)

// lossless value kinds returned by the reader. number is a numText.
type numText string

type objectEntries map[string]interface{}

// CheckCallLossless reports whether doc (the params of a tools/call) can be
// read without loss. The error never contains a value from doc.
func CheckCallLossless(doc []byte) error {
	_, err := readStrict(doc)
	return err
}

type strictReader struct {
	b []byte
	i int
}

func readStrict(doc []byte) (interface{}, error) {
	if !utf8.Valid(doc) {
		return nil, unreadable("invalid UTF-8")
	}
	r := &strictReader{b: doc}
	v, err := r.value(0)
	if err != nil {
		return nil, err
	}
	r.ws()
	if r.i != len(r.b) {
		return nil, unreadable("data after the JSON value")
	}
	return v, nil
}

func unreadable(why string) error {
	return fmt.Errorf("%w: %s", ErrCallUnreadable, why)
}

func (r *strictReader) ws() {
	for r.i < len(r.b) {
		switch r.b[r.i] {
		case ' ', '\t', '\n', '\r':
			r.i++
		default:
			return
		}
	}
}

func (r *strictReader) value(depth int) (interface{}, error) {
	r.ws()
	if r.i >= len(r.b) {
		return nil, unreadable("unexpected end")
	}
	switch c := r.b[r.i]; {
	case c == '{':
		return r.object(depth + 1)
	case c == '[':
		return r.array(depth + 1)
	case c == '"':
		return r.str()
	case c == 't':
		return r.lit("true", true)
	case c == 'f':
		return r.lit("false", false)
	case c == 'n':
		return r.lit("null", nil)
	case c == '-' || (c >= '0' && c <= '9'):
		return r.number()
	}
	return nil, unreadable("not JSON")
}

func (r *strictReader) lit(word string, v interface{}) (interface{}, error) {
	if len(r.b)-r.i < len(word) || string(r.b[r.i:r.i+len(word)]) != word {
		return nil, unreadable("not JSON")
	}
	r.i += len(word)
	return v, nil
}

func (r *strictReader) object(depth int) (interface{}, error) {
	if depth > maxCallNesting {
		return nil, unreadable("nested too deeply")
	}
	r.i++ // {
	out := objectEntries{}
	r.ws()
	if r.i < len(r.b) && r.b[r.i] == '}' {
		r.i++
		return out, nil
	}
	for {
		r.ws()
		if r.i >= len(r.b) || r.b[r.i] != '"' {
			return nil, unreadable("bad object key")
		}
		k, err := r.str()
		if err != nil {
			return nil, err
		}
		key := k.(string)
		if _, dup := out[key]; dup {
			return nil, unreadable("repeated object key")
		}
		r.ws()
		if r.i >= len(r.b) || r.b[r.i] != ':' {
			return nil, unreadable("missing colon")
		}
		r.i++
		v, err := r.value(depth)
		if err != nil {
			return nil, err
		}
		out[key] = v
		r.ws()
		if r.i >= len(r.b) {
			return nil, unreadable("unexpected end")
		}
		switch r.b[r.i] {
		case ',':
			r.i++
		case '}':
			r.i++
			return out, nil
		default:
			return nil, unreadable("bad object")
		}
	}
}

func (r *strictReader) array(depth int) (interface{}, error) {
	if depth > maxCallNesting {
		return nil, unreadable("nested too deeply")
	}
	r.i++ // [
	out := []interface{}{}
	r.ws()
	if r.i < len(r.b) && r.b[r.i] == ']' {
		r.i++
		return out, nil
	}
	for {
		v, err := r.value(depth)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
		r.ws()
		if r.i >= len(r.b) {
			return nil, unreadable("unexpected end")
		}
		switch r.b[r.i] {
		case ',':
			r.i++
		case ']':
			r.i++
			return out, nil
		default:
			return nil, unreadable("bad array")
		}
	}
}

func hexVal(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'F':
		return int(c-'A') + 10
	}
	return -1
}

// hex4 reads the four hex digits at r.b[at:at+4].
func (r *strictReader) hex4(at int) (rune, bool) {
	if at+4 > len(r.b) {
		return 0, false
	}
	var v rune
	for _, c := range r.b[at : at+4] {
		h := hexVal(c)
		if h < 0 {
			return 0, false
		}
		v = v<<4 | rune(h)
	}
	return v, true
}

func (r *strictReader) str() (interface{}, error) {
	r.i++ // opening quote
	buf := make([]byte, 0, 16)
	for {
		if r.i >= len(r.b) {
			return nil, unreadable("unterminated string")
		}
		c := r.b[r.i]
		switch {
		case c == '"':
			r.i++
			return string(buf), nil
		case c < 0x20:
			return nil, unreadable("control character in string")
		case c != '\\':
			buf = append(buf, c) // UTF-8 was validated for the whole body
			r.i++
		default:
			r.i++
			if r.i >= len(r.b) {
				return nil, unreadable("unterminated string")
			}
			e := r.b[r.i]
			r.i++
			switch e {
			case '"', '\\', '/':
				buf = append(buf, e)
			case 'b':
				buf = append(buf, '\b')
			case 'f':
				buf = append(buf, '\f')
			case 'n':
				buf = append(buf, '\n')
			case 'r':
				buf = append(buf, '\r')
			case 't':
				buf = append(buf, '\t')
			case 'u':
				u, ok := r.hex4(r.i)
				if !ok {
					return nil, unreadable("bad \\u escape")
				}
				r.i += 4
				switch {
				case u >= 0xDC00 && u <= 0xDFFF:
					return nil, unreadable("unpaired UTF-16 surrogate")
				case u >= 0xD800 && u <= 0xDBFF:
					if r.i+6 > len(r.b) || r.b[r.i] != '\\' || r.b[r.i+1] != 'u' {
						return nil, unreadable("unpaired UTF-16 surrogate")
					}
					lo, ok := r.hex4(r.i + 2)
					if !ok || lo < 0xDC00 || lo > 0xDFFF {
						return nil, unreadable("unpaired UTF-16 surrogate")
					}
					r.i += 6
					u = 0x10000 + (u-0xD800)<<10 + (lo - 0xDC00)
				}
				buf = utf8.AppendRune(buf, u)
			default:
				return nil, unreadable("bad escape")
			}
		}
	}
}

// number reads a JSON number and keeps its text. It does not convert it.
func (r *strictReader) number() (interface{}, error) {
	start := r.i
	if r.b[r.i] == '-' {
		r.i++
	}
	if r.i >= len(r.b) {
		return nil, unreadable("bad number")
	}
	intStart := r.i
	switch {
	case r.b[r.i] == '0':
		r.i++
	case r.b[r.i] >= '1' && r.b[r.i] <= '9':
		for r.i < len(r.b) && isDigit(r.b[r.i]) {
			r.i++
		}
	default:
		return nil, unreadable("bad number")
	}
	intEnd := r.i
	fracStart, fracEnd := r.i, r.i
	if r.i < len(r.b) && r.b[r.i] == '.' {
		r.i++
		fracStart = r.i
		for r.i < len(r.b) && isDigit(r.b[r.i]) {
			r.i++
		}
		fracEnd = r.i
		if fracEnd == fracStart {
			return nil, unreadable("bad number")
		}
	}
	exp := 0
	if r.i < len(r.b) && (r.b[r.i] == 'e' || r.b[r.i] == 'E') {
		r.i++
		neg := false
		if r.i < len(r.b) && (r.b[r.i] == '+' || r.b[r.i] == '-') {
			neg = r.b[r.i] == '-'
			r.i++
		}
		es := r.i
		for r.i < len(r.b) && isDigit(r.b[r.i]) {
			if r.i-es > 5 {
				return nil, unreadable("number exponent too large")
			}
			exp = exp*10 + int(r.b[r.i]-'0')
			r.i++
		}
		if r.i == es {
			return nil, unreadable("bad number")
		}
		if neg {
			exp = -exp
		}
	}
	if r.i-start > maxNumberText {
		return nil, unreadable("number too long")
	}
	// Significant digits and the decimal exponent of the leading digit.
	digits := append(append([]byte{}, r.b[intStart:intEnd]...), r.b[fracStart:fracEnd]...)
	lead := 0
	for lead < len(digits) && digits[lead] == '0' {
		lead++
	}
	sig := digits[lead:]
	for len(sig) > 0 && sig[len(sig)-1] == '0' {
		sig = sig[:len(sig)-1]
	}
	if len(sig) > maxNumberDigits {
		return nil, unreadable("number has too many digits")
	}
	if len(sig) > 0 { // zero in any spelling is exact
		// value = 0.d1d2... x 10^(intLen - lead + exp); leading digit is 10^(that-1).
		lead10 := (intEnd - intStart) - lead + exp - 1
		if lead10 > maxDecimalExp || lead10 < minDecimalExp {
			return nil, unreadable("number is outside the exact range")
		}
	}
	return numText(r.b[start:r.i]), nil
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }
