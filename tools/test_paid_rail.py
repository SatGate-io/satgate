"""Shared signed corpus for both independently distributed verifier entry points.

Synthetic keys/artifacts only. Every mirror is a deep copy of its signed source.
Run: python -B -m unittest discover -s tools -p 'test_*.py' -v
"""
import copy
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch
from typing import Any

from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PrivateKey
from cryptography.hazmat.primitives.serialization import Encoding, PublicFormat

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / 'demo' / '_vendor'))


def load_verifier(surface):
    spec = importlib.util.spec_from_file_location(
        'verify_' + surface, ROOT / surface / 'verify_evidence_pack.py')
    assert spec is not None and spec.loader is not None
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


VERIFIERS = {surface: load_verifier(surface) for surface in ('tools', 'demo')}
V = VERIFIERS['tools']
NOW = '2026-01-02T00:00:00Z'
PAID_FIELDS = ('rail', 'amount_sats', 'payment_hash', 'invoice_hash', 'macaroon_hash')
LEGACY_PROFILES = (
    ('allowed', 'budget_authorized', True),
    ('allowed', 'policy_allowed', True),
    ('allowed', 'sandbox_no_spend', False),
    ('allowed', 'observe_projected', False),
    ('denied', 'budget_exhausted', True),
    ('denied', 'policy_denied', True),
    ('denied', 'capability_invalid', True),
    ('denied', 'capability_expired', True),
    ('denied', 'auth_missing', False),
    ('denied', 'token_revoked', False),
    ('denied', 'payment_required', True),
)


class Fixture:
    def __init__(self):
        self.key = Ed25519PrivateKey.generate()
        self.x = V.b64url_encode(self.key.public_key().public_bytes(Encoding.Raw, PublicFormat.Raw))
        self.jwks = {'keys': [{'kid': 'test-key', 'kty': 'OKP', 'crv': 'Ed25519',
                               'alg': 'EdDSA', 'use': 'sig', 'x': self.x}]}

    def receipt(self, decision='allowed', reason='payment_verified'):
        r: dict[str, Any] = dict(schema_version='satgate.receipt.v1', canonicalization='jcs-rfc8785',
                 hash_algorithm='sha256', signature_algorithm='ed25519',
                 issuer='https://issuer.example', issuer_kid='test-key',
                 decision=decision, decision_reason=reason,
                 issued_at='2026-01-01T00:00:00Z', evidence_pack_id='test-pack',
                 evidence_url='https://issuer.example/synthetic-proof', receipt_id='test-receipt',
                 route_or_tool='/synthetic', policy_version='v1',
                 policy={'policy_id': 'test-policy', 'policy_version': 'v1'},
                 authority={'provenance_level': 'verified_macaroon_caveats'},
                 metadata={'public_key_ed25519_b64': self.x, 'tenant_id': 'test-tenant',
                           'request_id': 'test-request'})
        if reason == 'payment_verified':
            r.update(rail='l402', amount_sats=1, payment_hash='a' * 64,
                     invoice_hash='b' * 64, macaroon_hash='c' * 64)
            r['paid_rail_context'] = {k: r[k] for k in PAID_FIELDS}
            r['budget'] = dict(spend_mode='paid_rail', rail='l402', amount_sats=1)
        elif reason in ('capability_invalid', 'capability_expired', 'auth_missing',
                        'token_revoked', 'payment_required'):
            r['authority'] = {'provenance_level': 'no_verified_capability'}
            r['budget'] = {'spend_mode': 'not_evaluated', 'cost_credits': 0}
        elif reason == 'observe_projected':
            r['projected_cost_credits'] = 1
            r['budget'] = dict(spend_mode='projected_only', projected_cost_credits=1, credit_unit='credit')
            r['policy'].update(policy_decision_mode='observe', projected_cost_credits=1)
            r['metadata'].update(projected_cost_credits=1)
        else:
            r.update(attempted_amount_usd=1, remaining_budget_usd=0)
            r['budget'] = dict(spend_mode='evaluated', cost_credits=1, credit_unit='credit')
            if reason == 'sandbox_no_spend':
                r.update(attempted_amount_usd=0, remaining_budget_usd=0)
                r['authority'] = {'provenance_level': 'sandbox_no_capability'}
                r['budget'].update(spend_mode='sandbox_no_spend', cost_credits=0)
                r['policy'].update(policy_decision_mode='sandbox', cost_credits=0)
        return r

    def sign(self, receipt):
        r = copy.deepcopy(receipt)
        r['receipt_hash'] = V.sha256_receipt_hash(r)
        r['signature'] = 'ed25519:' + V.b64url_encode(self.key.sign(V.canonical_receipt_payload(r)))
        return r

    def pack(self, receipt=None):
        r = self.sign(receipt if receipt is not None else self.receipt())
        fields = ('issuer', 'decision', 'decision_reason', 'evidence_pack_id', 'evidence_url',
                  'issued_at', 'receipt_id', 'receipt_hash', 'route_or_tool', 'capability_hash',
                  'authority', 'budget', 'policy')
        p = {k: copy.deepcopy(r[k]) for k in fields if k in r}
        p.update(schema_version='satgate.evidence_pack.v1', receipts=[r])
        reason = r['decision_reason']
        if reason != 'payment_verified':
            if reason in ('observe_projected', 'capability_invalid', 'capability_expired',
                          'auth_missing', 'token_revoked', 'payment_required'):
                p['budget_state'] = copy.deepcopy(r['budget'])
            else:
                p['budget_state'] = {k: copy.deepcopy(r[k]) for k in
                                     ('attempted_amount_usd', 'remaining_budget_usd')}
                p['budget_state']['credit_unit'] = r['budget']['credit_unit']
            if reason == 'sandbox_no_spend':
                p.update(tenant_id='test-tenant', request_id='test-request', redaction={})
        return p

    @staticmethod
    def archive(pack):
        pack.update(archive_profile='satgate.archive.v1', archive_storage='test-memory',
                    archive_retention='test-only', archived_at='2026-01-01T00:00:00Z')
        payload = {k: v for k, v in pack.items() if k != 'evidence_pack_hash'}
        pack['evidence_pack_hash'] = 'sha256:' + V.b64url_encode(
            hashlib.sha256(V.rfc8785.dumps(payload)).digest())
        return pack


def set_amount(r, value):
    r['amount_sats'] = value
    r['paid_rail_context']['amount_sats'] = copy.deepcopy(value)
    r['budget']['amount_sats'] = copy.deepcopy(value)


def signed_corpus(f):
    for decision in ('allowed', 'paid'):
        for amount in (1, 100, 2**53 - 1):
            r = f.receipt(decision)
            set_amount(r, amount)
            yield f'{decision}-{amount}', f.pack(r), True
    mutations = []
    for value in (True, False, 0, -1, 1.0, 1.5, 100.0, '1', None, [], {}):
        mutations.append((f'amount-{value!r}', lambda r, v=value: set_amount(r, v)))
    for field in ('payment_hash', 'invoice_hash', 'macaroon_hash'):
        mutations.extend([
            ('missing-' + field, lambda r, k=field: r.pop(k)),
            ('blank-' + field, lambda r, k=field: (r.update({k: ' '}), r['paid_rail_context'].update({k: ' '}))),
            ('mismatch-' + field, lambda r, k=field: r['paid_rail_context'].update({k: 'wrong'})),
            ('type-' + field, lambda r, k=field: (r.update({k: []}), r['paid_rail_context'].update({k: []}))),
        ])
    for value in (None, [], {}, 'l402'):
        mutations.append((f'context-{value!r}', lambda r, v=value: r.update(paid_rail_context=v)))
    for field in ('budget', 'paid_rail_context'):
        for value in (True, 1.0, '1'):
            mutations.append((f'{field}-{value!r}', lambda r, k=field, v=value: r[k].update(amount_sats=v)))
        mutations.append(('extra-' + field, lambda r, k=field: r[k].update(extra=True)))
    mutations.extend([
        ('wrong-rail', lambda r: (r.update(rail='other'), r['paid_rail_context'].update(rail='other'), r['budget'].update(rail='other'))),
        ('denied', lambda r: r.update(decision='denied')),
    ])
    for field in sorted(V.CREDIT_USD_FIELDS):
        mutations.append(('credit-' + field, lambda r, k=field: r['metadata'].update(nested=[{'deep': {k: None}}])))
    for label, mutate in mutations:
        r = f.receipt()
        mutate(r)
        yield label, f.pack(r), False


class PaidRailTests(unittest.TestCase):
    def setUp(self):
        self.f = Fixture()

    def verify(self, surface, pack, **kwargs):
        options = dict(jwks=self.f.jwks, require_trusted_issuer=True, now=NOW)
        options.update(kwargs)
        return VERIFIERS[surface].verify_pack(copy.deepcopy(pack), **options)

    def cli(self, surface, raw, *args, now: str | None = NOW):
        env = dict(os.environ, PYTHONPATH=str(ROOT / 'demo' / '_vendor'), PYTHONDONTWRITEBYTECODE='1')
        proc = subprocess.run([sys.executable, '-B', str(ROOT / surface / 'verify_evidence_pack.py'),
                               '-', *(['--now', now] if now is not None else []), *args], input=raw, text=True,
                              capture_output=True, env=env, timeout=20)
        self.assertNotIn('Traceback', proc.stderr, proc.stderr)
        result = json.loads(proc.stdout)
        self.assertIs(type(result['valid']), bool)
        self.assertEqual(proc.returncode == 0, result['valid'], proc.stderr)
        return proc, result

    def test_shared_signed_corpus(self):
        for label, p, expected in signed_corpus(self.f):
            for surface in VERIFIERS:
                with self.subTest(case=label, surface=surface):
                    result = self.verify(surface, p)
                    self.assertTrue(result['checks']['signature_valid'], result)
                    self.assertTrue(result['trusted_issuer_valid'], result)
                    self.assertIs(result['valid'], expected, result)
                    if not expected:
                        self.assertTrue(result['reason_codes'])

    def test_shared_signed_cli_corpus(self):
        with tempfile.TemporaryDirectory() as td:
            jwks = Path(td) / 'synthetic-jwks.json'
            jwks.write_text(json.dumps(self.f.jwks))
            for label, p, expected in signed_corpus(self.f):
                for surface in VERIFIERS:
                    with self.subTest(case=label, surface=surface):
                        _, result = self.cli(surface, json.dumps(p), '--jwks-file', str(jwks), '--require-trusted-issuer')
                        self.assertIs(result['valid'], expected, result)
                        self.assertTrue(result['checks']['signature_valid'], result)

    def test_temporal_expiration_apis_and_clis(self):
        cases = [('absent', {}, True, None), ('null', {'expires_at': None}, True, None),
                 ('future', {'expires_at': '2026-01-03T00:00:00Z'}, True, None),
                 ('at-expiry', {'expires_at': NOW}, True, None),
                 ('expired', {'expires_at': '2026-01-01T12:00:00Z'}, False, 'receipt_expired')]
        for value in ({}, [], True, False, 17, 0, '', 'not-a-time'):
            code = 'malformed_expires_at' if value == 'not-a-time' else 'missing_expires_at'
            cases.append((repr(value), {'expires_at': value}, False, code))
        with tempfile.TemporaryDirectory() as td:
            keys = Path(td) / 'synthetic-jwks.json'
            keys.write_text(json.dumps(self.f.jwks))
            for label, fields, expected, code in cases:
                for index in (0, 1):
                    r = self.f.receipt()
                    r.update(fields)
                    p = self.f.pack(r) if index == 0 else self.f.pack()
                    if index == 1:
                        r['receipt_id'] = 'synthetic-secondary'
                        p['receipts'].append(self.f.sign(r))
                    for surface in VERIFIERS:
                        with self.subTest(surface=surface, case=label, receipt=index):
                            api = self.verify(surface, p)
                            proc, cli = self.cli(surface, json.dumps(p), '--jwks-file', str(keys),
                                                 '--require-trusted-issuer')
                            self.assertEqual(proc.returncode, 0 if expected else 1)
                            for result in (api, cli):
                                self.assertIs(result['valid'], expected, result)
                                self.assertTrue(result['checks']['signature_valid'], result)
                                self.assertTrue(result['trusted_issuer_valid'], result)
                                self.assertIs(result['checks']['time_valid'], expected, result)
                                self.assertIs(result['checks'][f'receipt_{index}_time_valid'], expected)
                                if index == 1:
                                    self.assertTrue(result['checks']['receipt_0_time_valid'])
                                if code:
                                    self.assertIn(code, result['reason_codes'])
                                else:
                                    self.assertEqual(result['reason_codes'], [])

    def test_malformed_now_apis_and_clis(self):
        with tempfile.TemporaryDirectory() as td:
            keys = Path(td) / 'synthetic-jwks.json'
            keys.write_text(json.dumps(self.f.jwks))
            # Reject before receipt evaluation, with or without an expired claim.
            for expiry in (None, '2026-01-01T12:00:00Z'):
                r = self.f.receipt()
                r['expires_at'] = expiry
                p = self.f.pack(r)
                for surface, module in VERIFIERS.items():
                    for now in ('not-a-time', '', ' ', {}, [], True, False, 17, 0):
                        with self.subTest(surface=surface, expiry=expiry, now=now):
                            with patch.object(module, 'verify_receipt', side_effect=AssertionError('receipt evaluated')):
                                result = self.verify(surface, p, now=now)
                            self.assertFalse(result['valid'], result)
                            self.assertFalse(result['checks']['time_valid'], result)
                            self.assertTrue(set(result['reason_codes']) & {'missing_now', 'malformed_now'})
                            if isinstance(now, str):
                                proc, cli = self.cli(surface, json.dumps(p), '--jwks-file', str(keys),
                                                     '--require-trusted-issuer', now=now)
                                self.assertEqual(proc.returncode, 1)
                                self.assertEqual({key: cli[key] for key in result}, result)

    def test_default_now_apis_and_clis(self):
        with tempfile.TemporaryDirectory() as td:
            keys = Path(td) / 'synthetic-jwks.json'
            keys.write_text(json.dumps(self.f.jwks))
            for fields in ({}, {'expires_at': None}):
                r = self.f.receipt()
                r.update(fields)
                p = self.f.pack(r)
                for surface, module in VERIFIERS.items():
                    with self.subTest(surface=surface, fields=fields):
                        self.assertTrue(module.verify_pack(p, jwks=self.f.jwks,
                                                           require_trusted_issuer=True)['valid'])
                        self.assertTrue(self.verify(surface, p, now=None)['valid'])
                        proc, result = self.cli(surface, json.dumps(p), '--jwks-file', str(keys),
                                                '--require-trusted-issuer', now=None)
                        self.assertEqual(proc.returncode, 0)
                        self.assertTrue(result['valid'], result)

    def test_no_fixture_aliasing(self):
        r = self.f.receipt()
        p = self.f.pack(r)
        for field in ('budget', 'authority', 'policy'):
            self.assertIsNot(p[field], p['receipts'][0][field])
            self.assertIsNot(p[field], r[field])
        p['budget']['amount_sats'] = True
        self.assertIs(type(p['receipts'][0]['budget']['amount_sats']), int)
        self.assertIs(type(r['budget']['amount_sats']), int)

    def test_recursive_mirror_types_even_after_pack_rehash(self):
        for field in ('budget', 'authority', 'policy'):
            pairs = ((1, True), (0, False), (1, 1.0), (1.0, 1), ([1], [True]),
                     ({'a': [0, {'b': 1}]}, {'a': [False, {'b': 1.0}]}))
            for original, changed in pairs:
                if field == 'budget' and type(original) is not int:
                    continue  # Paid budget shape/amount are exact, unlike policy/authority.
                r = self.f.receipt()
                if field != 'budget':
                    r[field]['nested'] = [{'value': original}]
                elif original != 1:
                    continue  # Positive paid sats only; zero/false covered in nested subtrees.
                p = self.f.pack(r)
                signed_bytes = V.canonical_receipt_payload(p['receipts'][0])
                if field == 'budget':
                    p[field]['amount_sats'] = changed
                else:
                    p[field]['nested'][0]['value'] = changed
                self.f.archive(p)
                self.assertEqual(signed_bytes, V.canonical_receipt_payload(p['receipts'][0]))
                for surface in VERIFIERS:
                    with self.subTest(surface=surface, field=field, original=original, changed=changed):
                        result = self.verify(surface, p)
                        self.assertFalse(result['valid'], result)
                        self.assertTrue(result['checks']['signature_valid'], result)
                        self.assertIn('pack_primary_mismatch', result['reason_codes'])
                        hash_check = 'archive_valid' if surface == 'demo' else 'evidence_pack_hash_match'
                        self.assertTrue(result['checks'][hash_check], result)

    def test_optional_paid_and_common_mirrors(self):
        for field in (*PAID_FIELDS, 'paid_rail_context', 'issued_at', 'evidence_url',
                      'timestamp', 'policy_version', 'jwks_url', 'schema_url', 'verify_url'):
            r = self.f.receipt()
            if field not in r:
                r[field] = '2026-01-01T00:00:00Z' if field == 'timestamp' else 'https://issuer.example/synthetic'
            for surface in VERIFIERS:
                with self.subTest(field=field, surface=surface):
                    p = self.f.pack(r)
                    p[field] = copy.deepcopy(r[field])
                    self.assertTrue(self.verify(surface, p)['valid'])
                    for value in (None, True, 'contradiction', {'nested': [1]}):
                        bad = copy.deepcopy(p)
                        bad[field] = value
                        result = self.verify(surface, bad)
                        self.assertFalse(result['valid'], result)
                        self.assertTrue(result['checks']['signature_valid'], result)
                        self.assertIn('pack_primary_mismatch', result['reason_codes'])
        # Optional paid claims without any signed counterpart fail even if null.
        for field in (*PAID_FIELDS, 'paid_rail_context'):
            p = self.f.pack(self.f.receipt('allowed', 'policy_allowed'))
            p[field] = None
            for surface in VERIFIERS:
                result = self.verify(surface, p)
                self.assertFalse(result['valid'], result)
                self.assertIn('pack_primary_mismatch', result['reason_codes'])

    def test_pack_contradictions(self):
        mutations = [('budget_state', value) for value in (None, {}, [], 'paid_rail')]
        mutations += [('extra', [{'deep': {field: None}}]) for field in V.CREDIT_USD_FIELDS]
        mutations += [(field, value) for field in ('budget', 'authority', 'policy') for value in ({}, [], None)]
        for field, value in mutations:
            p = self.f.pack()
            p[field] = value
            for surface in VERIFIERS:
                with self.subTest(surface=surface, field=field, value=value):
                    result = self.verify(surface, p)
                    self.assertFalse(result['valid'], result)
                    self.assertTrue(result['checks']['signature_valid'], result)
                    self.assertTrue(result['reason_codes'])

    def test_signature_hash_second_receipt_and_trust_controls(self):
        mutations = [
            lambda p: p['receipts'][0].update(amount_sats=2),
            lambda p: p['receipts'][0].pop('signature'),
            lambda p: p.update(receipt_hash='sha256:wrong'),
            lambda p: p.update(evidence_pack_hash='sha256:wrong'),
            lambda p: p['receipts'].append(dict(copy.deepcopy(p['receipts'][0]), invoice_hash='changed')),
        ]
        for mutate in mutations:
            p = self.f.pack()
            mutate(p)
            for surface in VERIFIERS:
                self.assertFalse(self.verify(surface, p)['valid'])
        p = self.f.pack()
        p['receipts'].append(copy.deepcopy(p['receipts'][0]))
        for surface in VERIFIERS:
            self.assertTrue(self.verify(surface, p)['valid'])
        bad_keys = [None, {}, {'keys': []}, {'keys': {}}, {'keys': [dict(self.f.jwks['keys'][0], kid='missing')]},
                    {'keys': [dict(self.f.jwks['keys'][0], x=V.b64url_encode(bytes(32)))]},
                    {'keys': self.f.jwks['keys'] * 2}, [],
                    {'keys': [dict(self.f.jwks['keys'][0], alg=[])]},
                    {'keys': [dict(self.f.jwks['keys'][0], use={})]}]
        for keys in bad_keys:
            for surface in VERIFIERS:
                with self.subTest(surface=surface, keys=keys):
                    result = self.verify(surface, p, jwks=keys)
                    self.assertFalse(result['valid'], result)
                    self.assertFalse(result['trusted_issuer_valid'], result)
        for surface in VERIFIERS:
            self.assertFalse(self.verify(surface, p, jwks_source='https://other.example/.well-known/jwks.json')['valid'])
            self.assertTrue(self.verify(surface, p, jwks_source='https://issuer.example/.well-known/jwks.json')['valid'])
            embedded = self.verify(surface, p, jwks=None, require_trusted_issuer=False)
            self.assertTrue(embedded['valid'], embedded)
            self.assertFalse(embedded['trusted_issuer_valid'])

    def test_legacy_profile_acceptance_and_rejection(self):
        for decision, reason, tools_accept in LEGACY_PROFILES:
            p = self.f.pack(self.f.receipt(decision, reason))
            for surface in VERIFIERS:
                expected = tools_accept if surface == 'tools' else True
                with self.subTest(surface=surface, decision=decision, reason=reason):
                    result = self.verify(surface, p)
                    self.assertIs(result['valid'], expected, result)
                    self.assertTrue(result['checks']['signature_valid'], result)
                    if not expected:
                        self.assertIn('unknown_decision_reason', result['reason_codes'])
                    bad = copy.deepcopy(p)
                    bad['receipts'][0]['issued_at'] = '2000-01-01T00:00:00Z'
                    self.assertFalse(self.verify(surface, bad)['valid'])
                    bad = copy.deepcopy(p)
                    bad['decision_reason'] = 'unsigned-change'
                    self.assertFalse(self.verify(surface, bad)['valid'])
        # Preserve pre-existing tools enum-only vs demo paired-profile behavior.
        p = self.f.pack(self.f.receipt('paid', 'budget_authorized'))
        self.assertTrue(self.verify('tools', p)['valid'])
        self.assertFalse(self.verify('demo', p)['valid'])
        for decision, reason in (('unknown', 'budget_authorized'), ('allowed', 'unknown')):
            p = self.f.pack(self.f.receipt(decision, reason))
            for surface in VERIFIERS:
                self.assertFalse(self.verify(surface, p)['valid'])

    def test_legacy_signed_rejections_and_cli_profiles(self):
        with tempfile.TemporaryDirectory() as td:
            keys = Path(td) / 'jwks.json'
            keys.write_text(json.dumps(self.f.jwks))
            for decision, reason, tools_accept in LEGACY_PROFILES:
                p = self.f.pack(self.f.receipt(decision, reason))
                for surface in VERIFIERS:
                    expected = tools_accept if surface == 'tools' else True
                    with self.subTest(surface=surface, reason=reason):
                        _, result = self.cli(surface, json.dumps(p), '--jwks-file', str(keys), '--require-trusted-issuer')
                        self.assertIs(result['valid'], expected, result)
                        # Cryptographically valid but unsupported protocol versions
                        # remain rejected on every profile and both surfaces.
                        bad_r = self.f.receipt(decision, reason)
                        bad_r['schema_version'] = 'unsupported'
                        bad = self.f.pack(bad_r)
                        result = self.verify(surface, bad)
                        self.assertTrue(result['checks']['signature_valid'], result)
                        self.assertFalse(result['valid'], result)
                        self.assertIn('invalid_receipt_schema', result['reason_codes'])
            # Demo's stricter provenance rejection is not replaced by tools'
            # older non-paid enum-only contract.
            for decision, reason, tools_accept in LEGACY_PROFILES:
                if reason not in ('sandbox_no_spend', 'observe_projected', 'capability_invalid',
                                  'capability_expired', 'auth_missing', 'token_revoked', 'payment_required'):
                    continue
                r = self.f.receipt(decision, reason)
                r['budget']['spend_mode'] = 'contradictory'
                p = self.f.pack(r)
                self.assertFalse(self.verify('demo', p)['valid'])
                self.assertIs(self.verify('tools', p)['valid'], tools_accept)

    def test_exponent_sats_and_mixed_receipts(self):
        raw = json.dumps(self.f.pack()).replace('"amount_sats": 1', '"amount_sats": 1e0')
        for surface in VERIFIERS:
            args = ['--allow-embedded-key'] if surface == 'demo' else []
            _, result = self.cli(surface, raw, *args)
            self.assertFalse(result['valid'], result)
            self.assertTrue(result['checks']['signature_valid'], result)
            self.assertIn('invalid_paid_rail_provenance', result['reason_codes'])
            p = self.f.pack(self.f.receipt('allowed', 'budget_authorized'))
            p['receipts'].append(self.f.sign(self.f.receipt()))
            result = self.verify(surface, p)
            self.assertFalse(result['valid'], result)
            self.assertTrue(result['checks']['signature_valid'], result)
            self.assertIn('invalid_paid_rail_provenance', result['reason_codes'])
            p = self.f.pack()
            bad_second = self.f.receipt('paid')
            bad_second['paid_rail_context']['amount_sats'] = True
            p['receipts'].append(self.f.sign(bad_second))
            result = self.verify(surface, p)
            self.assertFalse(result['valid'], result)
            self.assertTrue(result['checks']['signature_valid'], result)

    def test_legacy_budget_mirror_types(self):
        for field, original, changed in (('attempted_amount_usd', 1, True),
                                          ('remaining_budget_usd', 0, False),
                                          ('attempted_amount_usd', 1, 1.0)):
            r = self.f.receipt('allowed', 'budget_authorized')
            r[field] = original
            p = self.f.pack(r)
            p['budget_state'][field] = changed
            for surface in VERIFIERS:
                result = self.verify(surface, p)
                self.assertFalse(result['valid'], result)
                self.assertIn('budget_state_mismatch', result['reason_codes'])

    def test_cli_trust_and_mock_modes(self):
        p = self.f.pack()
        _, result = self.cli('tools', json.dumps(p))
        self.assertTrue(result['valid'])
        self.assertFalse(result['trusted_issuer_valid'])
        proc, result = self.cli('demo', json.dumps(p), '--allow-embedded-key')
        self.assertTrue(result['valid'])
        self.assertFalse(result['trusted_issuer_valid'])
        self.assertIn('WARNING', proc.stderr)
        with tempfile.TemporaryDirectory() as td:
            keys = Path(td) / 'jwks.json'
            keys.write_text(json.dumps(self.f.jwks))
            _, result = self.cli('demo', json.dumps(p), '--jwks-file', str(keys))
            self.assertTrue(result['trusted_issuer_valid'])
            keys.write_text('{"keys": []}')
            for surface in VERIFIERS:
                _, result = self.cli(surface, json.dumps(p), '--jwks-file', str(keys), '--require-trusted-issuer')
                self.assertFalse(result['valid'])
        p['mock_only'] = True
        for surface in VERIFIERS:
            args = ['--allow-embedded-key'] if surface == 'demo' else []
            self.assertFalse(self.cli(surface, json.dumps(p), *args)[1]['valid'])
            self.assertTrue(self.cli(surface, json.dumps(p), *args, '--allow-mock')[1]['valid'])

    def test_malformed_cli_and_api_inputs(self):
        raw_cases = ['{', '[]', 'null', '{"environment":{}}', '{"receipts":[{"decision":[]}]}',
                     '{"receipts":[{"decision":"allowed","decision_reason":[]}]}',
                     '{"receipts":[{"metadata":{"environment":{}}}]}',
                     '{"receipts":[{"budget":null}]}', '{"receipts":[[]]}',
                     '{"extra":"\\ud800"}', '[' * 1500 + '0' + ']' * 1500]
        for value in ('1e309', 'NaN', 'Infinity', '-Infinity', '9007199254740992'):
            raw_cases += ['{"extra":' + value + '}',
                          '{"receipts":[{"decision":"allowed","decision_reason":"payment_verified","amount_sats":' + value + '}]}']
        for surface in VERIFIERS:
            for raw in raw_cases:
                with self.subTest(surface=surface, raw=raw[:120]):
                    args = ['--allow-embedded-key'] if surface == 'demo' else []
                    proc, result = self.cli(surface, raw, *args)
                    self.assertNotEqual(proc.returncode, 0)
                    self.assertFalse(result['valid'])
                    try:
                        parsed = json.loads(raw)
                    except (ValueError, RecursionError):
                        continue
                    self.assertFalse(self.verify(surface, parsed)['valid'])
            for value in (float('inf'), float('nan'), 2**53):
                p = self.f.pack()
                p['extra'] = value
                self.assertFalse(self.verify(surface, p)['valid'])
            # Exercise actual canonicalizer failure at the API boundary, too.
            module = VERIFIERS[surface]
            with patch.object(module.rfc8785, 'dumps', side_effect=ValueError('bad JCS')):
                self.assertFalse(self.verify(surface, {})['valid'])


if __name__ == '__main__':
    unittest.main()
