# Makefile for Order Processing System
# Provides common commands for building, testing, and running the application.


.PHONY: help build clean test run

help: ## Show this help message
	@echo 'Usage : make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?##/ {printf "  %-15s %s\n", $$1, $$2} END {printf "\n"}' $(MAKEFILE_LIST)
	
# =================================================================
# Building
# =================================================================

build: ## Build the monolithic application
	@echo "Building monolithic version..."
	@go build -o bin/api-gateway ./cmd/api-gateway

# =================================================================
# Testing
# =================================================================

test: ## Run unit tests
	@echo "Running unit tests..."
	@go test ./... -v

test-coverage: ## Run unit tests with coverage
	@echo "Running unit tests with coverage..."
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-integration: ## Run integration tests
	@echo "Running integration tests..."
	@go test -v -tags=integration ./tests/integration/...

test-all: test test-integration ## Run all tests

# =================================================================
# Running
# =================================================================

run: ## Run API Gateway (monolith)
	@go run ./cmd/api-gateway

run air: ## Run API Gateway with air (Hot Reload) - requires infrastructure running
	@echo "Starting with Air hot reload..."
#	@echo "Make sure infrastructure is running: make dev-infra"
	@air -c .air.toml

run-api-gateway: run ## Alias for run

# =================================================================
# Docker
# =================================================================
docker-build: ## Build Docker image for API Gateway
	@echo "Building Docker image..."
	@docker build -f deployments/docker/Dockerfile.api-gateway -t api-gateway:latest .

docker-up: ## Start full stack with Docker compose (Production like)
	@docker-compose up -d

docker-down: ## Stop Docker compose services
	@docker-compose down

docker-logs: ## View logs from Docker compose services
	@docker-compose logs -f

# =================================================================
# Cleanup
# =================================================================

clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html

clean-all: clean ## Clean everyting including dependencies
	@go clean -modcache