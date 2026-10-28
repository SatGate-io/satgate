package config

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// repoRoot is the repository root, two levels above this package.
func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("repository root not found at %s", root)
	}
	return root
}

// topLevelRoutes matches a YAML document's top-level "routes:" key, which
// only a gateway configuration has.
var topLevelRoutes = regexp.MustCompile(`(?m)^routes:`)

// gatewayConfigFiles returns every YAML file in the repository that is a
// gateway configuration (has a top-level routes key), relative to root.
func gatewayConfigFiles(t *testing.T, root string) []string {
	t.Helper()
	var files []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
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
		if topLevelRoutes.Match(data) {
			rel, _ := filepath.Rel(root, path)
			files = append(files, rel)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

// Every example and test gateway configuration in the repository loads, so
// the refusals CheckUpstreamHeaders adds affect none of them.
func TestRepoGatewayConfigsStillLoad(t *testing.T) {
	root := repoRoot(t)
	files := gatewayConfigFiles(t, root)
	for _, want := range []string{"examples/gateway.yaml", "demo/configs/demo.yaml", "staging_config.yaml"} {
		found := false
		for _, f := range files {
			found = found || filepath.ToSlash(f) == want
		}
		if !found {
			t.Errorf("%s was not found as a gateway configuration", want)
		}
	}
	for _, f := range files {
		cfg, err := Load(filepath.Join(root, f))
		if err != nil {
			t.Errorf("%s does not load: %v", f, err)
			continue
		}
		if err := CheckUpstreamHeaders(cfg.Upstreams); err != nil {
			t.Errorf("%s: %v", f, err)
		}
	}
}

// Each YAML example in the configuration reference that configures an
// upstream's headers passes CheckUpstreamHeaders.
func TestDocsUpstreamHeaderExamplesAccepted(t *testing.T) {
	root := repoRoot(t)
	doc, err := os.ReadFile(filepath.Join(root, "docs", "reference", "configuration.md"))
	if err != nil {
		t.Fatal(err)
	}
	blocks := regexp.MustCompile("(?s)```yaml\n(.*?)```").FindAllStringSubmatch(string(doc), -1)
	checked := 0
	for _, b := range blocks {
		body := b[1]
		if !strings.Contains(body, "headers:") || !strings.HasPrefix(body, "upstreams:") {
			continue
		}
		var cfg Config
		if err := yaml.Unmarshal([]byte(body), &cfg); err != nil {
			t.Errorf("configuration.md example does not parse: %v", err)
			continue
		}
		if err := CheckUpstreamHeaders(cfg.Upstreams); err != nil {
			t.Errorf("configuration.md example refused: %v", err)
		}
		checked++
	}
	if checked == 0 {
		t.Error("no upstream headers example found in configuration.md")
	}
}
