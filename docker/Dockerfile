# =============================================================================
# SatGate - Lightning API Gateway - Docker Image
# =============================================================================

FROM node:20-alpine AS base

# Install security updates
RUN apk update && apk upgrade && apk add --no-cache dumb-init

# Create non-root user
RUN addgroup -g 1001 nodejs && adduser -S -u 1001 -G nodejs nodejs

WORKDIR /app

# =============================================================================
# Dependencies stage
# =============================================================================
FROM base AS deps

COPY package*.json ./
RUN npm ci --only=production && npm cache clean --force

# =============================================================================
# Production stage
# =============================================================================
FROM base AS production

ENV NODE_ENV=production
ENV BACKEND_PORT=8083

# Copy dependencies
COPY --from=deps /app/node_modules ./node_modules

# Copy application files
COPY --chown=nodejs:nodejs backend ./backend
COPY --chown=nodejs:nodejs frontend ./frontend

# Switch to non-root user
USER nodejs

# Expose port
EXPOSE 8083

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8083/health || exit 1

# Use dumb-init to handle signals properly
ENTRYPOINT ["/usr/bin/dumb-init", "--"]

# Start the application
CMD ["node", "backend/server.js"]


