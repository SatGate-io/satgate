"""Secret-marker contract for every shipped checker (synthetic values only).

Run with unittest discover. Set VERIFY_SECRET_BOUNDARY_PEER_ROOT to the other
repository to assert all four copies together; CI needs only its own checkout.
This file is identical in the public tools/ and enterprise tests/ directories.
"""
import importlib.util
import os
from pathlib import Path
import re
import string
import unittest

ROOT = Path(__file__).resolve().parents[1]
BOUNDARY = r"(?:(?<![A-Za-z0-9._~-])|(?<=%[0-9A-Fa-f]{2}))"
MARKERS = ("stk_fixture_not_a_credential", "sgcap_fixture_not_a_credential",
           "Bearer fixture_not_a_credential", "bEaReR fixture_not_a_credential")
ENCODED_STARTS = ("Bearer%20", "token%3D", "user%3A", "path%2F", "Bearer+",
                  "bEaReR+", "+", "token+", "token%3d", "user%3a", "path%2f")
EXPECTED = [
    (BOUNDARY + r"Bearer\s+[A-Za-z0-9._~+/=-]+", re.IGNORECASE | re.UNICODE),
    (BOUNDARY + r"sgcap_[A-Za-z0-9._~+/=-]+", re.UNICODE),
    (BOUNDARY + r"stk_[A-Za-z0-9._~+/=-]+", re.UNICODE),
]


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


class SecretBoundary(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.checkers = load_checkers()

    def assert_markers(self, value, expected):
        verdicts = [checker.contains_secret_marker(value) for checker in self.checkers]
        self.assertEqual(verdicts, [expected] * len(self.checkers),
                         "secret-marker verdicts in every copy")

    def test_all_copies_agree_on_patterns_and_flags(self):
        actual = [[(p.pattern, p.flags) for p in checker.SECRET_PATTERNS]
                  for checker in self.checkers]
        self.assertEqual(actual, [EXPECTED] * len(self.checkers))
        blocks = []
        for checker in self.checkers:
            source = Path(checker.__file__).read_text()
            blocks.append(source.split("# A marker cannot start", 1)[1].split("\n]", 1)[0])
        self.assertEqual(blocks, [blocks[0]] * len(blocks))

    def test_stk_inside_an_access_id_passes(self):
        access_id = "evid_" + "A" * 12 + "stk_" + "B" * 16
        self.assert_markers({"evidence_access_id": access_id,
                             "evidence_url": "https://issuer.example/v1/evidence/" + access_id}, False)

    def test_sgcap_inside_an_access_id_passes(self):
        access_id = "evid_" + "A" * 12 + "sgcap_" + "B" * 14
        self.assert_markers({"evidence_access_id": access_id,
                             "evidence_url": "https://issuer.example/v1/evidence/" + access_id}, False)

    def test_bearer_inside_an_id_or_text_passes(self):
        self.assert_markers({"request_id": "evid_AAAABearerBBBB"}, False)
        # Space is needed to trigger the old Bearer pattern. This is diagnostic
        # text, not an access ID (base64url IDs cannot contain whitespace).
        self.assert_markers({"note": "evid_AAAABearer fixture_not_a_credential"}, False)

    def test_token_alphabet_continuations_are_not_starts(self):
        # '+' is not in generated pack values; it is a form-encoded space.
        continuations = string.ascii_letters + string.digits + "._~-"
        for char in continuations:
            for marker in ("stk_fixture", "sgcap_fixture", "Bearer fixture"):
                self.assert_markers({"note": char + marker}, False)

    def test_encoded_token_starts_still_fail(self):
        for before in ENCODED_STARTS:
            for marker in MARKERS:
                self.assert_markers(before + marker, True)
                self.assert_markers({"nested": [{"note": before + marker}]}, True)
                for checker in self.checkers:
                    result = checker.verify_pack({"note": before + marker})
                    self.assertFalse(result["checks"]["no_secret_leakage"])
                    self.assertFalse(result["valid"])

    def test_every_percent_escape_is_a_start(self):
        for byte in range(256):
            for escape in ("%%%02X" % byte, "%%%02x" % byte):
                for marker in MARKERS:
                    self.assert_markers({"note": "value" + escape + marker}, True)

    def test_incomplete_or_nonhex_escapes_do_not_override_continuations(self):
        for before in ("%2", "%GG", "%2G", "%G2", "%123", "%2FA"):
            for marker in MARKERS:
                self.assert_markers({"note": before + marker}, False)

    def test_generated_pack_alphabets_exclude_encoded_delimiters(self):
        # Exercise the shipped Python encoder over every byte, not a random sample.
        alphabet = set(string.ascii_letters + string.digits + "-_")
        for checker in self.checkers:
            encoded = checker.b64url_encode(bytes(range(256)))
            self.assertEqual(set(encoded), alphabet)
            self.assertFalse(set(encoded) & set("%+"))
            for length in (16, 24, 32, 64):
                value = checker.b64url_encode(bytes(range(length)))
                self.assertLessEqual(set(value), alphabet)
        # Go generators cannot be imported by Python. Pin their actual encoder
        # calls as source contracts without building the gateway's heavy packages.
        roots = {ROOT}
        peer = os.environ.get("VERIFY_SECRET_BOUNDARY_PEER_ROOT")
        if peer:
            roots.add(Path(peer).resolve())
        for root in roots:
            gateway = root / "gateway" / "internal"
            if not gateway.is_dir():
                continue  # Public CI has no enterprise checkout.
            for surface in ("evidence", "proxy"):
                source = (gateway / surface / "receipt.go").read_text()
                for expression in (
                    '"evid_" + base64.RawURLEncoding.EncodeToString(b[:])',
                    '"rcpt_" + base64.RawURLEncoding.EncodeToString(receiptIDHash[:])[:22]',
                    '"ep_" + base64.RawURLEncoding.EncodeToString(evidencePackHash[:])[:22]',
                    '"sha256:" + base64.RawURLEncoding.EncodeToString(hash[:])',
                    '"ed25519:" + base64.RawURLEncoding.EncodeToString(signature)',
                    'base64.RawURLEncoding.EncodeToString(signer.PublicKey)',
                ):
                    self.assertIn(expression, source)
                self.assertNotRegex(source, r'base64\.(?:Raw)?StdEncoding\.EncodeToString')
            cursor = (gateway / "cloud" / "evidence_packs_api.go").read_text()
            self.assertIn('base64.RawURLEncoding.EncodeToString(append(nonce, sealed...))', cursor)
            nonce = (gateway / "mcpapproval" / "link.go").read_text()
            self.assertIn('return hex.EncodeToString(b[:]), nil', nonce)
        self.assertFalse(set(string.hexdigits + "-:") & set("%+"))

    def test_generated_value_marker_embeddings_still_pass(self):
        # Include all base64url predecessors and the fixed generated ID prefixes.
        for before in string.ascii_letters + string.digits + "_-":
            for marker in ("stk_fixture", "sgcap_fixture", "Bearerfixture"):
                for prefix in ("evid_", "rcpt_", "ep_", "sha256:", "ed25519:", ""):
                    self.assert_markers({"generated": prefix + before + marker}, False)

    def test_real_token_starts_still_fail(self):
        for marker in ("stk_fixture_not_a_credential", "sgcap_fixture_not_a_credential",
                       "Bearer fixture_not_a_credential", "bEaReR fixture_not_a_credential"):
            # A slash and equals are deliberately delimiters: preserve URL path
            # segments and query/assignment values, not just standalone values.
            for before in ("", " ", "=", ":", "(", "https://issuer.example/",
                           "https://issuer.example/?value=", "Authorization: "):
                self.assert_markers(before + marker, True)
                self.assert_markers({"note": before + marker}, True)
                self.assert_markers({"nested": [{"note": before + marker}]}, True)
                # Full verifier verdict must still reject a leaked marker.
                for checker in self.checkers:
                    result = checker.verify_pack({"note": before + marker})
                    self.assertFalse(result["valid"])
                    self.assertFalse(result["checks"]["no_secret_leakage"])
        # Exercise actual raw header whitespace, not JSON-escaped whitespace.
        for checker in self.checkers:
            self.assertTrue(checker.SECRET_PATTERNS[0].search("Authorization: Bearer\tfixture"))

    def test_sensitive_field_detection_is_unchanged(self):
        for field in ("raw_capability_token", "authorization_header", "payment_secret"):
            self.assert_markers({field: "fixture_without_a_prefix"}, True)
            self.assert_markers({field: "redacted"}, False)


if __name__ == "__main__":
    unittest.main()
