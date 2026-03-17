# GitOps Integration with ArgoCD and Flux

This guide covers integrating SatGate signed configurations with GitOps tools like ArgoCD and Flux.

## Overview

SatGate supports cryptographically signed configurations for GitOps workflows. This ensures:
- **Authenticity**: Only authorized signers can deploy config changes
- **Integrity**: Configs cannot be tampered with in transit
- **Auditability**: Every change has author attribution and timestamp

## Prerequisites

1. SatGate Gateway with GitOps enabled
2. ArgoCD or Flux installed in your cluster
3. `satgate-sign` CLI tool for signing configs

## Signing Key Setup

### Generate Keys

```bash
# Generate ECDSA key pair (recommended)
satgate-sign generate-key --algorithm ecdsa --output keys/platform-team

# Output:
# ✓ Generated ecdsa key pair
#   Private key: keys/platform-team.key (keep secure!)
#   Public key:  keys/platform-team.pub (commit to repo)
```

### Store Keys Securely

| Key Type | Storage Location | Access |
|----------|------------------|--------|
| Private key (`.key`) | HashiCorp Vault, AWS Secrets Manager, or CI/CD secrets | CI/CD pipeline only |
| Public key (`.pub`) | Git repository + Gateway ConfigMap | Read-only |

### Deploy Public Keys to Gateway

```yaml
# k8s/configmap-trusted-keys.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: satgate-trusted-keys
  namespace: satgate
data:
  platform-team.pub: |
    -----BEGIN PUBLIC KEY-----
    MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE...
    -----END PUBLIC KEY-----
  security-team.pub: |
    -----BEGIN PUBLIC KEY-----
    MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE...
    -----END PUBLIC KEY-----
```

## ArgoCD Integration

### 1. Repository Structure

```
satgate-config/
├── base/
│   ├── kustomization.yaml
│   └── gateway-config.yaml
├── overlays/
│   ├── staging/
│   │   ├── kustomization.yaml
│   │   └── gateway-config.signed.yaml  # Signed config
│   └── production/
│       ├── kustomization.yaml
│       └── gateway-config.signed.yaml  # Signed config
└── keys/
    └── platform-team.pub  # Trusted public key
```

### 2. ArgoCD Application

```yaml
# argocd/satgate-app.yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: satgate-gateway
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://github.com/your-org/satgate-config.git
    targetRevision: HEAD
    path: overlays/production
  destination:
    server: https://kubernetes.default.svc
    namespace: satgate
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
```

### 3. Pre-Sync Hook for Verification

```yaml
# argocd/verify-hook.yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: verify-satgate-config
  annotations:
    argocd.argoproj.io/hook: PreSync
    argocd.argoproj.io/hook-delete-policy: HookSucceeded
spec:
  template:
    spec:
      containers:
        - name: verify
          image: ghcr.io/satgate-io/satgate-gateway:latest
          command:
            - satgate-sign
            - verify
            - --trusted-keys=/keys
            - --require-signature=true
            - /config/gateway-config.signed.yaml
          volumeMounts:
            - name: config
              mountPath: /config
            - name: keys
              mountPath: /keys
      volumes:
        - name: config
          configMap:
            name: satgate-config
        - name: keys
          configMap:
            name: satgate-trusted-keys
      restartPolicy: Never
```

### 4. CI Pipeline (GitHub Actions)

```yaml
# .github/workflows/sign-and-deploy.yaml
name: Sign and Deploy SatGate Config

on:
  push:
    branches: [main]
    paths:
      - 'overlays/**/*.yaml'
      - '!overlays/**/*.signed.yaml'

jobs:
  sign:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Install satgate-sign
        run: |
          curl -sLO https://github.com/satgate-io/satgate-gateway/releases/latest/download/satgate-sign-linux-amd64
          chmod +x satgate-sign-linux-amd64
          sudo mv satgate-sign-linux-amd64 /usr/local/bin/satgate-sign

      - name: Sign staging config
        env:
          SIGNING_KEY: ${{ secrets.SATGATE_SIGNING_KEY }}
        run: |
          echo "$SIGNING_KEY" > /tmp/signing.key
          satgate-sign sign \
            --key /tmp/signing.key \
            --key-id platform-team \
            --author "${{ github.actor }}@${{ github.repository }}" \
            --message "${{ github.event.head_commit.message }}" \
            --input overlays/staging/gateway-config.yaml \
            --output overlays/staging/gateway-config.signed.yaml
          rm /tmp/signing.key

      - name: Commit signed config
        run: |
          git config user.name "GitHub Actions"
          git config user.email "actions@github.com"
          git add overlays/**/*.signed.yaml
          git commit -m "chore: sign configs for ${{ github.sha }}"
          git push

      - name: Trigger ArgoCD Sync
        run: |
          argocd app sync satgate-gateway --prune
```

## Flux Integration

### 1. Flux Kustomization

```yaml
# flux/satgate-kustomization.yaml
apiVersion: kustomize.toolkit.fluxcd.io/v1
kind: Kustomization
metadata:
  name: satgate-gateway
  namespace: flux-system
spec:
  interval: 5m
  path: ./overlays/production
  prune: true
  sourceRef:
    kind: GitRepository
    name: satgate-config
  healthChecks:
    - apiVersion: apps/v1
      kind: Deployment
      name: satgate-gateway
      namespace: satgate
  postBuild:
    substitute:
      CLUSTER_NAME: production
```

### 2. Flux Pre-Apply Verification

```yaml
# flux/verify-receiver.yaml
apiVersion: notification.toolkit.fluxcd.io/v1beta2
kind: Receiver
metadata:
  name: satgate-verify
  namespace: flux-system
spec:
  type: generic
  events:
    - "Kustomization/satgate-gateway.flux-system"
  resources:
    - kind: Job
      name: verify-satgate-config
      namespace: satgate
  secretRef:
    name: webhook-token
```

### 3. Verification Job

```yaml
# flux/verify-job.yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: verify-satgate-config
  namespace: satgate
spec:
  template:
    spec:
      containers:
        - name: verify
          image: ghcr.io/satgate-io/satgate-gateway:latest
          command: ["satgate-sign", "verify", "--trusted-keys=/keys", "--require-signature=true", "/config/gateway-config.signed.yaml"]
          volumeMounts:
            - name: config
              mountPath: /config
            - name: keys
              mountPath: /keys
      volumes:
        - name: config
          configMap:
            name: satgate-config
        - name: keys
          configMap:
            name: satgate-trusted-keys
      restartPolicy: Never
```

## Gateway Configuration

### Enable GitOps Verification

```yaml
# gateway.yaml
gitops:
  enabled: true
  trustedKeysDir: /etc/satgate/trusted-keys
  requireSignature: true  # Enterprise: reject unsigned configs
  watchPaths:
    - /etc/satgate/config.yaml
  pollInterval: 30s  # Fallback if fsnotify unavailable
```

### Kubernetes Deployment

```yaml
# k8s/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: satgate-gateway
spec:
  template:
    spec:
      containers:
        - name: gateway
          image: ghcr.io/satgate-io/satgate-gateway:latest
          env:
            - name: GITOPS_ENABLED
              value: "true"
            - name: GITOPS_REQUIRE_SIGNATURE
              value: "true"
          volumeMounts:
            - name: config
              mountPath: /etc/satgate
              readOnly: true
            - name: trusted-keys
              mountPath: /etc/satgate/trusted-keys
              readOnly: true
      volumes:
        - name: config
          configMap:
            name: satgate-config
        - name: trusted-keys
          configMap:
            name: satgate-trusted-keys
```

## Audit Trail

All config changes are logged with:
- Signer key ID
- Author email
- Timestamp
- Change message

```json
{
  "level": "info",
  "msg": "Loaded signed config",
  "path": "/etc/satgate/config.yaml",
  "keyId": "platform-team",
  "author": "alice@example.com",
  "timestamp": "2026-01-10T12:00:00Z",
  "message": "Enable rate limiting for /api/v1"
}
```

## Multi-Environment Strategy

| Environment | Signature Required | Trusted Signers |
|-------------|-------------------|-----------------|
| Development | No (optional) | Any developer key |
| Staging | Yes | Platform team, Security team |
| Production | Yes (Enterprise) | Platform team only |

## Troubleshooting

### Config Rejected - Unknown Key

```
Error: unknown signing key: developer-123
```

**Solution**: Add the public key to the trusted-keys ConfigMap.

### Config Rejected - Invalid Signature

```
Error: signature verification failed
```

**Solution**: Re-sign the config with a valid private key.

### Config Not Reloading

Check the GitOps controller logs:

```bash
kubectl logs -n satgate deployment/satgate-gateway | grep gitops
```

Ensure `watchPaths` includes your config file path.

## Best Practices

1. **Separate keys per team**: Use different keys for platform, security, and development teams
2. **Rotate keys regularly**: Generate new keys quarterly, update trusted-keys ConfigMap
3. **Sign in CI/CD only**: Never commit private keys to repositories
4. **Use Hardware Security Modules**: For production, consider HSM-backed signing via KMS
5. **Require signatures in production**: Set `requireSignature: true` for Enterprise deployments
