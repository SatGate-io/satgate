import json
import os
import sys
import unittest
import urllib.parse
from unittest.mock import patch

from test_paid_rail import Fixture, NOW, ROOT, VERIFIERS

CLOUD = 'https://mcp.satgate.io'
LEGACY = json.loads((ROOT / 'satgate-landing/public/evidence/'
                     'sample-mcp-budget-refusal-20260928.json').read_text())['receipts'][0]['issuer']
GATEWAY = 'https://api.satgate.io'
ISSUERS = (GATEWAY, LEGACY, CLOUD)
REJECTED = (
    'https://other.example.net',
    'https://mcp.satgate.io.example.net',
    'https://satgate-mcp-saas.fly.dev.example.net',
    'https://api-satgate.io',
    'https://xn--satgate-mcp-saas-example',
    'https://@mcp.satgate.io',
    'https://mcp.satgate.io/#',
)
INVALID = (
    'http://mcp.satgate.io', 'https://mcp.satgate.io:8443',
    'https://reader@mcp.satgate.io', 'https://mcp.satgate.io/#fragment',
    'https://127.0.0.1', 'https://192.168.1.1',
)
ACCEPTED = ISSUERS + (CLOUD + '/', CLOUD + '/path', CLOUD + ':443/path',
                       'https://MCP.SATGATE.IO/path?query=dummy')
REFUSAL = 'receipt.issuer is not a SatGate address; to trust another issuer, pass its keys with --jwks-url or --jwks-file'


def signed_pack(issuer):
    fixture = Fixture()
    receipt = fixture.receipt()
    receipt['issuer'] = issuer
    receipt['evidence_url'] = issuer + '/synthetic-proof'
    return fixture.pack(receipt), fixture.jwks


def integration_cases():
    for issuer in ACCEPTED + REJECTED + INVALID:
        pack, jwks = signed_pack(issuer)
        parsed = urllib.parse.urlparse(issuer)
        url = 'https://' + parsed.hostname + '/.well-known/jwks.json'
        yield dict(name='discover/' + issuer, pack=pack, jwks=jwks,
                   mode='discovery', valid=issuer in ACCEPTED,
                   calls=[url] if issuer in ACCEPTED else [],
                   code='invalid_issuer' if issuer in INVALID else
                        ('issuer_not_satgate' if issuer in REJECTED else None))
    for issuer in ISSUERS + REJECTED + INVALID:
        pack, jwks = signed_pack(issuer)
        for mode in ('file', 'embedded'):
            yield dict(name=mode + '/' + issuer, pack=pack, jwks=jwks,
                       mode=mode, valid=True, calls=[], code=None)
    for issuer in ISSUERS:
        for mutation in ('wrong-key', 'tampered'):
            pack, jwks = signed_pack(issuer)
            if mutation == 'wrong-key':
                jwks = Fixture().jwks
            else:
                pack['receipts'][0]['route_or_tool'] = '/tampered'
            for mode in ('discovery', 'file'):
                yield dict(name=mode + '/' + mutation + '/' + issuer, pack=pack, jwks=jwks,
                           mode=mode, valid=False,
                           calls=[issuer + '/.well-known/jwks.json'] if mode == 'discovery' else [],
                           code='invalid_signature')


def test_driver():
    from pathlib import Path
    from test_paid_rail import load_verifier
    case = json.loads(Path(os.environ['SATGATE_TEST_CASE']).read_text())
    verifier = load_verifier(os.environ['SATGATE_TEST_SURFACE'])
    args = sys.argv[1:]
    if case['mode'] == 'discovery':
        if '--jwks-file' in args:
            i = args.index('--jwks-file')
            del args[i:i + 2]
        args += ['--discover-jwks', '--require-trusted-issuer']
    calls = []

    def local_keys(source):
        calls.append(source)
        if source.startswith('https://'):
            return case['jwks']
        return json.loads(Path(source).read_text())

    with patch.object(verifier, 'load_jwks', side_effect=local_keys):
        code = verifier.main(args)
    # Only discovery calls count; explicit file loading remains native behavior.
    Path(os.environ['SATGATE_TEST_CALLS']).write_text(json.dumps(
        calls if case['mode'] in ('discovery', 'default-discovery') else []))
    raise SystemExit(code)


class CloudMCPIssuerTests(unittest.TestCase):
    def test_fixed_lists_equal(self):
        for verifier in VERIFIERS.values():
            self.assertEqual(verifier.SATGATE_ISSUER_ORIGINS, frozenset(ISSUERS))

    def test_signed_cloud_and_existing_issuers_with_pinned_jwks(self):
        for issuer in ISSUERS:
            pack, jwks = signed_pack(issuer)
            for surface, verifier in VERIFIERS.items():
                with self.subTest(issuer=issuer, surface=surface):
                    result = verifier.verify_pack(
                        pack, jwks=jwks, require_trusted_issuer=True, now=NOW,
                        jwks_source=issuer + '/.well-known/jwks.json')
                    self.assertTrue(result['valid'], result['reason_codes'])
                    self.assertTrue(result['trusted_issuer_valid'])

    def test_third_issuer_cannot_use_pinned_cloud_jwks(self):
        for issuer in tuple(i for i in REJECTED if urllib.parse.urlparse(i).hostname != 'mcp.satgate.io') + ('http://mcp.satgate.io',):
            pack, jwks = signed_pack(issuer)
            for surface, verifier in VERIFIERS.items():
                with self.subTest(issuer=issuer, surface=surface):
                    result = verifier.verify_pack(
                        pack, jwks=jwks, require_trusted_issuer=True, now=NOW,
                        jwks_source=CLOUD + '/.well-known/jwks.json')
                    self.assertFalse(result['valid'])
                    self.assertIn('jwks_issuer_mismatch', result['reason_codes'])

    def test_supplied_keys_and_embedded_mode_unchanged(self):
        for issuer in ISSUERS + REJECTED + INVALID:
            pack, jwks = signed_pack(issuer)
            for surface, verifier in VERIFIERS.items():
                for source in (None, issuer + '/.well-known/jwks.json'):
                    with self.subTest(issuer=issuer, surface=surface, source=source):
                        with patch.object(verifier, 'load_jwks', side_effect=AssertionError('network')):
                            result = verifier.verify_pack(pack, jwks=jwks,
                                require_trusted_issuer=True, discover_jwks=True,
                                jwks_source=source, now=NOW)
                        self.assertEqual(result['valid'], verifier.jwks_source_matches_issuer(source, issuer), result['reason_codes'])
                        self.assertNotIn('issuer_not_satgate', result['reason_codes'])
                with patch.object(verifier, 'load_jwks', side_effect=AssertionError('network')):
                    result = verifier.verify_pack(pack, discover_jwks=True, now=NOW)
                self.assertTrue(result['valid'], result['reason_codes'])
                self.assertFalse(result['trusted_issuer_valid'])

    def test_discovery_only_chooses_fixed_issuer_addresses(self):
        for case in integration_cases():
            if case['mode'] != 'discovery':
                continue
            for surface, verifier in VERIFIERS.items():
                with self.subTest(case=case['name'], surface=surface):
                    with patch.object(verifier, 'load_jwks', return_value=case['jwks']) as load:
                        result = verifier.verify_pack(case['pack'], require_trusted_issuer=True,
                                                     discover_jwks=True, now=NOW)
                    self.assertEqual([c.args[0] for c in load.call_args_list], case['calls'])
                    self.assertEqual(result['valid'], case['valid'], result['reason_codes'])
                    if case['code']:
                        self.assertIn(case['code'], result['reason_codes'])
                        if case['code'] == 'issuer_not_satgate':
                            self.assertIn(REFUSAL, result['reasons'])
                        else:
                            self.assertNotIn('issuer_not_satgate', result['reason_codes'])

    def test_discovery_checks_all_receipts_before_fetch(self):
        for issuer in REJECTED + INVALID:
            pack, _ = signed_pack(CLOUD)
            other, jwks = signed_pack(issuer)
            pack['receipts'].append(other['receipts'][0])
            for surface, verifier in VERIFIERS.items():
                with self.subTest(issuer=issuer, surface=surface):
                    with patch.object(verifier, 'load_jwks', return_value=jwks) as load:
                        result = verifier.verify_pack(pack, require_trusted_issuer=True,
                                                     discover_jwks=True, now=NOW)
                    load.assert_not_called()
                    self.assertFalse(result['valid'])
                    self.assertIn('invalid_issuer' if issuer in INVALID else 'issuer_not_satgate',
                                  result['reason_codes'])

    def test_supplied_keys_reject_wrong_key_and_tampering(self):
        for case in integration_cases():
            if case['mode'] != 'file' or case['valid']:
                continue
            for surface, verifier in VERIFIERS.items():
                with self.subTest(case=case['name'], surface=surface):
                    result = verifier.verify_pack(case['pack'], jwks=case['jwks'],
                        require_trusted_issuer=True, now=NOW)
                    self.assertFalse(result['valid'])
                    self.assertIn('invalid_signature', result['reason_codes'])

    def test_explicit_url_and_file_entrypoints(self):
        import contextlib
        import io
        import tempfile
        from pathlib import Path
        pack, jwks = signed_pack(REJECTED[0])
        for surface, verifier in VERIFIERS.items():
            with tempfile.TemporaryDirectory() as tmp:
                path = Path(tmp) / 'pack.json'
                path.write_text(json.dumps(pack))
                for flag, source in (('--jwks-url', REJECTED[0] + '/keys'),
                                     ('--jwks-file', str(Path(tmp) / 'keys.json'))):
                    with self.subTest(surface=surface, flag=flag):
                        output = io.StringIO()
                        with patch.object(verifier, 'load_jwks', return_value=jwks) as load:
                            with contextlib.redirect_stdout(output):
                                code = verifier.main([str(path), flag, source,
                                    '--discover-jwks', '--require-trusted-issuer', '--now', NOW])
                        self.assertEqual(code, 0)
                        load.assert_called_once_with(source)
                        self.assertTrue(json.loads(output.getvalue())['valid'])

    def test_existing_signed_cloud_samples_offline(self):
        # Embedded public keys: compatibility fixtures, not production authentication.
        folder = ROOT / 'satgate-landing/public/evidence'
        for path in sorted(folder.glob('*.json')):
            pack = json.loads(path.read_text())
            receipts = pack.get('receipts', [])
            if not receipts or receipts[0].get('issuer') != LEGACY:
                continue
            receipt = receipts[0]
            jwks = {'keys': [{'kid': receipt['issuer_kid'], 'kty': 'OKP',
                              'crv': 'Ed25519', 'alg': 'EdDSA',
                              'x': receipt['metadata']['public_key_ed25519_b64']}]}
            for surface, verifier in VERIFIERS.items():
                with self.subTest(sample=path.name, surface=surface):
                    result = verifier.verify_pack(
                        pack, jwks=jwks, require_trusted_issuer=True,
                        jwks_source=LEGACY + '/.well-known/jwks.json')
                    self.assertTrue(result['valid'], result['reason_codes'])
                    self.assertTrue(result['trusted_issuer_valid'])


if __name__ == '__main__':
    unittest.main()
