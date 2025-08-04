// Copyright Ricardo Oliveira 2025.
// SPDX-License-Identifier: MPL-2.0

// Package handlers provides unified error handling utilities
package handlers

import (
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/vagrant-mcp/server/internal/errors"
)

// UnifiedErrorHelper provides centralized error handling for MCP responses
type UnifiedErrorHelper struct{}

// NewUnifiedErrorHelper creates a new unified error helper
func NewUnifiedErrorHelper() *UnifiedErrorHelper {
	return &UnifiedErrorHelper{}
}

// ValidationError creates a standardized validation error response
func (h *UnifiedErrorHelper) ValidationError(field string, err error) *mcp.CallToolResult {
	msg := fmt.Sprintf("validation failed for '%s': %v", field, err)
	return mcp.NewToolResultError(msg)
}

// OperationError creates a standardized operation error response
func (h *UnifiedErrorHelper) OperationError(operation string, err error) *mcp.CallToolResult {
	msg := fmt.Sprintf("%s failed: %v", operation, err)
	return mcp.NewToolResultError(msg)
}

// VMStateError creates a standardized VM state error response
func (h *UnifiedErrorHelper) VMStateError(vmName string, expectedState string) *mcp.CallToolResult {
	msg := fmt.Sprintf("VM '%s' is not in %s state", vmName, expectedState)
	return mcp.NewToolResultError(msg)
}

// VMNotFoundError creates a standardized VM not found error response
func (h *UnifiedErrorHelper) VMNotFoundError(vmName string) *mcp.CallToolResult {
	msg := fmt.Sprintf("VM '%s' not found", vmName)
	return mcp.NewToolResultError(msg)
}

// InvalidParameterError creates a standardized invalid parameter error response
func (h *UnifiedErrorHelper) InvalidParameterError(paramName string, value interface{}, reason string) *mcp.CallToolResult {
	msg := fmt.Sprintf("invalid parameter '%s' = %v: %s", paramName, value, reason)
	return mcp.NewToolResultError(msg)
}

// RequiredParameterError creates a standardized required parameter error response
func (h *UnifiedErrorHelper) RequiredParameterError(paramName string) *mcp.CallToolResult {
	msg := fmt.Sprintf("required parameter '%s' is missing", paramName)
	return mcp.NewToolResultError(msg)
}

// SyncError creates a standardized sync operation error response
func (h *UnifiedErrorHelper) SyncError(operation string, vmName string, err error) *mcp.CallToolResult {
	msg := fmt.Sprintf("sync %s failed for VM '%s': %v", operation, vmName, err)
	return mcp.NewToolResultError(msg)
}

// ExecutionError creates a standardized command execution error response
func (h *UnifiedErrorHelper) ExecutionError(command string, vmName string, err error) *mcp.CallToolResult {
	msg := fmt.Sprintf("command execution failed in VM '%s': %v", vmName, err)
	return mcp.NewToolResultError(msg)
}

// WrapAppError converts an AppError to MCP format
func (h *UnifiedErrorHelper) WrapAppError(err error) *mcp.CallToolResult {
	if appErr, ok := err.(*errors.AppError); ok {
		return mcp.NewToolResultError(appErr.Message)
	}
	return mcp.NewToolResultError(err.Error())
}

// FromError creates an appropriate error response based on error type
func (h *UnifiedErrorHelper) FromError(err error) *mcp.CallToolResult {
	switch e := err.(type) {
	case *errors.AppError:
		return h.WrapAppError(e)
	default:
		return mcp.NewToolResultError(e.Error())
	}
}

// Global error helper instance for convenience
var Global = NewUnifiedErrorHelper()
