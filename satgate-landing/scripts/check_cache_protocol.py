#!/usr/bin/env python3
"""Regression guard for future metadata cache/freshness protocol notes."""
from __future__ import annotations

import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
REPO = ROOT.parent
CACHE_DOC = REPO / "docs" / "reference" / "cache-protocol.md"
DOC_INDEX = REPO / "docs" / "index.md"
TRUST_DOC = REPO / "docs" / "reference" / "satgate-trust-metadata.md"
ACCEPTOR_DOC = REPO / "docs" / "reference" / "acceptor.md"
PACKAGE = ROOT / "package.json"

errors: list[str] = []
for path, label in [
    (CACHE_DOC, "cache protocol doc"),
    (DOC_INDEX, "docs index"),
    (TRUST_DOC, "trust metadata doc"),
    (ACCEPTOR_DOC, "acceptor doc"),
    (PACKAGE, "package scripts"),
]:
    if not path.exists():
        errors.append(f"missing {label}: {path}")

if errors:
    print("cache protocol check failed:")
    for error in errors:
        print(f"- {error}")
    sys.exit(1)

cache_text = CACHE_DOC.read_text()
combined = "\n".join(path.read_text() for path in [DOC_INDEX, TRUST_DOC, ACCEPTOR_DOC, PACKAGE])

for needle in [
    "future-spec note — not on the wire in v0/v1",
    "/.well-known/satgate-pointer.json",
    "metadata_version",
    "metadata_digest",
    "refresh_after",
    "max-age=60",
    "must-revalidate",
    "HTTP cache headers are useful hints. They are not sufficient protocol freshness guarantees.",
    "Canonicalize JSON with JCS / RFC 8785",
    "Use `sha256`",
    "Fail closed for security-sensitive updates",
    "not required for",
    "satgate.trust_metadata.v1",
    "satgate.receipt.v1",
    "satgate.acceptor_metadata.v0",
    "satgate.evidence_pack.v1",
]:
    if needle not in cache_text:
        errors.append(f"cache protocol doc missing: {needle}")

for needle in [
    "reference/cache-protocol.md",
    "Metadata Cache Protocol Notes",
    "check_cache_protocol.py",
]:
    if needle not in combined:
        errors.append(f"cache protocol not wired/guarded: {needle}")

if errors:
    print("cache protocol check failed:")
    for error in errors:
        print(f"- {error}")
    sys.exit(1)

print("cache protocol check passed")
