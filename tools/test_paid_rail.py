import copy
import unittest
from datetime import datetime, timezone
from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PrivateKey
from cryptography.hazmat.primitives.serialization import Encoding, PublicFormat
import verify_evidence_pack as v


class PaidRailTests(unittest.TestCase):
    def setUp(self):
        self.key = Ed25519PrivateKey.generate()
        x = v.b64url_encode(self.key.public_key().public_bytes(Encoding.Raw, PublicFormat.Raw))
        self.jwks = {'keys': [{'kid': 'test-key', 'kty': 'OKP', 'crv': 'Ed25519', 'alg': 'EdDSA', 'use': 'sig', 'x': x}]}
        self.receipt = dict(schema_version='satgate.receipt.v1', canonicalization='jcs-rfc8785', hash_algorithm='sha256', signature_algorithm='ed25519', issuer='https://test.example', issuer_kid='test-key', decision='allowed', decision_reason='payment_verified', issued_at=datetime.now(timezone.utc).isoformat(), evidence_pack_id='test-pack', receipt_id='test-receipt', rail='l402', amount_sats=100, payment_hash='a'*64, invoice_hash='b'*64, macaroon_hash='c'*64)
        self.receipt['paid_rail_context'] = {k: self.receipt[k] for k in ('rail', 'amount_sats', 'payment_hash', 'invoice_hash', 'macaroon_hash')}
        self.receipt['budget'] = dict(spend_mode='paid_rail', rail='l402', amount_sats=100)

    def pack(self, receipt):
        receipt = copy.deepcopy(receipt)
        receipt['receipt_hash'] = v.sha256_receipt_hash(receipt)
        receipt['signature'] = 'ed25519:' + v.b64url_encode(self.key.sign(v.canonical_receipt_payload(receipt)))
        pack = {k: receipt[k] for k in ('issuer', 'decision', 'decision_reason', 'evidence_pack_id', 'receipt_id', 'receipt_hash', 'budget')}
        pack.update(schema_version='satgate.evidence_pack.v1', receipts=[receipt])
        return pack

    def verify(self, pack):
        return v.verify_pack(pack, jwks=self.jwks, require_trusted_issuer=True)

    def test_positive(self):
        for decision in ('allowed', 'paid'):
            self.receipt['decision'] = decision
            result = self.verify(self.pack(self.receipt))
            self.assertTrue(result['valid'], result)
            self.assertTrue(result['trusted_issuer_valid'])

    def test_signed_adverse_receipts(self):
        mutations = []
        for field in ('payment_hash', 'invoice_hash', 'macaroon_hash'):
            mutations += [lambda r, f=field: r.pop(f), lambda r, f=field: r['paid_rail_context'].update({f: 'wrong'})]
        for amount in (True, 0, -1):
            mutations.append(lambda r, a=amount: r.update(amount_sats=a))
        mutations += [lambda r: r.update(rail='other'), lambda r: r.update(decision='denied'), lambda r: r.update(metadata={'nested': [{'currency': 'USD'}]}), lambda r: r['budget'].update(amount_sats=True)]
        for mutate in mutations:
            with self.subTest(mutation=mutate):
                receipt = copy.deepcopy(self.receipt)
                mutate(receipt)
                result = self.verify(self.pack(receipt))
                self.assertTrue(result['checks']['signature_valid'])
                self.assertFalse(result['valid'])

    def test_pack_contradictions(self):
        for field, value in [('budget_state', None), ('budget_state', {}), ('metadata', {'cost_credits': 1}), ('budget', {}), ('authority', {'unexpected': True})]:
            with self.subTest(field=field, value=value):
                pack = self.pack(self.receipt)
                pack[field] = value
                self.assertFalse(self.verify(pack)['valid'])

    def test_tampering_and_wrong_key(self):
        pack = self.pack(self.receipt)
        pack['receipts'][0]['amount_sats'] = 101
        self.assertFalse(self.verify(pack)['valid'])
        self.jwks['keys'][0]['x'] = v.b64url_encode(Ed25519PrivateKey.generate().public_key().public_bytes(Encoding.Raw, PublicFormat.Raw))
        self.assertFalse(self.verify(self.pack(self.receipt))['valid'])


if __name__ == '__main__':
    unittest.main()
