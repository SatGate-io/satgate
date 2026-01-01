# Repository Strategy

SatGate follows an **OSS core** model: this repo contains the open-source proxy, SDKs, and documentation.

```
satgate/
├── proxy/              # L402 gateway (Node.js + Aperture)
├── sdk/
│   ├── python/         # pip install satgate
│   ├── js/             # npm install @satgate/sdk
│   └── go/             # go get github.com/satgate/satgate/sdk/go
├── docker/             # One-click deployment
├── deploy/             # Cloud deployment configs
├── docs/               # Architecture, security, guides
├── examples/           # Demo scripts
├── cli/                # Token inspection tools
└── satgate-landing/    # satgate.io marketing site
```

**License:** MIT

## Private / Internal

Some components and materials are intentionally not part of this OSS repository.

## Contributing

- **OSS gateway/SDKs:** [github.com/satgate/satgate](https://github.com/satgate/satgate)
