package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"
	"testing"
)

// Round 5 (Sol B2): the two refusal lines for a stored handoff limit that
// cannot be used carry no fingerprint of the parent (TokenID is a hash of the
// macaroon's identifier and signature) and no free error text.

type sqlStateErr struct{ code, msg string }

func (e *sqlStateErr) Error() string    { return e.msg }
func (e *sqlStateErr) SQLState() string { return e.code }

func logLines(t *testing.T, out string) []map[string]any {
	t.Helper()
	var lines []map[string]any
	for _, raw := range strings.Split(strings.TrimSpace(out), "\n") {
		if raw == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(raw), &m); err != nil {
			t.Fatalf("log line is not JSON: %q", raw)
		}
		lines = append(lines, m)
	}
	return lines
}

// DEFECT TEST: on 591a3090 the missing-row line has a "parent" field holding
// parent.TokenID, and the read-error line has it and the error's own text.
func TestDelegateStoredRefusalLogsNoFingerprintOrErrorText(t *testing.T) {
	const canary = "stored-refusal-error-canary-4711"
	cases := []struct {
		name    string
		prepare func(r *handoffRig, parent *TokenInfo)
		message string
		class   string // wanted error field; "" = no error field
	}{
		{"missing row", func(r *handoffRig, p *TokenInfo) { r.store.missing[p.BudgetID] = true },
			"delegation refused: the parent has no stored row", ""},
		{"arbitrary error", func(r *handoffRig, p *TokenInfo) { r.store.err = errors.New(canary) },
			"delegation refused: the stored handoff limit could not be read", "driver_error *errors.errorString"},
		{"database error", func(r *handoffRig, p *TokenInfo) { r.store.err = fmt.Errorf("read: %w", &sqlStateErr{"P0001", canary}) },
			"delegation refused: the stored handoff limit could not be read", "database error (sqlstate P0001)"},
		{"canceled", func(r *handoffRig, p *TokenInfo) { r.store.err = fmt.Errorf("%s: %w", canary, context.Canceled) },
			"delegation refused: the stored handoff limit could not be read", "canceled"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			buf := captureLogs(t)
			r := newHandoffRig(t)
			parent := r.root(nil)
			tc.prepare(r, parent)
			if resp := r.handoff(parent, 0); resp.Error == nil {
				t.Fatal("want a refusal")
			}
			out := buf.String()
			for _, secret := range []string{parent.TokenID, parent.BudgetID, parent.RawToken, canary} {
				if secret != "" && strings.Contains(out, secret) {
					t.Fatalf("the refusal log carries %q:\n%s", secret, out)
				}
			}
			var found bool
			for _, line := range logLines(t, out) {
				if line["message"] != tc.message {
					continue
				}
				found = true
				if _, has := line["parent"]; has {
					t.Fatalf("the line has a parent field: %v", line)
				}
				got, _ := line["error"].(string)
				if got != tc.class {
					t.Fatalf("error field %q, want %q", got, tc.class)
				}
				for k := range line {
					switch k {
					case "level", "message", "error", "time":
					default:
						t.Fatalf("unexpected field %q on the line: %v", k, line)
					}
				}
			}
			if !found {
				t.Fatalf("line %q not logged:\n%s", tc.message, out)
			}
		})
	}
}

func TestErrClassIsClosed(t *testing.T) {
	const secret = "tenant 7c1d9a42 owner a@customer.example"
	for _, c := range []struct {
		err  error
		want string
	}{
		{&sqlStateErr{"P0001", secret}, "database error (sqlstate P0001)"},
		{fmt.Errorf("store: %w", &sqlStateErr{"23505", secret}), "database error (sqlstate 23505)"},
		{&sqlStateErr{secret, secret}, "database error (sqlstate ?????)"},
		{&sqlStateErr{"p0001", secret}, "database error (sqlstate ?????)"},
		{fmt.Errorf("%s: %w", secret, context.Canceled), "canceled"},
		{fmt.Errorf("%s: %w", secret, context.DeadlineExceeded), "deadline_exceeded"},
		{errors.New(secret), "driver_error *errors.errorString"},
		{fmt.Errorf("%s", secret), "driver_error *errors.errorString"},
		{fmt.Errorf("list: %w", &net.OpError{Op: "dial", Err: errors.New(secret)}), "driver_error *errors.errorString"},
		{fmt.Errorf("list: %w", &net.OpError{Op: "dial"}), "driver_error *net.OpError"},
		{errors.Join(errors.New(secret), errors.New(secret)), "driver_error *errors.joinError"},
	} {
		got := errClass(c.err).Error()
		if got != c.want {
			t.Errorf("errClass = %q, want %q", got, c.want)
		}
		if strings.Contains(got, "7c1d9a42") || strings.Contains(got, "customer") {
			t.Errorf("class %q carries the error's text", got)
		}
	}
	if errClass(nil) != nil {
		t.Error("nil must stay nil")
	}
}
