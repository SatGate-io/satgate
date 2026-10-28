# API Overview

Lower-level gateway API reference. Application SDK examples should use `issue`, `pay`, and `verify` where available: issue a scoped capability, invoke upstream through `pay` with a max budget, then verify the receipt.


SatGate exposes these OSS/gateway compatibility API groups on port **8080** (same port as the proxy). They are useful for local gateways and lower-level integrations; app teams should prefer issue/pay/verify when available:

## Developer-first paths

Most application integrations should start with the developer voice rather than the low-level gateway API:

- [Quickstart](../getting-started/quickstart.md)
- [Raw HTTP issue/pay/verify](../guides/raw-http.md)
- [MCP integration](../guides/mcp-gateway.md)
- [Receipt schema](../reference/receipt-schema.md)
- [Capability schema](../reference/capability-schema.md)

## Capability APIs (`/api/capability/`)

Token management and validation endpoints. Require `ADMIN_TOKEN` for minting.

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/capability/mint` | Mint a new capability token (admin only) |
| `POST` | `/api/capability/validate` | Validate a capability token |
| `POST` | `/api/capability/delegate` | Delegate a token with additional caveats (reduced permissions, tighter budget) |
| `GET` | `/api/capability/ping` | Validate token and return success (health check with auth) |
| `GET` | `/api/capability/admin` | Admin-scoped endpoint (requires admin scope) |

## Governance APIs (`/api/governance/`)

Token lifecycle and observability endpoints.

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/governance/ban` | Ban/revoke a token |
| `GET` | `/api/governance/graph` | Token lineage graph (for dashboard visualization) |
| `POST` | `/api/governance/reset` | Reset dashboard/governance data |
| `GET` | `/api/governance/evidence-pack` | Export a canonical Policy-to-Proof Evidence Pack v1 |

## Revocation

`POST /api/governance/ban` with `{"tokenSignature": "<signature>", "reason": "..."}`
bans a token by the `signature` that `/api/capability/mint` and
`/api/capability/delegate` return.

**What a ban covers.** The banned token and every token delegated from it, at
any depth, whether it was delegated before or after the ban, through
`/api/capability/delegate` or by a holder adding caveats offline. It also
covers an L402 challenge macaroon and any proof a holder narrowed from it. A
ban never covers the token's parent or its siblings. (Two delegations of the
same token with exactly the same caveats are the same token, with the same
signature.)

The gateway decides this from the token itself: it recomputes the token's
signature chain from the root key, and the signature after each caveat is the
signature of the token that caveat was added to. A token is refused when any of
those signatures is banned. A holder cannot change the chain without the token
failing verification.

**Where it is checked.** Every place the gateway verifies a token, before
anything is served or created:

| Path | Refused with |
|------|--------------|
| Capability routes, including MCP-enabled routes | `401 Token revoked` |
| L402 routes (checked before the payment proof is consumed, so a refused proof is not used up) | `401 Token revoked` |
| `/api/capability/delegate` (no child is minted or recorded) | `403`, `code: TOKEN_BANNED` |
| `/api/capability/ping`, `/api/capability/admin`, `/api/l402/mock-pay` | `403`, `code: TOKEN_BANNED` |
| `/api/capability/validate` | `401`, `valid: false`, `code: TOKEN_BANNED` |
| `/api/agent/simulate` | decision `denied`, upstream not contacted |

The standalone MCP runtime (`satgate-mcp`) checks the same bans on every
transport; see [Revoking Tokens](../guides/mcp-gateway.md#revoking-tokens) and
"The MCP runtime" below.

**Views.** Token info, the token list, the stats and the lineage graph
(`/api/governance/graph`) show a token as banned when it, or a token it was
recorded as delegated from, is banned. With Redis they read the ban list from
Redis on each call (bounded to 2 seconds, like an access check), so a ban or
unban made on any gateway shows at once; they list the tokens this gateway
recorded. If Redis cannot be read, `/api/governance/graph` answers `503` with
`code: REVOCATION_UNAVAILABLE`, and the Go views without a context show no
token as active (status `unknown`).

**Propagation.**

- Without Redis, bans are kept in process memory. They take effect at once on
  that gateway and are seen by no other process, and they are lost on restart.
  This is correct for a single gateway instance only.
- With `redis.enabled: true`, bans are written to Redis (`satgate:banlist`) and
  every gateway on the same Redis reads them on the request path through a
  short cache. A request is allowed only if, at the moment it is allowed,
  every token in its lineage (the token and each token it was delegated from)
  has a "not banned" answer from a Redis lookup that **started less than 2
  seconds** (`governance.BanCacheTTL`) before that moment, whether the answer
  was cached or just read. That moment is taken when the check holds the ban
  cache, after its last cache update, so time spent waiting for the cache
  counts. Cached answers that reach that age while another token of the
  lineage is being read, or while the check waits for the cache, are read
  again (the whole lineage in one lookup); an answer that arrives when its
  lookup started 2 seconds ago or more is not used to allow. If no such answer
  can be had for every token before the check's deadline (below), the request
  is refused with `503`. So a
  ban takes effect on every other gateway within **2 seconds** of the moment
  it was written to Redis, however slowly Redis answers, and an unban likewise.
- A ban takes effect on the gateway that receives it from the moment it is
  made: before allowing, every check reads that gateway's own bans again, so a
  request whose check was in progress when the ban was made, or while the ban
  was being written to Redis, is refused, whatever Redis answered it. A "not
  banned" answer from a lookup that started before the ban's write to Redis
  ended is never cached or used for that token afterwards, even if the cache
  dropped its entries meanwhile, so later requests are refused too.

**The MCP runtime.** `satgate-mcp` with `revocation.redis.addr` set to the
gateways' Redis reads the same ban list, with the same 2-second bound, read
deadline and failure behaviour; it does not start if that Redis cannot be
reached. Without it, `satgate-mcp` keeps bans in its own memory: they apply to
that process only, and a ban made on a gateway does not reach it. Over stdio,
every token a request supplies (`params._meta.token`, and the connection
credential of a program that embeds the proxy) is checked before anything is
signed or passed on, on every method that is, with sign-every-action on or
off; a request that supplies none is answered as before (see [Revoking
Tokens](../guides/mcp-gateway.md#revoking-tokens)).

**Failure behaviour (Redis).** Every read of the ban state is bounded to **2
seconds** (`governance.BanStoreReadTimeout`): the Redis client applies the
deadline to the socket, a failed command is retried at most once within it,
and the check returns at the deadline even if Redis has not answered. The
whole check, all its reads and any wait for the ban cache included, has one
deadline, 2 seconds from its start: a check that has not allowed by then is
refused. If Redis cannot be read in that time for a token that has no fresh
cached answer, or the deadline passes, the request is refused with `503 Service Unavailable` (`Revocation status
unavailable`, or `code: REVOCATION_UNAVAILABLE` on the API endpoints, including
`/api/agent/simulate`, which then records nothing) and nothing is served,
minted or consumed. If a ban cannot be written to Redis,
`/api/governance/ban` answers `503` with `code: REVOCATION_UNAVAILABLE`: the
gateway that received it keeps honouring the ban, the others may not, and the
ban should be retried. A gateway whose configured Redis is unreachable at
startup does not start.

## Authentication

- **Admin endpoints** (mint, ban, reset): Require `Authorization: Bearer <ADMIN_TOKEN>`
- **Evidence Pack export**: Require tenant context (`X-Tenant-ID`) and a tenant API key or admin credential on protected deployments
- **Token endpoints** (validate, delegate, ping): Require a valid capability token (macaroon)
- **Proxied routes**: Authentication depends on the route's policy (public, capability, or L402)

## Example: Mint a Token

```bash
curl -X POST http://localhost:8080/api/capability/mint \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"scope": "read", "budget": 1000}'
```

## Example: Delegate a Token

```bash
curl -X POST http://localhost:8080/api/capability/delegate \
  -H "Authorization: Bearer <parent-token>" \
  -H "Content-Type: application/json" \
  -d '{"caveats": [{"type": "budget", "limit": 100}, {"type": "expiry", "ttl": "24h"}]}'
```

The delegated token has strictly fewer permissions than its parent — this is enforced cryptographically by the macaroon structure.
