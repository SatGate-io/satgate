"""Offline integration of AgentClient.verify_pack with the real public verifier."""
import json
import sys
from pathlib import Path

import pytest

pytest.importorskip('cryptography')
ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(ROOT / 'tools'))
from test_cloud_mcp_issuer import CLOUD, GATEWAY, LEGACY, REJECTED, signed_pack
from satgate.agent_client import SatGateAgentClient
from satgate.exceptions import SatGateError


@pytest.mark.parametrize('surface', ['tools', 'demo'])
@pytest.mark.parametrize('issuer', [CLOUD, LEGACY, GATEWAY])
def test_verify_pack_cloud_and_existing_issuers(monkeypatch, surface, issuer):
    monkeypatch.setenv('SATGATE_VERIFIER', str(ROOT / surface / 'verify_evidence_pack.py'))
    pack, jwks = signed_pack(issuer)
    client = SatGateAgentClient(gateway_url='http://127.0.0.1:1')
    result = client.verify_pack(pack, jwks)
    assert result['valid'] is True
    assert result['trusted_issuer_valid'] is True


@pytest.mark.parametrize('surface', ['tools', 'demo'])
@pytest.mark.parametrize('issuer', REJECTED)
def test_verify_pack_refuses_unlisted_issuer(monkeypatch, surface, issuer):
    monkeypatch.setenv('SATGATE_VERIFIER', str(ROOT / surface / 'verify_evidence_pack.py'))
    pack, jwks = signed_pack(issuer)
    client = SatGateAgentClient(gateway_url='http://127.0.0.1:1')
    with pytest.raises(SatGateError):
        client.verify_pack(pack, jwks)


@pytest.mark.parametrize('surface', ['tools', 'demo'])
def test_verify_pack_existing_cloud_samples_offline(monkeypatch, surface):
    monkeypatch.setenv('SATGATE_VERIFIER', str(ROOT / surface / 'verify_evidence_pack.py'))
    client = SatGateAgentClient(gateway_url='http://127.0.0.1:1')
    for path in sorted((ROOT / 'satgate-landing/public/evidence').glob('*.json')):
        pack = json.loads(path.read_text())
        receipts = pack.get('receipts', [])
        if not receipts or receipts[0].get('issuer') != LEGACY:
            continue
        r = receipts[0]
        # Fixture key only: not an independently fetched production trust anchor.
        jwks = {'keys': [{'kid': r['issuer_kid'], 'kty': 'OKP', 'crv': 'Ed25519',
                          'x': r['metadata']['public_key_ed25519_b64']}]}
        assert client.verify_pack(pack, jwks)['valid'] is True
