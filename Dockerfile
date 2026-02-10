# SatGate OSS Gateway
# ====================
# Multi-stage build for minimal production image

# Build stage
FROM golang:1.24-alpine AS builder

ARG VERSION=dev
ARG COMMIT=unknown

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Copy go mod files first for caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X main.Version=${VERSION} -X main.Commit=${COMMIT}" \
    -o satgate ./cmd/satgate

# Runtime stage
FROM alpine:3.21

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN addgroup -S satgate && adduser -S satgate -G satgate

WORKDIR /app

# Copy binary
COPY --from=builder /build/satgate /usr/local/bin/satgate

# Copy default config
COPY examples/gateway.yaml /app/gateway.yaml

# Set ownership
RUN chown -R satgate:satgate /app

USER satgate

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -qO- http://localhost:8080/health || exit 1

ENTRYPOINT ["satgate"]
CMD ["--config", "/app/gateway.yaml"]
