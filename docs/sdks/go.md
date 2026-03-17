# Go SDK

SatGate is written in Go. Instead of a separate SDK, import the packages directly:

```go
import (
    "github.com/SatGate-io/satgate/pkg/config"
    "github.com/SatGate-io/satgate/pkg/macaroon"
    "github.com/SatGate-io/satgate/pkg/mcp"
)
```

## Key Packages

| Package | Description |
|---------|-------------|
| `pkg/config` | Configuration loading, validation, route matching |
| `pkg/proxy` | HTTP reverse proxy with policy enforcement |
| `pkg/macaroon` | Macaroon token creation, verification, delegation |
| `pkg/mcp` | MCP JSON-RPC parser for tool-level attribution |
| `pkg/mcpserver` | MCP proxy server (stdio transport) |
| `pkg/lightning` | Lightning payment backends (NWC, LND, Phoenixd, Alby, LNbits) |
| `pkg/billing` | Billing engine (chargeback, fiat402, L402) |

## Example: Mint and Verify a Token

```go
package main

import (
    "fmt"
    "net/http"
)

func main() {
    // Mint via HTTP API
    req, _ := http.NewRequest("POST", "http://localhost:8080/api/capability/mint", nil)
    req.Header.Set("X-Admin-Token", "your-admin-token")
    req.Header.Set("Content-Type", "application/json")
    
    // ... standard Go HTTP client usage
}
```

For most Go integrations, use the HTTP API directly. The internal packages are useful if you're embedding SatGate in your own Go application.
