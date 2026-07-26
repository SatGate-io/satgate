#!/usr/bin/env python3
from __future__ import annotations

import json
import re
import unittest
from pathlib import Path

DEMO = Path(__file__).resolve().parents[1]
REPO = DEMO.parent

GATEWAY = "ghcr.io/satgate-io/satgate-gateway-demo@sha256:c20c826cf234b895299ab77f86c6e8f0df772858ba0d8e607d75339ac4e579c7"
DASHBOARD = "ghcr.io/satgate-io/satgate-dashboard-demo@sha256:9e84ea39ffce96f3930a28bf140037c67a307bac5389e4bbabea2b43e5f212f9"
EXPECTED_SERVICES = {"archive-init", "gateway", "dashboard", "postgres", "redis", "mock-internal"}
FORBIDDEN_TERMS = (
    "satgate-gateway-enterprise",
    "satgate-dashboard-enterprise",
    "satgate-enterprise:latest",
    "satgate-enterprise.git",
    "acme-corp",
    "hallmark",
    "fly.dev",
    "railway",
    "stripe",
    "phoenixd",
)


def text(path: Path) -> str:
    return path.read_text(encoding="utf-8")


class PublicDemoStaticTest(unittest.TestCase):
    def test_required_files_exist(self) -> None:
        required = {
            "README.md",
            "SOURCE-MANIFEST.json",
            "docker-compose.yml",
            "golden-path.sh",
            "verify_evidence_pack.py",
            "requirements-verifier.txt",
            "configs/demo.yaml",
            "init-db/00-cloud-tenants.sql",
            "init-db/01-billing-usage.sql",
            "mock-apis/internal/search.json",
            "_vendor/rfc8785/__init__.py",
            "_vendor/rfc8785/_impl.py",
            "_vendor/rfc8785/py.typed",
        }
        missing = sorted(p for p in required if not (DEMO / p).is_file())
        self.assertEqual(missing, [])

    def test_compose_is_pull_only_digest_pinned_and_isolated(self) -> None:
        compose = text(DEMO / "docker-compose.yml")
        self.assertIn(GATEWAY, compose)
        self.assertIn(DASHBOARD, compose)
        self.assertNotRegex(compose, r"(?m)^\s*build:")
        self.assertNotRegex(compose, r"(?m)^\s*container_name:")
        self.assertNotIn("../", compose)
        self.assertRegex(compose, r"(?m)^name:\s+satgate-public-demo\s*$")
        self.assertNotRegex(compose, r"(?m)^\s{4,}name:\s+.+$")
        images = re.findall(r"(?m)^\s*image:\s*([^\s#]+)", compose)
        self.assertGreaterEqual(len(images), 6)
        self.assertTrue(all("@sha256:" in image for image in images), images)
        ports = re.findall(r"(?m)^\s*-\s*[\"']?([^\"'\n]+:[0-9]+)[\"']?\s*(?:#.*)?$", compose)
        self.assertTrue(ports, "expected explicit loopback port bindings")
        self.assertTrue(all(port.startswith("127.0.0.1:") for port in ports), ports)
        service_block = compose.split("\nservices:\n", 1)[1].rsplit("\nvolumes:\n", 1)[0]
        services = set(re.findall(r"(?m)^  ([a-z0-9-]+):\s*$", service_block))
        self.assertEqual(services, EXPECTED_SERVICES)

    def test_no_private_or_customer_surfaces(self) -> None:
        operational_paths = [
            DEMO / "README.md",
            DEMO / "docker-compose.yml",
            DEMO / "golden-path.sh",
            DEMO / "configs/demo.yaml",
        ]
        corpus = "\n".join(text(path) for path in operational_paths).lower()
        for term in FORBIDDEN_TERMS:
            self.assertNotIn(term.lower(), corpus, term)
        forbidden_paths = [
            "agent-control-plane",
            "init-db/acme",
            "init-db/hallmark",
            "monitoring",
            "mock-apis/openai",
            "mock-apis/anthropic",
            "mock-mcp",
        ]
        present = sorted(p for p in forbidden_paths if (DEMO / p).exists())
        self.assertEqual(present, [])

    def test_runtime_uses_real_token_but_output_is_redacted(self) -> None:
        script = text(DEMO / "golden-path.sh")
        self.assertNotIn("Authorization: Bearer ***", script)
        self.assertGreaterEqual(script.count('Authorization: Bearer ${TOKEN}'), 3)
        self.assertIn('echo "Token: [redacted]"', script)
        self.assertIn('grep -Fq "$TOKEN" "$proof"', script)
        self.assertIn('"${ADMIN_BASE}/cloud/auth/verify"', script)
        self.assertIn('"${ADMIN_BASE}/cloud/auth/me"', script)

    def test_dev_token_fixture_is_loopback_config_only(self) -> None:
        config = text(DEMO / "configs/demo.yaml")
        compose = text(DEMO / "docker-compose.yml")
        self.assertIn("allowDevTokens: true", config)
        self.assertIn('baseURL: "http://localhost:8080"', config)
        self.assertNotIn("allowDevTokens", compose)

    def test_evidence_archive_is_required_and_project_scoped(self) -> None:
        compose = text(DEMO / "docker-compose.yml")
        self.assertIn("SATGATE_EVIDENCE_ARCHIVE_DIR: /data/evidence-archive", compose)
        self.assertIn('SATGATE_EVIDENCE_ARCHIVE_REQUIRED: "true"', compose)
        self.assertIn("SATGATE_RECEIPT_ISSUER: https://localhost", compose)
        self.assertIn("SATGATE_RECEIPT_SIGNING_KID: local-demo-v1", compose)
        self.assertIn('SATGATE_RECEIPT_SIGNING_KEY_REQUIRED: "true"', compose)
        self.assertIn("SATGATE_RECEIPT_SIGNING_KEY_B64: Qq9mCjlOdQfAfGs4Pje2Nk-UpjeuUx78H27wjk8m4_U", compose)
        self.assertNotIn("SATGATE_RECEIPT_ISSUER: https://api.satgate.io", compose)
        self.assertIn("demo_evidence_archive:/data/evidence-archive", compose)
        self.assertNotIn("name: demo_evidence_archive", compose)
        self.assertIn('user: "0:0"', compose)
        self.assertIn("chown 1000:1000 /data/evidence-archive", compose)
        gateway_block = compose.split("\n  gateway:\n", 1)[1].split("\n  dashboard:\n", 1)[0]
        self.assertNotIn('user: "0:0"', gateway_block)

    def test_readme_is_public_repo_and_claim_bounded(self) -> None:
        readme = text(DEMO / "README.md")
        self.assertIn("https://github.com/SatGate-io/satgate.git", readme)
        self.assertIn("docker compose up -d", readme)
        self.assertIn("./golden-path.sh", readme)
        self.assertIn("embedded-key", readme)
        self.assertIn("does not prove", readme.lower())
        self.assertNotIn("production ready", readme.lower())

    def test_source_manifest_covers_every_non_test_file(self) -> None:
        manifest = json.loads(text(DEMO / "SOURCE-MANIFEST.json"))
        self.assertEqual(manifest["source_commit"], "200cdec3a8d820ec6da19ad5848fa9392db3f5b9")
        entries = manifest["files"]
        paths = {entry["public_path"] for entry in entries}
        actual = {
            str(path.relative_to(REPO))
            for path in DEMO.rglob("*")
            if path.is_file()
            and path.name != "SOURCE-MANIFEST.json"
            and "tests" not in path.parts
            and "__pycache__" not in path.parts
        }
        self.assertEqual(paths, actual)
        for entry in entries:
            self.assertIn(entry["mode"], {"exact", "adapted", "new"})
            self.assertRegex(entry["sha256"], r"^[0-9a-f]{64}$")


if __name__ == "__main__":
    unittest.main()
