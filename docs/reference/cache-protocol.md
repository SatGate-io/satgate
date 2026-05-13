# SatGate Metadata Cache Protocol Notes

Status: **future-spec note — not on the wire in v0/v1**.

SatGate metadata lives on `.well-known` paths so issuers, acceptors, and verifiers can discover trust metadata, schemas, receipt formats, and JWKS locations. The live routes already publish HTTP cache headers, but protocol implementations should not assume HTTP caches are honest or timely. Corporate proxies, hosted agent runtimes, framework fetchers, and opaque intermediate caches can serve stale bytes longer than the upstream TTL.

This note records the pointer-artifact pattern before the lesson decays.

## Problem

A verifier needs to know whether its cached metadata is stale without fetching every full metadata artifact on every request.

Full artifacts can be relatively large and have longer TTLs:

- `/.well-known/satgate`
- `/.well-known/satgate.schema.json`
- `/.well-known/satgate-receipt.schema.json`
- `/.well-known/satgate-acceptor.schema.json`
- `/evidence-packs/evidence-pack.schema.v1.json`
- `/.well-known/jwks.json`

But stale metadata is security-relevant. A verifier might miss:

- a new schema field or profile requirement
- a revoked trust anchor
- a JWKS rotation
- a changed receipt decision set
- a corrected claim boundary
- a new acceptor `emitted_receipt_fields` declaration

HTTP cache headers are useful hints. They are not sufficient protocol freshness guarantees.

## Candidate pointer artifact

Candidate path:

```text
/.well-known/satgate-pointer.json
```

Candidate HTTP policy:

```text
Cache-Control: public, max-age=60, must-revalidate
Access-Control-Allow-Origin: *
X-Content-Type-Options: nosniff
Content-Type: application/json
```

Candidate body:

```json
{
  "schema_version": "satgate.pointer.v0",
  "issuer": "https://satgate.io",
  "metadata_url": "https://satgate.io/.well-known/satgate",
  "metadata_version": 12,
  "metadata_digest": "sha256:base64url-or-hex-digest",
  "metadata_updated_at": "2026-05-14T00:00:00Z",
  "refresh_after": "2026-05-14T00:01:00Z",
  "artifacts": [
    {
      "url": "https://satgate.io/.well-known/satgate",
      "kind": "issuer_metadata",
      "version": 12,
      "digest": "sha256:...",
      "refresh_after": "2026-05-14T00:01:00Z"
    },
    {
      "url": "https://satgate.io/.well-known/jwks.json",
      "kind": "jwks",
      "version": 4,
      "digest": "sha256:...",
      "next_rotation_at": "2026-06-01T00:00:00Z"
    }
  ]
}
```

The pointer is deliberately small, cheap to poll, and short-lived. It is not the trust metadata itself. It is a freshness detector for cached artifacts.

## Verifier behavior

A verifier MAY use this sequence:

1. Load cached full metadata if present.
2. Fetch `/.well-known/satgate-pointer.json` with a short timeout.
3. Compare `metadata_version` and `metadata_digest` with the cached artifact.
4. If either differs, refresh the full artifact even if the cached artifact's HTTP TTL has not expired.
5. Verify the full artifact digest against the pointer digest when present.
6. If the pointer cannot be fetched, fall back to cached metadata only within verifier policy limits.
7. Fail closed for security-sensitive updates such as trust-anchor revocation or JWKS rotation when cached freshness exceeds the verifier's maximum staleness window.

Pseudocode:

```ts
async function loadFreshMetadata(origin: string, cache: MetadataCache) {
  const cached = cache.get(`${origin}/.well-known/satgate`);
  const pointer = await tryFetchPointer(`${origin}/.well-known/satgate-pointer.json`);

  if (!pointer) {
    return useCachedWithinPolicy(cached);
  }

  if (!cached || cached.version !== pointer.metadata_version || cached.digest !== pointer.metadata_digest) {
    const metadata = await fetchJson(pointer.metadata_url);
    assertDigest(metadata, pointer.metadata_digest);
    cache.put(pointer.metadata_url, metadata, pointer.metadata_version, pointer.metadata_digest);
    return metadata;
  }

  return cached.body;
}
```

## Digest rules

Candidate digest rules should align with receipt canonicalization discipline:

- Canonicalize JSON with JCS / RFC 8785 before digesting JSON artifacts.
- Use `sha256` for v0/v1 pointer digests.
- Prefix digests with `sha256:`.
- Pin the digest algorithm in the pointer schema; do not negotiate it from untrusted data.

## Version rules

Candidate `metadata_version` rules:

- Monotonic integer per issuer origin.
- Increment whenever the issuer metadata body changes.
- Separate artifact-level `version` values can move independently under `artifacts`.
- Do not infer semantic compatibility from the integer alone; compatibility remains governed by each artifact's `schema_version`.

## Security notes

- The pointer is an optimization and stale-cache detector, not a replacement for signature/JWKS/receipt verification.
- A stale pointer can still be served by a dishonest cache; verifiers should keep a maximum staleness policy for sensitive use cases.
- A pointer should eventually be signed or covered by issuer metadata signature if SatGate adds signed metadata manifests.
- `refresh_after` is advisory. Verifier risk policy wins.
- If pointer digest and full artifact digest disagree, treat it as a fetch/cache inconsistency and refetch with cache-busting once before failing closed.

## Relationship to existing artifacts

This is not required for:

- `satgate.trust_metadata.v1`
- `satgate.receipt.v1`
- `satgate.acceptor_metadata.v0`
- `satgate.evidence_pack.v1`

It is a candidate v1/v1.1 freshness companion for metadata and schemas. The current `.well-known` artifacts remain valid without it.

## Why not inline these fields everywhere?

Fields such as `metadata_version`, `metadata_digest`, and `refresh_after` can be added to full artifacts later, but a small pointer has different operating properties:

- shorter TTL
- cheaper polling
- lower cache churn for large metadata
- one place to compare multiple artifact digests
- easier future signing or transparency-log anchoring

## Open questions

- Should the pointer cover JWKS only, metadata only, or every public protocol artifact?
- Should the pointer itself be signed in v0, or rely on HTTPS until signed metadata ships?
- Should `metadata_version` be per issuer, per artifact, or both?
- Should acceptors publish their own pointer or share the same issuer/acceptor metadata path?
- Should verifiers require pointer support for paid rails, revocation-sensitive scopes, or only high-assurance profiles?
