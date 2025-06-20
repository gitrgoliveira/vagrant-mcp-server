#!/bin/bash
# Simple script to run tests for the Vagrant MCP Server Go implementation

echo "Running tests for Vagrant MCP Server..."
go test -v ./...
