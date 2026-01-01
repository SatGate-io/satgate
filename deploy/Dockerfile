# SatGate Demo - Multi-service Dockerfile
# Runs Aperture (L402 gateway) + Backend (Demo API)

FROM node:20-slim AS backend-builder

WORKDIR /app/backend
COPY deploy/package.json ./
RUN npm install --production
COPY proxy/server.js ./
COPY proxy/public ./public

# -------------------------------------------
FROM ubuntu:22.04

# Install dependencies
RUN apt-get update && apt-get install -y \
    curl \
    ca-certificates \
    supervisor \
    && rm -rf /var/lib/apt/lists/*

# Install Node.js 20
RUN curl -fsSL https://deb.nodesource.com/setup_20.x | bash - \
    && apt-get install -y nodejs

# Install Aperture
RUN curl -L https://github.com/lightninglabs/aperture/releases/download/v0.3.6-beta/aperture-linux-amd64-v0.3.6-beta.tar.gz \
    | tar -xz -C /usr/local/bin --strip-components=1

# Create app directory
WORKDIR /app

# Copy backend from builder stage
COPY --from=backend-builder /app/backend /app/backend

# Copy dashboard UI (static files)
COPY proxy/public /app/backend/public

# Copy Aperture config
COPY deploy/aperture.cloud.yaml /app/aperture.yaml

# Copy supervisor config
COPY deploy/supervisord.conf /etc/supervisor/conf.d/satgate.conf

# Create data directories
RUN mkdir -p /app/data /root/.aperture

# Environment variables (set these in Railway)
ENV LNC_PASSPHRASE=""
ENV LNC_MAILBOX="mailbox.terminal.lightning.today:443"
ENV LNC_NETWORK="mainnet"
ENV BACKEND_PORT=8083
ENV PORT=8081

# Expose Aperture port
EXPOSE 8081

# Start supervisor (manages both Aperture and Backend)
CMD ["/usr/bin/supervisord", "-c", "/etc/supervisor/conf.d/satgate.conf"]

