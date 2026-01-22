package lightning

import (
	"fmt"
	"strings"
)

// Provider defines the interface for Lightning backends
type Provider interface {
	// CreateInvoice generates a new Lightning invoice
	CreateInvoice(amountSats int64, memo string) (*Invoice, error)
	
	// CheckPayment verifies if a payment has been received
	CheckPayment(paymentHash string) (bool, error)
	
	// GetBalance returns the node's balance in sats
	GetBalance() (int64, error)
	
	// GetInfo returns node information
	GetInfo() (*NodeInfo, error)
}

// Invoice represents a Lightning invoice
type Invoice struct {
	Bolt11      string // BOLT11 encoded invoice
	PaymentHash string // Hex-encoded payment hash
	Preimage    string // Hex-encoded preimage (for verification)
	Amount      int64  // Amount in sats
	Memo        string
	ExpiresAt   int64  // Unix timestamp
}

// NodeInfo contains Lightning node information
type NodeInfo struct {
	Alias      string
	PubKey     string
	Network    string // mainnet, testnet, signet
	BlockHeight int64
	Synced     bool
}

// NewProvider creates a Lightning provider by name
func NewProvider(name string, config map[string]interface{}) (Provider, error) {
	// Trim whitespace and handle empty provider name
	name = strings.TrimSpace(name)
	if name == "" {
		name = "mock" // Default to mock if no provider specified
	}
	
	switch name {
	case "phoenixd":
		return NewPhoenixdProvider(config)
	case "lnd":
		return NewLNDProvider(config)
	case "nwc", "alby":
		return NewNWCProvider(config)
	case "mock":
		return NewMockProvider(), nil
	default:
		return nil, fmt.Errorf("unknown lightning provider: %s", name)
	}
}

// getConfigString safely extracts a string from config
func getConfigString(config map[string]interface{}, key string) string {
	if v, ok := config[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}



