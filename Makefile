BINARY_NAME := mcp-kind-manager
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-s -w -X main.Version=$(VERSION)"
GO := go

.PHONY: all build test lint clean install run fmt vet

all: lint test build

build:
	$(GO) build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/mcp-kind-manager

install:
	$(GO) install $(LDFLAGS) ./cmd/mcp-kind-manager

run: build
	./bin/$(BINARY_NAME)

test:
	$(GO) test -race -coverprofile=coverage.out ./...
	$(GO) tool cover -func=coverage.out

test-verbose:
	$(GO) test -race -v ./...

lint: vet
	@which golangci-lint > /dev/null 2>&1 || echo "golangci-lint not installed, skipping"
	@which golangci-lint > /dev/null 2>&1 && golangci-lint run ./... || true

vet:
	$(GO) vet ./...

fmt:
	$(GO) fmt ./...
	gofumpt -w . 2>/dev/null || true

clean:
	rm -rf bin/ coverage.out coverage.html dist/

cover: test
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Cross-compilation targets
build-linux:
	GOOS=linux GOARCH=amd64 $(GO) build $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-amd64 ./cmd/mcp-kind-manager
	GOOS=linux GOARCH=arm64 $(GO) build $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-arm64 ./cmd/mcp-kind-manager

build-darwin:
	GOOS=darwin GOARCH=amd64 $(GO) build $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-amd64 ./cmd/mcp-kind-manager
	GOOS=darwin GOARCH=arm64 $(GO) build $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-arm64 ./cmd/mcp-kind-manager

build-windows:
	GOOS=windows GOARCH=amd64 $(GO) build $(LDFLAGS) -o bin/$(BINARY_NAME)-windows-amd64.exe ./cmd/mcp-kind-manager

build-all: build-linux build-darwin build-windows
