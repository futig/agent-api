.PHONY: help build run run-local run-prod test lint clean docker-build docker-up docker-down migrate-up migrate-down sqlc-generate

# Default target
help:
	@echo "Available commands:"
	@echo "  make build         - Build the application binary"
	@echo "  make run-local     - Run the application with .env.local"
	@echo "  make run-prod      - Run the application with .env.prod"
	@echo "  make test          - Run tests"
	@echo "  make lint          - Run linters"
	@echo "  make clean         - Clean build artifacts"
	@echo "  make docker-build  - Build Docker image"
	@echo "  make docker-up     - Start services with docker-compose"
	@echo "  make docker-down   - Stop services with docker-compose"
	@echo "  make migrate-up    - Run database migrations up"
	@echo "  make migrate-down  - Run database migrations down"
	@echo "  make sqlc-generate - Generate code from SQL queries"

# Build the application
build:
	@echo "Building agent-backend..."
	@go build -o bin/agent-backend ./cmd/agent-backend
	@echo "Build complete: bin/agent-backend"

# Run with local environment
run-local:
	@echo "Starting agent-backend (local environment)..."
	@go run ./cmd/agent-backend -env=local

# Run with production environment
run-prod:
	@echo "Starting agent-backend (production environment)..."
	@go run ./cmd/agent-backend -env=prod

# Run tests
test:
	@echo "Running tests..."
	@go test -v -race -cover ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run linters
lint:
	@echo "Running linters..."
	@golangci-lint run ./...

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...
	@gofmt -s -w .

# Tidy dependencies
tidy:
	@echo "Tidying dependencies..."
	@go mod tidy

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html
	@echo "Clean complete"

# Build Docker image
docker-build:
	@echo "Building Docker image..."
	@docker build -t agent-backend:latest .
	@echo "Docker image built: agent-backend:latest"

# Start services with docker-compose
docker-up:
	@echo "Starting services with docker-compose..."
	@docker-compose up -d
	@echo "Services started"

# Stop services with docker-compose
docker-down:
	@echo "Stopping services with docker-compose..."
	@docker-compose down
	@echo "Services stopped"

# View logs from docker-compose
docker-logs:
	@docker-compose logs -f

# Run database migrations up
migrate-up:
	@echo "Running database migrations..."
	@go run ./cmd/agent-backend -env=local &
	@sleep 2
	@pkill -f agent-backend || true
	@echo "Migrations complete"

# Run database migrations down
migrate-down:
	@echo "Rolling back database migrations..."
	@migrate -path internal/repository/migrations -database "$(DATABASE_URL)" down
	@echo "Rollback complete"

# Generate code from SQL queries using sqlc
sqlc-generate:
	@echo "Generating code from SQL queries..."
	@sqlc generate
	@echo "Code generation complete"

# Install development tools
install-tools:
	@echo "Installing development tools..."
	@go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	@go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install go.uber.org/mock/mockgen@latest
	@echo "Tools installed"

# Create necessary directories
setup-dirs:
	@echo "Creating necessary directories..."
	@mkdir -p data/projects data/sessions bin
	@echo "Directories created"

# Complete setup for new developers
setup: install-tools setup-dirs tidy
	@echo "Setup complete! Copy .env.local and update with your settings."
	@echo "Then run: make run-local"
