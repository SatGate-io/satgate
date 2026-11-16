package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// agentVerify exits the process on rejection, so exercise it in a child test.
func TestCloudMCPIssuerVerifyChild(t *testing.T) {
	if os.Getenv("SATGATE_CLOUD_ISSUER_TEST_CHILD") != "1" {
		return
	}
	agentVerify([]string{os.Getenv("SATGATE_CLOUD_ISSUER_TEST_PACK")})
	os.Exit(0)
}

func TestCloudMCPIssuer(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	// Generate dummy signatures in memory using the shared Python test fixture.
	// Existing sample keys are offline fixtures, not independently authenticated
	// production JWKS. No network discovery is used by this test.
	generate := exec.Command("python3", "-c", `
import json, sys
from pathlib import Path
sys.path.insert(0, str(Path(sys.argv[1]) / 'tools'))
from test_cloud_mcp_issuer import CLOUD, LEGACY, GATEWAY, REJECTED, signed_pack
cases = []
for issuer in (CLOUD, LEGACY, GATEWAY) + REJECTED:
    pack, jwks = signed_pack(issuer)
    cases.append(dict(name=issuer, pack=pack, jwks=jwks, valid=issuer not in REJECTED))
for path in sorted((Path(sys.argv[1]) / 'satgate-landing/public/evidence').glob('*.json')):
    pack = json.loads(path.read_text())
    receipts = pack.get('receipts', [])
    if not receipts or receipts[0].get('issuer') != LEGACY:
        continue
    r = receipts[0]
    jwks = {'keys': [{'kid': r['issuer_kid'], 'kty': 'OKP', 'crv': 'Ed25519',
                      'x': r['metadata']['public_key_ed25519_b64']}]}
    cases.append(dict(name=path.name, pack=pack, jwks=jwks, valid=True))
print(json.dumps(cases))
`, root)
	body, err := generate.CombinedOutput()
	if err != nil {
		t.Fatalf("Python verifier test dependencies required: %v\n%s", err, body)
	}
	var cases []struct {
		Name  string          `json:"name"`
		Pack  json.RawMessage `json:"pack"`
		JWKS  json.RawMessage `json:"jwks"`
		Valid bool            `json:"valid"`
	}
	if err := json.Unmarshal(body, &cases); err != nil {
		t.Fatal(err)
	}
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, surface := range []string{"tools", "demo"} {
		for _, tc := range cases {
			t.Run(surface+"/"+tc.Name, func(t *testing.T) {
				dir := t.TempDir()
				packPath := filepath.Join(dir, "pack.json")
				jwksPath := filepath.Join(dir, "jwks.json")
				if err := os.WriteFile(packPath, tc.Pack, 0600); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(jwksPath, tc.JWKS, 0600); err != nil {
					t.Fatal(err)
				}
				cmd := exec.Command(binary, "-test.run=^TestCloudMCPIssuerVerifyChild$")
				cmd.Env = append(os.Environ(),
					"SATGATE_CLOUD_ISSUER_TEST_CHILD=1",
					"SATGATE_CLOUD_ISSUER_TEST_PACK="+packPath,
					"SATGATE_VERIFIER="+filepath.Join(root, surface, "verify_evidence_pack.py"),
					"SATGATE_JWKS="+jwksPath)
				output, err := cmd.CombinedOutput()
				if !tc.Valid {
					if err == nil {
						t.Fatal("unlisted issuer accepted with caller-supplied local JWKS")
					}
					return
				}
				if err != nil {
					t.Fatalf("expected valid receipt: %v\n%s", err, output)
				}
				var verdict struct {
					Valid   bool `json:"valid"`
					Trusted bool `json:"trusted_issuer_valid"`
				}
				if err := json.Unmarshal(output, &verdict); err != nil {
					t.Fatal(err)
				}
				if !verdict.Valid || !verdict.Trusted {
					t.Fatal("receipt did not verify with supplied JWKS")
				}
			})
		}
	}
}
