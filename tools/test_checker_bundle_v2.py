"""Exercise the versioned v2 download outside the repository."""
import json
import os
from pathlib import Path
import shlex
import shutil
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch

TOOLS = Path(__file__).resolve().parent
BUNDLES = TOOLS / 'checker-bundles'
FILES = {
    'verify_evidence_pack.py', 'verify_receipt_record.py', 'receipt_bundle_v2.py',
    'receipt_observation_v2.py', 'receipt_decision_v2.py',
    'requirements-verify-evidence-pack.txt', 'VERIFIER-CONTRACT.md', 'RECEIPT-V2-CONTRACT.md',
    'SHA256SUMS',
}

# Run the same script and arguments as python3, recording a network attempt even
# if verification code were to catch the audit-hook exception.
GUARD = '''\
import os
from pathlib import Path
import runpy
import sys

def audit(event, _args):
    if event in ('socket.connect', 'socket.getaddrinfo', 'socket.sendto', 'socket.sendmsg'):
        Path(os.environ['SOCKET_ATTEMPT']).touch()
        raise AssertionError('network_forbidden')

sys.addaudithook(audit)
sys.argv.pop(0)
sys.path[0] = str(Path(sys.argv[0]).parent)
runpy.run_path(sys.argv[0], run_name='__main__')
'''


class CheckerBundleV2(unittest.TestCase):
    def setUp(self):
        with patch.object(sys, 'path', [str(TOOLS), *sys.path]):
            from tests.generate_bundle_v2 import NOW, seed_bundle
        bundle, pins, _ = seed_bundle()
        self.now = NOW
        self.tmp = Path(self.enterContext(tempfile.TemporaryDirectory())).resolve()
        self.work = self.tmp / 'work'
        self.work.mkdir()
        self.artifact = self.work / 'bundle.json'
        self.artifact.write_text(json.dumps(bundle))
        self.pins = self.work / 'pins.json'
        self.pins.write_text(json.dumps(pins))
        self.attempt = self.tmp / 'socket-attempt'
        guard = self.tmp / 'guard.py'
        guard.write_text(GUARD)
        self.python = self.tmp / 'python3'
        self.python.write_text(f'#!/bin/sh\nexec {shlex.quote(sys.executable)} -B {shlex.quote(str(guard))} "$@"\n')
        self.python.chmod(0o755)
        self.env = dict(os.environ, PATH=str(self.tmp) + os.pathsep + os.environ['PATH'],
                        SOCKET_ATTEMPT=str(self.attempt))
        self.env.pop('PYTHONPATH', None)

    def copy_checker(self, version):
        source = BUNDLES / version
        self.assertTrue(source.is_dir(), version)
        download = shutil.copytree(source, self.tmp / version)
        return download / 'verify_evidence_pack.py'

    def invoke(self, argv, **env):
        proc = subprocess.run(argv, cwd=self.work, env=dict(self.env, **env),
                              text=True, capture_output=True, timeout=30)
        self.assertFalse(self.attempt.exists(), 'network_forbidden')
        return proc

    def direct(self, script):
        return self.invoke([self.python, script, self.artifact, '--jwks-file', self.pins,
                            '--require-trusted-issuer', '--now', self.now])

    def test_download_matches_tools_sources(self):
        download = (BUNDLES / 'receipt-bundle-v2')
        self.assertTrue(download.is_dir(), download.name)
        self.assertEqual({p.name for p in download.iterdir()}, FILES)
        for name in FILES - {'SHA256SUMS'}:
            self.assertEqual((download / name).read_bytes(), (TOOLS / name).read_bytes(), name)

    def test_isolated_pack_and_cli_wrapper(self):
        script = self.copy_checker('receipt-bundle-v2')
        direct = self.direct(script)
        self.assertEqual(direct.returncode, 0, direct.stderr or direct.stdout)
        verdict = json.loads(direct.stdout)
        self.assertTrue(verdict['valid'], verdict)
        self.assertEqual(verdict['status'], 'verified')
        self.assertEqual(verdict['reason_codes'], ['v2_bundle_verified'])

        binary = self.tmp / 'satgate-cli'
        build = subprocess.run(['go', 'build', '-o', binary, './cmd/satgate-cli'],
                               cwd=TOOLS.parent, text=True, capture_output=True, timeout=180)
        self.assertEqual(build.returncode, 0, build.stderr)
        wrapped = self.invoke([binary, 'agent', 'verify', self.artifact],
                              SATGATE_VERIFIER=str(script), SATGATE_JWKS=str(self.pins))
        self.assertEqual(wrapped.returncode, 0, wrapped.stderr or wrapped.stdout)
        self.assertEqual(json.loads(wrapped.stdout), verdict)

        # Historical keys inside the signed bundle are not a source of trust.
        self.pins.write_text('{"keys": []}')
        untrusted = self.direct(script)
        self.assertEqual(untrusted.returncode, 1, untrusted.stderr)
        self.assertFalse(json.loads(untrusted.stdout)['valid'])

    def test_isolated_frozen_checker_rejects_v2(self):
        proc = self.direct(self.copy_checker('unavailable-check-v1'))
        self.assertEqual(proc.returncode, 1, proc.stderr)
        self.assertFalse(json.loads(proc.stdout)['valid'])
