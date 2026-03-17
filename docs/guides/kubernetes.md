# Kubernetes Deployment — Coming Soon

> 🚧 **Official Helm chart is under development.**

SatGate runs as a single Go binary with no external dependencies (except an optional Redis for distributed budgets). You can deploy it to Kubernetes today using a standard Deployment + Service manifest.

**Basic deployment:**

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: satgate
spec:
  replicas: 1
  selector:
    matchLabels:
      app: satgate
  template:
    metadata:
      labels:
        app: satgate
    spec:
      containers:
      - name: satgate
        image: ghcr.io/satgate-io/satgate:latest
        ports:
        - containerPort: 8080
        env:
        - name: ADMIN_TOKEN
          valueFrom:
            secretKeyRef:
              name: satgate-secrets
              key: admin-token
```

An official Helm chart with production defaults, HPA, and PodDisruptionBudget is coming.

**Want to contribute?** [Open an issue](https://github.com/SatGate-io/satgate/issues).
