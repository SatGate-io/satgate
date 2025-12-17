# SatGate Cloud Demo
# Build Aperture from source for reliability

# Stage 1: Build Aperture from source
FROM golang:1.22-alpine AS aperture-builder

RUN apk add --no-cache git make

WORKDIR /build
RUN git clone --depth 1 --branch v0.3.6-beta https://github.com/lightninglabs/aperture.git
WORKDIR /build/aperture
RUN go build -o /aperture ./cmd/aperture

# Stage 2: Final runtime image
FROM ubuntu:22.04

# Install dependencies
RUN apt-get update && apt-get install -y \
    curl \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Install Node.js 20
RUN curl -fsSL https://deb.nodesource.com/setup_20.x | bash - \
    && apt-get install -y nodejs

# Copy Aperture binary from builder
COPY --from=aperture-builder /aperture /usr/local/bin/aperture
RUN chmod +x /usr/local/bin/aperture

# Create app directory
WORKDIR /app

# Copy and install backend
COPY proxy/server.js /app/backend/server.js
RUN cd /app/backend && npm init -y && npm install express@4

# Copy configs
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
