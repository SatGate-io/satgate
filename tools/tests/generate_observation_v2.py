"""Dummy-only deterministic signed execution-observation vectors; never a producer."""
import copy
import json
from tests.generate_decision_v2 import KEY, NOW, b64, digest, seed, sign as sign_decision, commitment
import rfc8785

STATUSES = ('accepted', 'rejected', 'transport_failed_before_send', 'sent_outcome_unknown',
            'partially_filled', 'filled', 'cancelled', 'later_reconciled')


def sign(record):
    raw = {k: v for k, v in record.items() if k not in ('record_hash', 'signature')}
    record['record_hash'] = digest(raw)
    record['signature'] = 'ed25519:' + b64(KEY.sign(rfc8785.dumps(raw)))
    return record


def result_commitment(opening):
    return digest(dict(domain='satgate.execution_result.v1', **{
        k: v for k, v in opening.items() if k != 'schema_version'}))


def seed_observation(status='accepted'):
    parent, call = seed()
    private = dict(
        schema_version='satgate.execution_result.opening.v1',
        decision_receipt_hash=parent['receipt_hash'], operation_id=call['operation_id'],
        attempt_id=call['attempt_id'], opening=b64(bytes(range(96, 128))), status=status,
        account_binding=call['account_binding'], broker_order_id='dummy-private-order',
        client_order_id='dummy-private-client', venue_status='dummy-venue-status',
        venue_times={'received_at': '2026-11-16T11:01:00Z', 'updated_at': None},
        parser_version='dummy-parser-v1', target_order_id=None, replacement_link=None,
        financial_values={'quantity': '2', 'quantity_unit': 'dummy-units',
                          'price': '1.25', 'price_unit': 'dummy-currency'},
        response={'profile': 'satgate.binding_only.response.v1',
                  'headers_b64': b64(b'content-type:application/json\n'),
                  'body_b64': b64(b'{"id":"dummy-private-order","quantity":"2"}')})
    if status in ('transport_failed_before_send', 'sent_outcome_unknown'):
        for field in ('broker_order_id', 'client_order_id', 'venue_status'):
            private[field] = None
        private['response'] = None
        private['venue_times'] = dict(received_at=None, updated_at=None)
        private['financial_values'] = dict(quantity=None, quantity_unit=None, price=None, price_unit=None)
    record = dict(
        schema_version='satgate.receipt.v2', record_type='execution_observation',
        binding_profile='satgate.binding_only.v1', canonicalization='jcs-rfc8785',
        hash_algorithm='sha256', signature_algorithm='ed25519',
        record_id='dummy-observation-a', issuer=parent['issuer'], issuer_kid=parent['issuer_kid'],
        observed_at='2026-11-16T11:01:00Z', decision_receipt_hash=parent['receipt_hash'],
        call_commitment=copy.deepcopy(parent['call_commitment']), operation_id=call['operation_id'],
        attempt_id=call['attempt_id'], status=status,
        result_commitment={'profile': 'satgate.execution_result.v1', 'algorithm': 'sha256',
                           'digest': result_commitment(private)},
        capsule_digest=digest({'dummy': 'encrypted-capsule-not-supplied'}))
    return sign(record), parent, call, private


def leaves(value, path=(), excluded=()):
    if isinstance(value, dict):
        for key, child in value.items():
            if not path and key in excluded:
                continue
            yield from leaves(child, path + (key,))
    else:
        yield path


def change(value, path):
    target = value
    for key in path[:-1]:
        target = target[key]
    old = target[path[-1]]
    target[path[-1]] = 'dummy-changed' if old is None else str(old) + '-changed'


def corpus():
    from tests.generate_decision_v2 import corpus as decisions
    keys = decisions()['jwks']
    record, parent, call, private = seed_observation()
    cases = []

    def add(name, r=record, p=parent, o=call, result=private, valid=False, code=None, jwks=None):
        cases.append(copy.deepcopy(dict(name=name, receipt=r, parent=p, opening=o,
                                       result_opening=result, valid=valid, code=code,
                                       **({'jwks': jwks} if jwks is not None else {}))))

    add('valid-observation-opened', valid=True, code='v2_result_matched')
    add('valid-result-unopened', result=None, valid=True, code='v2_result_not_checked')
    add('public-attempt-not-checked', o=None, result=None, code='v2_attempt_not_checked')
    add('missing-parent', p=None, code='v2_parent_required')
    second = next(c for c in decisions()['vectors'] if c['name'] == 'valid-identical-order-new-attempt')
    p, o = second['receipt'], second['opening']
    r, result = copy.deepcopy((record, private))
    r.update(record_id='dummy-observation-b', decision_receipt_hash=p['receipt_hash'],
             call_commitment=copy.deepcopy(p['call_commitment']), attempt_id=o['attempt_id'])
    result.update(decision_receipt_hash=p['receipt_hash'], attempt_id=o['attempt_id'],
                  opening=b64(bytes(range(128, 160))))
    r['result_commitment']['digest'] = result_commitment(result)
    add('valid-identical-order-new-attempt', sign(r), p, o, result, valid=True)
    add('cross-attempt-authentic-parent', p=p, o=o, code='v2_parent_mismatch')
    # An authentic observation reparented to a fresh attempt cannot reuse the
    # first attempt's authentic result opening, even after its digest is signed.
    r['result_commitment']['digest'] = result_commitment(private)
    add('cross-attempt-authentic-result', sign(r), p, o, private, code='v2_result_link_mismatch')
    for status in STATUSES:
        r, p, o, result = seed_observation(status)
        add('status/' + status, r, p, o, result, valid=True)
    for operation in ('amend', 'cancel'):
        p, o, r, result = copy.deepcopy((parent, call, record, private))
        o['normalized_intent'].update(operation=operation, cancel_target='dummy-target',
                                      replacement_link='dummy-prior-attempt')
        p['call_commitment']['digest'] = commitment(o)
        sign_decision(p)
        r.update(decision_receipt_hash=p['receipt_hash'], call_commitment=copy.deepcopy(p['call_commitment']))
        result.update(decision_receipt_hash=p['receipt_hash'], target_order_id='dummy-target',
                      replacement_link='dummy-prior-attempt')
        r['result_commitment']['digest'] = result_commitment(result)
        add('valid-' + operation, sign(r), p, o, result, valid=True)
        wrong = copy.deepcopy(result)
        wrong['target_order_id'] = 'dummy-wrong-target'
        r['result_commitment']['digest'] = result_commitment(wrong)
        add('authentic-wrong-target/' + operation, sign(r), p, o, wrong, code='v2_result_link_mismatch')
    for path in leaves(record, excluded=('record_hash', 'signature')):
        r = copy.deepcopy(record)
        change(r, path)
        r['record_hash'] = digest({k: v for k, v in r.items() if k not in ('record_hash', 'signature')})
        add('signed-leaf/' + '/'.join(path), r)
    for path in (('call_commitment', 'digest'), ('result_commitment', 'digest'),
                 ('capsule_digest',), ('decision_receipt_hash',)):
        r = copy.deepcopy(record)
        target = r if len(path) == 1 else r[path[0]]
        target[path[-1]] = digest({'dummy': 'changed-signed-digest'})
        r['record_hash'] = digest({k: v for k, v in r.items() if k not in ('record_hash', 'signature')})
        add('signed-canonical-digest/' + '/'.join(path), r, code='v2_signature_invalid')
    for field in record:
        r = copy.deepcopy(record)
        del r[field]
        add('missing/' + field, sign(r) if field not in ('record_hash', 'signature') else r,
            code='v2_schema_invalid')
    for field, value in (
        ('schema_version', 'satgate.receipt.v99'), ('record_type', 'broker_result'),
        ('binding_profile', 'satgate.enrolled.v1'), ('canonicalization', 'json'),
        ('hash_algorithm', 'sha512'), ('signature_algorithm', 'rsa'), ('status', 'refused'),
        ('status', 'executed'), ('extra', 'dummy-private-canary'), ('broker_order_id', 'dummy-private-order'),
        ('amount', '2'), ('opening', b64(bytes(32))), ('observed_at', 'not-a-time'),
        ('observed_at', '9999-11-16T13:00:00Z'), ('issuer', 'http://issuer.example')):
        r = copy.deepcopy(record)
        r[field] = value
        add('closed/' + field + '/' + str(value), sign(r), code='v2_schema_invalid')
    for group in ('call_commitment', 'result_commitment'):
        for field in ('profile', 'algorithm', 'digest'):
            r = copy.deepcopy(record)
            r[group][field] = 'unknown'
            add('closed/' + group + '/' + field, sign(r), code='v2_schema_invalid')
    for field in ('record_hash', 'signature', 'capsule_digest'):
        r = copy.deepcopy(record)
        r[field] += '='
        add('padded/' + field, r, code='v2_schema_invalid')
    r = copy.deepcopy(record)
    r['record_hash'] = digest({})
    add('wrong-record-hash', r, code='v2_hash_mismatch')
    r = copy.deepcopy(record)
    r['signature'] = 'ed25519:' + b64(bytes(64))
    add('wrong-signature', r, code='v2_signature_invalid')
    for field in ('decision_receipt_hash', 'attempt_id', 'operation_id', 'issuer', 'call_commitment', 'observed_at'):
        r = copy.deepcopy(record)
        r[field] = ({**r[field], 'digest': digest({})} if field == 'call_commitment'
                    else digest({}) if field == 'decision_receipt_hash'
                    else '2026-11-16T10:59:00Z' if field == 'observed_at'
                    else 'https://other.example' if field == 'issuer' else 'dummy-other')
        add('authentic-wrong-link/' + field, sign(r), code='v2_parent_mismatch')
    p = copy.deepcopy(parent)
    p['receipt_id'] = 'dummy-other-parent'
    add('authentic-wrong-parent', p=sign_decision(p), code='v2_parent_mismatch')
    add('multiple-parents', p=[parent, parent], code='v2_parent_invalid')
    p = copy.deepcopy(parent)
    p['signature'] = 'ed25519:' + b64(bytes(64))
    add('invalid-parent-signature', p=p, code='v2_parent_invalid')
    for disposition in ('not_required', 'required_pending'):
        p = copy.deepcopy(parent)
        p.update(admission='denied', approval_disposition=disposition)
        sign_decision(p)
        r = copy.deepcopy(record)
        r['decision_receipt_hash'] = p['receipt_hash']
        add('denied-parent/' + disposition, sign(r), p, code='v2_parent_mismatch')
    for field in call:
        o = copy.deepcopy(call)
        del o[field]
        add('call-missing/' + field, o=o, code='v2_parent_invalid')
    for field in ('account_binding', 'attempt_id', 'operation_id', 'route_binding', 'opening'):
        o = copy.deepcopy(call)
        o[field] = b64(bytes(32)) if field == 'opening' else 'dummy-other'
        add('wrong-call/' + field, o=o, code='v2_parent_invalid')
    for path in leaves(private):
        if path == ('schema_version',):
            continue
        result = copy.deepcopy(private)
        change(result, path)
        add('result-tamper/' + '/'.join(path), result=result)
    for field in private:
        result = copy.deepcopy(private)
        del result[field]
        add('result-missing/' + field, result=result, code='v2_result_opening_invalid')
    for group in ('response', 'financial_values', 'venue_times'):
        for field in private[group]:
            result = copy.deepcopy(private)
            del result[group][field]
            add('result-nested-missing/' + group + '/' + field, result=result, code='v2_result_opening_invalid')
    result = copy.deepcopy(private)
    result['response']['body_b64'] = b64(b'{ "quantity":"2", "id":"dummy-private-order" }')
    add('response-exact-bytes-not-json-equivalence', result=result, code='v2_result_mismatch')
    for field in ('decision_receipt_hash', 'operation_id', 'attempt_id', 'account_binding',
                  'status', 'target_order_id', 'replacement_link'):
        result = copy.deepcopy(private)
        result[field] = digest({}) if field == 'decision_receipt_hash' else 'rejected' if field == 'status' else 'dummy-other'
        r = copy.deepcopy(record)
        r['result_commitment']['digest'] = result_commitment(result)
        add('authentic-wrong-result-link/' + field, sign(r), result=result, code='v2_result_link_mismatch')
    for field, value in (('schema_version', 'unknown'), ('opening', b64(bytes(31))),
                         ('opening', private['opening'] + '='), ('extra', 'dummy-private-canary'),
                         ('Account_Binding', 'dummy-collision')):
        result = copy.deepcopy(private)
        result[field] = value
        add('result-closed/' + field + '/' + str(value), result=result, code='v2_result_opening_invalid')
    for name, raw in (('duplicate', b'{"id":1,"id":2}'), ('case-collision', b'{"id":1,"ID":2}'),
                      ('credential', b'{"token":"dummy-credential"}'),
                      ('unsafe-integer', b'{"n":9007199254740992}'), ('unicode', b'{"n":"\\ud800"}')):
        result = copy.deepcopy(private)
        result['response']['body_b64'] = b64(raw)
        add('response-invalid/' + name, result=result, code='v2_result_opening_invalid')
    for value in (2, '02', '1e2', '-1'):
        result = copy.deepcopy(private)
        result['financial_values']['quantity'] = value
        add('ambiguous-quantity/' + str(value), result=result, code='v2_result_opening_invalid')
    result = copy.deepcopy(private)
    result['financial_values']['fee'] = '1'
    add('unknown-economic-field', result=result, code='v2_result_opening_invalid')
    result = copy.deepcopy(private)
    result['response']['headers_b64'] = b64(b'Authorization: dummy-credential')
    add('credential-header', result=result, code='v2_result_opening_invalid')
    result = copy.deepcopy(private)
    result['opening'] = call['opening']
    r = copy.deepcopy(record)
    r['result_commitment']['digest'] = result_commitment(result)
    add('reused-call-entropy', sign(r), result=result, code='v2_result_opening_invalid')
    for status in ('accepted', 'partially_filled', 'filled', 'cancelled'):
        for field in ('broker_order_id', 'venue_status', 'response'):
            r, p, o, result = seed_observation(status)
            result[field] = None
            r['result_commitment']['digest'] = result_commitment(result)
            add('missing-reliable-result/' + status + '/' + field, sign(r), p, o, result,
                code='v2_result_opening_invalid')
    r, p, o, result = seed_observation('sent_outcome_unknown')
    result.update(response=private['response'], broker_order_id=private['broker_order_id'],
                  venue_status=private['venue_status'])
    r['result_commitment']['digest'] = result_commitment(result)
    add('valid-unknown-with-response-not-inferred', sign(r), p, o, result, valid=True,
        code='v2_outcome_unknown')
    for case in decisions()['vectors']:
        if 'jwks' in case:
            add('key/' + case['name'], jwks=case['jwks'], code=case['code'])
    return dict(now=NOW, jwks=keys, vectors=cases)


if __name__ == '__main__':
    print(json.dumps(corpus(), indent=2, sort_keys=True))
