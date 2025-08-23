# Makefile for ArangoDB Backup/Restore CLI

# Variables
BINARY_NAME=arangodb-bk-restore
BUILD_DIR=build
VERSION?=1.0.0
GOOS?=$(shell go env GOOS)
GOARCH?=$(shell go env GOARCH)

# Default target
.PHONY: all
all: clean build

# Build the binary
.PHONY: build
build:
	@echo "Building $(BINARY_NAME) for $(GOOS)/$(GOARCH)..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) -ldflags="-X main.version=$(VERSION)" .
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Build for multiple platforms
.PHONY: build-all
build-all:
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	
	@echo "Building for Linux/amd64..."
	@GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 -ldflags="-X main.version=$(VERSION)" .
	
	@echo "Building for Darwin/amd64..."
	@GOOS=darwin GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 -ldflags="-X main.version=$(VERSION)" .
	
	@echo "Building for Windows/amd64..."
	@GOOS=windows GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe -ldflags="-X main.version=$(VERSION)" .
	
	@echo "Build complete for all platforms"

# Install dependencies
.PHONY: deps
deps:
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy
	@echo "Dependencies installed"

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	@go test -v ./...
	@echo "Tests complete"

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@echo "Clean complete"

# Install the binary
.PHONY: install
install: build
	@echo "Installing $(BINARY_NAME)..."
	@cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/
	@echo "Installation complete"

# Uninstall the binary
.PHONY: uninstall
uninstall:
	@echo "Uninstalling $(BINARY_NAME)..."
	@rm -f /usr/local/bin/$(BINARY_NAME)
	@echo "Uninstallation complete"

# Run the CLI with help
.PHONY: help
help: build
	@echo "CLI Help:"
	@./$(BUILD_DIR)/$(BINARY_NAME) --help

# Run backup command help
.PHONY: backup-help
backup-help: build
	@echo "Backup Command Help:"
	@./$(BUILD_DIR)/$(BINARY_NAME) backup --help

# Run restore command help
.PHONY: restore-help
restore-help: build
	@echo "Restore Command Help:"
	@./$(BUILD_DIR)/$(BINARY_NAME) restore --help

# Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	@go fmt ./...
	@echo "Code formatting complete"

# Lint code
.PHONY: lint
lint:
	@echo "Linting code..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found, skipping linting"; \
	fi

# Vet code
.PHONY: vet
vet:
	@echo "Vetting code..."
	@go vet ./...
	@echo "Code vetting complete"

# Check code quality
.PHONY: check
check: fmt vet lint
	@echo "Code quality checks complete"

# Create release package
.PHONY: release
release: clean build-all
	@echo "Creating release package..."
	@mkdir -p release
	@cd $(BUILD_DIR) && tar -czf ../release/$(BINARY_NAME)-$(VERSION)-$(GOOS)-$(GOARCH).tar.gz *
	@echo "Release package created: release/$(BINARY_NAME)-$(VERSION)-$(GOOS)-$(GOARCH).tar.gz"

# Show help
.PHONY: help-make
help-make:
	@echo "Available targets:"
	@echo "  build          - Build the binary for current platform"
	@echo "  build-all      - Build for multiple platforms (Linux, Darwin, Windows)"
	@echo "  deps           - Install dependencies"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo "  clean          - Clean build artifacts"
	@echo "  install        - Install binary to /usr/local/bin"
	@echo "  uninstall      - Remove binary from /usr/local/bin"
	@echo "  help           - Show CLI help"
	@echo "  backup-help    - Show backup command help"
	@echo "  restore-help   - Show restore command help"
	@echo "  fmt            - Format code"
	@echo "  lint           - Lint code (requires golangci-lint)"
	@echo "  vet            - Vet code"
	@echo "  check          - Run all code quality checks"
	@echo "  release        - Create release package"
	@echo "  help-make      - Show this help message"
