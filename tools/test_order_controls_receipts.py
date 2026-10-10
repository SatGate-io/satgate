"""Receipts that carry stateful-control fields still verify.

A spending-limit refusal keeps decision=denied / decision_reason=policy_denied
and adds signed top-level fields (denial_code, argument_field, spend_window,
spend_limit). An allowed receipt may add the window total. Every verifier
accepts them: they are inside the signed payload and are not an enumerated
field. This pins that, with a trusted issuer, for both public verifier copies.
Synthetic keys only. Run:
  python -B -m unittest discover -s tools -p 'test_order_controls_receipts.py' -v
"""
import copy
import sys
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / "tools"))

import test_paid_rail as paid  # noqa: E402

NOW = paid.NOW


def refusal(r):
    r.update(denial_code="TOOL_SPEND_LIMIT_DENIED", argument_field="dollar_amount",
             spend_window="day", spend_limit="200")
    return r


def allowed_with_total(r):
    r.update(spend_window="day", spend_limit="200", spend_window_total="150")
    return r


def approval_hold(r):
    r.update(denial_code="APPROVAL_REQUIRED", approval_id="apr_0123456789abcdef")
    return r


class OrderControlReceipts(unittest.TestCase):
    def setUp(self):
        self.f = paid.Fixture()

    def check(self, receipt, decision):
        pack = self.f.pack(receipt)
        for surface, verifier in paid.VERIFIERS.items():
            with self.subTest(surface=surface):
                result = verifier.verify_pack(pack, jwks=self.f.jwks, require_trusted_issuer=True,
                                              now=NOW, allow_mock=True)
                self.assertTrue(result["valid"], result)
                self.assertTrue(result["trusted_issuer_valid"], result)
                self.assertEqual(pack["receipts"][0]["decision"], decision)

    def test_spend_limit_refusal_verifies(self):
        self.check(refusal(self.f.receipt("denied", "policy_denied")), "denied")

    def test_allowed_receipt_with_window_total_verifies(self):
        self.check(allowed_with_total(self.f.receipt("allowed", "budget_authorized")), "allowed")

    def test_approval_hold_verifies(self):
        self.check(approval_hold(self.f.receipt("denied", "policy_denied")), "denied")

    def test_tampering_with_the_new_fields_is_caught(self):
        pack = self.f.pack(refusal(self.f.receipt("denied", "policy_denied")))
        tampered = copy.deepcopy(pack)
        tampered["receipts"][0]["spend_limit"] = "2000"
        for surface, verifier in paid.VERIFIERS.items():
            with self.subTest(surface=surface):
                result = verifier.verify_pack(tampered, jwks=self.f.jwks, require_trusted_issuer=True,
                                              now=NOW, allow_mock=True)
                self.assertFalse(result["valid"], result)


if __name__ == "__main__":
    unittest.main()
