package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rs/zerolog/log"
	"github.com/vagrant-mcp/server/internal/core"
	"github.com/vagrant-mcp/server/internal/exec"
	mcp_pkg "github.com/vagrant-mcp/server/pkg/mcp"
)

// RegisterExecTools registers all execution-related tools with the MCP server
func RegisterExecTools(srv *server.MCPServer, vmManager core.VMManager, syncEngine core.SyncEngine, executor *exec.Executor) {
	// Execute in VM tool
	type ExecInVMArgs struct {
		VMName     string `json:"vm_name"`
		Command    string `json:"command"`
		WorkingDir string `json:"working_dir"`
	}
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

	mcp_pkg.RegisterTypedTool(srv, execInVMTool, func(ctx context.Context, request mcp.CallToolRequest, args ExecInVMArgs) (*mcp.CallToolResult, error) {
		if args.VMName == "" || args.Command == "" {
			return mcp.NewToolResultError("Missing required parameter: vm_name or command"), nil
		}
		workingDir := args.WorkingDir
		if workingDir == "" {
			workingDir = "/home/vagrant"
		}
		execCtx := exec.ExecutionContext{
			VMName:     args.VMName,
			WorkingDir: workingDir,
			SyncBefore: false,
			SyncAfter:  false,
		}
		result, err := executor.ExecuteCommand(ctx, args.Command, execCtx, nil)
		if err != nil {
			return mcp.NewToolResultErrorf("Command execution failed: %v", err), nil
		}
		response := map[string]interface{}{
			"vm_name":    args.VMName,
			"command":    args.Command,
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
	})

	// Execute with sync tool
	type ExecWithSyncArgs struct {
		VMName     string `json:"vm_name"`
		Command    string `json:"command"`
		WorkingDir string `json:"working_dir"`
		SyncBefore bool   `json:"sync_before"`
		SyncAfter  bool   `json:"sync_after"`
	}
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

	mcp_pkg.RegisterTypedTool(srv, execWithSyncTool, func(ctx context.Context, request mcp.CallToolRequest, args ExecWithSyncArgs) (*mcp.CallToolResult, error) {
		if args.VMName == "" || args.Command == "" {
			return mcp.NewToolResultError("Missing required parameter: vm_name or command"), nil
		}
		workingDir := args.WorkingDir
		if workingDir == "" {
			workingDir = "/home/vagrant"
		}
		log.Info().
			Str("vm", args.VMName).
			Str("command", args.Command).
			Bool("sync_before", args.SyncBefore).
			Bool("sync_after", args.SyncAfter).
			Msg("Executing command with sync")
		execCtx := exec.ExecutionContext{
			VMName:     args.VMName,
			WorkingDir: workingDir,
			SyncBefore: args.SyncBefore,
			SyncAfter:  args.SyncAfter,
		}
		result, err := executor.ExecuteCommand(ctx, args.Command, execCtx, nil)
		if err != nil {
			return mcp.NewToolResultErrorf("Command execution failed: %v", err), nil
		}
		response := map[string]interface{}{
			"vm_name":     args.VMName,
			"command":     args.Command,
			"exit_code":   result.ExitCode,
			"stdout":      result.Stdout,
			"stderr":      result.Stderr,
			"duration_s":  result.Duration,
			"sync_before": args.SyncBefore,
			"sync_after":  args.SyncAfter,
		}
		jsonResponse, err := json.Marshal(response)
		if err != nil {
			return mcp.NewToolResultError("Failed to marshal response"), nil
		}
		return mcp.NewToolResultText(string(jsonResponse)), nil
	})

	// Run background task tool
	type RunBackgroundArgs struct {
		VMName     string `json:"vm_name"`
		Command    string `json:"command"`
		WorkingDir string `json:"working_dir"`
		SyncBefore bool   `json:"sync_before"`
	}
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

	mcp_pkg.RegisterTypedTool(srv, runBackgroundTool, func(ctx context.Context, request mcp.CallToolRequest, args RunBackgroundArgs) (*mcp.CallToolResult, error) {
		if args.VMName == "" || args.Command == "" {
			return mcp.NewToolResultError("Missing required parameter: vm_name or command"), nil
		}
		workingDir := args.WorkingDir
		if workingDir == "" {
			workingDir = "/home/vagrant"
		}
		execCtx := exec.ExecutionContext{
			VMName:     args.VMName,
			WorkingDir: workingDir,
			SyncBefore: args.SyncBefore,
			SyncAfter:  false, // No sync after for background tasks
		}
		bgCommand := fmt.Sprintf("nohup %s > /tmp/bg_%s.log 2>&1 &", args.Command, args.VMName)
		result, err := executor.ExecuteCommand(ctx, bgCommand, execCtx, nil)
		if err != nil {
			return mcp.NewToolResultErrorf("Background task start failed: %v", err), nil
		}
		response := map[string]interface{}{
			"vm_name":   args.VMName,
			"command":   args.Command,
			"status":    "started",
			"log_file":  fmt.Sprintf("/tmp/bg_%s.log", args.VMName),
			"exit_code": result.ExitCode,
		}
		jsonResponse, err := json.Marshal(response)
		if err != nil {
			return mcp.NewToolResultError("Failed to marshal response"), nil
		}
		return mcp.NewToolResultText(string(jsonResponse)), nil
	})

	log.Info().Msg("Execution tools registered")
}
