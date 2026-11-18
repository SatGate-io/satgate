"""Offline v2 contract vectors against both maintained public readers."""
import copy
from contextlib import redirect_stdout
import io
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch

from tests.generate_decision_v2 import corpus, commitment, digest
from test_reader_compatibility import PACK_READERS, RECORD_READERS, load_pack

ROOT = Path(__file__).resolve().parents[1]


class DecisionV2(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.data = corpus()
        cls.modules = {n: load_pack(PACK_READERS[n], 'v2_' + n.replace('-', '_'))
                       for n in ('main-tools', 'main-demo')}

    def verdict(self, module, case, **kw):
        args = dict(jwks=case.get('jwks', self.data['jwks']), require_trusted_issuer=True,
                    now=self.data['now'])
        if case['opening'] is not None:
            args['opening'] = copy.deepcopy(case['opening'])
        args.update(kw)
        with patch('socket.socket.connect', side_effect=AssertionError('network_forbidden')):
            return module.verify_pack(copy.deepcopy(case['receipt']), **args)

    def test_red_valid_v2_opening(self):
        result = self.verdict(self.modules['main-tools'], self.data['vectors'][0])
        self.assertTrue(result['valid'], result)
        self.assertEqual(result['call_binding'], 'matched')

    def test_fixed_generated_vectors(self):
        self.assertEqual(json.loads((ROOT / 'tools/fixtures/reader-v2-decisions.json').read_text()), self.data)

    def test_every_signed_leaf_is_authenticated(self):
        original = self.data['vectors'][0]

        def leaves(value, path=()):
            if isinstance(value, dict):
                for key, child in value.items():
                    if not path and key in ('signature', 'receipt_hash'):
                        continue
                    yield from leaves(child, path + (key,))
            else:
                yield path

        for path in leaves(original['receipt']):
            case = copy.deepcopy(original)
            target = case['receipt']
            for key in path[:-1]:
                target = target[key]
            old = target[path[-1]]
            target[path[-1]] = 'dummy-tampered' if old is None else str(old) + '-tampered'
            # Update the unsigned hash mirror only, never its signature.
            case['receipt']['receipt_hash'] = digest({k: v for k, v in case['receipt'].items()
                                                     if k not in ('receipt_hash', 'signature')})
            for module in self.modules.values():
                with self.subTest(path=path):
                    self.assertFalse(self.verdict(module, case)['valid'])

    def test_all_vectors_and_reader_parity(self):
        for case in self.data['vectors']:
            reference = self.verdict(self.modules['main-tools'], case)
            with self.subTest(vector=case['name']):
                self.assertIs(reference['valid'], case['valid'], reference)
                if case['code']:
                    self.assertIn(case['code'], reference['reason_codes'])
                self.assertEqual(reference['reasons'], reference['reason_codes'])
                self.assertNotIn('dummy-private-canary', json.dumps(reference))
                self.assertNotIn('dummy-account', json.dumps(reference))
                self.assertNotIn('dummy-credential', json.dumps(reference))
                self.assertNotIn('executed', reference.get('execution', ''))
                self.assertEqual(self.verdict(self.modules['main-demo'], case), reference)

    def test_unopened_and_held_are_bounded(self):
        unopened = self.verdict(self.modules['main-tools'], self.data['vectors'][1])
        self.assertTrue(unopened['valid'])
        self.assertEqual(unopened['call_binding'], 'not_checked')
        self.assertNotIn('v2_call_matched', unopened['reason_codes'])
        held = self.verdict(self.modules['main-tools'], self.data['vectors'][2])
        self.assertTrue(held['valid'])
        self.assertEqual(held['admission_state'], 'held')
        self.assertEqual(held['execution_state'], 'not_evidenced')

    def test_identical_orders_have_distinct_commitments(self):
        first = self.data['vectors'][0]
        second = next(c for c in self.data['vectors'] if c['name'] == 'valid-identical-order-new-attempt')
        self.assertEqual(first['opening']['transport'], second['opening']['transport'])
        self.assertNotEqual(commitment(first['opening']), commitment(second['opening']))

    def test_local_cli_vectors(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            key = root / 'jwks.json'
            opening = root / 'opening.json'
            for case in self.data['vectors']:
                key.write_text(json.dumps(case.get('jwks', self.data['jwks'])))
                args = ['--jwks-file', str(key), '--require-trusted-issuer', '--now', self.data['now']]
                if case['opening'] is not None:
                    opening.write_text(json.dumps(case['opening']))
                    args += ['--opening-file', str(opening)]
                reference = self.verdict(self.modules['main-tools'], case)
                for name in self.modules:
                    with self.subTest(reader=name, vector=case['name']):
                        p = subprocess.run([sys.executable, '-B', str(PACK_READERS[name]), '-', *args],
                                           input=json.dumps(case['receipt']), text=True,
                                           capture_output=True, timeout=30)
                        self.assertEqual(p.returncode, 0 if case['valid'] else 1, p.stderr + p.stdout)
                        self.assertEqual(json.loads(p.stdout), reference)

    def test_v2_never_enters_v1_record_or_pack(self):
        # The typed v2 bundle is a later row; no reuse of v1 NDJSON semantics.
        for name, module in self.modules.items():
            with self.subTest(reader=name):
                r = copy.deepcopy(self.data['vectors'][0]['receipt'])
                pack = {'schema_version': 'satgate.evidence_pack.v1', 'receipts': [r]}
                result = module.verify_pack(pack, jwks=self.data['jwks'], now=self.data['now'])
                self.assertFalse(result['valid'])

    def test_v2_no_embedded_fallback_or_discovery(self):
        case = self.data['vectors'][0]
        for module in self.modules.values():
            result = self.verdict(module, case, jwks=None, require_trusted_issuer=False,
                                  discover_jwks=True)
            self.assertFalse(result['valid'])
            self.assertIn('v2_key_required', result['reason_codes'])

    def test_actual_local_sdk_and_go_wrappers(self):
        sys.path.insert(0, str(ROOT / 'sdk/python'))
        from satgate.agent_client import SatGateAgentClient
        from satgate.exceptions import SatGateError
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            binary = root / 'satgate-cli'
            built = subprocess.run(['go', 'build', '-o', str(binary), './cmd/satgate-cli'],
                                   cwd=ROOT, capture_output=True, text=True, timeout=180)
            self.assertEqual(built.returncode, 0, built.stderr)
            key = root / 'jwks.json'
            record = root / 'decision.json'
            opening = root / 'opening.json'
            env = dict(os.environ, PATH=str(Path(sys.executable).parent) + os.pathsep + os.environ['PATH'],
                       SATGATE_JWKS=str(key), PYTHONDONTWRITEBYTECODE='1')
            env.pop('SATGATE_VERIFIER', None)
            client = SatGateAgentClient(gateway_url='https://gateway.example', admin_token=None)
            with patch.dict(os.environ, SATGATE_VERIFIER=str(PACK_READERS['main-tools'])), \
                 patch('requests.sessions.Session.request', side_effect=AssertionError('network_forbidden')):
                for case in self.data['vectors']:
                    reference = self.verdict(self.modules['main-tools'], case)
                    keys = case.get('jwks', self.data['jwks'])
                    with self.subTest(reader='sdk', vector=case['name']):
                        try:
                            actual = client.verify_pack(case['receipt'], keys, opening=case['opening'])
                        except SatGateError as error:
                            actual = json.loads(str(error))
                        self.assertEqual(actual, reference)
                    key.write_text(json.dumps(keys))
                    record.write_text(json.dumps(case['receipt']))
                    args = [str(binary), 'agent', 'verify', str(record)]
                    if case['opening'] is not None:
                        opening.write_text(json.dumps(case['opening']))
                        args += ['--opening-file', str(opening)]
                    with self.subTest(reader='go-cli', vector=case['name']):
                        process = subprocess.run(args, cwd=ROOT, env=env, capture_output=True,
                                                 text=True, timeout=30)
                        self.assertEqual(process.returncode, 0 if case['valid'] else 1)
                        self.assertEqual(json.loads(process.stdout), reference)

    def test_raw_json_duplicates_and_private_input_errors(self):
        case = self.data['vectors'][0]
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            key = root / 'jwks.json'
            key.write_text(json.dumps(self.data['jwks']))
            opening = root / 'opening.json'
            for name in self.modules:
                for raw, private in (
                    (json.dumps(case['receipt']).replace('"record_type": "decision"',
                      '"record_type":"decision","record_type":"decision"'), json.dumps(case['opening'])),
                    (json.dumps(case['receipt']), json.dumps(case['opening']).replace(
                        '"attempt_id": "dummy-attempt-a"', '"attempt_id":"a","attempt_id":"b"')),
                    (json.dumps(case['receipt']), 'null'),
                    (json.dumps(case['receipt']), '{"dummy-private-canary":'),
                ):
                    opening.write_text(private)
                    process = subprocess.run([sys.executable, '-B', str(PACK_READERS[name]), '-',
                        '--jwks-file', str(key), '--opening-file', str(opening)],
                        input=raw, capture_output=True, text=True, timeout=30)
                    with self.subTest(reader=name, raw=raw[:30], private=private[:30]):
                        self.assertEqual(process.returncode, 1, process.stderr)
                        result = json.loads(process.stdout)
                        self.assertFalse(result['valid'])
                        self.assertEqual(result['reason_codes'], result['reasons'])
                        self.assertNotIn('dummy-private-canary', process.stdout)

    def test_record_and_linked_boundaries_on_all_vectors(self):
        sys.path.insert(0, str(ROOT / 'tools/fixtures/linked-verifier-d62beb34/src'))
        import requests
        from evidence_pack_verifier import verifier
        with tempfile.TemporaryDirectory() as directory:
            key = Path(directory) / 'jwks.json'
            key.write_text(json.dumps(self.data['jwks']))
            for case in self.data['vectors']:
                for name, script in RECORD_READERS.items():
                    with self.subTest(reader='record-' + name, vector=case['name']):
                        process = subprocess.run([sys.executable, '-B', str(script), '-',
                            '--jwks-file', str(key), '--require-trusted-issuer'],
                            input=json.dumps(case['receipt']), capture_output=True, text=True, timeout=30)
                        self.assertEqual(process.returncode, 1)
                        self.assertEqual(json.loads(process.stdout)['status'], 'invalid')
                response = requests.Response()
                response.status_code = 200
                response._content = json.dumps(case['receipt']).encode()
                with patch.object(verifier.requests, 'get', return_value=response), \
                     patch('socket.socket.connect', side_effect=AssertionError('network_forbidden')):
                    result = verifier.verify_evidence_pack('https://artifact.example/decision').to_dict()
                self.assertFalse(result['valid'])
                self.assertIn('missing_receipt', result['reason_codes'])

    def test_private_cli_never_uses_network_and_prints_codes_only(self):
        for module in self.modules.values():
            for arguments in (
                ['https://artifact.example/dummy', '--opening-file', 'dummy-missing'],
                ['-', '--opening-file', 'dummy-missing', '--jwks-url', 'https://issuer.example/dummy'],
                ['-', '--opening-file', 'dummy-missing', '--discover-jwks'],
                ['dummy-missing', '--opening-file', 'dummy-missing'],
                ['-', '--opening-file', 'dummy-missing'],
            ):
                with self.subTest(args=arguments), \
                     patch('urllib.request.urlopen', side_effect=AssertionError('network_forbidden')), \
                     patch('socket.socket.connect', side_effect=AssertionError('network_forbidden')), \
                     patch('sys.stdin', io.StringIO('{"dummy-private-canary":')):
                    output = io.StringIO()
                    with redirect_stdout(output):
                        code = module.main(arguments)
                    result = json.loads(output.getvalue())
                    self.assertEqual(code, 1)
                    self.assertEqual(result['reasons'], result['reason_codes'])
                    self.assertNotIn('dummy-private-canary', output.getvalue())


if __name__ == '__main__':
    unittest.main()
