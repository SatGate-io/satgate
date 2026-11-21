"""Offline execution-observation reader, binding-only development profile."""
from datetime import datetime, timezone, timedelta
import importlib.util
from pathlib import Path
import re

import rfc8785
from cryptography.exceptions import InvalidSignature
from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PublicKey

# Resolve our own sibling, never an unrelated reader's cached import.
_spec = importlib.util.spec_from_file_location('observation_decision_v2',
                                              Path(__file__).with_name('receipt_decision_v2.py'))
d = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(d)

FIELDS = frozenset('''schema_version record_type binding_profile canonicalization hash_algorithm
signature_algorithm record_id issuer issuer_kid observed_at decision_receipt_hash call_commitment
operation_id attempt_id status result_commitment capsule_digest record_hash signature'''.split())
OPENING_FIELDS = frozenset('''schema_version decision_receipt_hash operation_id attempt_id opening
status account_binding broker_order_id client_order_id venue_status venue_times parser_version
response financial_values target_order_id replacement_link'''.split())
STATUSES = ('accepted', 'rejected', 'transport_failed_before_send', 'sent_outcome_unknown',
            'partially_filled', 'filled', 'cancelled', 'later_reconciled')
REASON_CODES = ('v2_parent_required', 'v2_parent_invalid', 'v2_parent_mismatch',
                'v2_attempt_not_checked', 'v2_observation_verified', 'v2_result_not_checked',
                'v2_result_matched', 'v2_result_mismatch', 'v2_result_opening_invalid',
                'v2_result_link_mismatch', 'v2_outcome_unknown')


def payload(record):
    return rfc8785.dumps({k: v for k, v in record.items() if k not in ('record_hash', 'signature')})


def commitment(value, profile):
    d.closed(value, {'profile', 'algorithm', 'digest'})
    d.require(value['profile'] == profile and value['algorithm'] == 'sha256')
    d.prefixed(value['digest'], 'sha256:', 32)


def schema(record, now):
    d.closed(record, FIELDS)
    expected = dict(schema_version=d.SCHEMA, record_type='execution_observation',
                    binding_profile=d.PROFILE, canonicalization='jcs-rfc8785',
                    hash_algorithm='sha256', signature_algorithm='ed25519')
    d.require(all(record[k] == v for k, v in expected.items()))
    d.require(all(d.text(record[k]) for k in ('record_id', 'issuer', 'issuer_kid', 'operation_id', 'attempt_id')))
    d.require(re.fullmatch(r'https://[a-z0-9](?:[a-z0-9.-]*[a-z0-9])?(?::443)?', record['issuer']) is not None)
    current = d.timestamp(now) if now is not None else datetime.now(timezone.utc)
    d.require(d.timestamp(record['observed_at']) <= current + timedelta(seconds=300))
    d.require(record['status'] in STATUSES)
    commitment(record['call_commitment'], 'satgate.exact_call.v1')
    commitment(record['result_commitment'], 'satgate.execution_result.v1')
    for field in ('decision_receipt_hash', 'capsule_digest', 'record_hash'):
        d.prefixed(record[field], 'sha256:', 32)
    d.prefixed(record['signature'], 'ed25519:', 64)
    rfc8785.dumps(record)


def envelope(opening):
    d.closed(opening, OPENING_FIELDS)
    d.require(opening['schema_version'] == 'satgate.execution_result.opening.v1')
    d.prefixed(opening['decision_receipt_hash'], 'sha256:', 32)
    d.require(all(d.text(opening[k]) for k in ('operation_id', 'attempt_id', 'account_binding', 'parser_version')))
    d.decode(opening['opening'], 32)
    d.require(opening['status'] in STATUSES)
    for field in ('broker_order_id', 'client_order_id', 'venue_status', 'target_order_id', 'replacement_link'):
        d.require(opening[field] is None or d.text(opening[field]))
    times = opening['venue_times']
    d.closed(times, {'received_at', 'updated_at'})
    for value in times.values():
        if value is not None:
            d.timestamp(value)
    values = opening['financial_values']
    d.closed(values, {'quantity', 'quantity_unit', 'price', 'price_unit'})
    for amount, unit in (('quantity', 'quantity_unit'), ('price', 'price_unit')):
        d.require((values[amount] is None and values[unit] is None)
                  or (d.decimal(values[amount]) and d.text(values[unit])))
    response = opening['response']
    if response is not None:
        d.closed(response, {'profile', 'headers_b64', 'body_b64'})
        d.require(response['profile'] == 'satgate.binding_only.response.v1')
        headers = d.decode(response['headers_b64']).decode('utf-8')
        d.require(re.search(r'(?i)(authorization|cookie|token|secret|password|api[-_]?key|bearer\s|sgcap_|stk_)', headers) is None)
        d.credential_free(d.strict_loads(d.decode(response['body_b64']).decode('utf-8')))
    # No raw response/ids inferred from HTTP success or prose; missing reliable
    # id/status cannot be upgraded by this development profile.
    if opening['status'] in ('accepted', 'partially_filled', 'filled', 'cancelled'):
        d.require(response is not None and d.text(opening['broker_order_id']) and d.text(opening['venue_status']))
    if opening['status'] == 'transport_failed_before_send':
        d.require(response is None and opening['broker_order_id'] is None and opening['venue_status'] is None)
    d.credential_free({k: v for k, v in opening.items()
                       if k not in ('opening', 'response', 'decision_receipt_hash')})
    result = dict(domain='satgate.execution_result.v1', **{
        k: v for k, v in opening.items() if k != 'schema_version'})
    rfc8785.dumps(result)
    return result


def failure(code):
    result = d.failure(code)
    result.update(parent_binding='not_checked', attempt_binding='not_checked', result_binding='not_checked',
                  observation_status='not_evidenced', broker_witness='not_evidenced', settlement_state='not_evidenced')
    return result


def fail(result, code):
    result.update(valid=False, reason_codes=[code], reasons=[code])
    return result


def verify(record, jwks=None, now=None, parent=None, opening=None, result_opening=None):
    result = failure('v2_schema_invalid')
    try:
        schema(record, now)
    except (ValueError, TypeError, OverflowError, RecursionError):
        return result
    raw = payload(record)
    if record['record_hash'] != d.digest(raw):
        return failure('v2_hash_mismatch')
    try:
        d.require(isinstance(jwks, dict) and isinstance(jwks.get('keys'), list))
        keys = [k for k in jwks['keys'] if isinstance(k, dict) and k.get('kid') == record['issuer_kid']]
        d.require(len(keys) == 1)
        key = keys[0]
        d.require(key.get('kty') == 'OKP' and key.get('crv') == 'Ed25519')
        d.require(key.get('alg') in (None, 'EdDSA') and key.get('use') in (None, 'sig'))
        public = Ed25519PublicKey.from_public_bytes(d.decode(key.get('x'), 32))
    except (ValueError, TypeError):
        return failure('v2_key_required')
    try:
        public.verify(d.prefixed(record['signature'], 'ed25519:', 64), raw)
    except (ValueError, InvalidSignature):
        return failure('v2_signature_invalid')
    result.update(trusted_issuer_valid=True, checks=dict(schema_valid=True, record_hash_match=True,
                                                       signature_valid=True, trusted_issuer_valid=True))
    if parent is None:
        return fail(result, 'v2_parent_required')
    # Exactly one independently verified decision object, never a search over
    # candidates or a pack/receipt hash substituted for the payload digest.
    parent_signature = d.verify(parent, jwks=jwks, now=now)
    if not parent_signature['valid']:
        return fail(result, 'v2_parent_invalid')
    result['checks']['parent_signature_valid'] = True
    if (record['decision_receipt_hash'] != d.digest(d.payload(parent))
            or record['call_commitment'] != parent['call_commitment']
            or record['issuer'] != parent['issuer'] or parent['admission'] != 'allowed'
            or parent['approval_disposition'] != 'not_required'
            or d.timestamp(record['observed_at']) < d.timestamp(parent['issued_at'])):
        result['parent_binding'] = 'mismatched'
        return fail(result, 'v2_parent_mismatch')
    result['parent_binding'] = 'matched'
    result['checks']['parent_digest_match'] = True
    # Row-03 decisions keep attempt/operation private inside call_commitment.
    # Without that opening, do NOT claim a verified attempt join.
    if opening is None:
        return fail(result, 'v2_attempt_not_checked')
    parent_opened = d.verify(parent, jwks=jwks, now=now, opening=opening)
    if not parent_opened['valid']:
        return fail(result, 'v2_parent_invalid')
    result['call_binding'] = 'matched'
    if record['attempt_id'] != opening['attempt_id'] or record['operation_id'] != opening['operation_id']:
        result['attempt_binding'] = 'mismatched'
        return fail(result, 'v2_parent_mismatch')
    result['attempt_binding'] = 'matched'
    result['checks']['attempt_match'] = True
    if result_opening is not None:
        try:
            exact = envelope(result_opening)
            d.require(result_opening['opening'] != opening['opening'])
        except (ValueError, TypeError, OverflowError, RecursionError):
            return fail(result, 'v2_result_opening_invalid')
        if d.digest(rfc8785.dumps(exact)) != record['result_commitment']['digest']:
            result['result_binding'] = 'mismatched'
            return fail(result, 'v2_result_mismatch')
        if (any(result_opening[k] != record[k] for k in ('decision_receipt_hash', 'operation_id', 'attempt_id', 'status'))
                or result_opening['account_binding'] != opening['account_binding']
                or result_opening['target_order_id'] != opening['normalized_intent']['cancel_target']
                or result_opening['replacement_link'] != opening['normalized_intent']['replacement_link']):
            result['result_binding'] = 'mismatched'
            return fail(result, 'v2_result_link_mismatch')
        result['result_binding'] = 'matched'
    codes = ['v2_observation_verified', 'v2_result_matched' if result_opening is not None else 'v2_result_not_checked',
             'v2_authority_not_evidenced']
    if record['status'] == 'sent_outcome_unknown':
        codes.append('v2_outcome_unknown')
    result.update(valid=True, reason_codes=codes, reasons=codes, admission_state='allowed',
                  observation_status=record['status'], execution_state='observed_' + record['status'])
    return result
