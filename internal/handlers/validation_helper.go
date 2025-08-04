package handlers

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/vagrant-mcp/server/internal/core"
)

// ValidationHelper provides common validation functionality
type ValidationHelper struct{}

// NewValidationHelper creates a new validation helper
func NewValidationHelper() *ValidationHelper {
	return &ValidationHelper{}
}

// ValidateVMExists validates that a VM exists
func (v *ValidationHelper) ValidateVMExists(ctx context.Context, vmManager core.VMManager, vmName string) (*mcp.CallToolResult, error) {
	_, err := vmManager.GetVMState(ctx, vmName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("VM '%s' does not exist: %v", vmName, err)), fmt.Errorf("VM validation failed: %w", err)
	}
	return nil, nil
}

// ValidateVMRunning validates that a VM exists and is running
func (v *ValidationHelper) ValidateVMRunning(ctx context.Context, vmManager core.VMManager, vmName string) (*mcp.CallToolResult, error) {
	state, err := vmManager.GetVMState(ctx, vmName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("VM '%s' does not exist: %v", vmName, err)), fmt.Errorf("VM validation failed: %w", err)
	}

	if state != core.Running {
		return mcp.NewToolResultError(fmt.Sprintf("VM '%s' is not running (current state: %s)", vmName, state)), fmt.Errorf("VM not running: %s", state)
	}

	return nil, nil
}

// ValidateRequiredString extracts and validates a required string parameter
func (v *ValidationHelper) ValidateRequiredString(request mcp.CallToolRequest, paramName string) (string, *mcp.CallToolResult, error) {
	value, err := request.RequireString(paramName)
	if err != nil {
		errorResult := mcp.NewToolResultError(fmt.Sprintf("Missing or invalid '%s' parameter: %v", paramName, err))
		return "", errorResult, fmt.Errorf("parameter validation failed: %w", err)
	}
	return value, nil, nil
}

// ValidateOptionalString extracts an optional string parameter with default
func (v *ValidationHelper) ValidateOptionalString(request mcp.CallToolRequest, paramName, defaultValue string) string {
	return request.GetString(paramName, defaultValue)
}

// ValidateOptionalInt extracts an optional int parameter with default
func (v *ValidationHelper) ValidateOptionalInt(request mcp.CallToolRequest, paramName string, defaultValue int) int {
	floatValue := request.GetFloat(paramName, float64(defaultValue))
	return int(floatValue)
}

// ValidateOptionalBool extracts an optional bool parameter with default
func (v *ValidationHelper) ValidateOptionalBool(request mcp.CallToolRequest, paramName string, defaultValue bool) bool {
	return request.GetBool(paramName, defaultValue)
}
