# Backup and Restore Runbook

Procedures for backing up and restoring SatGate Gateway data.

---

## Data Inventory

| Component | Data | Criticality | Loss Impact |
|-----------|------|-------------|-------------|
| PostgreSQL | Tokens, audit log, tenants, config | **Critical** | Service disruption, compliance failure |
| Redis | Ban list, rate limits, sessions | **High** | Temporary governance gap |
| Gateway Config | `gateway.yaml` | **Medium** | Manual reconfiguration needed |
| Secrets | Admin token, JWT secret, audit key | **Critical** | Complete lockout |

---

## RPO/RTO Targets

| Metric | Target | Method |
|--------|--------|--------|
| **RPO** (Recovery Point Objective) | ≤ 1 hour | WAL archiving + hourly snapshots |
| **RTO** (Recovery Time Objective) | ≤ 4 hours | Restore + verify + DNS failover |

---

## PostgreSQL Backup

### Automated Daily Backup

**Kubernetes CronJob:**

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: satgate-pg-backup
  namespace: satgate
spec:
  schedule: "0 2 * * *"  # Daily at 2 AM UTC
  jobTemplate:
    spec:
      template:
        spec:
          containers:
            - name: backup
              image: postgres:15-alpine
              command:
                - /bin/sh
                - -c
                - |
                  TIMESTAMP=$(date +%Y%m%d_%H%M%S)
                  pg_dump $DATABASE_URL | gzip > /backups/satgate_$TIMESTAMP.sql.gz
                  # Upload to S3 (if AWS CLI available)
                  # aws s3 cp /backups/satgate_$TIMESTAMP.sql.gz s3://backups/satgate/
                  # Cleanup old local backups (keep 7 days)
                  find /backups -name "satgate_*.sql.gz" -mtime +7 -delete
              env:
                - name: DATABASE_URL
                  valueFrom:
                    secretKeyRef:
                      name: satgate-database
                      key: DATABASE_URL
              volumeMounts:
                - name: backups
                  mountPath: /backups
          restartPolicy: OnFailure
          volumes:
            - name: backups
              persistentVolumeClaim:
                claimName: satgate-backups
```

### Manual Backup

```bash
# Get database URL
export DATABASE_URL=$(kubectl get secret satgate-database -n satgate \
  -o jsonpath='{.data.DATABASE_URL}' | base64 -d)

# Create backup
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
pg_dump $DATABASE_URL | gzip > satgate_backup_$TIMESTAMP.sql.gz

# Upload to S3 (optional)
aws s3 cp satgate_backup_$TIMESTAMP.sql.gz s3://your-bucket/satgate/backups/
```

### WAL Archiving (Point-in-Time Recovery)

For production, enable WAL archiving in PostgreSQL:

```sql
-- postgresql.conf
wal_level = replica
archive_mode = on
archive_command = 'aws s3 cp %p s3://your-bucket/satgate/wal/%f'
```

---

## PostgreSQL Restore

### From Backup File

```bash
# 1. Stop the gateway to prevent writes
kubectl scale deployment satgate-gateway -n satgate --replicas=0

# 2. Get database credentials
export DATABASE_URL=$(kubectl get secret satgate-database -n satgate \
  -o jsonpath='{.data.DATABASE_URL}' | base64 -d)

# 3. Drop and recreate database (DESTRUCTIVE!)
psql $DATABASE_URL -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"

# 4. Restore from backup
gunzip -c satgate_backup_YYYYMMDD_HHMMSS.sql.gz | psql $DATABASE_URL

# 5. Verify restoration
psql $DATABASE_URL -c "SELECT COUNT(*) FROM audit_log;"
psql $DATABASE_URL -c "SELECT COUNT(*) FROM tenants;"

# 6. Restart gateway
kubectl scale deployment satgate-gateway -n satgate --replicas=3

# 7. Verify health
kubectl port-forward svc/satgate-gateway 8080:8080 -n satgate &
curl http://localhost:8080/healthz
```

### Point-in-Time Recovery

```bash
# 1. Restore base backup
pg_restore --dbname=$DATABASE_URL base_backup.tar

# 2. Apply WAL files up to target time
# (Requires recovery.conf or recovery.signal in PG 12+)
```

---

## Redis Backup

### Manual Backup

```bash
# Connect to Redis pod
kubectl exec -it $(kubectl get pod -l app=redis -n satgate -o jsonpath='{.items[0].metadata.name}') \
  -n satgate -- redis-cli BGSAVE

# Copy RDB file
kubectl cp satgate/$(kubectl get pod -l app=redis -n satgate -o jsonpath='{.items[0].metadata.name}'):/data/dump.rdb \
  ./redis_backup_$(date +%Y%m%d).rdb
```

### Automated Backup (CronJob)

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: satgate-redis-backup
  namespace: satgate
spec:
  schedule: "30 2 * * *"  # Daily at 2:30 AM UTC
  jobTemplate:
    spec:
      template:
        spec:
          containers:
            - name: backup
              image: redis:7-alpine
              command:
                - /bin/sh
                - -c
                - |
                  redis-cli -h redis -a $REDIS_PASSWORD BGSAVE
                  sleep 5
                  cp /data/dump.rdb /backups/redis_$(date +%Y%m%d).rdb
              env:
                - name: REDIS_PASSWORD
                  valueFrom:
                    secretKeyRef:
                      name: satgate-redis
                      key: password
              volumeMounts:
                - name: redis-data
                  mountPath: /data
                - name: backups
                  mountPath: /backups
          restartPolicy: OnFailure
```

---

## Redis Restore

```bash
# 1. Stop gateway
kubectl scale deployment satgate-gateway -n satgate --replicas=0

# 2. Copy backup to Redis pod
kubectl cp redis_backup_YYYYMMDD.rdb \
  satgate/$(kubectl get pod -l app=redis -n satgate -o jsonpath='{.items[0].metadata.name}'):/data/dump.rdb

# 3. Restart Redis
kubectl delete pod -l app=redis -n satgate

# 4. Wait for Redis to load data
kubectl exec -it $(kubectl get pod -l app=redis -n satgate -o jsonpath='{.items[0].metadata.name}') \
  -n satgate -- redis-cli PING

# 5. Restart gateway
kubectl scale deployment satgate-gateway -n satgate --replicas=3
```

---

## Secrets Backup

**Critical:** Losing secrets means complete lockout.

### Export to Encrypted File

```bash
# Export secrets (store securely!)
kubectl get secret satgate-auth -n satgate -o yaml > satgate-secrets.yaml

# Encrypt with GPG (recommended)
gpg --encrypt --recipient security@yourcompany.com satgate-secrets.yaml
rm satgate-secrets.yaml

# Store encrypted file in secure location (Vault, encrypted S3, etc.)
```

### Restore Secrets

```bash
# Decrypt
gpg --decrypt satgate-secrets.yaml.gpg > satgate-secrets.yaml

# Apply (careful - this overwrites existing secrets)
kubectl apply -f satgate-secrets.yaml

# Clean up
rm satgate-secrets.yaml
```

---

## Gateway Config Backup

Configuration should be in version control (GitOps), but as a fallback:

```bash
# Export current config
kubectl get configmap satgate-gateway-config -n satgate -o yaml > satgate-config-backup.yaml

# Export Helm values
helm get values satgate-gateway -n satgate -o yaml > satgate-values-backup.yaml
```

---

## Disaster Recovery Procedure

### Complete Cluster Loss

1. **Provision new cluster**
   ```bash
   # Using Terraform
   cd deploy/terraform/aws  # or gcp
   terraform apply
   ```

2. **Restore secrets**
   ```bash
   gpg --decrypt satgate-secrets.yaml.gpg | kubectl apply -f -
   ```

3. **Install Helm chart**
   ```bash
   helm install satgate-gateway oci://ghcr.io/satgate-io/charts/satgate-gateway \
     --namespace satgate \
     --values satgate-values-backup.yaml
   ```

4. **Restore PostgreSQL**
   ```bash
   gunzip -c satgate_backup_latest.sql.gz | psql $DATABASE_URL
   ```

5. **Restore Redis (optional)**
   - Redis data can be rebuilt from PostgreSQL
   - Only needed if you want to preserve in-flight sessions

6. **Verify**
   ```bash
   curl https://api.example.com/healthz
   curl https://api.example.com/api/v1/system/preflight \
     -H "X-Admin-Token: $ADMIN_TOKEN"
   ```

7. **Update DNS**
   - Point domain to new load balancer
   - Wait for propagation (TTL)

---

## Verification Checklist

After any restore:

- [ ] Health endpoint returns `{"status":"ok"}`
- [ ] Preflight checks all green
- [ ] Can mint new tokens
- [ ] Existing tokens still work (if restored)
- [ ] Ban list is intact
- [ ] Audit log is complete
- [ ] Tenants are configured
- [ ] No data gaps in audit timeline

---

## Backup Retention Policy

| Data | Hot (Immediate) | Warm (Archive) | Cold (Long-term) |
|------|-----------------|----------------|------------------|
| PostgreSQL | 7 days | 90 days | 7 years (compliance) |
| Redis | 3 days | 30 days | N/A |
| Secrets | Current only | N/A | Rotate, don't archive |

---

## Monitoring Backup Health

Add Prometheus alerts:

```yaml
groups:
  - name: backup-alerts
    rules:
      - alert: BackupFailed
        expr: kube_job_status_failed{job_name=~"satgate-.*-backup.*"} > 0
        for: 1h
        labels:
          severity: critical
        annotations:
          summary: "SatGate backup job failed"

      - alert: BackupMissing
        expr: time() - max(kube_job_status_completion_time{job_name=~"satgate-pg-backup.*"}) > 86400 * 2
        for: 1h
        labels:
          severity: warning
        annotations:
          summary: "No successful backup in 2 days"
```



