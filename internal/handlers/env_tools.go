// Copyright Ricardo Oliveira 2025.
// SPDX-License-Identifier: MPL-2.0

package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rs/zerolog/log"
	"github.com/vagrant-mcp/server/internal/core"
	"github.com/vagrant-mcp/server/internal/errors"
	"github.com/vagrant-mcp/server/internal/exec"
	mcp_pkg "github.com/vagrant-mcp/server/pkg/mcp"
)

// RegisterEnvTools registers all environment-related tools with the MCP server
func RegisterEnvTools(srv *server.MCPServer, vmManager core.VMManager, executor *exec.Executor) {
	// Setup dev environment tool
	type SetupEnvArgs struct {
		VMName   string   `json:"vm_name"`
		Runtimes []string `json:"runtimes"`
		Tools    []string `json:"tools"`
	}
	setupEnvTool := mcp.NewTool("setup_dev_environment",
		mcp.WithDescription("Install language runtimes, tools, and dependencies in the VM"),
		mcp.WithString("vm_name",
			mcp.Required(),
			mcp.Description("Name of the development VM")),
		mcp.WithArray("runtimes",
			mcp.Required(),
			mcp.Description("Language runtimes to install (e.g., 'node', 'python', 'go', etc.)"),
			mcp.Items(map[string]any{"type": "string"})),
		mcp.WithArray("tools",
			mcp.Description("Additional tools to install"),
			mcp.Items(map[string]any{"type": "string"})),
	)

	mcp_pkg.RegisterTypedTool(srv, setupEnvTool, func(ctx context.Context, request mcp.CallToolRequest, args SetupEnvArgs) (*mcp.CallToolResult, error) {
		if args.VMName == "" {
			return mcp.NewToolResultError("missing or invalid 'vm_name' parameter"), nil
		}
		if len(args.Runtimes) == 0 {
			return mcp.NewToolResultError("missing or invalid 'runtimes' parameter"), nil
		}
		// Check VM state
		state, err := vmManager.GetVMState(ctx, args.VMName)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("VM '%s' does not exist: %v", args.VMName, err)), nil
		}

		if state != core.Running {
			return mcp.NewToolResultError(fmt.Sprintf("VM '%s' is not running (current state: %s)", args.VMName, state)), nil
		}

		// Process each runtime
		results := make(map[string]interface{})
		for _, runtime := range args.Runtimes {
			cmdResult, err := installRuntime(ctx, executor, args.VMName, runtime)
			results[runtime] = map[string]interface{}{
				"success": err == nil,
				"output":  cmdResult,
				"error":   err,
			}
		}

		// Get tools to install
		var tools []string
		toolsObj := request.GetArguments()["tools"]
		if toolsList, ok := toolsObj.([]interface{}); ok {
			for _, tool := range toolsList {
				if toolStr, ok := tool.(string); ok {
					tools = append(tools, toolStr)
				}
			}
		}

		// Process each tool
		if len(tools) > 0 {
			toolResults := make(map[string]interface{})
			for _, tool := range tools {
				cmdResult, err := installTool(ctx, executor, args.VMName, tool)
				toolResults[tool] = map[string]interface{}{
					"success": err == nil,
					"output":  cmdResult,
					"error":   err,
				}
			}
			results["tools"] = toolResults
		}

		// Return results
		return mcp.NewToolResultText(fmt.Sprintf("%v", results)), nil
	})

	// Install dev tools tool
	installToolsTool := mcp.NewTool("install_dev_tools",
		mcp.WithDescription("Install specific development tools in the VM"),
		mcp.WithString("vm_name",
			mcp.Required(),
			mcp.Description("Name of the development VM")),
		mcp.WithArray("tools",
			mcp.Required(),
			mcp.Description("Tools to install"),
			mcp.Items(map[string]any{"type": "string"})),
	)

	srv.AddTool(installToolsTool, handleInstallDevTools(vmManager, executor))

	// Configure shell tool
	configureShellTool := mcp.NewTool("configure_shell",
		mcp.WithDescription("Configure shell environment in the VM"),
		mcp.WithString("vm_name",
			mcp.Required(),
			mcp.Description("Name of the development VM")),
		mcp.WithString("shell_type",
			mcp.Description("Shell type to configure"),
			mcp.DefaultString("bash")),
		mcp.WithArray("aliases",
			mcp.Description("Shell aliases to configure"),
			mcp.Items(map[string]any{"type": "string"})),
		mcp.WithArray("env_vars",
			mcp.Description("Environment variables to set"),
			mcp.Items(map[string]any{"type": "string"})),
	)

	srv.AddTool(configureShellTool, handleConfigureShell(vmManager, executor))

	log.Info().Msg("Environment tools registered")
}

// handleInstallDevTools handles the install_dev_tools tool
func handleInstallDevTools(manager core.VMManager, executor *exec.Executor) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		vmName, err := request.RequireString("vm_name")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("missing or invalid 'vm_name' parameter: %v", err)), nil
		}

		toolsObj := request.GetArguments()["tools"]
		var tools []string

		if toolsList, ok := toolsObj.([]interface{}); ok {
			for _, tool := range toolsList {
				if toolStr, ok := tool.(string); ok {
					tools = append(tools, toolStr)
				}
			}
		}

		if len(tools) == 0 {
			return mcp.NewToolResultError("missing or invalid 'tools' parameter"), nil
		}

		// Check VM state
		state, err := manager.GetVMState(ctx, vmName)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("VM '%s' does not exist: %v", vmName, err)), nil
		}

		if state != core.Running {
			return mcp.NewToolResultError(fmt.Sprintf("VM '%s' is not running (current state: %s)", vmName, state)), nil
		}

		// Process each tool
		results := make(map[string]interface{})
		for _, tool := range tools {
			cmdResult, err := installTool(ctx, executor, vmName, tool)
			results[tool] = map[string]interface{}{
				"success": err == nil,
				"output":  cmdResult,
				"error":   err,
			}
		}

		// Return results
		jsonData, err := json.Marshal(results)
		if err != nil {
			return mcp.NewToolResultError("failed to marshal result: " + err.Error()), nil
		}
		return mcp.NewToolResultText(string(jsonData)), nil
	}
}

// handleConfigureShell handles the configure_shell tool
func handleConfigureShell(manager core.VMManager, executor *exec.Executor) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		vmName, err := request.RequireString("vm_name")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("missing or invalid 'vm_name' parameter: %v", err)), nil
		}

		shellType := request.GetString("shell_type", "bash")

		// Check VM state
		state, err := manager.GetVMState(ctx, vmName)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("VM '%s' does not exist: %v", vmName, err)), nil
		}

		if state != core.Running {
			return mcp.NewToolResultError(fmt.Sprintf("VM '%s' is not running (current state: %s)", vmName, state)), nil
		}

		// Process aliases
		aliasesObj := request.GetArguments()["aliases"]
		var aliases []string
		if aliasesList, ok := aliasesObj.([]interface{}); ok {
			for _, alias := range aliasesList {
				if aliasStr, ok := alias.(string); ok {
					aliases = append(aliases, aliasStr)
				}
			}
		}

		// Process env vars
		envVarsObj := request.GetArguments()["env_vars"]
		var envVars []string
		if envVarsList, ok := envVarsObj.([]interface{}); ok {
			for _, envVar := range envVarsList {
				if envVarStr, ok := envVar.(string); ok {
					envVars = append(envVars, envVarStr)
				}
			}
		}

		// Configure shell
		configResult, err := configureShellEnv(ctx, executor, vmName, shellType, aliases, envVars)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to configure shell: %v", err)), nil
		}

		// Return results
		result := map[string]interface{}{
			"vm_name":    vmName,
			"shell_type": shellType,
			"aliases":    aliases,
			"env_vars":   envVars,
			"output":     configResult,
		}

		jsonData, err := json.Marshal(result)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to marshal results: %v", err)), nil
		}

		return mcp.NewToolResultText(string(jsonData)), nil
	}
}

// Helper functions

// installRuntime installs a specific language runtime
func installRuntime(ctx context.Context, executor *exec.Executor, vmName string, runtime string) (string, error) {
	var cmd string

	switch runtime {
	case "node":
		cmd = "curl -sL https://deb.nodesource.com/setup_16.x | sudo -E bash - && sudo apt-get install -y nodejs"
	case "python":
		cmd = "sudo apt-get update && sudo apt-get install -y python3 python3-pip python3-venv"
	case "go":
		cmd = "sudo apt-get update && sudo apt-get install -y golang"
	case "ruby":
		cmd = "sudo apt-get update && sudo apt-get install -y ruby-full"
	case "php":
		cmd = "sudo apt-get update && sudo apt-get install -y php php-cli php-fpm php-json php-common php-mysql php-zip php-gd php-mbstring php-curl php-xml php-pear php-bcmath"
	case "java":
		cmd = "sudo apt-get update && sudo apt-get install -y default-jdk"
	default:
		return "", errors.InvalidInput(fmt.Sprintf("unsupported runtime: %s", runtime))
	}

	// Setup execution context
	execCtx := exec.ExecutionContext{
		VMName:     vmName,
		WorkingDir: "/home/vagrant",
		SyncBefore: false,
		SyncAfter:  false,
	}

	// Execute the command
	result, err := executor.ExecuteCommand(ctx, cmd, execCtx, nil)
	if err != nil {
		return "", errors.OperationFailed("install runtime", err)
	}

	return result.Stdout, nil
}

// installTool installs a specific development tool
func installTool(ctx context.Context, executor *exec.Executor, vmName string, tool string) (string, error) {
	var cmd string

	switch tool {
	case "git":
		cmd = "sudo apt-get update && sudo apt-get install -y git"
	case "docker":
		cmd = "curl -fsSL https://get.docker.com -o get-docker.sh && sudo sh get-docker.sh"
	case "docker-compose":
		cmd = "sudo curl -L \"https://github.com/docker/compose/releases/download/1.29.2/docker-compose-$(uname -s)-$(uname -m)\" -o /usr/local/bin/docker-compose && sudo chmod +x /usr/local/bin/docker-compose"
	case "nginx":
		cmd = "sudo apt-get update && sudo apt-get install -y nginx"
	case "postgresql":
		cmd = "sudo apt-get update && sudo apt-get install -y postgresql postgresql-contrib"
	case "mysql":
		cmd = "sudo apt-get update && sudo apt-get install -y mysql-server"
	case "mongodb":
		cmd = "sudo apt-get update && sudo apt-get install -y mongodb"
	case "redis":
		cmd = "sudo apt-get update && sudo apt-get install -y redis-server"
	default:
		// Try to install as a generic package
		cmd = fmt.Sprintf("sudo apt-get update && sudo apt-get install -y %s", tool)
	}

	// Setup execution context
	execCtx := exec.ExecutionContext{
		VMName:     vmName,
		WorkingDir: "/home/vagrant",
		SyncBefore: false,
		SyncAfter:  false,
	}

	// Execute the command
	result, err := executor.ExecuteCommand(ctx, cmd, execCtx, nil)
	if err != nil {
		return "", errors.OperationFailed("install tool", err)
	}

	return result.Stdout, nil
}

// configureShellEnv configures shell environment
func configureShellEnv(ctx context.Context, executor *exec.Executor, vmName string, shellType string, aliases []string, envVars []string) (string, error) {
	var rcFile string
	switch shellType {
	case "bash":
		rcFile = "/home/vagrant/.bashrc"
	case "zsh":
		rcFile = "/home/vagrant/.zshrc"
	default:
		return "", errors.InvalidInput(fmt.Sprintf("unsupported shell type: %s", shellType))
	}

	// Setup execution context
	execCtx := exec.ExecutionContext{
		VMName:     vmName,
		WorkingDir: "/home/vagrant",
		SyncBefore: false,
		SyncAfter:  false,
	}

	// Build shell configuration
	var config strings.Builder
	config.WriteString("\n# Configured by vagrant-mcp-server\n")

	// Add aliases
	if len(aliases) > 0 {
		config.WriteString("\n# Aliases\n")
		for _, alias := range aliases {
			config.WriteString(fmt.Sprintf("alias %s\n", alias))
		}
	}

	// Add environment variables
	if len(envVars) > 0 {
		config.WriteString("\n# Environment Variables\n")
		for _, envVar := range envVars {
			config.WriteString(fmt.Sprintf("export %s\n", envVar))
		}
	}

	// Write to rc file
	appendCmd := fmt.Sprintf("echo '%s' >> %s", config.String(), rcFile)
	result, err := executor.ExecuteCommand(ctx, appendCmd, execCtx, nil)
	if err != nil {
		return "", errors.OperationFailed("configure shell", err)
	}

	// Source the file to apply changes
	sourceCmd := fmt.Sprintf("source %s", rcFile)
	_, _ = executor.ExecuteCommand(ctx, sourceCmd, execCtx, nil)

	return result.Stdout, nil
}
