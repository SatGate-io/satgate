"""Offline private receipt bundles for the binding-only development profile."""
from datetime import datetime, timedelta, timezone
import importlib.util
from pathlib import Path
import re

import rfc8785
from cryptography.exceptions import InvalidSignature
from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PublicKey

_spec = importlib.util.spec_from_file_location('bundle_observation_v2',
                                              Path(__file__).with_name('receipt_observation_v2.py'))
o = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(o)
d = o.d

SCHEMA = 'satgate.receipt_bundle.v2'
MANIFEST_SCHEMA = 'satgate.receipt_bundle_manifest.v2'
FIELDS = {'schema_version', 'record_type', 'manifest', 'artifacts', 'capsules', 'openings'}
MANIFEST_FIELDS = frozenset('''schema_version record_type binding_profile canonicalization hash_algorithm
signature_algorithm issuer issuer_kid generated_at range artifacts disclosure outcomes retention_state
historical_keys manifest_hash signature'''.split())
OPENING_STATES = ('withheld', 'missing', 'expired', 'deleted')
OUTCOME_STATES = ('available', 'pending', 'unknown', 'not_expected')
RETENTION_STATES = ('retained', 'expired', 'deleted')
REASON_CODES = (
    'v2_bundle_verified', 'v2_bundle_artifact_mismatch', 'v2_bundle_capsule_mismatch',
    'v2_bundle_opening_mismatch',
    'v2_bundle_record_mismatch', 'v2_bundle_disclosure_mismatch', 'v2_bundle_outcome_mismatch',
    'v2_bundle_opening_withheld', 'v2_bundle_opening_missing', 'v2_bundle_opening_expired',
    'v2_bundle_opening_deleted', 'v2_bundle_outcome_pending', 'v2_bundle_outcome_unknown',
    'v2_bundle_retention_expired', 'v2_bundle_retention_deleted',
)
INPUT_ERRORS = (ValueError, TypeError, OverflowError, RecursionError)


class InvalidBundle(ValueError):
    pass


def require(ok, code):
    if not ok:
        raise InvalidBundle(code)


def verdict(status, codes, authenticated=False):
    green = status == 'verified'
    return dict(status=status, valid=green, complete=green, trusted_issuer_valid=authenticated,
                reason_codes=codes, reasons=codes, summary={})


def failure(code):
    return verdict('invalid', [code])


def digest_syntax(value):
    d.prefixed(value, 'sha256:', 32)


def timestamp_key(value):
    # Compare signed range boundaries without losing fractional precision.
    seconds = d.timestamp(value).replace(microsecond=0)
    return seconds, value[:-1].partition('.')[2].rstrip('0')


def manifest_schema(bundle, now):
    d.closed(bundle, FIELDS)
    d.require(bundle['schema_version'] == SCHEMA and bundle['record_type'] == 'private_bundle')
    man = bundle['manifest']
    d.closed(man, MANIFEST_FIELDS)
    expected = dict(schema_version=MANIFEST_SCHEMA, record_type='bundle_manifest', binding_profile=d.PROFILE,
                    canonicalization='jcs-rfc8785', hash_algorithm='sha256', signature_algorithm='ed25519')
    d.require(all(man[k] == v for k, v in expected.items()))
    d.require(d.text(man['issuer_kid']) and d.text(man['issuer']))
    d.require(re.fullmatch(r'https://[a-z0-9](?:[a-z0-9.-]*[a-z0-9])?(?::443)?', man['issuer']) is not None)
    d.closed(man['range'], {'from', 'to'})
    start, end = (timestamp_key(man['range'][k]) for k in ('from', 'to'))
    current = d.timestamp(now) if now is not None else datetime.now(timezone.utc)
    d.require(start <= end <= timestamp_key(man['generated_at'])
              and d.timestamp(man['generated_at']) <= current + timedelta(seconds=300))
    d.require(man['retention_state'] in RETENTION_STATES)
    d.require(isinstance(man['artifacts'], list) and isinstance(man['disclosure'], dict)
              and isinstance(man['outcomes'], dict) and isinstance(man['historical_keys'], list))
    for val in (*man['artifacts'], *man['disclosure'], *man['outcomes']):
        digest_syntax(val)
    for entry in man['disclosure'].values():
        d.closed(entry, {'state', 'capsule_digest', 'opening_digest'})
        if entry['capsule_digest'] is not None:
            digest_syntax(entry['capsule_digest'])
        if entry['state'] == 'disclosed':
            d.require(entry['capsule_digest'] is not None)
            digest_syntax(entry['opening_digest'])
        else:
            d.require(entry['state'] in OPENING_STATES and entry['opening_digest'] is None)
    d.require(all(state in OUTCOME_STATES for state in man['outcomes'].values()))
    seen = set()
    for key in man['historical_keys']:
        d.closed(key, {'kty', 'crv', 'alg', 'use', 'kid', 'x'})
        d.require(key['kty'] == 'OKP' and key['crv'] == 'Ed25519' and key['alg'] == 'EdDSA' and key['use'] == 'sig')
        d.require(d.text(key['kid']) and key['kid'] not in seen)
        seen.add(key['kid'])
        d.decode(key['x'], 32)
    digest_syntax(man['manifest_hash'])
    d.prefixed(man['signature'], 'ed25519:', 64)
    return man, start, end


def authenticate(man, jwks):
    try:
        raw = rfc8785.dumps({k: v for k, v in man.items() if k not in ('manifest_hash', 'signature')})
    except INPUT_ERRORS:
        raise InvalidBundle('v2_schema_invalid') from None
    require(man['manifest_hash'] == d.digest(raw), 'v2_hash_mismatch')
    try:
        d.require(isinstance(jwks, dict) and isinstance(jwks.get('keys'), list))
        keys = [k for k in jwks['keys'] if isinstance(k, dict) and k.get('kid') == man['issuer_kid']]
        d.require(len(keys) == 1)
        key = keys[0]
        d.require(key.get('kty') == 'OKP' and key.get('crv') == 'Ed25519')
        d.require(key.get('alg') in (None, 'EdDSA') and key.get('use') in (None, 'sig'))
        public = Ed25519PublicKey.from_public_bytes(d.decode(key.get('x'), 32))
    except INPUT_ERRORS:
        raise InvalidBundle('v2_key_required') from None
    try:
        public.verify(d.prefixed(man['signature'], 'ed25519:', 64), raw)
    except (ValueError, InvalidSignature):
        raise InvalidBundle('v2_signature_invalid') from None


def blobs(value, expected, mismatch_code, parse_code=None):
    require(isinstance(value, dict) and set(value) == expected, mismatch_code)
    parsed = {}
    for digest in sorted(expected):
        try:
            raw = d.decode(value[digest])
        except INPUT_ERRORS:
            raise InvalidBundle(mismatch_code) from None
        require(d.digest(raw) == digest, mismatch_code)
        if parse_code is None:
            require(bool(raw), mismatch_code)
            continue  # Ciphertext is opaque: no envelope/decryption protocol in this reader.
        try:
            parsed[digest] = d.strict_loads(raw.decode('utf-8'))
            d.require(isinstance(parsed[digest], dict))
        except INPUT_ERRORS:
            raise InvalidBundle(parse_code) from None
    return parsed


def check_result_without_call(record, opening):
    # The public parent link passed row 04, but its private attempt join did
    # not. Check the disclosed result itself without promoting that join.
    try:
        exact = o.envelope(opening)
    except INPUT_ERRORS:
        raise InvalidBundle('v2_result_opening_invalid') from None
    require(d.digest(rfc8785.dumps(exact)) == record['result_commitment']['digest'], 'v2_result_mismatch')
    require(all(opening[k] == record[k] for k in ('decision_receipt_hash', 'operation_id', 'attempt_id', 'status')),
            'v2_result_link_mismatch')


def check_records(man, start, end, records, openings, jwks, now):
    decisions, observations, ids, partial = {}, [], set(), set()
    for artifact_hash, record in records.items():
        selection = man['disclosure'][artifact_hash]
        capsule_hash = selection['capsule_digest']
        opening_hash = selection['opening_digest']
        opening = openings[opening_hash] if opening_hash is not None else None
        if opening is None:
            partial.add('v2_bundle_opening_' + selection['state'])
        kind = record.get('record_type')
        if kind == 'decision':
            result = d.verify(record, jwks=jwks, now=now, opening=opening)
            require(result['valid'], result['reason_codes'][0])
            record_id, time = record['receipt_id'], record['issued_at']
            decisions[record['receipt_hash']] = record, opening
        elif kind == 'execution_observation':
            try:
                o.schema(record, now)
            except INPUT_ERRORS:
                raise InvalidBundle('v2_schema_invalid') from None
            record_id, time = record['record_id'], record['observed_at']
            observations.append((record, opening, capsule_hash))
        else:
            raise InvalidBundle('v2_schema_invalid')
        require(record['issuer'] == man['issuer'] and (kind, record_id) not in ids and start <= timestamp_key(time) < end,
                'v2_bundle_record_mismatch')
        ids.add((kind, record_id))

    outcomes = {key: set() for key in decisions}
    for record, opening, capsule_hash in observations:
        parent_hash = record['decision_receipt_hash']
        parent, call = decisions.get(parent_hash, (None, None))
        if capsule_hash is not None:
            require(capsule_hash == record['capsule_digest'], 'v2_bundle_capsule_mismatch')
        result = o.verify(record, jwks=jwks, now=now, parent=parent, opening=call, result_opening=opening)
        if not result['valid']:
            code = result['reason_codes'][0]
            require(code == 'v2_attempt_not_checked', code)
            if opening is not None:
                check_result_without_call(record, opening)
            partial.add(code)
        outcomes[parent_hash].add(record['status'])

    require(set(man['outcomes']) == set(decisions), 'v2_bundle_outcome_mismatch')
    for key, (record, _) in decisions.items():
        state = man['outcomes'][key]
        seen = outcomes[key]
        if record['admission'] == 'denied':
            expected = ('not_expected',)
        elif not seen:
            expected = ('pending', 'unknown')
        elif 'sent_outcome_unknown' in seen:
            # Row 04 has no observation-to-observation reconciliation link.
            expected = ('unknown',)
        else:
            expected = ('available',)
        require(state in expected, 'v2_bundle_outcome_mismatch')
        if state in ('pending', 'unknown'):
            partial.add('v2_bundle_outcome_' + state)
    return partial


def verify(bundle, jwks=None, now=None):
    try:
        man, start, end = manifest_schema(bundle, now)
    except INPUT_ERRORS:
        return failure('v2_schema_invalid')
    try:
        authenticate(man, jwks)
        hashes = set(man['artifacts'])
        require(len(hashes) == len(man['artifacts']), 'v2_bundle_artifact_mismatch')
        require(set(man['disclosure']) == hashes, 'v2_bundle_disclosure_mismatch')
        def selected(field):
            digests = [entry[field] for entry in man['disclosure'].values() if entry[field] is not None]
            require(len(set(digests)) == len(digests), 'v2_bundle_disclosure_mismatch')
            return set(digests)

        capsule_hashes = selected('capsule_digest')
        opening_hashes = selected('opening_digest')
        records = blobs(bundle['artifacts'], hashes, 'v2_bundle_artifact_mismatch', 'v2_schema_invalid')
        blobs(bundle['capsules'], capsule_hashes, 'v2_bundle_capsule_mismatch')
        openings = blobs(bundle['openings'], opening_hashes, 'v2_bundle_opening_mismatch', 'v2_bundle_opening_mismatch')
        partial = check_records(man, start, end, records, openings, jwks, now)
        if man['retention_state'] != 'retained':
            partial.add('v2_bundle_retention_' + man['retention_state'])
    except InvalidBundle as exc:
        return failure(str(exc))
    if partial:
        return verdict('partial', sorted(partial), authenticated=True)
    return verdict('verified', ['v2_bundle_verified'], authenticated=True)
