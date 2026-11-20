# Browser receipt checker

Ported from the checker reviewed in draft #263 at d57e2634. This page is an
adapter around the unchanged public `tools/verify_evidence_pack.py`, not a new
implementation of its checks. The worker runs its `main` function with a MEMFS
input file, a separately fetched MEMFS JWKS file, `--require-trusted-issuer`,
and an explicit `--now`. Exit status, validity, and trusted issuer must all pass.
Only check names and booleans return to React; Python diagnostics and file values
are never rendered. `ReceiptChecker.tsx` has the exhaustive, ordered mapping.
`test-receipt-checker-mapping.mjs` extracts check names from the Python AST.

## One file, one receipt

The input is the Evidence Pack downloaded from the dashboard. NDJSON records,
record manifests and truncation terminals return the full-record instruction
before keys or Python are requested. The record CLI is neither copied nor loaded;
the builder also removes any stale generated record CLI.
The public sample is fetched inside the worker and then takes the identical
input/preflight/key/Python/result path as a chosen file. No File bytes are read
on the main thread or sent to a server.

## Runtime supply

`npm run build` and `npm run dev` first run `prepare-receipt-checker.mjs`.
It downloads the committed URL/hash/byte-length manifest into the ignored
`public/receipt-checker/` directory, verifies every byte stream, and copies the
pack verifier as bytes from `../tools`. No runtime binary is added to Git.
Pyodide 314.0.7 (Python 3.14.2), cryptography 47.0.0, cffi 2.0.0, pycparser 3.0,
six 1.17.0 and rfc8785 0.1.4 retain #263's pins. Runtime URLs are same-origin;
the manifest CDN and package-index URLs are build-time inputs only. Build size
and verifier hashes are recorded in build.json, not displayed in the page.

Redistribution includes the pinned Pyodide license/source-tag notice, CPython
license and incorporated-software notices, and Emscripten/musl notices. Local
and downloaded notices go through the same integrity gate. Wheels retain their
license files. The manifest and notice source are unchanged from #263.

## Trust, clock and privacy

Issuer routing uses three exact strings, each at its own origin's
`/.well-known/jwks.json`: `https://api.satgate.io`,
`https://satgate-mcp-saas.fly.dev`, and `https://mcp.satgate.io`. The pack and all
embedded receipts must name one identical allowed issuer. No file-provided URL
is fetched. Keys must be HTTP 200, without redirects, within 64 KiB and ten
seconds. The publication must contain unique nonempty key IDs, no duplicate
JSON members, supported OKP/Ed25519 metadata, and canonical 32-byte base64url
public keys importable by Web Crypto. Unusable keys are an availability failure;
a usable publication rejecting the signature is a failed signature check.

The worker requests same-origin build.json with cache no-store and sets --now
to the later of its Date response header and the device clock. If that request
fails, it uses the device clock. Browser tests move both page and worker clocks
back an hour; a freshly signed synthetic receipt still passes, and a negative
control disabling the server bound fails on dates.

File size is capped at 10 MiB and UTF-8 decoding is fatal, not lossy. Original
bytes go to MEMFS. All worker fetches are GET, omit credentials and referrers,
reject redirects and bypass caches; worker module requests also omit credentials.
No upload, local/session storage, IndexedDB mount, service worker or persistent
receipt cache is used. The worker is terminated after each result, a replacement
check, unmount or the two-minute watchdog.

The worker script alone receives a dedicated CSP: default-src none, script-src
self plus wasm-unsafe-eval, and connect-src self plus the three exact key URLs.
No JavaScript eval or inline-script permission is added. Browser-negative tests
remove each permission separately. No page CSP change is needed; every page's
nonce policy remains unchanged and is asserted exactly by check_csp.mjs.

## Copy and tests

Only the owner-approved 2026-11-16 copy and the specified existing sections are
rendered. `receipt-checker-copy.json` pins the complete reading-order text;
every browser state is compared for equality, not substring inclusion. No held
copy module, homepage words, docs/footer link, displayed hashes or size is added.
This remains a draft PR; do not merge or deploy it as part of this work.

Run locally from `satgate-landing` after npm ci and npm run build:

    python3.11 scripts/receipt-checker-corpus.py RESULTS/corpus
    node scripts/test-receipt-checker-mapping.mjs
    node scripts/test-receipt-checker-integrity.mjs
    npm start -- --hostname 127.0.0.1 --port 3928
    node scripts/test-receipt-checker.cjs RESULTS http://127.0.0.1:3928
    CSP_TEST_ORIGIN=http://127.0.0.1:3928 node scripts/check_csp.mjs

RESULTS needs `fixtures/jwks-mcp.json` and `fixtures/jwks-api.json` containing the
published public key sets supplied for the review, and an empty shots directory.
Do not fetch live services during tests: interception supplies those exact bytes.
No fixtures, bypass flags or interception enter the production bundle.

The corpus script runs every current tools/test_*.py test, captures every call
to the pack verifier, checks native CLI parity, and exports the exact arguments
for Pyodide replay (including historical dates and nonfinite invalid inputs).
Record tests still run natively but the browser never ships a record CLI. The
unchanged malformed-input tests hit deepcopy recursion before verification on
native Python 3.14; Python 3.11 runs them without that problem.

Browser coverage includes the real sample and the other published key set,
synthetic signed/mutated inputs, all four reasons, record early return, issuer
routing, key publication failures, runtime load/crash/timeout, clock correction,
exact words and metadata, worker termination, served verifier identity, notices,
request method/body/query/credential constraints, storage, redirect and 390 px
layout. Screenshots cover empty, pass, signature failure and unavailable keys at
1440 and 390 px. Download measurement sums actual decoded response bodies for
the first check, including its sample and intercepted published keys, separately
from the initial page load. It is not a compressed-wire-size estimate.
