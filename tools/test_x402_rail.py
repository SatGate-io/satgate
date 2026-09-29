"""Shared signed x402 (USDC on Base) corpus for both distributed verifier entry points.

Synthetic keys and addresses only. Mirrors tools/test_paid_rail.py: every case
is signed, then checked through the API and the CLI of tools/ and demo/.
Run: python -B -m unittest discover -s tools -p 'test_*.py' -v
"""
import copy
import json
import tempfile
import unittest
from pathlib import Path

from test_paid_rail import VERIFIERS, PaidRailTests

TX = '0x' + 'ab' * 32
PAYER = '0x70997970c51812dc3a010c7d01b50e0d17dc79c8'
PAY_TO = '0x131c044d0427216db8c4d24aa9923eca29a6e3f0'
USDC = {
    'eip155:8453': '0x833589fcd6edb6e08f4c7c32d4f71b54bda02913',
    'eip155:84532': '0x036cbd53842c5426634e7929541ec2318f3dcf7e',
}
FIELDS = ('rail', 'transaction', 'network', 'asset', 'payer', 'pay_to', 'amount_atomic')


def x402_receipt(f, network='eip155:84532', amount=10000, decision='allowed'):
    r = f.receipt(decision)
    for k in ('payment_hash', 'invoice_hash', 'macaroon_hash', 'amount_sats'):
        r.pop(k, None)
    r.update(rail='x402', transaction=TX, network=network, asset=USDC[network],
             payer=PAYER, pay_to=PAY_TO, amount_atomic=amount)
    r['paid_rail_context'] = {k: r[k] for k in FIELDS}
    r['budget'] = dict(spend_mode='paid_rail', rail='x402', amount_atomic=amount)
    # No macaroon is presented on x402: the authority is the settled authorization.
    r['authority'] = {'provenance_level': 'settled_x402_authorization'}
    return r


def set_field(r, key, value):
    r[key] = value
    r['paid_rail_context'][key] = copy.deepcopy(value)


def set_amount(r, value):
    set_field(r, 'amount_atomic', value)
    r['budget']['amount_atomic'] = copy.deepcopy(value)


def x402_corpus(f):
    for network in USDC:
        for decision in ('allowed', 'paid'):
            for amount in (1, 10000, 2**53 - 1):
                yield (f'{network}-{decision}-{amount}',
                       f.pack(x402_receipt(f, network, amount, decision)), True)
    mutations = []
    for value in (True, False, 0, -1, 1.0, 10000.0, '10000', None, 2**53):
        mutations.append((f'amount-{value!r}', lambda r, v=value: set_amount(r, v)))
    mutations += [
        ('unknown-network', lambda r: set_field(r, 'network', 'eip155:1')),
        ('wrong-asset', lambda r: set_field(r, 'asset', '0x' + '00' * 19 + 'ad')),
        ('mainnet-asset-on-sepolia', lambda r: set_field(r, 'asset', USDC['eip155:8453'])),
        ('uppercase-payer', lambda r: set_field(r, 'payer', PAYER.upper().replace('0X', '0x'))),
        ('short-pay-to', lambda r: set_field(r, 'pay_to', '0x1234')),
        ('short-tx', lambda r: set_field(r, 'transaction', '0x1234')),
        ('missing-tx', lambda r: (r.pop('transaction'), r['paid_rail_context'].pop('transaction'))),
        ('context-pay-to-mismatch', lambda r: r['paid_rail_context'].update(pay_to='0x' + '00' * 19 + 'ad')),
        ('context-extra', lambda r: r['paid_rail_context'].update(extra=True)),
        ('context-none', lambda r: r.update(paid_rail_context=None)),
        ('budget-amount-mismatch', lambda r: r['budget'].update(amount_atomic=9999)),
        ('l402-budget', lambda r: r.update(budget=dict(spend_mode='paid_rail', rail='l402', amount_sats=10))),
        ('extra-l402-field', lambda r: r.update(payment_hash='a' * 64)),
        ('macaroon-authority', lambda r: r.update(authority={'provenance_level': 'verified_macaroon_caveats'})),
        ('extra-authority-field', lambda r: r['authority'].update(token_id_hash='sha256:x')),
        ('capability-hash', lambda r: r.update(capability_hash='sha256:capability')),
        ('credit-field', lambda r: r['metadata'].update(nested=[{'deep': {'amount_usd': 0.01}}])),
        ('wrong-rail', lambda r: (set_field(r, 'rail', 'other'), r['budget'].update(rail='other'))),
        ('denied', lambda r: r.update(decision='denied')),
    ]
    for label, mutate in mutations:
        r = x402_receipt(f)
        mutate(r)
        yield label, f.pack(r), False


class X402PaidRailTests(unittest.TestCase):
    """Borrows PaidRailTests' fixture and verify/cli helpers without rerunning its L402 tests."""

    setUp = PaidRailTests.setUp
    verify = PaidRailTests.verify
    cli = PaidRailTests.cli

    def test_x402_signed_corpus(self):
        for label, p, expected in x402_corpus(self.f):
            for surface in VERIFIERS:
                with self.subTest(case=label, surface=surface):
                    result = self.verify(surface, p)
                    self.assertTrue(result['checks']['signature_valid'], result)
                    self.assertTrue(result['trusted_issuer_valid'], result)
                    self.assertIs(result['valid'], expected, result)
                    if not expected:
                        self.assertTrue(result['reason_codes'])

    def test_x402_signed_cli_corpus(self):
        with tempfile.TemporaryDirectory() as td:
            jwks = Path(td) / 'synthetic-jwks.json'
            jwks.write_text(json.dumps(self.f.jwks))
            for label, p, expected in x402_corpus(self.f):
                for surface in VERIFIERS:
                    with self.subTest(case=label, surface=surface):
                        _, result = self.cli(surface, json.dumps(p), '--jwks-file', str(jwks),
                                             '--require-trusted-issuer')
                        self.assertIs(result['valid'], expected, result)

    def test_l402_receipt_cannot_borrow_x402_authority(self):
        r = self.f.receipt()
        r['authority'] = {'provenance_level': 'settled_x402_authorization'}
        p = self.f.pack(r)
        for surface in VERIFIERS:
            with self.subTest(surface=surface):
                self.assertFalse(self.verify(surface, p)['valid'])

    def test_x402_pack_with_budget_state_fails(self):
        p = self.f.pack(x402_receipt(self.f))
        p['budget_state'] = {'spend_mode': 'paid_rail', 'rail': 'x402', 'amount_atomic': 10000}
        for surface in VERIFIERS:
            with self.subTest(surface=surface):
                result = self.verify(surface, p)
                self.assertFalse(result['valid'])
                self.assertIn('invalid_paid_rail_provenance', result['reason_codes'])

    def test_x402_pack_mirror_mismatch_fails(self):
        p = self.f.pack(x402_receipt(self.f))
        p['budget'] = {'spend_mode': 'paid_rail', 'rail': 'x402', 'amount_atomic': 1}
        for surface in VERIFIERS:
            with self.subTest(surface=surface):
                self.assertFalse(self.verify(surface, p)['valid'])


if __name__ == '__main__':
    unittest.main()
