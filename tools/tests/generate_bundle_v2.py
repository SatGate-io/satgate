"""Deterministic synthetic private bundles. No producer or real private evidence."""
import base64
import copy
import hashlib
import json

import rfc8785
from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PrivateKey
from cryptography.hazmat.primitives.serialization import Encoding, PublicFormat

from tests.generate_decision_v2 import KEY, NOW, b64, commitment, digest, sign as sign_decision
from tests.generate_observation_v2 import seed_observation, sign as sign_observation, result_commitment

OLD = Ed25519PrivateKey.from_private_bytes(bytes(range(32, 64)))
CALL_CAPSULE = b'\x81dummy-opaque-call-capsule'
RESULT_CAPSULE = b'\x82dummy-opaque-result-capsule'


def jwk(key, kid):
    return dict(kty='OKP', crv='Ed25519', use='sig', alg='EdDSA', kid=kid,
                x=b64(key.public_key().public_bytes(Encoding.Raw, PublicFormat.Raw)))


def blob(value):
    raw = value if isinstance(value, bytes) else rfc8785.dumps(value)
    return 'sha256:' + b64(hashlib.sha256(raw).digest()), b64(raw)


def manifest_payload(manifest):
    return {k: v for k, v in manifest.items() if k not in ('manifest_hash', 'signature')}


def sign(bundle, key=KEY):
    m = bundle['manifest']
    raw = manifest_payload(m)
    m.update(manifest_hash=digest(raw), signature='ed25519:' + b64(key.sign(rfc8785.dumps(raw))))
    return bundle


def add_artifact(bundle, record, opening, ciphertext):
    m = bundle['manifest']
    artifact_hash, artifact_bytes = blob(record)
    capsule_hash, capsule_bytes = blob(ciphertext)
    opening_hash, opening_bytes = blob(opening)
    bundle['artifacts'][artifact_hash] = artifact_bytes
    bundle['capsules'][capsule_hash] = capsule_bytes
    bundle['openings'][opening_hash] = opening_bytes
    m['artifacts'].append(artifact_hash)
    m['disclosure'][artifact_hash] = dict(state='disclosed', capsule_digest=capsule_hash, opening_digest=opening_hash)


def seed_bundle(status='accepted', admission='allowed', observation=True):
    record, decision, call, result = seed_observation(status)
    decision['admission'] = admission
    if admission == 'denied':
        decision['approval_disposition'] = 'required_pending'
    sign_decision(decision)
    pins = dict(keys=[jwk(KEY, decision['issuer_kid']), jwk(OLD, 'dummy-history-key')])

    outcome = ('not_expected' if admission == 'denied' else 'pending' if not observation else
               'unknown' if status == 'sent_outcome_unknown' else 'available')
    bundle = dict(schema_version='satgate.receipt_bundle.v2', record_type='private_bundle',
                  manifest=dict(schema_version='satgate.receipt_bundle_manifest.v2', record_type='bundle_manifest',
                                binding_profile='satgate.binding_only.v1', canonicalization='jcs-rfc8785',
                                hash_algorithm='sha256', signature_algorithm='ed25519',
                                issuer=decision['issuer'], issuer_kid=decision['issuer_kid'],
                                generated_at=NOW, range={'from': '2026-11-16T10:00:00Z', 'to': NOW},
                                artifacts=[], disclosure={},
                                outcomes={decision['receipt_hash']: outcome}, retention_state='retained',
                                historical_keys=copy.deepcopy(pins['keys'])),
                  artifacts={}, capsules={}, openings={})
    add_artifact(bundle, decision, call, CALL_CAPSULE)
    if observation:
        record['capsule_digest'] = blob(RESULT_CAPSULE)[0]
        add_artifact(bundle, sign_observation(record), result, RESULT_CAPSULE)
    return sign(bundle), pins, (decision, call, record, result)


def withhold(bundle, index, state):
    m = bundle['manifest']
    entry = m['disclosure'][m['artifacts'][index]]
    del bundle['openings'][entry['opening_digest']]
    entry.update(state=state, opening_digest=None)
    if state != 'withheld':
        del bundle['capsules'][entry['capsule_digest']]
        entry['capsule_digest'] = None
    return sign(bundle)


def replace(bundle, index, record, opening=None):
    m = bundle['manifest']
    old = m['artifacts'][index]
    del bundle['artifacts'][old]
    entry = m['disclosure'].pop(old)
    if opening is not None:
        del bundle['openings'][entry['opening_digest']]
        opening_hash, opening_bytes = blob(opening)
        bundle['openings'][opening_hash] = opening_bytes
        entry['opening_digest'] = opening_hash
    artifact_hash, artifact_bytes = blob(record)
    bundle['artifacts'][artifact_hash] = artifact_bytes
    m['artifacts'][index] = artifact_hash
    m['disclosure'][artifact_hash] = entry
    return sign(bundle)


def leaves(value, path=()):
    if isinstance(value, dict):
        for key, child in value.items():
            yield from leaves(child, (*path, key))
    elif isinstance(value, list):
        for i, child in enumerate(value):
            yield from leaves(child, (*path, i))
    else:
        yield path


def corpus():
    baseline, pins, records = seed_bundle()
    cases = []

    def add(name, bundle=baseline, status='invalid', code='v2_schema_invalid', jwks=pins):
        cases.append(dict(name=name, bundle=copy.deepcopy(bundle), status=status, code=code, jwks=copy.deepcopy(jwks)))

    add('mixed-decisions-and-results', status='verified', code='v2_bundle_verified')
    b = copy.deepcopy(baseline)
    d, call, r, result = copy.deepcopy(records)
    call.update(receipt_id='dummy-identical-receipt', attempt_id='dummy-identical-attempt', opening=b64(bytes(range(32, 64))))

    d.update(receipt_id=call['receipt_id'], call_commitment=dict(d['call_commitment'], digest=commitment(call)))
    sign_decision(d)
    result.update(decision_receipt_hash=d['receipt_hash'], attempt_id=call['attempt_id'], opening=b64(bytes(range(64, 96))))
    r.update(record_id='dummy-identical-result', decision_receipt_hash=d['receipt_hash'], attempt_id=call['attempt_id'],
             call_commitment=d['call_commitment'], result_commitment=dict(r['result_commitment'], digest=result_commitment(result)),
             capsule_digest=blob(RESULT_CAPSULE + b'b')[0])
    sign_observation(r)
    add_artifact(b, d, call, CALL_CAPSULE + b'b')
    add_artifact(b, r, result, RESULT_CAPSULE + b'b')
    b['manifest']['outcomes'][d['receipt_hash']] = 'available'
    add('identical-orders-distinct-attempts', sign(b), 'verified', 'v2_bundle_verified')
    # The exact byte hashes may change if the signer chooses a different JSON
    # encoding, but the decision/observation payload digests do not.
    b = copy.deepcopy(baseline)
    add('signed-exact-non-jcs-artifact-bytes', replace(b, 0, json.dumps(records[0], indent=2).encode()),
        'verified', 'v2_bundle_verified')
    b, _, (decision, call, record, _) = seed_bundle()
    b['manifest']['issuer_kid'] = 'dummy-history-key'
    add('pinned-rotation', sign(b, OLD), 'verified', 'v2_bundle_verified')
    b['manifest']['historical_keys'][1] = jwk(KEY, 'dummy-history-key')
    add('history-cannot-shadow-pin', sign(b, OLD), 'verified', 'v2_bundle_verified')
    add('bundle-key-not-pinned', sign(b, OLD), code='v2_key_required', jwks={'keys': pins['keys'][:1]})
    add('history-only-no-pins', jwks=None, code='v2_key_required')
    add('conflicting-pins', jwks={'keys': pins['keys'] + pins['keys'][:1]}, code='v2_key_required')
    add('held-without-outcome', seed_bundle(admission='denied', observation=False)[0], 'verified', 'v2_bundle_verified')
    add('signed-outcome-pending', seed_bundle(observation=False)[0], 'partial', 'v2_bundle_outcome_pending')
    b = seed_bundle(observation=False)[0]
    b['manifest']['outcomes'][decision['receipt_hash']] = 'unknown'
    add('signed-outcome-missing-unknown', sign(b), 'partial', 'v2_bundle_outcome_unknown')
    add('observed-outcome-unknown', seed_bundle('sent_outcome_unknown')[0], 'partial', 'v2_bundle_outcome_unknown')
    for index, record_type in enumerate(('decision', 'result')):
        for state in ('withheld', 'missing', 'expired', 'deleted'):
            add(record_type + '-opening-' + state, withhold(copy.deepcopy(baseline), index, state),
                'partial', 'v2_bundle_opening_' + state)
    b = withhold(withhold(copy.deepcopy(baseline), 0, 'withheld'), 1, 'withheld')
    add('all-openings-withheld-retained-ciphertexts', b, 'partial', 'v2_attempt_not_checked')
    bad = dict(record, capsule_digest=blob(b'dummy-wrong-ciphertext')[0])
    add('withheld-opening-does-not-excuse-wrong-ciphertext', replace(b, 1, sign_observation(bad)),
        code='v2_bundle_capsule_mismatch')
    for state in ('expired', 'deleted'):
        b = copy.deepcopy(baseline)
        b['manifest']['retention_state'] = state
        add('retention-' + state, sign(b), 'partial', 'v2_bundle_retention_' + state)

    for field in baseline:
        b = copy.deepcopy(baseline)
        del b[field]
        add('missing-bundle-' + field, b)
    for field in baseline['manifest']:
        b = copy.deepcopy(baseline)
        del b['manifest'][field]
        add('missing-manifest-' + field, b if field in ('manifest_hash', 'signature') else sign(b))
    for tag, field, value in [('future', 'schema_version', 'satgate.receipt_bundle.v99'),
                              ('downgrade', 'schema_version', 'satgate.evidence_pack.v1'),
                              ('type', 'record_type', 'unknown'), ('extra', 'extra', 'dummy-private')]:
        b = copy.deepcopy(baseline)
        b[field] = value
        add('closed-bundle-' + tag, b)
    for field, value in [('schema_version', 'satgate.receipt_bundle_manifest.v99'), ('record_type', 'unknown'),
                         ('binding_profile', 'unknown'), ('canonicalization', 'json'), ('hash_algorithm', 'sha512'),
                         ('signature_algorithm', 'rsa'), ('retention_state', 'unknown'), ('extra', 'dummy-private')]:
        b = copy.deepcopy(baseline)
        b['manifest'][field] = value
        add('closed-manifest-' + field, sign(b))
    for field in ('signature', 'manifest_hash'):
        b = copy.deepcopy(baseline)
        b['manifest'][field] += '='
        add('padded-' + field, b)
    b = copy.deepcopy(baseline)
    b['manifest']['manifest_hash'] = digest({})
    add('bad-manifest-digest', b, code='v2_hash_mismatch')
    b = copy.deepcopy(baseline)
    b['manifest']['signature'] = 'ed25519:' + b64(bytes(64))
    add('bad-manifest-signature', b, code='v2_signature_invalid')
    for field, change in [('range', {'from': '2026-11-16T10:01:00Z', 'to': NOW}),
                          ('retention_state', 'expired'), ('outcomes', {decision['receipt_hash']: 'unknown'}),
                          ('historical_keys', []), ('artifacts', list(reversed(baseline['manifest']['artifacts']))),
                          ('disclosure', withhold(copy.deepcopy(baseline), 0, 'missing')['manifest']['disclosure'])]:
        b = copy.deepcopy(baseline)
        b['manifest'][field] = change
        b['manifest']['manifest_hash'] = digest(manifest_payload(b['manifest']))
        add('unsigned-change-' + field, b, code='v2_signature_invalid')
    for path in leaves(manifest_payload(baseline['manifest'])):
        b = copy.deepcopy(baseline)
        target = b['manifest']
        for part in path[:-1]:
            target = target[part]
        target[path[-1]] = 'dummy-leaf-tamper'
        b['manifest']['manifest_hash'] = digest(manifest_payload(b['manifest']))
        add('rehashed-signed-leaf-' + '/'.join(map(str, path)), b, code=None)

    for field, code in [('artifacts', 'v2_bundle_artifact_mismatch'), ('capsules', 'v2_bundle_capsule_mismatch'),
                        ('openings', 'v2_bundle_opening_mismatch')]:
        for tamper in ('missing', 'extra', 'changed', 'same-json-different-bytes', 'invalid-base64'):
            b = copy.deepcopy(baseline)
            key = next(iter(b[field]))
            if tamper == 'missing':
                del b[field][key]
            elif tamper == 'extra':
                new_hash, new_bytes = blob({'dummy': 'dummy-unlisted'})
                b[field][new_hash] = new_bytes
            elif tamper == 'same-json-different-bytes':
                raw = base64.urlsafe_b64decode(b[field][key] + '=' * (-len(b[field][key]) % 4))
                b[field][key] = b64(raw + b'\n')
            else:
                b[field][key] = b64(b'{}') if tamper == 'changed' else '!'
            add(field + '-' + tamper, b, code=code)

    for field, expected in [('schema_version', 'v2_schema_invalid'), ('record_type', 'v2_schema_invalid'),
                            ('binding_profile', 'v2_schema_invalid'), ('issuer', 'v2_bundle_record_mismatch'),
                            ('decision_receipt_hash', 'v2_parent_required'), ('attempt_id', 'v2_parent_mismatch'),
                            ('capsule_digest', 'v2_bundle_capsule_mismatch')]:
        b = copy.deepcopy(baseline)
        r = copy.deepcopy(record)
        r[field] = {'issuer': 'https://other.example', 'decision_receipt_hash': digest({}),
                    'capsule_digest': digest({})}.get(field, 'unknown')
        add('record-disagrees-' + field, replace(b, 1, sign_observation(r)), code=expected)
    for index, signed in enumerate((decision, record)):
        for field in ('signature', 'binding_profile'):
            b = copy.deepcopy(baseline)
            r = copy.deepcopy(signed)
            del r[field]
            add(f'record-{index}-missing-{field}', replace(b, index, r))
        b = copy.deepcopy(baseline)
        r = dict(signed, signature='ed25519:' + b64(bytes(64)))
        add(f'recorded-bad-signature-{index}', replace(b, index, r), code='v2_signature_invalid')

    b = copy.deepcopy(baseline)
    b['manifest']['range']['from'] = '2026-11-16T11:00:30Z'
    add('signed-range-excludes-decision', sign(b), code='v2_bundle_record_mismatch')
    b['manifest']['range']['from'] = '2026-11-16T11:00:00.000000001Z'
    add('signed-range-excludes-decision-submicrosecond', sign(b), code='v2_bundle_record_mismatch')
    b['manifest']['range'] = {'from': '2026-11-16T11:00:00Z', 'to': '2026-11-16T11:01:00.000000001Z'}
    add('signed-range-includes-observation-submicrosecond', sign(b), 'verified', 'v2_bundle_verified')
    b['manifest']['range']['to'] = '2026-11-16T11:01:00Z'
    add('signed-range-excludes-end', sign(b), code='v2_bundle_record_mismatch')
    for value in ('pending', 'unknown', 'not_expected', None):
        b = copy.deepcopy(baseline)
        if value is None:
            b['manifest']['outcomes'].clear()
        else:
            b['manifest']['outcomes'][decision['receipt_hash']] = value
        add('signed-outcome-disagrees-' + str(value), sign(b), code='v2_bundle_outcome_mismatch')
    for field, value in [('state', 'unknown'), ('capsule_digest', None), ('opening_digest', None), ('extra', True)]:
        b = copy.deepcopy(baseline)
        b['manifest']['disclosure'][b['manifest']['artifacts'][0]][field] = value
        add('closed-disclosure-' + field, sign(b))
    b = copy.deepcopy(baseline)
    m = b['manifest']
    m['disclosure'].pop(m['artifacts'][0])
    add('signed-disclosure-missing-entry', sign(b), code='v2_bundle_disclosure_mismatch')
    b = copy.deepcopy(baseline)
    m = b['manifest']
    m['artifacts'].append(m['artifacts'][0])
    add('signed-duplicate-artifact-digest', sign(b), code='v2_bundle_artifact_mismatch')
    b = copy.deepcopy(baseline)
    m = b['manifest']
    m['disclosure'][m['artifacts'][1]] = copy.deepcopy(m['disclosure'][m['artifacts'][0]])
    add('signed-reused-capsule-digest', sign(b), code='v2_bundle_disclosure_mismatch')

    d, call, r, opening = records
    for path in [('opening',), ('account_binding',), ('route_binding',), ('attempt_id',), ('operation_id',), ('receipt_id',),
                 ('transport', 'tool'), ('transport', 'service'), ('transport', 'method'), ('transport', 'path'),
                 ('transport', 'body_b64'), ('transport', 'params_b64'), ('transport', 'query_b64'), ('transport', 'headers_b64'),
                 ('normalized_intent', 'cancel_target'), ('normalized_intent', 'replacement_link'),
                 ('normalized_intent', 'quantity'), ('normalized_intent', 'price'), ('normalized_intent', 'quantity_unit'),
                 ('normalized_intent', 'price_unit'), ('policy_projection', 'default_time_in_force')]:
        c = copy.deepcopy(call)
        target = c if len(path) == 1 else c[path[0]]
        target[path[-1]] = {'opening': b64(bytes(32)), 'body_b64': b64(b'{ "dummy": 2 }'),
                            'params_b64': b64(b'{"arguments":{}}'), 'headers_b64': b64(b'accept:application/json'),
                            'query_b64': b64(b'b=2&a=1'), 'quantity': '5', 'price': '9'}.get(path[-1], 'dummy-changed')
        add('changed-call-' + '/'.join(path), replace(copy.deepcopy(baseline), 0, d, c), code='v2_call_mismatch')
    for field, value in [('opening', b64(bytes(32))), ('broker_order_id', 'dummy-private-other-order'),
                         ('account_binding', 'dummy-account-other'), ('attempt_id', 'dummy-attempt-other'),
                         ('target_order_id', 'dummy-target-other'), ('replacement_link', 'dummy-replacement-other')]:
        for resig in (False, True):
            p = dict(opening, **{field: value})
            updated = dict(r, result_commitment=dict(r['result_commitment']))
            if resig:
                if field in ('broker_order_id', 'opening'):
                    continue  # Authentically signed data without a contradictory reference.
                updated['result_commitment']['digest'] = result_commitment(p)
            b = replace(copy.deepcopy(baseline), 1, sign_observation(updated), p)
            add(f'changed-result-{field}-recommit-{resig}', b,
                code='v2_result_link_mismatch' if resig else 'v2_result_mismatch')
    b = copy.deepcopy(baseline)
    p = dict(opening, opening=call['opening'])
    updated = dict(r, result_commitment=dict(r['result_commitment'], digest=result_commitment(p)))
    add('shared-opening-entropy', replace(b, 1, sign_observation(updated), p), code='v2_result_opening_invalid')
    b = copy.deepcopy(baseline)
    p = dict(opening, broker_order_id='dummy-private-other-order')
    replace(b, 1, r, p)
    add('tampered-result-even-without-parent-opening', withhold(b, 0, 'withheld'), code='v2_result_mismatch')
    for unsafe in (b'{"x":9007199254740992}', b'{"x":1e400}', b'{"x":"\\ud800"}', b'{"x":1,"x":1}', b'{"x":1,"X":1}', b'{"x":"\xff"}'):
        suffix = unsafe.hex()
        add('invalid-artifact-json-' + suffix, replace(copy.deepcopy(baseline), 0, unsafe))
        add('invalid-opening-json-' + suffix, replace(copy.deepcopy(baseline), 0, d, unsafe), code='v2_bundle_opening_mismatch')
    c = copy.deepcopy(call)
    c['transport']['params_b64'] = b64(b'{"meta":{"token":"dummy-private-credential"}}')
    bad = copy.deepcopy(d)
    bad['call_commitment']['digest'] = commitment(c)
    add('signed-private-credential-in-capsule', replace(copy.deepcopy(baseline), 0, sign_decision(bad), c), code='v2_opening_invalid')
    for canary_field in ('_meta', 'broker_order_id', 'symbol', 'amount', 'tenant_id', 'actor_email', 'request_path'):
        r_bad = dict(r, **{canary_field: 'dummy-private-canary'})
        add('private-field-in-public-observation-' + canary_field,
            replace(copy.deepcopy(baseline), 1, sign_observation(r_bad)))
    return dict(now=NOW, jwks=pins, vectors=cases)


if __name__ == '__main__':
    print(json.dumps(corpus(), separators=(',', ':'), sort_keys=True))
