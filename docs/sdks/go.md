# Go SDK

Official Go client for SatGate Gateway.

## Installation

```bash
go get github.com/satgate-io/satgate-go
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/satgate-io/satgate-go"
)

func main() {
    // Create client
    client := satgate.NewClient("https://api.example.com",
        satgate.WithAdminToken("your-admin-token"),
    )

    ctx := context.Background()

    // Mint a token
    token, err := client.Tokens.Mint(ctx, &satgate.MintRequest{
        Scope:     "api:read",
        ExpiresIn: 3600, // 1 hour
        Metadata: map[string]string{
            "user": "agent-1",
        },
    })
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Token: %s\n", token.Token)
    fmt.Printf("Expires: %s\n", token.ExpiresAt)
}
```

## Client Configuration

```go
// Basic configuration
client := satgate.NewClient("https://api.example.com",
    satgate.WithAdminToken("your-admin-token"),
)

// Full configuration
client := satgate.NewClient("https://api.example.com",
    satgate.WithAdminToken("your-admin-token"),
    satgate.WithTimeout(30 * time.Second),
    satgate.WithRetries(3),
    satgate.WithLogger(logger),
)

// With JWT authentication
client := satgate.NewClient("https://api.example.com",
    satgate.WithJWT("your-jwt-token"),
)
```

## Token Management

### Mint Token

```go
token, err := client.Tokens.Mint(ctx, &satgate.MintRequest{
    Scope:     "api:read,api:write",
    ExpiresIn: 86400, // 24 hours
    Metadata: map[string]string{
        "user":    "agent-1",
        "purpose": "data-pipeline",
    },
})
```

### List Tokens

```go
tokens, err := client.Tokens.List(ctx, &satgate.ListOptions{
    Limit:  100,
    Offset: 0,
})

for _, t := range tokens.Items {
    fmt.Printf("Token: %s, Scope: %s\n", t.Signature[:8], t.Scope)
}
```

### Get Token Details

```go
token, err := client.Tokens.Get(ctx, "token-signature")
```

### Revoke Token

```go
err := client.Tokens.Revoke(ctx, "token-signature")
```

### Delegate Token

```go
delegated, err := client.Tokens.Delegate(ctx, "parent-signature", &satgate.DelegateRequest{
    Caveats: []satgate.Caveat{
        {Type: "expires", Value: "1h"},
        {Type: "rate_limit", Value: "100/minute"},
        {Type: "scope", Value: "api:read"}, // Attenuate scope
    },
})
```

## Governance

### Ban Token

```go
err := client.Governance.Ban(ctx, &satgate.BanRequest{
    Signature: "token-to-ban",
    Reason:    "Compromised credentials",
})
```

### Unban Token

```go
err := client.Governance.Unban(ctx, "token-signature")
```

### Get Ban List

```go
banned, err := client.Governance.BanList(ctx)

for _, sig := range banned.Signatures {
    fmt.Printf("Banned: %s\n", sig)
}
```

### Get Token Lineage

```go
graph, err := client.Governance.Graph(ctx)

for _, node := range graph.Nodes {
    fmt.Printf("Token: %s, Parent: %s\n", node.Signature, node.Parent)
}
```

## Configuration

### Get Config

```go
config, err := client.Config.Get(ctx)

fmt.Printf("Routes: %d\n", len(config.Routes))
```

### Update Config

```go
err := client.Config.Update(ctx, &satgate.ConfigUpdate{
    Routes: []satgate.Route{
        {
            Name:     "new-api",
            Path:     "/v2/*",
            Upstream: "backend",
            Policy: satgate.Policy{
                Kind:  "capability",
                Scope: "api:v2",
            },
        },
    },
})
```

### Validate Config

```go
result, err := client.Config.Validate(ctx, configYAML)

if !result.Valid {
    for _, e := range result.Errors {
        fmt.Printf("Error at line %d: %s\n", e.Line, e.Message)
    }
}
```

## Statistics

### Get Gateway Stats

```go
stats, err := client.Stats.Get(ctx)

fmt.Printf("Total Requests: %d\n", stats.TotalRequests)
fmt.Printf("Active Tokens: %d\n", stats.ActiveTokens)
```

### Get Route Stats

```go
routes, err := client.Stats.Routes(ctx)

for name, stats := range routes {
    fmt.Printf("%s: %d requests, p99: %dms\n",
        name, stats.Requests, stats.LatencyP99)
}
```

## Making Protected Requests

Use the client to make requests to protected upstream APIs:

```go
// Using a capability token
resp, err := client.Request(ctx, "GET", "/api/users",
    satgate.WithToken(token.Token),
)
if err != nil {
    log.Fatal(err)
}
defer resp.Body.Close()

var users []User
json.NewDecoder(resp.Body).Decode(&users)
```

## WebSocket Telemetry

Stream real-time events:

```go
events, err := client.Telemetry.Subscribe(ctx)
if err != nil {
    log.Fatal(err)
}

for event := range events {
    switch e := event.(type) {
    case *satgate.TokenMintEvent:
        fmt.Printf("Token minted: %s\n", e.Signature)
    case *satgate.TokenBanEvent:
        fmt.Printf("Token banned: %s\n", e.Signature)
    case *satgate.RequestEvent:
        fmt.Printf("Request: %s %s (%d)\n", e.Method, e.Path, e.Status)
    }
}
```

## Error Handling

```go
token, err := client.Tokens.Mint(ctx, req)
if err != nil {
    var apiErr *satgate.APIError
    if errors.As(err, &apiErr) {
        switch apiErr.Code {
        case "RATE_LIMITED":
            // Wait and retry
            time.Sleep(time.Duration(apiErr.RetryAfter) * time.Second)
        case "UNAUTHORIZED":
            // Refresh credentials
        default:
            log.Printf("API error: %s", apiErr.Message)
        }
    }
    return err
}
```

## Context and Timeouts

```go
// With timeout
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

token, err := client.Tokens.Mint(ctx, req)

// With cancellation
ctx, cancel := context.WithCancel(context.Background())
go func() {
    // Cancel after signal
    <-sigChan
    cancel()
}()
```

## Testing

Use the mock client for testing:

```go
import "github.com/satgate-io/satgate-go/mock"

func TestMyService(t *testing.T) {
    mockClient := mock.NewClient()
    mockClient.Tokens.OnMint(func(req *satgate.MintRequest) (*satgate.Token, error) {
        return &satgate.Token{
            Token:     "mock-token",
            Signature: "mock-sig",
        }, nil
    })

    service := NewMyService(mockClient)
    // Test your service...
}
```



