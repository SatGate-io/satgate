from __future__ import annotations

import argparse
import json
import sys

from .verifier import DEFAULT_TRUSTED_ISSUERS, verify_evidence_pack


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        prog="satgate-verify-evidence-pack",
        description="Verify a SatGate Evidence Pack receipt hash and Ed25519 signature.",
    )
    parser.add_argument("url", help="Evidence Pack URL, e.g. https://api.satgate.io/v1/evidence/...")
    parser.add_argument(
        "--discover-jwks",
        action="store_true",
        help="Discover issuer JWKS from the receipt issuer origin and verify with issuer key.",
    )
    parser.add_argument(
        "--require-trusted-issuer",
        action="store_true",
        help="Fail unless the issuer is in the trusted issuer allow-list.",
    )
    parser.add_argument(
        "--trusted-issuer",
        action="append",
        default=[],
        help="Trusted issuer origin. Repeatable. Defaults include https://api.satgate.io.",
    )
    parser.add_argument("--timeout", type=float, default=20.0, help="HTTP timeout in seconds.")
    return parser


def main(argv: list[str] | None = None) -> int:
    args = build_parser().parse_args(argv)
    trusted = set(DEFAULT_TRUSTED_ISSUERS)
    trusted.update(issuer.rstrip("/") for issuer in args.trusted_issuer)
    result = verify_evidence_pack(
        args.url,
        require_trusted_issuer=args.require_trusted_issuer,
        discover_jwks=args.discover_jwks,
        trusted_issuers=trusted,
        timeout=args.timeout,
    )
    print(json.dumps(result.to_dict(), indent=2, sort_keys=True))
    return 0 if result.valid else 1


if __name__ == "__main__":
    sys.exit(main())
