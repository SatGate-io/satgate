"""Private bundles: closed manifests, exact bytes and offline verdicts."""
import json
from contextlib import redirect_stdout
import io
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch

from test_reader_compatibility import PACK_READERS, RECORD_READERS, load_pack
from tests.generate_bundle_v2 import corpus, seed_bundle


class BundleV2(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.module = load_pack(PACK_READERS['main-tools'], 'bundle_reference')
        cls.data = corpus()

    def test_red_mixed_decisions_and_results(self):
        bundle, pins, _ = seed_bundle()
        result = self.module.verify_pack(bundle, jwks=pins, now=self.data['now'])
        self.assertTrue(result['valid'], result)
        self.assertEqual(result['status'], 'verified')

    def test_reference_vectors(self):
        with patch('socket.socket.connect', side_effect=AssertionError('network_forbidden')):
            for case in self.data['vectors']:
                with self.subTest(vector=case['name']):
                    result = self.module.verify_pack(case['bundle'], jwks=case['jwks'], now=self.data['now'])
                    self.assertEqual(result['status'], case['status'], result)
                    self.assertIs(result['valid'], case['status'] == 'verified')
                    self.assertIs(result['complete'], result['valid'])
                    if case['code'] is not None:
                        self.assertIn(case['code'], result['reason_codes'])
                    self.assertEqual(result['reasons'], result['reason_codes'])
                    rendered = json.dumps(result)
                    for forbidden in ('dummy-', 'https://', 'sha256:', 'observation_status', 'financial_values',
                                      'broker_order_id', 'accepted', 'filled', 'amount', 'order'):
                        self.assertNotIn(forbidden, rendered)

    def test_cli_duplicate_top_nested_and_case_colliding_names(self):
        bundle, _, _ = seed_bundle()
        data = json.dumps(bundle)
        for raw in (data.replace('{', '{"schema_version":"satgate.evidence_pack.v1",', 1),
                    data.replace('"retention_state":', '"retention_state":"retained","retention_state":'),
                    data.replace('"retention_state":', '"Retention_State":"retained","retention_state":')):
            with self.subTest(raw=raw[:100]):
                output = io.StringIO()
                with patch('sys.stdin', io.StringIO(raw)), redirect_stdout(output):
                    code = self.module.main(['-'])
                self.assertEqual(code, 1)
                self.assertEqual(json.loads(output.getvalue())['reasons'], ['v2_input_invalid'])

    def test_explicit_private_mode_and_local_pins_only(self):
        bundle, pins, _ = seed_bundle()
        text = json.dumps(bundle)
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / 'pins.json'
            path.write_text(json.dumps(pins))
            for script in (PACK_READERS['main-tools'], PACK_READERS['main-demo'], RECORD_READERS['main-tools']):
                for source, raw, extra, reason in (
                    ('-', text, [], 'v2_bundle_verified'),
                    ('-', '{"dummy-private":', [], 'v2_input_invalid'),
                    ('http://127.0.0.1:1/private', '', [], 'v2_input_invalid'),
                    ('-', text, ['--jwks-url', 'http://127.0.0.1:1/keys'], 'v2_input_invalid'),
                ):
                    with self.subTest(script=script.name, reason=reason):
                        proc = subprocess.run([sys.executable, '-B', str(script), source, '--private-bundle',
                                               '--jwks-file', str(path), '--now', self.data['now'], *extra],
                                              input=raw, text=True, capture_output=True, timeout=10)
                        self.assertEqual(proc.returncode, 0 if reason == 'v2_bundle_verified' else 1)
                        self.assertEqual(proc.stderr, '')
                        self.assertEqual(json.loads(proc.stdout)['reasons'], [reason])
                # Auto-dispatch must not consult explicit remote key locations, either.
                proc = subprocess.run([sys.executable, '-B', str(script), '-', '--jwks-url', 'http://127.0.0.1:1/keys'],
                                      input=text, text=True, capture_output=True, timeout=10)
                self.assertEqual(proc.returncode, 1)
                self.assertEqual(json.loads(proc.stdout)['reasons'], ['v2_input_invalid'])


if __name__ == '__main__':
    unittest.main()
