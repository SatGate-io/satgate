from __future__ import annotations

import base64
import hashlib
import json
from dataclasses import asdict, dataclass, field
from typing import Any
from urllib.parse import urlparse

import requests
import rfc8785
from cryptography.exceptions import InvalidSignature
from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PublicKey

DEFAULT_TRUSTED_ISSUERS = {"https://api.satgate.io"}


@dataclass
class VerificationResult:
    valid: bool
    trusted_issuer_valid: bool
    protocol_profile: str
    trust_anchor: str | None
    http_status: int | None
    reason_codes: list[str] = field(default_factory=list)
    evidence_url: str | None = None
    issuer: str | None = None
    issuer_kid: str | None = None
    receipt_hash: str | None = None
    evidence_pack_id: str | None = None
    receipt_id: str | None = None

    def to_dict(self) -> dict[str, Any]:
        return asdict(self)


class VerificationError(Exception):
    def __init__(self, reason: str):
        self.reason = reason
        super().__init__(reason)


def _b64url_decode(value: str) -> bytes:
    padded = value + "=" * (-len(value) % 4)
    return base64.urlsafe_b64decode(padded.encode("ascii"))


def _b64url_sha256(data: bytes) -> str:
    digest = hashlib.sha256(data).digest()
    return "sha256:" + base64.urlsafe_b64encode(digest).rstrip(b"=").decode("ascii")


def _canonical_receipt(receipt: dict[str, Any]) -> bytes:
    payload = {k: v for k, v in receipt.items() if k not in {"signature", "receipt_hash"}}
    canonical = rfc8785.dumps(payload)
    return canonical if isinstance(canonical, bytes) else canonical.encode("utf-8")


def _issuer_from_pack(pack: dict[str, Any], receipt: dict[str, Any]) -> str:
    issuer = receipt.get("issuer") or pack.get("issuer")
    if not isinstance(issuer, str):
        raise VerificationError("missing_issuer")
    parsed = urlparse(issuer)
    if parsed.scheme != "https" or not parsed.netloc or parsed.path not in {"", "/"}:
        raise VerificationError("invalid_issuer_origin")
    return issuer.rstrip("/")


def _discover_jwks(issuer: str, timeout: float) -> dict[str, Any]:
    url = issuer.rstrip("/") + "/.well-known/jwks.json"
    response = requests.get(url, timeout=timeout, headers={"Accept": "application/jwk-set+json"})
    response.raise_for_status()
    return response.json()


def _public_key_from_jwks(jwks: dict[str, Any], kid: str) -> Ed25519PublicKey:
    for key in jwks.get("keys", []):
        if key.get("kid") == kid and key.get("kty") == "OKP" and key.get("crv") == "Ed25519":
            x = key.get("x")
            if not isinstance(x, str):
                break
            return Ed25519PublicKey.from_public_bytes(_b64url_decode(x))
    raise VerificationError("issuer_key_not_found")


def _public_key_from_receipt_metadata(receipt: dict[str, Any]) -> Ed25519PublicKey:
    metadata = receipt.get("metadata")
    if not isinstance(metadata, dict):
        raise VerificationError("missing_inline_public_key")
    public_key = metadata.get("public_key_ed25519_b64")
    if not isinstance(public_key, str):
        raise VerificationError("missing_inline_public_key")
    return Ed25519PublicKey.from_public_bytes(_b64url_decode(public_key))


def _first_receipt(pack: dict[str, Any]) -> dict[str, Any]:
    receipts = pack.get("receipts")
    if not isinstance(receipts, list) or not receipts:
        raise VerificationError("missing_receipt")
    receipt = receipts[0]
    if not isinstance(receipt, dict):
        raise VerificationError("invalid_receipt")
    return receipt


def verify_evidence_pack(
    evidence_url: str,
    *,
    require_trusted_issuer: bool = False,
    discover_jwks: bool = False,
    trusted_issuers: set[str] | None = None,
    timeout: float = 20.0,
) -> VerificationResult:
    trusted_issuers = trusted_issuers or DEFAULT_TRUSTED_ISSUERS
    http_status: int | None = None
    reasons: list[str] = []
    pack: dict[str, Any] = {}
    receipt: dict[str, Any] = {}

    try:
        response = requests.get(
            evidence_url,
            timeout=timeout,
            headers={"Accept": "application/json"},
        )
        http_status = response.status_code
        response.raise_for_status()
        pack = response.json()
        if not isinstance(pack, dict):
            raise VerificationError("invalid_evidence_pack")

        receipt = _first_receipt(pack)
        issuer = _issuer_from_pack(pack, receipt)
        issuer_kid = receipt.get("issuer_kid")
        if not isinstance(issuer_kid, str):
            raise VerificationError("missing_issuer_kid")

        canonical = _canonical_receipt(receipt)
        expected_hash = _b64url_sha256(canonical)
        if receipt.get("receipt_hash") != expected_hash:
            reasons.append("receipt_hash_mismatch")
            raise VerificationError("receipt_hash_mismatch")

        signature = receipt.get("signature")
        if not isinstance(signature, str) or not signature.startswith("ed25519:"):
            raise VerificationError("missing_or_invalid_signature")
        signature_bytes = _b64url_decode(signature.removeprefix("ed25519:"))

        if discover_jwks:
            public_key = _public_key_from_jwks(_discover_jwks(issuer, timeout), issuer_kid)
            protocol_profile = "issuer_jwks"
            trust_anchor = "issuer_jwks_anchored"
        else:
            public_key = _public_key_from_receipt_metadata(receipt)
            protocol_profile = "inline_public_key"
            trust_anchor = "receipt_metadata_public_key"

        try:
            public_key.verify(signature_bytes, canonical)
        except InvalidSignature as exc:
            reasons.append("signature_invalid")
            raise VerificationError("signature_invalid") from exc

        trusted_issuer_valid = issuer in trusted_issuers
        if require_trusted_issuer and not trusted_issuer_valid:
            reasons.append("issuer_not_trusted")
            return VerificationResult(
                valid=False,
                trusted_issuer_valid=False,
                protocol_profile=protocol_profile,
                trust_anchor=trust_anchor,
                http_status=http_status,
                reason_codes=reasons,
                evidence_url=evidence_url,
                issuer=issuer,
                issuer_kid=issuer_kid,
                receipt_hash=receipt.get("receipt_hash"),
                evidence_pack_id=pack.get("evidence_pack_id"),
                receipt_id=receipt.get("receipt_id"),
            )

        return VerificationResult(
            valid=True,
            trusted_issuer_valid=trusted_issuer_valid,
            protocol_profile=protocol_profile,
            trust_anchor=trust_anchor,
            http_status=http_status,
            reason_codes=["ok"],
            evidence_url=evidence_url,
            issuer=issuer,
            issuer_kid=issuer_kid,
            receipt_hash=receipt.get("receipt_hash"),
            evidence_pack_id=pack.get("evidence_pack_id"),
            receipt_id=receipt.get("receipt_id"),
        )
    except Exception as exc:  # noqa: BLE001 - CLI/library should return structured failure.
        reason = exc.reason if isinstance(exc, VerificationError) else type(exc).__name__
        if reason not in reasons:
            reasons.append(reason)
        return VerificationResult(
            valid=False,
            trusted_issuer_valid=False,
            protocol_profile="issuer_jwks" if discover_jwks else "inline_public_key",
            trust_anchor=None,
            http_status=http_status,
            reason_codes=reasons,
            evidence_url=evidence_url,
            issuer=(receipt.get("issuer") or pack.get("issuer")) if pack else None,
            issuer_kid=receipt.get("issuer_kid") if receipt else None,
            receipt_hash=receipt.get("receipt_hash") if receipt else None,
            evidence_pack_id=pack.get("evidence_pack_id") if pack else None,
            receipt_id=receipt.get("receipt_id") if receipt else None,
        )


def result_to_json(result: VerificationResult) -> str:
    return json.dumps(result.to_dict(), indent=2, sort_keys=True)
