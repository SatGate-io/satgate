# L402 Response Schema

This document defines the standard response format for L402 (402 Payment Required) challenges issued by SatGate.

## Overview

When a client requests a protected endpoint without a valid L402 token, SatGate returns a `402 Payment Required` response with all the information needed to complete payment and retry.

## Response Format

### HTTP Headers

```http
HTTP/1.1 402 Payment Required
Content-Type: application/json
WWW-Authenticate: L402 macaroon="<base64-encoded-macaroon>", invoice="<lightning-invoice>"
```

### Response Body

```json
{
  "status": 402,
  "type": "L402",
  "message": "Payment required to access this resource",
  "offer": {
    "endpoint": "/api/premium/data",
    "method": "GET",
    "price_sats": 10,
    "description": "Premium data access"
  },
  "payment": {
    "invoice": "lnbc100n1p3...",
    "macaroon": "AgELc2F0Z2F0ZS5pbwI...",
    "expires_at": "2025-12-04T12:00:00Z"
  },
  "instructions": {
    "step_1": "Pay the Lightning invoice",
    "step_2": "Extract the preimage from payment confirmation",
    "step_3": "Retry request with Authorization header",
    "header_format": "Authorization: L402 <macaroon>:<preimage>"
  }
}
```

## Field Definitions

### Root Fields

| Field | Type | Description |
|-------|------|-------------|
| `status` | integer | HTTP status code (always 402) |
| `type` | string | Protocol identifier (always "L402") |
| `message` | string | Human-readable description |
| `offer` | object | What's being sold |
| `payment` | object | How to pay |
| `instructions` | object | How to complete the flow |

### Offer Object

| Field | Type | Description |
|-------|------|-------------|
| `endpoint` | string | The protected resource path |
| `method` | string | HTTP method (GET, POST, etc.) |
| `price_sats` | integer | Price in satoshis |
| `description` | string | Human-readable description of the resource |

### Payment Object

| Field | Type | Description |
|-------|------|-------------|
| `invoice` | string | BOLT11 Lightning invoice to pay |
| `macaroon` | string | Base64-encoded macaroon (capability token) |
| `expires_at` | string | ISO 8601 timestamp when invoice expires |

### Instructions Object

| Field | Type | Description |
|-------|------|-------------|
| `step_1` | string | First step instruction |
| `step_2` | string | Second step instruction |
| `step_3` | string | Third step instruction |
| `header_format` | string | Exact format for Authorization header |

## Macaroon Caveats

The macaroon may contain caveats that constrain its use:

```json
{
  "caveats": [
    { "type": "scope", "value": "/api/premium/*" },
    { "type": "expiry", "value": 1735689600 },
    { "type": "budget", "value": 1000 },
    { "type": "calls", "value": 10 }
  ]
}
```

| Caveat | Description |
|--------|-------------|
| `scope` | Limits token to specific route patterns |
| `expiry` | Unix timestamp after which token is invalid |
| `budget` | Maximum sats that can be spent with this token |
| `calls` | Maximum number of API calls allowed |

## Successful Retry

After payment, retry the original request with the L402 token:

```http
GET /api/premium/data HTTP/1.1
Host: api.example.com
Authorization: L402 AgELc2F0Z2F0ZS5pbwI...:abc123preimage456
```

### Success Response

```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "data": { ... }
}
```

## Error Responses

### Invalid Token

```json
{
  "status": 401,
  "error": "invalid_token",
  "message": "The L402 token is invalid or malformed"
}
```

### Expired Token

```json
{
  "status": 401,
  "error": "token_expired",
  "message": "The L402 token has expired"
}
```

### Budget Exceeded

```json
{
  "status": 402,
  "error": "budget_exceeded",
  "message": "Token budget exhausted. New payment required.",
  "payment": { ... }
}
```

### Scope Violation

```json
{
  "status": 403,
  "error": "scope_violation",
  "message": "Token does not grant access to this endpoint"
}
```

## SDK Implementation

### Python

```python
from satgate import SatGateClient

client = SatGateClient(wallet=my_wallet)

# Automatic: handles 402 → pay → retry
response = client.get("https://api.example.com/api/premium/data")
```

### Node.js

```javascript
import { SatGateClient } from 'satgate-sdk';

const client = new SatGateClient();

// Automatic: handles 402 → pay → retry
const data = await client.get('https://api.example.com/api/premium/data');
```

### cURL (Manual)

```bash
# 1. Initial request
curl -i https://api.example.com/api/premium/data
# Returns 402 with invoice + macaroon

# 2. Pay invoice (via your Lightning wallet)
# Receive preimage: abc123preimage456

# 3. Retry with L402 token
curl -H "Authorization: L402 AgELc2F0Z2F0ZS5pbwI...:abc123preimage456" \
  https://api.example.com/api/premium/data
```

## Agent-Friendly Design

This schema is designed for autonomous agents:

1. **Machine-parseable**: JSON format with consistent field names
2. **Self-documenting**: Instructions embedded in response
3. **Stateless**: No session required between requests
4. **Capability-based**: Token encodes permissions, not identity

Agents can implement "pay and continue" without custom parsing—the schema provides everything needed to complete the L402 flow programmatically.

## See Also

- [L402 Protocol Specification](https://github.com/lightninglabs/L402)
- [SatGate Documentation](https://github.com/SatGate-io/satgate)
- [llms.txt](https://satgate.io/llms.txt) - Machine-readable spec for AI assistants

