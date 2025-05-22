.PHONY: all build run test clean docker-build docker-up docker-down migrate-up migrate-down migrate-create lint help

# Variables
BINARY_NAME=vote
MAIN_FILE=main.go
CONFIG_FILE=config.yaml
DOCKER_COMPOSE=docker-compose.yml

# Go related variables
GO=go
GOFMT=gofmt
GOLINT=golangci-lint
GOFILES=$(shell find . -name "*.go" -type f -not -path "./vendor/*")

# Docker related variables
DOCKER=docker
DOCKER_COMPOSE=docker-compose

# Default target
all: clean build

# Build the application
build:
	@echo "Building $(BINARY_NAME)..."
	$(GO) build -o $(BINARY_NAME) $(MAIN_FILE)

# Run the application
run: build
	@echo "Running $(BINARY_NAME)..."
	./$(BINARY_NAME) server

# Run tests
test:
	@echo "Running tests..."
	$(GO) test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GO) test -v -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out

# Clean build files
clean:
	@echo "Cleaning..."
	rm -f $(BINARY_NAME)
	rm -f coverage.out
	$(GO) clean

# Run linter
lint:
	@echo "Running linter..."
	$(GOLINT) run

# Format code
fmt:
	@echo "Formatting code..."
	$(GOFMT) -w $(GOFILES)

# Docker commands
docker-build:
	@echo "Building Docker image..."
	$(DOCKER_COMPOSE) build

docker-up:
	@echo "Starting Docker containers..."
	$(DOCKER_COMPOSE) up -d

docker-down:
	@echo "Stopping Docker containers..."
	$(DOCKER_COMPOSE) down

docker-logs:
	@echo "Showing Docker logs..."
	$(DOCKER_COMPOSE) logs -f

# Database migration commands
migrate-up:
	@echo "Running migrations up..."
	./$(BINARY_NAME) migrate up

migrate-down:
	@echo "Running migrations down..."
	./$(BINARY_NAME) migrate down

migrate-create:
	@echo "Creating new migration..."
	@read -p "Enter migration name: " name; \
	./$(BINARY_NAME) migrate create $$name

# Development environment setup
dev-setup: docker-up migrate-up
	@echo "Development environment is ready!"

# Create config file if it doesn't exist
config:
	@if [ ! -f $(CONFIG_FILE) ]; then \
		echo "Creating default config file..."; \
		echo "server:" > $(CONFIG_FILE); \
		echo "  port: 8080" >> $(CONFIG_FILE); \
		echo "  env: development" >> $(CONFIG_FILE); \
		echo "postgres:" >> $(CONFIG_FILE); \
		echo "  host: localhost" >> $(CONFIG_FILE); \
		echo "  port: 5432" >> $(CONFIG_FILE); \
		echo "  user: postgres" >> $(CONFIG_FILE); \
		echo "  password: postgres" >> $(CONFIG_FILE); \
		echo "  dbname: vote" >> $(CONFIG_FILE); \
		echo "  sslmode: disable" >> $(CONFIG_FILE); \
		echo "redis:" >> $(CONFIG_FILE); \
		echo "  host: localhost" >> $(CONFIG_FILE); \
		echo "  port: 6379" >> $(CONFIG_FILE); \
		echo "  password: \"\"" >> $(CONFIG_FILE); \
		echo "  db: 0" >> $(CONFIG_FILE); \
		echo "rabbitmq:" >> $(CONFIG_FILE); \
		echo "  host: localhost" >> $(CONFIG_FILE); \
		echo "  port: 5672" >> $(CONFIG_FILE); \
		echo "  user: guest" >> $(CONFIG_FILE); \
		echo "  password: guest" >> $(CONFIG_FILE); \
		echo "  vhost: /" >> $(CONFIG_FILE); \
		echo "migration:" >> $(CONFIG_FILE); \
		echo "  auto_migrate: true" >> $(CONFIG_FILE); \
	fi

# Show help
help:
	@echo "Available commands:"
	@echo "  make build          - Build the application"
	@echo "  make run           - Run the application"
	@echo "  make test          - Run tests"
	@echo "  make test-coverage - Run tests with coverage"
	@echo "  make clean         - Clean build files"
	@echo "  make lint          - Run linter"
	@echo "  make fmt           - Format code"
	@echo "  make docker-build  - Build Docker image"
	@echo "  make docker-up     - Start Docker containers"
	@echo "  make docker-down   - Stop Docker containers"
	@echo "  make docker-logs   - Show Docker logs"
	@echo "  make migrate-up    - Run migrations up"
	@echo "  make migrate-down  - Run migrations down"
	@echo "  make migrate-create - Create new migration"
	@echo "  make dev-setup     - Setup development environment"
	@echo "  make config        - Create default config file"
	@echo "  make help          - Show this help message" 