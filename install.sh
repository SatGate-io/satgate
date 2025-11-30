#!/usr/bin/env bash
# =============================================================================
# SatGate Installer
# =============================================================================
# One-command setup for SatGate L402 API Gateway
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/your-org/satgate/main/install.sh | bash
#   
# Or clone and run:
#   git clone https://github.com/your-org/satgate.git
#   cd satgate
#   ./install.sh
#
# =============================================================================

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Banner
echo -e "${PURPLE}"
echo "  ___       _    ___       _       "
echo " / __| __ _| |_ / __|__ _ | |_ ___ "
echo " \__ \/ _\` |  _| (_ / _\` ||  _/ -_)"
echo " |___/\__,_|\__|___\__,_| \__\___|"
echo -e "${NC}"
echo -e "${CYAN}L402 Lightning API Gateway${NC}"
echo ""

# -----------------------------------------------------------------------------
# Check Prerequisites
# -----------------------------------------------------------------------------
echo -e "${BLUE}[1/5]${NC} Checking prerequisites..."

# Check Docker
if ! command -v docker &> /dev/null; then
    echo -e "${RED}✗ Docker not found${NC}"
    echo "  Install Docker: https://docs.docker.com/get-docker/"
    exit 1
fi
echo -e "${GREEN}✓${NC} Docker found"

# Check Docker Compose
if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
    echo -e "${RED}✗ Docker Compose not found${NC}"
    echo "  Install Docker Compose: https://docs.docker.com/compose/install/"
    exit 1
fi
echo -e "${GREEN}✓${NC} Docker Compose found"

# Check if running from repo or need to clone
if [ ! -f "docker-compose.full.yml" ]; then
    echo -e "${BLUE}[2/5]${NC} Cloning SatGate repository..."
    git clone https://github.com/your-org/satgate.git satgate-install
    cd satgate-install
else
    echo -e "${BLUE}[2/5]${NC} Running from existing directory..."
fi

# -----------------------------------------------------------------------------
# Configure Environment
# -----------------------------------------------------------------------------
echo -e "${BLUE}[3/5]${NC} Configuring environment..."

if [ ! -f ".env" ]; then
    cp env.example .env
    echo -e "${YELLOW}⚠${NC}  Created .env file from template"
    echo ""
    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${YELLOW}IMPORTANT: You need to configure your Lightning Node Connect credentials${NC}"
    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    echo "1. Go to ${CYAN}https://terminal.lightning.engineering${NC}"
    echo "2. Navigate to Lightning Node Connect"
    echo "3. Create a new session (or use existing)"
    echo "4. Copy your 10-word pairing phrase"
    echo ""
    read -p "Enter your LNC pairing phrase: " LNC_PHRASE
    
    if [ -n "$LNC_PHRASE" ]; then
        # Update .env file with the phrase
        if [[ "$OSTYPE" == "darwin"* ]]; then
            sed -i '' "s/your ten word pairing phrase goes here/$LNC_PHRASE/" .env
        else
            sed -i "s/your ten word pairing phrase goes here/$LNC_PHRASE/" .env
        fi
        echo -e "${GREEN}✓${NC} LNC credentials saved to .env"
    else
        echo -e "${YELLOW}⚠${NC}  No phrase entered. Edit .env manually before starting."
    fi
else
    echo -e "${GREEN}✓${NC} .env file already exists"
fi

# -----------------------------------------------------------------------------
# Build Images
# -----------------------------------------------------------------------------
echo -e "${BLUE}[4/5]${NC} Building Docker images..."

if docker compose version &> /dev/null; then
    docker compose -f docker-compose.full.yml build
else
    docker-compose -f docker-compose.full.yml build
fi

echo -e "${GREEN}✓${NC} Images built successfully"

# -----------------------------------------------------------------------------
# Start Services
# -----------------------------------------------------------------------------
echo -e "${BLUE}[5/5]${NC} Starting SatGate..."

read -p "Start SatGate now? (y/n) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    if docker compose version &> /dev/null; then
        docker compose -f docker-compose.full.yml up -d
    else
        docker-compose -f docker-compose.full.yml up -d
    fi
    
    echo ""
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}✓ SatGate is running!${NC}"
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    echo -e "  ${CYAN}Playground:${NC}  http://localhost:8080/docs/"
    echo -e "  ${CYAN}API Gateway:${NC} http://localhost:8081/"
    echo ""
    echo -e "  ${PURPLE}Try it:${NC}"
    echo "    curl http://localhost:8081/api/basic/quote"
    echo ""
    echo -e "  ${PURPLE}View logs:${NC}"
    echo "    docker compose -f docker-compose.full.yml logs -f"
    echo ""
    echo -e "  ${PURPLE}Stop:${NC}"
    echo "    docker compose -f docker-compose.full.yml down"
    echo ""
else
    echo ""
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}✓ SatGate is ready!${NC}"
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    echo "To start SatGate, run:"
    echo -e "  ${CYAN}docker compose -f docker-compose.full.yml up -d${NC}"
    echo ""
fi

