#!/bin/bash
set -e

# SatGate Cloud - Fly.io Deployment Script
# Prerequisites: flyctl installed and authenticated
#
# Usage:
#   ./deploy-fly.sh all           # Interactive mode with prompts
#   ./deploy-fly.sh all --yes     # Non-interactive (CI/automation)
#   ./deploy-fly.sh postgres      # Just create Postgres
#   ./deploy-fly.sh test          # Just run smoke tests

echo "╔══════════════════════════════════════════════════════════════════╗"
echo "║              SatGate Cloud - Fly.io Deployment                    ║"
echo "╚══════════════════════════════════════════════════════════════════╝"

# Parse --yes flag
INTERACTIVE=true
for arg in "$@"; do
  if [ "$arg" = "--yes" ] || [ "$arg" = "-y" ]; then
    INTERACTIVE=false
  fi
done

# Configuration
REGION="${FLY_REGION:-ord}"
ORG="${FLY_ORG:-personal}"

CONTROL_PLANE_APP="satgate-cloud-control"
DATA_PLANE_APP="satgate-cloud-data"
DASHBOARD_APP="satgate-cloud-dashboard"
POSTGRES_APP="satgate-cloud-db"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

info() { echo -e "${GREEN}[INFO]${NC} $1"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1"; exit 1; }

# Check prerequisites
command -v fly >/dev/null 2>&1 || error "flyctl not installed. Run: curl -L https://fly.io/install.sh | sh"
fly auth whoami >/dev/null 2>&1 || error "Not logged in to Fly. Run: fly auth login"

echo ""
info "Region: $REGION"
info "Organization: $ORG"
echo ""

# Step 1: Create Postgres
create_postgres() {
    info "Step 1: Creating Postgres database..."
    
    if fly apps list | grep -q "$POSTGRES_APP"; then
        warn "Postgres app already exists: $POSTGRES_APP"
    else
        fly postgres create \
            --name "$POSTGRES_APP" \
            --region "$REGION" \
            --org "$ORG" \
            --initial-cluster-size 1 \
            --vm-size shared-cpu-1x \
            --volume-size 10
        
        info "Postgres created. Getting connection string..."
    fi
    
    # Get connection string
    DATABASE_URL=$(fly postgres config show --app "$POSTGRES_APP" | grep "Connection string" | cut -d: -f2- | xargs)
    
    if [ -z "$DATABASE_URL" ]; then
        warn "Could not get DATABASE_URL automatically."
        echo "Run: fly postgres connect -a $POSTGRES_APP"
        echo "Then copy the connection string."
    else
        info "DATABASE_URL obtained"
    fi
}

# Step 2: Apply schema
apply_schema() {
    info "Step 2: Applying database schema..."
    
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    SCHEMA_FILE="$SCRIPT_DIR/../db/schema.sql"
    
    if [ ! -f "$SCHEMA_FILE" ]; then
        error "Schema file not found: $SCHEMA_FILE"
    fi
    
    echo "Applying schema via fly postgres connect..."
    fly postgres connect -a "$POSTGRES_APP" < "$SCHEMA_FILE"
    
    info "Schema applied successfully"
}

# Step 3: Generate secrets
generate_secrets() {
    info "Step 3: Generating secrets..."
    
    SECRETS_ENCRYPTION_KEY=$(openssl rand -base64 32)
    INTERNAL_AUTH_TOKEN=$(openssl rand -hex 32)
    L402_ROOT_KEY=$(openssl rand -hex 32)
    
    echo ""
    echo "Generated secrets (save these!):"
    echo "================================"
    echo "SECRETS_ENCRYPTION_KEY=$SECRETS_ENCRYPTION_KEY"
    echo "INTERNAL_AUTH_TOKEN=$INTERNAL_AUTH_TOKEN"
    echo "L402_ROOT_KEY=$L402_ROOT_KEY"
    echo "================================"
    echo ""
    
    # Export for later use
    export SECRETS_ENCRYPTION_KEY
    export INTERNAL_AUTH_TOKEN
    export L402_ROOT_KEY
}

# Step 4: Deploy Control Plane
deploy_control_plane() {
    info "Step 4: Deploying Control Plane..."
    
    cd "$(dirname "${BASH_SOURCE[0]}")/../apps/control-plane"
    
    # Create app if doesn't exist
    if ! fly apps list | grep -q "$CONTROL_PLANE_APP"; then
        fly apps create "$CONTROL_PLANE_APP" --org "$ORG"
    fi
    
    # Set secrets
    echo "Setting secrets for control plane..."
    fly secrets set \
        DATABASE_URL="$DATABASE_URL" \
        SECRETS_ENCRYPTION_KEY="$SECRETS_ENCRYPTION_KEY" \
        DATA_PLANE_URL="https://$DATA_PLANE_APP.fly.dev" \
        DATA_PLANE_INTERNAL_TOKEN="$INTERNAL_AUTH_TOKEN" \
        --app "$CONTROL_PLANE_APP"
    
    warn "You still need to set email secrets (SMTP_HOST, SMTP_PORT, SMTP_USER, SMTP_PASS, EMAIL_FROM, APP_URL)"
    
    # Deploy
    fly deploy --app "$CONTROL_PLANE_APP"
    
    info "Control plane deployed: https://$CONTROL_PLANE_APP.fly.dev"
}

# Step 5: Deploy Data Plane
deploy_data_plane() {
    info "Step 5: Deploying Data Plane..."
    
    cd "$(dirname "${BASH_SOURCE[0]}")/../apps/data-plane"
    
    # Create app if doesn't exist
    if ! fly apps list | grep -q "$DATA_PLANE_APP"; then
        fly apps create "$DATA_PLANE_APP" --org "$ORG"
    fi
    
    # Set secrets
    echo "Setting secrets for data plane..."
    fly secrets set \
        DATABASE_URL="$DATABASE_URL" \
        SECRETS_ENCRYPTION_KEY="$SECRETS_ENCRYPTION_KEY" \
        INTERNAL_AUTH_TOKEN="$INTERNAL_AUTH_TOKEN" \
        L402_ROOT_KEY="$L402_ROOT_KEY" \
        --app "$DATA_PLANE_APP"
    
    # Deploy
    fly deploy --app "$DATA_PLANE_APP"
    
    info "Data plane deployed: https://$DATA_PLANE_APP.fly.dev"
}

# Step 6: Deploy Dashboard
deploy_dashboard() {
    info "Step 6: Deploying Dashboard..."
    
    cd "$(dirname "${BASH_SOURCE[0]}")/../apps/dashboard"
    
    # Create app if doesn't exist
    if ! fly apps list | grep -q "$DASHBOARD_APP"; then
        fly apps create "$DASHBOARD_APP" --org "$ORG"
    fi
    
    # Deploy with build arg
    fly deploy \
        --app "$DASHBOARD_APP" \
        --build-arg NEXT_PUBLIC_API_URL="https://$CONTROL_PLANE_APP.fly.dev"
    
    info "Dashboard deployed: https://$DASHBOARD_APP.fly.dev"
}

# Step 7: Configure wildcard domain
configure_wildcard() {
    info "Step 7: Configuring wildcard domain..."
    
    echo ""
    echo "To set up *.satgate.cloud:"
    echo "1. Run: fly certs create '*.satgate.cloud' -a $DATA_PLANE_APP"
    echo "2. Add DNS: *.satgate.cloud CNAME $DATA_PLANE_APP.fly.dev"
    echo ""
    echo "To set up cloud.satgate.io for dashboard:"
    echo "1. Run: fly certs create cloud.satgate.io -a $DASHBOARD_APP"
    echo "2. Add DNS: cloud.satgate.io CNAME $DASHBOARD_APP.fly.dev"
    echo ""
}

# Step 8: Smoke test
smoke_test() {
    info "Step 8: Running smoke tests..."
    
    echo ""
    echo "Testing control plane health..."
    curl -sf "https://$CONTROL_PLANE_APP.fly.dev/healthz" && echo " ✓ Control plane healthy" || echo " ✗ Control plane unhealthy"
    
    echo "Testing data plane health..."
    curl -sf "https://$DATA_PLANE_APP.fly.dev/healthz" && echo " ✓ Data plane healthy" || echo " ✗ Data plane unhealthy"
    
    echo "Testing dashboard..."
    curl -sf "https://$DASHBOARD_APP.fly.dev" > /dev/null && echo " ✓ Dashboard responding" || echo " ✗ Dashboard not responding"
    
    echo ""
    info "Deployment complete!"
    echo ""
    echo "Next steps:"
    echo "1. Set SMTP secrets on control plane for magic link emails"
    echo "2. Configure custom domains (see Step 7 output)"
    echo "3. Test login flow on dashboard"
    echo "4. Create a project and upload a config"
    echo ""
}

# Main
case "${1:-all}" in
    postgres)
        create_postgres
        ;;
    schema)
        apply_schema
        ;;
    secrets)
        generate_secrets
        ;;
    control-plane)
        deploy_control_plane
        ;;
    data-plane)
        deploy_data_plane
        ;;
    dashboard)
        deploy_dashboard
        ;;
    domains)
        configure_wildcard
        ;;
    test)
        smoke_test
        ;;
    all)
        create_postgres
        echo ""
        if [ "$INTERACTIVE" = true ]; then
            read -p "Press enter to continue after reviewing Postgres setup..."
        else
            info "Continuing (non-interactive mode)..."
            sleep 2
        fi
        apply_schema
        generate_secrets
        echo ""
        if [ "$INTERACTIVE" = true ]; then
            read -p "Press enter to continue after saving secrets..."
        else
            info "Continuing (non-interactive mode)..."
        fi
        deploy_control_plane
        deploy_data_plane
        deploy_dashboard
        configure_wildcard
        smoke_test
        ;;
    *)
        echo "Usage: $0 {all|postgres|schema|secrets|control-plane|data-plane|dashboard|domains|test}"
        exit 1
        ;;
esac

