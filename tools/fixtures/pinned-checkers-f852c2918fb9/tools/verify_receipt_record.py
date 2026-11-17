#!/usr/bin/env python3
"""Check a downloaded SatGate receipt record.

A record is what an owner downloads to see everything one agent key did in a
stretch of time: newline-delimited JSON (NDJSON), one signed Evidence Pack per
line in the order they were issued, then one last signed line: a complete
manifest or a cut-off terminal object. Each object uses its own issuer_kid.

Usage:
  python tools/verify_receipt_record.py record.ndjson --jwks-url https://api.satgate.io/.well-known/jwks.json --require-trusted-issuer
  python tools/verify_receipt_record.py record.ndjson --jwks-file jwks.json --require-trusted-issuer
  zcat record.ndjson.gz | python tools/verify_receipt_record.py - --jwks-file jwks.json

It needs no SatGate login. It checks:

  1. every pack, with the same checks tools/verify_evidence_pack.py makes, and
     that each pack carries exactly one receipt: the manifest's hashes cover
     that one receipt, so a second receipt in a pack would be outside them;
  2. the manifest's signature (against the issuer's JWKS, or the embedded key
     when you do not ask for a trusted issuer), and that the manifest comes
     from the same issuer as the receipts. Signing keys may differ, but in
     trusted mode every issuer_kid must be in the supplied issuer JWKS;
  3. the count: the manifest's count equals the packs in the file;
  4. nothing was dropped between the first and the last: the first and last
     receipt hashes, and a SHA-256 over the receipt hashes in order;
  5. order and range: issued_at never goes backwards and every receipt is
     inside the manifest's from (inclusive) and to (exclusive);
  6. no receipt twice (same receipt id, pack id or receipt hash);
  7. every receipt belongs to the one key the manifest names (key_hash);
  8. refusal windows: each window has at most one opener and one closer, the
     manifest lists the same windows, and a window whose closer is overdue (the
     window's end plus the 15 second grace is before the end of the export, or
     before the record was made, whichever is earlier) makes the record invalid.
     A window whose closer is not due yet, or that runs past the end of the
     range, is reported as pending or as a boundary window and does not;
  9. counts by decision reason match the manifest;
 10. the terminal schema's fixed field set, and explicit truncation status.

A signed satgate.receipt_record_truncated.v1 terminal authenticates only the
prefix before it: its counts, hashes and receipts are checked, but no complete
range or missing-closer claim is made. Its status is "truncated", valid and
complete are false, and verified_prefix and truncation_reason describe it.
Without a trusted issuer, each embedded key proves self-consistency only, not
issuer identity; trust_mode and a warning say so, including for mixed-key files.

What it cannot prove: the manifest is made at download time from what SatGate
holds. It shows nothing was dropped from that list, not that no call ever
failed to be stored. Calls before record_starts_at are not covered, and the
output says so. Calls that went through in Observe with no receipt show up as
signed gap receipts (observe_unrecorded) and are counted in the output.

Exit codes: 0 the record checks out (warnings, and pending or boundary windows,
are allowed), 1 at least one
problem, 2 the input could not be loaded, 3 the Python package 'cryptography'
is missing, 4 a signed cut-off terminal and its prefix verify (not complete).
Malformed NDJSON is a verification problem (1); file/JWKS loading and CLI usage
errors are 2. Output is JSON, with a one-line "summary" (also on stderr).
"""

from __future__ import annotations

import argparse
import hashlib
import json
import sys
from collections import Counter
from datetime import datetime, timedelta
from pathlib import Path
from typing import Any

sys.path.insert(0, str(Path(__file__).resolve().parent))

import verify_evidence_pack as vep  # noqa: E402  (same directory)

MANIFEST_SCHEMA = "satgate.receipt_record_manifest.v1"
TRUNCATED_SCHEMA = "satgate.receipt_record_truncated.v1"
TRUNCATED_EXIT = 4
PACK_SCHEMA = vep.EXPECTED_PACK_SCHEMA
# The gateway signs a window's closer once the window has been over for this
# long (RateWindowGraceSeconds in pkg/mcpserver/evidence.go). A closer that is
# still missing after window end plus this, by the end of the export, is overdue.
WINDOW_CLOSER_GRACE = timedelta(seconds=15)
MANIFEST_UNSIGNED_FIELDS = {"manifest_hash", "signature"}
# v1 is closed at the top level. Optional E3 coverage fields are signed, not
# evidence of completeness; truncated must be absent or exactly false here.
MANIFEST_FIELDS = {
    "schema_version", "canonicalization", "hash_algorithm", "signature_algorithm",
    "issuer", "issuer_kid", "metadata", "manifest_hash", "signature",
    "key_hash", "from", "to", "generated_at", "record_starts_at", "count",
    "first_receipt_hash", "last_receipt_hash", "receipt_hashes_sha256",
    "counts_by_decision_reason", "windows", "filter", "coverage",
    "legacy_identity_policy", "truncated",
}
TRUNCATED_FIELDS = MANIFEST_FIELDS | {"truncation_reason"}
KEY_HASH_FIELDS = ("token_id_hash", "budget_id_hash", "capability_hash")

Receipts = list  # of (line number, pack, primary receipt)


class Problems:
    def __init__(self) -> None:
        self.problems: list[dict[str, Any]] = []
        self.warnings: list[dict[str, Any]] = []
        self.checks: dict[str, bool] = {}
        self.window_states: list[dict[str, Any]] = []

    def check(self, name: str, ok: bool = True) -> None:
        self.checks[name] = self.checks.get(name, True) and ok

    def problem(self, check: str, code: str, message: str, line: int | None = None) -> None:
        self.check(check, False)
        entry: dict[str, Any] = {"code": code, "message": message}
        if line is not None:
            entry["line"] = line
        self.problems.append(entry)

    def warn(self, code: str, message: str) -> None:
        self.warnings.append({"code": code, "message": message})


def receipt_hash_list_digest(hashes: list[str]) -> str:
    """SHA-256 over the receipt hashes in order, each followed by a newline."""
    digest = hashlib.sha256("".join(h + "\n" for h in hashes).encode("utf-8")).digest()
    return "sha256:" + vep.b64url_encode(digest)


def manifest_payload(manifest: dict[str, Any]) -> bytes:
    if vep.rfc8785 is None:
        raise vep.VerificationError("rfc8785 canonicalization module unavailable")
    return vep.rfc8785.dumps({k: v for k, v in manifest.items() if k not in MANIFEST_UNSIGNED_FIELDS})


def manifest_hash(manifest: dict[str, Any]) -> str:
    return "sha256:" + vep.b64url_encode(hashlib.sha256(manifest_payload(manifest)).digest())


def verify_manifest_signature(manifest: dict[str, Any], jwks: dict[str, Any] | None, require_trusted_issuer: bool, p: Problems) -> None:
    """The manifest is signed over its canonical form without manifest_hash and
    signature, with its own key (found by issuer_kid). Also used for truncation
    terminals, whose signing payload and hash field are identical to E3's."""
    p.check("manifest_signature")
    if manifest.get("canonicalization") != vep.SUPPORTED_CANONICALIZATION or manifest.get("hash_algorithm") != "sha256" or manifest.get("signature_algorithm") != "ed25519":
        p.problem("manifest_signature", "manifest_unsupported_algorithm", "the manifest must be jcs-rfc8785, sha256 and ed25519")
        return
    try:
        expected = manifest_hash(manifest)
        payload = manifest_payload(manifest)
    except (vep.VerificationError, ValueError, OverflowError, RecursionError) as exc:
        p.problem("manifest_signature", "manifest_not_canonicalizable", f"the manifest could not be canonicalized ({type(exc).__name__})")
        return
    if manifest.get("manifest_hash") != expected:
        p.problem("manifest_signature", "manifest_hash_mismatch", "manifest_hash does not match the manifest")
    key_value = vep.jwks_key_for_receipt(manifest, jwks, [], []) if jwks else None
    trusted = key_value is not None
    if key_value is None:
        if require_trusted_issuer:
            p.problem("manifest_signature", "trusted_issuer_required", "a trusted issuer key (JWKS) is required and none matched the manifest's issuer_kid")
            return
        metadata = manifest.get("metadata") if isinstance(manifest.get("metadata"), dict) else {}
        key_value = metadata.get("public_key_ed25519_b64")
        if not isinstance(key_value, str) or not key_value:
            p.problem("manifest_signature", "manifest_key_missing", "no JWKS key matched and the manifest carries no embedded public key")
            return
    if vep.Ed25519PublicKey is None:
        p.problem("manifest_signature", "crypto_unavailable", vep.CRYPTO_MISSING_MESSAGE)
        return
    signature = manifest.get("signature")
    try:
        if not isinstance(signature, str) or not signature.startswith("ed25519:"):
            raise ValueError("signature must start with ed25519:")
        key_bytes = vep.b64url_decode(key_value)
        if len(key_bytes) != 32:
            raise ValueError("key must be 32 bytes")
        vep.Ed25519PublicKey.from_public_bytes(key_bytes).verify(vep.b64url_decode(signature.removeprefix("ed25519:")), payload)
    except (ValueError, vep.InvalidSignature):
        p.problem("manifest_signature", "manifest_signature_invalid", "the manifest's signature does not verify")
        return
    p.checks["manifest_trusted_issuer"] = trusted


def parse_time(value: Any) -> datetime | None:
    return vep.parse_rfc3339(value, [], [], "time")


def is_count(value: Any) -> bool:
    return type(value) is int and value >= 0


def primary_receipt(pack: dict[str, Any]) -> dict[str, Any]:
    receipts = pack.get("receipts")
    if isinstance(receipts, list) and receipts and isinstance(receipts[0], dict):
        return receipts[0]
    return {}


def authority_hashes(receipt: dict[str, Any]) -> set[str]:
    authority = receipt.get("authority") if isinstance(receipt.get("authority"), dict) else {}
    found = {authority[k] for k in KEY_HASH_FIELDS if isinstance(authority.get(k), str)}
    if isinstance(receipt.get("capability_hash"), str):
        found.add(receipt["capability_hash"])
    return found


def unreadable(message: str) -> dict[str, Any]:
    return {
        "valid": False,
        "complete": False,
        "status": "invalid",
        "trusted_issuer_valid": False,
        "trust_mode": "unverified",
        "problems": [{"code": "invalid_input", "message": message}],
        "warnings": [],
        "checks": {"input_valid": False},
        "summary": "record could not be read",
    }


def verify_record(
    text: str,
    jwks: dict[str, Any] | None = None,
    require_trusted_issuer: bool = False,
    allow_mock: bool = False,
    now: str | None = None,
) -> dict[str, Any]:
    p = Problems()
    lines: list[tuple[int, Any]] = []
    for number, raw in enumerate(text.splitlines(), start=1):
        if raw.strip():
            try:
                lines.append((number, json.loads(raw)))
            except json.JSONDecodeError:
                return unreadable(f"line {number} is not JSON")
    if not lines or not all(isinstance(obj, dict) for _, obj in lines):
        return unreadable("the record is empty or a line is not an object")

    # Exactly one terminal, complete OR truncated, and only at the end.
    terminal_schemas = (MANIFEST_SCHEMA, TRUNCATED_SCHEMA)
    manifests = [(n, o) for n, o in lines if o.get("schema_version") in terminal_schemas]
    packs = [(n, o) for n, o in lines if o.get("schema_version") not in terminal_schemas]
    p.check("manifest_present")
    if len(manifests) != 1 or lines[-1][1].get("schema_version") not in terminal_schemas:
        p.problem("manifest_present", "manifest_missing_or_misplaced", "the record must end with exactly one complete manifest or truncation terminal")
        return finish(p, len(packs), [], 0, 0)
    manifest = manifests[0][1]
    truncated = manifest.get("schema_version") == TRUNCATED_SCHEMA

    # Every pack, with the per-pack verifier.
    p.check("receipts_verified")
    p.check("receipts_trusted_issuer")
    receipts: Receipts = []
    for number, pack in packs:
        if pack.get("schema_version") != PACK_SCHEMA:
            p.problem("receipts_verified", "not_an_evidence_pack", "the line is neither a satgate.evidence_pack.v1 pack nor the manifest", number)
            continue
        embedded = pack.get("receipts")
        if not isinstance(embedded, list) or len(embedded) != 1:
            p.problem("receipts_verified", "pack_not_one_receipt", "each line must carry exactly one receipt: the manifest's hashes cover one receipt per line", number)
            continue
        verdict = vep.verify_pack(pack, jwks=jwks, require_trusted_issuer=require_trusted_issuer, allow_mock=allow_mock, now=now)
        p.check("receipts_trusted_issuer", verdict.get("trusted_issuer_valid", False))
        if not verdict.get("valid"):
            p.problem("receipts_verified", "receipt_invalid", "the pack does not verify: " + ", ".join(verdict.get("reason_codes", [])), number)
            continue
        receipts.append((number, pack, primary_receipt(pack)))

    # The manifest itself.
    verify_manifest_signature(manifest, jwks, require_trusted_issuer, p)
    p.check("manifest_fields")
    allowed_fields = TRUNCATED_FIELDS if truncated else MANIFEST_FIELDS
    if manifest.keys() - allowed_fields:
        p.problem("manifest_fields", "manifest_unknown_field", "the terminal has unknown top-level fields for its schema version")
    if truncated:
        reason = manifest.get("truncation_reason")
        if manifest.get("truncated") is not True or not isinstance(reason, str) or not reason.strip():
            p.problem("manifest_fields", "truncation_fields_invalid", "a truncation terminal requires truncated=true and a nonempty truncation_reason")
    elif "truncated" in manifest and manifest["truncated"] is not False:
        p.problem("manifest_fields", "manifest_truncated_invalid", "a complete manifest's truncated field must be absent or false; cut-off records need the truncated schema")
    # A record with no receipts has no first or last receipt.
    required = ["issuer", "issuer_kid", "key_hash", "receipt_hashes_sha256"]
    if packs:
        required += ["first_receipt_hash", "last_receipt_hash"]
    for field in required:
        if not isinstance(manifest.get(field), str) or not manifest[field]:
            p.problem("manifest_fields", "manifest_field_missing", f"manifest.{field} is missing")
    if not is_count(manifest.get("count")):
        p.problem("manifest_fields", "manifest_field_missing", "manifest.count is missing or not a count")
    counts_value = manifest.get("counts_by_decision_reason")
    if not isinstance(counts_value, dict):
        p.problem("manifest_fields", "manifest_field_missing", "manifest.counts_by_decision_reason is missing")
    elif not all(isinstance(k, str) and is_count(v) for k, v in counts_value.items()):
        # A JSON true or false compares equal to 1 and 0 in Python: a count must be an integer.
        p.problem("manifest_fields", "manifest_count_invalid", "manifest.counts_by_decision_reason must map each reason to a whole number")
    if not packs:
        # A record with no receipts has no first or last receipt: absent or null, nothing else.
        for field in ("first_receipt_hash", "last_receipt_hash"):
            if manifest.get(field) is not None:
                p.problem("manifest_fields", "manifest_endpoint_on_empty_record", f"manifest.{field} must be absent or null in a record with no receipts")
    if not isinstance(manifest.get("windows"), list):
        p.problem("manifest_fields", "manifest_field_missing", "manifest.windows is missing")
    start, end = parse_time(manifest.get("from")), parse_time(manifest.get("to"))
    generated_at = parse_time(manifest.get("generated_at"))
    if start is None or end is None or generated_at is None or not start < end:
        p.problem("manifest_fields", "manifest_range_invalid", "manifest.from, manifest.to and manifest.generated_at must be times, with from before to")
    issuer = manifest.get("issuer")

    # Count, hashes: nothing dropped between the first and the last.
    p.check("count_matches")
    if len(packs) != manifest.get("count"):
        p.problem("count_matches", "count_mismatch", f"the file has {len(packs)} receipts and the manifest says {manifest.get('count')}")
    p.check("hashes_match")
    hashes = [pack.get("receipt_hash") for _, pack, _ in receipts]
    if packs and (len(receipts) != len(packs) or not all(isinstance(h, str) for h in hashes)):
        p.problem("hashes_match", "receipt_hashes_mismatch", "some receipts could not be checked, so the list of hashes cannot be")
    else:
        if hashes and (hashes[0] != manifest.get("first_receipt_hash") or hashes[-1] != manifest.get("last_receipt_hash")):
            p.problem("hashes_match", "first_or_last_mismatch", "the first or last receipt is not the one the manifest names")
        if receipt_hash_list_digest(hashes) != manifest.get("receipt_hashes_sha256"):
            p.problem("hashes_match", "receipt_hashes_mismatch", "the receipts, in order, do not hash to the manifest's receipt_hashes_sha256: one was dropped, added, changed or moved")

    # Order and range.
    p.check("order_and_range")
    previous: datetime | None = None
    for number, _, receipt in receipts:
        issued = parse_time(receipt.get("issued_at"))
        if issued is None:
            p.problem("order_and_range", "issued_at_invalid", "issued_at is not a time", number)
            continue
        if previous is not None and issued < previous:
            p.problem("order_and_range", "out_of_order", "receipts are not in issued_at order", number)
        previous = issued
        if start is not None and end is not None and not (start <= issued < end):
            p.problem("order_and_range", "outside_range", "issued_at is outside the manifest's from..to", number)
        if isinstance(issuer, str) and receipt.get("issuer") != issuer:
            p.problem("order_and_range", "issuer_mismatch", "the receipt's issuer is not the manifest's issuer", number)

    # No receipt twice.
    p.check("no_duplicates")
    for field, from_pack in (("receipt_id", False), ("evidence_pack_id", True), ("receipt_hash", True)):
        seen: dict[Any, int] = {}
        for number, pack, receipt in receipts:
            value = (pack if from_pack else receipt).get(field)
            if value in seen:
                p.problem("no_duplicates", "duplicate_receipt", f"{field} appears twice (lines {seen[value]} and {number})", number)
            seen.setdefault(value, number)
    # A request id may repeat legitimately: an agent that reuses a JSON-RPC id
    # makes a new decision each time. It is worth a look, not a failure.
    request_ids = Counter(
        r["metadata"]["request_id"] for _, _, r in receipts
        if isinstance(r.get("metadata"), dict) and r["metadata"].get("request_id")
    )
    repeated = [rid for rid, n in request_ids.items() if n > 1]
    if repeated:
        p.warn("duplicate_request_id", f"{len(repeated)} request id(s) are on more than one receipt; each is a separate decision, check that is meant")

    # One key.
    p.check("one_key")
    for number, _, receipt in receipts:
        if manifest.get("key_hash") not in authority_hashes(receipt):
            p.problem("one_key", "key_hash_mismatch", "the receipt does not name the manifest's key_hash", number)

    # Refusal windows and gap receipts.
    p.check("windows")
    groups: dict[tuple[Any, Any, Any], dict[str, list[tuple[int, dict[str, Any], dict[str, Any]]]]] = {}
    unrecorded_calls = 0
    unrecorded_windows = 0
    for number, pack, receipt in receipts:
        if receipt.get("decision_reason") == "observe_unrecorded":
            unrecorded_windows += 1
            unrecorded_calls += receipt["unrecorded_calls"] if type(receipt.get("unrecorded_calls")) is int else 0
        phase = receipt.get("rate_limit_phase")
        if phase in ("opened", "closed"):
            key = (receipt.get("route_or_tool"), receipt.get("rate_limit_window_start"), receipt.get("denial_code"))
            groups.setdefault(key, {"opened": [], "closed": []})[phase].append((number, pack, receipt))
    computed: list[dict[str, Any]] = []
    for key in sorted(groups, key=str):
        route, window_start, code = key
        opened, closed = groups[key]["opened"], groups[key]["closed"]
        if len(opened) > 1 or len(closed) > 1:
            p.problem("windows", "window_repeated", f"the window at {window_start} for {route} has {len(opened)} opener(s) and {len(closed)} closer(s)", (opened or closed)[0][0])
        ws = parse_time(window_start)
        if not opened:
            if start is not None and ws is not None and ws < start:
                p.warn("window_started_before_record", f"the window at {window_start} began before this record; its opener is not here")
            else:
                p.problem("windows", "window_closer_without_opener", f"the window at {window_start} for {route} has a closer and no opener", closed[0][0])
        state = "closed" if closed else "open"
        if opened and not closed and truncated:
            # A closer may be in the part not downloaded. Do not apply the
            # complete-range absence claim to an authenticated prefix.
            state = "prefix_open"
        elif opened and not closed and ws is not None:
            window_end = ws + timedelta(seconds=vep.WINDOW_SECONDS)
            horizon = min((t for t in (end, generated_at) if t is not None), default=None)
            if end is not None and window_end > end:
                state = "boundary"
                p.warn("window_runs_past_range", f"the window at {window_start} for {route} runs past the end of the range; its closer is not expected in this record")
            elif horizon is not None and window_end + WINDOW_CLOSER_GRACE > horizon:
                state = "pending"
                p.warn("window_closer_pending", f"the window at {window_start} for {route} is not over, and its grace period not out, at the end of this record; its closer is not due yet")
            else:
                state = "overdue"
                p.problem("windows", "window_closer_overdue", f"the window at {window_start} for {route} was opened and its closer is overdue: the window ended more than {int(WINDOW_CLOSER_GRACE.total_seconds())} seconds before the end of the record and no closer is in it", opened[0][0])
        p.window_states.append({"route_or_tool": route, "window_start": window_start, "denial_code": code, "state": state})
        computed.append({
            "route_or_tool": route,
            "window_start": window_start,
            "denial_code": code,
            "opened_receipt_hash": opened[0][1].get("receipt_hash") if opened else None,
            "closed_receipt_hash": closed[0][1].get("receipt_hash") if closed else None,
            "refused_calls": closed[0][2].get("rate_limit_refused_calls") if closed else None,
        })
    listed = manifest.get("windows") if isinstance(manifest.get("windows"), list) else []
    if sorted(json.dumps(w, sort_keys=True) for w in listed) != sorted(json.dumps(w, sort_keys=True) for w in computed):
        p.problem("windows", "manifest_windows_mismatch", "the windows the manifest lists are not the windows in the receipts")
    if unrecorded_calls:
        p.warn("calls_without_receipts", f"{unrecorded_calls} call(s) in {unrecorded_windows} window(s) went through in Observe with no receipt; SatGate signed the count (observe_unrecorded)")

    # Counts by decision reason.
    p.check("counts_by_reason")
    by_reason = dict(Counter(str(r.get("decision_reason")) for _, _, r in receipts))
    if isinstance(counts_value, dict) and json.dumps(counts_value, sort_keys=True) != json.dumps(by_reason, sort_keys=True):
        p.problem("counts_by_reason", "counts_by_reason_mismatch", "counts_by_decision_reason does not match the receipts")

    # Where the record starts.
    if manifest.get("record_starts_at") is not None:
        record_starts = parse_time(manifest["record_starts_at"])
        if record_starts is None:
            p.problem("manifest_fields", "manifest_range_invalid", "manifest.record_starts_at is not a time")
        elif start is not None and start < record_starts:
            p.warn("record_starts_after_range", f"signing every action started at {manifest['record_starts_at']}; calls before that time in this range are not covered")

    return finish(p, len(packs), receipts, len(computed), unrecorded_calls, manifest)


def finish(p: Problems, pack_count: int, receipts: Receipts, windows: int, unrecorded_calls: int, manifest: dict[str, Any] | None = None) -> dict[str, Any]:
    refusals = sum(1 for _, _, r in receipts if r.get("decision") == "denied")
    verified = not p.problems
    truncated = verified and manifest is not None and manifest.get("schema_version") == TRUNCATED_SCHEMA
    trusted = verified and p.checks.get("manifest_trusted_issuer", False) and p.checks.get("receipts_trusted_issuer", False)
    if verified and not trusted:
        p.warn("issuer_not_authenticated", "embedded-key verification proves self-consistency only, not issuer identity")
    summary = f"{pack_count} receipts, {refusals} refusals, {windows} windows, {len(p.problems)} problems"
    if truncated:
        summary = f"truncated ({manifest['truncation_reason']}): verified prefix, " + summary
    return {
        "valid": verified and not truncated,
        "complete": verified and not truncated,
        "status": "truncated" if truncated else ("complete" if verified else "invalid"),
        "trusted_issuer_valid": trusted,
        "trust_mode": ("trusted_issuer" if trusted else "embedded_key_self_consistency") if verified else "unverified",
        "truncation_reason": manifest["truncation_reason"] if truncated else None,
        "verified_prefix": {k: manifest.get(k) for k in (
            "count", "first_receipt_hash", "last_receipt_hash", "receipt_hashes_sha256",
            "counts_by_decision_reason", "windows",
        )} if truncated else None,
        "problems": p.problems,
        "warnings": p.warnings,
        "checks": p.checks,
        "summary": summary,
        "window_states": p.window_states,
        "counts": {"receipts": pack_count, "refusals": refusals, "windows": windows, "unrecorded_calls": unrecorded_calls},
        "manifest": {k: manifest.get(k) for k in ("key_hash", "from", "to", "generated_at", "record_starts_at", "count")} if manifest else {},
    }


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(
        description="Verify a SatGate receipt record with a signed complete manifest or cut-off terminal; each issuer_kid verifies independently.",
        epilog="Exit codes: 0 complete, 1 invalid record (including malformed NDJSON), 2 input/JWKS loading or usage error, 3 missing cryptography, 4 verified truncated prefix. Embedded keys prove self-consistency, not issuer identity.",
    )
    parser.add_argument("source", help="record file (NDJSON), or '-' for stdin")
    parser.add_argument("--jwks-url", help="HTTPS issuer JWKS URL to trust")
    parser.add_argument("--jwks-file", help="local JWKS JSON file to trust")
    parser.add_argument("--require-trusted-issuer", action="store_true", help="fail unless every receipt and the complete/truncated terminal verify against the trusted JWKS")
    parser.add_argument("--allow-mock", action="store_true", help="allow receipts marked mock/demo/test")
    parser.add_argument("--now", help="RFC3339 time to use for temporal checks (tests/reproducibility)")
    args = parser.parse_args(argv)
    if vep.Ed25519PublicKey is None:
        print(vep.CRYPTO_MISSING_MESSAGE)
        return vep.CRYPTO_MISSING_EXIT
    try:
        if args.jwks_url and args.jwks_file:
            raise vep.VerificationError("use either --jwks-url or --jwks-file, not both")
        jwks = vep.load_jwks(args.jwks_url or args.jwks_file) if (args.jwks_url or args.jwks_file) else None
        text = sys.stdin.read() if args.source == "-" else Path(args.source).read_text(encoding="utf-8")
        result = verify_record(text, jwks=jwks, require_trusted_issuer=args.require_trusted_issuer, allow_mock=args.allow_mock, now=args.now)
    except (vep.VerificationError, ValueError, OSError, OverflowError, RecursionError) as exc:
        print(json.dumps({"valid": False, "error": f"Input could not be loaded or verified ({type(exc).__name__})"}, indent=2, sort_keys=True))
        return 2
    print(json.dumps(result, indent=2, sort_keys=True))
    print(result["summary"], file=sys.stderr)
    if result.get("status") == "truncated":
        return TRUNCATED_EXIT
    return 0 if result["valid"] else 1


if __name__ == "__main__":
    raise SystemExit(main())
