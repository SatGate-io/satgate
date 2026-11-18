"""Deterministic, dummy-only v2 reader corpus; no production signing path."""
import base64
import copy
import hashlib
import json
from pathlib import Path
import sys

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / '_vendor'))
import rfc8785
from cryptography.hazmat.primitives import serialization
from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PrivateKey

KEY = Ed25519PrivateKey.from_private_bytes(bytes(range(32)))
NOW = '2026-11-16T12:00:00Z'


def b64(value):
    return base64.urlsafe_b64encode(value).decode().rstrip('=')


def digest(value):
    return 'sha256:' + b64(hashlib.sha256(rfc8785.dumps(value)).digest())


def commitment(opening):
    return digest(dict(domain='satgate.exact_call.v1', **{
        k: v for k, v in opening.items() if k != 'schema_version'}))


def sign(receipt):
    payload = {k: v for k, v in receipt.items() if k not in ('receipt_hash', 'signature')}
    receipt['receipt_hash'] = digest(payload)
    receipt['signature'] = 'ed25519:' + b64(KEY.sign(rfc8785.dumps(payload)))
    return receipt


def seed():
    params = {'name': 'dummy_order', 'arguments': {'quantity': '2', 'account': 'dummy-account'},
              '_meta': {'meaningful': 'dummy-metadata'}}
    opening = {
        'schema_version': 'satgate.exact_call.opening.v1',
        'receipt_id': 'dummy-receipt-a', 'attempt_id': 'dummy-attempt-a',
        'operation_id': 'dummy-operation-a', 'authority_binding': 'dummy-authority',
        'account_binding': 'dummy-account', 'route_binding': 'dummy-route',
        'adapter_version': 'dummy-adapter-v1', 'opening': b64(bytes(range(32, 64))),
        'transport': {'profile': 'satgate.binding_only.transport.v1',
                      'service': 'dummy-service', 'tool': 'dummy_order', 'method': 'tools/call',
                      'path': '/dummy', 'query_b64': b64(b'a=1&b=2'),
                      'headers_b64': b64(b'content-type:application/json\n'),
                      'body_b64': b64(json.dumps(params, separators=(',', ':')).encode()),
                      'params_b64': b64(json.dumps(params, separators=(',', ':')).encode())},
        'normalized_intent': {'operation': 'create', 'quantity': '2', 'quantity_unit': 'dummy-units',
                              'price': '1.25', 'price_unit': 'dummy-currency',
                              'time_in_force': 'day', 'cancel_target': None, 'replacement_link': None},
        'policy_projection': {'credit_charge': {'evaluation_state': 'evaluated',
                                                'amount': '1', 'unit': 'dummy-credit'},
                              'default_time_in_force': 'day'},
    }
    receipt = {
        'schema_version': 'satgate.receipt.v2', 'record_type': 'decision', 'phase': 'pre_dispatch',
        'binding_profile': 'satgate.binding_only.v1', 'canonicalization': 'jcs-rfc8785',
        'hash_algorithm': 'sha256', 'signature_algorithm': 'ed25519',
        'receipt_id': opening['receipt_id'], 'issuer': 'https://issuer.example',
        'issuer_kid': 'dummy-reader-key', 'issued_at': '2026-11-16T11:00:00Z',
        'enforcement_mode': 'control', 'admission_revision': 'dummy-revision',
        'authority_evidence': 'not_evidenced', 'authorization_record_hash': None,
        'effective_limits_record_hash': None, 'approval_record_hash': None,
        'approval_disposition': 'not_required', 'admission': 'allowed', 'dispatch': 'not_sent',
        'call_commitment': {'profile': 'satgate.exact_call.v1', 'algorithm': 'sha256',
                            'digest': commitment(opening)},
    }
    return sign(receipt), opening


def corpus():
    receipt, opening = seed()
    cases = []

    def add(name, r=receipt, o=opening, valid=False, code=None, jwks=None):
        cases.append({'name': name, 'receipt': copy.deepcopy(r), 'opening': copy.deepcopy(o),
                      'valid': valid, 'code': code, **({'jwks': jwks} if jwks is not None else {})})

    add('valid-v2-opening', valid=True, code='v2_call_matched')
    add('valid-v2-unopened', o=None, valid=True, code='v2_call_not_checked')
    held = copy.deepcopy(receipt)
    held.update(admission='denied', approval_disposition='required_pending')
    add('valid-held', sign(held), valid=True, code='v2_hold_not_executed')
    denied = copy.deepcopy(receipt)
    denied.update(admission='denied')
    add('valid-denied', sign(denied), valid=True, code='v2_call_matched')
    other = copy.deepcopy(opening)
    other.update(receipt_id='dummy-receipt-b', attempt_id='dummy-attempt-b', opening=b64(bytes(range(64, 96))))
    second = copy.deepcopy(receipt)
    second.update(receipt_id=other['receipt_id'])
    second['call_commitment']['digest'] = commitment(other)
    add('valid-identical-order-new-attempt', sign(second), other, valid=True, code='v2_call_matched')
    for operation in ('amend', 'cancel'):
        o = copy.deepcopy(opening)
        o['normalized_intent'].update(operation=operation, cancel_target='dummy-order-target',
                                      replacement_link='dummy-prior-attempt')
        r = copy.deepcopy(receipt)
        r['call_commitment']['digest'] = commitment(o)
        add('valid-' + operation, sign(r), o, valid=True, code='v2_call_matched')
    o = copy.deepcopy(opening)
    o['policy_projection']['credit_charge'] = dict(evaluation_state='not_evaluated', amount=None, unit=None)
    r = copy.deepcopy(receipt)
    r['call_commitment']['digest'] = commitment(o)
    add('valid-charge-not-evaluated', sign(r), o, valid=True, code='v2_call_matched')
    hidden = copy.deepcopy(receipt)
    hidden['call_commitment']['digest'] = digest({'dummy': 'undisclosed'})
    add('valid-unopened-no-order-claim', sign(hidden), None, valid=True, code='v2_call_not_checked')
    o = copy.deepcopy(opening)
    o['opening'] = 'stk_' + 'A' * 39  # Canonical 32-byte dummy salt, not a credential.
    r = copy.deepcopy(receipt)
    r['call_commitment']['digest'] = commitment(o)
    add('valid-entropy-marker', sign(r), o, valid=True, code='v2_call_matched')

    def signed_leaves(value, path=()):
        if isinstance(value, dict):
            for key, child in value.items():
                if not path and key in ('receipt_hash', 'signature'):
                    continue
                yield from signed_leaves(child, path + (key,))
        else:
            yield path

    for path in signed_leaves(receipt):
        r = copy.deepcopy(receipt)
        target = r
        for key in path[:-1]:
            target = target[key]
        old = target[path[-1]]
        target[path[-1]] = 'dummy-tampered' if old is None else str(old) + '-tampered'
        r['receipt_hash'] = digest({k: v for k, v in r.items() if k not in ('receipt_hash', 'signature')})
        add('signed-leaf/' + '/'.join(path), r)
    for field in list(receipt):
        r = copy.deepcopy(receipt)
        del r[field]
        add('missing/' + field, sign(r) if field not in ('receipt_hash', 'signature') else r,
            code='v2_schema_invalid')
    for field, value in (
        ('schema_version', 'satgate.receipt.v99'), ('record_type', 'execution_observation'),
        ('phase', 'post_dispatch'), ('binding_profile', 'satgate.enrolled.v1'),
        ('canonicalization', 'json'), ('hash_algorithm', 'sha512'), ('signature_algorithm', 'rsa'),
        ('enforcement_mode', 'observe'), ('authority_evidence', 'verified'),
        ('authorization_record_hash', digest({})), ('effective_limits_record_hash', digest({})),
        ('approval_record_hash', digest({})), ('admission', 'executed'), ('dispatch', 'sent'),
        ('approval_disposition', 'consumed'), ('extra', 'dummy-private-canary'),
        ('issued_at', 'not-a-time'), ('issuer', 'http://issuer.example')):
        r = copy.deepcopy(receipt)
        r[field] = value
        add('closed/' + field, sign(r), code='v2_schema_invalid')
    for field in ('profile', 'algorithm', 'digest'):
        r = copy.deepcopy(receipt)
        r['call_commitment'][field] = 'unknown'
        add('closed/commitment-' + field, sign(r), code='v2_schema_invalid')
    r = copy.deepcopy(held)
    r['admission'] = 'allowed'
    add('held-as-allowed', sign(r), code='v2_schema_invalid')
    for field in ('receipt_id', 'issuer', 'issuer_kid', 'issued_at', 'admission_revision'):
        r = copy.deepcopy(receipt)
        r[field] += '-tampered'
        add('signed-tamper/' + field, r)
    for field, value in (('digest', digest({'dummy': 'tamper'})),):
        r = copy.deepcopy(receipt)
        r['call_commitment'][field] = value
        # Recompute the hash without resigning: unsigned mirrors cannot rescue it.
        r['receipt_hash'] = digest({k: v for k, v in r.items() if k not in ('receipt_hash', 'signature')})
        add('signed-tamper/nested-' + field, r, code='v2_signature_invalid')
    for field in ('receipt_hash', 'signature'):
        r = copy.deepcopy(receipt)
        r[field] = ('sha256:' + b64(bytes(32))) if field == 'receipt_hash' else 'ed25519:' + b64(bytes(64))
        add('tamper/' + field, r, code='v2_hash_mismatch' if field == 'receipt_hash' else 'v2_signature_invalid')
        r = copy.deepcopy(receipt)
        r[field] += '='
        add('padded/' + field, r, code='v2_schema_invalid')
    for field in ('receipt_id', 'attempt_id', 'operation_id', 'authority_binding', 'account_binding',
                  'route_binding', 'adapter_version', 'opening'):
        o = copy.deepcopy(opening)
        o[field] = b64(bytes(32)) if field == 'opening' else 'dummy-wrong'
        add('opening-tamper/' + field, o=o, code='v2_call_mismatch')
    for group in ('transport', 'normalized_intent', 'policy_projection'):
        for field in opening[group]:
            o = copy.deepcopy(opening)
            if field == 'profile':
                value = 'unknown'
            elif field.endswith('_b64'):
                value = b64(b'{"dummy":"changed"}')
            elif field == 'credit_charge':
                value = dict(o[group][field], unit='wrong-unit')
            elif field in ('quantity', 'price'):
                value = '3'
            else:
                value = 'dummy-wrong'
            o[group][field] = value
            add('opening-tamper/' + group + '/' + field, o=o)
    for field in opening:
        o = copy.deepcopy(opening)
        del o[field]
        add('opening-missing/' + field, o=o, code='v2_opening_invalid')
    for name, change in (
        ('unknown-economic-field', lambda o: o['normalized_intent'].update(unknown_fee='1')),
        ('ambiguous-quantity', lambda o: o['normalized_intent'].update(quantity=2)),
        ('ambiguous-default', lambda o: o['normalized_intent'].update(time_in_force=None)),
        ('short-entropy', lambda o: o.update(opening=b64(bytes(31)))),
        ('padded-entropy', lambda o: o.update(opening=o['opening'] + '=')),
        ('case-collision', lambda o: o.update(Receipt_ID='dummy')),
        ('invalid-unicode', lambda o: o.update(account_binding='\ud800')),
        ('unsafe-integer', lambda o: o['transport'].update(body_b64=b64(b'{"n":9007199254740992}'))),
        ('duplicate-params', lambda o: o['transport'].update(params_b64=b64(b'{"name":"a","name":"b"}'))),
        ('reserved-case-params', lambda o: o['transport'].update(params_b64=b64(b'{"name":"a","Name":"b"}'))),
        ('duplicate-body', lambda o: o['transport'].update(body_b64=b64(b'{"x":1,"x":2}'))),
        ('credential-header', lambda o: o['transport'].update(headers_b64=b64(b'Authorization: dummy-credential'))),
        ('credential-params', lambda o: o['transport'].update(params_b64=b64(b'{"token":"dummy-credential"}'))),
    ):
        o = copy.deepcopy(opening)
        change(o)
        add('malformed/' + name, o=o, code='v2_opening_invalid')
    add('missing-key', jwks={'keys': []}, code='v2_key_required')
    x = b64(KEY.public_key().public_bytes(serialization.Encoding.Raw, serialization.PublicFormat.Raw))
    good_key = {'kty': 'OKP', 'crv': 'Ed25519', 'alg': 'EdDSA', 'kid': 'dummy-reader-key', 'x': x}
    for name, keys in (
        ('wrong-key', [dict(good_key, x=b64(bytes(32)))]),
        ('wrong-kid', [dict(good_key, kid='dummy-wrong-key')]),
        ('duplicate-kid', [good_key, good_key]),
        ('wrong-key-algorithm', [dict(good_key, alg='RS256')]),
        ('wrong-key-use', [dict(good_key, use='enc')]),
        ('padded-key', [dict(good_key, x=x + '=')]),
    ):
        add(name, jwks={'keys': keys}, code='v2_signature_invalid' if name == 'wrong-key' else 'v2_key_required')
    return {'now': NOW, 'jwks': {'keys': [{'kty': 'OKP', 'crv': 'Ed25519', 'alg': 'EdDSA',
                                         'kid': 'dummy-reader-key', 'x': x}]}, 'vectors': cases}


if __name__ == '__main__':
    print(json.dumps(corpus(), indent=2, sort_keys=True))
