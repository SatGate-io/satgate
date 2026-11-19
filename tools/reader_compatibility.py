"""Offline v1 corpus construction. All identities and signing material are test dummies.

Regenerate deliberately with:
  python tools/reader_compatibility.py > tools/fixtures/reader-v1-goldens.json
The matrix also rebuilds the corpus to detect fixture-builder drift.
"""
import copy
import json

from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PrivateKey
from cryptography.hazmat.primitives.serialization import Encoding, PublicFormat

import test_paid_rail as paid
import test_every_action_receipts as action
import test_order_controls_receipts as controls
import test_receipt_record as records
import test_unavailable_check_receipts as unavailable
import test_x402_rail as x402


def corpus():
    f = paid.Fixture()
    # Public, deterministic test seed; never a production signing key.
    f.key = Ed25519PrivateKey.from_private_bytes(bytes(range(32)))
    f.x = paid.V.b64url_encode(f.key.public_key().public_bytes(Encoding.Raw, PublicFormat.Raw))
    f.jwks['keys'][0]['x'] = f.x
    packs = []

    def add(name, receipt, strict=False, valid=True):
        receipt = copy.deepcopy(receipt)
        pack = (paid.verified_not_evaluated_pack(f, receipt) if strict else f.pack(receipt))
        packs.append(dict(name=name, pack=pack, valid=valid))

    for decision, reason, valid in paid.LEGACY_PROFILES:
        add(decision + '/' + reason, f.receipt(decision, reason), valid=valid)
    for decision in ('allowed', 'paid'):
        add(decision + '/payment_verified/l402', f.receipt(decision))
        for network in x402.USDC:
            add(decision + '/payment_verified/x402/' + network,
                x402.x402_receipt(f, decision=decision, network=network))
    add('free-allow', action.allowed_free(f), True)
    add('verified-policy-refusal', paid.verified_not_evaluated(f), True)
    add('window-open', action.opener(f), True)
    add('window-close', action.closer(f), True)
    for reason in ('token_revoked', 'capability_expired'):
        for phase in ('opened', 'closed'):
            extra = (dict(rate_limit_phase=phase, rate_limit_refused_calls=3,
                          issued_at=action.CLOSED_AT) if phase == 'closed' else {})
            add('window-' + phase + '/' + reason, action.account_level(f, reason, **extra), True)
    add('observe-gap', action.gap(f), True)
    for reason in unavailable.REASONS:
        add('unavailable/' + reason, unavailable.refusal(f, reason), True)
    for name, mutate, decision, reason in (
        ('spend-refusal', controls.refusal, 'denied', 'policy_denied'),
        ('spend-allow', controls.allowed_with_total, 'allowed', 'budget_authorized'),
        ('shared-refusal', controls.shared_refusal, 'denied', 'policy_denied'),
        ('shared-allow', controls.shared_allowed, 'allowed', 'budget_authorized'),
        ('approval-hold', controls.approval_hold, 'denied', 'policy_denied'),
    ):
        add(name, mutate(f.receipt(decision, reason)))
    add('unknown-receipt-version', dict(f.receipt(), schema_version='satgate.receipt.v999'), valid=False)
    add('unknown-reason', f.receipt('denied', 'not_a_gateway_reason'), valid=False)
    add('legacy-unpaired-paid', f.receipt('paid', 'budget_authorized'))
    add('legacy-spend-remaining', dict(paid.verified_not_evaluated(f), spend_remaining='10'), True)
    for field in ('issued_at', 'evidence_url', 'authority', 'budget', 'policy'):
        p = f.pack(f.receipt('allowed', 'budget_authorized'))
        p.pop(field)
        packs.append(dict(name='legacy-missing-' + field, pack=p, valid=True))
    p = copy.deepcopy(packs[0]['pack'])
    p['receipts'][0]['route_or_tool'] = '/tampered'
    packs.append(dict(name='tampered', pack=p, valid=False))
    p = copy.deepcopy(packs[0]['pack'])
    p['schema_version'] = 'satgate.evidence_pack.v999'
    packs.append(dict(name='unknown-pack-version', pack=p, valid=False))
    # A caller-supplied local JWKS anchors keys, not an issuer-name allowlist.
    add('other-dummy-issuer', dict(f.receipt(), issuer='https://other.example'))
    r = f.receipt()
    r['issuer_kid'] = 'missing-dummy-key'
    add('missing-kid', r, valid=False)

    b = records.Builder(f)
    positive = [b.allowed(10), b.refused(20), b.opener(30), b.closer(80), b.gap(90)]
    positive += [b.pack(b.base(unavailable.refusal(f, reason), 100 + i))
                 for i, reason in enumerate(unavailable.REASONS)]
    window = [records.window_entry(positive[2], positive[3], 5)]
    record_cases = []

    def record(name, ps, terminal, status):
        record_cases.append(dict(name=name, text=records.text(ps, terminal), status=status))

    record('complete-all-shapes', positive, b.manifest(positive, windows=window), 'complete')
    record('truncated-all-shapes', positive, b.manifest(
        positive, windows=window, schema_version=records.REC.TRUNCATED_SCHEMA,
        truncated=True, truncation_reason='test_cutoff'), 'truncated')
    record('empty-complete', [], b.manifest([]), 'complete')
    record('empty-truncated', [], b.manifest([], schema_version=records.REC.TRUNCATED_SCHEMA,
           truncated=True, truncation_reason='test_cutoff'), 'truncated')
    terminal = b.manifest([])
    terminal['count'] = 99
    record('tampered-terminal', [], terminal, 'invalid')
    record('unknown-terminal-version', [], b.manifest([], schema_version='satgate.terminal.v999'), 'invalid')
    # Every identical pack vector also reaches every record reader. A record
    # has additional key/range/window constraints; its reference is the record
    # reader, never the pack-only verdict.
    for case in packs:
        ps = [case['pack']]
        record('pack/' + case['name'], ps, b.manifest(ps), None)
    return dict(now=paid.NOW, jwks=f.jwks, packs=packs, records=record_cases)


if __name__ == '__main__':
    print(json.dumps(corpus(), indent=2, sort_keys=True))
