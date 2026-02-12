VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"
BINARY := bin/alpine

.PHONY: build test test-integration test-coverage lint install clean setup

build: setup
	go build $(LDFLAGS) -o $(BINARY) ./cmd/alpine

test:
	go test -coverprofile=coverage.out ./cmd/alpine/
	@go tool cover -func=coverage.out | tail -1

test-integration:
	go test -tags=integration -v ./cmd/alpine/

test-coverage:
	go test -coverprofile=coverage.out ./cmd/alpine/
	go tool cover -html=coverage.out

lint:
	go vet ./...
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found, skipping (install: https://golangci-lint.run/usage/install/)"; \
	fi

install:
	go install $(LDFLAGS) ./cmd/alpine

setup:
	git config core.hooksPath .githooks

clean:
	rm -rf bin/ coverage.out
