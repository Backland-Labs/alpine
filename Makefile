.PHONY: all build test test-unit test-integration test-e2e test-coverage clean install lint fmt help

# Default target
all: build

# Build the Alpine binary and hooks
build: build-hooks
	@echo "Building Alpine..."
	@mkdir -p build
	@go build -o build/alpine cmd/alpine/main.go
	@echo "Build complete: ./build/alpine"

# Build hook binaries
build-hooks:
	@echo "Building hook binaries..."
	@$(MAKE) -C hooks all
	@echo "Hooks built successfully"

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
	@ALPINE_INTEGRATION_TESTS=true CLAUDE_INTEGRATION_TEST=true go test ./test/integration/... -v

# Run end-to-end tests (requires git)
test-e2e:
	@echo "Running end-to-end tests..."
	@go test -tags=e2e ./test/e2e/... -v

# Run all tests including e2e
test-all: test-unit test-integration test-e2e

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
	@$(MAKE) -C hooks clean
	@echo "Clean complete"

# Install the binary to GOPATH/bin
install: build
	@echo "Installing Alpine to GOPATH/bin..."
	@go install ./cmd/alpine
	@echo "Installation complete"

# Install to /usr/local/bin with proper code signing (macOS)
install-system: build
	@echo "Installing Alpine to /usr/local/bin..."
	@echo "Building optimized binary..."
	@go build -ldflags="-s -w" -o build/alpine cmd/alpine/main.go
	@echo "Signing binary (macOS ad-hoc)..."
	@codesign -s - build/alpine
	@echo "Copying to /usr/local/bin (requires sudo)..."
	@sudo cp build/alpine /usr/local/bin/alpine
	@echo "Setting permissions..."
	@sudo chmod 755 /usr/local/bin/alpine
	@echo "Removing quarantine attributes..."
	@sudo xattr -cr /usr/local/bin/alpine
	@echo "Installation complete: /usr/local/bin/alpine"

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
	@echo "Running Alpine..."
	@./build/alpine

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
	@echo "Alpine - CLI orchestrator for Claude Code"
	@echo ""
	@echo "Available targets:"
	@echo "  make build              - Build the Alpine binary to ./build/"
	@echo "  make test               - Run all tests (unit + integration)"
	@echo "  make test-unit          - Run unit tests only (fast)"
	@echo "  make test-integration   - Run integration tests"
	@echo "  make test-e2e           - Run end-to-end tests (requires git)"
	@echo "  make test-all           - Run all tests including e2e"
	@echo "  make test-coverage      - Run tests with coverage report"
	@echo "  make clean              - Remove build artifacts"
	@echo "  make install            - Install Alpine to GOPATH/bin"
	@echo "  make install-system     - Install to /usr/local/bin with code signing (macOS)"
	@echo "  make lint               - Run golangci-lint"
	@echo "  make fmt                - Format code with go fmt"
	@echo "  make run                - Build and run Alpine"
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
	@echo "  ALPINE_INTEGRATION_TESTS=true - Enable all integration tests"