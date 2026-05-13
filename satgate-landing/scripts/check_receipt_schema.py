#!/usr/bin/env python3
"""Regression guard for the SatGate receipt schema and mock receipt fixture."""
from __future__ import annotations

import json
import re
import sys
from datetime import datetime
from pathlib import Path
from urllib.parse import urlparse

ROOT = Path(__file__).resolve().parents[1]
REPO = ROOT.parent
SCHEMA_JSON = ROOT / "schemas" / "satgate-receipt.schema.json"
SCHEMA_ROUTE = ROOT / "app" / ".well-known" / "satgate-receipt.schema.json" / "route.ts"
MOCK_RECEIPT = ROOT / "public" / "examples" / "mock-accepted-satgate-receipt.v1.json"
RECEIPT_DOC = REPO / "docs" / "reference" / "receipt-schema.md"
TRUST_DOC = REPO / "docs" / "reference" / "satgate-trust-metadata.md"
ACCEPTOR_DOC = REPO / "docs" / "reference" / "acceptor.md"
ISSUER_ROUTE = ROOT / "app" / ".well-known" / "satgate" / "route.ts"
DOC_INDEX = REPO / "docs" / "index.md"

errors: list[str] = []

for path, label in [
    (SCHEMA_JSON, "receipt schema JSON"),
    (SCHEMA_ROUTE, "receipt schema route"),
    (MOCK_RECEIPT, "mock receipt fixture"),
    (RECEIPT_DOC, "receipt schema docs"),
]:
    if not path.exists():
        errors.append(f"missing {label}: {path}")

if errors:
    print("receipt schema regression check failed:")
    for error in errors:
        print(f"- {error}")
    sys.exit(1)

schema = json.loads(SCHEMA_JSON.read_text())
receipt = json.loads(MOCK_RECEIPT.read_text())
route_text = SCHEMA_ROUTE.read_text()
combined_docs = "\n".join(path.read_text() for path in [RECEIPT_DOC, TRUST_DOC, ACCEPTOR_DOC, ISSUER_ROUTE, DOC_INDEX] if path.exists())

# Schema shape.
if schema.get("$id") != "https://satgate.io/.well-known/satgate-receipt.schema.json":
    errors.append("schema $id must be canonical receipt schema URL")
if schema.get("properties", {}).get("schema_version", {}).get("const") != "satgate.receipt.v1":
    errors.append("schema must const-pin schema_version to satgate.receipt.v1")
if schema.get("properties", {}).get("decision", {}).get("enum") != ["allowed", "denied", "delegated", "revoked", "paid"]:
    errors.append("schema decision enum must stay closed and ordered")
for field in ["schema_version", "schema_url", "receipt_id", "evidence_pack_id", "issuer", "issuer_kid", "decision", "decision_reason", "policy_version", "timestamp", "canonicalization", "hash_algorithm", "signature_algorithm", "receipt_hash", "signature"]:
    if field not in schema.get("required", []):
        errors.append(f"schema missing required field: {field}")
for field in ["capability_id", "capability_hash"]:
    if field not in json.dumps(schema.get("anyOf", [])):
        errors.append(f"schema anyOf must allow capability binding field: {field}")
if "acceptor_id" not in json.dumps(schema.get("allOf", [])) or "capability_hash" not in json.dumps(schema.get("allOf", [])):
    errors.append("schema must require capability_hash when acceptor_id is present")
if "amount_usd" not in json.dumps(schema.get("allOf", [])) or "rail" not in json.dumps(schema.get("allOf", [])) or "currency" not in json.dumps(schema.get("allOf", [])):
    errors.append("schema must require positive amount, currency, and rail for paid receipts")
if "exclusiveMinimum" not in json.dumps(schema.get("properties", {}).get("amount_usd", {})) or "0\\\\." not in json.dumps(schema.get("properties", {}).get("amount_usd", {})):
    errors.append("schema must constrain amount_usd to positive numeric values")

# Route and docs wiring.
for needle in ["satgate-receipt.schema.json", "Response.json", "Cache-Control", "public, max-age=86400", "Access-Control-Allow-Origin", "X-Content-Type-Options"]:
    if needle not in route_text:
        errors.append(f"receipt schema route missing: {needle}")
for needle in [
    "https://satgate.io/.well-known/satgate-receipt.schema.json",
    "satgate.receipt.v1",
    "schema_url",
    "allowed", "denied", "delegated", "revoked", "paid",
    "RFC 8785",
    "Ed25519",
    "trust-anchor validation",
    "force-refresh JWKS once",
    "reference/receipt-schema.md",
    "receipt_schema_url",
]:
    if needle not in combined_docs:
        errors.append(f"receipt docs/wiring missing: {needle}")

# Minimal schema validation for the mock receipt fixture.
for field in schema["required"]:
    if field not in receipt:
        errors.append(f"mock receipt missing required schema field: {field}")
if receipt.get("schema_version") != "satgate.receipt.v1":
    errors.append("mock receipt schema_version mismatch")
if receipt.get("schema_url") != "https://satgate.io/.well-known/satgate-receipt.schema.json":
    errors.append("mock receipt schema_url must point to canonical receipt schema")
if receipt.get("decision") not in schema["properties"]["decision"]["enum"]:
    errors.append("mock receipt decision is outside schema enum")
if not (receipt.get("capability_id") or receipt.get("capability_hash")):
    errors.append("mock receipt must include capability_id or capability_hash")
if receipt.get("acceptor_id") and not receipt.get("capability_hash"):
    errors.append("acceptor-bound mock receipt must include capability_hash")
for uri_field in ["issuer", "acceptor_id"]:
    value = receipt.get(uri_field)
    if value:
        parsed = urlparse(value)
        if parsed.scheme not in {"https", "http"} or not parsed.netloc:
            errors.append(f"mock receipt {uri_field} must be URI-like: {value}")
try:
    datetime.fromisoformat(receipt["timestamp"].replace("Z", "+00:00"))
except Exception:
    errors.append("mock receipt timestamp must be date-time parseable")
for field, prefix in [("receipt_hash", "sha256:"), ("capability_hash", "sha256:"), ("signature", "ed25519:")]:
    value = receipt.get(field)
    if value and not str(value).startswith(prefix):
        errors.append(f"mock receipt {field} must start with {prefix}")
if receipt.get("mock_only") is not True:
    errors.append("mock receipt fixture must remain mock_only: true")

if errors:
    print("receipt schema regression check failed:")
    for error in errors:
        print(f"- {error}")
    sys.exit(1)

print("receipt schema regression check passed")
