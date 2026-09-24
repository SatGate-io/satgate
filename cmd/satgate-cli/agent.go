package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

func cmdAgent() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: satgate-cli agent policy|simulate|promote")
		os.Exit(1)
	}
	gateway := envOr("SATGATE_GATEWAY", "http://127.0.0.1:8080")
	token := os.Getenv("SATGATE_ADMIN_TOKEN")
	if token == "" {
		fmt.Fprintln(os.Stderr, "SATGATE_ADMIN_TOKEN is required")
		os.Exit(1)
	}
	switch os.Args[2] {
	case "policy":
		agentPost(gateway, token, "/api/agent/policy", mustRead(os.Args[3:]))
	case "simulate":
		agentPost(gateway, token, "/api/agent/simulate", mustRead(os.Args[3:]))
	case "promote":
		agentPost(gateway, token, "/api/agent/promote", mustRead(os.Args[3:]))
	default:
		fmt.Fprintf(os.Stderr, "Unknown agent command: %s\n", os.Args[2])
		os.Exit(1)
	}
}

func agentPost(gateway, token, path string, body []byte) {
	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(gateway, "/")+path, bytes.NewReader(body))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Admin-Token", token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	out, _ := io.ReadAll(resp.Body)
	fmt.Println(string(out))
	if resp.StatusCode >= 300 {
		os.Exit(1)
	}
}

func mustRead(args []string) []byte {
	if len(args) == 0 {
		body, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return body
	}
	if args[0] == "--json" && len(args) > 1 {
		if !json.Valid([]byte(args[1])) {
			fmt.Fprintln(os.Stderr, "invalid json")
			os.Exit(1)
		}
		return []byte(args[1])
	}
	fmt.Fprintln(os.Stderr, "pass --json '{...}' or JSON on stdin")
	os.Exit(1)
	return nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
