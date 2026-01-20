package lightning

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// LNDProvider implements the Lightning Provider interface for LND
type LNDProvider struct {
	baseURL  string
	macaroon string
	client   *http.Client
}

// NewLNDProvider creates a new LND provider
// 
// Security: TLS verification is ENABLED by default. You must provide either:
// - tlsCert: Base64-encoded PEM certificate for custom CA
// - tlsServerName: Expected server name for SNI verification
// - insecureSkipVerify: true (ONLY for development/testing)
func NewLNDProvider(config map[string]interface{}) (*LNDProvider, error) {
	baseURL := getConfigString(config, "restUrl")
	if baseURL == "" {
		return nil, fmt.Errorf("LND requires restUrl")
	}

	macaroon := getConfigString(config, "macaroon")
	if macaroon == "" {
		return nil, fmt.Errorf("LND requires macaroon")
	}

	// Build TLS config - secure by default
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	// Check for explicit insecure mode (development only)
	insecureSkipVerify := getConfigBool(config, "insecureSkipVerify")
	if insecureSkipVerify {
		// Log warning - this should only be used in development
		tlsConfig.InsecureSkipVerify = true
	}

	// If TLS cert is provided, use it for CA verification
	if tlsCert := getConfigString(config, "tlsCert"); tlsCert != "" {
		certPool := x509.NewCertPool()
		
		// Support both base64-encoded and raw PEM
		var certBytes []byte
		decoded, err := base64.StdEncoding.DecodeString(tlsCert)
		if err != nil {
			// Assume it's raw PEM
			certBytes = []byte(tlsCert)
		} else {
			certBytes = decoded
		}
		
		if ok := certPool.AppendCertsFromPEM(certBytes); !ok {
			return nil, fmt.Errorf("failed to add TLS cert to pool - invalid PEM format")
		}
		tlsConfig.RootCAs = certPool
		tlsConfig.InsecureSkipVerify = false // Ensure verification is on when cert provided
	}

	// Support custom server name for SNI verification
	if serverName := getConfigString(config, "tlsServerName"); serverName != "" {
		tlsConfig.ServerName = serverName
	}

	// Validate: if not insecure and no custom cert, we use system CA pool (secure default)
	// This will work for publicly-signed certs but fail for self-signed without explicit config

	return &LNDProvider{
		baseURL:  baseURL,
		macaroon: macaroon,
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: tlsConfig,
			},
		},
	}, nil
}

// getConfigBool safely extracts a boolean from config
func getConfigBool(config map[string]interface{}, key string) bool {
	if v, ok := config[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

// CreateInvoice creates a Lightning invoice via LND
func (l *LNDProvider) CreateInvoice(amountSats int64, memo string) (*Invoice, error) {
	reqBody := map[string]interface{}{
		"value": amountSats,
		"memo":  memo,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", l.baseURL+"/v1/invoices", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Grpc-Metadata-macaroon", l.macaroon)
	req.Header.Set("Content-Type", "application/json")

	resp, err := l.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("LND request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("LND error: %s", string(body))
	}

	var result struct {
		RHash          string `json:"r_hash"`
		PaymentRequest string `json:"payment_request"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Decode r_hash from base64
	rHashBytes, _ := base64.StdEncoding.DecodeString(result.RHash)
	paymentHash := hex.EncodeToString(rHashBytes)

	return &Invoice{
		Bolt11:      result.PaymentRequest,
		PaymentHash: paymentHash,
		Amount:      amountSats,
		Memo:        memo,
		ExpiresAt:   time.Now().Add(10 * time.Minute).Unix(),
	}, nil
}

// CheckPayment checks if an invoice has been paid via LND
func (l *LNDProvider) CheckPayment(paymentHash string) (bool, error) {
	// Convert hex to base64 for LND API
	hashBytes, err := hex.DecodeString(paymentHash)
	if err != nil {
		return false, err
	}
	hashBase64 := base64.URLEncoding.EncodeToString(hashBytes)

	req, err := http.NewRequest("GET", l.baseURL+"/v1/invoice/"+hashBase64, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Grpc-Metadata-macaroon", l.macaroon)

	resp, err := l.client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, nil
	}

	var result struct {
		Settled bool `json:"settled"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, err
	}

	return result.Settled, nil
}

// GetBalance returns the wallet balance via LND
func (l *LNDProvider) GetBalance() (int64, error) {
	req, err := http.NewRequest("GET", l.baseURL+"/v1/balance/channels", nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Grpc-Metadata-macaroon", l.macaroon)

	resp, err := l.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var result struct {
		Balance int64 `json:"balance,string"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}

	return result.Balance, nil
}

// GetInfo returns node information via LND
func (l *LNDProvider) GetInfo() (*NodeInfo, error) {
	req, err := http.NewRequest("GET", l.baseURL+"/v1/getinfo", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Grpc-Metadata-macaroon", l.macaroon)

	resp, err := l.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Alias           string `json:"alias"`
		IdentityPubkey  string `json:"identity_pubkey"`
		Chains          []struct {
			Chain   string `json:"chain"`
			Network string `json:"network"`
		} `json:"chains"`
		BlockHeight     int64 `json:"block_height"`
		SyncedToChain   bool  `json:"synced_to_chain"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	network := "unknown"
	if len(result.Chains) > 0 {
		network = result.Chains[0].Network
	}

	return &NodeInfo{
		Alias:       result.Alias,
		PubKey:      result.IdentityPubkey,
		Network:     network,
		BlockHeight: result.BlockHeight,
		Synced:      result.SyncedToChain,
	}, nil
}

