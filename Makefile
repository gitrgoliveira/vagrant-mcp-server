# Makefile for Vagrant MCP Server

APP_NAME = vagrant-mcp-server
PKG = ./...

# Build-time variables
VERSION ?= $(shell git describe --tags --always --dirty)
GIT_COMMIT ?= $(shell git rev-parse HEAD)
BUILD_TIME ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GO_VERSION ?= $(shell go version | cut -d' ' -f3)

# Linker flags for version injection
LDFLAGS = -ldflags "-X main.Version=$(VERSION) -X main.GitCommit=$(GIT_COMMIT) -X main.BuildTime=$(BUILD_TIME) -X main.GoVersion=$(GO_VERSION) -w -s"

.PHONY: all test test-integration test-vm-start lint fmt sec build clean check-vagrant help-test

all: fmt lint sec test build

fmt:
	gofmt -s -w .

lint:
	golangci-lint run

sec:
	@echo "Checking for high severity issues only..."
	@gosec -severity high $(PKG)

# Check if Vagrant is installed
check-vagrant:
	@echo "Checking if Vagrant CLI is installed..."
	@vagrant --version >/dev/null 2>&1 || (echo "Error: Vagrant CLI is not installed or not in your PATH. Please install Vagrant: https://www.vagrantup.com/downloads" && exit 1)
	@echo "✓ Vagrant CLI is installed."

# Run fast unit tests (no VM creation)
test: check-vagrant
	@echo "Running unit tests..."
	go test -race -cover $(PKG)

# Run integration tests (creates VMs but doesn't start them)
test-integration: check-vagrant
	@echo "Running integration tests (creates VMs for testing)..."
	TEST_LEVEL=integration go test -race -cover $(PKG)

# Run VM start tests (actually starts VMs - very slow)
test-vm-start: check-vagrant
	@echo "Running VM start tests (starts actual VMs - may take several minutes)..."
	TEST_LEVEL=vm-start go test -race -cover $(PKG)

build:
	go build $(LDFLAGS) -o bin/$(APP_NAME) ./cmd/server

clean:
	rm -rf bin/

# Show available test targets and their purposes
help-test:
	@echo "Available test targets:"
	@echo "  make test           - Fast unit tests (no VMs created)"
	@echo "  make test-integration - Integration tests (creates but doesn't start VMs)"
	@echo "  make test-vm-start  - Full VM tests (starts actual VMs - very slow)" 
	@echo ""
	@echo "Environment variables:"
	@echo "  TEST_LEVEL=integration - Enable integration tests (creates VMs)"
	@echo "  TEST_LEVEL=vm-start    - Enable VM start tests (starts VMs)"

# Install tools if not present
tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest

RELEASE_VERSION ?= $(shell git describe --tags --always --dirty)
RELEASE_DIR = dist/$(APP_NAME)-$(RELEASE_VERSION)

release: clean fmt lint sec test
	@echo "Building release version $(RELEASE_VERSION)"
	mkdir -p $(RELEASE_DIR)
	GOOS=linux   GOARCH=amd64 go build $(LDFLAGS) -o $(RELEASE_DIR)/$(APP_NAME)-linux-amd64 ./cmd/server
	GOOS=linux   GOARCH=arm64 go build $(LDFLAGS) -o $(RELEASE_DIR)/$(APP_NAME)-linux-arm64 ./cmd/server
	GOOS=darwin  GOARCH=amd64 go build $(LDFLAGS) -o $(RELEASE_DIR)/$(APP_NAME)-darwin-amd64 ./cmd/server
	GOOS=darwin  GOARCH=arm64 go build $(LDFLAGS) -o $(RELEASE_DIR)/$(APP_NAME)-darwin-arm64 ./cmd/server
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(RELEASE_DIR)/$(APP_NAME)-windows-amd64.exe ./cmd/server
	cp -r README.md docs/ $(RELEASE_DIR)/ 2>/dev/null || true
	@echo "Generating checksums..."
	cd $(RELEASE_DIR) && shasum -a 256 $(APP_NAME)-* > checksums.txt
	cd dist && tar -czvf $(APP_NAME)-$(RELEASE_VERSION).tar.gz $(APP_NAME)-$(RELEASE_VERSION)
	@echo "Release created at dist/$(APP_NAME)-$(RELEASE_VERSION).tar.gz"
