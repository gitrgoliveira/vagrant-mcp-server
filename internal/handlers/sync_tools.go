// Copyright Ricardo Oliveira 2025.
// SPDX-License-Identifier: MPL-2.0

package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rs/zerolog/log"
	"github.com/vagrant-mcp/server/internal/core"
	"github.com/vagrant-mcp/server/pkg/mcp"
)

// RegisterSyncTools registers all sync-related tools with the MCP server
func RegisterSyncTools(srv *server.MCPServer, syncEngine core.SyncEngine, vmManager core.VMManager) {
	// Configure sync tool
	configureSyncTool := mcpgo.NewTool("configure_sync",
		mcpgo.WithDescription("Configure sync method and options"),
		mcpgo.WithString("vm_name", mcpgo.Required(), mcpgo.Description("Name of the development VM")),
		mcpgo.WithString("sync_type", mcpgo.Required(), mcpgo.Description("Type of sync to use (rsync, nfs, etc.)")),
		mcpgo.WithString("host_path", mcpgo.Description("Host path to sync")),
		mcpgo.WithString("guest_path", mcpgo.Description("Guest path to sync")),
		mcpgo.WithArray("exclude_patterns",
			mcpgo.Description("Patterns to exclude from sync"),
			mcpgo.Items(map[string]any{"type": "string"})),
	)

	srv.AddTool(configureSyncTool, handleConfigureSync(vmManager, syncEngine))

	// Sync to VM tool
	syncToVMTool := mcpgo.NewTool("sync_to_vm",
		mcpgo.WithDescription("Sync files from host to VM"),
		mcpgo.WithString("vm_name", mcpgo.Required(), mcpgo.Description("Name of the development VM")),
	)

	srv.AddTool(syncToVMTool, handleSyncToVM(syncEngine, vmManager))

	// Sync from VM tool
	syncFromVMTool := mcpgo.NewTool("sync_from_vm",
		mcpgo.WithDescription("Sync files from VM to host"),
		mcpgo.WithString("vm_name", mcpgo.Required(), mcpgo.Description("Name of the development VM")),
	)

	srv.AddTool(syncFromVMTool, handleSyncFromVM(syncEngine, vmManager))

	// Upload to VM tool
	uploadToVMTool := mcpgo.NewTool("upload_to_vm",
		mcpgo.WithDescription("Upload files from host to VM"),
		mcpgo.WithString("vm_name", mcpgo.Required(), mcpgo.Description("Name of the development VM")),
		mcpgo.WithString("source", mcpgo.Required(), mcpgo.Description("Source file or directory path on host")),
		mcpgo.WithString("destination", mcpgo.Required(), mcpgo.Description("Destination path on VM")),
		mcpgo.WithBoolean("compress", mcpgo.Description("Whether to compress the file before upload")),
		mcpgo.WithString("compression_type", mcpgo.Description("Compression type to use (tgz, zip)")),
	)

	srv.AddTool(uploadToVMTool, handleUploadToVM(vmManager))

	// Sync status tool
	syncStatusTool := mcpgo.NewTool("sync_status",
		mcpgo.WithDescription("Get sync status information"),
		mcpgo.WithString("vm_name", mcpgo.Required(), mcpgo.Description("Name of the development VM")),
	)

	srv.AddTool(syncStatusTool, handleSyncStatus(syncEngine, vmManager))

	// Resolve sync conflicts tool
	resolveSyncConflictTool := mcpgo.NewTool("resolve_sync_conflicts",
		mcpgo.WithDescription("Handle sync conflicts interactively"),
		mcpgo.WithString("vm_name", mcpgo.Required(), mcpgo.Description("Name of the development VM")),
		mcpgo.WithString("path", mcpgo.Required(), mcpgo.Description("Path of the conflicted file")),
		mcpgo.WithString("resolution", mcpgo.Required(),
			mcpgo.Description("Resolution method: 'use_host', 'use_vm', 'merge', 'keep_both'")),
	)

	srv.AddTool(resolveSyncConflictTool, handleResolveSyncConflict(vmManager, syncEngine))

	// Semantic search tool
	semanticSearchTool := mcpgo.NewTool("search_code",
		mcpgo.WithDescription("Search code semantically in the VM"),
		mcpgo.WithString("vm_name", mcpgo.Required(), mcpgo.Description("Name of the development VM")),
		mcpgo.WithString("query", mcpgo.Required(), mcpgo.Description("Search query")),
		mcpgo.WithString("search_type", mcpgo.Description("Type of search: 'semantic', 'exact', or 'fuzzy'"),
			mcpgo.DefaultString("semantic")),
		mcpgo.WithNumber("max_results", mcpgo.Description("Maximum number of results to return"),
			mcpgo.DefaultNumber(20)),
		mcpgo.WithBoolean("case_sensitive", mcpgo.Description("Whether the search is case sensitive")),
	)

	srv.AddTool(semanticSearchTool, handleSearchCode(vmManager, syncEngine))

	log.Info().Msg("Sync tools registered")
}

// handleConfigureSync handles the configure_sync tool
func handleConfigureSync(manager core.VMManager, syncEngine core.SyncEngine) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		vmName, err := request.RequireString("vm_name")
		if err != nil {
			return mcpgo.NewToolResultError(fmt.Sprintf("Missing or invalid 'vm_name' parameter: %v", err)), nil
		}

		syncType, err := request.RequireString("sync_type")
		if err != nil {
			return mcpgo.NewToolResultError(fmt.Sprintf("Missing or invalid 'sync_type' parameter: %v", err)), nil
		}

		hostPath := request.GetString("host_path", "")
		guestPath := request.GetString("guest_path", "")

		// Get exclude patterns
		var excludePatterns []string
		if patterns, ok := request.GetArguments()["exclude_patterns"].([]interface{}); ok {
			for _, p := range patterns {
				if pattern, ok := p.(string); ok {
					excludePatterns = append(excludePatterns, pattern)
				}
			}
		}

		// Check VM state
		state, err := manager.GetVMState(ctx, vmName)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("VM '%s' does not exist: %v", vmName, err)), nil
		}

		// Get VM config
		config, err := manager.GetVMConfig(ctx, vmName)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get VM config: %v", err)), nil
		}

		// Update sync config
		config.SyncType = syncType
		if hostPath != "" {
			config.HostPath = hostPath
		}
		if guestPath != "" {
			config.GuestPath = guestPath
		}
		if len(excludePatterns) > 0 {
			config.SyncExcludePatterns = excludePatterns
		}

		// Update config file
		if err := manager.UpdateVMConfig(ctx, vmName, config); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to update VM config: %v", err)), nil
		}

		// Return result using MCP-Go's helper
		result := map[string]interface{}{
			"vm_name":          vmName,
			"state":            state,
			"sync_type":        syncType,
			"host_path":        config.HostPath,
			"guest_path":       config.GuestPath,
			"exclude_patterns": config.SyncExcludePatterns,
		}

		jsonData, err := json.Marshal(result)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal result: %v", err)), nil
		}

		return mcp.NewToolResultText(string(jsonData)), nil
	}
}

// handleSyncToVM handles the sync_to_vm tool
func handleSyncToVM(syncEngine core.SyncEngine, vmManager core.VMManager) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Use validation helper
		validator := NewValidationHelper()
		responseHelper := NewResponseHelper()

		// Validate required parameters
		vmName, errorResult, err := validator.ValidateRequiredString(request, "vm_name")
		if err != nil {
			return errorResult, nil
		}

		// Validate VM is running
		if errorResult, err := validator.ValidateVMRunning(ctx, vmManager, vmName); err != nil {
			return errorResult, nil
		}

		// Perform sync to VM
		result, err := syncEngine.SyncToVM(ctx, vmName, "")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Sync to VM failed: %v", err)), nil
		}

		// Create standardized response using helper
		response := responseHelper.CreateSyncResponse(vmName, result.SyncedFiles, result.SyncTimeMs, "sync_to_vm")
		return responseHelper.MarshalSuccessResponse(response)
	}
}

// handleSyncFromVM handles the sync_from_vm tool
func handleSyncFromVM(syncEngine core.SyncEngine, vmManager core.VMManager) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Use validation helper
		validator := NewValidationHelper()
		responseHelper := NewResponseHelper()

		// Validate required parameters
		vmName, errorResult, err := validator.ValidateRequiredString(request, "vm_name")
		if err != nil {
			return errorResult, nil
		}

		// Validate VM is running
		if errorResult, err := validator.ValidateVMRunning(ctx, vmManager, vmName); err != nil {
			return errorResult, nil
		}

		// Perform sync from VM
		result, err := syncEngine.SyncFromVM(ctx, vmName, "")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Sync from VM failed: %v", err)), nil
		}

		// Create standardized response using helper
		response := responseHelper.CreateSyncResponse(vmName, result.SyncedFiles, result.SyncTimeMs, "sync_from_vm")
		return responseHelper.MarshalSuccessResponse(response)
	}
}

// handleSyncStatus handles the sync_status tool
func handleSyncStatus(syncEngine core.SyncEngine, vmManager core.VMManager) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		vmName, err := request.RequireString("vm_name")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("missing or invalid 'vm_name' parameter: %v", err)), nil
		}

		// Check VM state
		state, err := vmManager.GetVMState(ctx, vmName)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("VM '%s' does not exist: %v", vmName, err)), nil
		}

		// Get sync status
		status, err := syncEngine.GetSyncStatus(ctx, vmName)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get sync status: %v", err)), nil
		}

		// Return status using MCP-Go's JSON result
		result := map[string]interface{}{
			"vm_name":            vmName,
			"vm_state":           state,
			"sync_status":        status,
			"last_sync_time":     status.LastSyncTime,
			"in_progress":        status.InProgress,
			"conflicts":          status.Conflicts,
			"synchronized_files": status.SynchronizedFiles,
			"total_syncs":        status.TotalSyncs,
			"total_files_synced": status.TotalFilesSynced,
			"total_sync_time_ms": status.TotalSyncTimeMs,
		}

		jsonData, err := json.Marshal(result)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal result: %v", err)), nil
		}
		return mcp.NewToolResultText(string(jsonData)), nil
	}
}

// handleResolveSyncConflict handles the resolve_sync_conflicts tool
func handleResolveSyncConflict(manager core.VMManager, syncEngine core.SyncEngine) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Extract parameters
		vmName, err := request.RequireString("vm_name")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Missing or invalid 'vm_name' parameter: %v", err)), nil
		}

		path, err := request.RequireString("path")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Missing or invalid 'path' parameter: %v", err)), nil
		}

		resolution, err := request.RequireString("resolution")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Missing or invalid 'resolution' parameter: %v", err)), nil
		}

		// Check VM state
		state, err := manager.GetVMState(ctx, vmName)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("VM '%s' does not exist: %v", vmName, err)), nil
		}

		if state != core.Running {
			return mcp.NewToolResultError(fmt.Sprintf("VM '%s' is not running (current state: %s)", vmName, state)), nil
		}

		// Resolve conflict
		err = syncEngine.ResolveSyncConflict(ctx, vmName, path, resolution)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to resolve conflict: %v", err)), nil
		}

		// Return success response
		result := map[string]interface{}{
			"status":     "success",
			"message":    fmt.Sprintf("Conflict for path '%s' resolved using '%s' strategy", path, resolution),
			"vm_name":    vmName,
			"path":       path,
			"resolution": resolution,
		}

		// Convert to JSON
		jsonData, err := json.Marshal(result)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal result: %v", err)), nil
		}

		return mcp.NewToolResultText(string(jsonData)), nil
	}
}

// handleSearchCode handles the search_code tool
func handleSearchCode(manager core.VMManager, syncEngine core.SyncEngine) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Extract parameters
		vmName, err := request.RequireString("vm_name")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Missing or invalid 'vm_name' parameter: %v", err)), nil
		}

		query, err := request.RequireString("query")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Missing or invalid 'query' parameter: %v", err)), nil
		}

		searchType := request.GetString("search_type", "semantic")
		maxResultsFloat := request.GetFloat("max_results", 20.0)
		maxResults := int(maxResultsFloat)

		// Extract case_sensitive parameter if it exists
		var caseSensitive bool
		if val, ok := request.GetArguments()["case_sensitive"]; ok {
			if boolVal, ok := val.(bool); ok {
				caseSensitive = boolVal
			}
		}

		// Check VM state
		state, err := manager.GetVMState(ctx, vmName)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("VM '%s' does not exist: %v", vmName, err)), nil
		}

		if state != core.Running {
			return mcp.NewToolResultError(fmt.Sprintf("VM '%s' is not running (current state: %s)", vmName, state)), nil
		}

		// Perform search based on type
		var results interface{}
		var searchErr error

		switch searchType {
		case "semantic":
			results, searchErr = syncEngine.SemanticSearch(ctx, vmName, query, maxResults)
		case "exact":
			results, searchErr = syncEngine.ExactSearch(ctx, vmName, query, caseSensitive, maxResults)
		case "fuzzy":
			results, searchErr = syncEngine.FuzzySearch(ctx, vmName, query, maxResults)
		default:
			return mcp.NewToolResultError(fmt.Sprintf("Invalid search type: %s (must be 'semantic', 'exact', or 'fuzzy')", searchType)), nil
		}

		if searchErr != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Search failed: %v", searchErr)), nil
		}

		// Format the response
		response := map[string]interface{}{
			"status":      "success",
			"vm_name":     vmName,
			"query":       query,
			"search_type": searchType,
			"results":     results,
			"total":       len(results.([]interface{})),
		}

		// Convert to JSON
		jsonData, err := json.Marshal(response)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal results: %v", err)), nil
		}

		return mcp.NewToolResultText(string(jsonData)), nil
	}
}

// handleUploadToVM handles the upload_to_vm tool
func handleUploadToVM(manager core.VMManager) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		vmName, err := request.RequireString("vm_name")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Missing or invalid 'vm_name' parameter: %v", err)), nil
		}

		source, err := request.RequireString("source")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Missing or invalid 'source' parameter: %v", err)), nil
		}

		destination, err := request.RequireString("destination")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Missing or invalid 'destination' parameter: %v", err)), nil
		}

		// Optional parameters
		compress := request.GetBool("compress", false)
		compressionType := request.GetString("compression_type", "") // Default will be decided by vagrant

		// Check VM state
		state, err := manager.GetVMState(ctx, vmName)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("VM '%s' does not exist: %v", vmName, err)), nil
		}

		if state != core.Running {
			return mcp.NewToolResultError(fmt.Sprintf("VM '%s' is not running (current state: %s)", vmName, state)), nil
		}

		// Upload file to VM
		err = manager.UploadToVM(ctx, vmName, source, destination, compress, compressionType)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Upload to VM failed: %v", err)), nil
		}

		// Format the response
		response := map[string]interface{}{
			"status":      "success",
			"vm_name":     vmName,
			"source":      source,
			"destination": destination,
			"upload_time": time.Now().Format(time.RFC3339),
		}

		// Convert to JSON
		jsonData, err := json.Marshal(response)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal result: %v", err)), nil
		}

		return mcp.NewToolResultText(string(jsonData)), nil
	}
}
