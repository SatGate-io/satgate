"""Exercise the exact versioned download, isolated from repository imports."""
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import unittest

import test_paid_rail as paid
import test_receipt_record as records
import test_unavailable_check_receipts as unavailable

ROOT = Path(__file__).resolve().parents[1]
BUNDLE = ROOT / "tools" / "checker-bundles" / "unavailable-check-v1"
FILES = {"verify_receipt_record.py", "verify_evidence_pack.py",
         "requirements-verify-evidence-pack.txt", "VERIFIER-CONTRACT.md"}


class CheckerBundle(unittest.TestCase):
    def test_hashes_and_matching_sources(self):
        entries = dict(line.split("  ", 1)[::-1]
                       for line in (BUNDLE / "SHA256SUMS").read_text().splitlines())
        self.assertEqual(set(entries), FILES)
        for name, digest in entries.items():
            data = (BUNDLE / name).read_bytes()
            self.assertEqual(hashlib.sha256(data).hexdigest(), digest)
            # The released contract is immutable. Producer documentation may
            # advance after the reader-only release; executable/requirements
            # bytes must still match the released checker pair exactly.
            if name != "VERIFIER-CONTRACT.md":
                self.assertEqual(data, (ROOT / "tools" / name).read_bytes())

    def test_isolated_bundle_pack_and_record_cli_old_and_new_receipts(self):
        f = paid.Fixture()
        b = records.Builder(f)
        packs = [b.allowed(10), b.refused(20)]
        packs += [b.pack(b.base(unavailable.refusal(f, reason), 30 + index))
                  for index, reason in enumerate(unavailable.REASONS)]
        with tempfile.TemporaryDirectory() as td:
            directory = Path(td)
            for name in FILES:
                shutil.copyfile(BUNDLE / name, directory / name)
            jwks = directory / "public-jwks.json"
            jwks.write_text(json.dumps(f.jwks))
            env = dict(os.environ, PYTHONDONTWRITEBYTECODE="1")
            env.pop("PYTHONPATH", None)
            # The isolated directory contains only the downloaded bundle and
            # explicit test-public JWKS, never repository Python imports.
            for script, value in [*(('verify_evidence_pack.py', json.dumps(p)) for p in packs),
                                  ('verify_receipt_record.py', records.text(packs, b.manifest(packs)))]:
                proc = subprocess.run(
                    [sys.executable, "-B", str(directory / script), "-", "--now", paid.NOW,
                     "--jwks-file", str(jwks), "--require-trusted-issuer"],
                    cwd=directory, input=value, text=True, capture_output=True, env=env, timeout=30)
                self.assertEqual(proc.returncode, 0, proc.stderr)
                result = json.loads(proc.stdout)
                self.assertTrue(result["valid"])
                self.assertTrue(result.get("trusted_issuer_valid", result.get("checks", {}).get("trusted_issuer_valid")))


if __name__ == "__main__":
    unittest.main()
