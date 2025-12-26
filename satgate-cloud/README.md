# SatGate Cloud

**Hosted L402 Gateway for SMBs** — Connect your API, set pricing, go live in minutes.

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                        SatGate Cloud                                │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌─────────────────────┐         ┌─────────────────────┐           │
│  │   Control Plane     │         │    Data Plane       │           │
│  │   (api.satgate.io)  │         │  (*.satgate.cloud)  │           │
│  │                     │         │                     │           │
│  │  • Auth (magic link)│         │  • Multi-tenant GW  │           │
│  │  • Project CRUD     │         │  • Config loader    │           │
│  │  • Config mgmt      │         │  • L402 enforcement │           │
│  │  • Secrets vault    │         │  • Invoice creation │           │
│  │  • Usage dashboard  │         │  • Usage logging    │           │
│  └──────────┬──────────┘         └──────────┬──────────┘           │
│             │                                │                      │
│             ▼                                ▼                      │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │                     PostgreSQL                               │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │                     Managed Lightning (phoenixd)             │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

## Project Structure

```
satgate-cloud/
├── apps/
│   ├── control-plane/     # Auth, projects, configs, usage API
│   └── data-plane/        # Multi-tenant gateway
├── packages/
│   ├── gateway-config/    # Config schema, validation, normalization
│   ├── l402-core/         # Macaroon, challenge, validation
│   └── common/            # Logging, errors, IDs, HTTP utils
└── db/
    └── schema.sql         # PostgreSQL schema
```

## Quick Start

```bash
# Install dependencies
pnpm install

# Build all packages
pnpm build

# Start control plane (dev)
pnpm --filter @satgate/control-plane dev

# Start data plane (dev)
pnpm --filter @satgate/data-plane dev
```

## Packages

### @satgate/gateway-config

Shared config logic:
- JSON Schema validation
- Cloud policy validation (HTTPS only, no private upstreams)
- Config normalization
- Route summary generation

### @satgate/l402-core

L402 protocol implementation:
- SimpleMacaroon (JSON-based, HMAC-SHA256)
- Challenge creation (402 + invoice + macaroon)
- LSAT validation
- Caveat parsing (exp, aud, scope, etc.)

### @satgate/common

Shared utilities:
- ID generation (nanoid)
- Error classes
- Structured logging
- HTTP header utilities

## Database

PostgreSQL schema in `db/schema.sql`:

- `tenants` - Users
- `auth_codes` - Magic link codes (hashed, single-use)
- `sessions` - Session tokens
- `projects` - Customer projects (slug → gateway URL)
- `config_versions` - Immutable config history
- `project_secrets` - Encrypted secrets for upstream auth
- `api_keys` - Automation keys
- `usage_events` - Time-series usage data
- `usage_daily` - Aggregated daily stats

## Environment Variables

### Control Plane

```bash
PORT=3000
DATABASE_URL=postgres://...
SMTP_HOST=smtp.example.com
SMTP_USER=...
SMTP_PASS=...
SESSION_SECRET=<random-32-chars>
```

### Data Plane

```bash
PORT=8080
DATABASE_URL=postgres://...
L402_ROOT_KEY=<random-32-chars>
PHOENIXD_URL=http://phoenixd:9740
PHOENIXD_PASSWORD=...
```

## Deployment

Recommended: Fly.io for both control plane and data plane.

```bash
# Control plane
cd apps/control-plane
fly launch
fly deploy

# Data plane (with wildcard TLS)
cd apps/data-plane
fly launch --name satgate-data
fly certs add "*.satgate.cloud"
fly deploy
```

## API Endpoints

### Control Plane

| Method | Path | Description |
|--------|------|-------------|
| POST | `/auth/magic-link` | Send magic link email |
| POST | `/auth/verify` | Verify code, create session |
| POST | `/auth/logout` | End session |
| GET | `/api/projects` | List my projects |
| POST | `/api/projects` | Create project |
| GET | `/api/projects/:slug` | Get project |
| DELETE | `/api/projects/:slug` | Delete project |
| POST | `/api/projects/:slug/config` | Upload config |
| GET | `/api/projects/:slug/config` | Get active config |
| GET | `/api/projects/:slug/usage` | Get usage stats |
| POST | `/api/projects/:slug/test` | Test a route |

### Data Plane

| Method | Path | Description |
|--------|------|-------------|
| GET | `/healthz` | Health check |
| * | `/*` | Tenant gateway (L402 enforcement + proxy) |

## License

Proprietary - SatGate, Inc.

