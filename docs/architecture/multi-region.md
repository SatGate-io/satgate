# Multi-Region Deployment Architecture

Guidance for deploying SatGate Gateway across multiple geographic regions for high availability, disaster recovery, and low-latency access.

---

## Architecture Patterns

### Pattern 1: Active-Active (Recommended)

Each region operates independently with full read/write capability.

```
                              ┌─────────────────┐
                              │   Global DNS    │
                              │  (GeoDNS/GTM)   │
                              └────────┬────────┘
                                       │
           ┌───────────────────────────┼───────────────────────────┐
           │                           │                           │
           ▼                           ▼                           ▼
    ┌─────────────┐            ┌─────────────┐            ┌─────────────┐
    │  Region A   │            │  Region B   │            │  Region C   │
    │  (us-east)  │            │  (eu-west)  │            │  (ap-south) │
    │             │            │             │            │             │
    │ ┌─────────┐ │            │ ┌─────────┐ │            │ ┌─────────┐ │
    │ │ Gateway │ │            │ │ Gateway │ │            │ │ Gateway │ │
    │ │ Cluster │ │            │ │ Cluster │ │            │ │ Cluster │ │
    │ └────┬────┘ │            │ └────┬────┘ │            │ └────┬────┘ │
    │      │      │            │      │      │            │      │      │
    │ ┌────┴────┐ │            │ ┌────┴────┐ │            │ ┌────┴────┐ │
    │ │  Redis  │◄┼────────────┼►│  Redis  │◄┼────────────┼►│  Redis  │ │
    │ │ Cluster │ │  Sync      │ │ Cluster │ │  Sync      │ │ Cluster │ │
    │ └─────────┘ │            │ └─────────┘ │            │ └─────────┘ │
    │             │            │             │            │             │
    │ ┌─────────┐ │            │ ┌─────────┐ │            │ ┌─────────┐ │
    │ │Postgres │◄┼────────────┼►│Postgres │◄┼────────────┼►│Postgres │ │
    │ │ Primary │ │  Logical   │ │ Replica │ │  Logical   │ │ Replica │ │
    │ └─────────┘ │  Repl.     │ └─────────┘ │  Repl.     │ └─────────┘ │
    └─────────────┘            └─────────────┘            └─────────────┘
```

**Characteristics:**
- All regions handle read/write traffic
- GeoDNS routes users to nearest region
- Redis: Active-Active replication (Redis Enterprise) or CRDT-based sync
- PostgreSQL: Logical replication for audit logs, config sync
- Conflict resolution required for concurrent writes

**Best for:**
- Global user base requiring low latency
- High availability requirements (99.99%+)
- Compliance requiring data locality

---

### Pattern 2: Active-Passive (Simpler)

One primary region, others are hot standbys.

```
                              ┌─────────────────┐
                              │   Global DNS    │
                              │   (Failover)    │
                              └────────┬────────┘
                                       │
                    Primary ───────────┼───────────── Standby
                                       │
           ┌───────────────────────────┴───────────────────────────┐
           │                                                       │
           ▼                                                       ▼
    ┌─────────────┐                                        ┌─────────────┐
    │  Region A   │                                        │  Region B   │
    │  (Primary)  │          Async Replication             │  (Standby)  │
    │             │ ─────────────────────────────────────► │             │
    │ ┌─────────┐ │                                        │ ┌─────────┐ │
    │ │ Gateway │ │                                        │ │ Gateway │ │
    │ │ Active  │ │                                        │ │ Standby │ │
    │ └─────────┘ │                                        │ └─────────┘ │
    │             │                                        │             │
    │ ┌─────────┐ │                                        │ ┌─────────┐ │
    │ │  Redis  │ │                                        │ │  Redis  │ │
    │ │ Primary │ │ ───────────────────────────────────► │ │ Replica │ │
    │ └─────────┘ │                                        │ └─────────┘ │
    │             │                                        │             │
    │ ┌─────────┐ │                                        │ ┌─────────┐ │
    │ │Postgres │ │                                        │ │Postgres │ │
    │ │ Primary │ │ ───────────────────────────────────► │ │Streaming│ │
    │ └─────────┘ │                                        │ └─────────┘ │
    └─────────────┘                                        └─────────────┘
```

**Characteristics:**
- Simpler to operate
- No conflict resolution needed
- RTO: Minutes (DNS propagation)
- RPO: Depends on replication lag

**Best for:**
- DR requirements without global latency needs
- Smaller deployments
- Budget-conscious implementations

---

## Component-Specific Guidance

### PostgreSQL Multi-Region

#### Option A: Logical Replication (Recommended)

```sql
-- Primary region: Create publication
CREATE PUBLICATION satgate_pub FOR TABLE 
    audit_log, tenants, users, tokens, config;

-- Replica regions: Create subscription
CREATE SUBSCRIPTION satgate_sub
    CONNECTION 'host=primary.region-a.internal port=5432 dbname=satgate'
    PUBLICATION satgate_pub;
```

**Pros:** Selective replication, cross-version compatible
**Cons:** No DDL replication, conflict resolution needed

#### Option B: Streaming Replication (Active-Passive)

```yaml
# Primary postgresql.conf
wal_level = replica
max_wal_senders = 10
synchronous_standby_names = 'region_b'

# Standby recovery.conf
primary_conninfo = 'host=primary.region-a.internal'
```

#### Option C: CockroachDB / YugabyteDB (Distributed SQL)

For true multi-region active-active with automatic conflict resolution.

```sql
-- CockroachDB: Pin data to regions
ALTER DATABASE satgate SURVIVE REGION FAILURE;
ALTER TABLE audit_log SET LOCALITY REGIONAL BY ROW;
```

---

### Redis Multi-Region

#### Option A: Redis Enterprise Active-Active

```yaml
# redis-enterprise-cluster.yaml
apiVersion: redis.cloud/v1
kind: RedisEnterpriseActiveActiveDatabase
metadata:
  name: satgate-cache
spec:
  participatingClusters:
    - name: us-east-cluster
    - name: eu-west-cluster
  replication: true
  causalConsistency: true
```

#### Option B: Redis Cluster with Cross-Region Replicas

```bash
# Region A (Primary)
redis-cli --cluster create node1:6379 node2:6379 node3:6379

# Region B (Read replicas)
redis-cli --cluster add-node --cluster-slave region-b-node:6379 region-a-node:6379
```

#### Option C: Application-Level Sync

For simpler setups, sync critical data (ban list, rate limits) at application layer:

```go
// Sync ban list across regions every 5 seconds
func (g *Gateway) syncBanListToRegion(targetRegion string) {
    banList := g.governance.GetBanList()
    client := getRegionClient(targetRegion)
    client.PostBanListSync(banList)
}
```

---

### Gateway Deployment

#### Kubernetes with Federation

```yaml
# kubefed-satgate.yaml
apiVersion: types.kubefed.io/v1beta1
kind: FederatedDeployment
metadata:
  name: satgate-gateway
spec:
  template:
    spec:
      replicas: 3
      template:
        spec:
          containers:
            - name: gateway
              image: ghcr.io/satgate-io/satgate-gateway:latest
  placement:
    clusters:
      - name: us-east-cluster
      - name: eu-west-cluster
      - name: ap-south-cluster
  overrides:
    - clusterName: us-east-cluster
      clusterOverrides:
        - path: "/spec/replicas"
          value: 5  # Higher capacity in primary region
```

#### Helm Values per Region

```yaml
# values-us-east.yaml (Primary)
global:
  region: us-east
  isPrimary: true

postgresql:
  primary:
    enabled: true
  replica:
    enabled: false

redis:
  cluster:
    enabled: true
    nodes: 6
```

```yaml
# values-eu-west.yaml (Replica)
global:
  region: eu-west
  isPrimary: false

postgresql:
  primary:
    enabled: false
  replica:
    enabled: true
    replicaOf: postgres.us-east.internal

redis:
  cluster:
    enabled: true
    replicaOf: redis.us-east.internal
```

---

## Traffic Routing

### GeoDNS Configuration (AWS Route 53)

```hcl
# terraform/route53.tf
resource "aws_route53_record" "gateway_geo" {
  zone_id = aws_route53_zone.main.zone_id
  name    = "api.satgate.io"
  type    = "A"

  set_identifier = "us-east"
  geolocation_routing_policy {
    continent = "NA"
  }
  alias {
    name    = aws_lb.gateway_us_east.dns_name
    zone_id = aws_lb.gateway_us_east.zone_id
  }
}

resource "aws_route53_record" "gateway_geo_eu" {
  zone_id = aws_route53_zone.main.zone_id
  name    = "api.satgate.io"
  type    = "A"

  set_identifier = "eu-west"
  geolocation_routing_policy {
    continent = "EU"
  }
  alias {
    name    = aws_lb.gateway_eu_west.dns_name
    zone_id = aws_lb.gateway_eu_west.zone_id
  }
}
```

### Health Check Based Failover

```hcl
resource "aws_route53_health_check" "gateway_us_east" {
  fqdn              = "gateway.us-east.internal"
  port              = 443
  type              = "HTTPS"
  resource_path     = "/healthz"
  failure_threshold = "3"
  request_interval  = "30"
}

resource "aws_route53_record" "gateway_failover_primary" {
  zone_id         = aws_route53_zone.main.zone_id
  name            = "api.satgate.io"
  type            = "A"
  set_identifier  = "primary"
  
  failover_routing_policy {
    type = "PRIMARY"
  }
  
  health_check_id = aws_route53_health_check.gateway_us_east.id
  
  alias {
    name    = aws_lb.gateway_us_east.dns_name
    zone_id = aws_lb.gateway_us_east.zone_id
  }
}
```

---

## Data Consistency Model

### Eventual Consistency (Default)

For most data (tokens, rate limits), eventual consistency is acceptable:

| Data Type | Consistency | Sync Delay | Impact |
|-----------|-------------|------------|--------|
| Rate limits | Eventual | ~1s | Slight over/under limiting |
| Token usage | Eventual | ~5s | Metrics may lag |
| Ban list | Strong | ~100ms | Critical - use Redis Streams |
| Audit log | Eventual | ~30s | Acceptable for compliance |
| Config | Strong | Manual | Deploy through GitOps |

### Strong Consistency for Ban List

```go
// Use Redis Streams for real-time ban propagation
func (g *Gateway) publishBan(signature, reason string) {
    g.redisClient.XAdd(ctx, &redis.XAddArgs{
        Stream: "satgate:bans",
        Values: map[string]interface{}{
            "signature": signature,
            "reason":    reason,
            "timestamp": time.Now().Unix(),
        },
    })
}

// Each region subscribes to the stream
func (g *Gateway) subscribeBanStream() {
    for {
        msgs, _ := g.redisClient.XRead(ctx, &redis.XReadArgs{
            Streams: []string{"satgate:bans", "$"},
            Block:   0,
        }).Result()
        
        for _, msg := range msgs[0].Messages {
            g.governance.Ban(msg.Values["signature"], msg.Values["reason"], "sync")
        }
    }
}
```

---

## Failover Procedures

### Automatic Failover

1. Health check fails in primary region
2. DNS TTL expires (set low: 60s)
3. Traffic routes to healthy region
4. PostgreSQL standby promotes (if needed)
5. Redis replica becomes primary

### Manual Failover (Planned Maintenance)

```bash
# 1. Drain primary region
kubectl drain --ignore-daemonsets --delete-emptydir-data

# 2. Update DNS weight (if using weighted routing)
aws route53 change-resource-record-sets --hosted-zone-id Z123 \
  --change-batch '{"Changes":[{"Action":"UPSERT","ResourceRecordSet":{"Name":"api.satgate.io","Type":"A","SetIdentifier":"us-east","Weight":0,...}}]}'

# 3. Wait for drain (monitor active connections)
watch 'kubectl get pods -l app=satgate-gateway'

# 4. Perform maintenance

# 5. Restore traffic
aws route53 change-resource-record-sets --hosted-zone-id Z123 \
  --change-batch '{"Changes":[{"Action":"UPSERT","ResourceRecordSet":{"Name":"api.satgate.io","Type":"A","SetIdentifier":"us-east","Weight":100,...}}]}'
```

---

## Monitoring Multi-Region

### Key Metrics

| Metric | Alert Threshold | Description |
|--------|-----------------|-------------|
| `replication_lag_seconds` | > 30s | PostgreSQL/Redis replication delay |
| `region_health` | < 1 | Region health check failure |
| `cross_region_latency_ms` | > 200ms | Inter-region communication delay |
| `dns_resolution_time_ms` | > 500ms | GeoDNS resolution time |

### Grafana Dashboard

```json
{
  "title": "Multi-Region Overview",
  "panels": [
    {
      "title": "Requests by Region",
      "targets": [{
        "expr": "sum(rate(satgate_requests_total[5m])) by (region)"
      }]
    },
    {
      "title": "Replication Lag",
      "targets": [{
        "expr": "pg_replication_lag_seconds"
      }]
    }
  ]
}
```

---

## Cost Considerations

| Component | Single Region | Multi-Region (3) | Notes |
|-----------|---------------|------------------|-------|
| Compute | $500/mo | $1,500/mo | 3x gateway clusters |
| Database | $200/mo | $800/mo | Primary + 2 replicas + transfer |
| Redis | $100/mo | $400/mo | Enterprise for Active-Active |
| Traffic | $50/mo | $200/mo | Cross-region data transfer |
| DNS | $1/mo | $10/mo | Health checks per region |
| **Total** | **$851/mo** | **$2,910/mo** | ~3.4x cost |

**Cost optimization:**
- Use reserved instances (40% savings)
- Compress cross-region data
- Use regional data residency to minimize transfer

---

## Recommended Starting Point

For most enterprise deployments, start with:

1. **Two regions** (Primary + DR)
2. **Active-Passive** pattern
3. **PostgreSQL streaming replication**
4. **Redis with read replicas**
5. **Route 53 failover routing**

Graduate to Active-Active when:
- Latency requirements < 100ms globally
- 99.99%+ availability SLA
- Regulatory requirements for data locality



