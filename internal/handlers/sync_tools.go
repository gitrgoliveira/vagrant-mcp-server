package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rs/zerolog/log"
	"github.com/vagrant-mcp/server/internal/exec"
	"github.com/vagrant-mcp/server/internal/vm"
)

// RegisterSyncTools registers all sync-related tools with the MCP server
func RegisterSyncTools(srv *server.MCPServer, vmManager exec.VMManager, syncEngine exec.SyncEngine) {
	// Configure sync tool
	configureSyncTool := mcp.NewTool("configure_sync",
		mcp.WithDescription("Configure sync method and options"),
		mcp.WithString("vm_name", mcp.Required(), mcp.Description("Name of the development VM")),
		mcp.WithString("sync_type", mcp.Required(), mcp.Description("Type of sync to use (rsync, nfs, etc.)")),
		mcp.WithString("host_path", mcp.Description("Host path to sync")),
		mcp.WithString("guest_path", mcp.Description("Guest path to sync")),
		mcp.WithArray("exclude_patterns", mcp.Description("Patterns to exclude from sync")),
	)

	srv.AddTool(configureSyncTool, handleConfigureSync(vmManager, syncEngine))

	// Sync to VM tool
	syncToVMTool := mcp.NewTool("sync_to_vm",
		mcp.WithDescription("Sync files from host to VM"),
		mcp.WithString("vm_name", mcp.Required(), mcp.Description("Name of the development VM")),
	)

	srv.AddTool(syncToVMTool, handleSyncToVM(vmManager, syncEngine))

	// Sync from VM tool
	syncFromVMTool := mcp.NewTool("sync_from_vm",
		mcp.WithDescription("Sync files from VM to host"),
		mcp.WithString("vm_name", mcp.Required(), mcp.Description("Name of the development VM")),
	)

	srv.AddTool(syncFromVMTool, handleSyncFromVM(vmManager, syncEngine))

	// Sync status tool
	syncStatusTool := mcp.NewTool("sync_status",
		mcp.WithDescription("Get sync status information"),
		mcp.WithString("vm_name", mcp.Required(), mcp.Description("Name of the development VM")),
	)

	srv.AddTool(syncStatusTool, handleSyncStatus(vmManager, syncEngine))

	// Resolve sync conflicts tool
	resolveSyncConflictTool := mcp.NewTool("resolve_sync_conflicts",
		mcp.WithDescription("Handle sync conflicts interactively"),
		mcp.WithString("vm_name", mcp.Required(), mcp.Description("Name of the development VM")),
		mcp.WithString("path", mcp.Required(), mcp.Description("Path of the conflicted file")),
		mcp.WithString("resolution", mcp.Required(),
			mcp.Description("Resolution method: 'use_host', 'use_vm', 'merge', 'keep_both'")),
	)

	srv.AddTool(resolveSyncConflictTool, handleResolveSyncConflict(vmManager, syncEngine))

	// Semantic search tool
	semanticSearchTool := mcp.NewTool("search_code",
		mcp.WithDescription("Search code semantically in the VM"),
		mcp.WithString("vm_name", mcp.Required(), mcp.Description("Name of the development VM")),
		mcp.WithString("query", mcp.Required(), mcp.Description("Search query")),
		mcp.WithString("search_type", mcp.Description("Type of search: 'semantic', 'exact', or 'fuzzy'"),
			mcp.DefaultString("semantic")),
		mcp.WithNumber("max_results", mcp.Description("Maximum number of results to return"),
			mcp.DefaultNumber(20)),
		mcp.WithBoolean("case_sensitive", mcp.Description("Whether the search is case sensitive")),
	)

	srv.AddTool(semanticSearchTool, handleSearchCode(vmManager, syncEngine))

	log.Info().Msg("Sync tools registered")
}

// handleConfigureSync handles the configure_sync tool
func handleConfigureSync(manager exec.VMManager, syncEngine exec.SyncEngine) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		vmName, err := request.RequireString("vm_name")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Missing or invalid 'vm_name' parameter: %v", err)), nil
		}

		syncType, err := request.RequireString("sync_type")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Missing or invalid 'sync_type' parameter: %v", err)), nil
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
		state, err := manager.GetVMState(vmName)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("VM '%s' does not exist: %v", vmName, err)), nil
		}

		// Get VM config
		config, err := manager.GetVMConfig(vmName)
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
		if err := manager.UpdateVMConfig(vmName, config); err != nil {
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
func handleSyncToVM(manager exec.VMManager, syncEngine exec.SyncEngine) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		vmName, err := request.RequireString("vm_name")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Missing or invalid 'vm_name' parameter: %v", err)), nil
		}

		// Check VM state
		state, err := manager.GetVMState(vmName)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("VM '%s' does not exist: %v", vmName, err)), nil
		}

		if state != vm.Running {
			return mcp.NewToolResultError(fmt.Sprintf("VM '%s' is not running (current state: %s)", vmName, state)), nil
		}

		// Perform sync to VM
		result, err := syncEngine.SyncToVM(vmName, "")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Sync to VM failed: %v", err)), nil
		}

		// Format the response
		response := map[string]interface{}{
			"status":         "success",
			"vm_name":        vmName,
			"synced_files":   result.SyncedFiles,
			"sync_time_ms":   result.SyncTimeMs,
			"file_count":     len(result.SyncedFiles),
			"last_sync_time": time.Now().Format(time.RFC3339),
		}

		// Convert to JSON
		jsonData, err := json.Marshal(response)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal result: %v", err)), nil
		}

		return mcp.NewToolResultText(string(jsonData)), nil
	}
}

// handleSyncFromVM handles the sync_from_vm tool
func handleSyncFromVM(manager exec.VMManager, syncEngine exec.SyncEngine) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		vmName, err := request.RequireString("vm_name")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Missing or invalid 'vm_name' parameter: %v", err)), nil
		}

		// Check VM state
		state, err := manager.GetVMState(vmName)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("VM '%s' does not exist: %v", vmName, err)), nil
		}

		if state != vm.Running {
			return mcp.NewToolResultError(fmt.Sprintf("VM '%s' is not running (current state: %s)", vmName, state)), nil
		}

		// Perform sync from VM
		result, err := syncEngine.SyncFromVM(vmName, "")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Sync from VM failed: %v", err)), nil
		}

		// Format the response
		response := map[string]interface{}{
			"status":         "success",
			"vm_name":        vmName,
			"synced_files":   result.SyncedFiles,
			"sync_time_ms":   result.SyncTimeMs,
			"file_count":     len(result.SyncedFiles),
			"last_sync_time": time.Now().Format(time.RFC3339),
		}

		// Convert to JSON
		jsonData, err := json.Marshal(response)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal result: %v", err)), nil
		}

		return mcp.NewToolResultText(string(jsonData)), nil
	}
}

// handleSyncStatus handles the sync_status tool
func handleSyncStatus(manager exec.VMManager, syncEngine exec.SyncEngine) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		vmName, err := request.RequireString("vm_name")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Missing or invalid 'vm_name' parameter: %v", err)), nil
		}

		// Check VM state
		state, err := manager.GetVMState(vmName)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("VM '%s' does not exist: %v", vmName, err)), nil
		}

		// Get sync status
		status, err := syncEngine.GetSyncStatus(vmName)
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
func handleResolveSyncConflict(manager exec.VMManager, syncEngine exec.SyncEngine) server.ToolHandlerFunc {
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
		state, err := manager.GetVMState(vmName)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("VM '%s' does not exist: %v", vmName, err)), nil
		}

		if state != vm.Running {
			return mcp.NewToolResultError(fmt.Sprintf("VM '%s' is not running (current state: %s)", vmName, state)), nil
		}

		// Resolve conflict
		err = syncEngine.ResolveSyncConflict(vmName, path, resolution)
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
func handleSearchCode(manager exec.VMManager, syncEngine exec.SyncEngine) server.ToolHandlerFunc {
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
		state, err := manager.GetVMState(vmName)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("VM '%s' does not exist: %v", vmName, err)), nil
		}

		if state != vm.Running {
			return mcp.NewToolResultError(fmt.Sprintf("VM '%s' is not running (current state: %s)", vmName, state)), nil
		}

		// Perform search based on type
		var results interface{}
		var searchErr error

		switch searchType {
		case "semantic":
			results, searchErr = syncEngine.SemanticSearch(vmName, query, maxResults)
		case "exact":
			results, searchErr = syncEngine.ExactSearch(vmName, query, caseSensitive, maxResults)
		case "fuzzy":
			results, searchErr = syncEngine.FuzzySearch(vmName, query, maxResults)
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
