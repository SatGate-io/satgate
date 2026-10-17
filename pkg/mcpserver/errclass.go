package mcpserver

import (
	"context"
	"errors"
	"fmt"
)

// errClass is the form in which an error read from a HandoffLimiter (a
// database, in a gateway) may be written to a log line: a closed
// classification, never the error's text.
//
//   - anything in the chain with a SQLState() method (a database driver's
//     error) is "database error (sqlstate XXXXX)";
//   - a context error is "canceled" or "deadline_exceeded";
//   - any other error is "driver_error" and the Go type name of the innermost
//     error of its chain (for example "driver_error *net.OpError").
//
// The text of an error can hold anything its producer put there: a row the
// database refused, a tenant id, an email address. None of it is kept. A nil
// error stays nil.
func errClass(err error) error {
	if err == nil {
		return nil
	}
	var sq interface{ SQLState() string }
	if errors.As(err, &sq) {
		return classifiedErr("database error (sqlstate " + sqlstateOrUnknown(sq.SQLState()) + ")")
	}
	if errors.Is(err, context.Canceled) {
		return classifiedErr("canceled")
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return classifiedErr("deadline_exceeded")
	}
	return classifiedErr("driver_error " + innermostType(err))
}

// classifiedErr is an error whose text is one of errClass's classifications.
type classifiedErr string

func (c classifiedErr) Error() string { return string(c) }

// sqlstateOrUnknown keeps a SQLSTATE only when it is one: five characters,
// digits and upper-case letters.
func sqlstateOrUnknown(code string) string {
	if len(code) != 5 {
		return "?????"
	}
	for _, r := range code {
		if !(r >= '0' && r <= '9' || r >= 'A' && r <= 'Z') {
			return "?????"
		}
	}
	return code
}

// innermostType follows the single-error Unwrap chain to its end and returns
// that error's Go type. A joined error (Unwrap() []error) ends the walk.
func innermostType(err error) string {
	for {
		next := errors.Unwrap(err)
		if next == nil {
			return fmt.Sprintf("%T", err)
		}
		err = next
	}
}
