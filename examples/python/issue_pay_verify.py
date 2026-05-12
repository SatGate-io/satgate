#!/usr/bin/env python3
"""Minimal SatGate issue/pay/verify private-beta example.

Run from the repo root against local SDK source:

    PYTHONPATH=sdk/python SATGATE_API_KEY=sg_live_... python3 examples/python/issue_pay_verify.py

Without a beta key this exits cleanly with SatGateAuthError and the docs CTA.
"""

import os

from satgate import SatGate, SatGateAuthError


def main() -> int:
    satgate = SatGate(api_key=os.getenv("SATGATE_API_KEY"))

    try:
        capability = satgate.issue(
            task="research market prices",
            agent="research-agent",
            allow=["https://api.example.com/search"],
            budget_usd=2.00,
            expires_in="10m",
        )

        receipt = satgate.pay(
            upstream="https://api.example.com/search",
            capability=capability,
            max_usd=0.25,
        )

        verified = satgate.verify(receipt)
        print(verified.decision, verified.evidence_pack_id)
        return 0
    except SatGateAuthError as exc:
        print(f"SatGateAuthError: {exc}")
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
