package governance

import (
	"testing"
	"time"
)

func TestService_BanUnban(t *testing.T) {
	svc := NewService(nil)

	signature := "abc123"

	// Initially not banned
	if svc.IsBanned(signature) {
		t.Error("Token should not be banned initially")
	}

	// Ban the token
	svc.Ban(signature, "test reason", "admin")

	// Should now be banned
	if !svc.IsBanned(signature) {
		t.Error("Token should be banned after Ban()")
	}

	// Unban the token
	if !svc.Unban(signature) {
		t.Error("Unban should return true for banned token")
	}

	// Should no longer be banned
	if svc.IsBanned(signature) {
		t.Error("Token should not be banned after Unban()")
	}
}

func TestService_GetBanList(t *testing.T) {
	svc := NewService(nil)

	// Ban some tokens
	svc.Ban("token1", "reason1", "admin")
	svc.Ban("token2", "reason2", "operator")
	svc.Ban("token3", "reason3", "admin")

	banList := svc.GetBanList()

	if len(banList) != 3 {
		t.Errorf("Expected 3 banned tokens, got %d", len(banList))
	}
}

func TestService_RecordUsage(t *testing.T) {
	svc := NewService(nil)

	signature := "token123"

	// Record some usage
	svc.RecordUsage(signature, "/api/users")
	svc.RecordUsage(signature, "/api/users")
	svc.RecordUsage(signature, "/api/orders")

	stats := svc.GetUsage(signature)

	if stats == nil {
		t.Fatal("Expected usage stats, got nil")
	}

	if stats.TotalRequests != 3 {
		t.Errorf("Expected 3 total requests, got %d", stats.TotalRequests)
	}

	if stats.Routes["/api/users"] != 2 {
		t.Errorf("Expected 2 requests to /api/users, got %d", stats.Routes["/api/users"])
	}

	if stats.Routes["/api/orders"] != 1 {
		t.Errorf("Expected 1 request to /api/orders, got %d", stats.Routes["/api/orders"])
	}
}

func TestService_GetAllTokenInfo(t *testing.T) {
	svc := NewService(nil)

	// Record usage for active tokens
	svc.RecordUsage("active1", "/api/test")
	svc.RecordUsage("active2", "/api/test")

	// Ban a token
	svc.Ban("banned1", "compromised", "admin")

	info := svc.GetAllTokenInfo()

	// Should have 3 tokens (2 active + 1 banned)
	if len(info) != 3 {
		t.Errorf("Expected 3 tokens, got %d", len(info))
	}

	// Count by status
	activeCount := 0
	bannedCount := 0
	for _, token := range info {
		switch token.Status {
		case "active":
			activeCount++
		case "banned":
			bannedCount++
		}
	}

	if activeCount != 2 {
		t.Errorf("Expected 2 active tokens, got %d", activeCount)
	}

	if bannedCount != 1 {
		t.Errorf("Expected 1 banned token, got %d", bannedCount)
	}
}

func TestService_GetStats(t *testing.T) {
	svc := NewService(nil)

	// Register minted tokens (required to count as "active")
	future := time.Now().Add(time.Hour)
	svc.RegisterMint("token1", "api:read", future)
	svc.RegisterMint("token2", "api:write", future)

	// Record usage
	svc.RecordUsage("token1", "/api/test")
	svc.RecordUsage("token1", "/api/test")
	svc.RecordUsage("token2", "/api/test")

	// Ban a separate token
	svc.RegisterMint("token3", "api:*", future)
	svc.Ban("token3", "test", "admin")

	stats := svc.GetStats()

	if stats["active"].(int) != 2 {
		t.Errorf("Expected 2 active, got %v", stats["active"])
	}

	if stats["banned"].(int) != 1 {
		t.Errorf("Expected 1 banned, got %v", stats["banned"])
	}

	if stats["totalRequests"].(int64) != 3 {
		t.Errorf("Expected 3 total requests, got %v", stats["totalRequests"])
	}
}

func TestService_Reset(t *testing.T) {
	svc := NewService(nil)

	// Add some data
	svc.RecordUsage("token1", "/api/test")
	svc.Ban("token2", "test", "admin")

	// Reset
	svc.Reset()

	// Check everything is empty
	stats := svc.GetStats()
	if stats["active"].(int) != 0 {
		t.Error("Expected 0 active tokens after reset")
	}
	if stats["banned"].(int) != 0 {
		t.Error("Expected 0 banned tokens after reset")
	}
}

func TestService_Concurrent(t *testing.T) {
	svc := NewService(nil)

	// Run concurrent operations
	done := make(chan bool)

	for i := 0; i < 100; i++ {
		go func(i int) {
			svc.RecordUsage("concurrent-token", "/api/test")
			done <- true
		}(i)
	}

	for i := 0; i < 50; i++ {
		go func(i int) {
			if i%2 == 0 {
				svc.Ban("ban-token", "test", "admin")
			} else {
				svc.Unban("ban-token")
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 150; i++ {
		<-done
	}

	// Should not panic and data should be consistent
	stats := svc.GetUsage("concurrent-token")
	if stats.TotalRequests != 100 {
		t.Errorf("Expected 100 requests, got %d", stats.TotalRequests)
	}
}

func BenchmarkRecordUsage(b *testing.B) {
	svc := NewService(nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		svc.RecordUsage("bench-token", "/api/test")
	}
}

func BenchmarkIsBanned(b *testing.B) {
	svc := NewService(nil)

	// Add some banned tokens
	for i := 0; i < 1000; i++ {
		svc.Ban("banned-"+string(rune(i)), "test", "admin")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		svc.IsBanned("test-token")
	}
}
