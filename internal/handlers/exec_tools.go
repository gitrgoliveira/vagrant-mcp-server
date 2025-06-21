package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rs/zerolog/log"
	"github.com/vagrant-mcp/server/internal/exec"
)

// RegisterExecTools registers all execution-related tools with the MCP server
func RegisterExecTools(srv *server.MCPServer, vmManager exec.VMManager, syncEngine exec.SyncEngine, executor *exec.Executor) {
	// Execute in VM tool
	execInVMTool := mcp.NewTool("exec_in_vm",
		mcp.WithDescription("Execute a command in the VM without file synchronization"),
		mcp.WithString("vm_name",
			mcp.Required(),
			mcp.Description("Name of the development VM")),
		mcp.WithString("command",
			mcp.Required(),
			mcp.Description("Command to execute")),
		mcp.WithString("working_dir",
			mcp.Description("Working directory"),
			mcp.DefaultString("/home/vagrant")),
	)

	srv.AddTool(execInVMTool, handleExecInVM(vmManager, executor))

	// Execute with sync tool
	execWithSyncTool := mcp.NewTool("exec_with_sync",
		mcp.WithDescription("Execute a command in the VM with file synchronization before and after"),
		mcp.WithString("vm_name",
			mcp.Required(),
			mcp.Description("Name of the development VM")),
		mcp.WithString("command",
			mcp.Required(),
			mcp.Description("Command to execute")),
		mcp.WithString("working_dir",
			mcp.Description("Working directory"),
			mcp.DefaultString("/home/vagrant")),
		mcp.WithBoolean("sync_before",
			mcp.Description("Sync files to VM before execution"),
			mcp.DefaultBool(true)),
		mcp.WithBoolean("sync_after",
			mcp.Description("Sync files from VM after execution"),
			mcp.DefaultBool(true)),
	)

	srv.AddTool(execWithSyncTool, handleExecWithSync(vmManager, syncEngine, executor))

	// Run background task tool
	runBackgroundTool := mcp.NewTool("run_background_task",
		mcp.WithDescription("Run a command in the VM as a background task"),
		mcp.WithString("vm_name",
			mcp.Required(),
			mcp.Description("Name of the development VM")),
		mcp.WithString("command",
			mcp.Required(),
			mcp.Description("Command to execute")),
		mcp.WithString("working_dir",
			mcp.Description("Working directory"),
			mcp.DefaultString("/home/vagrant")),
		mcp.WithBoolean("sync_before",
			mcp.Description("Sync files to VM before execution"),
			mcp.DefaultBool(true)),
	)

	srv.AddTool(runBackgroundTool, handleRunBackground(vmManager, syncEngine, executor))

	log.Info().Msg("Execution tools registered")
}

// handleExecInVM handles the exec_in_vm tool
func handleExecInVM(manager exec.VMManager, executor *exec.Executor) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Extract parameters
		vmName, err := request.RequireString("vm_name")
		if err != nil {
			return mcp.NewToolResultError("Missing required parameter: vm_name"), nil
		}

		command, err := request.RequireString("command")
		if err != nil {
			return mcp.NewToolResultError("Missing required parameter: command"), nil
		}

		workingDir := request.GetString("working_dir", "/home/vagrant")

		// Create execution context
		execCtx := exec.ExecutionContext{
			VMName:     vmName,
			WorkingDir: workingDir,
			SyncBefore: false,
			SyncAfter:  false,
		}

		// Execute command
		result, err := executor.ExecuteCommand(ctx, command, execCtx, nil)
		if err != nil {
			return mcp.NewToolResultErrorf("Command execution failed: %v", err), nil
		}

		// Format result
		response := map[string]interface{}{
			"vm_name":    vmName,
			"command":    command,
			"exit_code":  result.ExitCode,
			"stdout":     result.Stdout,
			"stderr":     result.Stderr,
			"duration_s": result.Duration,
		}

		jsonResponse, err := json.Marshal(response)
		if err != nil {
			return mcp.NewToolResultError("Failed to marshal response"), nil
		}

		return mcp.NewToolResultText(string(jsonResponse)), nil
	}
}

// handleExecWithSync handles the exec_with_sync tool
func handleExecWithSync(manager exec.VMManager, syncEngine exec.SyncEngine, executor *exec.Executor) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Extract parameters
		vmName, err := request.RequireString("vm_name")
		if err != nil {
			return mcp.NewToolResultError("Missing required parameter: vm_name"), nil
		}

		command, err := request.RequireString("command")
		if err != nil {
			return mcp.NewToolResultError("Missing required parameter: command"), nil
		}

		workingDir := request.GetString("working_dir", "/home/vagrant")
		syncBefore := request.GetBool("sync_before", true)
		syncAfter := request.GetBool("sync_after", true)

		// Log sync strategy
		log.Info().
			Str("vm", vmName).
			Str("command", command).
			Bool("sync_before", syncBefore).
			Bool("sync_after", syncAfter).
			Msg("Executing command with sync")

		// Create execution context
		execCtx := exec.ExecutionContext{
			VMName:     vmName,
			WorkingDir: workingDir,
			SyncBefore: syncBefore,
			SyncAfter:  syncAfter,
		}

		// Execute command
		result, err := executor.ExecuteCommand(ctx, command, execCtx, nil)
		if err != nil {
			return mcp.NewToolResultErrorf("Command execution failed: %v", err), nil
		}

		// Format result
		response := map[string]interface{}{
			"vm_name":     vmName,
			"command":     command,
			"exit_code":   result.ExitCode,
			"stdout":      result.Stdout,
			"stderr":      result.Stderr,
			"duration_s":  result.Duration,
			"sync_before": syncBefore,
			"sync_after":  syncAfter,
		}

		jsonResponse, err := json.Marshal(response)
		if err != nil {
			return mcp.NewToolResultError("Failed to marshal response"), nil
		}

		return mcp.NewToolResultText(string(jsonResponse)), nil
	}
}

// handleRunBackground handles the run_background_task tool
func handleRunBackground(manager exec.VMManager, syncEngine exec.SyncEngine, executor *exec.Executor) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Extract parameters
		vmName, err := request.RequireString("vm_name")
		if err != nil {
			return mcp.NewToolResultError("Missing required parameter: vm_name"), nil
		}

		command, err := request.RequireString("command")
		if err != nil {
			return mcp.NewToolResultError("Missing required parameter: command"), nil
		}

		workingDir := request.GetString("working_dir", "/home/vagrant")
		syncBefore := request.GetBool("sync_before", true)

		// Create execution context
		execCtx := exec.ExecutionContext{
			VMName:     vmName,
			WorkingDir: workingDir,
			SyncBefore: syncBefore,
			SyncAfter:  false, // No sync after for background tasks
		}

		// Modify command to run in background with nohup
		bgCommand := fmt.Sprintf("nohup %s > /tmp/bg_%s.log 2>&1 &", command,
			vmName)

		// Execute command
		result, err := executor.ExecuteCommand(ctx, bgCommand, execCtx, nil)
		if err != nil {
			return mcp.NewToolResultErrorf("Background task start failed: %v", err), nil
		}

		// Format result
		response := map[string]interface{}{
			"vm_name":   vmName,
			"command":   command,
			"status":    "started",
			"log_file":  fmt.Sprintf("/tmp/bg_%s.log", vmName),
			"exit_code": result.ExitCode,
		}

		jsonResponse, err := json.Marshal(response)
		if err != nil {
			return mcp.NewToolResultError("Failed to marshal response"), nil
		}

		return mcp.NewToolResultText(string(jsonResponse)), nil
	}
}
