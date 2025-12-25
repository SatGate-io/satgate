# phoenixd - ACINQ's headless Lightning daemon
# Provides automatic liquidity management via LSP
#
# Benefits:
# - Zero channel management
# - Self-custodial
# - ~1% fee per inbound payment (to ACINQ LSP)

FROM debian:bookworm-slim

# Install dependencies
RUN apt-get update && apt-get install -y \
    curl \
    unzip \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Set phoenixd version
ARG PHOENIXD_VERSION=0.3.4
ARG TARGETARCH

# Download and install phoenixd
WORKDIR /opt
RUN case "${TARGETARCH}" in \
        "amd64") ARCH="linux-x64" ;; \
        "arm64") ARCH="linux-arm64" ;; \
        *) echo "Unsupported arch: ${TARGETARCH}" && exit 1 ;; \
    esac && \
    curl -L "https://github.com/ACINQ/phoenixd/releases/download/v${PHOENIXD_VERSION}/phoenix-${PHOENIXD_VERSION}-${ARCH}.zip" -o phoenixd.zip && \
    unzip phoenixd.zip && \
    mv phoenix-${PHOENIXD_VERSION}-${ARCH} phoenixd && \
    rm phoenixd.zip && \
    chmod +x /opt/phoenixd/phoenixd /opt/phoenixd/phoenix-cli

# Add to path
ENV PATH="/opt/phoenixd:${PATH}"

# Create data directory
RUN mkdir -p /data/.phoenix

# Expose HTTP API port
EXPOSE 9740

# Environment variables
ENV PHOENIX_DATADIR=/data/.phoenix
ENV PHOENIX_HTTP_BIND_IP=0.0.0.0
ENV PHOENIX_HTTP_BIND_PORT=9740
ENV PHOENIX_CHAIN=mainnet

# Copy startup script
COPY start.sh /start.sh
RUN chmod +x /start.sh

# Persistent data volume
VOLUME ["/data"]

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
    CMD curl -f http://localhost:9740/getinfo || exit 1

CMD ["/start.sh"]


