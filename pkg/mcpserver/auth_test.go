package mcpserver

import (
	"context"
	"testing"

	"github.com/satgate-io/satgate/pkg/macaroon"
	"time"
)

func TestNoAuthAuthenticator(t *testing.T) {
	auth := &NoAuthAuthenticator{DefaultTokenID: "test"}
	info, err := auth.Verify(context.Background(), "anything")
	if err != nil {
		t.Fatal(err)
	}
	if info.TokenID != "test" {
		t.Errorf("expected test, got %s", info.TokenID)
	}
	if info.Scope != "*" {
		t.Errorf("expected *, got %s", info.Scope)
	}
}

func TestStaticTokenAuthenticator(t *testing.T) {
	auth := &StaticTokenAuthenticator{Token: "secret123", TokenID: "tok1"}

	// Valid token
	info, err := auth.Verify(context.Background(), "secret123")
	if err != nil {
		t.Fatal(err)
	}
	if info.TokenID != "tok1" {
		t.Errorf("expected tok1, got %s", info.TokenID)
	}

	// Invalid token
	_, err = auth.Verify(context.Background(), "wrong")
	if err == nil {
		t.Fatal("expected error for invalid token")
	}

	// Empty token
	_, err = auth.Verify(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty token")
	}
}

func TestMacaroonAuthenticator(t *testing.T) {
	svc, err := macaroon.NewService("test-root-key")
	if err != nil {
		t.Fatal(err)
	}

	auth := &MacaroonAuthenticator{Service: svc}

	// Mint a token
	mac, err := svc.Mint("api:*", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	token := svc.Encode(mac)

	// Verify
	info, err := auth.Verify(context.Background(), token)
	if err != nil {
		t.Fatal(err)
	}
	if info.TokenID == "" {
		t.Error("expected non-empty token ID")
	}
	if info.Scope != "api:*" {
		t.Errorf("expected api:*, got %s", info.Scope)
	}
	if info.RawToken != token {
		t.Error("expected raw token to be preserved")
	}

	// Verify with Bearer prefix
	info2, err := auth.Verify(context.Background(), "Bearer "+token)
	if err != nil {
		t.Fatal(err)
	}
	if info2.TokenID != info.TokenID {
		t.Error("Bearer prefix should be stripped")
	}

	// Invalid token
	_, err = auth.Verify(context.Background(), "garbage")
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
}

func TestMacaroonAuthenticator_WithBudgetID(t *testing.T) {
	svc, err := macaroon.NewService("test-root-key")
	if err != nil {
		t.Fatal(err)
	}

	auth := &MacaroonAuthenticator{Service: svc}

	// Mint a token with budget_id caveat
	mac, err := svc.Mint("api:*", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	mac.AddCaveat("budget_id", "budget-abc")
	mac.Signature = svc.RecalculateSignature(mac)
	token := svc.Encode(mac)

	info, err := auth.Verify(context.Background(), token)
	if err != nil {
		t.Fatal(err)
	}
	if info.BudgetID != "budget-abc" {
		t.Errorf("expected budget-abc, got %s", info.BudgetID)
	}
}

func TestNewAuthenticator(t *testing.T) {
	// None mode
	a, err := NewAuthenticator(AuthConfig{Mode: "none"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := a.(*NoAuthAuthenticator); !ok {
		t.Error("expected NoAuthAuthenticator")
	}

	// Config mode
	a, err = NewAuthenticator(AuthConfig{Mode: "config", Token: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := a.(*StaticTokenAuthenticator); !ok {
		t.Error("expected StaticTokenAuthenticator")
	}

	// Header mode
	a, err = NewAuthenticator(AuthConfig{Mode: "header", RootKey: "testkey"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := a.(*MacaroonAuthenticator); !ok {
		t.Error("expected MacaroonAuthenticator")
	}

	// Config mode without token
	_, err = NewAuthenticator(AuthConfig{Mode: "config"})
	if err == nil {
		t.Fatal("expected error for config mode without token")
	}
}
