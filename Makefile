CMD_DIR := ./cmd/setup-sprite-opencode
BINARY := setup-sprite-opencode

ifeq ($(origin GOBIN), undefined)
GOBIN := $(shell go env GOBIN)
endif
ifeq ($(GOBIN),)
GOBIN := $(shell go env GOPATH)/bin
endif

.PHONY: install

install:
	@mkdir -p "$(GOBIN)"
	go build -o "$(GOBIN)/$(BINARY)" "$(CMD_DIR)"
	@printf "installed $(BINARY) to %s\n" "$(GOBIN)"
