# AWS RAG System Makefile

.PHONY: help dev-start test-all lint clean deploy infra-deploy infra-destroy backend-build frontend-build

# Default target
help:
	@echo "Available commands:"
	@echo "  dev-start      - Start development environment (frontend and backend)"
	@echo "  test-all       - Run all tests (backend and frontend)"
	@echo "  lint          - Run linters for all code"
	@echo "  clean         - Clean build artifacts"
	@echo "  backend-build - Build backend Lambda function"
	@echo "  frontend-build- Build frontend for production"
	@echo "  infra-deploy  - Deploy infrastructure with Terraform"
	@echo "  deploy        - Deploy application with SAM"
	@echo "  infra-destroy - Destroy infrastructure"

# Development
dev-start:
	@echo "Starting development environment..."
	@echo "Starting backend server..."
	@cd backend && go run ./src/main.go &
	@echo "Starting frontend development server..."
	@cd frontend && npm install && npm run dev &
	@echo "Development environment started. Backend on :8080, Frontend on :3000"

# Testing
test-all: test-backend test-frontend

test-backend:
	@echo "Running backend tests..."
	@cd backend && go test -v ./...

test-frontend:
	@echo "Running frontend tests..."
	@cd frontend && npm test

# Linting
lint: lint-backend lint-frontend

lint-backend:
	@echo "Running Go linter..."
	@cd backend && golangci-lint run

lint-frontend:
	@echo "Running TypeScript/React linter..."
	@cd frontend && npm run lint

# Building
backend-build:
	@echo "Building backend Lambda function..."
	@cd backend && make build

frontend-build:
	@echo "Building frontend for production..."
	@cd frontend && npm install && npm run build

# Infrastructure
infra-deploy:
	@echo "Deploying infrastructure with Terraform..."
	@cd infrastructure && \
		terraform init && \
		terraform plan && \
		terraform apply -auto-approve

infra-destroy:
	@echo "Destroying infrastructure with Terraform..."
	@cd infrastructure && \
		terraform destroy -auto-approve

# Application deployment
deploy: backend-build
	@echo "Deploying application with SAM..."
	@cd backend && \
		sam build && \
		sam deploy --guided

# Cleanup
clean:
	@echo "Cleaning build artifacts..."
	@cd backend && make clean
	@cd frontend && rm -rf dist node_modules/.vite
	@cd infrastructure && rm -f terraform.tfstate.backup .terraform.lock.hcl
	@cd infrastructure && rm -rf .terraform/

# Install dependencies
install:
	@echo "Installing dependencies..."
	@echo "Installing backend dependencies..."
	@cd backend && go mod download
	@echo "Installing frontend dependencies..."
	@cd frontend && npm install