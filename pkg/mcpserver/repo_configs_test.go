package mcpserver

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// Every example and test MCP proxy configuration in the repository (a YAML
// file with top-level server and upstreams keys and no routes key) still
// loads.
func TestRepoMCPConfigsStillLoad(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("repository root not found at %s", root)
	}
	server := regexp.MustCompile(`(?m)^server:`)
	upstreams := regexp.MustCompile(`(?m)^upstreams:`)
	routes := regexp.MustCompile(`(?m)^routes:`)
	var files []string
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "node_modules", "vendor":
				return filepath.SkipDir
			}
			return nil
		}
		if ext := filepath.Ext(path); ext != ".yaml" && ext != ".yml" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if server.Match(data) && upstreams.Match(data) && !routes.Match(data) {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) < 4 {
		t.Fatalf("found %d MCP configurations, want the four under examples/mcp-proxy at least", len(files))
	}
	for _, f := range files {
		rel, _ := filepath.Rel(root, f)
		if _, err := LoadConfig(f); err != nil {
			t.Errorf("%s does not load: %v", rel, err)
		}
	}
}
