package mcpserver

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_Valid(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "test.yaml")

	yaml := `
server:
  transport: stdio
  name: test-proxy

upstreams:
  mock:
    transport: stdio
    command: ["echo", "hello"]

budget:
  backend: memory
  limit: 500

tools:
  defaultCost: 5
  costs:
    web_search: 5
    dalle_generate: 50

enforcement:
  mode: hard
`
	os.WriteFile(cfgPath, []byte(yaml), 0644)

	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Server.Name != "test-proxy" {
		t.Errorf("expected test-proxy, got %s", cfg.Server.Name)
	}
	if cfg.Budget.Limit != 500 {
		t.Errorf("expected 500, got %d", cfg.Budget.Limit)
	}
	if cfg.Tools.Costs["dalle_generate"] != 50 {
		t.Errorf("expected 50, got %d", cfg.Tools.Costs["dalle_generate"])
	}
	if cfg.DefaultUpstream != "mock" {
		t.Errorf("expected mock as default upstream, got %s", cfg.DefaultUpstream)
	}
	if cfg.Budget.FailMode != "closed" {
		t.Errorf("expected closed failMode, got %s", cfg.Budget.FailMode)
	}
}

func TestLoadConfig_NoUpstreams(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "test.yaml")

	os.WriteFile(cfgPath, []byte("server:\n  transport: stdio\n"), 0644)

	_, err := LoadConfig(cfgPath)
	if err == nil {
		t.Fatal("expected error for no upstreams")
	}
}

func TestLoadConfig_InvalidEnforcement(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "test.yaml")

	yaml := `
upstreams:
  mock:
    transport: stdio
    command: ["echo"]
enforcement:
  mode: invalid
`
	os.WriteFile(cfgPath, []byte(yaml), 0644)

	_, err := LoadConfig(cfgPath)
	if err == nil {
		t.Fatal("expected error for invalid enforcement mode")
	}
}

func TestLoadConfig_EnvExpansion(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "test.yaml")

	os.Setenv("TEST_ROOT_KEY", "my-secret-key")
	defer os.Unsetenv("TEST_ROOT_KEY")

	yaml := `
upstreams:
  mock:
    transport: stdio
    command: ["echo"]
auth:
  mode: header
  rootKey: ${TEST_ROOT_KEY}
`
	os.WriteFile(cfgPath, []byte(yaml), 0644)

	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Auth.RootKey != "my-secret-key" {
		t.Errorf("expected my-secret-key, got %s", cfg.Auth.RootKey)
	}
}

func TestLoadConfig_Routing(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "test.yaml")

	yaml := `
upstreams:
  db:
    transport: stdio
    command: ["db-server"]
  ai:
    transport: stdio
    command: ["ai-server"]

defaultUpstream: db

routing:
  - tools: ["gpt4_*", "dalle_*"]
    upstream: ai
  - tools: ["db_*"]
    upstream: db
`
	os.WriteFile(cfgPath, []byte(yaml), 0644)

	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Routing) != 2 {
		t.Errorf("expected 2 routing rules, got %d", len(cfg.Routing))
	}
	if cfg.Routing[0].Upstream != "ai" {
		t.Errorf("expected ai upstream, got %s", cfg.Routing[0].Upstream)
	}
}
