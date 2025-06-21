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

// RegisterVMTools registers all VM-related tools with the MCP server
func RegisterVMTools(srv *server.MCPServer, vmManager exec.VMManager, syncEngine exec.SyncEngine) {
	// Create dev VM tool
	createVMTool := mcp.NewTool("create_dev_vm",
		mcp.WithDescription("Create and configure a development VM with Vagrant"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name for the development VM")),
		mcp.WithString("project_path",
			mcp.Required(),
			mcp.Description("Path to the project directory to sync")),
		mcp.WithNumber("cpu",
			mcp.Description("Number of CPU cores"),
			mcp.DefaultNumber(2)),
		mcp.WithNumber("memory",
			mcp.Description("Amount of memory in MB"),
			mcp.DefaultNumber(2048)),
		mcp.WithString("box",
			mcp.Description("Vagrant box to use"),
			mcp.DefaultString("ubuntu/focal64")),
		mcp.WithString("sync_type",
			mcp.Description("Sync type to use"),
			mcp.DefaultString("rsync")),
		mcp.WithArray("ports",
			mcp.Description("Ports to forward (format: [host:guest])")),
		mcp.WithArray("exclude_patterns",
			mcp.Description("Patterns to exclude from sync")),
	)

	srv.AddTool(createVMTool, handleCreateDevVM(vmManager))

	// Ensure dev VM tool
	ensureVMTool := mcp.NewTool("ensure_dev_vm",
		mcp.WithDescription("Ensure development VM is running, create if it doesn't exist"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the development VM")),
		mcp.WithString("project_path",
			mcp.Description("Path to the project directory to sync (only needed for creation)")),
	)

	srv.AddTool(ensureVMTool, handleEnsureDevVM(vmManager, syncEngine))

	// Destroy dev VM tool
	destroyVMTool := mcp.NewTool("destroy_dev_vm",
		mcp.WithDescription("Clean up development VM and associated resources"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the development VM")),
	)

	srv.AddTool(destroyVMTool, handleDestroyDevVM(vmManager))

	// Get VM status tool
	getStatusTool := mcp.NewTool("get_vm_status",
		mcp.WithDescription("Get status of one or all development VMs"),
		mcp.WithString("name",
			mcp.Description("Name of the development VM (optional)")),
	)

	srv.AddTool(getStatusTool, handleGetVMStatus(vmManager))
}

// handleCreateDevVM handles the create_dev_vm tool
func handleCreateDevVM(manager exec.VMManager) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Extract parameters
		name, err := request.RequireString("name")
		if err != nil {
			return mcp.NewToolResultError("Missing required parameter: name"), nil
		}

		projectPath, err := request.RequireString("project_path")
		if err != nil {
			return mcp.NewToolResultError("Missing required parameter: project_path"), nil
		}

		// Extract optional parameters with defaults
		cpu := request.GetFloat("cpu", 2.0)
		memory := request.GetFloat("memory", 2048.0)
		box := request.GetString("box", "ubuntu/focal64")
		syncType := request.GetString("sync_type", "rsync")

		// Extract array parameters
		var ports []vm.Port
		args := request.GetArguments()
		if args != nil {
			if portsArray, ok := args["ports"].([]interface{}); ok && len(portsArray) > 0 {
				for _, portMapping := range portsArray {
					if portMap, ok := portMapping.(map[string]interface{}); ok {
						var port vm.Port
						if guest, ok := portMap["guest"].(float64); ok {
							port.Guest = int(guest)
						}
						if host, ok := portMap["host"].(float64); ok {
							port.Host = int(host)
						}
						ports = append(ports, port)
					}
				}
			} else {
				// Default ports for common development services
				ports = []vm.Port{
					{Guest: 3000, Host: 3000}, // Node.js/React
					{Guest: 8000, Host: 8000}, // Django/Flask/etc.
					{Guest: 5432, Host: 5432}, // PostgreSQL
					{Guest: 3306, Host: 3306}, // MySQL
					{Guest: 6379, Host: 6379}, // Redis
				}
			}
		}

		// Extract exclude patterns
		var excludePatterns []string
		if args != nil {
			if excludePatternsArray, ok := args["exclude_patterns"].([]interface{}); ok {
				for _, pattern := range excludePatternsArray {
					if patternStr, ok := pattern.(string); ok {
						excludePatterns = append(excludePatterns, patternStr)
					}
				}
			} else {
				// Default exclude patterns for common build artifacts and dependencies
				excludePatterns = []string{
					"node_modules",
					".git",
					"*.log",
					"dist",
					"build",
					"__pycache__",
					"*.pyc",
					"venv",
					".venv",
					"*.o",
					"*.out",
				}
			}
		}

		// Create VM config
		config := vm.VMConfig{
			Box:                 box,
			CPU:                 int(cpu),
			Memory:              int(memory),
			SyncType:            syncType,
			Ports:               ports,
			SyncExcludePatterns: excludePatterns,
		}

		// Create VM
		if err := manager.CreateVM(name, projectPath, config); err != nil {
			return mcp.NewToolResultErrorf("Failed to create VM: %v", err), nil
		}

		// Return success
		response := map[string]interface{}{
			"name":         name,
			"project_path": projectPath,
			"config":       config,
			"status":       "created",
			"timestamp":    time.Now().Format(time.RFC3339),
		}

		jsonResponse, err := json.Marshal(response)
		if err != nil {
			return mcp.NewToolResultError("Failed to marshal response"), nil
		}

		return mcp.NewToolResultText(string(jsonResponse)), nil
	}
}

// handleEnsureDevVM handles the ensure_dev_vm tool
func handleEnsureDevVM(manager exec.VMManager, syncEngine exec.SyncEngine) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Extract parameters
		name, err := request.RequireString("name")
		if err != nil {
			return mcp.NewToolResultError("Missing required parameter: name"), nil
		}

		// Get VM state
		state, err := manager.GetVMState(name)
		if err != nil {
			// VM doesn't exist, see if we can create it
			projectPath, err := request.RequireString("project_path")
			if err != nil {
				return mcp.NewToolResultError("VM doesn't exist. Missing required parameter for creation: project_path"), nil
			}

			// Create default config
			config := vm.VMConfig{
				Box:    "ubuntu/focal64",
				CPU:    2,
				Memory: 2048,
				Ports: []vm.Port{
					{Guest: 3000, Host: 3000},
					{Guest: 8000, Host: 8000},
					{Guest: 5432, Host: 5432},
				},
				SyncType: "rsync",
				SyncExcludePatterns: []string{
					"node_modules", ".git", "*.log", "dist", "build",
				},
			}

			// Create VM
			if err := manager.CreateVM(name, projectPath, config); err != nil {
				return mcp.NewToolResultErrorf("Failed to create VM: %v", err), nil
			}

			// Register VM for syncing
			if err := syncEngine.RegisterVM(name); err != nil {
				log.Error().Err(err).Msg("Failed to register VM with sync engine")
			}

			return mcp.NewToolResultText(fmt.Sprintf("VM '%s' created and started", name)), nil
		}

		// VM exists, check if it's running
		if state != vm.Running {
			// Start VM
			if err := manager.StartVM(name); err != nil {
				return mcp.NewToolResultErrorf("Failed to start VM: %v", err), nil
			}
			return mcp.NewToolResultText(fmt.Sprintf("VM '%s' started", name)), nil
		}

		// VM is already running
		return mcp.NewToolResultText(fmt.Sprintf("VM '%s' is already running", name)), nil
	}
}

// handleDestroyDevVM handles the destroy_dev_vm tool
func handleDestroyDevVM(manager exec.VMManager) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Extract parameters
		name, err := request.RequireString("name")
		if err != nil {
			return mcp.NewToolResultError("Missing required parameter: name"), nil
		}

		// Destroy VM
		if err := manager.DestroyVM(name); err != nil {
			return mcp.NewToolResultErrorf("Failed to destroy VM: %v", err), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("VM '%s' destroyed", name)), nil
	}
}

// handleGetVMStatus handles the get_vm_status tool
func handleGetVMStatus(manager exec.VMManager) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Extract parameters
		name := request.GetString("name", "")
		if name != "" {
			// Get status of specific VM
			state, err := manager.GetVMState(name)
			if err != nil {
				return mcp.NewToolResultErrorf("Failed to get VM status: %v", err), nil
			}

			response := map[string]interface{}{
				"name":  name,
				"state": state,
			}

			jsonResponse, err := json.Marshal(response)
			if err != nil {
				return mcp.NewToolResultError("Failed to marshal response"), nil
			}

			return mcp.NewToolResultText(string(jsonResponse)), nil
		}

		// Get status of all VMs
		// Here we'd need to implement a method to list all VMs, but for now we'll return an error
		return mcp.NewToolResultText("Feature to list all VMs not yet implemented"), nil
	}
}
