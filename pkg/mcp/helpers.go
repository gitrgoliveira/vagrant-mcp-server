// Copyright Ricardo Oliveira 2025.
// SPDX-License-Identifier: MPL-2.0

package mcp

import (
	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// RegisterTypedTool registers a tool with a typed handler using MCP-go's NewTypedToolHandler pattern.
func RegisterTypedTool[T any](
	s *server.MCPServer,
	tool mcpgo.Tool, // not *mcpgo.Tool
	handler mcpgo.TypedToolHandlerFunc[T],
) {
	s.AddTool(tool, mcpgo.NewTypedToolHandler(handler))
}
