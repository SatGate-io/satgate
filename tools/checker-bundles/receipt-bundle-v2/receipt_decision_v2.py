"""Reader-only v2 development profile; no enrollment or producer implementation."""
import base64
from datetime import datetime, timezone, timedelta
import hashlib
import json
import re

import rfc8785
from cryptography.exceptions import InvalidSignature
from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PublicKey

SCHEMA = 'satgate.receipt.v2'
PROFILE = 'satgate.binding_only.v1'
FIELDS = frozenset('''schema_version record_type phase binding_profile canonicalization
hash_algorithm signature_algorithm receipt_id issuer issuer_kid issued_at enforcement_mode
admission_revision authority_evidence authorization_record_hash effective_limits_record_hash
approval_record_hash approval_disposition admission dispatch call_commitment receipt_hash signature'''.split())
OPENING_FIELDS = frozenset('''schema_version receipt_id attempt_id operation_id authority_binding
account_binding route_binding adapter_version opening transport normalized_intent policy_projection'''.split())
TRANSPORT_FIELDS = frozenset('''profile service tool method path query_b64 headers_b64 body_b64 params_b64'''.split())
INTENT_FIELDS = frozenset('''operation quantity quantity_unit price price_unit time_in_force
cancel_target replacement_link'''.split())
REASON_CODES = (
    'v2_schema_invalid', 'v2_hash_mismatch', 'v2_key_required', 'v2_signature_invalid',
    'v2_opening_invalid', 'v2_call_mismatch', 'v2_call_matched', 'v2_call_not_checked',
    'v2_hold_not_executed', 'v2_authority_not_evidenced', 'v2_unsupported_container',
    'v2_input_invalid',
)


class InvalidV2(ValueError):
    pass


def require(ok):
    if not ok:
        raise InvalidV2('v2_input_invalid')


def closed(value, fields):
    require(isinstance(value, dict) and set(value) == fields)


def text(value):
    return isinstance(value, str) and bool(value.strip()) and len(value) <= 4096


def decode(value, length=None):
    require(isinstance(value, str) and re.fullmatch('[A-Za-z0-9_-]*', value) is not None)
    try:
        raw = base64.b64decode(value + '=' * (-len(value) % 4), altchars=b'-_', validate=True)
    except ValueError:
        raise InvalidV2('v2_input_invalid') from None
    require(base64.urlsafe_b64encode(raw).decode().rstrip('=') == value)
    require(length is None or len(raw) == length)
    return raw


def prefixed(value, prefix, length):
    require(isinstance(value, str) and value.startswith(prefix))
    return decode(value[len(prefix):], length)


def pairs(items):
    result = {}
    seen = set()
    for key, value in items:
        # A closed development wrapper reserves every spelling, at every depth.
        require(key.casefold() not in seen)
        seen.add(key.casefold())
        result[key] = value
    return result


def strict_loads(text):
    value = json.loads(text, object_pairs_hook=pairs,
                       parse_constant=lambda _: require(False))
    rfc8785.dumps(value)
    return value


def payload(receipt):
    return rfc8785.dumps({k: v for k, v in receipt.items() if k not in ('receipt_hash', 'signature')})


def digest(raw):
    return 'sha256:' + base64.urlsafe_b64encode(hashlib.sha256(raw).digest()).decode().rstrip('=')


def timestamp(value):
    require(isinstance(value, str) and re.fullmatch(
        r'[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(?:\.[0-9]+)?Z', value) is not None)
    return datetime.fromisoformat(value[:-1] + '+00:00')


def schema(receipt, now):
    closed(receipt, FIELDS)
    expected = dict(schema_version=SCHEMA, record_type='decision', phase='pre_dispatch',
                    binding_profile=PROFILE, canonicalization='jcs-rfc8785', hash_algorithm='sha256',
                    signature_algorithm='ed25519', enforcement_mode='control',
                    authority_evidence='not_evidenced', authorization_record_hash=None,
                    effective_limits_record_hash=None, approval_record_hash=None, dispatch='not_sent')
    require(all(receipt[k] == v for k, v in expected.items()))
    require(all(text(receipt[k]) for k in ('receipt_id', 'issuer', 'issuer_kid', 'admission_revision')))
    # Origin only, no receipt-directed discovery. Keys are local caller input.
    require(re.fullmatch(r'https://[a-z0-9](?:[a-z0-9.-]*[a-z0-9])?(?::443)?', receipt['issuer']) is not None)
    issued = timestamp(receipt['issued_at'])
    current = timestamp(now) if now is not None else datetime.now(timezone.utc)
    require(issued <= current + timedelta(seconds=300))
    require(receipt['admission'] in ('allowed', 'denied'))
    require(receipt['approval_disposition'] in ('not_required', 'required_pending'))
    require(receipt['approval_disposition'] != 'required_pending' or receipt['admission'] == 'denied')
    commitment = receipt['call_commitment']
    closed(commitment, {'profile', 'algorithm', 'digest'})
    require(commitment['profile'] == 'satgate.exact_call.v1' and commitment['algorithm'] == 'sha256')
    prefixed(commitment['digest'], 'sha256:', 32)
    prefixed(receipt['receipt_hash'], 'sha256:', 32)
    prefixed(receipt['signature'], 'ed25519:', 64)
    rfc8785.dumps(receipt)


def credential_free(value):
    if isinstance(value, dict):
        for key, child in value.items():
            folded = re.sub('[^a-z0-9]', '', key.casefold())
            require(not any(marker in folded for marker in (
                'authorization', 'cookie', 'token', 'secret', 'password', 'apikey')))
            credential_free(child)
    elif isinstance(value, list):
        for child in value:
            credential_free(child)
    elif isinstance(value, str):
        require(re.search(r'(?i)(?:bearer\s|sgcap_|stk_)', value) is None)


def decimal(value):
    return isinstance(value, str) and re.fullmatch(r'(?:0|[1-9][0-9]*)(?:\.[0-9]+)?', value) is not None


def envelope(opening):
    closed(opening, OPENING_FIELDS)
    require(opening['schema_version'] == 'satgate.exact_call.opening.v1')
    require(all(text(opening[k]) for k in (
        'receipt_id', 'attempt_id', 'operation_id', 'authority_binding', 'account_binding',
        'route_binding', 'adapter_version')))
    decode(opening['opening'], 32)
    transport = opening['transport']
    closed(transport, TRANSPORT_FIELDS)
    require(transport['profile'] == 'satgate.binding_only.transport.v1')
    require(all(text(transport[k]) for k in ('service', 'tool', 'method', 'path')))
    for field in ('query_b64', 'headers_b64', 'body_b64', 'params_b64'):
        raw = decode(transport[field])
        # This development transport supports UTF-8 JSON bodies/params only;
        # exact byte order/whitespace remains in the committed base64url.
        decoded = raw.decode('utf-8')
        if field in ('body_b64', 'params_b64'):
            credential_free(strict_loads(decoded))
        else:
            require(re.search(r'(?i)(authorization|cookie|token|secret|password|api[-_]?key|bearer\s|sgcap_|stk_)', decoded) is None)
    intent = opening['normalized_intent']
    closed(intent, INTENT_FIELDS)
    require(intent['operation'] in ('create', 'amend', 'cancel'))
    require(all(decimal(intent[k]) for k in ('quantity', 'price')))
    require(all(text(intent[k]) for k in ('quantity_unit', 'price_unit', 'time_in_force')))
    require(all(intent[k] is None or text(intent[k]) for k in ('cancel_target', 'replacement_link')))
    projection = opening['policy_projection']
    closed(projection, {'credit_charge', 'default_time_in_force'})
    require(text(projection['default_time_in_force']))
    charge = projection['credit_charge']
    closed(charge, {'evaluation_state', 'amount', 'unit'})
    require(charge['evaluation_state'] in ('evaluated', 'not_evaluated'))
    require((decimal(charge['amount']) and text(charge['unit'])) if charge['evaluation_state'] == 'evaluated'
            else charge['amount'] is None and charge['unit'] is None)
    # Generated encodings can coincidentally contain credential-like markers.
    # Scan meaningful text and decoded transport, never random/base64 bytes.
    credential_free({k: v for k, v in opening.items() if k not in ('opening', 'transport')})
    credential_free({k: v for k, v in transport.items() if not k.endswith('_b64')})
    result = dict(domain='satgate.exact_call.v1', **{
        k: v for k, v in opening.items() if k != 'schema_version'})
    rfc8785.dumps(result)
    return result


def failure(code):
    return {'valid': False, 'trusted_issuer_valid': False, 'reason_codes': [code], 'reasons': [code],
            'checks': {}, 'summary': {}, 'call_binding': 'not_checked',
            'admission_state': 'not_evidenced', 'execution_state': 'not_evidenced',
            'authority_state': 'not_evidenced', 'protocol_profile': PROFILE}


def verify(receipt, jwks=None, now=None, opening=None, **unused):
    result = failure('v2_schema_invalid')
    try:
        schema(receipt, now)
    except (ValueError, TypeError, OverflowError, RecursionError):
        return result
    raw = payload(receipt)
    if receipt['receipt_hash'] != digest(raw):
        return failure('v2_hash_mismatch')
    try:
        require(isinstance(jwks, dict) and isinstance(jwks.get('keys'), list))
        keys = [k for k in jwks['keys'] if isinstance(k, dict) and k.get('kid') == receipt['issuer_kid']]
        require(len(keys) == 1)
        key = keys[0]
        require(key.get('kty') == 'OKP' and key.get('crv') == 'Ed25519')
        require(key.get('alg') in (None, 'EdDSA') and key.get('use') in (None, 'sig'))
        public = Ed25519PublicKey.from_public_bytes(decode(key.get('x'), 32))
    except (ValueError, TypeError):
        return failure('v2_key_required')
    try:
        public.verify(prefixed(receipt['signature'], 'ed25519:', 64), raw)
    except (ValueError, InvalidSignature):
        return failure('v2_signature_invalid')
    result.update(trusted_issuer_valid=True, checks={'schema_valid': True, 'receipt_hash_match': True,
                                                   'signature_valid': True, 'trusted_issuer_valid': True})
    if opening is not None:
        try:
            exact = envelope(opening)
        except (ValueError, TypeError, OverflowError, RecursionError):
            result.update(reason_codes=['v2_opening_invalid'], reasons=['v2_opening_invalid'])
            return result
        if opening['receipt_id'] != receipt['receipt_id'] or digest(rfc8785.dumps(exact)) != receipt['call_commitment']['digest']:
            result.update(call_binding='mismatched', reason_codes=['v2_call_mismatch'], reasons=['v2_call_mismatch'])
            return result
        result['call_binding'] = 'matched'
    codes = ['v2_call_matched' if opening is not None else 'v2_call_not_checked', 'v2_authority_not_evidenced']
    held = receipt['approval_disposition'] == 'required_pending'
    if held:
        codes.append('v2_hold_not_executed')
    result.update(valid=True, reason_codes=codes, reasons=codes,
                  admission_state='held' if held else receipt['admission'])
    return result
