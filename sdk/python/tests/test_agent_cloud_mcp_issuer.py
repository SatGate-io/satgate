import json
import sys
from pathlib import Path

import pytest

pytest.importorskip('cryptography')
ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(ROOT / 'tools'))
from test_cloud_mcp_issuer import (CLOUD, LEGACY, GATEWAY, REJECTED, INVALID,
                                  REFUSAL, signed_pack, integration_cases)
from satgate.agent_client import SatGateAgentClient
from satgate.exceptions import SatGateError


@pytest.mark.parametrize('surface', ['tools', 'demo'])
@pytest.mark.parametrize('issuer', [CLOUD, LEGACY, GATEWAY] + list(REJECTED + INVALID))
def test_verify_pack_supplied_keys(monkeypatch, surface, issuer):
    monkeypatch.setenv('SATGATE_VERIFIER', str(ROOT / surface / 'verify_evidence_pack.py'))
    pack, jwks = signed_pack(issuer)
    client = SatGateAgentClient(gateway_url='http://127.0.0.1:1')
    result = client.verify_pack(pack, jwks)
    assert result['valid'] is True
    assert result['trusted_issuer_valid'] is True


@pytest.mark.parametrize('surface', ['tools', 'demo'])
@pytest.mark.parametrize('case', [c for c in integration_cases() if c['mode'] == 'discovery'], ids=lambda c: c['name'])
def test_verify_pack_delegated_discovery(monkeypatch, tmp_path, surface, case):
    # Test-only adapter; SDK itself has no issuer-derived discovery mode.
    driver = tmp_path / 'driver.py'
    driver.write_text('import sys\nsys.path.insert(0, ' + repr(str(ROOT / 'tools')) +
                      ')\nfrom test_cloud_mcp_issuer import test_driver\ntest_driver()\n')
    case_path = tmp_path / 'case.json'
    case_path.write_text(json.dumps(case))
    calls_path = tmp_path / 'calls.json'
    monkeypatch.setenv('SATGATE_VERIFIER', str(driver))
    monkeypatch.setenv('SATGATE_TEST_CASE', str(case_path))
    monkeypatch.setenv('SATGATE_TEST_CALLS', str(calls_path))
    monkeypatch.setenv('SATGATE_TEST_SURFACE', surface)
    client = SatGateAgentClient(gateway_url='http://127.0.0.1:1')
    if case['valid']:
        assert client.verify_pack(case['pack'], case['jwks'])['valid'] is True
    else:
        with pytest.raises(SatGateError) as error:
            client.verify_pack(case['pack'], case['jwks'])
        result = json.loads(str(error.value))
        assert case['code'] in result['reason_codes']
        if case['code'] == 'issuer_not_satgate':
            assert REFUSAL in result['reasons']
    assert json.loads(calls_path.read_text()) == case['calls']


@pytest.mark.parametrize('surface', ['tools', 'demo'])
@pytest.mark.parametrize('case', [c for c in integration_cases() if c['mode'] == 'file' and not c['valid']], ids=lambda c: c['name'])
def test_verify_pack_native_crypto_failure(monkeypatch, surface, case):
    monkeypatch.setenv('SATGATE_VERIFIER', str(ROOT / surface / 'verify_evidence_pack.py'))
    client = SatGateAgentClient(gateway_url='http://127.0.0.1:1')
    with pytest.raises(SatGateError) as error:
        client.verify_pack(case['pack'], case['jwks'])
    assert 'invalid_signature' in json.loads(str(error.value))['reason_codes']


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
        jwks = {'keys': [{'kid': r['issuer_kid'], 'kty': 'OKP', 'crv': 'Ed25519',
                          'x': r['metadata']['public_key_ed25519_b64']}]}
        assert client.verify_pack(pack, jwks)['valid'] is True
