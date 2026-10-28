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

// ParseExpires reads the value of an "expires" caveat. Two absolute forms are
// accepted, and nothing else:
//
//   - Unix time in milliseconds, written as decimal digits only, the form Mint
//     and MintL402 write (for example "1793577600000");
//   - an RFC 3339 time with a zone (for example "2026-11-01T00:00:00Z").
//
// Surrounding spaces are ignored. A duration such as "24h" or "5m", a signed
// or fractional number, an empty value and any other text return an error
// wrapping ErrExpiresNotAbsolute.
func ParseExpires(value string) (time.Time, error) {
	v := strings.TrimSpace(value)
	if v != "" && allDigits(v) {
		ms, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return time.Time{}, fmt.Errorf("%w: %q is out of range", ErrExpiresNotAbsolute, v)
		}
		return time.UnixMilli(ms), nil
	}
	if ts, err := time.Parse(time.RFC3339, v); err == nil {
		return ts, nil
	}
	return time.Time{}, fmt.Errorf("%w: got %q", ErrExpiresNotAbsolute, v)
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
