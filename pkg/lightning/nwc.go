package lightning

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip04"
)

// NWCProvider implements the Lightning Provider interface using Nostr Wallet Connect (NIP-47)
type NWCProvider struct {
	walletPubkey string
	relayURL     string
	clientSecret string
	clientPubkey string
	sharedSecret []byte // Precomputed shared secret for NIP-04 encryption

	mu       sync.Mutex
	conn     *websocket.Conn
	pending  map[string]chan *nwcResponse
}

// NWC request/response structures (NIP-47)
type nwcRequest struct {
	Method string      `json:"method"`
	Params interface{} `json:"params"`
}

type nwcResponse struct {
	ResultType string          `json:"result_type"`
	Error      *nwcError       `json:"error,omitempty"`
	Result     json.RawMessage `json:"result,omitempty"`
}

type nwcError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type makeInvoiceParams struct {
	Amount      int64  `json:"amount"` // millisats
	Description string `json:"description,omitempty"`
}

type makeInvoiceResult struct {
	Type        string `json:"type"`
	Invoice     string `json:"invoice"`
	PaymentHash string `json:"payment_hash"`
}

type lookupInvoiceParams struct {
	PaymentHash string `json:"payment_hash,omitempty"`
	Invoice     string `json:"invoice,omitempty"`
}

type lookupInvoiceResult struct {
	Type        string `json:"type"`
	Invoice     string `json:"invoice"`
	Preimage    string `json:"preimage,omitempty"`
	PaymentHash string `json:"payment_hash"`
	Amount      int64  `json:"amount"` // millisats
	SettledAt   int64  `json:"settled_at,omitempty"`
}

type getBalanceResult struct {
	Balance int64 `json:"balance"` // millisats
}

type getInfoResult struct {
	Alias       string   `json:"alias,omitempty"`
	Color       string   `json:"color,omitempty"`
	Pubkey      string   `json:"pubkey,omitempty"`
	Network     string   `json:"network,omitempty"`
	BlockHeight int64    `json:"block_height,omitempty"`
	Methods     []string `json:"methods,omitempty"`
}

// NewNWCProvider creates a new NWC provider from a connection string
// Format: nostr+walletconnect://walletPubkey?relay=wss://...&secret=...
func NewNWCProvider(config map[string]interface{}) (*NWCProvider, error) {
	connString := getConfigString(config, "connectionString")
	if connString == "" {
		return nil, fmt.Errorf("NWC requires connectionString")
	}

	// Parse the connection string
	walletPubkey, relayURL, secret, err := parseNWCConnectionString(connString)
	if err != nil {
		return nil, fmt.Errorf("invalid NWC connection string: %w", err)
	}

	// Derive client pubkey from secret
	clientPubkey, err := nostr.GetPublicKey(secret)
	if err != nil {
		return nil, fmt.Errorf("failed to derive client pubkey: %w", err)
	}

	// Compute shared secret for NIP-04 encryption
	sharedSecret, err := nip04.ComputeSharedSecret(walletPubkey, secret)
	if err != nil {
		return nil, fmt.Errorf("failed to compute shared secret: %w", err)
	}

	return &NWCProvider{
		walletPubkey: walletPubkey,
		relayURL:     relayURL,
		clientSecret: secret,
		clientPubkey: clientPubkey,
		sharedSecret: sharedSecret,
		pending:      make(map[string]chan *nwcResponse),
	}, nil
}

// parseNWCConnectionString extracts components from a nostr+walletconnect:// URL
func parseNWCConnectionString(connString string) (walletPubkey, relayURL, secret string, err error) {
	// Remove the nostr+walletconnect:// prefix
	connString = strings.TrimPrefix(connString, "nostr+walletconnect://")

	// Split at ? to get pubkey and query params
	parts := strings.SplitN(connString, "?", 2)
	if len(parts) != 2 {
		return "", "", "", fmt.Errorf("missing query parameters")
	}

	walletPubkey = parts[0]
	if len(walletPubkey) != 64 {
		return "", "", "", fmt.Errorf("invalid wallet pubkey length")
	}

	// Parse query params
	params, err := url.ParseQuery(parts[1])
	if err != nil {
		return "", "", "", fmt.Errorf("failed to parse query params: %w", err)
	}

	relayURL = params.Get("relay")
	if relayURL == "" {
		return "", "", "", fmt.Errorf("missing relay parameter")
	}

	secret = params.Get("secret")
	if secret == "" {
		return "", "", "", fmt.Errorf("missing secret parameter")
	}

	return walletPubkey, relayURL, secret, nil
}

// connect establishes a WebSocket connection to the relay
func (n *NWCProvider) connect() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.conn != nil {
		return nil // Already connected
	}

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.Dial(n.relayURL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to relay: %w", err)
	}

	n.conn = conn

	// Start response listener
	go n.listenForResponses()

	return nil
}

// listenForResponses handles incoming messages from the relay
func (n *NWCProvider) listenForResponses() {
	for {
		n.mu.Lock()
		conn := n.conn
		n.mu.Unlock()

		if conn == nil {
			return
		}

		_, message, err := conn.ReadMessage()
		if err != nil {
			n.mu.Lock()
			n.conn = nil
			n.mu.Unlock()
			return
		}

		// Parse Nostr message
		var envelope []json.RawMessage
		if err := json.Unmarshal(message, &envelope); err != nil {
			continue
		}

		if len(envelope) < 2 {
			continue
		}

		var msgType string
		if err := json.Unmarshal(envelope[0], &msgType); err != nil {
			continue
		}

		if msgType == "EVENT" && len(envelope) >= 3 {
			var event nostr.Event
			if err := json.Unmarshal(envelope[2], &event); err != nil {
				continue
			}

			// Check if this is a response to one of our requests (kind 23195)
			if event.Kind == 23195 && event.PubKey == n.walletPubkey {
				n.handleResponse(&event)
			}
		}
	}
}

// handleResponse processes an NWC response event
func (n *NWCProvider) handleResponse(event *nostr.Event) {
	// Get the request ID from the 'e' tag
	var requestID string
	for _, tag := range event.Tags {
		if len(tag) >= 2 && tag[0] == "e" {
			requestID = tag[1]
			break
		}
	}

	if requestID == "" {
		return
	}

	// Decrypt the content using precomputed shared secret
	decrypted, err := nip04.Decrypt(event.Content, n.sharedSecret)
	if err != nil {
		return
	}

	var response nwcResponse
	if err := json.Unmarshal([]byte(decrypted), &response); err != nil {
		return
	}

	// Send to waiting handler
	n.mu.Lock()
	if ch, ok := n.pending[requestID]; ok {
		ch <- &response
		delete(n.pending, requestID)
	}
	n.mu.Unlock()
}

// sendRequest sends an NWC request and waits for response
func (n *NWCProvider) sendRequest(ctx context.Context, method string, params interface{}) (*nwcResponse, error) {
	if err := n.connect(); err != nil {
		return nil, err
	}

	// Build request
	req := nwcRequest{
		Method: method,
		Params: params,
	}
	reqJSON, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	// Encrypt content using precomputed shared secret
	encrypted, err := nip04.Encrypt(string(reqJSON), n.sharedSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt request: %w", err)
	}

	// Create event (kind 23194 = NWC request)
	event := nostr.Event{
		PubKey:    n.clientPubkey,
		CreatedAt: nostr.Timestamp(time.Now().Unix()),
		Kind:      23194,
		Tags:      nostr.Tags{{"p", n.walletPubkey}},
		Content:   encrypted,
	}

	// Sign the event
	if err := event.Sign(n.clientSecret); err != nil {
		return nil, fmt.Errorf("failed to sign event: %w", err)
	}

	// Create response channel
	responseChan := make(chan *nwcResponse, 1)
	n.mu.Lock()
	n.pending[event.ID] = responseChan
	n.mu.Unlock()

	// Send event
	eventMsg, _ := json.Marshal([]interface{}{"EVENT", event})
	n.mu.Lock()
	err = n.conn.WriteMessage(websocket.TextMessage, eventMsg)
	n.mu.Unlock()
	if err != nil {
		return nil, fmt.Errorf("failed to send event: %w", err)
	}

	// Subscribe to responses
	subID := generateSubID()
	subMsg, _ := json.Marshal([]interface{}{
		"REQ",
		subID,
		map[string]interface{}{
			"kinds":   []int{23195},
			"authors": []string{n.walletPubkey},
			"#e":      []string{event.ID},
		},
	})
	n.mu.Lock()
	err = n.conn.WriteMessage(websocket.TextMessage, subMsg)
	n.mu.Unlock()
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe: %w", err)
	}

	// Wait for response with timeout
	select {
	case resp := <-responseChan:
		return resp, nil
	case <-ctx.Done():
		n.mu.Lock()
		delete(n.pending, event.ID)
		n.mu.Unlock()
		return nil, ctx.Err()
	case <-time.After(30 * time.Second):
		n.mu.Lock()
		delete(n.pending, event.ID)
		n.mu.Unlock()
		return nil, fmt.Errorf("NWC request timeout")
	}
}

func generateSubID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// CreateInvoice creates a Lightning invoice via NWC
func (n *NWCProvider) CreateInvoice(amountSats int64, memo string) (*Invoice, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	params := makeInvoiceParams{
		Amount:      amountSats * 1000, // Convert to millisats
		Description: memo,
	}

	resp, err := n.sendRequest(ctx, "make_invoice", params)
	if err != nil {
		return nil, fmt.Errorf("NWC make_invoice failed: %w", err)
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("NWC error: %s - %s", resp.Error.Code, resp.Error.Message)
	}

	var result makeInvoiceResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to parse invoice result: %w", err)
	}

	return &Invoice{
		Bolt11:      result.Invoice,
		PaymentHash: result.PaymentHash,
		Amount:      amountSats,
		Memo:        memo,
		ExpiresAt:   time.Now().Add(10 * time.Minute).Unix(),
	}, nil
}

// CheckPayment checks if an invoice has been paid via NWC
func (n *NWCProvider) CheckPayment(paymentHash string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	params := lookupInvoiceParams{
		PaymentHash: paymentHash,
	}

	resp, err := n.sendRequest(ctx, "lookup_invoice", params)
	if err != nil {
		return false, fmt.Errorf("NWC lookup_invoice failed: %w", err)
	}

	if resp.Error != nil {
		// Not found is not an error - invoice just hasn't been paid
		if resp.Error.Code == "NOT_FOUND" {
			return false, nil
		}
		return false, fmt.Errorf("NWC error: %s - %s", resp.Error.Code, resp.Error.Message)
	}

	var result lookupInvoiceResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return false, fmt.Errorf("failed to parse lookup result: %w", err)
	}

	// Invoice is settled if it has a preimage or settled_at timestamp
	return result.Preimage != "" || result.SettledAt > 0, nil
}

// GetBalance returns the wallet balance via NWC
func (n *NWCProvider) GetBalance() (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := n.sendRequest(ctx, "get_balance", struct{}{})
	if err != nil {
		return 0, fmt.Errorf("NWC get_balance failed: %w", err)
	}

	if resp.Error != nil {
		return 0, fmt.Errorf("NWC error: %s - %s", resp.Error.Code, resp.Error.Message)
	}

	var result getBalanceResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return 0, fmt.Errorf("failed to parse balance result: %w", err)
	}

	// Convert from millisats to sats
	return result.Balance / 1000, nil
}

// GetInfo returns wallet information via NWC
func (n *NWCProvider) GetInfo() (*NodeInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := n.sendRequest(ctx, "get_info", struct{}{})
	if err != nil {
		return nil, fmt.Errorf("NWC get_info failed: %w", err)
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("NWC error: %s - %s", resp.Error.Code, resp.Error.Message)
	}

	var result getInfoResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to parse info result: %w", err)
	}

	network := result.Network
	if network == "" {
		network = "mainnet" // Default assumption for Alby
	}

	return &NodeInfo{
		Alias:       result.Alias,
		PubKey:      result.Pubkey,
		Network:     network,
		BlockHeight: result.BlockHeight,
		Synced:      true, // NWC wallets are typically always synced
	}, nil
}

// Close closes the NWC connection
func (n *NWCProvider) Close() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.conn != nil {
		err := n.conn.Close()
		n.conn = nil
		return err
	}
	return nil
}
