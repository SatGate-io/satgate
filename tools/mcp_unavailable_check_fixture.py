"""Test-only signing/archive recorder driven by actual public MCP proxy calls.

Line protocol on stdin/stdout; only a selected decision projection (hashed
identities, no raw credentials, tenant ids or arguments) crosses this boundary.
The ephemeral Ed25519 private key stays in this process. Public packs, JWKS and
a whole record are fsynced under the supplied test directory. Not a hosted
recorder implementation or evidence of enterprise integration.
"""
import copy
import json
import os
from pathlib import Path
import subprocess
import sys

import test_paid_rail as paid
import test_receipt_record as record

ROOT = Path(__file__).resolve().parents[1]
OUT = Path(sys.argv[1])
OUT.mkdir(parents=True, exist_ok=True)
f = paid.Fixture()
b = record.Builder(f)
packs = []
mode = ""


def durable(path, value):
    raw = value if isinstance(value, str) else json.dumps(value, sort_keys=True)
    with path.open("w", encoding="utf-8") as stream:
        stream.write(raw)
        stream.flush()
        os.fsync(stream.fileno())


def emit(d):
    if mode == "sign":
        return {"error": "sign unavailable"}
    assert d["decision"] == "denied"
    assert d["budget_not_evaluated"] is True
    # Map the actual envelope, not a canned reason/decision supplied by the test.
    r = paid.verified_not_evaluated(f, decision=d["decision"], decision_reason=d["reason"])
    r["metadata"].pop("tenant" + "_id", None)
    n = len(packs) + 1
    r.update(receipt_id=f"proxy-refusal-{n}", evidence_pack_id=f"proxy-pack-{n}",
             evidence_url=f"https://issuer.example/evidence/proxy-pack-{n}",
             route_or_tool=d["route_or_tool"], capability_hash=d["capability_hash"])
    r["authority"].update(capability_hash=d["capability_hash"], budget_id_hash=d["key_hash"])
    r["policy"]["policy_decision_mode"] = d["policy_mode"]
    r["metadata"]["request_id"] = d["request_id"]
    p = paid.verified_not_evaluated_pack(f, r)
    f.archive(p)
    if mode == "archive":
        return {"error": "archive unavailable"}
    durable(OUT / f"pack-{n}.json", p)
    packs.append(p)
    return {k: p[k] for k in ("receipt_id", "receipt_hash", "evidence_pack_id", "evidence_url")}


def cli(script, path):
    run = subprocess.run([sys.executable, "-B", str(script), str(path),
                          "--jwks-file", str(OUT / "public-jwks.json"),
                          "--require-trusted-issuer", "--now", paid.NOW],
                         text=True, capture_output=True, timeout=30)
    verdict = json.loads(run.stdout)
    assert run.returncode == 0, (script.name, verdict.get("reason_codes"), verdict.get("problems"))
    assert verdict["valid"] is True and verdict["trusted_issuer_valid"] is True
    return verdict


def verify():
    durable(OUT / "public-jwks.json", f.jwks)
    bundle = ROOT / "tools/checker-bundles/unavailable-check-v1"
    surfaces = [ROOT / "tools", ROOT / "demo", bundle]
    pack_checks = []
    for index, p in enumerate(packs, 1):
        for surface in surfaces:
            v = cli(surface / "verify_evidence_pack.py", OUT / f"pack-{index}.json")
            pack_checks.append({"pack": index, "checker": str(surface.relative_to(ROOT)),
                                "reason": p["decision_reason"], "valid": v["valid"],
                                "trusted": v["trusted_issuer_valid"], "exit_code": 0})
    key = packs[0]["authority"]["budget_id_hash"]
    terminal = b.manifest(packs, key_hash=key)
    path = OUT / "record.ndjson"
    durable(path, record.text(packs, terminal))
    records = []
    for surface in (ROOT / "tools", bundle):
        v = cli(surface / "verify_receipt_record.py", path)
        assert v["complete"] is True
        records.append({"checker": str(surface.relative_to(ROOT)), "valid": True,
                        "trusted": True, "complete": True, "exit_code": 0})
    summary = {"packs": len(packs), "pack_checks": pack_checks, "record_checks": records,
               "counts_by_decision_reason": terminal["counts_by_decision_reason"]}
    durable(OUT / "results.json", summary)
    return summary


for line in sys.stdin:
    try:
        msg = json.loads(line)
        if msg["action"] == "record":
            answer = emit(msg["decision"])
        elif msg["action"] == "failure":
            mode = msg["mode"]
            answer = {}
        elif msg["action"] == "verify":
            answer = verify()
        else:
            raise ValueError("unknown fixture action")
        print(json.dumps(answer), flush=True)
    except Exception as exc:
        # No echo of the input/proxy identity. Details are checker diagnostics.
        print(json.dumps({"error": type(exc).__name__, "detail": str(exc)}), flush=True)
