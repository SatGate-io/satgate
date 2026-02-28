// Package main provides the SatGate CLI for managing API gateway configuration,
// minting tokens, and checking status from the command line.
//
// Usage:
//
//	satgate-cli init                    # Interactive setup wizard
//	satgate-cli status                  # Show gateway and agent status
//	satgate-cli mint --subject agent-1  # Mint a capability token
//	satgate-cli token validate <token>  # Validate a macaroon token
//	satgate-cli version                 # Show version info
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

const (
	defaultCloudURL = "https://cloud.satgate.io"
	configFileName  = "satgate.yaml"
)

// CLIConfig holds persisted CLI configuration
type CLIConfig struct {
	CloudURL string `json:"cloud_url"`
	APIKey   string `json:"api_key"`
	TenantID string `json:"tenant_id"`
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "init":
		cmdInit()
	case "status":
		cmdStatus()
	case "mint":
		cmdMint()
	case "token":
		if len(os.Args) > 2 && os.Args[2] == "validate" {
			cmdTokenValidate()
		} else {
			fmt.Println("Usage: satgate-cli token validate <macaroon>")
			os.Exit(1)
		}
	case "wrap":
		cmdWrap()
	case "version":
		fmt.Printf("satgate-cli %s (commit: %s, built: %s)\n", Version, Commit, BuildDate)
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`SatGate CLI — Economic Firewall for APIs

Usage:
  satgate-cli <command> [options]

Commands:
  init                         Interactive setup — configure gateway connection
  status                       Show gateway health, active agents, and spend
  mint --subject <name>        Mint a capability token for an agent
  wrap --token <tok> -- <cmd>  Run any command through SatGate proxy
  token validate <token>       Validate a macaroon token
  version                      Show version info
  help                         Show this help

Get started:
  satgate-cli init
  satgate-cli status

Run agents through SatGate:
  satgate-cli wrap --token <macaroon> -- python my_agent.py
  satgate-cli wrap --token <macaroon> --gateway https://gw.example.com -- node agent.js

Documentation: https://cloud.satgate.io/docs
GitHub: https://github.com/SatGate-io/satgate`)
}

// ── Init ──────────────────────────────────────────────────────────

func cmdInit() {
	fmt.Println("🚀 SatGate CLI Setup")
	fmt.Println("═══════════════════════════════════════")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	// Cloud URL
	fmt.Printf("Cloud URL [%s]: ", defaultCloudURL)
	cloudURL := readLine(reader)
	if cloudURL == "" {
		cloudURL = defaultCloudURL
	}

	// API Key
	fmt.Print("API Key (from cloud.satgate.io/cloud/api-keys): ")
	apiKey := readLine(reader)
	if apiKey == "" {
		fmt.Println("\n⚠️  No API key provided. Get one at: https://cloud.satgate.io/cloud/api-keys")
		fmt.Println("   You can also sign up free at: https://cloud.satgate.io/cloud/login")
		os.Exit(1)
	}

	// Validate the key
	fmt.Print("\nValidating API key... ")
	cfg := CLIConfig{CloudURL: cloudURL, APIKey: apiKey}
	resp, err := apiRequest(&cfg, "GET", "/api/cloud/tenant/info", nil)
	if err != nil {
		fmt.Printf("❌ %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		fmt.Printf("❌ Invalid API key (HTTP %d)\n", resp.StatusCode)
		os.Exit(1)
	}

	var tenantInfo struct {
		TenantID string `json:"tenant_id"`
		Slug     string `json:"slug"`
		Plan     string `json:"plan"`
	}
	json.NewDecoder(resp.Body).Decode(&tenantInfo)
	cfg.TenantID = tenantInfo.TenantID

	fmt.Println("✅")
	fmt.Printf("   Tenant: %s\n", tenantInfo.Slug)
	fmt.Printf("   Plan:   %s\n", tenantInfo.Plan)

	// Save config
	configPath := getConfigPath()
	if err := saveConfig(&cfg, configPath); err != nil {
		fmt.Printf("\n❌ Failed to save config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n✅ Config saved to %s\n", configPath)
	fmt.Println("\nNext steps:")
	fmt.Println("  satgate-cli status          # Check your gateway")
	fmt.Println("  satgate-cli mint --subject my-agent  # Create a token")
}

// ── Status ────────────────────────────────────────────────────────

func cmdStatus() {
	cfg := mustLoadConfig()

	fmt.Println("📊 SatGate Status")
	fmt.Println("═══════════════════════════════════════")

	// Tenant info
	resp, err := apiRequest(cfg, "GET", "/api/cloud/tenant/info", nil)
	if err != nil {
		fmt.Printf("❌ Cannot reach cloud: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var info map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&info)

	fmt.Printf("\n  Cloud:    %s\n", cfg.CloudURL)
	if slug, ok := info["slug"].(string); ok {
		fmt.Printf("  Tenant:   %s\n", slug)
	}
	if plan, ok := info["plan"].(string); ok {
		fmt.Printf("  Plan:     %s\n", plan)
	}

	// Usage stats
	usageResp, err := apiRequest(cfg, "GET", "/api/cloud/usage/summary", nil)
	if err == nil {
		defer usageResp.Body.Close()
		if usageResp.StatusCode == 200 {
			var usage map[string]interface{}
			json.NewDecoder(usageResp.Body).Decode(&usage)
			fmt.Println()
			if total, ok := usage["total_requests"].(float64); ok {
				fmt.Printf("  Requests: %.0f\n", total)
			}
			if blocked, ok := usage["blocked_requests"].(float64); ok {
				fmt.Printf("  Blocked:  %.0f\n", blocked)
			}
			if spend, ok := usage["total_spend"].(float64); ok {
				fmt.Printf("  Spend:    $%.2f\n", spend)
			}
		}
	}

	// Active agents
	agentsResp, err := apiRequest(cfg, "GET", "/api/cloud/delegation/tree", nil)
	if err == nil {
		defer agentsResp.Body.Close()
		if agentsResp.StatusCode == 200 {
			var tree map[string]interface{}
			json.NewDecoder(agentsResp.Body).Decode(&tree)
			if tokens, ok := tree["tree"].([]interface{}); ok {
				active := 0
				for _, t := range tokens {
					if tok, ok := t.(map[string]interface{}); ok {
						if status, ok := tok["status"].(string); ok && status == "active" {
							active++
						}
					}
				}
				fmt.Printf("  Agents:   %d active\n", active)
			}
		}
	}

	fmt.Println()
	fmt.Println("  Dashboard: " + cfg.CloudURL + "/cloud/dashboard")
}

// ── Mint ──────────────────────────────────────────────────────────

func cmdMint() {
	cfg := mustLoadConfig()

	subject := ""
	audience := "satgate"
	for i := 2; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "--subject", "-s":
			if i+1 < len(os.Args) {
				subject = os.Args[i+1]
				i++
			}
		case "--audience", "-a":
			if i+1 < len(os.Args) {
				audience = os.Args[i+1]
				i++
			}
		}
	}

	if subject == "" {
		fmt.Println("Usage: satgate-cli mint --subject <agent-name> [--audience <aud>]")
		os.Exit(1)
	}

	fmt.Printf("🔑 Minting token for \"%s\" (audience: %s)...\n", subject, audience)

	// Step 1: Get JWT from mock IdP (or configured IdP)
	idpPayload := map[string]string{"sub": subject, "aud": audience}
	idpBody, _ := json.Marshal(idpPayload)

	idpResp, err := http.Post("https://satgate-mock-idp.fly.dev/token", "application/json", bytes.NewReader(idpBody))
	if err != nil {
		fmt.Printf("❌ IdP error: %v\n", err)
		os.Exit(1)
	}
	defer idpResp.Body.Close()

	var idpResult map[string]string
	json.NewDecoder(idpResp.Body).Decode(&idpResult)
	jwt := idpResult["token"]
	if jwt == "" {
		fmt.Println("❌ No JWT returned from IdP")
		os.Exit(1)
	}
	fmt.Println("  ✓ JWT obtained from Identity Provider")

	// Step 2: Exchange for macaroon
	exchangePayload := map[string]string{"credentials": jwt}
	exchangeBody, _ := json.Marshal(exchangePayload)

	exchangeResp, err := apiRequest(cfg, "POST", "/api/mint/exchange", exchangeBody)
	if err != nil {
		fmt.Printf("❌ Mint exchange error: %v\n", err)
		os.Exit(1)
	}
	defer exchangeResp.Body.Close()

	body, _ := io.ReadAll(exchangeResp.Body)
	if exchangeResp.StatusCode != 200 {
		fmt.Printf("❌ Mint failed (HTTP %d): %s\n", exchangeResp.StatusCode, string(body))
		os.Exit(1)
	}

	var mintResult map[string]interface{}
	json.Unmarshal(body, &mintResult)

	token := ""
	if t, ok := mintResult["token"].(string); ok {
		token = t
	} else if t, ok := mintResult["macaroon"].(string); ok {
		token = t
	}

	fmt.Println("  ✓ Macaroon minted by SatGate")
	fmt.Println()

	if policy, ok := mintResult["policy"].(string); ok {
		fmt.Printf("  Policy:  %s\n", policy)
	}
	if budget, ok := mintResult["budget"].(map[string]interface{}); ok {
		if limit, ok := budget["limit"].(float64); ok {
			fmt.Printf("  Budget:  %.0f credits\n", limit)
		}
	}

	fmt.Println()
	fmt.Println("  Token:")
	fmt.Println("  " + token)
	fmt.Println()
	fmt.Println("  Use with curl:")
	fmt.Printf("  curl -H \"Authorization: Bearer %s\" https://your-api.example.com/endpoint\n", truncate(token, 20)+"...")
}

// ── Token Validate ───────────────────────────────────────────────

func cmdTokenValidate() {
	cfg := mustLoadConfig()

	if len(os.Args) < 4 {
		fmt.Println("Usage: satgate-cli token validate <macaroon>")
		os.Exit(1)
	}
	token := os.Args[3]

	payload := map[string]string{"token": token}
	body, _ := json.Marshal(payload)

	resp, err := apiRequest(cfg, "POST", "/api/cloud/tokens/validate", body)
	if err != nil {
		fmt.Printf("❌ %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		fmt.Printf("❌ Invalid token (HTTP %d): %s\n", resp.StatusCode, string(respBody))
		os.Exit(1)
	}

	var result map[string]interface{}
	json.Unmarshal(respBody, &result)

	fmt.Println("✅ Valid token")
	if valid, ok := result["valid"].(bool); ok && valid {
		prettyJSON, _ := json.MarshalIndent(result, "  ", "  ")
		fmt.Println("  " + string(prettyJSON))
	}
}

// ── Wrap ─────────────────────────────────────────────────────────

func cmdWrap() {
	token := ""
	gateway := ""
	var childArgs []string
	dashDash := false

	for i := 2; i < len(os.Args); i++ {
		if os.Args[i] == "--" {
			dashDash = true
			childArgs = os.Args[i+1:]
			break
		}
		switch os.Args[i] {
		case "--token", "-t":
			if i+1 < len(os.Args) {
				token = os.Args[i+1]
				i++
			}
		case "--gateway", "-g":
			if i+1 < len(os.Args) {
				gateway = os.Args[i+1]
				i++
			}
		}
	}

	if !dashDash || len(childArgs) == 0 {
		fmt.Println("Usage: satgate-cli wrap --token <macaroon> [--gateway <url>] -- <command> [args...]")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  satgate-cli wrap --token mac_abc123 -- python my_agent.py")
		fmt.Println("  satgate-cli wrap --token mac_abc123 --gateway https://gw.satgate.io -- curl https://api.openai.com/v1/chat/completions")
		os.Exit(1)
	}

	// Try to get token from env if not provided
	if token == "" {
		token = os.Getenv("SATGATE_TOKEN")
	}
	if token == "" {
		fmt.Println("❌ Token required. Use --token <macaroon> or set SATGATE_TOKEN env var")
		os.Exit(1)
	}

	// Try to get gateway from config if not provided
	if gateway == "" {
		if cfg, err := loadConfig(); err == nil && cfg.CloudURL != "" {
			gateway = cfg.CloudURL
		}
	}

	// Start local proxy
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Printf("❌ Failed to start proxy: %v\n", err)
		os.Exit(1)
	}
	proxyAddr := listener.Addr().String()

	// Build proxy handler
	proxyHandler := &wrapProxy{
		token:   token,
		gateway: gateway,
	}

	server := &http.Server{Handler: proxyHandler}
	go server.Serve(listener)

	proxyURL := "http://" + proxyAddr

	fmt.Printf("🛡️  SatGate proxy running on %s\n", proxyURL)
	if gateway != "" {
		fmt.Printf("   Gateway: %s\n", gateway)
	}
	fmt.Printf("   Token:   %s...\n", truncate(token, 20))
	fmt.Printf("   Running: %s\n", strings.Join(childArgs, " "))
	fmt.Println()

	// Start child process with proxy env vars
	cmd := exec.Command(childArgs[0], childArgs[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Env = append(os.Environ(),
		"HTTP_PROXY="+proxyURL,
		"HTTPS_PROXY="+proxyURL,
		"http_proxy="+proxyURL,
		"https_proxy="+proxyURL,
		"SATGATE_TOKEN="+token,
	)

	// Forward signals to child
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		if cmd.Process != nil {
			cmd.Process.Signal(sig)
		}
	}()

	err = cmd.Run()
	server.Close()

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Printf("❌ %v\n", err)
		os.Exit(1)
	}
}

// wrapProxy is an HTTP proxy that injects SatGate tokens into requests.
// If a gateway URL is set, it rewrites requests to route through the gateway.
type wrapProxy struct {
	token   string
	gateway string
}

func (p *wrapProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Handle CONNECT (HTTPS tunneling)
	if r.Method == http.MethodConnect {
		p.handleConnect(w, r)
		return
	}

	// For regular HTTP requests, proxy with token injection
	target := r.URL
	if p.gateway != "" {
		// Route through gateway — preserve original Host as header
		gwURL, err := url.Parse(p.gateway)
		if err != nil {
			http.Error(w, "Bad gateway URL", http.StatusBadGateway)
			return
		}
		target = &url.URL{
			Scheme:   gwURL.Scheme,
			Host:     gwURL.Host,
			Path:     r.URL.Path,
			RawQuery: r.URL.RawQuery,
		}
		r.Header.Set("X-Forwarded-Host", r.Host)
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Director = func(req *http.Request) {
		req.URL = target
		req.Host = target.Host
		req.Header.Set("Authorization", "Bearer "+p.token)
	}
	proxy.ServeHTTP(w, r)
}

func (p *wrapProxy) handleConnect(w http.ResponseWriter, r *http.Request) {
	// For HTTPS CONNECT, we do a man-in-the-middle to inject the token
	// This works for development/testing — agents should trust the local proxy

	destConn, err := net.DialTimeout("tcp", r.Host, 10*time.Second)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		destConn.Close()
		return
	}

	w.WriteHeader(http.StatusOK)
	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		destConn.Close()
		return
	}

	// Simple TCP tunnel (token injection happens at HTTP level for non-TLS,
	// for TLS we pass through — the agent should use HTTP_PROXY for token injection)
	go func() {
		defer destConn.Close()
		defer clientConn.Close()
		io.Copy(destConn, clientConn)
	}()
	go func() {
		defer destConn.Close()
		defer clientConn.Close()
		io.Copy(clientConn, destConn)
	}()
}

// ── Helpers ──────────────────────────────────────────────────────

func apiRequest(cfg *CLIConfig, method, path string, body []byte) (*http.Response, error) {
	url := cfg.CloudURL + path
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	}
	if cfg.TenantID != "" {
		req.Header.Set("X-SatGate-Tenant", cfg.TenantID)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	return client.Do(req)
}

func getConfigPath() string {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".satgate")
	os.MkdirAll(dir, 0700)
	return filepath.Join(dir, "config.json")
}

func saveConfig(cfg *CLIConfig, path string) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func loadConfig() (*CLIConfig, error) {
	path := getConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg CLIConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func mustLoadConfig() *CLIConfig {
	cfg, err := loadConfig()
	if err != nil {
		fmt.Println("❌ Not configured. Run: satgate-cli init")
		os.Exit(1)
	}
	return cfg
}

func readLine(reader *bufio.Reader) string {
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
