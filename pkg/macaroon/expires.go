package macaroon

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ErrExpiresNotAbsolute is wrapped when an "expires" caveat does not hold an
// absolute time. Verify refuses such a token. Test for it with errors.Is.
var ErrExpiresNotAbsolute = errors.New("expires caveat must be an absolute time: Unix milliseconds or RFC 3339")

// ErrTokenExpired is wrapped when an expiry caveat is in the past.
var ErrTokenExpired = errors.New("token expired")

// ParseExpires reads the value of an "expires" caveat. It is the one reader
// of that value: Verify, GetExpiry, CheckL402Request and Delegate all call it.
// Two absolute forms are accepted, and nothing else:
//
//   - Unix time in milliseconds, written as decimal digits only, the form Mint
//     and MintL402 write (for example "1793577600000");
//   - an RFC 3339 date-time with a zone, spelled exactly as the RFC 3339
//     grammar allows (see isRFC3339DateTime), for example
//     "2026-11-01T00:00:00Z" or "2026-11-01T02:30:00.5+02:30".
//
// Surrounding spaces are ignored. A duration such as "24h" or "5m", a signed
// or fractional number, an empty value, a date in any other spelling and any
// other text return an error wrapping ErrExpiresNotAbsolute.
func ParseExpires(value string) (time.Time, error) {
	v := strings.TrimSpace(value)
	if v != "" && allDigits(v) {
		ms, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return time.Time{}, fmt.Errorf("%w: %q is out of range", ErrExpiresNotAbsolute, v)
		}
		return time.UnixMilli(ms), nil
	}
	if isRFC3339DateTime(v) {
		if ts, err := time.Parse(time.RFC3339, v); err == nil {
			return ts, nil
		}
	}
	return time.Time{}, fmt.Errorf("%w: got %q", ErrExpiresNotAbsolute, v)
}

// isRFC3339DateTime reports whether s is spelled exactly as an RFC 3339
// date-time (RFC 3339 section 5.6):
//
//	YYYY-MM-DDTHH:MM:SS[.F...](Z|+hh:mm|-hh:mm)
//
// with a four-digit year, every other field two digits and in range (month
// 01-12, day 01-31, hour 00-23, minute 00-59, second 00-59, offset hour
// 00-23, offset minute 00-59), an upper-case "T" and "Z", and an optional
// fraction of one or more digits after a ".". It checks spelling only;
// time.Parse then checks the day against the month. A leap second ("60") is
// refused because a Go time cannot hold it. time.Parse alone also accepts a
// one-digit hour, a "," before the fraction and an offset hour of 24, which
// is why this check runs first.
func isRFC3339DateTime(s string) bool {
	// Fixed part: "YYYY-MM-DDTHH:MM:SS" is 19 bytes.
	if len(s) < 20 {
		return false
	}
	if s[4] != '-' || s[7] != '-' || s[10] != 'T' || s[13] != ':' || s[16] != ':' {
		return false
	}
	if !allDigits(s[0:4]) ||
		!twoDigitsIn(s[5:7], 1, 12) ||
		!twoDigitsIn(s[8:10], 1, 31) ||
		!twoDigitsIn(s[11:13], 0, 23) ||
		!twoDigitsIn(s[14:16], 0, 59) ||
		!twoDigitsIn(s[17:19], 0, 59) {
		return false
	}
	rest := s[19:]
	if frac, ok := strings.CutPrefix(rest, "."); ok {
		n := 0
		for n < len(frac) && frac[n] >= '0' && frac[n] <= '9' {
			n++
		}
		if n == 0 {
			return false
		}
		rest = frac[n:]
	}
	if rest == "Z" {
		return true
	}
	return len(rest) == 6 &&
		(rest[0] == '+' || rest[0] == '-') &&
		twoDigitsIn(rest[1:3], 0, 23) &&
		rest[3] == ':' &&
		twoDigitsIn(rest[4:6], 0, 59)
}

// twoDigitsIn reports whether s is exactly two decimal digits whose value is
// between lo and hi inclusive.
func twoDigitsIn(s string, lo, hi int) bool {
	if len(s) != 2 || !allDigits(s) {
		return false
	}
	n := int(s[0]-'0')*10 + int(s[1]-'0')
	return n >= lo && n <= hi
}

// parseTimeBefore reads the value of a "time < N" caveat: Unix time in
// seconds, decimal digits only.
func parseTimeBefore(value string) (time.Time, error) {
	v := strings.TrimSpace(value)
	if v == "" || !allDigits(v) {
		return time.Time{}, fmt.Errorf("invalid time caveat: want Unix seconds, got %q", v)
	}
	sec, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid time caveat: %q is out of range", v)
	}
	return time.Unix(sec, 0), nil
}

func allDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}
