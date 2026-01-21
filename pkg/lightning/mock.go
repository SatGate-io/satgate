package lightning

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"
)

// MockProvider implements the Lightning Provider interface for testing
type MockProvider struct {
	invoices map[string]*Invoice // paymentHash -> invoice
	paid     map[string]bool
	balance  int64
	mu       sync.RWMutex
}

// NewMockProvider creates a new mock provider
func NewMockProvider() *MockProvider {
	return &MockProvider{
		invoices: make(map[string]*Invoice),
		paid:     make(map[string]bool),
		balance:  1000000, // 1M sats
	}
}

// CreateInvoice creates a mock invoice
func (m *MockProvider) CreateInvoice(amountSats int64, memo string) (*Invoice, error) {
	// Generate random preimage
	preimageBytes := make([]byte, 32)
	rand.Read(preimageBytes)
	preimage := hex.EncodeToString(preimageBytes)
	
	// Calculate payment hash
	paymentHashBytes := sha256.Sum256(preimageBytes)
	paymentHash := hex.EncodeToString(paymentHashBytes[:])

	// Generate realistic-looking mock BOLT11
	// Format: lnbc + amount + random data (bech32-like) + signature
	// This is NOT a valid invoice - it's for demo/testing UI only
	amountStr := "1u" // Default to 1u (100 sats)
	if amountSats >= 1000 {
		amountStr = "1m" // 1000 sats
	} else if amountSats >= 100 {
		amountStr = "100u" // 100 sats
	} else if amountSats >= 10 {
		amountStr = "10u" // 10 sats  
	}
	
	// Generate enough random data to look realistic (real invoices are 200+ chars)
	extraData := make([]byte, 100)
	rand.Read(extraData)
	bolt11 := "lnbc" + amountStr + "1p" + hex.EncodeToString(paymentHashBytes[:])[:40] + 
		"pp" + hex.EncodeToString(extraData)[:120] + 
		"qp" + hex.EncodeToString(preimageBytes)[:32]

	invoice := &Invoice{
		Bolt11:      bolt11,
		PaymentHash: paymentHash,
		Preimage:    preimage,
		Amount:      amountSats,
		Memo:        memo,
		ExpiresAt:   time.Now().Add(10 * time.Minute).Unix(),
	}

	m.mu.Lock()
	m.invoices[paymentHash] = invoice
	m.mu.Unlock()

	return invoice, nil
}

// CheckPayment always returns true for mock
func (m *MockProvider) CheckPayment(paymentHash string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if paid, ok := m.paid[paymentHash]; ok {
		return paid, nil
	}
	return false, nil
}

// SimulatePayment marks an invoice as paid (for testing)
func (m *MockProvider) SimulatePayment(paymentHash string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.paid[paymentHash] = true
	if inv, ok := m.invoices[paymentHash]; ok {
		m.balance += inv.Amount
	}
}

// GetBalance returns mock balance
func (m *MockProvider) GetBalance() (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.balance, nil
}

// GetInfo returns mock node info
func (m *MockProvider) GetInfo() (*NodeInfo, error) {
	return &NodeInfo{
		Alias:       "MockNode",
		PubKey:      "02mock" + hex.EncodeToString(make([]byte, 30)),
		Network:     "testnet",
		BlockHeight: 800000,
		Synced:      true,
	}, nil
}



