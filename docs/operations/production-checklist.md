# Production Checklist

Before deploying SatGate to production, verify each item.

## Security (Critical)

- [ ] Set `ADMIN_TOKEN` to a strong random value (`openssl rand -hex 32`)
- [ ] Set `CAPABILITY_ROOT_KEY` to a cryptographically random 32+ byte value
- [ ] Set `L402_ROOT_KEY` if using L402 payments (separate from capability key)
- [ ] `admin.enableDevLogin` is `false`
- [ ] `admin.enableSwaggerUI` is `false` (or restricted by IP)
- [ ] `server.trustTenantHeaderInDev` is `false`
- [ ] No credentials in upstream URLs (use `headers` instead)
- [ ] CORS origins are explicit (no wildcards in `corsAllowedOrigins`)
- [ ] `admin.separateListener` set to internal-only address if using separate admin port
- [ ] `admin.allowedIps` restricts admin API access

## TLS

- [ ] Run SatGate behind a TLS-terminating reverse proxy (nginx, Caddy, cloud LB)
- [ ] Or use upstream TLS with proper CA verification (`tls.caCertFile`)

## Lightning (if using L402)

- [ ] Real Lightning provider configured (not `mock`)
- [ ] `lightning.requireInvoiceRecord: true` (fail-closed)
- [ ] L402 root key is separate from capability root key
- [ ] Test payment flow end-to-end

## Monitoring

- [ ] Health check available at `/health` or `/healthz`
- [ ] OpenTelemetry tracing configured if needed
- [ ] Notification webhooks configured for budget alerts

## High Availability (Enterprise)

- [ ] Redis configured for distributed budget state
- [ ] PostgreSQL configured for persistent storage
- [ ] Multiple instances behind a load balancer
- [ ] Circuit breakers configured on upstreams

## Configuration

- [ ] All secrets via environment variables, not config files
- [ ] Config validated: `./satgate --config gateway.yaml --validate`
- [ ] Rate limits set on admin API
- [ ] Upstream timeouts configured
