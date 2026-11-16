"""Offline Cloud MCP address compatibility across both public verifier scripts.

Only dummy or embedded sample keys are used, and discovery is mocked. The
refusal cases assert the coordinator's requested fixed-issuer policy; they also
expose that main does not currently implement that policy.
"""
import json
import unittest
from unittest.mock import patch

from test_paid_rail import Fixture, NOW, ROOT, VERIFIERS

CLOUD = 'https://mcp.satgate.io'
LEGACY = json.loads((ROOT / 'satgate-landing/public/evidence/'
                     'sample-mcp-budget-refusal-20260928.json').read_text())['receipts'][0]['issuer']
GATEWAY = 'https://api.satgate.io'
REJECTED = (
    'https://mcp.satgate.io.example',
    'http://mcp.satgate.io',
    'https://mcp.satgate.io:8443',
    'https://mcp.satgate.io/path',
)


def signed_pack(issuer):
    fixture = Fixture()
    receipt = fixture.receipt()
    receipt['issuer'] = issuer
    receipt['evidence_url'] = issuer + '/synthetic-proof'
    return fixture.pack(receipt), fixture.jwks


class CloudMCPIssuerTests(unittest.TestCase):
    def test_signed_cloud_and_existing_issuers_with_pinned_jwks(self):
        for issuer in (CLOUD, LEGACY, GATEWAY):
            pack, jwks = signed_pack(issuer)
            for surface, verifier in VERIFIERS.items():
                with self.subTest(issuer=issuer, surface=surface):
                    result = verifier.verify_pack(
                        pack, jwks=jwks, require_trusted_issuer=True, now=NOW,
                        jwks_source=issuer + '/.well-known/jwks.json')
                    self.assertTrue(result['valid'], result)
                    self.assertTrue(result['trusted_issuer_valid'], result)

    def test_third_issuer_cannot_use_pinned_cloud_jwks(self):
        for issuer in REJECTED:
            pack, jwks = signed_pack(issuer)
            for surface, verifier in VERIFIERS.items():
                with self.subTest(issuer=issuer, surface=surface):
                    result = verifier.verify_pack(
                        pack, jwks=jwks, require_trusted_issuer=True, now=NOW,
                        jwks_source=CLOUD + '/.well-known/jwks.json')
                    self.assertFalse(result['valid'], result)
                    self.assertFalse(result['trusted_issuer_valid'], result)

    def test_unlisted_issuer_refused_with_local_jwks(self):
        for issuer in REJECTED:
            pack, jwks = signed_pack(issuer)
            for surface, verifier in VERIFIERS.items():
                with self.subTest(issuer=issuer, surface=surface):
                    result = verifier.verify_pack(
                        pack, jwks=jwks, require_trusted_issuer=True, now=NOW)
                    self.assertFalse(result['valid'], 'unlisted issuer accepted')

    def test_discovery_only_chooses_fixed_issuer_addresses(self):
        for issuer in (CLOUD, LEGACY, GATEWAY) + REJECTED:
            pack, jwks = signed_pack(issuer)
            for surface, verifier in VERIFIERS.items():
                with self.subTest(issuer=issuer, surface=surface):
                    with patch.object(verifier, 'load_jwks', return_value=jwks) as load:
                        result = verifier.verify_pack(
                            pack, require_trusted_issuer=True, discover_jwks=True, now=NOW)
                    if issuer in REJECTED:
                        load.assert_not_called()
                        self.assertFalse(result['valid'])
                    else:
                        load.assert_called_once_with(issuer + '/.well-known/jwks.json')
                        self.assertTrue(result['valid'], result)

    def test_existing_signed_cloud_samples_offline(self):
        # The embedded public key supplies an offline fixture, NOT independent
        # authentication of the production issuer's published JWKS.
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
                    self.assertTrue(result['valid'], result)
                    self.assertTrue(result['trusted_issuer_valid'], result)


if __name__ == '__main__':
    unittest.main()
