.PHONY: all build clean test lint run install dev-setup help

# Build variables
BINARY_NAME=mindbalancer
CLI_NAME=mindsql
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

# Go variables
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOVET=$(GOCMD) vet

# Directories
BIN_DIR=bin
CMD_DIR=cmd

all: build

## build: Build all binaries
build: build-server build-cli

## build-server: Build the main server binary
build-server:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BIN_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME) ./$(CMD_DIR)/$(BINARY_NAME)

## build-cli: Build the CLI binary
build-cli:
	@echo "Building $(CLI_NAME)..."
	@mkdir -p $(BIN_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BIN_DIR)/$(CLI_NAME) ./$(CMD_DIR)/$(CLI_NAME)

## clean: Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf $(BIN_DIR)

## test: Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v -race -cover ./...

## test-coverage: Run tests with coverage report
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

## lint: Run linter
lint:
	@echo "Running linter..."
	golangci-lint run ./...

## vet: Run go vet
vet:
	@echo "Running go vet..."
	$(GOVET) ./...

## fmt: Format code
fmt:
	@echo "Formatting code..."
	$(GOCMD) fmt ./...

## run: Run the server
run: build-server
	@echo "Running $(BINARY_NAME)..."
	./$(BIN_DIR)/$(BINARY_NAME)

## run-dev: Run with hot reload (requires air)
run-dev:
	@echo "Running in development mode..."
	air

## install: Install binaries to GOPATH
install:
	@echo "Installing..."
	$(GOCMD) install ./$(CMD_DIR)/$(BINARY_NAME)
	$(GOCMD) install ./$(CMD_DIR)/$(CLI_NAME)

## dev-setup: Set up development environment
dev-setup:
	@echo "Setting up development environment..."
	$(GOMOD) download
	$(GOMOD) tidy
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/air-verse/air@latest
	@echo "Development setup complete!"

## deps: Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

## docker-build: Build Docker image
docker-build:
	@echo "Building Docker image..."
	docker build -t mindbalancer:$(VERSION) .

## docker-run: Run Docker container
docker-run:
	@echo "Running Docker container..."
	docker run -d -p 6032:6032 -p 6033:6033 -p 6034:6034 -p 9090:9090 --name mindbalancer mindbalancer:$(VERSION)

## release: Create release builds for multiple platforms
release:
	@echo "Creating release builds..."
	@mkdir -p $(BIN_DIR)/release
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BIN_DIR)/release/$(BINARY_NAME)-linux-amd64 ./$(CMD_DIR)/$(BINARY_NAME)
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BIN_DIR)/release/$(BINARY_NAME)-linux-arm64 ./$(CMD_DIR)/$(BINARY_NAME)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BIN_DIR)/release/$(BINARY_NAME)-darwin-amd64 ./$(CMD_DIR)/$(BINARY_NAME)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BIN_DIR)/release/$(BINARY_NAME)-darwin-arm64 ./$(CMD_DIR)/$(BINARY_NAME)
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BIN_DIR)/release/$(BINARY_NAME)-windows-amd64.exe ./$(CMD_DIR)/$(BINARY_NAME)
	@echo "Release builds created in $(BIN_DIR)/release/"

## help: Show this help message
help:
	@echo "MindBalancer - The ProxySQL for AI"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'
