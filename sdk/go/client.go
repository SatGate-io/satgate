// Package satgate provides automatic L402 payment handling for Go applications.
//
// The "Stripe Moment" for Go: Give your services a Lightning wallet in a few lines.
//
// Quick Start:
//
//	wallet := satgate.NewLNBitsWallet("https://legend.lnbits.com", "your-admin-key")
//	client := satgate.NewClient(wallet)
//
//	// 402 → Pay → Retry happens automatically
//	resp, err := client.Get("https://api.example.com/premium")
package satgate

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sync"
	"time"
)

// LightningWallet is the interface that must be implemented by any Lightning wallet.
type LightningWallet interface {
	// PayInvoice pays a BOLT11 invoice and returns the preimage (hex string).
	PayInvoice(invoice string) (preimage string, err error)
}

// PaymentInfo contains information about a completed payment.
type PaymentInfo struct {
	Invoice   string    `json:"invoice"`
	Preimage  string    `json:"preimage"`
	Macaroon  string    `json:"macaroon"`
	Endpoint  string    `json:"endpoint"`
	Timestamp time.Time `json:"timestamp"`
}

// TokenCache stores L402 tokens for reuse.
type TokenCache struct {
	mu     sync.RWMutex
	tokens map[string]*cachedToken
}

type cachedToken struct {
	macaroon  string
	preimage  string
	expiresAt time.Time
}

// Client is the SatGate HTTP client that automatically handles L402 payments.
type Client struct {
	wallet     LightningWallet
	httpClient *http.Client
	cache      *TokenCache
	cacheTTL   time.Duration
	verbose    bool

	// Callbacks
	OnPayment func(info PaymentInfo)

	// Stats
	mu           sync.Mutex
	TotalPaidSat int64
}

// ClientOption configures a Client.
type ClientOption func(*Client)

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(c *http.Client) ClientOption {
	return func(client *Client) {
		client.httpClient = c
	}
}

// WithCacheTTL sets the token cache TTL.
func WithCacheTTL(ttl time.Duration) ClientOption {
	return func(client *Client) {
		client.cacheTTL = ttl
	}
}

// WithVerbose enables verbose logging.
func WithVerbose(v bool) ClientOption {
	return func(client *Client) {
		client.verbose = v
	}
}

// WithPaymentCallback sets a callback for payment events.
func WithPaymentCallback(fn func(PaymentInfo)) ClientOption {
	return func(client *Client) {
		client.OnPayment = fn
	}
}

// NewClient creates a new SatGate client.
func NewClient(wallet LightningWallet, opts ...ClientOption) *Client {
	c := &Client{
		wallet:     wallet,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		cache: &TokenCache{
			tokens: make(map[string]*cachedToken),
		},
		cacheTTL: 5 * time.Minute,
		verbose:  true,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// Get performs a GET request, automatically handling L402 payment challenges.
func (c *Client) Get(url string) (*http.Response, error) {
	return c.Do("GET", url, nil)
}

// Post performs a POST request with JSON body.
func (c *Client) Post(url string, body interface{}) (*http.Response, error) {
	return c.Do("POST", url, body)
}

// Do performs an HTTP request, handling L402 challenges automatically.
func (c *Client) Do(method, url string, body interface{}) (*http.Response, error) {
	// Check cache first
	if token := c.getCachedToken(url); token != nil {
		if c.verbose {
			fmt.Printf("⚡ Using cached L402 token for %s\n", url)
		}
		return c.doWithAuth(method, url, body, token.macaroon, token.preimage)
	}

	// Make initial request
	resp, err := c.doRequest(method, url, body, nil)
	if err != nil {
		return nil, err
	}

	// Handle 402 Payment Required
	if resp.StatusCode == http.StatusPaymentRequired {
		return c.handlePaymentChallenge(resp, method, url, body)
	}

	return resp, nil
}

func (c *Client) handlePaymentChallenge(resp *http.Response, method, url string, body interface{}) (*http.Response, error) {
	authHeader := resp.Header.Get("WWW-Authenticate")
	if authHeader == "" {
		return resp, nil
	}

	// Parse L402/LSAT header
	macaroon, invoice := parseL402Header(authHeader)
	if macaroon == "" || invoice == "" {
		return resp, fmt.Errorf("invalid L402 header format")
	}

	if c.verbose {
		fmt.Printf("⚡ 402 Detected. Invoice: %s...%s\n", invoice[:20], invoice[len(invoice)-10:])
	}

	// Pay the invoice
	preimage, err := c.wallet.PayInvoice(invoice)
	if err != nil {
		return nil, fmt.Errorf("payment failed: %w", err)
	}

	if c.verbose {
		fmt.Printf("✅ Payment Confirmed. Preimage: %s...\n", preimage[:10])
	}

	// Cache the token
	c.cacheToken(url, macaroon, preimage)

	// Track payment
	c.mu.Lock()
	c.TotalPaidSat++ // Simplified; ideally decode invoice for amount
	c.mu.Unlock()

	if c.OnPayment != nil {
		c.OnPayment(PaymentInfo{
			Invoice:   invoice,
			Preimage:  preimage,
			Macaroon:  macaroon,
			Endpoint:  url,
			Timestamp: time.Now(),
		})
	}

	// Retry with L402 token
	if c.verbose {
		fmt.Println("🔄 Retrying request with L402 Token...")
	}
	return c.doWithAuth(method, url, body, macaroon, preimage)
}

func (c *Client) doWithAuth(method, url string, body interface{}, macaroon, preimage string) (*http.Response, error) {
	authValue := fmt.Sprintf("LSAT %s:%s", macaroon, preimage)
	return c.doRequest(method, url, body, map[string]string{"Authorization": authValue})
}

func (c *Client) doRequest(method, url string, body interface{}, headers map[string]string) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return c.httpClient.Do(req)
}

func (c *Client) getCachedToken(url string) *cachedToken {
	c.cache.mu.RLock()
	defer c.cache.mu.RUnlock()

	token, ok := c.cache.tokens[url]
	if !ok || time.Now().After(token.expiresAt) {
		return nil
	}
	return token
}

func (c *Client) cacheToken(url, macaroon, preimage string) {
	c.cache.mu.Lock()
	defer c.cache.mu.Unlock()

	c.cache.tokens[url] = &cachedToken{
		macaroon:  macaroon,
		preimage:  preimage,
		expiresAt: time.Now().Add(c.cacheTTL),
	}
}

// parseL402Header extracts macaroon and invoice from WWW-Authenticate header.
func parseL402Header(header string) (macaroon, invoice string) {
	macaroonRe := regexp.MustCompile(`macaroon="([^"]+)"`)
	invoiceRe := regexp.MustCompile(`invoice="([^"]+)"`)

	if m := macaroonRe.FindStringSubmatch(header); len(m) > 1 {
		macaroon = m[1]
	}
	if m := invoiceRe.FindStringSubmatch(header); len(m) > 1 {
		invoice = m[1]
	}
	return
}

// ============================================================================
// LNBits Wallet Implementation
// ============================================================================

// LNBitsWallet implements LightningWallet using LNBits API.
type LNBitsWallet struct {
	BaseURL  string
	AdminKey string
	client   *http.Client
}

// NewLNBitsWallet creates a new LNBits wallet.
func NewLNBitsWallet(baseURL, adminKey string) *LNBitsWallet {
	return &LNBitsWallet{
		BaseURL:  baseURL,
		AdminKey: adminKey,
		client:   &http.Client{Timeout: 30 * time.Second},
	}
}

// PayInvoice pays a BOLT11 invoice via LNBits.
func (w *LNBitsWallet) PayInvoice(invoice string) (string, error) {
	payload := map[string]interface{}{
		"out":    true,
		"bolt11": invoice,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", w.BaseURL+"/api/v1/payments", bytes.NewReader(jsonPayload))
	if err != nil {
		return "", err
	}

	req.Header.Set("X-Api-Key", w.AdminKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := w.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("LNBits API error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("LNBits payment failed: %s", string(body))
	}

	var result struct {
		PaymentHash string `json:"payment_hash"`
		Preimage    string `json:"preimage"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if result.Preimage == "" {
		return "", fmt.Errorf("LNBits did not return preimage")
	}

	return result.Preimage, nil
}

// ============================================================================
// Alby Wallet Implementation
// ============================================================================

// AlbyWallet implements LightningWallet using Alby API.
type AlbyWallet struct {
	AccessToken string
	client      *http.Client
}

// NewAlbyWallet creates a new Alby wallet.
func NewAlbyWallet(accessToken string) *AlbyWallet {
	return &AlbyWallet{
		AccessToken: accessToken,
		client:      &http.Client{Timeout: 30 * time.Second},
	}
}

// PayInvoice pays a BOLT11 invoice via Alby API.
func (w *AlbyWallet) PayInvoice(invoice string) (string, error) {
	payload := map[string]string{"invoice": invoice}
	jsonPayload, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", "https://api.getalby.com/payments", bytes.NewReader(jsonPayload))
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+w.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := w.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Alby API error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Alby payment failed: %s", string(body))
	}

	var result struct {
		Preimage string `json:"preimage"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if result.Preimage == "" {
		return "", fmt.Errorf("Alby did not return preimage")
	}

	return result.Preimage, nil
}

// ============================================================================
// LND Wallet Implementation (for direct node access)
// ============================================================================

// LNDWallet implements LightningWallet using LND's REST API.
type LNDWallet struct {
	Host       string // e.g., "localhost:8080"
	Macaroon   string // hex-encoded admin macaroon
	TLSCert    []byte // TLS certificate (optional for local)
	client     *http.Client
}

// NewLNDWallet creates a new LND wallet.
func NewLNDWallet(host, macaroonHex string) *LNDWallet {
	return &LNDWallet{
		Host:     host,
		Macaroon: macaroonHex,
		client:   &http.Client{Timeout: 60 * time.Second},
	}
}

// PayInvoice pays a BOLT11 invoice via LND REST API.
func (w *LNDWallet) PayInvoice(invoice string) (string, error) {
	payload := map[string]string{"payment_request": invoice}
	jsonPayload, _ := json.Marshal(payload)

	url := fmt.Sprintf("https://%s/v1/channels/transactions", w.Host)
	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonPayload))
	if err != nil {
		return "", err
	}

	// Decode macaroon from hex
	macaroonBytes, err := hex.DecodeString(w.Macaroon)
	if err != nil {
		return "", fmt.Errorf("invalid macaroon hex: %w", err)
	}

	req.Header.Set("Grpc-Metadata-macaroon", hex.EncodeToString(macaroonBytes))
	req.Header.Set("Content-Type", "application/json")

	resp, err := w.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("LND API error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("LND payment failed: %s", string(body))
	}

	var result struct {
		PaymentPreimage string `json:"payment_preimage"`
		PaymentError    string `json:"payment_error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if result.PaymentError != "" {
		return "", fmt.Errorf("LND payment error: %s", result.PaymentError)
	}

	// LND returns base64, we need hex
	preimageBytes, err := hex.DecodeString(result.PaymentPreimage)
	if err != nil {
		// It might already be hex
		return result.PaymentPreimage, nil
	}

	return hex.EncodeToString(preimageBytes), nil
}

