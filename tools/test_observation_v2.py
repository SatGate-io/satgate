"""Execution-observation schema, links, privacy and every maintained wrapper offline."""
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

from tests.generate_observation_v2 import corpus, STATUSES
from test_reader_compatibility import ROOT, PACK_READERS, RECORD_READERS, load_pack


class ObservationV2(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.data = corpus()
        cls.modules = {n: load_pack(PACK_READERS[n], 'observation_' + n.replace('-', '_'))
                       for n in ('main-tools', 'main-demo')}

    def verdict(self, module, case, **kw):
        args = dict(jwks=case.get('jwks', self.data['jwks']), now=self.data['now'],
                    require_trusted_issuer=True, opening=copy.deepcopy(case['opening']),
                    parent=copy.deepcopy(case['parent']), result_opening=copy.deepcopy(case['result_opening']))
        args.update(kw)
        with patch('socket.socket.connect', side_effect=AssertionError('network_forbidden')):
            return module.verify_pack(copy.deepcopy(case['receipt']), **args)

    def test_red_valid_observation(self):
        result = self.verdict(self.modules['main-tools'], self.data['vectors'][0])
        self.assertTrue(result['valid'], result)
        self.assertEqual(result['parent_binding'], 'matched')
        self.assertEqual(result['result_binding'], 'matched')

    def test_generated_vectors(self):
        self.assertEqual(json.loads((ROOT / 'tools/fixtures/reader-v2-observations.json').read_text()), self.data)
        names = [c['name'] for c in self.data['vectors']]
        self.assertEqual(len(names), len(set(names)))

    def test_all_vectors(self):
        for case in self.data['vectors']:
            with self.subTest(vector=case['name']):
                reference = self.verdict(self.modules['main-tools'], case)
                self.assertIs(reference['valid'], case['valid'], reference)
                if case['code']:
                    self.assertIn(case['code'], reference['reason_codes'])
                self.assertEqual(reference['reasons'], reference['reason_codes'])
                for canary in ('dummy-private', 'dummy-account', 'dummy-credential', 'dummy-target'):
                    self.assertNotIn(canary, json.dumps(reference))
                self.assertEqual(self.verdict(self.modules['main-demo'], case), reference)

    def test_statuses_never_upgrade_or_infer(self):
        for status in STATUSES:
            case = next(c for c in self.data['vectors'] if c['name'] == 'status/' + status)
            result = self.verdict(self.modules['main-tools'], case)
            self.assertTrue(result['valid'], result)
            self.assertEqual(result['observation_status'], status)
            self.assertEqual(result['execution_state'], 'observed_' + status)
            self.assertEqual(result['broker_witness'], 'not_evidenced')
            self.assertEqual(result['settlement_state'], 'not_evidenced')
            if status == 'accepted':
                self.assertNotIn('filled', json.dumps(result))
            if status == 'sent_outcome_unknown':
                self.assertNotIn('rejected', json.dumps(result))
                self.assertNotIn('refus', json.dumps(result))

    def test_public_only_is_explicitly_partial(self):
        case = self.data['vectors'][0]
        result = self.verdict(self.modules['main-tools'], case, opening=None, result_opening=None)
        self.assertFalse(result['valid'])
        self.assertTrue(result['checks']['signature_valid'])
        self.assertTrue(result['checks']['parent_signature_valid'])
        self.assertEqual(result['attempt_binding'], 'not_checked')
        self.assertIn('v2_attempt_not_checked', result['reason_codes'])
        result = self.verdict(self.modules['main-tools'], case, result_opening=None)
        self.assertTrue(result['valid'])
        self.assertEqual(result['result_binding'], 'not_checked')
        self.assertNotIn('v2_result_matched', result['reason_codes'])

    def test_no_embedded_fallback_or_network(self):
        for module in self.modules.values():
            result = self.verdict(module, self.data['vectors'][0], jwks=None,
                                  require_trusted_issuer=False, discover_jwks=True)
            self.assertFalse(result['valid'])
            self.assertIn('v2_key_required', result['reason_codes'])

    def test_no_v1_container_or_malformed_private_input_fallback(self):
        case = self.data['vectors'][0]
        for module in self.modules.values():
            wrapped = {'schema_version': 'satgate.evidence_pack.v1', 'receipts': [case['receipt']]}
            result = module.verify_pack(wrapped, jwks=self.data['jwks'], now=self.data['now'])
            self.assertFalse(result['valid'])
            self.assertEqual(result['reason_codes'], ['v2_unsupported_container'])
            for field in ('parent', 'opening', 'result_opening'):
                for value in (False, 1, [], 'dummy-private-canary', {}):
                    with self.subTest(field=field, value=value):
                        result = self.verdict(module, case, **{field: value})
                        self.assertFalse(result['valid'])
                        self.assertEqual(result['reasons'], result['reason_codes'])
                        self.assertNotIn('dummy-private-canary', json.dumps(result))

    def test_every_cli_vector(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            for case in self.data['vectors']:
                args = ['--now', self.data['now']]
                for field, option in (('jwks', '--jwks-file'), ('parent', '--parent-file'),
                                      ('opening', '--opening-file'), ('result_opening', '--result-opening-file')):
                    value = case.get(field, self.data['jwks'] if field == 'jwks' else None)
                    if value is not None:
                        path = root / (field + '.json')
                        path.write_text(json.dumps(value))
                        args += [option, str(path)]
                reference = self.verdict(self.modules['main-tools'], case)
                for name in self.modules:
                    with self.subTest(reader=name, vector=case['name']):
                        process = subprocess.run([sys.executable, '-B', str(PACK_READERS[name]), '-', *args],
                            input=json.dumps(case['receipt']), text=True, capture_output=True, timeout=30)
                        self.assertEqual(process.returncode, 0 if case['valid'] else 1, process.stderr)
                        self.assertEqual(json.loads(process.stdout), reference)

    def test_actual_sdk_and_go_wrappers(self):
        sys.path.insert(0, str(ROOT / 'sdk/python'))
        from satgate.agent_client import SatGateAgentClient
        from satgate.exceptions import SatGateError
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            binary = root / 'satgate-cli'
            built = subprocess.run(['go', 'build', '-o', str(binary), './cmd/satgate-cli'],
                cwd=ROOT, capture_output=True, text=True, timeout=180)
            self.assertEqual(built.returncode, 0, built.stderr)
            client = SatGateAgentClient(gateway_url='https://gateway.example', admin_token=None)
            env = dict(os.environ, PATH=str(Path(sys.executable).parent) + os.pathsep + os.environ['PATH'],
                       PYTHONDONTWRITEBYTECODE='1', SATGATE_JWKS=str(root / 'jwks.json'))
            env.pop('SATGATE_VERIFIER', None)
            with patch.dict(os.environ, SATGATE_VERIFIER=str(PACK_READERS['main-tools'])), \
                 patch('requests.sessions.Session.request', side_effect=AssertionError('network_forbidden')):
                for case in self.data['vectors']:
                    keys = case.get('jwks', self.data['jwks'])
                    reference = self.verdict(self.modules['main-tools'], case)
                    with self.subTest(reader='sdk', vector=case['name']):
                        try:
                            actual = client.verify_pack(case['receipt'], keys, opening=case['opening'],
                                parent=case['parent'], result_opening=case['result_opening'])
                        except SatGateError as error:
                            actual = json.loads(str(error))
                        self.assertEqual(actual, reference)
                    (root / 'jwks.json').write_text(json.dumps(keys))
                    record = root / 'record.json'
                    record.write_text(json.dumps(case['receipt']))
                    args = [str(binary), 'agent', 'verify', str(record)]
                    for field, option in (('parent', '--parent-file'), ('opening', '--opening-file'),
                                          ('result_opening', '--result-opening-file')):
                        if case[field] is not None:
                            path = root / (field + '.json')
                            path.write_text(json.dumps(case[field]))
                            args += [option, str(path)]
                    with self.subTest(reader='go', vector=case['name']):
                        process = subprocess.run(args, cwd=ROOT, env=env, text=True, capture_output=True, timeout=30)
                        self.assertEqual(process.returncode, 0 if case['valid'] else 1, process.stdout + process.stderr)
                        self.assertEqual(json.loads(process.stdout), reference)

    def test_historical_record_and_linked_boundaries(self):
        sys.path.insert(0, str(ROOT / 'tools/fixtures/linked-verifier-d62beb34/src'))
        import requests
        from evidence_pack_verifier import verifier
        with tempfile.TemporaryDirectory() as directory:
            key = Path(directory) / 'jwks.json'
            key.write_text(json.dumps(self.data['jwks']))
            for case in self.data['vectors']:
                for name, script in RECORD_READERS.items():
                    with self.subTest(reader='record-' + name, vector=case['name']):
                        process = subprocess.run([sys.executable, '-B', str(script), '-', '--jwks-file', str(key)],
                            input=json.dumps(case['receipt']), text=True, capture_output=True, timeout=30)
                        self.assertEqual(process.returncode, 1)
                        self.assertEqual(json.loads(process.stdout)['status'], 'invalid')
                response = requests.Response()
                response.status_code = 200
                response._content = json.dumps(case['receipt']).encode()
                with patch.object(verifier.requests, 'get', return_value=response), \
                     patch('socket.socket.connect', side_effect=AssertionError('network_forbidden')):
                    result = verifier.verify_evidence_pack('https://artifact.example/observation').to_dict()
                self.assertFalse(result['valid'])
                self.assertIn('missing_receipt', result['reason_codes'])

    def test_private_cli_errors_and_duplicates_are_codes_only(self):
        case = self.data['vectors'][0]
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            for field in ('parent', 'opening', 'result_opening'):
                (root / (field + '.json')).write_text(json.dumps(case[field]))
            key = root / 'jwks.json'
            key.write_text(json.dumps(self.data['jwks']))
            args = ['--jwks-file', str(key), '--parent-file', str(root / 'parent.json'),
                    '--opening-file', str(root / 'opening.json'),
                    '--result-opening-file', str(root / 'result_opening.json')]
            for module in self.modules.values():
                for raw in (json.dumps(case['receipt']).replace('"record_type": "execution_observation"',
                              '"record_type":"execution_observation","record_type":"execution_observation"'),
                            '{"dummy-private-canary":'):
                    output = io.StringIO()
                    with patch('sys.stdin', io.StringIO(raw)), redirect_stdout(output):
                        code = module.main(['-', *args])
                    self.assertEqual(code, 1)
                    result = json.loads(output.getvalue())
                    self.assertEqual(result['reason_codes'], result['reasons'])
                    self.assertNotIn('dummy-private', output.getvalue())
                for option in ('--parent-file', '--result-opening-file'):
                    for source, extras in (('https://artifact.example/private', []),
                                           ('-', ['--discover-jwks']),
                                           ('-', ['--jwks-url', 'https://issuer.example/keys'])):
                        output = io.StringIO()
                        with patch('urllib.request.urlopen', side_effect=AssertionError('network_forbidden')), \
                             redirect_stdout(output):
                            code = module.main([source, option, 'dummy-missing', *extras])
                        self.assertEqual(code, 1)
                        self.assertEqual(json.loads(output.getvalue())['reasons'], ['v2_input_invalid'])
                for field in ('parent', 'opening', 'result_opening'):
                    path = root / (field + '.json')
                    original = json.dumps(case[field])
                    for raw in ('null', '{"dummy-private-canary":',
                                original.replace('{', '{"schema_version":"dummy-duplicate",', 1),
                                original.replace('{', '{"SCHEMA_VERSION":"dummy-collision",', 1)):
                        path.write_text(raw)
                        output = io.StringIO()
                        with patch('sys.stdin', io.StringIO(json.dumps(case['receipt']))), redirect_stdout(output):
                            code = module.main(['-', *args])
                        self.assertEqual(code, 1)
                        result = json.loads(output.getvalue())
                        self.assertFalse(result['valid'])
                        self.assertEqual(result['reasons'], result['reason_codes'])
                        self.assertNotIn('dummy-private-canary', output.getvalue())
                    path.write_text(original)


if __name__ == '__main__':
    unittest.main()
