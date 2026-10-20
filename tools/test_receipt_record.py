"""tools/verify_receipt_record.py checks a downloaded record: NDJSON packs plus a
signed manifest. Synthetic keys only. Run:
  python -B -m unittest discover -s tools -p 'test_receipt_record.py' -v
"""
import copy
import importlib.util
import json
import os
import subprocess
import sys
import tempfile
import unittest
from datetime import datetime, timedelta, timezone
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / "tools"))

import test_paid_rail as paid  # noqa: E402
import test_every_action_receipts as ea  # noqa: E402

spec = importlib.util.spec_from_file_location("verify_receipt_record", ROOT / "tools" / "verify_receipt_record.py")
REC = importlib.util.module_from_spec(spec)
spec.loader.exec_module(REC)

NOW = paid.NOW  # 2026-01-02T00:00:00Z
KEY = "sha256:agent-key-1"
OTHER_KEY = "sha256:agent-key-2"
T0 = datetime(2026, 1, 1, 0, 0, 0, tzinfo=timezone.utc)
FROM, TO = "2026-01-01T00:00:00Z", "2026-01-02T00:00:00Z"
GENERATED = "2026-01-01T12:00:00Z"


def stamp(dt):
    return dt.strftime("%Y-%m-%dT%H:%M:%SZ")


class Builder:
    def __init__(self, f):
        self.f = f
        self.n = 0

    def base(self, receipt, seconds, key=KEY):
        """Give a receipt its own ids, a time and the agent key."""
        self.n += 1
        r = copy.deepcopy(receipt)
        r["receipt_id"] = f"rcpt-{self.n}"
        r["evidence_pack_id"] = f"ep-{self.n}"
        r["issued_at"] = stamp(T0 + timedelta(seconds=seconds))
        r["metadata"] = dict(r["metadata"], request_id=f"req-{self.n}")
        r["authority"] = dict(r["authority"], token_id_hash=key)
        return r

    def pack(self, receipt):
        return paid.verified_not_evaluated_pack(self.f, receipt)

    def allowed(self, seconds, **extra):
        r = ea.allowed_free(self.f)
        r.update(extra)
        return self.pack(self.base(r, seconds))

    def refused(self, seconds, **extra):
        r = paid.verified_not_evaluated(self.f)
        r.update(denial_code="TOOL_ARGUMENT_DENIED")
        r.update(extra)
        return self.pack(self.base(r, seconds))

    def opener(self, seconds, **extra):
        r = ea.opener(self.f)
        r.update(extra)
        return self.pack(self.base(r, seconds))

    def closer(self, seconds, count=5, **extra):
        r = ea.closer(self.f, count=count)
        r.update(extra)
        return self.pack(self.base(r, seconds))

    def gap(self, seconds, count=4):
        return self.pack(self.base(ea.gap(self.f, count=count), seconds))

    def manifest(self, packs, windows=None, sign=True, **overrides):
        hashes = [p["receipt_hash"] for p in packs]
        reasons = {}
        for p in packs:
            reasons[p["decision_reason"]] = reasons.get(p["decision_reason"], 0) + 1
        m = {
            "schema_version": REC.MANIFEST_SCHEMA,
            "canonicalization": "jcs-rfc8785", "hash_algorithm": "sha256", "signature_algorithm": "ed25519",
            "issuer": "https://issuer.example", "issuer_kid": "test-key",
            "key_hash": KEY, "from": FROM, "to": TO, "generated_at": GENERATED,
            "count": len(packs),
            "receipt_hashes_sha256": REC.receipt_hash_list_digest(hashes),
            "counts_by_decision_reason": reasons,
            "windows": windows if windows is not None else [],
            "metadata": {"public_key_ed25519_b64": self.f.x},
        }
        if hashes:
            m["first_receipt_hash"], m["last_receipt_hash"] = hashes[0], hashes[-1]
        m.update(overrides)
        return self.sign(m) if sign else m

    def sign(self, m, key=None):
        m = {k: v for k, v in m.items() if k not in REC.MANIFEST_UNSIGNED_FIELDS}
        m["manifest_hash"] = REC.manifest_hash(m)
        m["signature"] = "ed25519:" + paid.V.b64url_encode((key or self.f.key).sign(REC.manifest_payload(m)))
        return m


def window_entry(opener_pack, closer_pack=None, calls=None):
    r = opener_pack["receipts"][0]
    return {
        "route_or_tool": r["route_or_tool"], "window_start": ea.WINDOW, "denial_code": "RATE_LIMITED",
        "opened_receipt_hash": opener_pack["receipt_hash"],
        "closed_receipt_hash": closer_pack["receipt_hash"] if closer_pack else None,
        "refused_calls": calls,
    }


def text(packs, manifest):
    return "\n".join(json.dumps(x, sort_keys=True) for x in [*packs, manifest]) + "\n"


class ReceiptRecord(unittest.TestCase):
    def setUp(self):
        self.f = paid.Fixture()
        self.b = Builder(self.f)

    def verify(self, record, **kw):
        options = dict(jwks=self.f.jwks, require_trusted_issuer=True, now=NOW, allow_mock=True)
        options.update(kw)
        return REC.verify_record(record, **options)

    def clean(self):
        packs = [self.b.allowed(10), self.b.refused(20), self.b.allowed(30), self.b.allowed(40)]
        return packs, self.b.manifest(packs)

    def codes(self, result):
        return {p["code"] for p in result["problems"]}

    # ---- a clean record ----

    def test_receipt_record_clean_record_passes(self):
        packs, manifest = self.clean()
        result = self.verify(text(packs, manifest))
        self.assertTrue(result["valid"], result)
        self.assertEqual(result["problems"], [])
        self.assertEqual(result["summary"], "4 receipts, 1 refusals, 0 windows, 0 problems")
        self.assertTrue(result["checks"]["manifest_trusted_issuer"])

    def test_clean_record_passes_without_a_trusted_issuer_by_the_embedded_key(self):
        packs, manifest = self.clean()
        result = self.verify(text(packs, manifest), jwks=None, require_trusted_issuer=False)
        self.assertTrue(result["valid"], result)
        self.assertFalse(result["checks"]["manifest_trusted_issuer"])

    def test_trusted_issuer_is_required_when_asked_for(self):
        packs, manifest = self.clean()
        result = self.verify(text(packs, manifest), jwks=None, require_trusted_issuer=True)
        self.assertFalse(result["valid"])

    def test_an_empty_record_of_one_manifest_is_valid(self):
        manifest = self.b.manifest([])
        self.assertTrue(self.verify(text([], manifest))["valid"])

    # ---- nothing dropped, added, moved or changed ----

    def test_receipt_record_dropped_receipt_fails(self):
        packs, manifest = self.clean()
        result = self.verify(text(packs[:1] + packs[2:], manifest))
        self.assertFalse(result["valid"])
        self.assertIn("count_mismatch", self.codes(result))
        self.assertIn("receipt_hashes_mismatch", self.codes(result))

    def test_dropped_receipt_with_the_count_fixed_and_re_signed_still_fails(self):
        packs, manifest = self.clean()
        fixed = self.b.sign(dict(manifest, count=3))
        result = self.verify(text(packs[:1] + packs[2:], fixed))
        self.assertFalse(result["valid"])
        self.assertNotIn("count_mismatch", self.codes(result))
        self.assertIn("receipt_hashes_mismatch", self.codes(result))

    def test_dropping_the_last_receipt_is_caught(self):
        packs, manifest = self.clean()
        result = self.verify(text(packs[:-1], manifest))
        self.assertIn("first_or_last_mismatch", self.codes(result))

    def test_receipt_record_duplicate_fails(self):
        packs, manifest = self.clean()
        doubled = packs[:2] + [packs[1]] + packs[2:]
        # Even with the manifest made to match the doubled list.
        again = self.b.manifest(doubled)
        result = self.verify(text(doubled, again))
        self.assertFalse(result["valid"])
        self.assertIn("duplicate_receipt", self.codes(result))

    def test_receipt_record_reordered_fails(self):
        packs, manifest = self.clean()
        swapped = [packs[0], packs[2], packs[1], packs[3]]
        result = self.verify(text(swapped, manifest))
        self.assertFalse(result["valid"])
        self.assertIn("receipt_hashes_mismatch", self.codes(result))
        # Re-signed over the swapped order, the times still go backwards.
        result = self.verify(text(swapped, self.b.manifest(swapped)))
        self.assertIn("out_of_order", self.codes(result))

    def test_a_changed_receipt_fails(self):
        packs, manifest = self.clean()
        packs[1]["receipts"][0]["decision_reason"] = "budget_exhausted"
        result = self.verify(text(packs, manifest))
        self.assertFalse(result["valid"])
        self.assertIn("receipt_invalid", self.codes(result))

    def test_a_receipt_signed_by_another_key_fails(self):
        other = paid.Fixture()
        packs, _ = self.clean()
        r = self.b.base(ea.allowed_free(other), 25)
        packs.insert(2, paid.verified_not_evaluated_pack(other, r))
        result = self.verify(text(packs, self.b.manifest(packs)))
        self.assertFalse(result["valid"])
        self.assertIn("receipt_invalid", self.codes(result))

    # ---- the manifest ----

    def test_a_changed_manifest_fails(self):
        packs, manifest = self.clean()
        for field, value in (("count", 3), ("key_hash", OTHER_KEY), ("to", "2026-01-05T00:00:00Z")):
            with self.subTest(field=field):
                tampered = dict(manifest, **{field: value})
                result = self.verify(text(packs, tampered))
                self.assertFalse(result["valid"])
                self.assertIn("manifest_hash_mismatch", self.codes(result))

    def test_a_manifest_signed_by_another_key_fails(self):
        packs, manifest = self.clean()
        other = paid.Fixture()
        forged = self.b.sign(manifest, key=other.key)
        forged["metadata"] = {"public_key_ed25519_b64": self.f.x}
        result = self.verify(text(packs, forged))
        self.assertFalse(result["valid"])
        self.assertIn("manifest_signature_invalid", self.codes(result))

    def test_a_record_with_no_manifest_fails(self):
        packs, _ = self.clean()
        result = self.verify(text(packs[:-1], packs[-1]))
        self.assertIn("manifest_missing_or_misplaced", self.codes(result))

    def test_a_manifest_that_is_not_last_fails(self):
        packs, manifest = self.clean()
        record = "\n".join(json.dumps(x) for x in [manifest, *packs]) + "\n"
        self.assertIn("manifest_missing_or_misplaced", self.codes(self.verify(record)))

    def test_counts_by_reason_must_match(self):
        packs, manifest = self.clean()
        wrong = self.b.sign(dict(manifest, counts_by_decision_reason={"policy_allowed": 4}))
        self.assertIn("counts_by_reason_mismatch", self.codes(self.verify(text(packs, wrong))))

    # ---- range, order, one key ----

    def test_receipts_outside_the_range_fail(self):
        packs, manifest = self.clean()
        narrow = self.b.manifest(packs, **{"from": "2026-01-01T00:00:25Z"})
        self.assertIn("outside_range", self.codes(self.verify(text(packs, narrow))))
        early = self.b.manifest(packs, to="2026-01-01T00:00:40Z")
        self.assertIn("outside_range", self.codes(self.verify(text(packs, early))), "to is exclusive")

    def test_a_receipt_of_another_key_fails(self):
        packs = [self.b.allowed(10), self.pack_for(OTHER_KEY, 20), self.b.allowed(30)]
        result = self.verify(text(packs, self.b.manifest(packs)))
        self.assertFalse(result["valid"])
        self.assertIn("key_hash_mismatch", self.codes(result))

    def pack_for(self, key, seconds):
        return self.b.pack(self.b.base(ea.allowed_free(self.f), seconds, key=key))

    def test_a_repeated_request_id_is_a_warning_not_a_failure(self):
        a = self.b.allowed(10)
        b = self.b.allowed(20)
        b["receipts"][0]["metadata"]["request_id"] = a["receipts"][0]["metadata"]["request_id"]
        # re-sign b after the edit
        r = self.f.sign({k: v for k, v in b["receipts"][0].items() if k not in ("receipt_hash", "signature")})
        b = paid.verified_not_evaluated_pack(self.f, r)
        packs = [a, b]
        result = self.verify(text(packs, self.b.manifest(packs)))
        self.assertTrue(result["valid"], result)
        self.assertIn("duplicate_request_id", {w["code"] for w in result["warnings"]})

    # ---- refusal windows ----

    def test_a_window_with_opener_and_closer_passes(self):
        o, c = self.b.opener(10), self.b.closer(80, count=57)
        packs = [o, self.b.allowed(30), c]
        manifest = self.b.manifest(packs, windows=[window_entry(o, c, 57)])
        result = self.verify(text(packs, manifest))
        self.assertTrue(result["valid"], result)
        self.assertEqual(result["counts"]["windows"], 1)
        self.assertEqual(result["summary"], "3 receipts, 2 refusals, 1 windows, 0 problems")

    # An opener with no closer: invalid when the closer is overdue, an explicit
    # pending or boundary result when it is not.

    def states(self, result):
        return [w["state"] for w in result["window_states"]]

    def test_receipt_record_window_without_close_fails_when_the_closer_is_overdue(self):
        o = self.b.opener(10)
        packs = [o]
        manifest = self.b.manifest(packs, windows=[window_entry(o)])  # generated hours after the window
        result = self.verify(text(packs, manifest))
        self.assertFalse(result["valid"], "a window whose closer is overdue is a missing receipt")
        self.assertIn("window_closer_overdue", self.codes(result))
        self.assertEqual(self.states(result), ["overdue"])
        self.assertNotEqual(self.run_cli(text(packs, manifest)).returncode, 0)

    def test_the_closer_is_due_when_the_window_end_plus_grace_has_passed(self):
        o = self.b.opener(10)  # the window is 00:00:00 to 00:01:00; the closer is due at 00:01:15
        for generated, state in (("2026-01-01T00:01:14Z", "pending"), ("2026-01-01T00:01:15Z", "overdue"), ("2026-01-01T00:03:00Z", "overdue")):
            with self.subTest(generated_at=generated):
                manifest = self.b.manifest([o], windows=[window_entry(o)], generated_at=generated)
                result = self.verify(text([o], manifest))
                self.assertEqual(self.states(result), [state])
                self.assertEqual(result["valid"], state == "pending", result["problems"])

    def test_a_window_that_is_not_yet_due_is_pending_and_valid(self):
        o = self.b.opener(10)
        manifest = self.b.manifest([o], windows=[window_entry(o)], generated_at="2026-01-01T00:01:10Z")
        result = self.verify(text([o], manifest))
        self.assertTrue(result["valid"], result)
        self.assertEqual(self.states(result), ["pending"])
        self.assertIn("window_closer_pending", {w["code"] for w in result["warnings"]})
        self.assertEqual(self.run_cli(text([o], manifest)).returncode, 0)

    def test_the_end_of_the_export_bounds_the_wait_as_well(self):
        o = self.b.opener(10)
        # The range ends 10 seconds after the window; the closer is due 5 seconds later.
        pending = self.b.manifest([o], windows=[window_entry(o)], **{"to": "2026-01-01T00:01:10Z"})
        self.assertEqual(self.states(self.verify(text([o], pending))), ["pending"])
        self.assertTrue(self.verify(text([o], pending))["valid"])
        overdue = self.b.manifest([o], windows=[window_entry(o)], **{"to": "2026-01-01T00:01:20Z"})
        self.assertEqual(self.states(self.verify(text([o], overdue))), ["overdue"])

    def test_a_window_that_runs_past_the_end_of_the_range_is_a_boundary_not_a_failure(self):
        o = self.b.opener(10)
        manifest = self.b.manifest([o], windows=[window_entry(o)], **{"to": "2026-01-01T00:00:50Z"})
        result = self.verify(text([o], manifest))
        self.assertTrue(result["valid"], result)
        self.assertEqual(self.states(result), ["boundary"])
        self.assertIn("window_runs_past_range", {w["code"] for w in result["warnings"]})

    def test_a_closed_window_is_reported_closed(self):
        o, c = self.b.opener(10), self.b.closer(80, count=3)
        manifest = self.b.manifest([o, c], windows=[window_entry(o, c, 3)])
        self.assertEqual(self.states(self.verify(text([o, c], manifest))), ["closed"])

    def test_a_closer_with_no_opener_fails_unless_the_window_began_before_the_record(self):
        c = self.b.closer(80)
        manifest = self.b.manifest([c], windows=[window_entry(c, c, 5)])
        self.assertIn("window_closer_without_opener", self.codes(self.verify(text([c], manifest))))
        later = self.b.manifest([c], windows=[window_entry(c, c, 5)], **{"from": "2026-01-01T00:01:00Z"})
        result = self.verify(text([c], later))
        self.assertNotIn("window_closer_without_opener", self.codes(result))
        self.assertIn("window_started_before_record", {w["code"] for w in result["warnings"]})

    def test_two_openers_for_one_window_fail(self):
        a, b = self.b.opener(10), self.b.opener(11)
        packs = [a, b]
        result = self.verify(text(packs, self.b.manifest(packs, windows=[window_entry(a)])))
        self.assertIn("window_repeated", self.codes(result))

    def test_the_manifest_must_list_the_windows_the_receipts_hold(self):
        o, c = self.b.opener(10), self.b.closer(80)
        packs = [o, c]
        for label, windows in (("none", []), ("wrong count", [window_entry(o, c, 99)])):
            with self.subTest(label):
                result = self.verify(text(packs, self.b.manifest(packs, windows=windows)))
                self.assertIn("manifest_windows_mismatch", self.codes(result))

    # ---- Observe gaps and where the record starts ----

    def test_observe_gap_receipts_are_counted_and_reported(self):
        packs = [self.b.allowed(10), self.b.gap(80, count=4), self.b.gap(150, count=3)]
        # The second gap's window is the same minute as the first in the
        # fixture; give it its own receipt time only.
        result = self.verify(text(packs, self.b.manifest(packs)))
        self.assertTrue(result["valid"], result)
        self.assertEqual(result["counts"]["unrecorded_calls"], 7)
        self.assertIn("calls_without_receipts", {w["code"] for w in result["warnings"]})

    def test_a_record_that_starts_mid_range_says_so(self):
        packs, _ = self.clean()
        manifest = self.b.manifest(packs, record_starts_at="2026-01-01T00:00:15Z")
        result = self.verify(text(packs, manifest))
        self.assertTrue(result["valid"], result)
        self.assertIn("record_starts_after_range", {w["code"] for w in result["warnings"]})

    # ---- the command line ----

    def run_cli(self, record, *args, jwks_doc=None):
        with tempfile.TemporaryDirectory() as tmp:
            rec = Path(tmp) / "record.ndjson"
            rec.write_text(record, encoding="utf-8")
            jwks = Path(tmp) / "jwks.json"
            jwks.write_text(json.dumps(jwks_doc if jwks_doc is not None else self.f.jwks), encoding="utf-8")
            env = dict(os.environ, PYTHONPATH=str(ROOT / "demo" / "_vendor"), PYTHONDONTWRITEBYTECODE="1")
            proc = subprocess.run(
                [sys.executable, "-B", str(ROOT / "tools" / "verify_receipt_record.py"), str(rec),
                 "--jwks-file", str(jwks), "--require-trusted-issuer", "--allow-mock", "--now", NOW, *args],
                capture_output=True, text=True, env=env, timeout=30)
        self.assertNotIn("Traceback", proc.stderr, proc.stderr)
        return proc

    # ---- round 2: one receipt per line, the signing key, exact counts ----

    def with_receipts(self, pack, receipts):
        edited = copy.deepcopy(pack)
        edited["receipts"] = receipts
        return edited

    def cli_fails(self, record, code, label):
        """The command line, with a trusted JWKS, exits non-zero and names the problem."""
        proc = self.run_cli(record)
        self.assertEqual(proc.returncode, 1, f"{label}: {proc.stdout}")
        result = json.loads(proc.stdout)
        self.assertFalse(result["valid"], label)
        self.assertIn(code, {p["code"] for p in result["problems"]}, f"{label}: {result['problems']}")

    def test_cli_a_receipt_added_to_a_pack_fails(self):
        packs, manifest = self.clean()
        extra = copy.deepcopy(self.b.allowed(50)["receipts"][0])  # a valid, signed receipt
        edited = [self.with_receipts(packs[1], packs[1]["receipts"] + [extra])]
        record = text(packs[:1] + edited + packs[2:], manifest)
        self.cli_fails(record, "pack_not_one_receipt", "receipt added")

    def test_cli_a_receipt_dropped_from_a_pack_fails(self):
        packs, manifest = self.clean()
        edited = self.with_receipts(packs[1], [])
        self.assertNotEqual(self.run_cli(text(packs[:1] + [edited] + packs[2:], manifest)).returncode, 0)
        self.cli_fails(text(packs[:1] + [edited] + packs[2:], manifest), "pack_not_one_receipt", "receipt dropped")

    def test_cli_a_receipt_duplicated_inside_a_pack_fails(self):
        packs, manifest = self.clean()
        r = packs[1]["receipts"][0]
        edited = self.with_receipts(packs[1], [r, copy.deepcopy(r)])
        self.cli_fails(text(packs[:1] + [edited] + packs[2:], manifest), "pack_not_one_receipt", "receipt duplicated")

    def test_cli_receipts_reordered_inside_a_pack_fail(self):
        packs, manifest = self.clean()
        r1, r2 = packs[1]["receipts"][0], copy.deepcopy(self.b.allowed(50)["receipts"][0])
        # Even a record whose manifest was made from the two-receipt packs is refused, in either order.
        for order in ([r1, r2], [r2, r1]):
            edited = self.with_receipts(packs[1], order)
            self.cli_fails(text(packs[:1] + [edited] + packs[2:], manifest), "pack_not_one_receipt", "reordered")

    def test_an_issuer_kid_that_is_not_the_receipts_fails(self):
        packs, manifest = self.clean()
        # The same key under a second name: the manifest verifies, but it says a
        # different key signed than the one that signed the receipts.
        rotated = self.b.sign(dict(manifest, issuer_kid="rotated-key"))
        jwks = copy.deepcopy(self.f.jwks)
        jwks["keys"].append(dict(jwks["keys"][0], kid="rotated-key"))
        proc = self.run_cli(text(packs, rotated), jwks_doc=jwks)
        self.assertEqual(proc.returncode, 1, proc.stdout)
        self.assertIn("issuer_kid_mismatch", {p["code"] for p in json.loads(proc.stdout)["problems"]})
        # Without a trusted issuer, through the embedded key.
        result = self.verify(text(packs, rotated), jwks=None, require_trusted_issuer=False)
        self.assertIn("issuer_kid_mismatch", self.codes(result))

    def test_cli_boolean_counts_by_decision_reason_fail(self):
        packs, manifest = self.clean()
        reasons = dict(manifest["counts_by_decision_reason"])
        one = next(k for k, v in reasons.items() if v == 1)
        self.assertEqual(reasons[one], 1)
        for label, value in (("true for 1", True),):
            edited = self.b.sign(dict(manifest, counts_by_decision_reason=dict(reasons, **{one: value})))
            self.cli_fails(text(packs, edited), "manifest_count_invalid", label)
        zero = self.b.sign(dict(manifest, counts_by_decision_reason=dict(reasons, never_seen=False)))
        self.cli_fails(text(packs, zero), "manifest_count_invalid", "false for a reason that never happened")
        for bad in (True, 1.0, "4"):
            edited = self.b.sign(dict(manifest, count=bad))
            self.assertFalse(self.verify(text(packs, edited))["valid"], bad)

    def test_cli_an_empty_record_with_false_endpoints_fails(self):
        for field in ("first_receipt_hash", "last_receipt_hash"):
            for value in (False, "", 0, "sha256:invented"):
                manifest = self.b.manifest([], **{field: value})
                self.cli_fails(text([], manifest), "manifest_endpoint_on_empty_record", f"{field}={value!r}")
        # Absent, or null, is an empty record.
        self.assertEqual(self.run_cli(text([], self.b.manifest([]))).returncode, 0)
        self.assertEqual(self.run_cli(text([], self.b.manifest([], first_receipt_hash=None, last_receipt_hash=None))).returncode, 0)

    def test_cli_exit_codes_and_summary_line(self):
        packs, manifest = self.clean()
        ok = self.run_cli(text(packs, manifest))
        self.assertEqual(ok.returncode, 0, ok.stdout + ok.stderr)
        self.assertEqual(json.loads(ok.stdout)["valid"], True)
        self.assertIn("4 receipts, 1 refusals, 0 windows, 0 problems", ok.stderr)
        bad = self.run_cli(text(packs[:1] + packs[2:], manifest))
        self.assertEqual(bad.returncode, 1, bad.stdout)
        self.assertIn("problems", bad.stderr)
        self.assertNotIn("0 problems", bad.stderr)

    def test_cli_input_that_cannot_be_read_exits_2_or_reports_it(self):
        proc = self.run_cli("this is not json\n")
        self.assertEqual(proc.returncode, 1)
        self.assertIn("invalid_input", proc.stdout)
        with tempfile.TemporaryDirectory() as tmp:
            env = dict(os.environ, PYTHONPATH=str(ROOT / "demo" / "_vendor"), PYTHONDONTWRITEBYTECODE="1")
            gone = subprocess.run([sys.executable, "-B", str(ROOT / "tools" / "verify_receipt_record.py"), str(Path(tmp) / "missing.ndjson")],
                                  capture_output=True, text=True, env=env, timeout=30)
            self.assertEqual(gone.returncode, 2, gone.stdout)


if __name__ == "__main__":
    unittest.main()
