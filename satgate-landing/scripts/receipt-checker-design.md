# Browser receipt checker

This is an adapter around the two unchanged public Python verifiers, not a new
implementation of their checks. The browser runs their `main` functions with a
MEMFS input file, a separately fetched MEMFS JWKS file, and
`--require-trusted-issuer`. The record result must also have
`checks.manifest_trusted_issuer=true`. The pack result must have
`trusted_issuer_valid=true`. Exit status, validity, and trust must all pass.
A failed record can run the same pack verifier again for its detailed check names;
that selects a dashboard sentence and cannot change the record verdict.

## Runtime supply

`npm run build` and `npm run dev` first run `prepare-receipt-checker.mjs`.
It downloads the committed URL/hash/byte-length manifest to the ignored
`public/receipt-checker/` directory, verifies every byte stream, and copies both
verifiers as bytes from `../tools`. No binary is added to Git. Pyodide 314.0.7
(Python 3.14.2) includes cryptography 47.0.0, cffi 2.0.0, pycparser 3.0 and
six 1.17.0. The pure Python rfc8785 wheel is pinned at 0.1.4.
Package hashes also agree with the pinned Pyodide lock file. Runtime URLs are
same-origin; the CDN and Python package index are build-time inputs only.
The build computes the displayed verifier hashes and raw checker size.

The two allowed issuer strings are exactly `https://api.satgate.io` and
`https://satgate-mcp-saas.fly.dev`. Every pack, embedded receipt and manifest
must name the same allowed issuer. No dropped-file URL is fetched. Key requests
omit credentials and referrer, reject redirects, time out after ten seconds,
and have a 64 KiB response cap. File size is capped at 10 MiB. Runtime failures,
unreadable keys and other issuers never become Valid. Before key CORS is live,
the checker reports that it could not check, not Not valid.

File objects go directly to a module Web Worker, not a server action or a form.
The file is read in memory, UTF-8 decoding is fatal rather than lossy, and the
original file bytes (not re-encoded text) are written to MEMFS. There is no upload,
local/session storage, IndexedDB
mount, service worker or persistent receipt cache. The worker is destroyed after
each result, a new file or component unmount. A two-minute watchdog also destroys
it. Only static checker files can be browser-cached between checks.

## Held copy and release order

`app/verify-evidence-pack/held-receipt-copy.ts` exports `HELD_RECEIPT_COPY` for the
lead, first step and complete receipt-coverage section. They remain the draft
words until the production every-action feature and its reviewed replacement
wording are ready. `receipt-checker-copy.json` pins all fixed draft text in
rendered order. Copy approval is still required.

Part B's enterprise key-handler CORS change must be live on both services before
Part A merges. Part A is also held for proof of every action in production and
replacement copy reviewed against that shipped code. Merging the public repo's
main branch publishes the landing app through its Vercel production integration;
a draft PR is not a release gate by itself. Do not merge this draft.

## Tests

Run from `satgate-landing`, against `npm run build` plus `npm start -- -p 3928`:

    python3.11 scripts/receipt-checker-corpus.py /absolute/path/to/results/corpus
    NODE_PATH=/path/to/playwright/node_modules \
      CHROMIUM_EXECUTABLE=/path/to/chrome-headless-shell \
      node scripts/test-receipt-checker.cjs /absolute/path/to/results

The result directory also needs `real-pack.json` and `mcp-jwks.json` fetched with
curl from the current page's public sample and the live MCP key endpoint. The
real key bytes are fulfilled by Playwright, unchanged, because production CORS
is not live. No fixture or interception flag enters the production bundle.
The corpus script invokes all current tools/test_*.py tests and captures every
call to the tools pack/record verifiers, checks native CLI parity, and exports
those exact arguments for browser Pyodide replay. The task's named
`tests/test_verify_evidence_pack.py` does not exist at the starting main commit.
The original malformed-input test has a Python 3.14 deepcopy recursion error
before entering the verifier; Python 3.11 runs the unchanged corpus successfully.

The browser suite checks real and synthetic files, signed-field edits, different
keys, embedded-only trust, input errors, issuer routing, blocked/malformed keys,
ignored file key URLs, drag and drop, exact served verifier bytes and displayed
hashes, fixed copy order, request bodies, storage, a rendered-text scan and 390 px
width on the checker, docs and home. It saves desktop/phone screenshots before
and after Valid and every request list. Historical-date corpus replay runs in a
test-side worker fulfilled by Playwright, never through a product test mode.

Record-only structure/count checks and the public verifier's temporal checks
have new draft sentences, listed for copy approval. Remaining unmapped checks
use the dashboard's existing fallback. The dashboard's 90-day-history-view
sentence is intentionally not reused for the public verifier's different
temporal rules.
