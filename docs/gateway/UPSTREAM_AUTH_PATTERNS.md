# Upstream Authentication Patterns

Your upstream API likely requires authentication. This guide covers common patterns for passing auth headers through SatGate Gateway.

---

## Key Concept

SatGate Gateway uses `addHeaders` in the upstream config to inject headers into requests forwarded to your API.

```yaml
upstreams:
  my_api:
    url: "https://api.yourcompany.com"
    addHeaders:
      X-API-Key: "sk_live_abc123"  # Injected on every request
```

> ⚠️ **Important:** SatGate does **not** interpolate `${VAR}` in YAML. Values are literal strings. See [Deploy-Time Templating](#deploy-time-templating) for secrets management.

---

## Common Patterns

### Pattern 1: API Key Header

```yaml
upstreams:
  my_api:
    url: "https://api.yourcompany.com"
    addHeaders:
      X-API-Key: "sk_live_your_api_key_here"
```

### Pattern 2: Bearer Token

```yaml
upstreams:
  my_api:
    url: "https://api.yourcompany.com"
    addHeaders:
      Authorization: "Bearer eyJhbGciOiJIUzI1NiIs..."
```

### Pattern 3: Basic Auth

```yaml
upstreams:
  my_api:
    url: "https://api.yourcompany.com"
    addHeaders:
      # Base64 encode "username:password"
      Authorization: "Basic dXNlcm5hbWU6cGFzc3dvcmQ="
```

Generate the base64 value:
```bash
echo -n "username:password" | base64
# Output: dXNlcm5hbWU6cGFzc3dvcmQ=
```

### Pattern 4: Multiple Headers

```yaml
upstreams:
  my_api:
    url: "https://api.yourcompany.com"
    addHeaders:
      X-API-Key: "your-key"
      X-Tenant-ID: "tenant-123"
      X-Custom-Header: "custom-value"
```

### Pattern 5: Host Header Passthrough

By default, SatGate uses the upstream's host. To pass the original client's Host header:

```yaml
upstreams:
  my_api:
    url: "https://api.yourcompany.com"
    passHostHeader: true  # Forwards client's Host header
```

**When to use:**
- Your upstream validates the Host header
- Multi-tenant upstreams that route by Host
- Virtual hosting scenarios

---

## Deploy-Time Templating

Since SatGate doesn't interpolate `${VAR}` in YAML, use these approaches for secrets:

### Option A: envsubst (CI/CD)

Create a template file:
```yaml
# satgate.gateway.template.yaml
upstreams:
  my_api:
    url: "${UPSTREAM_URL}"
    addHeaders:
      X-API-Key: "${API_KEY}"
```

Render at deploy time:
```bash
# In your CI/CD pipeline
envsubst < satgate.gateway.template.yaml > satgate.gateway.yaml
```

### Option B: Helm (Kubernetes)

```yaml
# values.yaml
upstream:
  url: "https://api.yourcompany.com"
  apiKey: "sk_live_abc123"

# templates/configmap.yaml
upstreams:
  my_api:
    url: {{ .Values.upstream.url | quote }}
    addHeaders:
      X-API-Key: {{ .Values.upstream.apiKey | quote }}
```

### Option C: Railway/Fly Template

**Railway:** Use [railway.json](https://docs.railway.app/deploy/config-as-code) with environment variable substitution at build time.

**Fly.io:** Use [fly.toml](https://fly.io/docs/reference/configuration/) with secrets, then render config in a startup script.

Example startup script:
```bash
#!/bin/bash
# render-config.sh
cat > /app/satgate.gateway.yaml << EOF
version: 1
upstreams:
  my_api:
    url: "${UPSTREAM_URL}"
    addHeaders:
      X-API-Key: "${API_KEY}"
routes:
  # ... rest of config
EOF

exec node proxy/server.js
```

### Option D: Docker Build Args

```dockerfile
ARG API_KEY
RUN sed -i "s/__API_KEY__/${API_KEY}/g" satgate.gateway.yaml
```

```bash
docker build --build-arg API_KEY=sk_live_abc123 .
```

---

## Header Allow/Deny Lists

### Blocking Sensitive Headers

Prevent clients from injecting sensitive headers:

```yaml
upstreams:
  my_api:
    url: "https://api.yourcompany.com"
    denyRequestHeaders:
      - "x-admin-token"
      - "x-internal-key"
      - "x-debug"
      - "x-forwarded-for"  # Prevent IP spoofing
```

### Allowing Specific Headers

Only forward specific client headers:

```yaml
upstreams:
  my_api:
    url: "https://api.yourcompany.com"
    allowRequestHeaders:
      - "content-type"
      - "accept"
      - "accept-language"
      - "user-agent"
      - "x-request-id"
```

---

## Common Pitfalls

### Pitfall 1: Hardcoded Secrets in Git

**Problem:** Committing API keys to version control.

**Solution:** 
- Use deploy-time templating (see above)
- Add `satgate.gateway.yaml` to `.gitignore` if it contains secrets
- Commit `satgate.gateway.template.yaml` instead

### Pitfall 2: TLS/SSL Errors

**Problem:** `UNABLE_TO_VERIFY_LEAF_SIGNATURE` or similar TLS errors.

**Solutions:**
- Ensure upstream uses valid TLS certificate
- For self-signed certs (dev only): set `NODE_TLS_REJECT_UNAUTHORIZED=0` (not recommended for prod)
- Check certificate chain is complete

### Pitfall 3: CORS Issues

**Problem:** Browser requests blocked by CORS.

**Solution:** Configure CORS in `satgate.gateway.yaml`:

```yaml
cors:
  origins:
    - "https://your-frontend.com"
    - "http://localhost:3000"
  allowCredentials: true
```

### Pitfall 4: 502 Bad Gateway

**Problem:** Gateway can't reach upstream.

**Checklist:**
1. Is upstream URL correct? (no typos, correct port)
2. Is upstream publicly accessible? (not behind firewall)
3. Is upstream responding? (`curl` it directly)
4. Check gateway logs for specific error

### Pitfall 5: Host Header Mismatch

**Problem:** Upstream rejects requests or returns wrong content.

**Cause:** Upstream validates Host header, but gateway sends upstream's host.

**Solution:**
```yaml
upstreams:
  my_api:
    url: "https://api.yourcompany.com"
    passHostHeader: true  # Forward original Host
```

### Pitfall 6: Client Auth Header Overwritten

**Problem:** You want to inject an API key but also allow client Authorization headers.

**Solution:** Use a different header name for gateway auth:

```yaml
upstreams:
  my_api:
    url: "https://api.yourcompany.com"
    addHeaders:
      X-Gateway-Auth: "your-internal-key"  # Gateway's auth
    allowRequestHeaders:
      - "authorization"  # Client's auth passes through
```

---

## Security Recommendations

1. **Never commit secrets** - Use templating or platform secrets
2. **Use denyRequestHeaders** - Block admin/internal headers from clients
3. **Rotate keys regularly** - Update `addHeaders` values periodically
4. **Audit header flow** - Log (in dev) what headers reach your upstream
5. **Use HTTPS everywhere** - Both gateway-to-upstream and client-to-gateway

---

## Examples by Use Case

### OpenAI API Proxy

```yaml
upstreams:
  openai:
    url: "https://api.openai.com"
    addHeaders:
      Authorization: "Bearer sk-..."
    allowRequestHeaders:
      - "content-type"
```

### Stripe API Proxy

```yaml
upstreams:
  stripe:
    url: "https://api.stripe.com"
    addHeaders:
      Authorization: "Bearer sk_live_..."
    allowRequestHeaders:
      - "content-type"
      - "idempotency-key"
```

### Internal Microservice

```yaml
upstreams:
  users_service:
    url: "http://users.internal:8080"
    addHeaders:
      X-Internal-Service: "satgate"
      X-Trace-ID: "propagated-from-client"  # If you add tracing
```

### Multi-Tenant SaaS

```yaml
upstreams:
  tenant_a:
    url: "https://tenant-a.yourapp.com"
    addHeaders:
      X-Tenant-ID: "tenant-a"
      
  tenant_b:
    url: "https://tenant-b.yourapp.com"
    addHeaders:
      X-Tenant-ID: "tenant-b"
```

---

## Debugging

### Check What Headers Reach Upstream

Use [httpbin.org](https://httpbin.org) as a test upstream:

```yaml
upstreams:
  debug:
    url: "https://httpbin.org"
    addHeaders:
      X-API-Key: "test-key"
```

Then:
```bash
curl https://your-gateway/v1/basic/headers
# httpbin echoes back all headers it received
```

### Enable Gateway Debug Logging

```bash
GATEWAY_DEBUG=true node proxy/server.js
```

This logs request/response details (don't use in prod).

