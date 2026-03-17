# Backup & Restore

## What to Back Up

| Component | Location | Critical |
|-----------|----------|----------|
| Config file | `gateway.yaml` | ✅ Yes |
| Admin token | `ADMIN_TOKEN` env var | ✅ Yes |
| Capability root key | `CAPABILITY_ROOT_KEY` env var | ✅ Yes — losing this invalidates all tokens |
| L402 root key | `L402_ROOT_KEY` env var | ✅ Yes — losing this invalidates all L402 tokens |
| PostgreSQL data | Database backup | ✅ If using persistent storage |
| Redis data | Redis snapshot/AOF | ⚠️ If using distributed budgets |

## Key Recovery

**If you lose `CAPABILITY_ROOT_KEY`:** All existing macaroon tokens become invalid. You must re-mint and distribute new tokens.

**If you lose `L402_ROOT_KEY`:** All existing L402 payment proofs become invalid. Paid users will need to re-pay.

**Recommendation:** Store keys in a secrets manager (AWS Secrets Manager, HashiCorp Vault, etc.) with versioning enabled.

## Config Backup

```bash
# Simple backup
cp gateway.yaml gateway.yaml.backup.$(date +%Y%m%d)

# With key export (store securely!)
echo "ADMIN_TOKEN=$ADMIN_TOKEN" > .env.backup
echo "CAPABILITY_ROOT_KEY=$CAPABILITY_ROOT_KEY" >> .env.backup
echo "L402_ROOT_KEY=$L402_ROOT_KEY" >> .env.backup
chmod 600 .env.backup
```

## PostgreSQL Backup

```bash
pg_dump $DATABASE_URL > satgate-backup-$(date +%Y%m%d).sql
```

## Restore

```bash
./satgate --config gateway.yaml.backup
```
