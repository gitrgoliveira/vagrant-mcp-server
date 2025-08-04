// Copyright Ricardo Oliveira 2025.
// SPDX-License-Identifier: MPL-2.0

package mcp

import (
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Type aliases for MCP-Go types for public API stability

// Tool is an alias for mcp.Tool
type Tool = mcp.Tool

// Resource is an alias for mcp.Resource
type Resource = mcp.Resource

// MCPServer is an alias for server.MCPServer
type MCPServer = server.MCPServer

// CallToolRequest is an alias for mcp.CallToolRequest
type CallToolRequest = mcp.CallToolRequest

// CallToolResult is an alias for mcp.CallToolResult
type CallToolResult = mcp.CallToolResult

// ReadResourceRequest is an alias for mcp.ReadResourceRequest
type ReadResourceRequest = mcp.ReadResourceRequest

// ResourceContents is an alias for mcp.ResourceContents
type ResourceContents = mcp.ResourceContents

// TextResourceContents is an alias for mcp.TextResourceContents
type TextResourceContents = mcp.TextResourceContents
