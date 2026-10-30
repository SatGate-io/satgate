"""Test-only signing/archive recorder for the remaining sign-every-action stops.

Driven by actual public MCP proxy calls (pkg/mcpserver every_action_gaps_test.go).
Line protocol on stdin/stdout, like mcp_unavailable_check_fixture.py. Two kinds
of record cross this boundary, each a selected projection with hashed identities
and no raw credential, tenant id or argument:

  record  an unavailable-check refusal the proxy asked to be signed;
  gap     a signed count of Observe calls that went through with no receipt,
          for one key, route and UTC minute, as a window recorder signs it.

The ephemeral Ed25519 private key stays in this process. Packs, JWKS and one
whole record are fsynced under the supplied test directory, then checked with
the public checkers: tools/, demo/ and the frozen unavailable-check-v1 bundle.
Not a hosted recorder implementation or evidence of enterprise integration.
"""
from datetime import datetime, timedelta, timezone
import json
import os
from pathlib import Path
import subprocess
import sys

import test_every_action_receipts as ea
import test_paid_rail as paid
import test_receipt_record as record

ROOT = Path(__file__).resolve().parents[1]
OUT = Path(sys.argv[1])
OUT.mkdir(parents=True, exist_ok=True)
f = paid.Fixture()
b = record.Builder(f)
packs = []
START = datetime.now(timezone.utc).replace(microsecond=0)


def stamp(dt):
    return dt.strftime("%Y-%m-%dT%H:%M:%SZ")


def durable(path, value):
    raw = value if isinstance(value, str) else json.dumps(value, sort_keys=True)
    with path.open("w", encoding="utf-8") as stream:
        stream.write(raw)
        stream.flush()
        os.fsync(stream.fileno())


def identify(r, d, n):
    r["metadata"].pop("tenant" + "_id", None)
    r.update(receipt_id=f"proxy-receipt-{n}", evidence_pack_id=f"proxy-pack-{n}",
             evidence_url=f"https://issuer.example/evidence/proxy-pack-{n}",
             route_or_tool=d["route_or_tool"], capability_hash=d["capability_hash"])
    r["authority"].update(capability_hash=d["capability_hash"], budget_id_hash=d["key_hash"])
    r["policy"]["policy_decision_mode"] = d["policy_mode"]
    r["metadata"]["request_id"] = d["request_id"]


def finish(r):
    n = len(packs) + 1
    p = paid.verified_not_evaluated_pack(f, r)
    if r.get("decision_reason") == "observe_unrecorded":
        # A gap Pack repeats issued_at and evidence_url (receipt-schema.md).
        p.update(issued_at=r["issued_at"], evidence_url=r["evidence_url"])
    f.archive(p)
    durable(OUT / f"pack-{n}.json", p)
    packs.append(p)
    return {k: p[k] for k in ("receipt_id", "receipt_hash", "evidence_pack_id", "evidence_url")}


def refusal(d):
    # Map the actual envelope, not a canned reason/decision supplied by the test.
    assert d["decision"] == "denied" and d["budget_not_evaluated"] is True
    r = paid.verified_not_evaluated(f, decision=d["decision"], decision_reason=d["reason"])
    identify(r, d, len(packs) + 1)
    r["issued_at"] = stamp(datetime.now(timezone.utc))
    return finish(r)


def gap(d, window_start, count):
    assert d["decision"] == "allowed" and d["reason"] == "observe_unrecorded" and d["budget_not_evaluated"] is True
    start = datetime.strptime(window_start, "%Y-%m-%dT%H:%M:%SZ").replace(tzinfo=timezone.utc)
    issued = max(datetime.now(timezone.utc), start + timedelta(seconds=61))
    r = ea.gap(f, count=count, unrecorded_window_start=window_start, issued_at=stamp(issued))
    identify(r, d, len(packs) + 1)
    return finish(r)


def cli(script, path, now):
    run = subprocess.run([sys.executable, "-B", str(script), str(path),
                          "--jwks-file", str(OUT / "public-jwks.json"),
                          "--require-trusted-issuer", "--now", now],
                         text=True, capture_output=True, timeout=30)
    verdict = json.loads(run.stdout)
    assert run.returncode == 0, (script.name, verdict.get("reason_codes"), verdict.get("problems"))
    assert verdict["valid"] is True and verdict["trusted_issuer_valid"] is True
    return verdict


def verify():
    durable(OUT / "public-jwks.json", f.jwks)
    # A gap is issued once its minute is over; check at a time after all of them.
    latest = max(datetime.strptime(p["issued_at"], "%Y-%m-%dT%H:%M:%SZ").replace(tzinfo=timezone.utc) for p in packs)
    now = stamp(max(datetime.now(timezone.utc), latest) + timedelta(seconds=1))
    bundle = ROOT / "tools/checker-bundles/unavailable-check-v1"
    surfaces = [ROOT / "tools", ROOT / "demo", bundle]
    pack_checks = []
    for index, p in enumerate(packs, 1):
        for surface in surfaces:
            v = cli(surface / "verify_evidence_pack.py", OUT / f"pack-{index}.json", now)
            pack_checks.append({"pack": index, "checker": str(surface.relative_to(ROOT)),
                                "reason": p["decision_reason"], "route": p["route_or_tool"],
                                "valid": v["valid"], "trusted": v["trusted_issuer_valid"]})
    key = packs[0]["authority"]["budget_id_hash"]
    terminal = b.manifest(packs, key_hash=key, **{"from": stamp(START - timedelta(minutes=5)),
                                                    "to": now, "generated_at": now})
    path = OUT / "record.ndjson"
    durable(path, record.text(packs, terminal))
    records = []
    unrecorded = None
    for surface in (ROOT / "tools", bundle):
        v = cli(surface / "verify_receipt_record.py", path, now)
        assert v["complete"] is True
        unrecorded = v["counts"]["unrecorded_calls"]
        records.append({"checker": str(surface.relative_to(ROOT)), "valid": True,
                        "trusted": True, "complete": True,
                        "unrecorded_calls": v["counts"]["unrecorded_calls"]})
    summary = {"packs": len(packs), "pack_checks": pack_checks, "record_checks": records,
               "counts_by_decision_reason": terminal["counts_by_decision_reason"],
               "unrecorded_calls": unrecorded}
    durable(OUT / "results.json", summary)
    return summary


for line in sys.stdin:
    try:
        msg = json.loads(line)
        if msg["action"] == "record":
            answer = refusal(msg["decision"])
        elif msg["action"] == "gap":
            answer = gap(msg["decision"], msg["window_start"], msg["count"])
        elif msg["action"] == "verify":
            answer = verify()
        else:
            raise ValueError("unknown fixture action")
        print(json.dumps(answer), flush=True)
    except Exception as exc:
        # No echo of the input/proxy identity. Details are checker diagnostics.
        print(json.dumps({"error": type(exc).__name__, "detail": str(exc)}), flush=True)
