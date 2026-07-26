#!/usr/bin/env python3
"""Verify a public SatGate Evidence Pack.

Usage:
  python demo/verify_evidence_pack.py https://api.satgate.io/v1/evidence/evid_...
  python demo/verify_evidence_pack.py ./evidence-pack.json
  curl -fsS ... | python demo/verify_evidence_pack.py -

The verifier needs no SatGate auth. It validates the proof artifact itself:
- Evidence Pack schema version
- embedded receipt hash
- Ed25519 receipt signature against either a caller-supplied issuer JWKS or the embedded public key
- top-level pack fields matching the embedded receipt
- obvious bearer-token leakage markers

By default the verifier requires issuer-anchored verification and auto-discovers the
issuer JWKS from the receipt issuer origin. Pass --allow-embedded-key to downgrade to
embedded-key self-verification for demo/self-signed packs; that mode only proves the
artifact is internally self-consistent, NOT that a trusted issuer signed it.
"""

from __future__ import annotations

import argparse
import base64
import hashlib
import ipaddress
from datetime import datetime, timezone, timedelta
from decimal import Decimal, InvalidOperation
import json
import re
import sys
import urllib.error
import urllib.parse
import urllib.request
from pathlib import Path
from typing import Any

VENDOR_DIR = Path(__file__).resolve().parent / "_vendor"
if VENDOR_DIR.exists():
    sys.path.insert(0, str(VENDOR_DIR))

try:
    import rfc8785
except Exception:  # pragma: no cover - vendored package should be present
    rfc8785 = None

try:
    from cryptography.exceptions import InvalidSignature
    from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PublicKey
except Exception:  # pragma: no cover - exercised only in minimal Python envs
    InvalidSignature = Exception
    Ed25519PublicKey = None

EXPECTED_PACK_SCHEMA = "satgate.evidence_pack.v1"
EXPECTED_RECEIPT_SCHEMA = "satgate.receipt.v1"
SUPPORTED_CANONICALIZATION = "jcs-rfc8785"
DECISION_REASON_PROFILES = {
    ("allowed", "budget_authorized"): "evaluated",
    ("allowed", "policy_allowed"): "evaluated",
    ("allowed", "sandbox_no_spend"): "sandbox_no_spend",
    ("allowed", "observe_projected"): "projected_only",
    ("allowed", "payment_verified"): "paid_rail",
    ("paid", "payment_verified"): "paid_rail",
    ("denied", "budget_exhausted"): "evaluated",
    ("denied", "policy_denied"): "evaluated",
    ("denied", "capability_invalid"): "not_evaluated",
    ("denied", "capability_expired"): "not_evaluated",
    ("denied", "auth_missing"): "not_evaluated",
    ("denied", "token_revoked"): "not_evaluated",
    ("denied", "payment_required"): "not_evaluated",
}
ALLOWED_DECISIONS = {decision for decision, _ in DECISION_REASON_PROFILES}
ALLOWED_DECISION_REASONS = {reason for _, reason in DECISION_REASON_PROFILES}
DEFAULT_CLOCK_SKEW_SECONDS = 300
SECRET_PATTERNS = [
    re.compile(r"Bearer\s+[A-Za-z0-9._~+/=-]+", re.IGNORECASE),
    re.compile(r"sgcap_[A-Za-z0-9._~+/=-]+"),
    re.compile(r"stk_[A-Za-z0-9._~+/=-]+"),
]
PAID_RAIL_FIELDS = {"rail", "payment_hash", "invoice_hash", "macaroon_hash", "amount_sats"}
CREDIT_USD_FIELDS = {
    "amount_usd", "attempted_amount_usd", "remaining_budget_usd", "currency",
    "credit_unit", "cost_credits", "limit_credits", "remaining_before_credits",
    "remaining_after_credits", "remaining_credits", "projected_cost_credits",
}


def is_zero_amount(value):
    if value is None:
        return True
    try:
        return Decimal(str(value)) == 0
    except (InvalidOperation, ValueError):
        return False


def contains_any_key(value: Any, forbidden: set[str]) -> bool:
    if isinstance(value, dict):
        return any(key in forbidden or contains_any_key(child, forbidden) for key, child in value.items())
    if isinstance(value, list):
        return any(contains_any_key(child, forbidden) for child in value)
    return False


class VerificationError(Exception):
    pass


def add_reason(reasons: list[str], reason_codes: list[str] | None, code: str, message: str) -> None:
    reasons.append(message)
    if reason_codes is not None and code not in reason_codes:
        reason_codes.append(code)


def parse_rfc3339(value: Any, reasons: list[str], reason_codes: list[str], field: str) -> datetime | None:
    if not isinstance(value, str) or not value:
        add_reason(reasons, reason_codes, f"missing_{field}", f"receipt.{field} missing")
        return None
    try:
        normalized = value[:-1] + "+00:00" if value.endswith("Z") else value
        # Go's time.RFC3339Nano emits variable fractional precision, while this
        # Python runtime accepts exactly three or six fractional digits. Normalize
        # to six digits by truncating nanoseconds or padding shorter fractions;
        # the original string still participates in hash/signature verification.
        if "." in normalized:
            prefix, suffix = normalized.split(".", 1)
            frac = suffix
            tz = ""
            for marker in ("+", "-"):
                if marker in suffix:
                    frac, tz = suffix.split(marker, 1)
                    tz = marker + tz
                    break
            if not frac or not frac.isdigit():
                raise ValueError("fractional seconds must contain digits")
            normalized = prefix + "." + frac[:6].ljust(6, "0") + tz
        parsed = datetime.fromisoformat(normalized)
        if parsed.tzinfo is None:
            raise ValueError("timestamp must include timezone")
        return parsed.astimezone(timezone.utc)
    except ValueError:
        add_reason(reasons, reason_codes, f"malformed_{field}", f"receipt.{field} is not RFC3339")
        return None


def issuer_jwks_url(issuer: str, reasons: list[str], reason_codes: list[str]) -> str | None:
    parsed = urllib.parse.urlparse(issuer)
    if parsed.scheme != "https" or parsed.username or parsed.password or parsed.fragment or not parsed.hostname or parsed.port not in {None, 443}:
        add_reason(reasons, reason_codes, "invalid_issuer", "receipt.issuer must be an https origin without userinfo, fragment, or non-default port")
        return None
    try:
        host_ip = ipaddress.ip_address(parsed.hostname)
    except ValueError:
        host_ip = None
    if host_ip and (host_ip.is_private or host_ip.is_loopback or host_ip.is_link_local or host_ip.is_multicast or host_ip.is_reserved):
        add_reason(reasons, reason_codes, "invalid_issuer", "receipt.issuer must not target private, loopback, link-local, multicast, or reserved IP space")
        return None
    origin = f"https://{parsed.hostname}"
    return origin + "/.well-known/jwks.json"


def jwks_source_matches_issuer(jwks_source: str | None, issuer: str) -> bool:
    if not jwks_source or not jwks_source.startswith("https://"):
        return True
    source = urllib.parse.urlparse(jwks_source)
    issuer_parsed = urllib.parse.urlparse(issuer)
    return source.scheme == "https" and issuer_parsed.scheme == "https" and source.hostname == issuer_parsed.hostname and source.port in {None, 443}


def b64url_decode(value: str) -> bytes:
    return base64.urlsafe_b64decode(value + "=" * ((4 - len(value) % 4) % 4))


def b64url_encode(value: bytes) -> str:
    return base64.urlsafe_b64encode(value).decode("ascii").rstrip("=")


def canonical_receipt_payload(receipt: dict[str, Any]) -> bytes:
    """Return the RFC8785/JCS canonical receipt payload.

    `receipt_hash` and `signature` are excluded from the signed payload.
    """
    if rfc8785 is None:
        raise VerificationError("rfc8785 canonicalization module unavailable")
    payload = {k: v for k, v in receipt.items() if k not in {"receipt_hash", "signature"}}
    return rfc8785.dumps(payload)


def sha256_receipt_hash(receipt: dict[str, Any]) -> str:
    return "sha256:" + b64url_encode(hashlib.sha256(canonical_receipt_payload(receipt)).digest())


def sha256_evidence_pack_hash(pack: dict[str, Any]) -> str:
    """Hash the RFC8785/JCS archived pack with only evidence_pack_hash omitted.

    previous_evidence_pack_hash remains in the payload so the archive chain pointer is bound.
    """
    if rfc8785 is None:
        raise VerificationError("rfc8785 canonicalization module unavailable")
    payload = {k: v for k, v in pack.items() if k != "evidence_pack_hash"}
    return "sha256:" + b64url_encode(hashlib.sha256(rfc8785.dumps(payload)).digest())


ARCHIVE_FIELDS = (
    "archive_profile", "archive_storage", "archive_retention", "archived_at",
    "evidence_pack_hash", "previous_evidence_pack_hash",
)


def validate_archive(pack: dict[str, Any], reasons: list[str], reason_codes: list[str]) -> bool:
    """Validate archive metadata for every pack profile, not only sandbox packs."""
    if not any(field in pack and pack.get(field) is not None for field in ARCHIVE_FIELDS):
        return True
    archive_profile = pack.get("archive_profile")
    archive_storage = pack.get("archive_storage")
    archive_retention = pack.get("archive_retention")
    archived_at = pack.get("archived_at")
    evidence_pack_hash = pack.get("evidence_pack_hash")
    previous_hash = pack.get("previous_evidence_pack_hash")
    valid = (
        archive_profile == "satgate.archive.v1"
        and isinstance(archive_storage, str) and bool(archive_storage)
        and isinstance(archive_retention, str) and bool(archive_retention)
        and isinstance(archived_at, str)
        and parse_rfc3339(archived_at, reasons, reason_codes, "archived_at") is not None
        and isinstance(evidence_pack_hash, str)
        and re.fullmatch(r"sha256:[A-Za-z0-9_-]{43}", evidence_pack_hash) is not None
        and evidence_pack_hash == sha256_evidence_pack_hash(pack)
        and (previous_hash is None or (
            isinstance(previous_hash, str)
            and re.fullmatch(r"sha256:[A-Za-z0-9_-]{43}", previous_hash) is not None
        ))
    )
    if not valid:
        add_reason(reasons, reason_codes, "invalid_archive_provenance", "archive metadata or canonical evidence_pack_hash is invalid")
    return valid


def sha256_string_hash(value: str) -> str:
    return "sha256:" + b64url_encode(hashlib.sha256(value.encode("utf-8")).digest())


# Fields the gateway folds into the effective-policy object before hashing.
# The order is irrelevant because RFC8785/JCS canonicalization sorts keys.
POLICY_EFFECTIVE_FIELDS = [
    "policy_id",
    "policy_version",
    "policy_decision_mode",
    "policy_source",
    "route_or_tool",
    "cost_credits",
    "projected_cost_credits",
    "credit_unit",
]


def canonical_effective_policy_hash(policy: dict[str, Any]) -> str | None:
    """Recompute policy.policy_effective_hash from the fields the gateway signed."""
    if rfc8785 is None:
        return None
    effective = {field: policy[field] for field in POLICY_EFFECTIVE_FIELDS if field in policy}
    return "sha256:" + b64url_encode(hashlib.sha256(rfc8785.dumps(effective)).digest())


def strict_json_equal(left: Any, right: Any) -> bool:
    """Return True only when JSON values match in value and concrete type.

    Python considers ``1 == 1.0 == True`` for some comparisons because ``bool``
    subclasses ``int`` and numeric types coerce. Evidence Pack top-level mirrors
    are human-readable views of signed receipt subtrees; accept only byte-faithful
    JSON type/value equality so mirror tampering cannot change ``1`` to ``true``
    or ``20`` to ``20.0`` while still passing verification.
    """
    if type(left) is not type(right):
        return False
    if isinstance(left, dict):
        if left.keys() != right.keys():
            return False
        return all(strict_json_equal(left[key], right[key]) for key in left)
    if isinstance(left, list):
        if len(left) != len(right):
            return False
        return all(strict_json_equal(a, b) for a, b in zip(left, right))
    return left == right


def verify_ed25519_signature_with_public_key(receipt: dict[str, Any], public_key_value: str, reasons: list[str], reason_codes: list[str] | None, label: str) -> bool:
    if Ed25519PublicKey is None:
        add_reason(reasons, reason_codes, "crypto_unavailable", "cryptography package is required for Ed25519 verification")
        return False

    signature_value = receipt.get("signature")
    if not isinstance(signature_value, str) or not signature_value.startswith("ed25519:"):
        add_reason(reasons, reason_codes, "missing_signature", "receipt.signature missing ed25519: prefix")
        return False

    try:
        key_bytes = b64url_decode(public_key_value)
        if len(key_bytes) != 32:
            raise ValueError(f"Ed25519 public key must be 32 bytes, got {len(key_bytes)}")
        public_key = Ed25519PublicKey.from_public_bytes(key_bytes)
        signature = b64url_decode(signature_value.removeprefix("ed25519:"))
        public_key.verify(signature, canonical_receipt_payload(receipt))
        return True
    except (ValueError, InvalidSignature) as exc:
        add_reason(reasons, reason_codes, "invalid_signature", f"receipt signature invalid with {label}: {exc.__class__.__name__}")
        return False


def verify_embedded_ed25519_signature(receipt: dict[str, Any], reasons: list[str], reason_codes: list[str] | None = None) -> bool:
    metadata = receipt.get("metadata")
    if not isinstance(metadata, dict):
        add_reason(reasons, reason_codes, "missing_metadata", "receipt.metadata missing")
        return False

    public_key_value = metadata.get("public_key_ed25519_b64")
    if not isinstance(public_key_value, str) or not public_key_value:
        add_reason(reasons, reason_codes, "missing_embedded_public_key", "receipt.metadata.public_key_ed25519_b64 missing")
        return False

    return verify_ed25519_signature_with_public_key(receipt, public_key_value, reasons, reason_codes, "embedded public key")


def jwks_key_for_receipt(receipt: dict[str, Any], jwks: dict[str, Any] | None, reasons: list[str], reason_codes: list[str] | None = None) -> str | None:
    if not jwks:
        return None
    kid = receipt.get("issuer_kid")
    if not isinstance(kid, str) or not kid:
        add_reason(reasons, reason_codes, "missing_issuer_kid", "receipt.issuer_kid missing; cannot verify against issuer JWKS")
        return None
    keys = jwks.get("keys")
    if not isinstance(keys, list):
        add_reason(reasons, reason_codes, "malformed_jwks", "trusted JWKS must contain a keys array")
        return None

    matches = [key for key in keys if isinstance(key, dict) and key.get("kid") == kid]
    if not matches:
        add_reason(reasons, reason_codes, "jwks_kid_not_found", f"issuer JWKS has no key matching kid {kid!r}")
        return None
    if len(matches) > 1:
        add_reason(reasons, reason_codes, "jwks_duplicate_kid", f"issuer JWKS has multiple keys matching kid {kid!r}")
        return None

    key = matches[0]
    if key.get("kty") != "OKP" or key.get("crv") != "Ed25519":
        add_reason(reasons, reason_codes, "malformed_jwks_key", f"issuer JWKS key {kid!r} must be OKP/Ed25519")
        return None
    if key.get("alg") not in {None, "EdDSA"}:
        add_reason(reasons, reason_codes, "malformed_jwks_key", f"issuer JWKS key {kid!r} has unsupported alg {key.get('alg')!r}")
        return None
    if key.get("use") not in {None, "sig"}:
        add_reason(reasons, reason_codes, "malformed_jwks_key", f"issuer JWKS key {kid!r} has unsupported use {key.get('use')!r}")
        return None
    x = key.get("x")
    if not isinstance(x, str) or not x:
        add_reason(reasons, reason_codes, "malformed_jwks_key", f"issuer JWKS key {kid!r} missing x coordinate")
        return None
    try:
        if len(b64url_decode(x)) != 32:
            add_reason(reasons, reason_codes, "malformed_jwks_key", f"issuer JWKS key {kid!r} x coordinate is not a 32-byte Ed25519 key")
            return None
    except Exception:
        add_reason(reasons, reason_codes, "malformed_jwks_key", f"issuer JWKS key {kid!r} x coordinate is not valid base64url")
        return None
    return x


def verify_receipt_signature(receipt: dict[str, Any], reasons: list[str], reason_codes: list[str], jwks: dict[str, Any] | None = None, require_trusted_issuer: bool = False) -> tuple[bool, bool, str]:
    issuer_key = jwks_key_for_receipt(receipt, jwks, reasons, reason_codes)
    if issuer_key:
        issuer_valid = verify_ed25519_signature_with_public_key(receipt, issuer_key, reasons, reason_codes, "issuer JWKS")
        if issuer_valid:
            return True, True, "issuer_jwks_anchored"
        if require_trusted_issuer:
            return False, False, "issuer_jwks_failed"

    if require_trusted_issuer:
        add_reason(reasons, reason_codes, "trusted_issuer_required", "issuer JWKS verification required; embedded public key fallback is not trusted")
        return False, False, "untrusted"

    embedded_valid = verify_embedded_ed25519_signature(receipt, reasons, reason_codes)
    return embedded_valid, False, "embedded_public_key_self_verifying" if embedded_valid else "unverified"


def contains_secret_marker(value: Any) -> bool:
    raw = json.dumps(value, sort_keys=True, separators=(",", ":"), default=str)
    # Redaction labels are expected. Actual bearer/capability prefixes are not.
    raw = raw.replace('"raw_capability_token":"redacted"', "")
    raw = raw.replace('"authorization_header":"redacted"', "")
    for pattern in SECRET_PATTERNS:
        if pattern.search(raw):
            return True
    return contains_unredacted_secret_field(value)


def contains_unredacted_secret_field(value: Any) -> bool:
    if isinstance(value, dict):
        for key, child in value.items():
            key_lower = str(key).lower()
            if any(marker in key_lower for marker in ["token", "secret", "authorization", "budget_id", "payment_secret"]):
                if isinstance(child, str) and child and child.lower() not in {"redacted", "[redacted]"}:
                    if not key_lower.endswith("hash") and "fingerprint" not in key_lower and "public_key" not in key_lower:
                        return True
            if contains_unredacted_secret_field(child):
                return True
    elif isinstance(value, list):
        return any(contains_unredacted_secret_field(child) for child in value)
    return False


def verify_receipt(receipt: dict[str, Any], index: int, reasons: list[str], reason_codes: list[str], jwks: dict[str, Any] | None = None, require_trusted_issuer: bool = False, now_dt: datetime | None = None, allow_mock: bool = False) -> tuple[dict[str, bool], str, str, bool]:
    prefix = f"receipt_{index}"
    checks: dict[str, bool] = {}

    def fail(code: str, message: str) -> None:
        add_reason(reasons, reason_codes, code, message)

    checks[f"{prefix}_schema_version"] = receipt.get("schema_version") == EXPECTED_RECEIPT_SCHEMA
    if not checks[f"{prefix}_schema_version"]:
        fail("invalid_receipt_schema", f"receipts[{index}].schema_version is {receipt.get('schema_version')!r}, expected {EXPECTED_RECEIPT_SCHEMA!r}")

    checks[f"{prefix}_canonicalization"] = receipt.get("canonicalization") == SUPPORTED_CANONICALIZATION
    if not checks[f"{prefix}_canonicalization"]:
        fail("unsupported_canonicalization", f"receipts[{index}].canonicalization must be {SUPPORTED_CANONICALIZATION!r}")

    checks[f"{prefix}_hash_algorithm"] = receipt.get("hash_algorithm") == "sha256"
    if receipt.get("hash_algorithm") is None:
        fail("missing_hash_algorithm", f"receipts[{index}].hash_algorithm missing")
    elif not checks[f"{prefix}_hash_algorithm"]:
        fail("unsupported_hash_algorithm", f"receipts[{index}].hash_algorithm is not sha256")

    checks[f"{prefix}_signature_algorithm"] = receipt.get("signature_algorithm") == "ed25519"
    if receipt.get("signature_algorithm") is None:
        fail("missing_signature_algorithm", f"receipts[{index}].signature_algorithm missing")
    elif not checks[f"{prefix}_signature_algorithm"]:
        fail("unsupported_signature_algorithm", f"receipts[{index}].signature_algorithm is not ed25519")

    checks[f"{prefix}_decision"] = receipt.get("decision") in ALLOWED_DECISIONS
    if not checks[f"{prefix}_decision"]:
        fail("unknown_decision", f"receipts[{index}].decision is not a supported protocol decision")

    checks[f"{prefix}_decision_reason"] = receipt.get("decision_reason") in ALLOWED_DECISION_REASONS
    if not checks[f"{prefix}_decision_reason"]:
        fail("unknown_decision_reason", f"receipts[{index}].decision_reason is not a supported protocol reason")

    decision = str(receipt.get("decision", ""))
    decision_reason = str(receipt.get("decision_reason", ""))
    decision_profile = DECISION_REASON_PROFILES.get((decision, decision_reason))
    checks[f"{prefix}_decision_reason_combination"] = decision_profile is not None
    if not checks[f"{prefix}_decision_reason_combination"]:
        fail("invalid_decision_reason_combination", f"receipts[{index}] decision/reason combination is not legal")

    if decision_profile == "not_evaluated":
        authority_value = receipt.get("authority")
        budget_value = receipt.get("budget")
        authority = authority_value if isinstance(authority_value, dict) else {}
        budget = budget_value if isinstance(budget_value, dict) else {}
        forbidden_receipt_fields = {
            "amount_usd", "attempted_amount_usd", "capability_hash", "currency",
            "rail", "remaining_budget_usd",
        }
        checks[f"{prefix}_not_evaluated"] = (
            receipt.get("decision") == "denied"
            and authority == {"provenance_level": "no_verified_capability"}
            and budget == {"spend_mode": "not_evaluated", "cost_credits": 0}
            and not any(field in receipt for field in forbidden_receipt_fields)
        )
        if not checks[f"{prefix}_not_evaluated"]:
            fail("invalid_not_evaluated_provenance", f"receipts[{index}] no-capability denial carries authority or evaluated budget claims")

    if decision_profile == "paid_rail":
        amount_sats = receipt.get("amount_sats")
        paid_context_value = receipt.get("paid_rail_context")
        paid_context = paid_context_value if isinstance(paid_context_value, dict) else {}
        budget_value = receipt.get("budget")
        budget = budget_value if isinstance(budget_value, dict) else {}
        hashes_complete = all(isinstance(receipt.get(field), str) and bool(receipt[field].strip()) for field in ("payment_hash", "invoice_hash", "macaroon_hash"))
        amount_valid = isinstance(amount_sats, int) and not isinstance(amount_sats, bool) and amount_sats > 0
        context_matches = (
            isinstance(paid_context_value, dict)
            and set(paid_context) == PAID_RAIL_FIELDS
            and all(paid_context.get(field) == receipt.get(field) for field in PAID_RAIL_FIELDS)
        )
        checks[f"{prefix}_paid_rail"] = (
            decision in {"allowed", "paid"}
            and receipt.get("rail") == "l402"
            and hashes_complete
            and amount_valid
            and context_matches
            and isinstance(budget_value, dict)
            and set(budget) == {"spend_mode", "rail", "amount_sats"}
            and budget.get("spend_mode") == "paid_rail"
            and budget.get("rail") == "l402"
            and budget.get("amount_sats") == amount_sats
            and not contains_any_key(receipt, CREDIT_USD_FIELDS)
        )
        if not checks[f"{prefix}_paid_rail"]:
            fail("invalid_paid_rail_provenance", f"receipts[{index}] payment_verified L402 context is missing, mismatched, or carries contradictory credit/USD state")

    if receipt.get("decision_reason") == "sandbox_no_spend":
        authority_value = receipt.get("authority")
        budget_value = receipt.get("budget")
        metadata_value = receipt.get("metadata")
        policy_value = receipt.get("policy")
        authority = authority_value if isinstance(authority_value, dict) else {}
        budget = budget_value if isinstance(budget_value, dict) else {}
        metadata = metadata_value if isinstance(metadata_value, dict) else {}
        policy = policy_value if isinstance(policy_value, dict) else {}
        allowed_authority = {"provenance_level"}
        allowed_budget = {"spend_mode", "cost_credits", "credit_unit"}
        allowed_metadata = {
            "cost_credits", "credit_unit", "method", "path", "public_key_ed25519_b64",
            "remaining_credits", "request_id", "tenant_id",
        }
        allowed_policy = {
            "cost_credits", "credit_unit", "hash", "id", "name", "policy_decision_mode",
            "policy_effective_hash", "policy_id", "policy_id_hash", "policy_source",
            "policy_version", "policy_version_hash", "route_or_tool", "version",
        }
        allowed_receipt = {
            "attempted_amount_usd", "authority", "budget", "canonicalization", "decision",
            "decision_reason", "event_history_ref", "evidence_access_id", "evidence_pack_id",
            "evidence_url", "expires_at", "hash_algorithm", "issued_at", "issuer", "issuer_kid",
            "issuer_trusted", "jwks_url", "metadata", "policy", "policy_version", "receipt_hash",
            "receipt_id", "remaining_budget_usd", "route_or_tool", "schema_url", "schema_version",
            "signature", "signature_algorithm", "timestamp", "trust_anchor", "verify_url",
        }
        checks[f"{prefix}_sandbox_no_spend"] = (
            receipt.get("decision") == "allowed"
            and authority.get("provenance_level") == "sandbox_no_capability"
            and all(key in allowed_authority for key in authority)
            and all(key in allowed_budget for key in budget)
            and isinstance(metadata_value, dict)
            and isinstance(metadata.get("tenant_id"), str) and bool(metadata.get("tenant_id"))
            and isinstance(metadata.get("request_id"), str) and bool(metadata.get("request_id"))
            and all(key in allowed_metadata for key in metadata)
            and is_zero_amount(metadata.get("cost_credits"))
            and is_zero_amount(metadata.get("remaining_credits"))
            and isinstance(policy_value, dict)
            and all(key in allowed_policy for key in policy)
            and policy.get("policy_decision_mode") == "sandbox"
            and is_zero_amount(policy.get("cost_credits"))
            and all(key in allowed_receipt for key in receipt)
            and budget.get("spend_mode") == "sandbox_no_spend"
            and budget.get("cost_credits") == 0
            and "attempted_amount_usd" in receipt
            and is_zero_amount(receipt["attempted_amount_usd"])
            and "remaining_budget_usd" in receipt
            and is_zero_amount(receipt["remaining_budget_usd"])
        )
        if not checks[f"{prefix}_sandbox_no_spend"]:
            fail("invalid_sandbox_provenance", f"receipts[{index}] sandbox_no_spend authority or spend claims are inconsistent")

    if receipt.get("decision_reason") == "observe_projected":
        authority_value = receipt.get("authority")
        budget_value = receipt.get("budget")
        metadata_value = receipt.get("metadata")
        policy_value = receipt.get("policy")
        authority = authority_value if isinstance(authority_value, dict) else {}
        budget = budget_value if isinstance(budget_value, dict) else {}
        metadata = metadata_value if isinstance(metadata_value, dict) else {}
        policy = policy_value if isinstance(policy_value, dict) else {}
        projected = receipt.get("projected_cost_credits")
        projected_valid = isinstance(projected, (int, float)) and not isinstance(projected, bool) and projected >= 0
        allowed_authority = {
            "budget_id_hash", "budget_subject_hash", "capability_hash", "delegation_budget",
            "delegation_depth", "parent_token_id_hash", "provenance_level", "raw_budget_id",
            "raw_capability_token", "raw_parent_token_id", "raw_token_id", "token_id_hash",
        }
        allowed_budget = {"credit_unit", "projected_cost_credits", "spend_mode"}
        allowed_metadata = {
            "credit_unit", "method", "path", "projected_cost_credits",
            "public_key_ed25519_b64", "request_id", "tenant_id",
        }
        allowed_policy = {
            "credit_unit", "hash", "id", "name", "policy_decision_mode",
            "policy_effective_hash", "policy_id", "policy_id_hash", "policy_source",
            "policy_version", "policy_version_hash", "projected_cost_credits",
            "route_or_tool", "version",
        }
        allowed_receipt = {
            "authority", "budget", "canonicalization", "capability_hash", "decision",
            "decision_reason", "event_history_ref", "evidence_access_id", "evidence_pack_id",
            "evidence_url", "expires_at", "hash_algorithm", "issued_at", "issuer", "issuer_kid",
            "issuer_trusted", "jwks_url", "metadata", "policy", "policy_version",
            "projected_cost_credits", "receipt_hash", "receipt_id", "route_or_tool", "schema_url",
            "schema_version", "signature", "signature_algorithm", "timestamp", "trust_anchor", "verify_url",
        }
        raw_authority_fields = ("raw_budget_id", "raw_capability_token", "raw_parent_token_id", "raw_token_id")
        checks[f"{prefix}_observe_projected"] = (
            receipt.get("decision") == "allowed"
            and projected_valid
            and isinstance(authority_value, dict)
            and authority.get("provenance_level") == "verified_macaroon_caveats"
            and all(key in allowed_authority for key in authority)
            and all(authority.get(key) in (None, "redacted") for key in raw_authority_fields)
            and isinstance(budget_value, dict)
            and set(budget) == allowed_budget
            and budget.get("spend_mode") == "projected_only"
            and budget.get("projected_cost_credits") == projected
            and isinstance(metadata_value, dict)
            and all(key in allowed_metadata for key in metadata)
            and isinstance(metadata.get("tenant_id"), str) and bool(metadata.get("tenant_id"))
            and isinstance(metadata.get("request_id"), str) and bool(metadata.get("request_id"))
            and metadata.get("projected_cost_credits") == projected
            and isinstance(policy_value, dict)
            and all(key in allowed_policy for key in policy)
            and policy.get("policy_decision_mode") == "observe"
            and policy.get("projected_cost_credits") == projected
            and all(key in allowed_receipt for key in receipt)
        )
        if not checks[f"{prefix}_observe_projected"]:
            fail("invalid_observe_projection", f"receipts[{index}] observe_projected authority or spend claims are inconsistent")

    mock_markers = [receipt.get("mock_only"), receipt.get("demo"), receipt.get("environment") in {"mock", "test", "demo"}]
    metadata = receipt.get("metadata") if isinstance(receipt.get("metadata"), dict) else {}
    mock_markers.extend([metadata.get("mock_only"), metadata.get("demo"), metadata.get("environment") in {"mock", "test", "demo"}])
    checks[f"{prefix}_production_artifact"] = allow_mock or not any(marker is True for marker in mock_markers)
    if not checks[f"{prefix}_production_artifact"]:
        fail("mock_artifact_not_allowed", f"receipts[{index}] is marked mock/demo/test but --allow-mock was not supplied")

    policy = receipt.get("policy")
    checks[f"{prefix}_policy_provenance_present"] = isinstance(policy, dict) and bool(policy)
    if not checks[f"{prefix}_policy_provenance_present"]:
        fail("missing_policy_provenance", f"receipts[{index}].policy missing named policy provenance")
    else:
        policy_version = policy.get("policy_version", policy.get("version"))
        checks[f"{prefix}_policy_version_consistent"] = policy_version == receipt.get("policy_version")
        if not checks[f"{prefix}_policy_version_consistent"]:
            fail("policy_version_mismatch", f"receipts[{index}].policy.policy_version does not match receipt.policy_version")
        policy_id = policy.get("policy_id", policy.get("id"))
        policy_id_hash = policy.get("policy_id_hash")
        checks[f"{prefix}_policy_id_hash_match"] = True
        if policy_id_hash is not None:
            checks[f"{prefix}_policy_id_hash_match"] = isinstance(policy_id, str) and policy_id_hash == sha256_string_hash(policy_id)
            if not checks[f"{prefix}_policy_id_hash_match"]:
                fail("policy_hash_mismatch", f"receipts[{index}].policy.policy_id_hash does not match policy_id")
        policy_version_hash = policy.get("policy_version_hash")
        checks[f"{prefix}_policy_version_hash_match"] = True
        if policy_version_hash is not None:
            checks[f"{prefix}_policy_version_hash_match"] = isinstance(policy_version, str) and policy_version_hash == sha256_string_hash(policy_version)
            if not checks[f"{prefix}_policy_version_hash_match"]:
                fail("policy_hash_mismatch", f"receipts[{index}].policy.policy_version_hash does not match policy_version")
        policy_effective_hash = policy.get("policy_effective_hash")
        checks[f"{prefix}_policy_effective_hash_match"] = True
        if policy_effective_hash is not None:
            recomputed_effective = canonical_effective_policy_hash(policy)
            if recomputed_effective is not None:
                checks[f"{prefix}_policy_effective_hash_match"] = policy_effective_hash == recomputed_effective
                if not checks[f"{prefix}_policy_effective_hash_match"]:
                    fail("policy_hash_mismatch", f"receipts[{index}].policy.policy_effective_hash does not match the canonical effective policy object")

    issued_at = parse_rfc3339(receipt.get("issued_at"), reasons, reason_codes, "issued_at")
    timestamp = parse_rfc3339(receipt.get("timestamp"), reasons, reason_codes, "timestamp") if receipt.get("timestamp") is not None else issued_at
    expires_at = parse_rfc3339(receipt.get("expires_at"), reasons, reason_codes, "expires_at") if receipt.get("expires_at") is not None else None
    checks[f"{prefix}_time_valid"] = issued_at is not None and timestamp is not None
    if now_dt and issued_at:
        if issued_at > now_dt + timedelta(seconds=DEFAULT_CLOCK_SKEW_SECONDS):
            checks[f"{prefix}_time_valid"] = False
            fail("receipt_issued_in_future", f"receipts[{index}].issued_at is beyond allowed clock skew")
    if now_dt and expires_at:
        if now_dt > expires_at:
            checks[f"{prefix}_time_valid"] = False
            fail("receipt_expired", f"receipts[{index}].expires_at is in the past")

    expected_hash = sha256_receipt_hash(receipt)
    checks[f"{prefix}_receipt_hash_match"] = receipt.get("receipt_hash") == expected_hash
    if not checks[f"{prefix}_receipt_hash_match"]:
        fail("receipt_hash_mismatch", f"receipts[{index}].receipt_hash does not match canonical receipt")

    signature_valid, trusted_issuer_valid, trust_anchor = verify_receipt_signature(receipt, reasons, reason_codes, jwks, require_trusted_issuer)
    checks[f"{prefix}_signature_valid"] = signature_valid
    checks[f"{prefix}_trusted_issuer_valid"] = trusted_issuer_valid
    return checks, expected_hash, trust_anchor, trusted_issuer_valid

def verify_pack(pack: dict[str, Any], jwks: dict[str, Any] | None = None, require_trusted_issuer: bool = False, allow_mock: bool = False, now: str | None = None, discover_jwks: bool = False, jwks_source: str | None = None) -> dict[str, Any]:
    reasons: list[str] = []
    reason_codes: list[str] = []
    checks: dict[str, bool] = {}
    now_dt = parse_rfc3339(now, reasons, reason_codes, "now") if now else datetime.now(timezone.utc)

    checks["schema_version"] = pack.get("schema_version") == EXPECTED_PACK_SCHEMA
    if not checks["schema_version"]:
        add_reason(reasons, reason_codes, "invalid_pack_schema", f"schema_version is {pack.get('schema_version')!r}, expected {EXPECTED_PACK_SCHEMA!r}")

    checks["production_artifact"] = allow_mock or not (pack.get("mock_only") is True or pack.get("demo") is True or pack.get("environment") in {"mock", "test", "demo"})
    if not checks["production_artifact"]:
        add_reason(reasons, reason_codes, "mock_artifact_not_allowed", "pack is marked mock/demo/test but --allow-mock was not supplied")

    receipts_value = pack.get("receipts")
    receipts = receipts_value if isinstance(receipts_value, list) else []
    checks["embedded_receipts_present"] = bool(receipts) and all(isinstance(r, dict) for r in receipts)
    if not checks["embedded_receipts_present"]:
        add_reason(reasons, reason_codes, "missing_receipts", "pack.receipts must contain one or more receipt objects")
        receipts = []

    if require_trusted_issuer and discover_jwks and jwks is None and receipts and isinstance(receipts[0], dict):
        issuer_value = receipts[0].get("issuer")
        if isinstance(issuer_value, str):
            discovered = issuer_jwks_url(issuer_value, reasons, reason_codes)
            if discovered:
                jwks_source = discovered
                jwks = load_jwks(discovered)

    if require_trusted_issuer and jwks_source and receipts and isinstance(receipts[0], dict):
        issuer_value = receipts[0].get("issuer")
        if isinstance(issuer_value, str) and not jwks_source_matches_issuer(jwks_source, issuer_value):
            checks["jwks_issuer_match"] = False
            add_reason(reasons, reason_codes, "jwks_issuer_mismatch", "trusted JWKS source does not match receipt issuer origin")
        else:
            checks["jwks_issuer_match"] = True

    receipt_checks: dict[str, bool] = {}
    receipt_hashes: list[str] = []
    receipt_trust_anchors: list[str] = []
    receipt_trusted_issuer: list[bool] = []
    for index, receipt in enumerate(receipts):
        per_receipt, expected_hash, trust_anchor, trusted_issuer_valid = verify_receipt(receipt, index, reasons, reason_codes, jwks, require_trusted_issuer, now_dt, allow_mock)
        receipt_checks.update(per_receipt)
        receipt_hashes.append(expected_hash)
        receipt_trust_anchors.append(trust_anchor)
        receipt_trusted_issuer.append(trusted_issuer_valid)
    checks.update(receipt_checks)
    checks["receipt_hash_match"] = bool(receipts) and all(checks.get(f"receipt_{i}_receipt_hash_match", False) for i in range(len(receipts)))
    checks["time_valid"] = bool(receipts) and all(checks.get(f"receipt_{i}_time_valid", False) for i in range(len(receipts)))
    checks["signature_valid"] = bool(receipts) and all(checks.get(f"receipt_{i}_signature_valid", False) for i in range(len(receipts)))
    checks["trusted_issuer_valid"] = bool(receipts) and all(receipt_trusted_issuer) and checks.get("jwks_issuer_match", True)
    if require_trusted_issuer and not checks["trusted_issuer_valid"]:
        add_reason(reasons, reason_codes, "trusted_issuer_required", "trusted issuer verification was required but at least one receipt was not issuer-JWKS anchored")

    checks["archive_valid"] = validate_archive(pack, reasons, reason_codes)

    primary = receipts[0] if receipts else {}
    if primary.get("decision_reason") == "sandbox_no_spend":
        allowed_pack = {
            "archive_profile", "archive_retention", "archive_storage", "archived_at", "authority",
            "budget", "budget_state", "capability_hash", "caveat", "correlation", "decision",
            "decision_reason", "evidence_pack_hash", "evidence_pack_id", "evidence_url", "issued_at",
            "issuer", "jwks_url", "policy", "previous_evidence_pack_hash", "receipt_hash", "receipt_id",
            "receipts", "redaction", "request_context", "request_id", "route_or_tool", "schema_url",
            "schema_version", "tenant_id", "verify_url",
        }
        budget_state_value = pack.get("budget_state")
        budget_state = budget_state_value if isinstance(budget_state_value, dict) else {}
        correlation_value = pack.get("correlation")
        request_context_value = pack.get("request_context")
        redaction_value = pack.get("redaction")
        correlation = correlation_value if isinstance(correlation_value, dict) else {}
        request_context = request_context_value if isinstance(request_context_value, dict) else {}
        redaction = redaction_value if isinstance(redaction_value, dict) else {}
        allowed_budget_state = {"attempted_amount_usd", "remaining_budget_usd", "credit_unit"}
        allowed_correlation = {
            "decision", "decision_reason", "evidence_pack_id", "receipt_hash", "receipt_id",
            "request_id", "route_or_tool", "tenant_id",
        }
        allowed_request_context = {"method", "path", "request_id", "route_or_tool", "tenant_id"}
        allowed_redaction = {"authorization_header", "raw_budget_id", "raw_capability_token"}
        primary_metadata_value = primary.get("metadata")
        primary_metadata = primary_metadata_value if isinstance(primary_metadata_value, dict) else {}
        expected_correlation = {
            "decision": primary.get("decision"),
            "decision_reason": primary.get("decision_reason"),
            "evidence_pack_id": primary.get("evidence_pack_id"),
            "receipt_hash": primary.get("receipt_hash"),
            "receipt_id": primary.get("receipt_id"),
            "request_id": primary_metadata.get("request_id"),
            "route_or_tool": primary.get("route_or_tool"),
            "tenant_id": primary_metadata.get("tenant_id"),
        }
        expected_request_context = {
            "method": primary_metadata.get("method"),
            "path": primary_metadata.get("path"),
            "request_id": primary_metadata.get("request_id"),
            "route_or_tool": primary.get("route_or_tool"),
            "tenant_id": primary_metadata.get("tenant_id"),
        }
        archive_profile = pack.get("archive_profile")
        archive_fields = ("archive_storage", "archive_retention", "archived_at", "evidence_pack_hash", "previous_evidence_pack_hash")
        if archive_profile is None:
            archive_valid = all(pack.get(field) is None for field in archive_fields)
        else:
            archive_storage = pack.get("archive_storage")
            archive_retention = pack.get("archive_retention")
            archived_at = pack.get("archived_at")
            evidence_pack_hash = pack.get("evidence_pack_hash")
            previous_hash = pack.get("previous_evidence_pack_hash")
            archive_valid = (
                archive_profile == "satgate.archive.v1"
                and isinstance(archive_storage, str) and bool(archive_storage)
                and isinstance(archive_retention, str) and bool(archive_retention)
                and isinstance(archived_at, str) and parse_rfc3339(archived_at, reasons, reason_codes, "archived_at") is not None
                and isinstance(evidence_pack_hash, str)
                and re.fullmatch(r"sha256:[A-Za-z0-9_-]{43}", evidence_pack_hash) is not None
                and evidence_pack_hash == sha256_evidence_pack_hash(pack)
                and (previous_hash is None or (isinstance(previous_hash, str) and re.fullmatch(r"sha256:[A-Za-z0-9_-]{43}", previous_hash) is not None))
            )
        primary_budget_value = primary.get("budget")
        primary_budget = primary_budget_value if isinstance(primary_budget_value, dict) else {}
        checks["pack_sandbox_no_spend"] = (
            archive_valid
            and isinstance(budget_state_value, dict)
            and all(key in allowed_pack for key in pack)
            and isinstance(pack.get("tenant_id"), str)
            and bool(pack.get("tenant_id"))
            and pack.get("tenant_id") == primary_metadata.get("tenant_id")
            and isinstance(pack.get("request_id"), str)
            and bool(pack.get("request_id"))
            and pack.get("request_id") == primary_metadata.get("request_id")
            and ("caveat" not in pack or isinstance(pack.get("caveat"), str))
            and all(
                field not in pack or pack.get(field) == primary.get(field)
                for field in ("issued_at", "jwks_url", "schema_url", "verify_url")
            )
            and all(key in allowed_budget_state for key in budget_state)
            and (correlation_value is None or isinstance(correlation_value, dict))
            and all(key in allowed_correlation for key in correlation)
            and all(value == expected_correlation[key] for key, value in correlation.items())
            and (request_context_value is None or isinstance(request_context_value, dict))
            and all(key in allowed_request_context for key in request_context)
            and all(value == expected_request_context[key] for key, value in request_context.items())
            and isinstance(redaction_value, dict)
            and all(key in allowed_redaction for key in redaction)
            and all(value == "redacted" for value in redaction.values())
            and ("capability_hash" not in pack or pack.get("capability_hash") is None)
            and "attempted_amount_usd" in budget_state
            and is_zero_amount(budget_state["attempted_amount_usd"])
            and "remaining_budget_usd" in budget_state
            and is_zero_amount(budget_state["remaining_budget_usd"])
            and budget_state.get("credit_unit") == primary_budget.get("credit_unit")
        )
        if not checks["pack_sandbox_no_spend"]:
            add_reason(reasons, reason_codes, "invalid_sandbox_provenance", "sandbox_no_spend pack claims capability, budget, delegation, or payment authority")

    if primary.get("decision_reason") == "observe_projected":
        allowed_pack = {
            "archive_profile", "archive_retention", "archive_storage", "archived_at", "authority",
            "budget", "budget_state", "capability_hash", "caveat", "correlation", "decision",
            "decision_reason", "evidence_pack_hash", "evidence_pack_id", "evidence_url", "issued_at",
            "issuer", "jwks_url", "policy", "previous_evidence_pack_hash", "receipt_hash", "receipt_id",
            "receipts", "redaction", "request_context", "request_id", "route_or_tool", "schema_url",
            "schema_version", "tenant_id", "verify_url",
        }
        projected = primary.get("projected_cost_credits")
        budget_state_value = pack.get("budget_state")
        budget_state = budget_state_value if isinstance(budget_state_value, dict) else {}
        checks["pack_observe_projected"] = (
            all(key in allowed_pack for key in pack)
            and isinstance(budget_state_value, dict)
            and set(budget_state) == {"credit_unit", "projected_cost_credits", "spend_mode"}
            and budget_state.get("spend_mode") == "projected_only"
            and budget_state.get("projected_cost_credits") == projected
            and budget_state.get("credit_unit") == primary.get("budget", {}).get("credit_unit")
        )
        if not checks["pack_observe_projected"]:
            add_reason(reasons, reason_codes, "invalid_observe_projection", "observe_projected pack carries inconsistent or billable budget state")

    primary_decision = str(primary.get("decision", ""))
    primary_reason = str(primary.get("decision_reason", ""))
    decision_profile = DECISION_REASON_PROFILES.get((primary_decision, primary_reason))
    if decision_profile == "paid_rail":
        checks["pack_paid_rail"] = "budget_state" not in pack and not contains_any_key(pack, CREDIT_USD_FIELDS)
        if not checks["pack_paid_rail"]:
            add_reason(reasons, reason_codes, "invalid_paid_rail_provenance", "payment_verified L402 Pack carries budget_state or contradictory credit/USD fields")

    checks["pack_hash_matches_primary_receipt"] = bool(primary) and pack.get("receipt_hash") == primary.get("receipt_hash")
    if not checks["pack_hash_matches_primary_receipt"]:
        add_reason(reasons, reason_codes, "pack_receipt_hash_not_primary", "pack.receipt_hash does not match primary embedded receipt.receipt_hash")

    for field in ["evidence_pack_id", "evidence_url", "receipt_id", "issuer", "issued_at", "decision", "decision_reason", "route_or_tool", "capability_hash"]:
        key = f"pack_{field}_matches_primary_receipt"
        checks[key] = bool(primary) and pack.get(field) == primary.get(field)
        if not checks[key]:
            add_reason(reasons, reason_codes, "pack_primary_mismatch", f"pack.{field} does not match primary embedded receipt.{field}")

    for field in ["authority", "budget", "policy"]:
        key = f"pack_{field}_matches_primary_receipt"
        pack_value = pack.get(field)
        receipt_value = primary.get(field) if isinstance(primary, dict) else None
        checks[key] = bool(primary) and strict_json_equal(pack_value, receipt_value)
        if not checks[key]:
            add_reason(reasons, reason_codes, "pack_primary_mismatch", f"pack.{field} mirror does not match primary embedded receipt.{field}")

    budget_value = pack.get("budget_state")
    budget: dict[str, Any] = budget_value if isinstance(budget_value, dict) else {}
    primary_budget_value = primary.get("budget")
    primary_budget: dict[str, Any] = primary_budget_value if isinstance(primary_budget_value, dict) else {}
    primary_decision = str(primary.get("decision", ""))
    primary_reason = str(primary.get("decision_reason", ""))
    decision_profile = DECISION_REASON_PROFILES.get((primary_decision, primary_reason))
    if decision_profile == "paid_rail":
        checks["budget_state_matches_primary_receipt"] = (
            bool(primary)
            and "budget_state" not in pack
            and primary_budget == {
                "spend_mode": "paid_rail",
                "rail": "l402",
                "amount_sats": primary.get("amount_sats"),
            }
        )
    elif decision_profile == "projected_only":
        checks["budget_state_matches_primary_receipt"] = (
            bool(primary)
            and budget.get("spend_mode") == "projected_only"
            and primary_budget.get("spend_mode") == "projected_only"
            and "projected_cost_credits" in budget
            and budget["projected_cost_credits"] == primary.get("projected_cost_credits")
            and budget["projected_cost_credits"] == primary_budget.get("projected_cost_credits")
            and budget.get("credit_unit") == primary_budget.get("credit_unit")
        )
    elif decision_profile == "not_evaluated":
        checks["budget_state_matches_primary_receipt"] = (
            bool(primary)
            and budget == {"spend_mode": "not_evaluated", "cost_credits": 0}
            and primary_budget == budget
            and primary.get("authority") == {"provenance_level": "no_verified_capability"}
            and pack.get("capability_hash") is None
        )
    else:
        checks["budget_state_matches_primary_receipt"] = (
            bool(primary)
            and "attempted_amount_usd" in budget
            and "attempted_amount_usd" in primary
            and budget["attempted_amount_usd"] == primary["attempted_amount_usd"]
            and "remaining_budget_usd" in budget
            and "remaining_budget_usd" in primary
            and budget["remaining_budget_usd"] == primary["remaining_budget_usd"]
            and budget.get("credit_unit") == primary_budget.get("credit_unit")
        )
    if not checks["budget_state_matches_primary_receipt"]:
        add_reason(reasons, reason_codes, "budget_state_mismatch", "pack.budget_state does not match primary embedded receipt budget fields")

    checks["no_secret_leakage"] = not contains_secret_marker(pack)
    if not checks["no_secret_leakage"]:
        add_reason(reasons, reason_codes, "secret_leakage", "pack contains a bearer/capability secret marker or unredacted secret-like field")

    summary = {
        "evidence_pack_id": pack.get("evidence_pack_id"),
        "receipt_id": pack.get("receipt_id"),
        "receipt_hash": pack.get("receipt_hash"),
        "issuer": pack.get("issuer"),
        "decision": pack.get("decision"),
        "decision_reason": pack.get("decision_reason"),
        "route_or_tool": pack.get("route_or_tool"),
        "budget_state": pack.get("budget_state"),
        "receipt_count": len(receipts),
        "trust_anchor": "issuer_jwks_anchored" if checks.get("trusted_issuer_valid") else (receipt_trust_anchors[0] if receipt_trust_anchors else "unverified"),
        "receipt_trust_anchors": receipt_trust_anchors,
        "policy": pack.get("policy"),
    }

    return {
        "valid": all(value for key, value in checks.items() if not key.endswith("trusted_issuer_valid")) and (checks.get("trusted_issuer_valid", False) if require_trusted_issuer else True),
        "trusted_issuer_valid": checks.get("trusted_issuer_valid", False),
        "protocol_profile": "issuer_jwks" if require_trusted_issuer else "artifact_integrity",
        "reason_codes": reason_codes,
        "checks": checks,
        "summary": summary,
        "reasons": reasons,
        "caveat": None if checks.get("trusted_issuer_valid") else "Receipt verified against embedded public key only. This proves artifact integrity/self-consistency; pass --jwks-url/--jwks-file and --require-trusted-issuer for issuer-anchored verification.",
    }



def validate_explicit_jwks_url(url: str) -> None:
    parsed = urllib.parse.urlparse(url)
    if parsed.scheme != "https":
        raise VerificationError("JWKS URL must use https")
    if parsed.username or parsed.password or parsed.fragment:
        raise VerificationError("JWKS URL must not contain userinfo or fragments")
    if not parsed.hostname:
        raise VerificationError("JWKS URL must include a hostname")
    if parsed.port not in {None, 443}:
        raise VerificationError("JWKS URL must use the default HTTPS port")
    try:
        host_ip = ipaddress.ip_address(parsed.hostname)
    except ValueError:
        host_ip = None
    if host_ip and (host_ip.is_private or host_ip.is_loopback or host_ip.is_link_local or host_ip.is_multicast or host_ip.is_reserved):
        raise VerificationError("JWKS URL must not target private, loopback, link-local, multicast, or reserved IP space")


def load_jwks(source: str) -> dict[str, Any]:
    if source.startswith("https://"):
        validate_explicit_jwks_url(source)
        req = urllib.request.Request(source, headers={"Accept": "application/json", "User-Agent": "satgate-evidence-pack-verifier/1"})
        try:
            with urllib.request.urlopen(req, timeout=10) as resp:
                if resp.status != 200:
                    raise VerificationError(f"JWKS fetch returned HTTP {resp.status}")
                text = resp.read().decode("utf-8")
        except urllib.error.HTTPError as exc:
            body = exc.read().decode("utf-8", errors="replace")[:500]
            raise VerificationError(f"HTTP {exc.code} fetching JWKS: {body}") from exc
        except urllib.error.URLError as exc:
            raise VerificationError(f"failed to fetch JWKS: {exc.reason}") from exc
    else:
        text = Path(source).read_text(encoding="utf-8")
    try:
        jwks = json.loads(text)
    except json.JSONDecodeError as exc:
        raise VerificationError(f"JWKS source is not JSON: {exc}") from exc
    if not isinstance(jwks, dict):
        raise VerificationError("JWKS source JSON must be an object")
    return jwks

def load_source(source: str) -> tuple[dict[str, Any], dict[str, Any]]:
    metadata: dict[str, Any] = {"source": source}
    if source == "-":
        text = sys.stdin.read()
        metadata["http_status"] = None
    elif source.startswith("http://") or source.startswith("https://"):
        req = urllib.request.Request(source, headers={"Accept": "application/json", "User-Agent": "satgate-evidence-pack-verifier/1"})
        try:
            with urllib.request.urlopen(req, timeout=15) as resp:
                metadata["http_status"] = resp.status
                metadata["content_type"] = resp.headers.get("Content-Type")
                text = resp.read().decode("utf-8")
        except urllib.error.HTTPError as exc:
            body = exc.read().decode("utf-8", errors="replace")[:500]
            raise VerificationError(f"HTTP {exc.code} fetching Evidence Pack: {body}") from exc
        except urllib.error.URLError as exc:
            raise VerificationError(f"failed to fetch Evidence Pack: {exc.reason}") from exc
    else:
        path = Path(source)
        text = path.read_text(encoding="utf-8")
        metadata["http_status"] = None

    try:
        pack = json.loads(text)
    except json.JSONDecodeError as exc:
        raise VerificationError(f"source is not JSON: {exc}") from exc
    if not isinstance(pack, dict):
        raise VerificationError("source JSON must be an object")
    return pack, metadata


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Verify a public SatGate Evidence Pack URL or JSON file.")
    parser.add_argument("source", help="Evidence Pack URL, JSON file path, or '-' for stdin")
    parser.add_argument("--pretty", action="store_true", help="pretty-print JSON output (default)")
    parser.add_argument("--jwks-url", help="HTTPS issuer JWKS URL to trust for receipt issuer_kid lookup")
    parser.add_argument("--jwks-file", help="local JWKS JSON file to trust for receipt issuer_kid lookup")
    parser.add_argument("--require-trusted-issuer", action="store_true", help="require issuer-JWKS-anchored verification (now the default; kept for explicitness/back-compat)")
    parser.add_argument("--allow-embedded-key", action="store_true", help="downgrade to embedded-key self-verification (demo/self-signed packs); proves artifact self-consistency only, NOT issuer trust")
    parser.add_argument("--discover-jwks", action="store_true", help="discover issuer JWKS from receipt.issuer at /.well-known/jwks.json (on by default in trusted mode when no --jwks-url/--jwks-file is given)")
    parser.add_argument("--allow-mock", action="store_true", help="allow packs or receipts marked mock/demo/test")
    parser.add_argument("--now", help="RFC3339 timestamp to use for temporal verification (tests/reproducibility)")
    args = parser.parse_args(argv)

    try:
        if args.jwks_url and args.jwks_file:
            raise VerificationError("use either --jwks-url or --jwks-file, not both")
        # Safe by default: require issuer-anchored verification unless the caller
        # explicitly opts into embedded-key self-verification.
        require_trusted = not args.allow_embedded_key
        has_explicit_jwks = bool(args.jwks_url or args.jwks_file)
        # In trusted mode, auto-discover the issuer JWKS when no explicit source
        # was supplied so a bare `verify_evidence_pack.py <url>` is issuer-anchored.
        discover = require_trusted and (args.discover_jwks or not has_explicit_jwks)
        jwks = load_jwks(args.jwks_url or args.jwks_file) if has_explicit_jwks else None
        pack, source_meta = load_source(args.source)
        result = verify_pack(pack, jwks=jwks, require_trusted_issuer=require_trusted, allow_mock=args.allow_mock, now=args.now, discover_jwks=discover, jwks_source=args.jwks_url)
        result.update(source_meta)
        print(json.dumps(result, indent=2, sort_keys=True))
        if not result["valid"]:
            return 1
        if args.allow_embedded_key and not result.get("trusted_issuer_valid"):
            print(
                "WARNING: verified against the embedded public key only (--allow-embedded-key). "
                "This proves the artifact is internally self-consistent, NOT that a trusted SatGate "
                "issuer signed it. A forged self-signed pack would also pass this mode. Re-run without "
                "--allow-embedded-key (optionally with --jwks-url/--jwks-file) for issuer-anchored proof.",
                file=sys.stderr,
            )
        return 0
    except VerificationError as exc:
        print(json.dumps({"valid": False, "source": args.source, "error": str(exc)}, indent=2, sort_keys=True))
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
