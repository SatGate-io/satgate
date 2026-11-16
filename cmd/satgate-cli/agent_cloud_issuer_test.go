package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
)

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
	generate := exec.Command("python3", "-c", `
import json, sys
from pathlib import Path
sys.path.insert(0, str(Path(sys.argv[1]) / 'tools'))
from test_cloud_mcp_issuer import integration_cases, LEGACY
cases = list(integration_cases())
for path in sorted((Path(sys.argv[1]) / 'satgate-landing/public/evidence').glob('*.json')):
    pack = json.loads(path.read_text())
    receipts = pack.get('receipts', [])
    if not receipts or receipts[0].get('issuer') != LEGACY:
        continue
    r = receipts[0]
    jwks = {'keys': [{'kid': r['issuer_kid'], 'kty': 'OKP', 'crv': 'Ed25519',
                      'x': r['metadata']['public_key_ed25519_b64']}]}
    cases.append(dict(name=path.name, pack=pack, jwks=jwks, mode='file', valid=True, calls=[]))
print(json.dumps(cases))
`, root)
	body, err := generate.CombinedOutput()
	if err != nil {
		t.Fatalf("Python fixtures: %v\n%s", err, body)
	}
	var cases []struct {
		Name  string          `json:"name"`
		Pack  json.RawMessage `json:"pack"`
		JWKS  json.RawMessage `json:"jwks"`
		Mode  string          `json:"mode"`
		Valid bool            `json:"valid"`
		Calls []string        `json:"calls"`
		Code  string          `json:"code"`
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
			if surface == "demo" && tc.Mode == "embedded" {
				continue // demo CLI defaults to discovery, not embedded-key mode
			}
			t.Run(surface+"/"+tc.Name, func(t *testing.T) {
				dir := t.TempDir()
				write := func(name string, body []byte) string {
					path := filepath.Join(dir, name)
					if err := os.WriteFile(path, body, 0600); err != nil {
						t.Fatal(err)
					}
					return path
				}
				packPath := write("pack.json", tc.Pack)
				jwksPath := write("jwks.json", tc.JWKS)
				script := filepath.Join(root, surface, "verify_evidence_pack.py")
				callsPath := filepath.Join(dir, "calls.json")
				driverCase := tc
				if surface == "demo" && tc.Mode == "discovery" {
					driverCase.Mode = "default-discovery"
				}
				caseBody, _ := json.Marshal(driverCase)
				casePath := write("case.json", caseBody)
				if tc.Mode == "discovery" {
					// Local key stand-in; inject discovery only for the tools CLI.
					toolsJSON, _ := json.Marshal(filepath.Join(root, "tools"))
					script = write("driver.py", []byte("import sys\nsys.path.insert(0, "+string(toolsJSON)+")\nfrom test_cloud_mcp_issuer import test_driver\ntest_driver()\n"))
				}
				cmd := exec.Command(binary, "-test.run=^TestCloudMCPIssuerVerifyChild$")
				jwksEnv := jwksPath
				if tc.Mode == "embedded" || tc.Mode == "discovery" {
					jwksEnv = ""
				}
				cmd.Env = append(os.Environ(),
					"SATGATE_CLOUD_ISSUER_TEST_CHILD=1",
					"SATGATE_CLOUD_ISSUER_TEST_PACK="+packPath,
					"SATGATE_VERIFIER="+script,
					"SATGATE_JWKS="+jwksEnv,
					"SATGATE_TEST_CASE="+casePath,
					"SATGATE_TEST_CALLS="+callsPath,
					"SATGATE_TEST_SURFACE="+surface)
				output, err := cmd.CombinedOutput()
				var verdict struct {
					Valid   bool     `json:"valid"`
					Trusted bool     `json:"trusted_issuer_valid"`
					Codes   []string `json:"reason_codes"`
					Reasons []string `json:"reasons"`
				}
				if e := json.Unmarshal(output, &verdict); e != nil {
					t.Fatalf("verdict: %v", e)
				}
				if (err == nil) != tc.Valid || verdict.Valid != tc.Valid {
					t.Errorf("exit/verdict: %v valid=%v", err, verdict.Valid)
				}
				if tc.Valid && verdict.Trusted != (tc.Mode != "embedded") {
					t.Error("trust verdict")
				}
				if tc.Code != "" {
					found := false
					for _, code := range verdict.Codes {
						if code == tc.Code {
							found = true
						}
					}
					if !found {
						t.Errorf("reason code: %v", verdict.Codes)
					}
					if tc.Code == "issuer_not_satgate" {
						found = false
						for _, reason := range verdict.Reasons {
							if reason == "receipt.issuer is not a SatGate address; to trust another issuer, pass its keys with --jwks-url or --jwks-file" {
								found = true
							}
						}
						if !found {
							t.Error("refusal text")
						}
					}
				}
				if tc.Mode == "discovery" {
					body, e := os.ReadFile(callsPath)
					if e != nil {
						t.Fatal(e)
					}
					var calls []string
					if e := json.Unmarshal(body, &calls); e != nil {
						t.Fatal(e)
					}
					if !reflect.DeepEqual(calls, tc.Calls) {
						t.Errorf("calls=%v expected=%v", calls, tc.Calls)
					}
				}
			})
		}
	}
}
