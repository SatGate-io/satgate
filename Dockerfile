# SatGate Cloud Demo
# Single Dockerfile at repo root for easier Railway deployment

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

# Install Aperture v0.3.6-beta
RUN curl -L https://github.com/lightninglabs/aperture/releases/download/v0.3.6-beta/aperture-linux-amd64-v0.3.6-beta.tar.gz \
    | tar -xz -C /usr/local/bin --strip-components=1 \
    && chmod +x /usr/local/bin/aperture

# Create app directory
WORKDIR /app

# Copy and install backend
COPY proxy/server.js /app/backend/server.js
RUN cd /app/backend && npm init -y && npm install express

# Copy configs
COPY deploy/aperture.cloud.yaml /app/aperture.yaml
COPY deploy/supervisord.conf /etc/supervisor/conf.d/satgate.conf

# Create data directories
RUN mkdir -p /app/data /root/.aperture

# Environment variables
ENV LNC_PASSPHRASE=""
ENV LNC_MAILBOX="mailbox.terminal.lightning.today:443"
ENV LNC_NETWORK="mainnet"
ENV BACKEND_PORT=8083
ENV PORT=8081

EXPOSE 8081

CMD ["/usr/bin/supervisord", "-c", "/etc/supervisor/conf.d/satgate.conf"]

