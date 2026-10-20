"""Receipts that make every action provable still verify, and nothing else does.

Three receipt shapes were added so that every call an agent makes can be signed:

  * allowed / policy_allowed with spend_mode not_evaluated: a call that was
    allowed and read no balance. It claims none.
  * window receipts: the opener and the closer of a run of refused calls for
    one key and tool in one UTC minute (the closer carries the count).
  * the Observe gap receipt, allowed / observe_unrecorded: a signed count of the
    calls that went through in one UTC minute with no receipt.

Both public verifier copies (tools and demo) are held to the same cases with
the same synthetic keys. Run:
  python -B -m unittest discover -s tools -p 'test_every_action_receipts.py' -v
"""
import copy
import sys
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / "tools"))

import test_paid_rail as paid  # noqa: E402

NOW = paid.NOW
WINDOW = "2026-01-01T00:00:00Z"
OPENED_AT = "2026-01-01T00:00:07Z"
CLOSED_AT = "2026-01-01T00:01:20Z"


def allowed_free(f, **extra):
    """A verified key's call that was allowed and read no balance."""
    r = paid.verified_not_evaluated(f)
    r.update(decision="allowed", decision_reason="policy_allowed")
    r.update(extra)
    return r


def opener(f, **extra):
    r = paid.verified_not_evaluated(f)
    r.update(denial_code="RATE_LIMITED", rate_limit_phase="opened", rate_limit_window_start=WINDOW,
             rate_limit_window_seconds=60, issued_at=OPENED_AT)
    r.update(extra)
    return r


def closer(f, count=57, **extra):
    r = opener(f, rate_limit_phase="closed", rate_limit_refused_calls=count, issued_at=CLOSED_AT)
    r.update(extra)
    return r


def gap(f, count=12, **extra):
    r = allowed_free(f, decision_reason="observe_unrecorded", unrecorded_calls=count,
                     unrecorded_window_start=WINDOW, unrecorded_window_seconds=60, issued_at=CLOSED_AT)
    r.update(extra)
    return r


def account_level(f, reason, **extra):
    """A refusal of a key that verified and is revoked or expired: the account
    is known, the key is not named."""
    r = f.receipt("denied", reason)
    r.update(denial_code="TOKEN_REVOKED" if reason == "token_revoked" else "CREDENTIAL_EXPIRED",
             rate_limit_phase="opened", rate_limit_window_start=WINDOW, rate_limit_window_seconds=60,
             issued_at=OPENED_AT)
    r.update(extra)
    return r


class EveryActionReceipts(unittest.TestCase):
    def setUp(self):
        self.f = paid.Fixture()

    def pack(self, receipt):
        return paid.verified_not_evaluated_pack(self.f, receipt)

    def verify(self, surface, pack):
        return paid.VERIFIERS[surface].verify_pack(
            copy.deepcopy(pack), jwks=self.f.jwks, require_trusted_issuer=True, now=NOW, allow_mock=True)

    def accepted(self, receipt, label=""):
        pack = self.pack(receipt)
        for surface in paid.VERIFIERS:
            with self.subTest(case=label, surface=surface):
                result = self.verify(surface, pack)
                self.assertTrue(result["valid"], result)
                self.assertTrue(result["trusted_issuer_valid"], result)

    def rejected(self, receipt, label="", code=None):
        pack = self.pack(receipt)
        for surface in paid.VERIFIERS:
            with self.subTest(case=label, surface=surface):
                result = self.verify(surface, pack)
                self.assertTrue(result["checks"]["signature_valid"], "the case must fail on its profile, not its signature")
                self.assertFalse(result["valid"], result)
                if code:
                    self.assertIn(code, result["reason_codes"])

    # ---- allowed / policy_allowed, no balance claim ----

    def test_allowed_policy_allowed_not_evaluated_is_valid(self):
        self.accepted(allowed_free(self.f))

    def test_allowed_policy_allowed_with_balance_fields_is_invalid(self):
        for label, extra in {
            "remaining_budget_usd": {"remaining_budget_usd": "0.000000"},
            "attempted_amount_usd": {"attempted_amount_usd": "0.000000"},
            "amount_usd": {"amount_usd": "0.000000"},
            "rail": {"rail": "credits"},
            "remaining_credits": {"remaining_credits": 0},
            "limit_credits": {"limit_credits": 5},
            "nested_balance": {"metadata": {"public_key_ed25519_b64": self.f.x, "tenant_id": "t", "request_id": "r", "balance": 3}},
        }.items():
            self.rejected(allowed_free(self.f, **extra), label)

    def test_allowed_policy_allowed_not_evaluated_budget_must_be_exact(self):
        self.rejected(allowed_free(self.f, budget={"spend_mode": "not_evaluated", "cost_credits": 1}), "nonzero cost")
        self.rejected(allowed_free(self.f, budget={"spend_mode": "not_evaluated", "cost_credits": 0, "limit_credits": 5}), "limit in budget")

    def test_balance_claim_in_the_pack_wrapper_is_invalid(self):
        pack = self.pack(allowed_free(self.f))
        pack["budget_state"] = {"spend_mode": "not_evaluated", "cost_credits": 0, "remaining_budget_usd": "0.000000"}
        for surface in paid.VERIFIERS:
            with self.subTest(surface=surface):
                self.assertFalse(self.verify(surface, pack)["valid"])

    def test_old_allowed_policy_allowed_with_balance_is_still_valid(self):
        # The evaluated profile, as the HTTP gateway signs it today.
        pack = self.f.pack(self.f.receipt("allowed", "policy_allowed"))
        for surface in paid.VERIFIERS:
            with self.subTest(surface=surface):
                self.assertTrue(self.verify(surface, pack)["valid"])

    def test_old_receipts_still_valid(self):
        for decision, reason, valid in paid.LEGACY_PROFILES:
            pack = self.f.pack(self.f.receipt(decision, reason))
            for surface in paid.VERIFIERS:
                with self.subTest(pair=f"{decision}/{reason}", surface=surface):
                    self.assertEqual(self.verify(surface, pack)["valid"], valid)

    def test_tampering_is_caught(self):
        pack = self.pack(allowed_free(self.f))
        tampered = copy.deepcopy(pack)
        tampered["receipts"][0]["decision_reason"] = "budget_authorized"
        for surface in paid.VERIFIERS:
            with self.subTest(surface=surface):
                self.assertFalse(self.verify(surface, tampered)["valid"])

    def test_token_invalid_is_still_not_a_legal_reason(self):
        # Why the enterprise gateway must sign capability_invalid, not this.
        pack = self.f.pack(self.f.receipt("denied", "capability_invalid"))
        pack["receipts"][0]["decision_reason"] = "token_invalid"
        for surface in paid.VERIFIERS:
            with self.subTest(surface=surface):
                self.assertFalse(self.verify(surface, pack)["valid"])

    # ---- window receipts: opener and closer ----

    def test_opener_and_closer_are_valid(self):
        self.accepted(opener(self.f), "opener")
        self.accepted(closer(self.f), "closer")
        self.accepted(closer(self.f, count=1), "closer of a window of one")

    def test_account_level_window_receipts_are_valid(self):
        for reason in ("token_revoked", "capability_expired"):
            r = account_level(self.f, reason)
            for surface in paid.VERIFIERS:
                with self.subTest(reason=reason, surface=surface):
                    result = self.verify(surface, self.f.pack(r))
                    self.assertTrue(result["valid"], result)

    def test_closer_without_a_count_is_invalid(self):
        r = closer(self.f)
        del r["rate_limit_refused_calls"]
        self.rejected(r, code="invalid_window_receipt")

    def test_closer_count_must_be_a_whole_number_of_at_least_one(self):
        for value in (0, -1, 1.5, 2.0, True, "7", None):
            self.rejected(closer(self.f, count=value), repr(value), code="invalid_window_receipt")

    def test_opener_with_a_count_is_invalid(self):
        self.rejected(opener(self.f, rate_limit_refused_calls=3), code="invalid_window_receipt")

    def test_window_fields_are_checked(self):
        cases = {
            "unknown phase": {"rate_limit_phase": "half"},
            "no phase": {"rate_limit_phase": None},
            "window not a minute": {"rate_limit_window_start": "2026-01-01T00:00:30Z"},
            "window not UTC": {"rate_limit_window_start": "2026-01-01T01:00:00+01:00"},
            "window not a time": {"rate_limit_window_start": "yesterday"},
            "window length": {"rate_limit_window_seconds": 30},
            "window length as float": {"rate_limit_window_seconds": 60.0},
            "window length as text": {"rate_limit_window_seconds": "60"},
            "no denial code": {"denial_code": ""},
            "opener issued before its window": {"issued_at": "2025-12-31T23:59:59Z"},
        }
        for label, extra in cases.items():
            r = opener(self.f, **extra)
            if extra.get("rate_limit_phase") is None and "rate_limit_phase" in extra:
                r["rate_limit_phase"] = None
            self.rejected(r, label, code="invalid_window_receipt")

    def test_closer_issued_before_its_window_ended_is_invalid(self):
        self.rejected(closer(self.f, issued_at="2026-01-01T00:00:59Z"), code="invalid_window_receipt")
        self.accepted(closer(self.f, issued_at="2026-01-01T00:01:00Z"), "at the end of the window")

    def test_window_fields_on_another_decision_are_invalid(self):
        for decision, reason in (("allowed", "budget_authorized"), ("allowed", "policy_allowed"), ("denied", "budget_exhausted")):
            r = self.f.receipt(decision, reason)
            r.update(rate_limit_phase="opened", rate_limit_window_start=WINDOW, rate_limit_window_seconds=60, denial_code="RATE_LIMITED")
            pack = self.f.pack(r)
            for surface in paid.VERIFIERS:
                with self.subTest(pair=f"{decision}/{reason}", surface=surface):
                    self.assertFalse(self.verify(surface, pack)["valid"])

    def test_window_receipt_with_a_balance_claim_is_invalid(self):
        self.rejected(opener(self.f, remaining_budget_usd="0.000000"), "remaining")
        self.rejected(closer(self.f, remaining_credits=0), "remaining credits")
        self.rejected(opener(self.f, budget={"spend_mode": "not_evaluated", "cost_credits": 0, "limit_credits": 1}), "limit")

    def test_window_tampering_is_caught(self):
        pack = self.pack(closer(self.f, count=57))
        for surface in paid.VERIFIERS:
            tampered = copy.deepcopy(pack)
            tampered["receipts"][0]["rate_limit_refused_calls"] = 2
            with self.subTest(surface=surface):
                self.assertFalse(self.verify(surface, tampered)["valid"])

    # ---- the Observe gap receipt ----

    def test_gap_receipt_is_valid(self):
        self.accepted(gap(self.f))
        self.accepted(gap(self.f, count=1), "one call")

    def test_gap_receipt_without_a_count_is_invalid(self):
        r = gap(self.f)
        del r["unrecorded_calls"]
        self.rejected(r, code="invalid_unrecorded_receipt")

    def test_gap_receipt_without_a_window_is_invalid(self):
        for field in ("unrecorded_window_start", "unrecorded_window_seconds"):
            r = gap(self.f)
            del r[field]
            self.rejected(r, field, code="invalid_unrecorded_receipt")

    def test_gap_receipt_fields_are_checked(self):
        for label, extra in {
            "count zero": {"unrecorded_calls": 0},
            "count negative": {"unrecorded_calls": -3},
            "count float": {"unrecorded_calls": 2.5},
            "count bool": {"unrecorded_calls": True},
            "count text": {"unrecorded_calls": "12"},
            "window not a minute": {"unrecorded_window_start": "2026-01-01T00:00:01Z"},
            "window length": {"unrecorded_window_seconds": 120},
            "issued before the window ended": {"issued_at": "2026-01-01T00:00:30Z"},
            "refusal fields mixed in": {"rate_limit_phase": "opened"},
        }.items():
            self.rejected(gap(self.f, **extra), label)

    def test_gap_receipt_with_a_balance_claim_is_invalid(self):
        self.rejected(gap(self.f, remaining_budget_usd="0.000000"), "remaining")
        self.rejected(gap(self.f, budget={"spend_mode": "not_evaluated", "cost_credits": 0, "limit_credits": 1}), "limit")

    def test_gap_reason_is_legal_only_as_allowed_with_no_budget_evaluated(self):
        # Denied, paid, or with an evaluated budget: not a gap receipt.
        self.rejected(gap(self.f, decision="denied"), "denied")
        self.rejected(gap(self.f, budget={"spend_mode": "evaluated", "cost_credits": 1, "credit_unit": "credit"}), "evaluated budget")

    def test_gap_tampering_is_caught(self):
        pack = self.pack(gap(self.f, count=12))
        for surface in paid.VERIFIERS:
            tampered = copy.deepcopy(pack)
            tampered["receipts"][0]["unrecorded_calls"] = 1
            with self.subTest(surface=surface):
                self.assertFalse(self.verify(surface, tampered)["valid"])

    # ---- round 2: nothing nested, and one reading of provenance in both copies ----

    # Claims a new-profile receipt must not make, at any depth, spelled the way a
    # signer might hide them. Each is signed, so only the profile can refuse it.
    HIDDEN_CLAIMS = {
        "payment_hash in metadata": ("metadata", {"public_key_ed25519_b64": "k", "tenant_id": "t", "request_id": "r", "payment_hash": "ab"}),
        "rail nested in metadata": ("metadata", {"public_key_ed25519_b64": "k", "tenant_id": "t", "request_id": "r", "nested": {"rail": "credits"}}),
        "amount_sats nested twice": ("metadata", {"public_key_ed25519_b64": "k", "tenant_id": "t", "request_id": "r", "a": {"b": [{"amount_sats": 5}]}}),
        "camelCase payment key": ("metadata", {"public_key_ed25519_b64": "k", "tenant_id": "t", "request_id": "r", "Payment-Hash": "ab"}),
        "invoice in policy": ("policy", {"policy_id": "p", "policy_version": "v1", "invoice_hash": "x"}),
        "spend_mode nested": ("metadata", {"public_key_ed25519_b64": "k", "tenant_id": "t", "request_id": "r", "x": {"spend_mode": "credits"}}),
        "cost_credits nested": ("policy", {"policy_id": "p", "policy_version": "v1", "x": {"cost_credits": 3}}),
        "remaining nested": ("metadata", {"public_key_ed25519_b64": "k", "tenant_id": "t", "request_id": "r", "x": {"remaining_credits": 3}}),
        "balance nested": ("metadata", {"public_key_ed25519_b64": "k", "tenant_id": "t", "request_id": "r", "x": [{"balance": 3}]}),
    }

    def with_hidden(self, build, field, value):
        r = build()
        base = r.get(field)
        merged = dict(base) if isinstance(base, dict) else {}
        merged.update(value)
        r[field] = merged
        return r

    def test_nested_payment_spend_and_balance_claims_are_invalid_in_every_new_profile(self):
        builders = {
            "allowed": lambda: allowed_free(self.f),
            "opener": lambda: opener(self.f),
            "closer": lambda: closer(self.f),
            "gap": lambda: gap(self.f),
        }
        for shape, build in builders.items():
            for label, (field, value) in self.HIDDEN_CLAIMS.items():
                r = self.with_hidden(build, field, value)
                if field == "metadata":
                    r["metadata"].setdefault("public_key_ed25519_b64", self.f.x)
                self.rejected(r, f"{shape}: {label}")

    def test_nested_claims_are_invalid_on_account_level_window_receipts(self):
        for reason in ("token_revoked", "capability_expired"):
            for label, (field, value) in self.HIDDEN_CLAIMS.items():
                r = account_level(self.f, reason)
                merged = dict(r.get(field) or {})
                merged.update(value)
                r[field] = merged
                pack = self.f.pack(r)
                for surface in paid.VERIFIERS:
                    with self.subTest(reason=reason, case=label, surface=surface):
                        result = self.verify(surface, pack)
                        self.assertTrue(result["checks"]["signature_valid"])
                        self.assertFalse(result["valid"], result)

    def test_unsigned_pack_wrapper_claims_are_invalid(self):
        wrappers = {
            "metadata.balance": {"metadata": {"balance": 3}},
            "nested payment": {"extra": {"nested": {"payment_hash": "ab"}}},
            "nested rail": {"extra": [{"rail": "l402"}]},
            "remaining": {"remaining_credits": 4},
            "spend": {"x": {"spend_mode": "credits"}},
        }
        for shape, build in {"allowed": lambda: allowed_free(self.f), "closer": lambda: closer(self.f), "gap": lambda: gap(self.f)}.items():
            for label, extra in wrappers.items():
                pack = self.pack(build())
                pack.update(copy.deepcopy(extra))
                for surface in paid.VERIFIERS:
                    with self.subTest(shape=shape, case=label, surface=surface):
                        result = self.verify(surface, pack)
                        self.assertFalse(result["valid"], result)

    def test_pack_wrapper_budget_state_must_be_the_exact_no_balance_budget(self):
        for label, state in {"missing": None, "nonzero cost": {"spend_mode": "not_evaluated", "cost_credits": 2},
                             "extra key": {"spend_mode": "not_evaluated", "cost_credits": 0, "note": "x"}}.items():
            pack = self.pack(allowed_free(self.f))
            if state is None:
                del pack["budget_state"]
            else:
                pack["budget_state"] = state
            for surface in paid.VERIFIERS:
                with self.subTest(case=label, surface=surface):
                    self.assertFalse(self.verify(surface, pack)["valid"])

    # ---- the Pack wrapper's authority, budget and policy mirrors ----

    def test_a_new_profile_pack_must_carry_every_mirror_in_both_copies(self):
        # The contract (tools/VERIFIER-CONTRACT.md, "The Pack wrapper's
        # mirrors"): a Pack of an allowed / policy_allowed no-balance receipt, a
        # window opener or closer, or a gap record repeats the receipt's
        # authority, budget and policy. Leaving one out is refused by both
        # copies. Each case is signed and the signature is valid: it fails on
        # the wrapper only. 4 shapes x 3 fields = 12 cases, and the same for the
        # account-level window receipts of a revoked or expired key.
        shapes = {"allowed": allowed_free, "opener": opener, "closer": closer, "gap": gap}
        cases = 0
        for shape, build in shapes.items():
            whole = self.pack(build(self.f))
            for surface in paid.VERIFIERS:
                with self.subTest(shape=shape, case="whole", surface=surface):
                    self.assertTrue(self.verify(surface, whole)["valid"], "the pack with every mirror must be valid")
            for field in ("authority", "budget", "policy"):
                pack = self.pack(build(self.f))
                del pack[field]
                cases += 1
                for surface in paid.VERIFIERS:
                    with self.subTest(shape=shape, removed=field, surface=surface):
                        result = self.verify(surface, pack)
                        self.assertTrue(result["checks"]["signature_valid"], "the case must fail on the wrapper, not the signature")
                        self.assertFalse(result["valid"], result)
                        self.assertIn("pack_primary_mismatch", result["reason_codes"])
        self.assertEqual(cases, 12)
        for reason in ("token_revoked", "capability_expired"):
            for field in ("authority", "budget", "policy"):
                pack = self.pack(account_level(self.f, reason))
                del pack[field]
                for surface in paid.VERIFIERS:
                    with self.subTest(shape=reason, removed=field, surface=surface):
                        self.assertFalse(self.verify(surface, pack)["valid"])

    def test_a_mirror_that_differs_from_the_receipt_is_refused_in_both_copies(self):
        for shape, build in {"allowed": allowed_free, "opener": opener, "closer": closer, "gap": gap}.items():
            for field, value in (("authority", {"provenance_level": "no_verified_capability"}),
                                 ("budget", {"spend_mode": "not_evaluated", "cost_credits": 1}),
                                 ("policy", {"policy_id": "other", "policy_version": "v1"})):
                pack = self.pack(build(self.f))
                pack[field] = value
                for surface in paid.VERIFIERS:
                    with self.subTest(shape=shape, changed=field, surface=surface):
                        self.assertFalse(self.verify(surface, pack)["valid"])

    def test_legacy_packs_keep_their_mirror_rules(self):
        # Not changed by the new-profile rule: a legacy Pack that leaves a
        # mirror out is accepted by tools (the field is optional there) and
        # refused by demo, exactly as on main. The new-profile rule does not
        # reach them.
        for decision, reason, _ in paid.LEGACY_PROFILES:
            for field in ("authority", "budget", "policy"):
                pack = self.f.pack(self.f.receipt(decision, reason))
                del pack[field]
                with self.subTest(profile=f"{decision}/{reason}", removed=field):
                    self.assertTrue(self.verify("tools", pack)["valid"])
                    self.assertFalse(self.verify("demo", pack)["valid"])

    def test_authority_of_a_new_profile_receipt_must_name_a_verified_key(self):
        # A call or refusal by a key that verified: verified_macaroon_caveats, the
        # capability hash bound, raw fields redacted. Missing authority, the
        # no-capability form, an unredacted field or a hash that is not the
        # receipt's fail in both copies the same way.
        def without_authority(r):
            r.pop("authority")
            return r

        def other_authority(**fields):
            def build(r):
                r["authority"] = {"provenance_level": "verified_macaroon_caveats", "capability_hash": r["capability_hash"],
                                  "raw_capability_token": "redacted", "raw_budget_id": "redacted",
                                  "raw_token_id": "redacted", "raw_parent_token_id": "redacted"}
                r["authority"].update(fields)
                return r
            return build

        bad = {
            "authority missing": without_authority,
            "no_verified_capability": lambda r: r.update(authority={"provenance_level": "no_verified_capability"}) or r,
            "sandbox form": lambda r: r.update(authority={"provenance_level": "sandbox_no_capability"}) or r,
            "raw token not redacted": other_authority(raw_capability_token="sgcap_x"),
            "unknown authority field": other_authority(surprise="x"),
            "hash not the receipt's": other_authority(capability_hash="sha256:another"),
        }
        for shape, build in {"allowed": lambda: allowed_free(self.f), "opener": lambda: opener(self.f),
                             "closer": lambda: closer(self.f), "gap": lambda: gap(self.f)}.items():
            for label, edit in bad.items():
                self.rejected(edit(build()), f"{shape}: {label}")

    def test_authority_that_is_not_an_object_is_malformed_input_in_both_copies(self):
        for build in (lambda: allowed_free(self.f), lambda: closer(self.f), lambda: gap(self.f)):
            r = build()
            r["authority"] = "verified_macaroon_caveats"
            pack = self.pack(r)
            for surface in paid.VERIFIERS:
                with self.subTest(surface=surface):
                    result = self.verify(surface, pack)
                    self.assertFalse(result["valid"], result)
                    self.assertEqual(result["reason_codes"], ["invalid_input"])

    def test_revoked_and_expired_window_receipts_name_no_capability(self):
        for reason in ("token_revoked", "capability_expired"):
            for label, edit in {
                "verified authority": lambda r: r.update(authority=paid.verified_not_evaluated(self.f)["authority"], capability_hash="sha256:verified-capability") or r,
                "authority missing": lambda r: r.pop("authority") and r,
                "capability hash present": lambda r: r.update(capability_hash="sha256:verified-capability") or r,
            }.items():
                r = edit(account_level(self.f, reason))
                pack = self.f.pack(r)
                for surface in paid.VERIFIERS:
                    with self.subTest(reason=reason, case=label, surface=surface):
                        result = self.verify(surface, pack)
                        self.assertTrue(result["checks"]["signature_valid"])
                        self.assertFalse(result["valid"], result)

    def test_old_policy_denied_with_a_verified_key_is_still_valid_without_window_fields(self):
        # Not a new profile: judged as before, in both copies.
        self.accepted(paid.verified_not_evaluated(self.f), "verified policy denial")


if __name__ == "__main__":
    unittest.main()
