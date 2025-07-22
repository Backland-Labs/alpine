.PHONY: all build test test-unit test-integration test-coverage clean install lint fmt help

# Default target
all: build

# Build the River binary
build:
	@echo "Building River..."
	@mkdir -p build
	@go build -o build/river cmd/river/main.go
	@echo "Build complete: ./build/river"

# Run all tests (unit + integration)
test: test-unit test-integration

# Run unit tests only (fast)
test-unit:
	@echo "Running unit tests..."
	@go test -short ./...

# Run integration tests (requires external dependencies)
test-integration:
	@echo "Running integration tests..."
	@go test ./test/integration/... -v

# Run integration tests with real services
test-integration-full:
	@echo "Running full integration tests (requires claude command)..."
	@RIVER_INTEGRATION_TESTS=true CLAUDE_INTEGRATION_TEST=true go test ./test/integration/... -v

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf build
	@rm -f coverage.out coverage.html
	@rm -f claude_state.json
	@echo "Clean complete"

# Install the binary to GOPATH/bin
install: build
	@echo "Installing River to GOPATH/bin..."
	@go install ./cmd/river
	@echo "Installation complete"

# Run linter
lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		exit 1; \
	fi

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...
	@echo "Formatting complete"

# Run specific integration test
test-integration-workflow:
	@echo "Running workflow integration tests..."
	@go test ./test/integration/... -run TestFullWorkflow -v


test-integration-claude:
	@echo "Running Claude integration tests..."
	@go test ./test/integration/... -run TestClaude -v

# Development helpers
run: build
	@echo "Running River..."
	@./build/river

# Watch for changes and rebuild (requires entr)
watch:
	@if command -v entr >/dev/null 2>&1; then \
		find . -name '*.go' | entr -c make build; \
	else \
		echo "entr not found. Install with: brew install entr (macOS) or apt-get install entr (Linux)"; \
		exit 1; \
	fi

# Validate GitHub Actions workflows
validate-workflows:
	@echo "Validating GitHub Actions workflows..."
	@go run test/validate_workflows.go

# Help target
help:
	@echo "River - CLI orchestrator for Claude Code"
	@echo ""
	@echo "Available targets:"
	@echo "  make build              - Build the River binary to ./build/"
	@echo "  make test               - Run all tests (unit + integration)"
	@echo "  make test-unit          - Run unit tests only (fast)"
	@echo "  make test-integration   - Run integration tests"
	@echo "  make test-coverage      - Run tests with coverage report"
	@echo "  make clean              - Remove build artifacts"
	@echo "  make install            - Install River to GOPATH/bin"
	@echo "  make lint               - Run golangci-lint"
	@echo "  make fmt                - Format code with go fmt"
	@echo "  make run                - Build and run River"
	@echo "  make watch              - Watch for changes and rebuild"
	@echo "  make validate-workflows - Validate GitHub Actions workflows"
	@echo ""
	@echo "Integration test targets:"
	@echo "  make test-integration-full     - Run with real services"
	@echo "  make test-integration-workflow - Run workflow tests only"
	@echo "  make test-integration-claude   - Run Claude tests only"
	@echo ""
	@echo "Environment variables:"
	@echo "  CLAUDE_INTEGRATION_TEST=true - Enable real Claude command tests"
	@echo "  RIVER_INTEGRATION_TESTS=true - Enable all integration tests"