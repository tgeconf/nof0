.PHONY: help run build clean test test-unit test-integration test-all test-bench test-cover test-race lint fmt check install deps migrate-up migrate-down

# Variables
APP_NAME=nof0-api
CONFIG_FILE=etc/nof0.yaml
BUILD_DIR=bin
TEST_SCRIPT=scripts/run_tests.py

help: ## Display this help message
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

# Dependencies and setup
deps: ## Install dependencies
	go mod download
	go mod tidy

migrate-up: ## Run DB migrations up (requires golang-migrate cli)
	migrate -path migrations -database "$(POSTGRES_DSN)" -verbose up

migrate-down: ## Rollback last migration (requires golang-migrate cli)
	migrate -path migrations -database "$(POSTGRES_DSN)" -verbose down 1

# Build commands
run: ## Run the application in development mode
	go run nof0.go -f $(CONFIG_FILE)

build: ## Build the application
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(APP_NAME) nof0.go
	@echo "Build complete: $(BUILD_DIR)/$(APP_NAME)"

build-linux: ## Build for Linux
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(APP_NAME)-linux nof0.go
	@echo "Build complete: $(BUILD_DIR)/$(APP_NAME)-linux"

build-windows: ## Build for Windows
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 go build -o $(BUILD_DIR)/$(APP_NAME).exe nof0.go
	@echo "Build complete: $(BUILD_DIR)/$(APP_NAME).exe"

clean: ## Clean build artifacts
	rm -rf $(BUILD_DIR)
	go clean

# Test commands
test: ## Run basic Go tests
	go test -v ./...

test-unit: ## Run unit tests using Python test script
	python $(TEST_SCRIPT) unit

test-integration: ## Run integration tests using Python test script
	python $(TEST_SCRIPT) integration

test-all: ## Run all tests (unit + integration) using Python test script
	python $(TEST_SCRIPT) all

test-bench: ## Run benchmarks
	go test -bench=. -benchmem ./...

test-cover: ## Generate test coverage report
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-race: ## Run tests with race detector
	go test -race ./...

# Code quality commands
lint: ## Run linter
	golangci-lint run

fmt: ## Format code
	go fmt ./...
	gofmt -s -w .

check: fmt lint test ## Run all checks (format, lint, test)

check-all: fmt lint test-all ## Run all checks including integration tests

# Install
install: build ## Install the binary
	@cp $(BUILD_DIR)/$(APP_NAME) $(GOPATH)/bin/
	@echo "Installed to $(GOPATH)/bin/$(APP_NAME)"
