# SatGate Cloud Demo
# Build Aperture from source for reliability

# Stage 1: Build Aperture from source
FROM golang:1.22-alpine AS aperture-builder

RUN apk add --no-cache git make

WORKDIR /build
RUN git clone --depth 1 --branch v0.3.6-beta https://github.com/lightninglabs/aperture.git
WORKDIR /build/aperture
# CGO_ENABLED=0 ensures static linking (no musl/glibc dependency)
RUN CGO_ENABLED=0 go build -o /aperture ./cmd/aperture

# Stage 2: Final runtime image
FROM node:20-slim

# Ensure system CA bundle is present for TLS connections (LNC mailbox, etc.)
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Copy Aperture binary from builder
COPY --from=aperture-builder /aperture /usr/local/bin/aperture
RUN chmod +x /usr/local/bin/aperture

# Create app directory
WORKDIR /app

# Copy and install backend deps (pinned in proxy/package.json)
WORKDIR /app/backend
COPY proxy/package.json /app/backend/package.json
RUN npm install --omit=dev --no-audit --no-fund

# Copy backend source
COPY proxy/server.js /app/backend/server.js

# Copy dashboard UI
COPY proxy/public /app/backend/public

# Copy configs
WORKDIR /app
COPY deploy/aperture.cloud.yaml /app/aperture.yaml
COPY deploy/start.sh /app/start.sh
RUN chmod +x /app/start.sh

# Create data directories
RUN mkdir -p /app/data /root/.aperture

# Environment variables
ENV LNC_PASSPHRASE=""
ENV LNC_MAILBOX="mailbox.terminal.lightning.today:443"
ENV LNC_NETWORK="mainnet"
ENV BACKEND_PORT=8083
ENV PORT=8080
ENV NODE_ENV=production

EXPOSE 8080

CMD ["/app/start.sh"]
