"""Shared four-copy contract. All pack cases are signed; fixtures are made up.

VERIFY_SECRET_BOUNDARY_PEER_ROOT adds the peer repository. Each CI checkout pins
its own two copies to the same constant shared-block SHA-256. Keep this file
byte-identical in public tools/ and enterprise tests/.
"""
import copy
import hashlib
import importlib.util
import os
from pathlib import Path
import re
import string
import unittest

from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PrivateKey
from cryptography.hazmat.primitives.serialization import Encoding, PublicFormat

ROOT = Path(__file__).resolve().parents[1]
SHARED_BLOCK_SHA256 = "47739e6611ea893fe390ec1ee6e263de098f6cb196b999617ea2817358308d60"
MARKERS = ("stk_fixture", "sgcap_fixture", "Bearer fixture", "bEaReR fixture")
ENCODED_STARTS = ("Bearer%20", "token%3D", "user%3A", "path%2F", "Bearer+",
                  "bEaReR+", "+", "token+", "token%3d", "user%3a", "path%2f")
ALPHABET = string.ascii_letters + string.digits + "_-"
SHAPES = (("evid_", 32), ("rcpt_", 22), ("ep_", 22), ("receipt-", 22),
          ("sha256:", 43), ("ed25519:", 86))
# Sealed policy budget ids are 64 characters TOTAL (mint- plus 59 hex).
# No-budget mint-/dlg- row ids are not HTTP spend subjects.
NEW_SHAPES = (
    ("quote", "quote_", 32, "access_id"),
    *((subject, subject + "-", 22, "request_id")
      for subject in ("observe", "req", "budget", "policy", "degradation")),
    ("mint_budget", "mint-" + "a" * 59 + "-", 22, "request_id"),
    ("delegation_budget", "a" * 32 + "-", 22, "request_id"),
    ("child_budget", "del-" + "a" * 24 + "-", 22, "request_id"),
    ("uuid_budget", "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa-", 22, "request_id"),
)
EXPECTED = [(r"Bearer\s+[A-Za-z0-9._~+/=-]+", re.IGNORECASE | re.UNICODE),
            (r"sgcap_[A-Za-z0-9._~+/=-]+", re.UNICODE),
            (r"stk_[A-Za-z0-9._~+/=-]+", re.UNICODE)]
NOW = "2026-01-02T00:00:00Z"


def checker_paths(root):
    if (root / "tools" / "verify_evidence_pack.py").is_file():
        return [root / surface / "verify_evidence_pack.py" for surface in ("tools", "demo")]
    return [root / "demo" / "verify_evidence_pack.py",
            root / "dashboard" / "public" / "verify_evidence_pack.py"]


def load_checkers():
    paths = checker_paths(ROOT)
    peer = os.environ.get("VERIFY_SECRET_BOUNDARY_PEER_ROOT")
    if peer:
        paths.extend(checker_paths(Path(peer).resolve()))
    modules = []
    for index, path in enumerate(paths):
        spec = importlib.util.spec_from_file_location("secret_boundary_copy_" + str(index), path)
        module = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(module)
        modules.append(module)
    return modules


def generated(prefix, length, marker, offset):
    return prefix + "A" * offset + marker + "B" * (length - offset - len(marker))


class SecretBoundary(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.checkers = load_checkers()
        # Publicly specified synthetic seed; never a production key.
        cls.key = Ed25519PrivateKey.from_private_bytes(bytes(range(32)))
        cls.x = cls.checkers[0].b64url_encode(cls.key.public_key().public_bytes(Encoding.Raw, PublicFormat.Raw))
        cls.jwks = {"keys": [{"kid": "fixture-key", "kty": "OKP", "crv": "Ed25519",
                              "alg": "EdDSA", "use": "sig", "x": cls.x}]}

    def signed_pack(self, value=None, location="path"):
        receipt = dict(schema_version="satgate.receipt.v1", canonicalization="jcs-rfc8785",
                       hash_algorithm="sha256", signature_algorithm="ed25519",
                       issuer="https://issuer.example", issuer_kid="fixture-key",
                       decision="allowed", decision_reason="payment_verified",
                       issued_at="2026-01-01T00:00:00Z", evidence_pack_id="fixture-pack",
                       evidence_url="https://issuer.example/synthetic-proof", receipt_id="fixture-receipt",
                       route_or_tool="/fixture", policy_version="v1",
                       policy={"policy_id": "fixture-policy", "policy_version": "v1"},
                       authority={"provenance_level": "verified_macaroon_caveats"},
                       metadata={"public_key_ed25519_b64": self.x, "request_id": "fixture-request"},
                       rail="l402", amount_sats=1, payment_hash="a" * 64,
                       invoice_hash="b" * 64, macaroon_hash="c" * 64,
                       budget={"spend_mode": "paid_rail", "rail": "l402", "amount_sats": 1})
        if location == "policy":
            receipt["policy"]["policy_id"] = value
        elif location == "nested":
            receipt["metadata"]["caller"] = [{"note": value}]
        elif location == "invoice_hash":
            receipt["invoice_hash"] = value
        elif location == "public_key":
            receipt["metadata"]["public_key_ed25519_b64"] = value
        elif location == "fake_key":
            receipt["metadata"]["caller"] = {"public_key_ed25519_b64": value}
        elif location == "access_id":
            receipt.update(evidence_access_id=value, evidence_url="https://issuer.example/v1/evidence/" + value)
        elif value is not None:
            receipt["metadata"][location] = value
        receipt["paid_rail_context"] = {k: receipt[k] for k in
                                       ("rail", "amount_sats", "payment_hash", "invoice_hash", "macaroon_hash")}
        checker = self.checkers[0]
        receipt["receipt_hash"] = checker.sha256_receipt_hash(receipt)
        receipt["signature"] = "ed25519:" + checker.b64url_encode(self.key.sign(checker.canonical_receipt_payload(receipt)))
        fields = ("issuer", "decision", "decision_reason", "evidence_pack_id", "evidence_url",
                  "issued_at", "receipt_id", "receipt_hash", "route_or_tool", "authority", "budget", "policy")
        pack = {k: copy.deepcopy(receipt[k]) for k in fields}
        pack.update(schema_version="satgate.evidence_pack.v1", receipts=[receipt])
        return pack

    def assert_pack(self, value, leakage, location="path"):
        pack = self.signed_pack(value, location)
        snapshot = copy.deepcopy(pack)
        for index, checker in enumerate(self.checkers):
            result = checker.verify_pack(pack, jwks=self.jwks, require_trusted_issuer=True, now=NOW)
            # No fixture text, keys, signatures or credential values in failures.
            self.assertEqual(result["checks"]["no_secret_leakage"], not leakage,
                             "leakage verdict copy " + str(index))
            self.assertEqual(result["valid"], not leakage, "full verdict copy " + str(index))
            self.assertTrue(result["trusted_issuer_valid"], "signed fixture must remain trusted")
            self.assertEqual(set(result["reason_codes"]), {"secret_leakage"} if leakage else set())
        self.assertTrue(pack == snapshot, "verifier must not mutate signed input")

    def test_all_copies_agree_on_patterns_and_flags(self):
        for checker in self.checkers:
            self.assertEqual([(p.pattern, p.flags) for p in checker.SECRET_PATTERNS], EXPECTED)
            source = Path(checker.__file__).read_text()
            start = source.index("# BEGIN SHARED SECRET SCAN")
            end = source.index("# END SHARED SECRET SCAN") + len("# END SHARED SECRET SCAN")
            self.assertEqual(hashlib.sha256(source[start:end].encode()).hexdigest(), SHARED_BLOCK_SHA256)
            self.assertIn("def contains_secret_marker(value: Any) -> bool:\n    raw = secret_scan_text(value)", source)

    def test_caller_and_config_placements(self):
        starts = ("", " ", "=", ":", "(", '"', ",", "/", "?", "&", "[", "]",
                  "\n", "\t", "\r", "\b", "\f", ".", "~", "A", "7", "_", "-",
                  "https://issuer.example/.", "https://issuer.example/~",
                  "https://issuer.example/?value=", "Authorization: ")
        for number, before in enumerate(starts):
            for marker_index, marker in enumerate(MARKERS):
                for location in ("path", "policy", "nested"):
                    with self.subTest(start=number, marker=marker_index, location=location):
                        self.assert_pack(before + marker, True, location)

    def test_bearer_after_every_ascii_character(self):
        for byte in range(128):
            for marker in MARKERS[2:]:
                with self.subTest(byte=byte, mixed=marker == MARKERS[3]):
                    self.assert_pack(chr(byte) + marker, True)

    def test_encoded_token_starts(self):
        for number, before in enumerate(ENCODED_STARTS):
            for index, marker in enumerate(MARKERS):
                with self.subTest(start=number, marker=index):
                    self.assert_pack(before + marker, True, "nested")

    def test_every_percent_escape(self):
        for byte in range(256):
            for lower in (False, True):
                escape = ("%%%02x" if lower else "%%%02X") % byte
                for index, marker in enumerate(MARKERS):
                    with self.subTest(byte=byte, lower=lower, marker=index):
                        self.assert_pack("value" + escape + marker, True)

    def test_malformed_escapes_and_all_continuations(self):
        for number, before in enumerate(("%2", "%GG", "%2G", "%G2", "%123", "%2FA") + tuple(ALPHABET + ".~")):
            for index, marker in enumerate(MARKERS):
                with self.subTest(start=number, marker=index):
                    self.assert_pack(before + marker, True)

    def test_generated_shapes_at_start_middle_end(self):
        # Shape fixtures are signed data, not claimed preimages or signatures.
        # Actual cryptographic fields are independently computed by signed_pack.
        for prefix, length in SHAPES:
            for marker in ("stk_Z", "sgcap_Z", "BearerZ"):
                for offset in (0, length // 2, length - len(marker)):
                    for wrap in (False, True):
                        with self.subTest(shape=prefix, marker_size=len(marker), offset=offset, url=wrap):
                            value = generated(prefix, length, marker, offset)
                            self.assert_pack("https://issuer.example/" + value + "?v=1" if wrap else value, False)

    def test_generated_shapes_all_base64url_predecessors(self):
        for prefix, length in SHAPES:
            for char_index, char in enumerate(ALPHABET):
                for marker in ("stk_Z", "sgcap_Z", "BearerZ"):
                    value = prefix + char + marker + "B" * (length - 1 - len(marker))
                    with self.subTest(shape=prefix, char=char_index, marker_size=len(marker)):
                        self.assert_pack(value, False)

    def test_nonwhole_shapes_are_still_scanned(self):
        for prefix, length in SHAPES:
            for marker in ("stk_Z", "sgcap_Z"):
                value = generated(prefix, length, marker, 1)
                variants = (value[:-1], value + "B") + tuple(value + c for c in ALPHABET) + tuple(c + value for c in ALPHABET)
                for number, variant in enumerate(variants):
                    with self.subTest(shape=prefix, marker_size=len(marker), variant=number):
                        self.assert_pack(variant, True)

    def test_generated_value_does_not_hide_adjacent_credential(self):
        for prefix, length in SHAPES:
            value = prefix + "A" * length
            for marker in MARKERS:
                for separator in ("", "/", ".", "~", " ", "\n", "\t", "\r"):
                    with self.subTest(shape=prefix, marker_size=len(marker), separator=ord(separator) if separator else -1):
                        self.assert_pack(value + separator + marker, True)
                        self.assert_pack(marker + separator + value, True)

    def test_public_key_only_in_its_own_field(self):
        for marker in ("stk_ZZ", "sgcap_ZZ", "BearerZ"):
            for offset in (0, 21, 43 - len(marker)):
                value = generated("", 43, marker, offset)
                with self.subTest(marker_size=len(marker), offset=offset):
                    self.assert_pack(value, False, "public_key")
                    if marker != "BearerZ":
                        for location in ("path", "fake_key"):
                            self.assert_pack(value, True, location)
                if marker != "BearerZ":
                    for variant in (value[:-1], value + "B", "A" + value, value + "/" + "stk_Z"):
                        self.assert_pack(variant, True, "public_key")
        # Same name outside receipts[*].metadata is not a generated-key slot.
        value = generated("", 43, "stk_Z", 0)
        pack = self.signed_pack()
        pack["public_key_ed25519_b64"] = value
        for checker in self.checkers:
            result = checker.verify_pack(pack, jwks=self.jwks, require_trusted_issuer=True, now=NOW)
            self.assertFalse(result["valid"])
            self.assertFalse(result["checks"]["no_secret_leakage"])

    def test_real_generated_digest_at_body_zero(self):
        # Independently reproducible SHA-256 counterexample from round 1 review.
        digest = "sha256:" + self.checkers[0].b64url_encode(hashlib.sha256(b"independent-review-45571039").digest())
        self.assertTrue(digest.startswith("sha256:" + "stk_"), "counterexample preimage")
        self.assert_pack(digest, False, "invoice_hash")

    def test_access_id_signed_pack(self):
        self.assert_pack("evid_" + "A" * 12 + "stk_" + "B" * 16, False, "access_id")

    def test_bearer_whole_generated_values_matches_main(self):
        # This match starts OUTSIDE the generated value. The placeholder must
        # preserve it even if the body itself has no credential marker.
        shapes = SHAPES + tuple((p, n) for _, p, n, _ in NEW_SHAPES)
        for index, (prefix, length) in enumerate(shapes):
            for scheme in ("Bearer", "bEaReR"):
                for spaces in (" ", "  "):
                    with self.subTest(shape=index, mixed=scheme != "Bearer", spaces=len(spaces)):
                        self.assert_pack(scheme + spaces + prefix + "A" * length, True)
            # JSON escaping means main does not see these as whitespace.
            for whitespace in ("\t", "\n", "\r"):
                with self.subTest(shape=index, whitespace=ord(whitespace)):
                    self.assert_pack("Bearer" + whitespace + prefix + "A" * length, False)

    def test_unaudited_request_subjects_stay_scanned(self):
        # These are NOT sealed HTTP budget subjects, despite resembling row ids.
        prefixes = ("caller-", "route-name-", "mint-" + "a" * 24 + "-",
                    "mint-" + "a" * 58 + "-", "mint-" + "a" * 60 + "-",
                    "mint-" + "a" * 64 + "-", "dlg-" + "a" * 24 + "-",
                    "del-" + "a" * 23 + "-", "del-" + "a" * 25 + "-",
                    "a" * 31 + "-", "a" * 33 + "-", "A" * 32 + "-",
                    "mint-" + "A" * 59 + "-", "del-" + "A" * 24 + "-",
                    "aaaaaaaa-aaaa-3aaa-8aaa-aaaaaaaaaaaa-",
                    "aaaaaaaa-aaaa-4aaa-7aaa-aaaaaaaaaaaa-")
        for index, prefix in enumerate(prefixes):
            for marker in ("stk_Q", "sgcap_Q"):
                with self.subTest(prefix=index, marker_size=len(marker)):
                    self.assert_pack(generated(prefix, 22, marker, 1), True, "request_id")
        # A secret in the prefix must not be hidden by a generated suffix.
        for marker in MARKERS:
            self.assert_pack(marker + "-" + "A" * 22, True, "request_id")

    def test_sensitive_fields_are_unchanged(self):
        for field in ("raw_capability_token", "authorization_header", "payment_secret"):
            for value in ("fixture_without_a_prefix", "evid_" + "A" * 32):
                self.assert_pack(value, True, field)
            self.assert_pack("redacted", False, field)

    def test_generator_alphabets_and_lengths(self):
        for checker in self.checkers:
            self.assertEqual(set(checker.b64url_encode(bytes(range(256)))), set(ALPHABET))
            for byte_length, length in ((16, 22), (24, 32), (32, 43), (64, 86)):
                self.assertEqual(len(checker.b64url_encode(bytes(range(byte_length)))), length)
        roots = {ROOT}
        if os.environ.get("VERIFY_SECRET_BOUNDARY_PEER_ROOT"):
            roots.add(Path(os.environ["VERIFY_SECRET_BOUNDARY_PEER_ROOT"]).resolve())
        for root in roots:
            gateway = root / "gateway" / "internal"
            if not gateway.is_dir():
                continue
            for surface in ("evidence", "proxy"):
                source = (gateway / surface / "receipt.go").read_text()
                for expression in ('"evid_" + base64.RawURLEncoding.EncodeToString(b[:])',
                                   '"rcpt_" + base64.RawURLEncoding.EncodeToString(receiptIDHash[:])[:22]',
                                   '"ep_" + base64.RawURLEncoding.EncodeToString(evidencePackHash[:])[:22]',
                                   '"sha256:" + base64.RawURLEncoding.EncodeToString(hash[:])',
                                   '"ed25519:" + base64.RawURLEncoding.EncodeToString(signature)',
                                   'base64.RawURLEncoding.EncodeToString(signer.PublicKey)', 'var b [24]byte'):
                    self.assertIn(expression, source)
            evidence = (gateway / "evidence" / "receipt.go").read_text()
            self.assertIn('generateEvidenceRequestID("receipt")', evidence)
            self.assertIn('var b [16]byte', evidence)
            self.assertIn('fmt.Sprintf("%s-%s", subject, base64.RawURLEncoding.EncodeToString(b[:]))', evidence)
            proxy = (gateway / "proxy" / "proxy.go").read_text()
            for expression in ('var b [16]byte', 'subject = "req"',
                               'bindDecisionRequestID(r, "budget")',
                               'generateBudgetRequestID("degradation")',
                               'bindAndStampDecisionRequestID(w, r, "observe")',
                               'subject := "policy"',
                               'bindDecisionRequestID(r, principal.BudgetSubjectID)'):
                self.assertIn(expression, proxy)
            receipt = (gateway / "proxy" / "receipt.go").read_text()
            self.assertIn('"quote_" + strings.TrimPrefix(evidenceAccessID, "evid_")', receipt)
            mint = (gateway / "mint" / "mint.go").read_text()
            self.assertIn('budgetID := fmt.Sprintf("mint-%s", sha256Sum(', mint)
            self.assertIn('budgetID = budgetID[:64]', mint)
            delegation = (gateway / "cloud" / "delegation" / "delegation_v2.go").read_text()
            self.assertIn('b := make([]byte, 16)', delegation)
            self.assertIn('return hex.EncodeToString(b)', delegation)
            cloud = (gateway / "cloud" / "cloud.go").read_text()
            for expression in ('b[6] = (b[6] & 0x0f) | 0x40',
                               'b[8] = (b[8] & 0x3f) | 0x80',
                               '"budget_id = "+tokenID'):
                self.assertIn(expression, cloud)
            cursor = (gateway / "cloud" / "evidence_packs_api.go").read_text()
            self.assertIn('base64.RawURLEncoding.EncodeToString(append(nonce, sealed...))', cursor)


def new_shape_contract(prefix, length, location):
    def test(self):
        for marker in ("stk_Q", "sgcap_Q"):
            for offset in (0, length // 2, length - len(marker)):
                body = generated("", length, marker, offset)
                value = prefix + body
                with self.subTest(marker_size=len(marker), offset=offset):
                    self.assert_pack(value, False, location)
                    self.assert_pack("https://issuer.example/" + value + "?v=1", False, location)
                    # Same random body, but no audited subject.
                    self.assert_pack("caller-" + body, True, location)
            # Shorten padding, not the marker itself. All 64 continuations on
            # BOTH sides, and short/long bodies, must still be scanned.
            value = generated(prefix, length, marker, 1)
            variants = (value[:-1], value + "B") + tuple(c + value for c in ALPHABET) + tuple(value + c for c in ALPHABET)
            for index, variant in enumerate(variants):
                with self.subTest(marker_size=len(marker), variant=index):
                    self.assert_pack(variant, True, location)
            # A credential adjacent to a generated value is never erased.
            for separator in ("", "/", ".", "~", " ", "\n", "\t", "\r"):
                self.assert_pack(value + separator + marker, True, location)
                self.assert_pack(marker + separator + value, True, location)
    return test


for _name, _prefix, _length, _location in NEW_SHAPES:
    setattr(SecretBoundary, "test_round3_shape_" + _name,
            new_shape_contract(_prefix, _length, _location))


if __name__ == "__main__":
    unittest.main()
