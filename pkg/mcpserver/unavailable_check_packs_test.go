package mcpserver

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
)

// outagePackRecorder exercises the public recorder contract with actual
// Ed25519 signatures and fsynced packs. It is a test adapter, not the enterprise
// recorder. Never serialize MCPDecision wholesale: it contains raw identities.
type outagePackRecorder struct {
	*eaWindowRecorder
	mu     sync.Mutex
	input  io.WriteCloser
	output *json.Decoder
}

func (r *outagePackRecorder) exchange(v any) (json.RawMessage, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := json.NewEncoder(r.input).Encode(v); err != nil {
		return nil, err
	}
	var raw json.RawMessage
	if err := r.output.Decode(&raw); err != nil {
		return nil, err
	}
	var failure struct {
		Error  string `json:"error"`
		Detail string `json:"detail"`
	}
	if err := json.Unmarshal(raw, &failure); err != nil {
		return nil, err
	}
	if failure.Error != "" {
		return nil, errors.New(failure.Error + ": " + failure.Detail)
	}
	return raw, nil
}
func outageHash(s string) string {
	h := sha256.Sum256([]byte(s))
	return "sha256:" + base64.RawURLEncoding.EncodeToString(h[:])
}
func (r *outagePackRecorder) RecordMCPDecision(ctx context.Context, d MCPDecision) (*MCPEvidence, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	projection := map[string]any{"decision": d.Decision, "reason": d.DecisionReason, "budget_not_evaluated": d.BudgetNotEvaluated,
		"request_id": d.RequestID, "route_or_tool": d.RouteOrTool, "policy_mode": d.PolicyMode,
		"capability_hash": outageHash(d.TokenID), "key_hash": outageHash(d.BudgetID)}
	raw, err := r.exchange(map[string]any{"action": "record", "decision": projection})
	if err != nil {
		return nil, err
	}
	var ev MCPEvidence
	if err = json.Unmarshal(raw, &ev); err != nil {
		return nil, err
	}
	return &ev, nil
}

func TestUnavailableCheck_RealPacksAndRecord(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	dir := os.Getenv("OUTAGE_PACK_DIR")
	if dir == "" {
		dir = t.TempDir()
	}
	// Child lifetime is owned by this test. Closing stdin and Wait reap it on
	// success; the context kills and reaps it even when an assertion fails.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cmd := exec.CommandContext(ctx, "python3", "-B", filepath.Join(root, "tools/mcp_unavailable_check_fixture.py"), dir)
	cmd.Env = append(os.Environ(), "PYTHONDONTWRITEBYTECODE=1", "PYTHONPATH="+filepath.Join(root, "demo/_vendor"))
	in, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err = cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		in.Close()
		cancel()
		if err := cmd.Wait(); err != nil && !errors.Is(ctx.Err(), context.Canceled) {
			t.Error("recorder process failed")
		}
	}()
	rec := &outagePackRecorder{input: in, output: json.NewDecoder(out)}
	// Each call crosses real JSON-RPC transports, signs one archive entry, and
	// returns handles bound to that exact file. No synthetic replacement of a
	// proxy response, recorder decision or checker verdict is used.
	count := 0
	for _, c := range outageCases() {
		if c.reason == "" {
			continue
		}
		for _, road := range eaTransports {
			t.Run(c.name+"/"+road.name, func(t *testing.T) {
				e, s := outageEnv(t, c, false)
				outageSwitch(t, e.proxy, true)
				rec.eaWindowRecorder = newEAWindowRecorder(e.base)
				e.proxy.SetEvidenceRecorder(rec)
				if err = e.proxy.SetSignEveryAction(true); err != nil {
					t.Fatal(err)
				}
				closeRoad := road.via(t, e)
				defer closeRoad()
				before := e.remaining()
				resp := outageCall(e, c)
				data := outageEvidence(resp)
				if resp.Error == nil || data["receipt_id"] == nil {
					t.Fatal("real recorder returned no handles")
				}
				count++
				raw, err := os.ReadFile(filepath.Join(dir, "pack-"+mustJSONString(count)+".json"))
				if err != nil {
					t.Fatal(err)
				}
				var pack map[string]any
				if err = json.Unmarshal(raw, &pack); err != nil {
					t.Fatal(err)
				}
				for _, field := range []string{"receipt_id", "receipt_hash", "evidence_pack_id", "evidence_url"} {
					if data[field] != pack[field] {
						t.Fatal("response is not bound to archived pack")
					}
				}
				if pack["decision_reason"] != c.reason || e.rr.forwarded() != 0 || e.remaining() != before {
					t.Fatal("real pack reason/effects mismatch")
				}
				if a, b, c, d := s.counts(); a+b+c+d != 0 {
					t.Fatal("pre-hold stop touched budget")
				}
			})
		}
	}
	// Independently fail signing and archival, after successful proof preflight:
	// neither creates a new durable refusal, returns handles, sends or debits.
	for _, mode := range []string{"sign", "archive"} {
		if _, err = rec.exchange(map[string]any{"action": "failure", "mode": mode}); err != nil {
			t.Fatal(err)
		}
		e, _ := outageEnv(t, outageCases()[0], false)
		outageSwitch(t, e.proxy, true)
		rec.eaWindowRecorder = newEAWindowRecorder(e.base)
		e.proxy.SetEvidenceRecorder(rec)
		if err = e.proxy.SetSignEveryAction(true); err != nil {
			t.Fatal(err)
		}
		resp := outageCall(e, outageCases()[0])
		if !eaIsProofUnavailable(resp) || outageEvidence(resp)["receipt_id"] != nil || e.rr.forwarded() != 0 {
			t.Fatal("failed real recorder produced refusal")
		}
	}
	raw, err := rec.exchange(map[string]any{"action": "verify"})
	if err != nil {
		t.Fatal(err)
	}
	var result struct {
		Packs  int            `json:"packs"`
		Counts map[string]int `json:"counts_by_decision_reason"`
	}
	if err = json.Unmarshal(raw, &result); err != nil {
		t.Fatal(err)
	}
	if result.Packs != count {
		t.Fatal("failure created an extra durable pack")
	}
	want := map[string]int{"tool_price_unavailable": 2, "tool_controls_unavailable": 6, "budget_backend_error": 2, "pending_capacity_exhausted": 2}
	if len(result.Counts) != len(want) {
		t.Fatal("wrong record reason count")
	}
	for reason, n := range want {
		if result.Counts[reason] != n {
			t.Fatal("record counts_by_decision_reason mismatch")
		}
	}
	t.Logf("%d emitted packs valid and trusted in tools, demo and frozen bundle; whole record valid, trusted and complete in tools and bundle", count)
}
