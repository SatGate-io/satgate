package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const redactMarker = "UPSTREAM-TEXT-9c1e"

func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	old := log.Logger
	oldLevel := zerolog.GlobalLevel()
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	log.Logger = zerolog.New(&syncWriter{w: &buf})
	t.Cleanup(func() { log.Logger = old; zerolog.SetGlobalLevel(oldLevel) })
	return &buf
}

type syncWriter struct {
	mu sync.Mutex
	w  *bytes.Buffer
}

func (s *syncWriter) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.w.Write(p)
}

var redactModes = map[string]func(w http.ResponseWriter, method string, id json.RawMessage){
	"initialize JSON-RPC error": func(w http.ResponseWriter, method string, id json.RawMessage) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32000,"message":%q}}`, id, redactMarker)
	},
	"tools/list JSON-RPC error": func(w http.ResponseWriter, method string, id json.RawMessage) {
		w.Header().Set("Content-Type", "application/json")
		switch method {
		case "initialize":
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":{}}`, id)
		case "notifications/initialized":
			w.WriteHeader(http.StatusAccepted)
		default:
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32601,"message":%q}}`, id, redactMarker)
		}
	},
	"HTTP 500 body": func(w http.ResponseWriter, method string, id json.RawMessage) {
		http.Error(w, redactMarker, http.StatusInternalServerError)
	},
}

func runRedactStart(t *testing.T, redact bool, names ...string) (string, *bytes.Buffer) {
	t.Helper()
	buf := captureLogs(t)
	cfg := map[string]UpstreamConfig{}
	for i, name := range names {
		mode := redactModes[name]
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var env struct {
				Method string          `json:"method"`
				ID     json.RawMessage `json:"id"`
			}
			_ = json.NewDecoder(r.Body).Decode(&env)
			if len(env.ID) == 0 {
				env.ID = json.RawMessage("1")
			}
			mode(w, env.Method, env.ID)
		}))
		t.Cleanup(srv.Close)
		cfg[fmt.Sprintf("up%d", i)] = UpstreamConfig{Transport: "http", URL: srv.URL, Timeout: 5 * time.Second}
	}
	m := NewUpstreamManager(cfg, nil, "", true)
	m.SetRedactUpstreamText(redact)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	err := m.Start(ctx)
	_ = m.Close()
	if err == nil {
		t.Fatal("expected Start to fail")
	}
	return err.Error(), buf
}

func TestRedactUpstreamTextKeepsUpstreamTextOutOfErrorsAndLogs(t *testing.T) {
	for name := range redactModes {
		for _, n := range []int{1, 2} { // 1: sequential start, 2: parallel start
			names := make([]string, n)
			for i := range names {
				names[i] = name
			}
			t.Run(fmt.Sprintf("%s/%d upstreams", name, n), func(t *testing.T) {
				errText, logs := runRedactStart(t, true, names...)
				if strings.Contains(errText, redactMarker) || strings.Contains(logs.String(), redactMarker) {
					t.Fatalf("upstream text leaked\nerr: %s\nlogs: %s", errText, logs.String())
				}
			})
		}
	}
}

// Unset, nothing changes: the text is still there, as before.
func TestRedactUpstreamTextOffKeepsExistingBehaviour(t *testing.T) {
	errText, _ := runRedactStart(t, false, "initialize JSON-RPC error")
	if !strings.Contains(errText, "initialize error: "+redactMarker) {
		t.Fatalf("default behaviour changed: %s", errText)
	}
}

// Redirect refusal keeps working and names no target when redacted.
func TestRedactUpstreamTextRedirectRefusalNamesNoTarget(t *testing.T) {
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer other.Close()
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, other.URL+"/x", http.StatusTemporaryRedirect)
	}))
	defer up.Close()
	for _, redact := range []bool{false, true} {
		tr := NewStreamableHTTPTransport(up.URL, map[string]string{"Authorization": "Bearer x"}, false, true)
		tr.SetRedactUpstreamText(redact)
		err := tr.WriteMessage(context.Background(), []byte(`{"jsonrpc":"2.0","id":1,"method":"ping"}`))
		tr.Close()
		if err == nil || !strings.Contains(err.Error(), "upstream redirect status 307") {
			t.Fatalf("redirect not refused (redact=%v): %v", redact, err)
		}
		has := strings.Contains(err.Error(), strings.TrimPrefix(other.URL, "http://"))
		if has == redact {
			t.Fatalf("redact=%v but target present=%v: %v", redact, has, err)
		}
	}
}
