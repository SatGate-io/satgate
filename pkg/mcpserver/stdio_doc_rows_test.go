package mcpserver

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/satgate-io/satgate/pkg/governance"
)

// The stdio table of docs/guides/mcp-gateway.md says, row by row, whether a
// supplied token is checked, and the paragraph under it says when both
// supplied credentials (params._meta.token and the connection credential)
// are checked. These tests hold the guide to the stdio dispatch: both
// credentials are checked on the rows that check a token, and the row of
// local replies and notifications checks neither.

// countingChecker counts lineage checks and passes them to next.
type countingChecker struct {
	next  LineageRevocationChecker
	calls atomic.Int64
}

func (c *countingChecker) IsAnyBanned(ctx context.Context, sigs []string) (bool, error) {
	c.calls.Add(1)
	return c.next.IsAnyBanned(ctx, sigs)
}

type stdioDocRow struct {
	methods      []string
	notification bool // the row covers notifications
	checked      bool
}

// stdioDocTable returns the rows of the stdio table and the paragraph that
// follows it.
func stdioDocTable(t *testing.T) ([]stdioDocRow, string) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "docs", "guides", "mcp-gateway.md"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(string(raw), "\n")
	start := -1
	for i, l := range lines {
		if strings.HasPrefix(l, "| stdio request |") {
			start = i
			break
		}
	}
	if start < 0 {
		t.Fatal("no stdio table in the guide")
	}
	method := regexp.MustCompile("`([a-z]+(?:/[a-z_]+)?)`")
	var rows []stdioDocRow
	i := start + 2 // header and separator
	for ; i < len(lines) && strings.HasPrefix(lines[i], "|"); i++ {
		cells := strings.Split(strings.Trim(lines[i], "|"), "|")
		if len(cells) < 3 {
			t.Fatalf("stdio table row %q", lines[i])
		}
		var row stdioDocRow
		for _, m := range method.FindAllStringSubmatch(cells[0], -1) {
			row.methods = append(row.methods, m[1])
		}
		row.notification = strings.Contains(cells[0], "notifications")
		row.checked = !strings.HasPrefix(strings.TrimSpace(cells[1]), "not checked")
		if len(row.methods) == 0 {
			t.Fatalf("stdio table row names no method: %q", lines[i])
		}
		rows = append(rows, row)
	}
	var para []string
	for i++; i < len(lines) && strings.TrimSpace(lines[i]) != ""; i++ {
		para = append(para, lines[i])
	}
	return rows, strings.Join(strings.Fields(strings.Join(para, " ")), " ")
}

// Every row of the table is what the stdio dispatch does with two supplied
// credentials, one a descendant of a banned token, in either position and
// under both switch states: a row that checks a token refuses ("token
// revoked"); the row that does not answers initialize and ping and drops
// notifications without consulting the checker.
func TestMCPStdioDoc_TableRowsMatchTwoCredentialDispatch(t *testing.T) {
	rows, _ := stdioDocTable(t)
	var checkedRows, exemptRows int
	for _, sign := range []bool{false, true} {
		for _, where := range []string{"connection", "meta"} {
			t.Run(switchName(sign)+"/banned_"+where, func(t *testing.T) {
				c := newStdioCreds(t)
				gov := governance.NewService(nil)
				gov.Ban(c.badRoot.Signature, "test", "admin")
				checker := &countingChecker{next: gov}
				conn, meta := c.bad, c.healthy
				if where == "meta" {
					conn, meta = c.healthy, c.bad
				}
				r := stdioRigWith(t, checker, conn, sign)
				checkedRows, exemptRows = 0, 0
				for _, row := range rows {
					if row.checked {
						checkedRows++
						for _, m := range row.methods {
							resp := r.call(m, meta)
							if resp.Error == nil || !strings.Contains(resp.Error.Message, "token revoked") {
								t.Fatalf("%s: the guide says checked; got %+v", m, resp.Error)
							}
						}
						continue
					}
					exemptRows++
					before := checker.calls.Load()
					for _, m := range row.methods {
						if resp := r.call(m, meta); resp.Error != nil {
							t.Fatalf("%s: the guide says not checked; refused %+v", m, resp.Error)
						}
					}
					if row.notification {
						note := map[string]any{"jsonrpc": "2.0", "method": MethodInitialized, "params": map[string]any{"_meta": map[string]string{"token": meta}}}
						if err := r.client.WriteMessage(r.ctx, mustJSON(t, note)); err != nil {
							t.Fatal(err)
						}
						// The next reply answers the next request, not the
						// notification.
						resp := r.call(MethodPing, meta)
						if resp.Error != nil || string(resp.ID) != strconv.Itoa(r.nextID) {
							t.Fatalf("after a notification: reply id %s error %+v, want the ping's reply", resp.ID, resp.Error)
						}
					}
					if n := checker.calls.Load() - before; n != 0 {
						t.Fatalf("the guide says not checked; the checker was consulted %d times", n)
					}
				}
			})
		}
	}
	if checkedRows != 2 || exemptRows != 1 {
		t.Fatalf("stdio table: %d checking rows and %d exempt rows, want 2 and 1", checkedRows, exemptRows)
	}
}

// The paragraph under the table scopes the two-credential check to the rows
// that check a token and keeps the local-reply exemption: it does not claim
// both credentials are checked on every row, and it names initialize, ping
// and notifications as checking neither.
func TestMCPStdioDoc_TwoCredentialCheckScopedToCheckingRows(t *testing.T) {
	_, para := stdioDocTable(t)
	if !strings.Contains(para, "both are checked") {
		t.Fatalf("the paragraph under the stdio table no longer states the two-credential check: %q", para)
	}
	if strings.Contains(para, "every row") {
		t.Fatalf("the paragraph claims both credentials are checked on every row, but initialize, ping and notifications are not checked: %q", para)
	}
	for _, want := range []string{"`initialize`", "`ping`", "notifications", "neither credential is checked"} {
		if !strings.Contains(para, want) {
			t.Fatalf("the paragraph does not keep the local-reply exemption (%s): %q", want, para)
		}
	}
}

// Astra's case: initialize and ping are answered with a refused connection
// credential beside a healthy params._meta.token, under both switch states,
// and the checker is never consulted.
func TestMCPStdio_LocalRepliesLeaveBothCredentialsUnchecked(t *testing.T) {
	for _, sign := range []bool{false, true} {
		t.Run(switchName(sign), func(t *testing.T) {
			c := newStdioCreds(t)
			checker := &countingChecker{next: refuseAll{}}
			r := stdioRigWith(t, checker, c.bad, sign)
			for _, m := range []string{MethodInitialize, MethodPing} {
				if resp := r.call(m, c.healthy); resp.Error != nil {
					t.Fatalf("%s refused: %+v", m, resp.Error)
				}
			}
			if n := checker.calls.Load(); n != 0 {
				t.Fatalf("local replies consulted the checker %d times", n)
			}
		})
	}
}

// refuseAll reports every lineage banned.
type refuseAll struct{}

func (refuseAll) IsAnyBanned(context.Context, []string) (bool, error) { return true, nil }
