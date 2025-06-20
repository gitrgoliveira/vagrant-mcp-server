# Makefile for Vagrant MCP Server

APP_NAME = vagrant-mcp-server
PKG = ./...

.PHONY: all test lint fmt sec build integration clean

all: fmt lint sec test build

fmt:
	gofmt -s -w .

lint:
	golangci-lint run

sec:
	gosec $(PKG)

test:
	go test -race -cover $(PKG)

integration:
	INTEGRATION_TESTS=1 go test -v ./cmd/server

build:
	go build -o bin/$(APP_NAME) ./cmd/server

clean:
	rm -rf bin/

# Install tools if not present
tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest

RELEASE_VERSION ?= $(shell git describe --tags --always --dirty)
RELEASE_DIR = dist/$(APP_NAME)-$(RELEASE_VERSION)

release: clean fmt lint sec test
	@echo "Building release version $(RELEASE_VERSION)"
	mkdir -p $(RELEASE_DIR)
	GOOS=linux   GOARCH=amd64 go build -o $(RELEASE_DIR)/$(APP_NAME)-linux-amd64 ./cmd/server
	GOOS=darwin  GOARCH=amd64 go build -o $(RELEASE_DIR)/$(APP_NAME)-darwin-amd64 ./cmd/server
	GOOS=darwin  GOARCH=arm64 go build -o $(RELEASE_DIR)/$(APP_NAME)-darwin-arm64 ./cmd/server
	GOOS=windows GOARCH=amd64 go build -o $(RELEASE_DIR)/$(APP_NAME)-windows-amd64.exe ./cmd/server
	cp -r README.md PROMPT.md $(RELEASE_DIR) 2>/dev/null || true
	cd dist && tar -czvf $(APP_NAME)-$(RELEASE_VERSION).tar.gz $(APP_NAME)-$(RELEASE_VERSION)
	@echo "Release created at dist/$(APP_NAME)-$(RELEASE_VERSION).tar.gz"
