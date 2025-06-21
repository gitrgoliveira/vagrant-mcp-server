package mcp

import (
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// This package contains type aliases and helper functions for working with the MCP-Go library.
// It's maintained for backwards compatibility and to provide a clean public API.

// ToolHandlerFunc is an alias for server.ToolHandlerFunc
type ToolHandlerFunc = server.ToolHandlerFunc

// ResourceHandlerFunc is an alias for server.ResourceHandlerFunc
type ResourceHandlerFunc = server.ResourceHandlerFunc

// NewToolResultText is an alias for mcp.NewToolResultText
func NewToolResultText(text string) *mcp.CallToolResult {
	return mcp.NewToolResultText(text)
}

// NewToolResultError is an alias for mcp.NewToolResultError
func NewToolResultError(err string) *mcp.CallToolResult {
	return mcp.NewToolResultError(err)
}

// NewToolResultErrorf is an alias for mcp.NewToolResultErrorf
func NewToolResultErrorf(format string, args ...interface{}) *mcp.CallToolResult {
	return mcp.NewToolResultErrorf(format, args...)
}
