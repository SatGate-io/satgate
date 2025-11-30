# =============================================================================
# SatGate Makefile
# =============================================================================
# Common commands for development and deployment
# =============================================================================

.PHONY: help install dev start stop logs clean build test

# Default target
help:
	@echo ""
	@echo "  SatGate - L402 Lightning API Gateway"
	@echo ""
	@echo "  Usage: make <target>"
	@echo ""
	@echo "  Targets:"
	@echo "    install    - Run the installer script"
	@echo "    dev        - Start development environment (local)"
	@echo "    start      - Start production stack (Docker)"
	@echo "    stop       - Stop all services"
	@echo "    logs       - View logs"
	@echo "    clean      - Remove containers and volumes"
	@echo "    build      - Build Docker images"
	@echo "    test       - Run tests"
	@echo ""

# Run installer
install:
	@./install.sh

# Development (local, no Docker)
dev:
	@echo "Starting development environment..."
	@echo "Backend: http://localhost:8083"
	@echo "Aperture: http://localhost:8081 (run separately)"
	@echo "Docs: http://localhost:8080"
	@npm run start &
	@python3 -m http.server 8080 &
	@echo "Done. Run Aperture separately with your LNC credentials."

# Start production stack
start:
	@docker compose -f docker-compose.full.yml up -d
	@echo ""
	@echo "SatGate started!"
	@echo "  Playground: http://localhost:8080/docs/"
	@echo "  API:        http://localhost:8081/"

# Stop all services
stop:
	@docker compose -f docker-compose.full.yml down

# View logs
logs:
	@docker compose -f docker-compose.full.yml logs -f

# Clean up
clean:
	@docker compose -f docker-compose.full.yml down -v --rmi local
	@echo "Cleaned up containers, volumes, and images."

# Build images
build:
	@docker compose -f docker-compose.full.yml build

# Run tests
test:
	@echo "Testing free endpoint..."
	@curl -s http://localhost:8081/api/free/ping | jq
	@echo ""
	@echo "Testing paid endpoint (should return 402)..."
	@curl -s -w "\nStatus: %{http_code}\n" http://localhost:8081/api/basic/quote

