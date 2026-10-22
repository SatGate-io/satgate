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
BOUNDARY = r"(?<![A-Za-z0-9._~+-])"
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
        continuations = string.ascii_letters + string.digits + "._~+-"
        for char in continuations:
            for marker in ("stk_fixture", "sgcap_fixture", "Bearer fixture"):
                self.assert_markers({"note": char + marker}, False)

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
