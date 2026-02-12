VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"
BINARY := bin/alpine

.PHONY: build test lint install clean

build:
	go build $(LDFLAGS) -o $(BINARY) ./cmd/alpine

test:
	go test ./...

lint:
	go vet ./...
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found, skipping (install: https://golangci-lint.run/usage/install/)"; \
	fi

install:
	go install $(LDFLAGS) ./cmd/alpine

clean:
	rm -rf bin/
