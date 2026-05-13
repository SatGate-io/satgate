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
ACCEPTANCE_PAGE = ROOT / "app" / "accept-satgate-capabilities" / "page.tsx"

errors: list[str] = []

for path, label in [
    (SCHEMA_JSON, "receipt schema JSON"),
    (SCHEMA_ROUTE, "receipt schema route"),
    (MOCK_RECEIPT, "mock receipt fixture"),
    (RECEIPT_DOC, "receipt schema docs"),
    (ACCEPTANCE_PAGE, "acceptance page"),
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
page_text = ACCEPTANCE_PAGE.read_text()


def _check_type(value: object, expected: object) -> bool:
    expected_types = expected if isinstance(expected, list) else [expected]
    for item in expected_types:
        if item == "object" and isinstance(value, dict):
            return True
        if item == "array" and isinstance(value, list):
            return True
        if item == "string" and isinstance(value, str):
            return True
        if item == "number" and isinstance(value, (int, float)) and not isinstance(value, bool):
            return True
        if item == "boolean" and isinstance(value, bool):
            return True
    return False


def validate_receipt_against_schema(instance: dict[str, object], schema_obj: dict[str, object], label: str) -> None:
    """Small Draft-2020-12 subset for the SatGate receipt schema.

    This intentionally avoids adding a CI dependency while still validating the
    published mock against the contract invariants this schema uses.
    """
    for field in schema_obj.get("required", []):
        if field not in instance:
            errors.append(f"{label} missing required schema field: {field}")

    properties = schema_obj.get("properties", {})
    for field, rules in properties.items():
        if field not in instance:
            continue
        value = instance[field]
        if "const" in rules and value != rules["const"]:
            errors.append(f"{label} field {field} must equal {rules['const']}")
        if "enum" in rules and value not in rules["enum"]:
            errors.append(f"{label} field {field} outside enum: {value}")
        if "type" in rules and not _check_type(value, rules["type"]):
            errors.append(f"{label} field {field} has wrong type")
        if "pattern" in rules and isinstance(value, str) and not re.match(rules["pattern"], value):
            errors.append(f"{label} field {field} does not match pattern {rules['pattern']}: {value}")
        if "minLength" in rules and isinstance(value, str) and len(value) < rules["minLength"]:
            errors.append(f"{label} field {field} is shorter than minLength")
        if "oneOf" in rules:
            matches = 0
            for option in rules["oneOf"]:
                if "type" in option and not _check_type(value, option["type"]):
                    continue
                if "exclusiveMinimum" in option and isinstance(value, (int, float)) and not value > option["exclusiveMinimum"]:
                    continue
                if "pattern" in option and isinstance(value, str) and not re.match(option["pattern"], value):
                    continue
                matches += 1
            if matches != 1:
                errors.append(f"{label} field {field} must match exactly one schema option")

    any_of = schema_obj.get("anyOf", [])
    if any_of and not any(all(field in instance for field in option.get("required", [])) for option in any_of):
        errors.append(f"{label} must satisfy at least one capability binding in anyOf")

    if instance.get("decision") == "paid":
        for field in ["amount_usd", "currency", "rail"]:
            if field not in instance:
                errors.append(f"{label} paid receipt missing required payment field: {field}")
    if "acceptor_id" in instance and "capability_hash" not in instance:
        errors.append(f"{label} acceptor-bound receipt missing capability_hash")


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

# Schema validation for the mock receipt fixture.
validate_receipt_against_schema(receipt, schema, "mock receipt")
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

for field in schema["required"] + ["capability_hash", "mock_only"]:
    if field not in page_text:
        errors.append(f"acceptance page mock receipt block missing schema-conforming field: {field}")
for needle in [
    "satgate.receipt.v1",
    "https://satgate.io/.well-known/satgate-receipt.schema.json",
    "2026-05-13T00:00:00Z",
    "jcs-rfc8785",
    "sha256",
    "ed25519",
]:
    if needle not in page_text:
        errors.append(f"acceptance page mock receipt block missing value: {needle}")
if "Verifiers MUST reject `mock_only: true` receipts when operating in a production context" not in combined_docs:
    errors.append("receipt docs must state production verifiers reject mock_only receipts")

if errors:
    print("receipt schema regression check failed:")
    for error in errors:
        print(f"- {error}")
    sys.exit(1)

print("receipt schema regression check passed")
