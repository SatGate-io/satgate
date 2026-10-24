"""Reader-only unavailable-check refusal contract; ephemeral Ed25519 keys only.

Both independently distributed pack APIs/CLIs, and frozen released readers,
consume the same signed inputs. No network is needed to run these tests.
"""
import copy
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

import test_paid_rail as paid

ROOT = Path(__file__).resolve().parents[1]
REASONS = (
    "tool_price_unavailable", "tool_controls_unavailable",
    "budget_backend_error", "pending_capacity_exhausted",
)
FROZEN = ROOT / "tools" / "fixtures" / "released-checkers-20261022"


def refusal(f, reason):
    r = paid.verified_not_evaluated(f, decision_reason=reason)
    r["metadata"].pop("tenant" + "_id", None)
    return r


def load(path, name):
    spec = importlib.util.spec_from_file_location(name, path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


class UnavailableCheckReceipts(unittest.TestCase):
    def setUp(self):
        self.f = paid.Fixture()

    def verify(self, module, pack):
        return module.verify_pack(copy.deepcopy(pack), jwks=self.f.jwks,
                                  require_trusted_issuer=True, now=paid.NOW)

    def check(self, pack, valid, signature=True):
        for surface, module in paid.VERIFIERS.items():
            with self.subTest(surface=surface):
                result = self.verify(module, pack)
                self.assertEqual(result["valid"], valid, result["reason_codes"])
                self.assertEqual(result["checks"].get("signature_valid"), signature,
                                 result["reason_codes"])
                if valid:
                    self.assertTrue(result["trusted_issuer_valid"])

    def pack(self, r):
        return paid.verified_not_evaluated_pack(self.f, r)

    def test_four_signed_positive_profiles(self):
        self.assertEqual(paid.V.UNAVAILABLE_CHECK_REASONS, frozenset(REASONS))
        for reason in REASONS:
            with self.subTest(reason=reason):
                self.check(self.pack(refusal(self.f, reason)), True)
                for module in paid.VERIFIERS.values():
                    self.assertNotIn(("denied", reason), module.RATE_WINDOW_PAIRS)

    def test_exact_budget_wrong_decision_and_aggregate_contamination(self):
        cases = [
            {"decision": "allowed"}, {"decision": "paid"},
            {"budget": {"spend_mode": "evaluated", "cost_credits": 0}},
            {"budget": {"spend_mode": "not_evaluated", "cost_credits": 1}},
            {"budget": {"spend_mode": "not_evaluated", "cost_credits": False}},
            {"budget": {"spend_mode": "not_evaluated", "cost_credits": 0.0}},
            {"budget": {"spend_mode": "not_evaluated", "cost_credits": 0, "note": "extra"}},
            {"rate_limit_phase": "opened", "rate_limit_window_start": "2026-01-01T00:00:00Z",
             "rate_limit_window_seconds": 60, "denial_code": "RATE_LIMITED"},
            {"rate_limit_refused_calls": 1},
            {"unrecorded_calls": 1, "unrecorded_window_start": "2026-01-01T00:00:00Z",
             "unrecorded_window_seconds": 60},
        ]
        for reason in REASONS:
            for index, extra in enumerate(cases):
                with self.subTest(reason=reason, case=index):
                    r = refusal(self.f, reason)
                    r.update(copy.deepcopy(extra))
                    self.check(self.pack(r), False)

    def test_authority_binding_redaction_and_required_signed_objects(self):
        for reason in REASONS:
            base = refusal(self.f, reason)
            edits = [
                lambda r: r.pop("authority"), lambda r: r.pop("capability_hash"),
                lambda r: r.pop("budget"), lambda r: r.pop("policy"),
                lambda r: r.update(authority={"provenance_level": "no_verified_capability"}),
                lambda r: r["authority"].update(provenance_level="sandbox_no_capability"),
                lambda r: r["authority"].update(capability_hash="sha256:other"),
                lambda r: r["authority"].update(unrecognised="x"),
            ]
            for field in paid.V.VERIFIED_CAPABILITY_RAW_FIELDS:
                edits.append(lambda r, k=field: r["authority"].pop(k))
                edits.append(lambda r, k=field: r["authority"].update({k: "unredacted"}))
            for index, edit in enumerate(edits):
                with self.subTest(reason=reason, case=index):
                    r = copy.deepcopy(base)
                    edit(r)
                    self.check(self.pack(r), False)

    def test_no_claims_at_any_depth_signed_or_unsigned(self):
        claims = (
            "balance", "attempted_amount_usd", "remaining_credits", "limit", "projected_cost_credits",
            "spent_credits", "used_credits", "cost_credits", "spend_mode", "payment_hash",
            "Payment-Hash", "invoice_hash", "rail", "amount_sats", "transaction", "payer",
        )
        for reason in REASONS:
            for claim in claims:
                for field in ("metadata", "policy", "extension"):
                    with self.subTest(reason=reason, claim=claim, field=field):
                        r = refusal(self.f, reason)
                        r.setdefault(field, {})["nested"] = [{"deeper": {claim: None}}]
                        self.check(self.pack(r), False)
                p = self.pack(refusal(self.f, reason))
                p["extension"] = [{"nested": {claim: None}}]
                self.check(p, False)

    def test_required_mirrors_missing_mismatched_and_type_strict(self):
        for reason in REASONS:
            for field in ("authority", "budget", "policy", "budget_state"):
                for replacement in (None, {}, {"spend_mode": "not_evaluated", "cost_credits": False}):
                    p = self.pack(refusal(self.f, reason))
                    if replacement is None:
                        p.pop(field)
                    else:
                        p[field] = replacement
                    with self.subTest(reason=reason, field=field, replacement=replacement):
                        self.check(p, False)

    def test_signature_tampering_and_secondary_receipt_not_skipped(self):
        for reason in REASONS:
            p = self.pack(refusal(self.f, reason))
            p["receipts"][0]["route_or_tool"] = "/changed"
            self.check(p, False, signature=False)
            p = self.pack(refusal(self.f, reason))
            r = refusal(self.f, reason)
            r["decision"] = "allowed"
            p["receipts"].append(self.f.sign(r))
            self.check(p, False)

    def test_unredacted_raw_fields_in_extensions_still_fail_secret_scan(self):
        # An unredacted sentinel, not an actual credential or secret.
        for reason in REASONS:
            r = refusal(self.f, reason)
            r["metadata"]["nested"] = [{"raw_capability_token": "unredacted"}]
            self.check(self.pack(r), False)
            p = self.pack(refusal(self.f, reason))
            p["extension"] = {"nested": [{"raw_budget_id": "unredacted"}]}
            self.check(p, False)

    def test_both_actual_cli_entrypoints(self):
        with tempfile.TemporaryDirectory() as td:
            jwks = Path(td) / "public-jwks.json"
            jwks.write_text(json.dumps(self.f.jwks))
            for reason in REASONS:
                for surface in paid.VERIFIERS:
                    for valid in (True, False):
                        r = refusal(self.f, reason)
                        if not valid:
                            r["decision"] = "allowed"
                        proc = subprocess.run(
                            [sys.executable, "-B", str(ROOT / surface / "verify_evidence_pack.py"),
                             "-", "--now", paid.NOW, "--jwks-file", str(jwks), "--require-trusted-issuer"],
                            input=json.dumps(self.pack(r)), text=True, capture_output=True, timeout=30,
                            env=dict(os.environ, PYTHONDONTWRITEBYTECODE="1"))
                        result = json.loads(proc.stdout)
                        self.assertEqual(proc.returncode, 0 if valid else 1)
                        self.assertEqual(result["valid"], valid)
                        self.assertTrue(result["checks"]["signature_valid"])

    def test_frozen_released_and_dashboard_pinned_checkers_reject(self):
        provenance = json.loads((FROZEN / "PROVENANCE.json").read_text())
        self.assertEqual(provenance["module_version"], "v0.5.3-0.20261022091107-d599b7e3b9af")
        self.assertEqual(provenance["dashboard_commit"], "3395b48f8a2d87001708160a09de6dc551141549")
        self.assertEqual(provenance["module_commit"], "d599b7e3b9af92af1dbc3d90bfac6755c820a44b")
        expected_hashes = {
            "tools/verify_evidence_pack.py": "1ddc777dd77415ee229df3b1c6cbe9c71c168c3e94575c6e167e48d14f70af11",
            "tools/verify_receipt_record.py": "883687f530de0b7216577b037d507410271ea32170c7a5a6b215724e06bbadfa",
            "demo/verify_evidence_pack.py": "2ef3f4c50db9304f2026fab943f693020e616c4a7b60799344c2b47583a33a35",
        }
        self.assertEqual({entry["frozen_path"]: entry["sha256"] for entry in provenance["files"]},
                         expected_hashes)
        for entry in provenance["files"]:
            path = FROZEN / entry["frozen_path"]
            self.assertEqual(hashlib.sha256(path.read_bytes()).hexdigest(), entry["sha256"])
        for surface in ("tools", "demo"):
            old = load(FROZEN / surface / "verify_evidence_pack.py", "frozen_" + surface)
            # The same key, signature, and no-budget control pass the old reader.
            control = refusal(self.f, "policy_denied")
            self.assertTrue(self.verify(old, self.pack(control))["valid"])
            for reason in REASONS:
                with self.subTest(surface=surface, reason=reason):
                    result = self.verify(old, self.pack(refusal(self.f, reason)))
                    self.assertTrue(result["checks"]["signature_valid"])
                    self.assertFalse(result["valid"])
                    self.assertIn("unknown_decision_reason", result["reason_codes"])


if __name__ == "__main__":
    unittest.main()
