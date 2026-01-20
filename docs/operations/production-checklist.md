# Production Checklist

Complete this checklist before going live.

## Security

### Authentication

- [ ] **Admin token** — Generated with `openssl rand -hex 32`
- [ ] **JWT secret** — Separate from admin token
- [ ] **Audit hash chain key** — Generated with `openssl rand -base64 48`
- [ ] **Secrets in Kubernetes** — Using `existingSecret`, not values

```bash
# Verify secrets are not in values
helm get values my-gateway -n satgate | grep -i token
# Should return empty or reference to secret name
```

### Network

- [ ] **Admin plane private** — Not exposed via ingress
- [ ] **NetworkPolicy enabled** — `adminPlane.networkPolicy.enabled: true`
- [ ] **TLS enabled** — cert-manager configured
- [ ] **Host allowlist** — `server.allowedHosts` configured

```bash
# Verify admin not exposed
kubectl get ingress -n satgate -o yaml | grep admin
# Should return nothing
```

### Tenant Isolation

- [ ] **Default tenant set** — `global.tenantId` configured
- [ ] **Validation mode** — `tenantIsolation.enabled: true`
- [ ] **Strict mode ready** — Preflight green before enabling

## High Availability

### Replicas

- [ ] **Multiple replicas** — `dataPlane.replicaCount >= 2`
- [ ] **PDB configured** — `podDisruptionBudget.enabled: true`
- [ ] **Pod anti-affinity** — Spread across nodes

```bash
# Verify replicas
kubectl get pods -n satgate -l app.kubernetes.io/name=satgate-gateway
# Should show multiple pods on different nodes
```

### Dependencies

- [ ] **PostgreSQL HA** — Managed database or replica set
- [ ] **Redis HA** — Sentinel or cluster mode
- [ ] **External secrets** — Not bundled with chart

## Monitoring

### Metrics

- [ ] **Prometheus scraping** — `metrics.serviceMonitor.enabled: true`
- [ ] **Grafana dashboards** — Imported from `monitoring/grafana/`
- [ ] **Alerting rules** — Configured for error rates

```bash
# Verify metrics
curl -s http://localhost:9090/metrics | grep satgate_requests_total
```

### Logging

- [ ] **JSON logs** — `--json-logs` flag enabled
- [ ] **Log aggregation** — Shipped to central logging
- [ ] **Audit logs** — Persisted to PostgreSQL

## Backup & Recovery

### Database

- [ ] **Automated backups** — Daily snapshots
- [ ] **Point-in-time recovery** — WAL archiving enabled
- [ ] **Tested restore** — Verified restore procedure

### Configuration

- [ ] **GitOps** — Values in version control
- [ ] **Helm release history** — `helm history my-gateway`
- [ ] **Rollback tested** — `helm rollback my-gateway`

## Performance

### Resources

- [ ] **Resource requests** — CPU and memory set
- [ ] **Resource limits** — Prevent runaway usage
- [ ] **HPA configured** — `autoscaling.enabled: true`

### Load Testing

- [ ] **Baseline established** — Normal traffic patterns
- [ ] **Stress tested** — 2x expected load
- [ ] **Latency SLOs** — p95 < 100ms

```bash
# Run load test
cd gateway/tests/load
k6 run k6_load_test.js
```

## Operational Readiness

### Runbooks

- [ ] **Production cutover** — [Runbook](production-cutover.md)
- [ ] **Incident response** — Escalation path defined
- [ ] **Rollback procedure** — Documented and tested

### Access

- [ ] **Break-glass access** — Admin token stored securely
- [ ] **RBAC configured** — Operator/viewer roles assigned
- [ ] **Audit trail** — All actions logged

## Final Verification

```bash
# Run preflight check
curl -s http://localhost:9090/api/v1/system/preflight \
  -H "X-Admin-Token: $ADMIN_TOKEN" | jq .

# All checks should be green
```

## Go Live

Once all items are checked:

1. Enable strict tenant isolation (see [Production Cutover](production-cutover.md))
2. Monitor for 24 hours
3. Declare production ready



