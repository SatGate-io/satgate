#!/usr/bin/env python3
"""Verify a public SatGate Evidence Pack.

Usage:
  python tools/verify_evidence_pack.py https://api.satgate.io/v1/evidence/evid_... --discover-jwks --require-trusted-issuer
  python tools/verify_evidence_pack.py ./evidence-pack.json --jwks-url https://api.satgate.io/.well-known/jwks.json --require-trusted-issuer
  curl -fsS ... | python tools/verify_evidence_pack.py - --discover-jwks --require-trusted-issuer

The verifier needs no SatGate auth. It validates the proof artifact itself:
- Evidence Pack schema version
- embedded receipt hash
- Ed25519 receipt signature against either a caller-supplied issuer JWKS or the embedded public key
- top-level pack mirror fields matching the embedded signed receipt
- budget-state mirror fields matching the embedded signed receipt
- obvious bearer-token/capability-secret leakage markers

By default the verifier remains backwards-compatible with embedded-key demo packs.
Use --jwks-url/--jwks-file plus --require-trusted-issuer when issuer anchoring is required.
"""

from __future__ import annotations

import argparse
import base64
import hashlib
import ipaddress
from datetime import datetime, timezone, timedelta
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
ALLOWED_DECISIONS = {"allowed", "paid", "denied"}
ALLOWED_DECISION_REASONS = {"budget_authorized", "budget_exhausted", "policy_allowed", "policy_denied", "payment_required", "capability_invalid", "capability_expired"}
DEFAULT_CLOCK_SKEW_SECONDS = 300
SECRET_PATTERNS = [
    re.compile(r"Bearer\s+[A-Za-z0-9._~+/=-]+", re.IGNORECASE),
    re.compile(r"sgcap_[A-Za-z0-9._~+/=-]+"),
    re.compile(r"stk_[A-Za-z0-9._~+/=-]+"),
]


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
        # Go's time.RFC3339Nano can emit nanosecond precision, while Python's
        # datetime parser accepts only microseconds. Truncate fractional seconds
        # to six digits for validation; the original string still participates
        # in receipt hash/signature verification unchanged.
        if "." in normalized:
            prefix, suffix = normalized.split(".", 1)
            frac = suffix
            tz = ""
            for marker in ("+", "-"):
                if marker in suffix:
                    frac, tz = suffix.split(marker, 1)
                    tz = marker + tz
                    break
            normalized = prefix + "." + frac[:6] + tz
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
    if key.get("alg") not in (None, "EdDSA"):
        add_reason(reasons, reason_codes, "malformed_jwks_key", f"issuer JWKS key {kid!r} has unsupported alg {key.get('alg')!r}")
        return None
    if key.get("use") not in (None, "sig"):
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
            if any(marker in key_lower for marker in ["raw_token", "capability_token", "secret", "authorization", "raw_budget_id", "payment_secret"]):
                if isinstance(child, str) and child and child.lower() not in {"redacted", "[redacted]"}:
                    if not key_lower.endswith("hash") and "fingerprint" not in key_lower and "public_key" not in key_lower:
                        return True
            if contains_unredacted_secret_field(child):
                return True
    elif isinstance(value, list):
        return any(contains_unredacted_secret_field(child) for child in value)
    return False


CREDIT_USD_FIELDS = {
    "amount_usd", "attempted_amount_usd", "remaining_budget_usd", "currency",
    "credit_unit", "cost_credits", "limit_credits", "remaining_before_credits",
    "remaining_after_credits", "remaining_credits", "projected_cost_credits",
}


def contains_credit_state(value):
    if isinstance(value, dict):
        return any(k in CREDIT_USD_FIELDS or contains_credit_state(v) for k, v in value.items())
    return isinstance(value, list) and any(contains_credit_state(v) for v in value)


def strict_json_equal(left: Any, right: Any) -> bool:
    """Compare JSON recursively without Python's bool/int/float coercion."""
    if type(left) is not type(right):
        return False
    if isinstance(left, dict):
        return left.keys() == right.keys() and all(
            strict_json_equal(left[key], right[key]) for key in left
        )
    if isinstance(left, list):
        return len(left) == len(right) and all(
            strict_json_equal(a, b) for a, b in zip(left, right)
        )
    return left == right


def valid_paid_rail(receipt):
    """L402 amounts are positive integer sats within the JCS safe domain.

    Keep this narrow contract aligned with the other distributed verifier;
    tools/test_paid_rail.py runs the same signed corpus against both copies.
    """
    amount = receipt.get("amount_sats")
    fields = {"rail", "payment_hash", "invoice_hash", "macaroon_hash", "amount_sats"}
    return (
        receipt.get("decision") in ("allowed", "paid")
        and receipt.get("rail") == "l402"
        and type(amount) is int and 0 < amount <= 2**53 - 1
        and all(isinstance(receipt.get(k), str) and receipt[k].strip()
                for k in ("payment_hash", "invoice_hash", "macaroon_hash"))
        and strict_json_equal(receipt.get("paid_rail_context"),
                              {k: receipt.get(k) for k in fields})
        and strict_json_equal(receipt.get("budget"),
                              {"spend_mode": "paid_rail", "rail": "l402", "amount_sats": amount})
        and not contains_credit_state(receipt)
    )


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

    paid_rail = receipt.get("decision_reason") == "payment_verified"
    if paid_rail:
        checks[f"{prefix}_paid_rail"] = bool(valid_paid_rail(receipt))
        if not checks[f"{prefix}_paid_rail"]:
            fail("invalid_paid_rail_provenance", "Incomplete or contradictory paid-rail receipt")
    checks[f"{prefix}_decision_reason"] = paid_rail or receipt.get("decision_reason") in ALLOWED_DECISION_REASONS
    if not checks[f"{prefix}_decision_reason"]:
        fail("unknown_decision_reason", f"receipts[{index}].decision_reason is not a supported protocol reason")

    mock_markers = [receipt.get("mock_only"), receipt.get("demo"), receipt.get("environment") in {"mock", "test", "demo"}]
    metadata = receipt.get("metadata") if isinstance(receipt.get("metadata"), dict) else {}
    mock_markers.extend([metadata.get("mock_only"), metadata.get("demo"), metadata.get("environment") in {"mock", "test", "demo"}])
    checks[f"{prefix}_production_artifact"] = allow_mock or not any(marker is True for marker in mock_markers)
    if not checks[f"{prefix}_production_artifact"]:
        fail("mock_artifact_not_allowed", f"receipts[{index}] is marked mock/demo/test but --allow-mock was not supplied")

    issued_at = parse_rfc3339(receipt.get("issued_at"), reasons, reason_codes, "issued_at")
    timestamp = parse_rfc3339(receipt.get("timestamp"), reasons, reason_codes, "timestamp") if receipt.get("timestamp") is not None else issued_at
    expires_at = parse_rfc3339(receipt.get("expires_at"), reasons, reason_codes, "expires_at") if receipt.get("expires_at") is not None else None
    checks[f"{prefix}_time_valid"] = (
        issued_at is not None and timestamp is not None
        and (receipt.get("expires_at") is None or expires_at is not None)
    )
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

def validate_input_shapes(pack: Any, jwks: Any) -> None:
    """Reject malformed shapes and non-JCS data before semantic operations.

    Canonicalize the entire input, including unsigned extensions: otherwise a
    nonfinite value outside the signed subtree can leak into JSON output.
    """
    if not isinstance(pack, dict):
        raise VerificationError("Evidence Pack must be an object")
    if jwks is not None and not isinstance(jwks, dict):
        raise VerificationError("JWKS must be an object")
    if rfc8785 is None:
        raise VerificationError("rfc8785 canonicalization module unavailable")
    rfc8785.dumps(pack)
    if jwks is not None:
        rfc8785.dumps(jwks)
    receipts = pack.get("receipts")
    records = [pack] + (receipts if isinstance(receipts, list) else [])
    for record in records:
        if not isinstance(record, dict):
            continue  # The normal receipt-presence check reports this shape.
        for field in ("decision", "decision_reason", "environment", "issuer"):
            if field in record and not isinstance(record[field], str):
                raise VerificationError(f"{field} must be a string")
        for field in ("authority", "budget", "policy", "metadata"):
            if record is not pack and field in record and not isinstance(record[field], dict):
                raise VerificationError(f"{field} must be an object")
        metadata = record.get("metadata") if isinstance(record.get("metadata"), dict) else {}
        if "environment" in metadata and not isinstance(metadata["environment"], str):
            raise VerificationError("metadata.environment must be a string")


def verify_pack(pack: dict[str, Any], jwks: dict[str, Any] | None = None, require_trusted_issuer: bool = False, allow_mock: bool = False, now: str | None = None, discover_jwks: bool = False, jwks_source: str | None = None) -> dict[str, Any]:
    """Public API: untrusted input failures are verdicts, never tracebacks."""
    try:
        validate_input_shapes(pack, jwks)
        return _verify_pack(pack, jwks, require_trusted_issuer, allow_mock, now, discover_jwks, jwks_source)
    except (VerificationError, ValueError, OverflowError, RecursionError) as exc:
        # Do not echo untrusted values (or bearer locators) in the error verdict.
        return {
            "valid": False, "trusted_issuer_valid": False,
            "protocol_profile": "issuer_jwks" if require_trusted_issuer else "artifact_integrity",
            "reason_codes": ["invalid_input"], "checks": {"input_valid": False},
            "summary": {}, "reasons": [f"Malformed or non-canonicalizable input ({type(exc).__name__})"],
        }


def _verify_pack(pack: dict[str, Any], jwks: dict[str, Any] | None = None, require_trusted_issuer: bool = False, allow_mock: bool = False, now: str | None = None, discover_jwks: bool = False, jwks_source: str | None = None) -> dict[str, Any]:
    reasons: list[str] = []
    reason_codes: list[str] = []
    checks: dict[str, bool] = {}
    now_dt = parse_rfc3339(now, reasons, reason_codes, "now") if now is not None else datetime.now(timezone.utc)
    if now_dt is None:
        return {
            "valid": False, "trusted_issuer_valid": False,
            "protocol_profile": "issuer_jwks" if require_trusted_issuer else "artifact_integrity",
            "reason_codes": reason_codes, "checks": {"time_valid": False},
            "summary": {}, "reasons": reasons,
        }

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

    primary = receipts[0] if receipts else {}
    # Optional display mirrors are not new authority. If present, they must
    # have a signed counterpart and match it recursively in both type and value.
    for field in ("rail", "amount_sats", "payment_hash", "invoice_hash", "macaroon_hash",
                  "paid_rail_context", "issued_at", "evidence_url", "timestamp",
                  "policy_version", "jwks_url", "schema_url", "verify_url"):
        if field in pack:
            key = f"pack_optional_{field}_matches_primary_receipt"
            checks[key] = field in primary and strict_json_equal(pack[field], primary[field])
            if not checks[key]:
                add_reason(reasons, reason_codes, "pack_primary_mismatch",
                           f"pack.{field} does not match primary embedded receipt.{field}")

    checks["pack_hash_matches_primary_receipt"] = bool(primary) and pack.get("receipt_hash") == primary.get("receipt_hash")
    if not checks["pack_hash_matches_primary_receipt"]:
        add_reason(reasons, reason_codes, "pack_receipt_hash_not_primary", "pack.receipt_hash does not match primary embedded receipt.receipt_hash")

    for field in ["evidence_pack_id", "receipt_id", "issuer", "decision", "decision_reason", "route_or_tool", "capability_hash"]:
        key = f"pack_{field}_matches_primary_receipt"
        checks[key] = bool(primary) and strict_json_equal(pack.get(field), primary.get(field))
        if not checks[key]:
            add_reason(reasons, reason_codes, "pack_primary_mismatch", f"pack.{field} does not match primary embedded receipt.{field}")

    if any(r.get("decision_reason") == "payment_verified" for r in receipts):
        checks["pack_paid_rail"] = "budget_state" not in pack and not contains_credit_state(pack)
        if not checks["pack_paid_rail"]:
            add_reason(reasons, reason_codes, "invalid_paid_rail_provenance", "Paid-rail Pack carries budget_state or credit/USD state")

    for field in ("budget", "policy", "authority"):
        if field in pack or any(r.get("decision_reason") == "payment_verified" for r in receipts):
            key = f"pack_{field}_matches_primary_receipt"
            checks[key] = strict_json_equal(pack.get(field), primary.get(field))
            if not checks[key]:
                add_reason(reasons, reason_codes, "pack_primary_mismatch",
                           f"pack.{field} does not match primary embedded receipt.{field}")

    budget = pack.get("budget_state") if isinstance(pack.get("budget_state"), dict) else {}
    checks["budget_state_matches_primary_receipt"] = bool(primary) and strict_json_equal(budget.get("attempted_amount_usd"), primary.get("attempted_amount_usd")) and strict_json_equal(budget.get("remaining_budget_usd"), primary.get("remaining_budget_usd"))
    if not checks["budget_state_matches_primary_receipt"]:
        add_reason(reasons, reason_codes, "budget_state_mismatch", "pack.budget_state does not match primary embedded receipt budget fields")

    if isinstance(pack.get("evidence_pack_hash"), str) and pack.get("evidence_pack_hash"):
        if rfc8785 is None:
            checks["evidence_pack_hash_match"] = False
            add_reason(reasons, reason_codes, "canonicalization_unavailable", "rfc8785 canonicalization module unavailable for evidence_pack_hash")
        else:
            pack_payload = {k: v for k, v in pack.items() if k != "evidence_pack_hash"}
            expected_pack_hash = "sha256:" + b64url_encode(hashlib.sha256(rfc8785.dumps(pack_payload)).digest())
            checks["evidence_pack_hash_match"] = pack.get("evidence_pack_hash") == expected_pack_hash
            if not checks["evidence_pack_hash_match"]:
                add_reason(reasons, reason_codes, "evidence_pack_hash_mismatch", "pack.evidence_pack_hash does not match canonical pack payload")

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
    parser.add_argument("--require-trusted-issuer", action="store_true", help="fail unless every receipt verifies against the trusted JWKS")
    parser.add_argument("--discover-jwks", action="store_true", help="discover issuer JWKS from receipt.issuer at /.well-known/jwks.json when trusted issuer verification is required")
    parser.add_argument("--allow-mock", action="store_true", help="allow packs or receipts marked mock/demo/test")
    parser.add_argument("--now", help="RFC3339 timestamp to use for temporal verification (tests/reproducibility)")
    args = parser.parse_args(argv)

    try:
        if args.jwks_url and args.jwks_file:
            raise VerificationError("use either --jwks-url or --jwks-file, not both")
        jwks = load_jwks(args.jwks_url or args.jwks_file) if (args.jwks_url or args.jwks_file) else None
        pack, source_meta = load_source(args.source)
        result = verify_pack(pack, jwks=jwks, require_trusted_issuer=args.require_trusted_issuer, allow_mock=args.allow_mock, now=args.now, discover_jwks=args.discover_jwks, jwks_source=args.jwks_url)
        result.update(source_meta)
        print(json.dumps(result, indent=2, sort_keys=True))
        return 0 if result["valid"] else 1
    except (VerificationError, ValueError, OSError, OverflowError, RecursionError) as exc:
        print(json.dumps({"valid": False, "reason_codes": ["invalid_input"],
                          "error": f"Input could not be loaded or verified ({type(exc).__name__})"},
                         indent=2, sort_keys=True))
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
