# SatGate Cloud - Dashboard
FROM node:20-slim AS base

# Install pnpm
RUN npm install -g pnpm

WORKDIR /app

# Copy workspace files
COPY package.json pnpm-workspace.yaml turbo.json ./
COPY apps/dashboard/package.json ./apps/dashboard/
COPY packages/gateway-config/package.json ./packages/gateway-config/

# Install dependencies
RUN pnpm install --frozen-lockfile

# Copy source
COPY packages/gateway-config/ ./packages/gateway-config/
COPY apps/dashboard/ ./apps/dashboard/

# Build
WORKDIR /app/apps/dashboard

ARG NEXT_PUBLIC_API_URL
ENV NEXT_PUBLIC_API_URL=${NEXT_PUBLIC_API_URL}

RUN pnpm build

# Production stage
FROM node:20-slim AS runner

WORKDIR /app

ENV NODE_ENV=production
ENV PORT=3000

# Copy Next.js standalone output
COPY --from=base /app/apps/dashboard/.next/standalone ./
COPY --from=base /app/apps/dashboard/.next/static ./.next/static
COPY --from=base /app/apps/dashboard/public ./public

EXPOSE 3000

CMD ["node", "server.js"]

