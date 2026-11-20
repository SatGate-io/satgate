"""One offline v1 matrix across main, pin, releases, APIs, CLIs and wrappers."""
import copy
from contextlib import redirect_stdout
import hashlib
import importlib.util
import io
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch

import reader_compatibility as vectors

ROOT = Path(__file__).resolve().parents[1]
FIXTURES = ROOT / 'tools/fixtures'
PIN = FIXTURES / 'pinned-checkers-f852c2918fb9'
OLD = FIXTURES / 'released-checkers-20261022'
BUNDLES = ROOT / 'tools/checker-bundles'
GOLDENS = FIXTURES / 'reader-v1-goldens.json'
LINKED = FIXTURES / 'linked-verifier-d62beb34'
LINKED_HASHES = {
    'src/evidence_pack_verifier/__init__.py': '96dcb2170317073668c6bd3c1ae44784c4d8bf112b3cd7c65e01cf7c9ba34881',
    'src/evidence_pack_verifier/cli.py': '1314c0b06515bc95504bdf3dec101476fa8517f9564e354c9f5e615b19200f76',
    'src/evidence_pack_verifier/verifier.py': '831b76ef46951456047641457bff8d677aeea516653fb692aba5c4735bebf63b',
    'LICENSE': '06cd3906766c6eb50384e7549d5e408b571d3f3914f4abd9d5d90d96fa9ba87f',
}
PACK_READERS = {
    'main-tools': ROOT / 'tools/verify_evidence_pack.py',
    'main-demo': ROOT / 'demo/verify_evidence_pack.py',
    'pin-tools': PIN / 'tools/verify_evidence_pack.py',
    'pin-demo': PIN / 'demo/verify_evidence_pack.py',
    'october-tools': OLD / 'tools/verify_evidence_pack.py',
    'october-demo': OLD / 'demo/verify_evidence_pack.py',
    'bundle-tools': BUNDLES / 'unavailable-check-v1/verify_evidence_pack.py',
}
RECORD_READERS = {name: path.with_name('verify_receipt_record.py')
                  for name, path in PACK_READERS.items() if not name.endswith('demo')}
# These digest pins are independent of each release's editable manifest.
IMMUTABLE_BUNDLE_MANIFESTS = {
    'unavailable-check-v1': '96f4c6353961c3f1f9c2e01aa548497d22171592b801b04920281443528d48b0',
}
PIN_HASHES = {
    'tools/verify_evidence_pack.py': '251369698bd2461a65f4624bdc0f0759091c2271b56f8096d5e47e2837b13575',
    'tools/verify_receipt_record.py': '883687f530de0b7216577b037d507410271ea32170c7a5a6b215724e06bbadfa',
    'demo/verify_evidence_pack.py': '55470ac71ca16ce93caea60c40ffc000bcba5240db4513c6079b03ab8ee0d617',
}
# Single named, exact-cell allowlist. Each tuple is readers, vectors, expected
# valid/status, reason. No exception can match an unlisted future vector.
DELIBERATE_V1_DIFFERENCES = (
    (('linked-signature-only',),
     ('unknown-receipt-version', 'unknown-pack-version', 'unknown-reason'), True,
     'Linked v0.1.0 checks only the first receipt hash/signature, not semantic versions or reasons; it is not a full checker.'),
    (('main-demo', 'pin-demo', 'october-demo'),
     ('legacy-unpaired-paid', 'legacy-spend-remaining', 'legacy-missing-issued_at',
      'legacy-missing-evidence_url', 'legacy-missing-authority',
      'legacy-missing-budget', 'legacy-missing-policy'), False,
     'Demo retains paired profiles, provenance and required legacy mirrors; tools keeps legacy enum/optional-mirror rules.'),
    (('october-tools', 'october-demo'),
     tuple('unavailable/' + reason for reason in vectors.unavailable.REASONS), False,
     'Immutable October readers predate the four unavailable-check refusal reasons.'),
    (('october-tools',), ('complete-all-shapes', 'truncated-all-shapes'), 'invalid',
     'The same October pack reader rejects unavailable-check receipts inside records.'),
    (('october-tools',), tuple('pack/unavailable/' + reason for reason in vectors.unavailable.REASONS),
     'invalid', 'October record reader cannot skip unsupported receipt reasons.'),
)


def expected_for(reader, case, reference):
    matches = [value for readers, cases, value, reason in DELIBERATE_V1_DIFFERENCES
               if reader in readers and case in cases]
    if len(matches) > 1:
        raise AssertionError('overlapping deliberate differences')
    return matches[0] if matches else reference


def load_pack(path, name):
    spec = importlib.util.spec_from_file_location(name, path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def pack_verdict(module, case, data, **kw):
    options = dict(jwks=data['jwks'], require_trusted_issuer=True, now=data['now'])
    options.update(kw)
    return module.verify_pack(copy.deepcopy(case['pack']), **options)


class ReaderCompatibility(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.data = json.loads(GOLDENS.read_text())
        cls.modules = {name: load_pack(path, 'matrix_' + name.replace('-', '_'))
                       for name, path in PACK_READERS.items()}
        cls.tmp = tempfile.TemporaryDirectory()
        cls.addClassCleanup(cls.tmp.cleanup)
        cls.directory = Path(cls.tmp.name)
        cls.jwks = cls.directory / 'jwks.json'
        cls.jwks.write_text(json.dumps(cls.data['jwks']))
        cls.env = dict(os.environ, PYTHONDONTWRITEBYTECODE='1',
                       PATH=str(Path(sys.executable).parent) + os.pathsep + os.environ['PATH'])
        cls.env.pop('PYTHONPATH', None)

    def cli(self, script, text, *args):
        proc = subprocess.run([sys.executable, '-B', str(script), '-',
                               '--jwks-file', str(self.jwks), '--require-trusted-issuer',
                               '--now', self.data['now'], *args], input=text,
                              text=True, capture_output=True, cwd=self.directory,
                              env=self.env, timeout=30)
        self.assertNotIn('Traceback', proc.stderr)
        self.assertIn(proc.returncode, (0, 1, 4), proc.stderr)
        return proc.returncode, json.loads(proc.stdout)

    def test_fixed_goldens_and_coverage(self):
        self.assertEqual(self.data, vectors.corpus())
        reasons = {case['pack']['receipts'][0]['decision_reason'] for case in self.data['packs']
                   if case['valid'] and not case['name'].startswith('legacy-')}
        self.assertEqual(reasons, self.modules['main-tools'].ALLOWED_DECISION_REASONS | {'payment_verified'})
        self.assertEqual({c['pack']['decision'] for c in self.data['packs'] if c['valid']},
                         {'allowed', 'paid', 'denied'})
        names = [c['name'] for c in self.data['packs']]
        self.assertEqual(len(names), len(set(names)))
        for readers, cases, value, reason in DELIBERATE_V1_DIFFERENCES:
            self.assertTrue(reason)
            self.assertTrue(set(readers) <= set(PACK_READERS) | set(RECORD_READERS) | {'linked-signature-only'})
            self.assertTrue(set(cases) <= set(names) | {c['name'] for c in self.data['records']})

    def test_reader_discovery_and_immutable_manifests(self):
        # A new distributed reader cannot quietly escape the matrix.
        found = {p.relative_to(ROOT).as_posix() for p in ROOT.rglob('verify_evidence_pack.py')
                 if '.git' not in p.parts}
        self.assertEqual(found, {p.relative_to(ROOT).as_posix() for p in PACK_READERS.values()})
        found_records = {p.relative_to(ROOT).as_posix() for p in ROOT.rglob('verify_receipt_record.py')}
        self.assertEqual(found_records, {p.relative_to(ROOT).as_posix() for p in RECORD_READERS.values()})
        self.assertEqual({p.name for p in BUNDLES.iterdir() if p.is_dir()}, set(IMMUTABLE_BUNDLE_MANIFESTS))
        for bundle, digest in IMMUTABLE_BUNDLE_MANIFESTS.items():
            directory = BUNDLES / bundle
            manifest = directory / 'SHA256SUMS'
            self.assertEqual(hashlib.sha256(manifest.read_bytes()).hexdigest(), digest)
            entries = dict(line.split('  ', 1)[::-1] for line in manifest.read_text().splitlines())
            self.assertEqual({p.name for p in directory.iterdir() if p.is_file()}, set(entries) | {'SHA256SUMS'})
            for name, expected in entries.items():
                self.assertEqual(hashlib.sha256((directory / name).read_bytes()).hexdigest(), expected)
        for directory, commit, hashes in (
            (PIN, 'f852c2918fb9a0640bf40f8f46919f7379d28c35', PIN_HASHES),
            (LINKED, 'd62beb34ffa413f71f62ff56125ed8a3972b075d', LINKED_HASHES),
        ):
            provenance = json.loads((directory / 'PROVENANCE.json').read_text())
            self.assertEqual(provenance['commit'], commit)
            self.assertEqual(provenance['files'], hashes)
            for path, digest in hashes.items():
                self.assertEqual(hashlib.sha256((directory / path).read_bytes()).hexdigest(), digest)
        # Existing demo provenance and October provenance stay authoritative too.
        for entry in json.loads((ROOT / 'demo/SOURCE-MANIFEST.json').read_text())['files']:
            self.assertEqual(hashlib.sha256((ROOT / entry['public_path']).read_bytes()).hexdigest(), entry['sha256'])
        for entry in json.loads((OLD / 'PROVENANCE.json').read_text())['files']:
            self.assertEqual(hashlib.sha256((OLD / entry['frozen_path']).read_bytes()).hexdigest(), entry['sha256'])

    def assert_pack_cell(self, name, case, result, reference):
        expected = expected_for(name, case['name'], reference['valid'])
        self.assertIs(result['valid'], expected, (name, case['name'], result.get('reason_codes')))
        self.assertEqual(result['trusted_issuer_valid'], reference['trusted_issuer_valid'],
                         (name, case['name'], 'trust drift'))
        if not expected and reference['valid']:
            self.assertTrue(result['checks']['signature_valid'], 'exception must not hide broken signatures')

    def test_every_pack_api_and_cli(self):
        for case in self.data['packs']:
            reference = pack_verdict(self.modules['main-tools'], case, self.data)
            self.assertIs(reference['valid'], case['valid'], case['name'])
            for name, module in self.modules.items():
                with self.subTest(reader=name, vector=case['name']):
                    with patch('socket.socket.connect', side_effect=AssertionError('network forbidden')):
                        result = pack_verdict(module, case, self.data)
                    self.assert_pack_cell(name, case, result, reference)
                    code, result = self.cli(PACK_READERS[name], json.dumps(case['pack']))
                    self.assert_pack_cell(name, case, result, reference)
                    self.assertEqual(code, 0 if result['valid'] else 1)

    def test_every_record_cli_with_its_own_sibling(self):
        # Subprocess isolation is essential: record imports verify_evidence_pack
        # by bare name, which otherwise can accidentally use main's cached module.
        for case in self.data['records']:
            code, reference = self.cli(RECORD_READERS['main-tools'], case['text'])
            if case['status'] is not None:
                self.assertEqual(reference['status'], case['status'], (case['name'], reference))
            for name, path in RECORD_READERS.items():
                with self.subTest(reader=name, vector=case['name']):
                    actual_code, result = self.cli(path, case['text'])
                    expected_status = expected_for(name, case['name'], reference['status'])
                    self.assertEqual(result['status'], expected_status, result)
                    self.assertEqual(actual_code, {'complete': 0, 'invalid': 1, 'truncated': 4}[expected_status])
                    self.assertEqual(result['valid'], expected_status == 'complete')
                    self.assertEqual(result['complete'], expected_status == 'complete')
                    if expected_status == reference['status']:
                        self.assertEqual(result['trusted_issuer_valid'], reference['trusted_issuer_valid'])

    def test_actual_sdk_and_go_cli_wrappers(self):
        sys.path.insert(0, str(ROOT / 'sdk/python'))
        from satgate.agent_client import SatGateAgentClient
        from satgate.exceptions import SatGateError
        binary = self.directory / 'satgate-cli'
        build = subprocess.run(['go', 'build', '-o', str(binary), './cmd/satgate-cli'],
                               cwd=ROOT, text=True, capture_output=True, timeout=180)
        self.assertEqual(build.returncode, 0, build.stderr)
        client = SatGateAgentClient(gateway_url='https://gateway.example', admin_token=None)
        with patch.dict(os.environ, SATGATE_VERIFIER=str(PACK_READERS['main-tools'])):
            for case in self.data['packs']:
                reference = pack_verdict(self.modules['main-tools'], case, self.data)
                with self.subTest(reader='sdk', vector=case['name']):
                    if reference['valid']:
                        result = client.verify_pack(case['pack'], self.data['jwks'])
                        self.assert_pack_cell('main-tools', case, result, reference)
                    else:
                        with self.assertRaises(SatGateError):
                            client.verify_pack(case['pack'], self.data['jwks'])
                path = self.directory / 'pack.json'
                path.write_text(json.dumps(case['pack']))
                env = dict(self.env, SATGATE_JWKS=str(self.jwks))
                env.pop('SATGATE_VERIFIER', None)  # Exercise the real default path.
                proc = subprocess.run([str(binary), 'agent', 'verify', str(path)],
                                      cwd=ROOT, env=env, text=True, capture_output=True, timeout=30)
                with self.subTest(reader='go-cli', vector=case['name']):
                    self.assertEqual(proc.returncode, 0 if reference['valid'] else 1, proc.stdout)
                    self.assert_pack_cell('main-tools', case, json.loads(proc.stdout), reference)

    def test_linked_signature_only_api_and_cli(self):
        # This linked package accepts URLs only. Feed the exact signed golden
        # bytes at the HTTP transport boundary; never contact any host. These
        # are test transport responses, not fabricated verification verdicts.
        sys.path.insert(0, str(LINKED / 'src'))
        import requests
        from evidence_pack_verifier import verifier, cli

        for case in self.data['packs']:
            def transport(url, **kwargs):
                if url == 'https://artifact.example/pack':
                    value = case['pack']
                elif url in ('https://issuer.example/.well-known/jwks.json',
                             'https://other.example/.well-known/jwks.json'):
                    value = self.data['jwks']
                else:
                    raise AssertionError('unexpected network target')
                response = requests.Response()
                response.status_code = 200
                response._content = json.dumps(value).encode()
                return response

            reference = pack_verdict(self.modules['main-tools'], case, self.data)
            with self.subTest(reader='linked-signature-only', vector=case['name']):
                with patch.object(verifier.requests, 'get', side_effect=transport), \
                     patch('socket.socket.connect', side_effect=AssertionError('network forbidden')):
                    result = verifier.verify_evidence_pack(
                        'https://artifact.example/pack', discover_jwks=True,
                        require_trusted_issuer=True,
                        trusted_issuers={'https://issuer.example', 'https://other.example'}).to_dict()
                    self.assert_pack_cell('linked-signature-only', case, result, reference)
                    output = io.StringIO()
                    with redirect_stdout(output):
                        code = cli.main(['https://artifact.example/pack', '--discover-jwks',
                                         '--require-trusted-issuer', '--trusted-issuer',
                                         'https://issuer.example', '--trusted-issuer', 'https://other.example'])
                    self.assert_pack_cell('linked-signature-only', case, json.loads(output.getvalue()), reference)
                    self.assertEqual(code, 0 if result['valid'] else 1)

    def test_explicit_trust_modes(self):
        case = self.data['packs'][0]
        for name, module in self.modules.items():
            with self.subTest(reader=name):
                embedded = pack_verdict(module, case, self.data, jwks=None, require_trusted_issuer=False)
                self.assertTrue(embedded['valid'])
                self.assertFalse(embedded['trusted_issuer_valid'])
                self.assertFalse(pack_verdict(module, case, self.data, jwks=None)['valid'])
                wrong = copy.deepcopy(self.data['jwks'])
                wrong['keys'][0]['x'] = vectors.paid.V.b64url_encode(bytes(32))
                self.assertFalse(pack_verdict(module, case, self.data, jwks=wrong)['valid'])
                args = ['--allow-embedded-key'] if name.endswith('demo') else []
                proc = subprocess.run([sys.executable, '-B', str(PACK_READERS[name]), '-',
                                       '--now', self.data['now'], *args],
                                      input=json.dumps(case['pack']), text=True, capture_output=True,
                                      env=self.env, timeout=30)
                self.assertEqual(proc.returncode, 0, proc.stderr)
                self.assertFalse(json.loads(proc.stdout)['trusted_issuer_valid'])

    def test_deliberately_stale_copy_is_detected(self):
        path = self.directory / 'stale-pack-reader.py'
        source = PACK_READERS['main-tools'].read_text()
        needle = '    "auth_missing",'
        self.assertEqual(source.count(needle), 1)
        path.write_text(source.replace(needle, '', 1))
        stale = load_pack(path, 'deliberately_stale_reader')
        case = next(c for c in self.data['packs'] if c['name'] == 'denied/auth_missing')
        reference = pack_verdict(self.modules['main-tools'], case, self.data)
        # Same production matrix assertion, not a separate weaker detector.
        with self.assertRaises(AssertionError):
            self.assert_pack_cell('main-tools', case, pack_verdict(stale, case, self.data), reference)
        if os.environ.get('SATGATE_MATRIX_RED') == '1':
            self.assert_pack_cell('main-tools', case, pack_verdict(stale, case, self.data), reference)

    def test_v1_verdicts_byte_for_byte(self):
        # Pin snapshots are byte-identical to the row-01 base's reader scripts.
        # Compare entire verdicts, not just acceptance booleans.
        for case in self.data['packs']:
            for kind in ('tools', 'demo'):
                for options in ({}, {'require_trusted_issuer': False, 'jwks': None},
                                {'jwks': {'keys': []}}):
                    with self.subTest(reader=kind, vector=case['name'], mode=str(options)):
                        before = pack_verdict(self.modules['pin-' + kind], case, self.data, **options)
                        after = pack_verdict(self.modules['main-' + kind], case, self.data, **options)
                        self.assertEqual(json.dumps(after, sort_keys=True), json.dumps(before, sort_keys=True))
                legacy_values = [case['pack']['receipts'][0], dict(case['pack'], record_type='dummy-v1-extension')]
                for value in legacy_values:
                    if value.get('schema_version') not in ('satgate.receipt.v1', 'satgate.evidence_pack.v1'):
                        continue  # New unknown-version dispatch is not a v1 contract.
                    options = dict(jwks=self.data['jwks'], now=self.data['now'], require_trusted_issuer=True)
                    before = self.modules['pin-' + kind].verify_pack(copy.deepcopy(value), **options)
                    after = self.modules['main-' + kind].verify_pack(copy.deepcopy(value), **options)
                    self.assertEqual(json.dumps(after, sort_keys=True), json.dumps(before, sort_keys=True))

    def test_v2_decision_matrix(self):
        from tests.generate_decision_v2 import corpus
        data = corpus()
        found = {p.relative_to(ROOT).as_posix() for p in ROOT.rglob('receipt_decision_v2.py')}
        self.assertEqual(found, {'tools/receipt_decision_v2.py', 'demo/receipt_decision_v2.py'})
        self.assertEqual((ROOT / 'tools/receipt_decision_v2.py').read_bytes(),
                         (ROOT / 'demo/receipt_decision_v2.py').read_bytes())
        for case in data['vectors']:
            options = dict(jwks=case.get('jwks', data['jwks']), now=data['now'],
                           require_trusted_issuer=True)
            with patch('socket.socket.connect', side_effect=AssertionError('network forbidden')):
                reference = self.modules['main-tools'].verify_pack(
                    case['receipt'], opening=case['opening'], **options)
                self.assertIs(reference['valid'], case['valid'], case['name'])
                for name, module in self.modules.items():
                    with self.subTest(reader=name, vector=case['name']):
                        if name in ('main-tools', 'main-demo'):
                            actual = module.verify_pack(case['receipt'], opening=case['opening'], **options)
                            self.assertEqual(actual, reference)
                        else:
                            # Frozen v1 formats do not gain a v2 API. Their exact
                            # raw-v2 boundary is rejection, not v1 reinterpretation.
                            self.assertFalse(module.verify_pack(case['receipt'], **options)['valid'])


    def test_v2_observation_matrix(self):
        from tests.generate_observation_v2 import corpus
        data = corpus()
        found = {p.relative_to(ROOT).as_posix() for p in ROOT.rglob('receipt_observation_v2.py')}
        self.assertEqual(found, {'tools/receipt_observation_v2.py', 'demo/receipt_observation_v2.py'})
        self.assertEqual((ROOT / 'tools/receipt_observation_v2.py').read_bytes(),
                         (ROOT / 'demo/receipt_observation_v2.py').read_bytes())
        for case in data['vectors']:
            options = dict(jwks=case.get('jwks', data['jwks']), now=data['now'], require_trusted_issuer=True)
            private = dict(parent=case['parent'], opening=case['opening'], result_opening=case['result_opening'])
            with patch('socket.socket.connect', side_effect=AssertionError('network_forbidden')):
                reference = self.modules['main-tools'].verify_pack(case['receipt'], **private, **options)
                self.assertIs(reference['valid'], case['valid'], case['name'])
                for name, module in self.modules.items():
                    with self.subTest(reader=name, vector=case['name']):
                        if name in ('main-tools', 'main-demo'):
                            self.assertEqual(module.verify_pack(case['receipt'], **private, **options), reference)
                        else:
                            self.assertFalse(module.verify_pack(case['receipt'], **options)['valid'])


if __name__ == '__main__':
    unittest.main()
