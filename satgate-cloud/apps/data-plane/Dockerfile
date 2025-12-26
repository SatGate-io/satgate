# SatGate Cloud - Data Plane
FROM node:20-slim AS base

# Install pnpm
RUN npm install -g pnpm

WORKDIR /app

# Copy workspace files
COPY package.json pnpm-workspace.yaml turbo.json ./
COPY apps/data-plane/package.json ./apps/data-plane/
COPY packages/common/package.json ./packages/common/
COPY packages/gateway-config/package.json ./packages/gateway-config/
COPY packages/l402-core/package.json ./packages/l402-core/

# Install dependencies
RUN pnpm install --frozen-lockfile

# Copy source
COPY packages/ ./packages/
COPY apps/data-plane/ ./apps/data-plane/

# Build packages first, then data plane
RUN pnpm --filter @satgate/common build
RUN pnpm --filter @satgate/l402-core build
RUN pnpm --filter @satgate/gateway-config build
RUN pnpm --filter @satgate/data-plane build

# Production stage
FROM node:20-slim AS runner

WORKDIR /app

# Copy built artifacts
COPY --from=base /app/node_modules ./node_modules
COPY --from=base /app/apps/data-plane/dist ./dist
COPY --from=base /app/apps/data-plane/package.json ./
COPY --from=base /app/packages/common/dist ./node_modules/@satgate/common/dist
COPY --from=base /app/packages/gateway-config/dist ./node_modules/@satgate/gateway-config/dist
COPY --from=base /app/packages/l402-core/dist ./node_modules/@satgate/l402-core/dist

ENV NODE_ENV=production
EXPOSE 8080

CMD ["node", "dist/index.js"]

