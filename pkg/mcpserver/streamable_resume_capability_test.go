package mcpserver

import "testing"

// The capability a wrapper asks before it forwards an unknown session id
// (satgate-enterprise wrapMCPSessionIdentity): true only where resume runs.
func TestStreamableResumesStreamSessionsOnlyInHeaderMode(t *testing.T) {
	if !newResumeInstance(t, newResumeTokens()).sse.ResumesStreamSessions() {
		t.Fatal("header auth mode: ResumesStreamSessions() = false")
	}
	cfg := &Config{
		Server:  ServerConfig{Transport: "sse", Name: "resume-capability", Version: "test"},
		Auth:    AuthConfig{Mode: "none"},
		Budget:  BudgetConfig{Limit: 1, FailMode: "closed"},
		Logging: LoggingConfig{Level: "error"},
	}
	proxy, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if NewSSEServer(proxy, ":0").ResumesStreamSessions() {
		t.Fatal("auth mode none: ResumesStreamSessions() = true")
	}
	var nilServer *SSEServer
	if nilServer.ResumesStreamSessions() {
		t.Fatal("nil server: ResumesStreamSessions() = true")
	}
	// The method is visible on the handler the server hands out.
	if _, ok := newResumeInstance(t, newResumeTokens()).sse.Handler().(interface{ ResumesStreamSessions() bool }); !ok {
		t.Fatal("Handler() does not expose ResumesStreamSessions")
	}
}
