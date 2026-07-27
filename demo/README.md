# SatGate Public Demo

A local, synthetic, pull-only demonstration of SatGate's HTTP Policy-to-Proof path. It starts the public demo gateway and dashboard images at immutable multi-architecture digests plus pinned PostgreSQL, Redis, and WireMock dependencies.

## Boundaries

- Local demo only; every published port binds to `127.0.0.1`.
- Uses synthetic traffic and local-only committed demo credentials.
- Makes no calls to SatGate production, staging, billing, identity, model-provider, or customer systems.
- Demonstrates one configured HTTP Control route: quote before spend, no-debit price discovery, two allowed calls, budget exhaustion, and buyer-fetchable signed evidence.
- Stores local Evidence Pack bytes only in a project-scoped Docker volume removed by the documented cleanup command.
- Uses a one-shot helper from the already-pinned PostgreSQL image to assign that volume to the non-root gateway UID; the gateway itself never runs as root.
- It **does not prove** settlement, accounting, upstream truth, production readiness, durable retention/export, or MCP behavior.

## Prerequisites

- Docker with Compose
- `curl`, `jq`, and Python 3
- Free local ports `3000` and `8080`

Both SatGate images support `linux/amd64` and `linux/arm64`.

## Quick start

```bash
git clone https://github.com/SatGate-io/satgate.git
cd satgate/demo
docker compose up -d
docker compose ps
curl -fsS http://localhost:8080/healthz
curl -fsS http://localhost:3000/ >/dev/null
./golden-path.sh
```

The script prints `PASS`, the expected `200,200,402` sequence, a receipt hash, and a local Evidence Pack URL. Raw bearer tokens are used only for live local requests, never printed, and deleted on exit.

Open the dashboard at <http://localhost:3000>.

If port `3000` is already occupied, keep the gateway on `8080` and use the documented alternate dashboard port:

```bash
SATGATE_DEMO_DASHBOARD_PORT=13000 docker compose up -d
curl -fsS http://localhost:13000/ >/dev/null
```

Then open <http://localhost:13000>. The buyer defaults remain `3000` and `8080`.

## Verify the Evidence Pack

The kit uses an intentionally public, deterministic local-demo signing seed and identifies its self-signed runtime as `https://localhost` with `kid=local-demo-v1`; it never claims the canonical SatGate issuer. The key provides repeatable demo bytes, not secrecy or external trust. Create an isolated Python environment, then run the verifier against the Evidence Pack URL printed by `golden-path.sh`:

```bash
python3 -m venv .venv-verifier
. .venv-verifier/bin/activate
python -m pip install -r requirements-verifier.txt
python verify_evidence_pack.py 'http://localhost:8080/v1/evidence/evid_...' --allow-embedded-key
```

Expected result: `valid=true`, `trusted_issuer_valid=false`, and the explicit embedded-key caveat. That proves artifact integrity and internal self-consistency only. The local HTTPS issuer identity does not serve a trusted JWKS, so default trusted-issuer verification is expected to fail. Do not present this local result as buyer trust in a canonical SatGate issuer.

## Image identities

- Gateway: `ghcr.io/satgate-io/satgate-gateway-demo@sha256:b10848b4847be4b62c80a004f18af50e450937d92b9238a6c697a319880ab8c9`
- Dashboard: `ghcr.io/satgate-io/satgate-dashboard-demo@sha256:06a6a6fceeedeaa31f9fedcdbe3f73fc990dfab98172bf644985a7c9a4f7b878`
- Source revision embedded in both images: `e7df444bc6caf49966959037330c4e887665b181`

`SOURCE-MANIFEST.json` records where each public-demo file came from and whether it was copied exactly, adapted for the public boundary, or created for this kit.

## Status and logs

```bash
docker compose ps -a
docker compose logs gateway dashboard
docker compose config --images
```

## Cleanup

```bash
docker compose down -v --remove-orphans
```

After cleanup, ports `3000` and `8080` should no longer answer and the project-scoped network and volume should be gone.
