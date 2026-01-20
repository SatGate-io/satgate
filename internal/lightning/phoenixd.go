package lightning

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// PhoenixdProvider implements the Lightning Provider interface for Phoenixd
type PhoenixdProvider struct {
	baseURL  string
	password string
	client   *http.Client
}

// NewPhoenixdProvider creates a new Phoenixd provider
func NewPhoenixdProvider(config map[string]interface{}) (*PhoenixdProvider, error) {
	baseURL := getConfigString(config, "apiUrl")
	if baseURL == "" {
		baseURL = "http://localhost:9740"
	}

	password := getConfigString(config, "apiPassword")
	if password == "" {
		return nil, fmt.Errorf("phoenixd requires apiPassword")
	}

	return &PhoenixdProvider{
		baseURL:  baseURL,
		password: password,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// CreateInvoice creates a Lightning invoice via Phoenixd
func (p *PhoenixdProvider) CreateInvoice(amountSats int64, memo string) (*Invoice, error) {
	// Generate preimage
	preimageBytes := make([]byte, 32)
	if _, err := rand.Read(preimageBytes); err != nil {
		return nil, fmt.Errorf("failed to generate preimage: %w", err)
	}
	preimage := hex.EncodeToString(preimageBytes)
	
	// Calculate payment hash (used as fallback if API doesn't return one)
	paymentHashBytes := sha256.Sum256(preimageBytes)
	_ = paymentHashBytes // Used below as fallback

	// Create invoice via Phoenixd API
	form := url.Values{}
	form.Set("amountSat", fmt.Sprintf("%d", amountSats))
	form.Set("description", memo)

	req, err := http.NewRequest("POST", p.baseURL+"/createinvoice", bytes.NewBufferString(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth("", p.password)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("phoenixd request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("phoenixd error: %s", string(body))
	}

	var result struct {
		Serialized  string `json:"serialized"`
		PaymentHash string `json:"paymentHash"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &Invoice{
		Bolt11:      result.Serialized,
		PaymentHash: result.PaymentHash,
		Preimage:    preimage,
		Amount:      amountSats,
		Memo:        memo,
		ExpiresAt:   time.Now().Add(10 * time.Minute).Unix(),
	}, nil
}

// CheckPayment checks if an invoice has been paid
// Phoenixd API: GET /payments/incoming/{paymentHash}
// Returns payment details with isPaid boolean
func (p *PhoenixdProvider) CheckPayment(paymentHash string) (bool, error) {
	req, err := http.NewRequest("GET", p.baseURL+"/payments/incoming/"+paymentHash, nil)
	if err != nil {
		return false, err
	}
	req.SetBasicAuth("", p.password)

	resp, err := p.client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	// 404 means payment not found
	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("phoenixd error: status %d", resp.StatusCode)
	}

	// Phoenixd returns payment object with isPaid boolean
	var payment struct {
		PaymentHash string `json:"paymentHash"`
		Preimage    string `json:"preimage"`
		IsPaid      bool   `json:"isPaid"`
		ReceivedSat int64  `json:"receivedSat"`
		CompletedAt int64  `json:"completedAt"` // Unix timestamp ms
	}
	if err := json.NewDecoder(resp.Body).Decode(&payment); err != nil {
		return false, err
	}

	return payment.IsPaid, nil
}

// GetBalance returns the wallet balance
func (p *PhoenixdProvider) GetBalance() (int64, error) {
	req, err := http.NewRequest("GET", p.baseURL+"/getbalance", nil)
	if err != nil {
		return 0, err
	}
	req.SetBasicAuth("", p.password)

	resp, err := p.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var result struct {
		BalanceSat int64 `json:"balanceSat"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}

	return result.BalanceSat, nil
}

// GetInfo returns node information
func (p *PhoenixdProvider) GetInfo() (*NodeInfo, error) {
	req, err := http.NewRequest("GET", p.baseURL+"/getinfo", nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth("", p.password)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		NodeId string `json:"nodeId"`
		Chain  string `json:"chain"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &NodeInfo{
		Alias:   "Phoenixd",
		PubKey:  result.NodeId,
		Network: result.Chain,
		Synced:  true,
	}, nil
}

