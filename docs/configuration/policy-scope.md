# Policy Scope Configuration

## Overview

SatGate Gateway supports fine-grained access control through **policy scopes**. When a route is configured with `policy.kind: observe`, `control`, or `charge`, you can optionally require a specific scope in the capability token.

## Scope Format

Scopes follow a hierarchical pattern:

```
<resource>:<action>
<resource>:<sub-resource>:<action>
<resource>:*
```

### Examples

| Scope | Description |
|-------|-------------|
| `api:read` | Read access to API |
| `api:write` | Write access to API |
| `api:*` | All API permissions |
| `admin:users:read` | Read access to admin user management |
| `admin:*` | All admin permissions |
| `*` | Full access (all scopes) |

## Route Configuration

### Gateway YAML

```yaml
routes:
  - name: "public-api"
    upstream: "backend"
    match:
      pathPrefix: "/api/v1/public"
    policy:
      kind: "public"  # No auth required

  - name: "read-api"
    upstream: "backend"
    match:
      pathPrefix: "/api/v1/data"
    policy:
      kind: "observe"   # Monitor and meter
      scope: "api:read" # Requires api:read scope

  - name: "controlled-api"
    upstream: "backend"
    match:
      pathPrefix: "/api/v1/internal"
    policy:
      kind: "control"   # Budget enforcement
      scope: "api:internal"
      budget:
        default: 10000
        period: monthly

  - name: "admin-api"
    upstream: "backend"
    match:
      pathPrefix: "/api/v1/admin"
    policy:
      kind: "observe"
      scope: "admin:*"  # Requires any admin scope
```

### Cloud Dashboard

In the Cloud Dashboard, when configuring a route:

1. Select **Policy Mode**: `observe`, `control`, or `charge`
2. Enter **Required Scope**: e.g., `api:read`
3. Configure additional options based on mode (budgets for Control, pricing for Charge)

## Scope Matching Rules

| Token Scope | Required Scope | Match? |
|-------------|----------------|--------|
| `api:read` | `api:read` | ✅ Yes |
| `api:*` | `api:read` | ✅ Yes (wildcard) |
| `*` | `api:read` | ✅ Yes (full access) |
| `api:write` | `api:read` | ❌ No |
| `admin:read` | `api:read` | ❌ No |
| `api:read` | (empty) | ✅ Yes (any valid token) |

## Default Behavior

- **Empty scope**: Any valid capability token is accepted
- **Non-empty scope**: Token must include the required scope or a parent scope

## Security Considerations

1. **Principle of Least Privilege**: Issue tokens with minimal required scopes
2. **Wildcard Caution**: Use `*` scopes sparingly (typically for admin tokens only)
3. **Scope Hierarchy**: Design your scope hierarchy before deploying
4. **Token Revocation**: Use the governance API to ban compromised tokens

## Minting Tokens with Scopes

### Admin API

```bash
curl -X POST https://gateway/api/v1/tokens \
  -H "X-Admin-Token: $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "scope": "api:read",
    "expires_in": "24h"
  }'
```

### Delegation

Tokens can be delegated with narrower scope:

```bash
# Original token has scope: api:*
# Delegated token will have scope: api:read
curl -X POST https://gateway/api/v1/tokens/delegate \
  -H "Authorization: Capability $ORIGINAL_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "scope": "api:read",
    "expires_in": "1h"
  }'
```

## Policy Mode + Scope Summary

| Policy Mode | Scope Required | What Happens |
|-------------|----------------|--------------|
| `public` | N/A | No token needed |
| `observe` | Optional | Token verified, usage metered |
| `control` | Optional | Token verified, budget enforced |
| `charge` | Optional | Token verified, payment required |

## Audit Logging

Scope enforcement is logged:

```json
{
  "level": "info",
  "msg": "Capability token authorized",
  "route": "read-api",
  "scope_required": "api:read",
  "scope_granted": "api:*",
  "policy_mode": "observe",
  "tenant_id": "..."
}
```

Failed scope checks:

```json
{
  "level": "warn",
  "msg": "Scope check failed",
  "route": "admin-api",
  "scope_required": "admin:*",
  "scope_granted": "api:read",
  "policy_mode": "observe",
  "error": "insufficient scope"
}
```
