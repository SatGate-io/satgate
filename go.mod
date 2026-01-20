module github.com/satgate-io/satgate

go 1.24.0

toolchain go1.24.5

require (
	github.com/redis/go-redis/v9 v9.17.2
	github.com/rs/zerolog v1.34.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	golang.org/x/sys v0.40.0 // indirect
)

// Note: Run `go mod tidy` after extraction to update dependencies
