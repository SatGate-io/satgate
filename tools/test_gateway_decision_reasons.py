"""Gateway decision reasons and a missing-cryptography exit.

Synthetic keys only. Run:
  python -B -m unittest discover -s tools -p 'test_gateway_decision_reasons.py' -v
"""
import importlib.util
import json
import os
import subprocess
import sys
import textwrap
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / "tools"))

import test_paid_rail as paid  # noqa: E402


def load_tools():
    spec = importlib.util.spec_from_file_location(
        "verify_tools_reasons", ROOT / "tools" / "verify_evidence_pack.py")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


TOOLS = load_tools()
NOW = paid.NOW


class GatewayDecisionReasons(unittest.TestCase):
    def setUp(self):
        self.f = paid.Fixture()

    def verify(self, pack):
        return TOOLS.verify_pack(pack, jwks=self.f.jwks, now=NOW, allow_mock=True)

    def test_auth_missing_and_token_revoked_verify(self):
        for reason in ("auth_missing", "token_revoked"):
            with self.subTest(reason=reason):
                result = self.verify(self.f.pack(self.f.receipt("denied", reason)))
                self.assertTrue(result["valid"], result)
                self.assertNotIn("unknown_decision_reason", result.get("reason_codes", []))

    def test_observe_sandbox_and_partial_budget_reasons_verify(self):
        cases = (
            ("allowed", "observe_projected"),
            ("allowed", "sandbox_no_spend"),
            ("denied", "insufficient_budget"),
        )
        for decision, reason in cases:
            with self.subTest(reason=reason):
                result = self.verify(self.f.pack(self.f.receipt(decision, reason)))
                self.assertTrue(result["valid"], result)
                self.assertNotIn("unknown_decision_reason", result.get("reason_codes", []))

    def test_invented_reason_still_fails(self):
        result = self.verify(self.f.pack(self.f.receipt("denied", "not_a_gateway_reason")))
        self.assertFalse(result["valid"], result)
        self.assertIn("unknown_decision_reason", result.get("reason_codes", []))

    def test_missing_cryptography_is_not_a_bad_receipt(self):
        pack = self.f.pack(self.f.receipt("denied", "auth_missing"))
        blocker = textwrap.dedent(
            """
            import sys
            script = sys.argv[1]
            sys.argv = [script, "-"]
            class Block:
                def find_spec(self, fullname, path, target=None):
                    if fullname == "cryptography" or fullname.startswith("cryptography."):
                        raise ModuleNotFoundError(fullname)
                    return None
            sys.meta_path.insert(0, Block())
            import runpy
            raise SystemExit(runpy.run_path(script, run_name="__main__"))
            """
        )
        for surface in ("tools", "demo"):
            script = ROOT / surface / "verify_evidence_pack.py"
            with self.subTest(surface=surface):
                proc = subprocess.run(
                    [sys.executable, "-c", blocker, str(script), "-"],
                    input=json.dumps(pack).encode(),
                    capture_output=True,
                    cwd=ROOT,
                    env={**os.environ, "PYTHONDONTWRITEBYTECODE": "1"},
                )
                output = (proc.stdout + proc.stderr).decode()
                self.assertEqual(proc.returncode, 3, output)
                self.assertIn("Signatures could not be checked", output)
                self.assertIn("cryptography", output)
                self.assertIn("pip install", output)
                self.assertNotIn("unknown_decision_reason", output)
                self.assertNotIn('"valid"', output)
