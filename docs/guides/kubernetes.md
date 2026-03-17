# Kubernetes Deployment Guide

Deploy SatGate Gateway to production Kubernetes.

## Prerequisites

- Kubernetes cluster (1.24+)
- Helm 3.x
- kubectl configured
- cert-manager (for TLS)

## Quick Install

```bash
# Install from OCI registry (published by the release pipeline)
helm registry login ghcr.io

# Install with minimal config
# NOTE: Replace <org> with your GitHub org/user that hosts the chart.
helm install my-gateway oci://ghcr.io/satgate-io/charts/satgate-gateway \
  --version v1.0.0 \
  --namespace satgate \
  --create-namespace \
  --set global.tenantId=my-company \
  --set ingress.hosts[0].host=api.example.com \
  --set auth.adminToken=$(openssl rand -hex 32)
```

## Dashboard Deployment

The SatGate Dashboard requires explicit configuration:

### Environment Variables (Required)

```bash
# Dashboard MUST know the Gateway URL (no defaults shipped)
NEXT_PUBLIC_GATEWAY_URL=https://api.example.com
```

### Gateway CORS Configuration

If deploying the Dashboard on a different domain than the Gateway, configure CORS:

```yaml
# values-production.yaml
admin:
  corsAllowedOrigins:
    - "https://dashboard.example.com"

# Or use the dashboard helper:
dashboard:
  extraCorsAllowedOrigins:
    - "https://dashboard.example.com"
```

**Security Note:** Do NOT use wildcard `*` origins. Explicitly list your Dashboard domain(s).

## Production Installation

### Step 1: Create Namespace

```bash
kubectl create namespace satgate
```

### Step 2: Create Secrets

```bash
# Generate secure values
ADMIN_TOKEN=$(openssl rand -hex 32)
JWT_SECRET=$(openssl rand -hex 32)
AUDIT_KEY=$(openssl rand -base64 48)

# Create secret
kubectl create secret generic satgate-auth \
  --namespace satgate \
  --from-literal=ADMIN_TOKEN=$ADMIN_TOKEN \
  --from-literal=JWT_SECRET=$JWT_SECRET \
  --from-literal=AUDIT_HASH_CHAIN_KEY=$AUDIT_KEY
```

### Step 3: Create Values File

```yaml
# values-production.yaml

global:
  tenantId: "my-company"

image:
  repository: ghcr.io/satgate-io/satgate-gateway
  tag: "1.0.0"

dataPlane:
  replicaCount: 3
  resources:
    requests:
      cpu: 200m
      memory: 256Mi
    limits:
      cpu: 1000m
      memory: 512Mi
  autoscaling:
    enabled: true
    minReplicas: 3
    maxReplicas: 10

adminPlane:
  enabled: true
  networkPolicy:
    enabled: true

ingress:
  enabled: true
  className: nginx
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
  hosts:
    - host: api.example.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: satgate-tls
      hosts:
        - api.example.com

tenantIsolation:
  enabled: true
  required: false  # Enable after validation

auth:
  existingSecret: satgate-auth

postgresql:
  enabled: true
  auth:
    database: satgate
    username: satgate
  primary:
    persistence:
      size: 20Gi

redis:
  enabled: true
  auth:
    enabled: true
  master:
    persistence:
      size: 5Gi

metrics:
  enabled: true
  serviceMonitor:
    enabled: true
```

### Step 4: Install

```bash
helm install my-gateway satgate/satgate-gateway \
  --namespace satgate \
  --values values-production.yaml
```

### Step 5: Verify

```bash
# Check pods
kubectl get pods -n satgate

# Check health
kubectl port-forward svc/my-gateway 8080:8080 -n satgate &
curl http://localhost:8080/healthz

# Check preflight
kubectl port-forward svc/my-gateway-admin 9090:9090 -n satgate &
curl http://localhost:9090/api/v1/system/preflight \
  -H "X-Admin-Token: $ADMIN_TOKEN"
```

## Enable Strict Tenant Isolation

Follow the [Production Cutover Runbook](../operations/production-cutover.md) after validation.

## High Availability

### Multiple Replicas

```yaml
dataPlane:
  replicaCount: 3
  podDisruptionBudget:
    enabled: true
    minAvailable: 2
```

### Pod Anti-Affinity

```yaml
affinity:
  podAntiAffinity:
    preferredDuringSchedulingIgnoredDuringExecution:
      - weight: 100
        podAffinityTerm:
          labelSelector:
            matchLabels:
              app.kubernetes.io/name: satgate-gateway
          topologyKey: kubernetes.io/hostname
```

### External Database

For production, use managed PostgreSQL:

```yaml
postgresql:
  enabled: false
  external:
    enabled: true
    host: "my-db.rds.amazonaws.com"
    port: 5432
    database: satgate
    existingSecret: db-credentials
```

## Monitoring

### Prometheus ServiceMonitor

```yaml
metrics:
  enabled: true
  serviceMonitor:
    enabled: true
    interval: 30s
    labels:
      release: prometheus
```

### Grafana Dashboards

Import dashboards from `monitoring/grafana/dashboards/`:

- `gateway-overview.json` — Request metrics
- `protection-mode.json` — Token and governance
- `payments-mode.json` — L402 revenue

## Troubleshooting

### Pods Not Starting

```bash
# Check events
kubectl describe pod -l app.kubernetes.io/name=satgate-gateway -n satgate

# Check logs
kubectl logs -l app.kubernetes.io/name=satgate-gateway -n satgate
```

### Database Connection Failed

```bash
# Verify PostgreSQL is ready
kubectl get pods -l app.kubernetes.io/name=postgresql -n satgate

# Check connection string
kubectl exec -it deploy/my-gateway -n satgate -- env | grep DATABASE
```

### Ingress Not Working

```bash
# Check ingress status
kubectl describe ingress my-gateway -n satgate

# Verify TLS certificate
kubectl get certificate -n satgate
```

## Next Steps

- [Production Checklist](../operations/production-checklist.md)
- [Production Cutover](../operations/production-cutover.md)
- [Monitoring Setup](../operations/monitoring.md)


