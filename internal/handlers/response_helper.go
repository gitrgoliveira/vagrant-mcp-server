package handlers

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

// ResponseHelper provides common response formatting functionality
type ResponseHelper struct{}

// NewResponseHelper creates a new response helper
func NewResponseHelper() *ResponseHelper {
	return &ResponseHelper{}
}

// MarshalSuccessResponse marshals a response map to JSON and returns a successful MCP result
func (h *ResponseHelper) MarshalSuccessResponse(response map[string]interface{}) (*mcp.CallToolResult, error) {
	jsonData, err := json.Marshal(response)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal result: %v", err)), nil
	}
	return mcp.NewToolResultText(string(jsonData)), nil
}

// CreateSyncResponse creates a standardized sync response
func (h *ResponseHelper) CreateSyncResponse(vmName string, syncedFiles []string, syncTimeMs int, operation string) map[string]interface{} {
	return map[string]interface{}{
		"status":       "success",
		"operation":    operation,
		"vm_name":      vmName,
		"synced_files": syncedFiles,
		"sync_time_ms": syncTimeMs,
		"file_count":   len(syncedFiles),
		"timestamp":    getCurrentTimestamp(),
	}
}

// getCurrentTimestamp returns the current timestamp in RFC3339 format
func getCurrentTimestamp() string {
	return time.Now().Format(time.RFC3339)
}
