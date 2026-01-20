# Contributing to SatGate

Thank you for your interest in contributing to SatGate! This document provides guidelines and instructions for contributing.

## Code of Conduct

By participating in this project, you agree to abide by our Code of Conduct. Please be respectful and constructive in all interactions.

## How to Contribute

### Reporting Issues

- **Bug Reports**: Use the [bug report template](.github/ISSUE_TEMPLATE/bug_report.md)
- **Feature Requests**: Use the [feature request template](.github/ISSUE_TEMPLATE/feature_request.md)
- **Security Issues**: Email security@satgate.io (do not open public issues)

### Pull Requests

1. **Fork the repository** and create your branch from `main`
2. **Write tests** for any new functionality
3. **Follow the code style** (run `go fmt` and `go vet`)
4. **Update documentation** if needed
5. **Submit a PR** with a clear description

### Development Setup

```bash
# Clone your fork
git clone https://github.com/YOUR_USERNAME/satgate.git
cd satgate

# Add upstream remote
git remote add upstream https://github.com/satgate-io/satgate.git

# Install dependencies
go mod download

# Run tests
go test ./...

# Run linter
go vet ./...
golangci-lint run

# Build
go build -o satgate ./cmd/satgate
```

### Running Locally

```bash
# Start with example config
./satgate --config examples/gateway.yaml

# Test health endpoint
curl http://localhost:8080/health
```

## Code Style

### Go

- Follow [Effective Go](https://go.dev/doc/effective_go) guidelines
- Use `go fmt` for formatting
- Use `go vet` and `golangci-lint` for linting
- Write tests for all new code
- Keep functions focused and small

### Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add new capability caveat type
fix: resolve macaroon verification edge case
docs: update quickstart guide
test: add integration tests for L402
refactor: simplify proxy middleware
```

### Pull Request Guidelines

1. **Title**: Use conventional commit format
2. **Description**: Explain what and why, not just how
3. **Tests**: Include tests for new functionality
4. **Docs**: Update relevant documentation
5. **Size**: Keep PRs focused and reviewable (<500 lines ideal)

## Architecture

### Package Structure

```
satgate/
├── cmd/satgate/        # Main entrypoint
├── internal/
│   ├── proxy/          # Core HTTP proxy
│   ├── config/         # Configuration loading
│   ├── macaroon/       # Capability token handling
│   ├── governance/     # Ban lists, revocation
│   └── lightning/      # L402 payment providers
├── sdk/                # Client SDKs
├── docs/               # Documentation
└── examples/           # Example configurations
```

### Key Interfaces

```go
// Policy enforcement
type PolicyHandler interface {
    Handle(w http.ResponseWriter, r *http.Request, route *Route)
}

// Lightning provider
type LightningProvider interface {
    CreateInvoice(ctx context.Context, sats int64, memo string) (*Invoice, error)
    VerifyPayment(ctx context.Context, preimage string) (bool, error)
}
```

## Testing

### Running Tests

```bash
# All tests
go test ./...

# With coverage
go test -cover ./...

# Specific package
go test ./internal/proxy/...

# Integration tests (requires Docker)
go test -tags=integration ./...
```

### Writing Tests

```go
func TestMacaroonVerification(t *testing.T) {
    // Arrange
    svc := macaroon.New(testRootKey)
    mac, _ := svc.Mint(macaroon.MintOptions{
        Scope: "api:read",
    })

    // Act
    verified, err := svc.Verify(mac)

    // Assert
    require.NoError(t, err)
    assert.True(t, verified.HasScope("api:read"))
}
```

## Documentation

- Update README.md for user-facing changes
- Update docs/ for detailed documentation
- Add godoc comments for exported functions
- Include examples in documentation

## Release Process

Releases are managed by maintainers:

1. Update CHANGELOG.md
2. Tag release: `git tag v1.2.3`
3. Push tag: `git push origin v1.2.3`
4. GitHub Actions builds and publishes

## Getting Help

- **Questions**: Open a [Discussion](https://github.com/satgate-io/satgate/discussions)
- **Bugs**: Open an [Issue](https://github.com/satgate-io/satgate/issues)
- **Chat**: Join our [Discord](https://discord.gg/satgate)

## Recognition

Contributors are recognized in:
- CONTRIBUTORS.md
- Release notes
- Project README

Thank you for contributing to SatGate! 🎉
