# SatGate Go SDK

**Give your services a Lightning wallet in a few lines of code.**

Automatic L402 payment handling — the "Stripe Moment" for Go microservices.

## Installation

```bash
go get github.com/SatGate-io/satgate-go
```

## Quick Start

```go
package main

import (
    "fmt"
    "io"
    
    satgate "github.com/SatGate-io/satgate-go"
)

func main() {
    // 1. Connect your wallet
    wallet := satgate.NewLNBitsWallet(
        "https://legend.lnbits.com",
        "your-admin-key",
    )

    // 2. Create client
    client := satgate.NewClient(wallet)

    // 3. That's it. 402 → Pay → Retry happens automatically.
    resp, err := client.Get("https://api.example.com/premium/data")
    if err != nil {
        panic(err)
    }
    defer resp.Body.Close()

    body, _ := io.ReadAll(resp.Body)
    fmt.Println(string(body))
}
```

## What Happens Under the Hood

```
1. GET /premium/data
   ↓
2. Server returns 402 + Lightning Invoice
   ↓
3. SDK automatically pays invoice
   ↓
4. SDK retries with L402 token
   ↓
5. You get the response ✓
```

## Wallet Options

### LNBits

```go
wallet := satgate.NewLNBitsWallet(
    "https://legend.lnbits.com",  // or your own instance
    "your-admin-key",
)
```

### Alby

```go
wallet := satgate.NewAlbyWallet("your-alby-access-token")
```

### LND (Direct Node Access)

```go
wallet := satgate.NewLNDWallet(
    "localhost:8080",           // LND REST host
    "0201036c6e6400...",        // hex-encoded admin macaroon
)
```

### Custom Wallet

Implement the `LightningWallet` interface:

```go
type LightningWallet interface {
    PayInvoice(invoice string) (preimage string, err error)
}

type MyWallet struct {
    // Your node connection
}

func (w *MyWallet) PayInvoice(invoice string) (string, error) {
    // Connect to your CLN, Eclair, etc.
    preimage, err := myNode.Pay(invoice)
    return preimage, err  // preimage as hex string
}
```

## Configuration Options

```go
client := satgate.NewClient(wallet,
    // Custom HTTP client (for TLS, timeouts, etc.)
    satgate.WithHTTPClient(&http.Client{
        Timeout: 60 * time.Second,
    }),
    
    // Token cache TTL (default: 5 minutes)
    satgate.WithCacheTTL(10 * time.Minute),
    
    // Verbose logging (default: true)
    satgate.WithVerbose(true),
    
    // Payment callback
    satgate.WithPaymentCallback(func(info satgate.PaymentInfo) {
        log.Printf("Paid for %s at %s", info.Endpoint, info.Timestamp)
    }),
)
```

## Making Requests

### GET

```go
resp, err := client.Get("https://api.example.com/premium")
```

### POST with JSON Body

```go
resp, err := client.Post("https://api.example.com/premium", map[string]interface{}{
    "query": "market analysis",
})
```

### Generic Request

```go
resp, err := client.Do("PUT", "https://api.example.com/resource", body)
```

## Token Caching

Tokens are cached by URL to avoid paying twice:

```go
// First call: pays invoice
client.Get("/premium")

// Second call within TTL: uses cached token (no payment)
client.Get("/premium")
```

## Payment Tracking

```go
// Track total spent
fmt.Printf("Total spent: %d sats\n", client.TotalPaidSat)

// Track individual payments
client := satgate.NewClient(wallet,
    satgate.WithPaymentCallback(func(info satgate.PaymentInfo) {
        // Log to your analytics
        analytics.TrackPayment(info.Endpoint, info.Preimage)
    }),
)
```

## Kubernetes / Microservices

Perfect for sidecar patterns or service mesh:

```go
// In your service initialization
func main() {
    wallet := satgate.NewLNDWallet(
        os.Getenv("LND_HOST"),
        os.Getenv("LND_MACAROON"),
    )
    
    satgateClient := satgate.NewClient(wallet)
    
    // Use for inter-service calls to paid APIs
    http.HandleFunc("/analyze", func(w http.ResponseWriter, r *http.Request) {
        // This call might hit a paid API
        resp, err := satgateClient.Get("https://premium-api.internal/analyze")
        if err != nil {
            http.Error(w, err.Error(), 500)
            return
        }
        defer resp.Body.Close()
        
        io.Copy(w, resp.Body)
    })
    
    http.ListenAndServe(":8080", nil)
}
```

## Error Handling

```go
resp, err := client.Get("/premium")
if err != nil {
    // Payment failed, network error, etc.
    log.Printf("Request failed: %v", err)
    return
}

if resp.StatusCode >= 400 {
    // API returned an error (after successful payment)
    body, _ := io.ReadAll(resp.Body)
    log.Printf("API error: %s", body)
}
```

## Thread Safety

The client is safe for concurrent use:

```go
client := satgate.NewClient(wallet)

var wg sync.WaitGroup
for i := 0; i < 10; i++ {
    wg.Add(1)
    go func() {
        defer wg.Done()
        client.Get("/premium")  // Safe for concurrent calls
    }()
}
wg.Wait()
```

## License

MIT

