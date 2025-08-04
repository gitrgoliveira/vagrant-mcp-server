package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rs/zerolog/log"
	"github.com/vagrant-mcp/server/internal/core"
	"github.com/vagrant-mcp/server/internal/exec"
)

// RegisterMCPResources registers all resources with the MCP server
func RegisterMCPResources(srv *server.MCPServer, vmManager core.VMManager, executor *exec.Executor) {
	// Register VM status resource
	registerVMStatusResource(srv, vmManager)

	// Register VM config resource
	registerVMConfigResource(srv, vmManager)

	// Register VM files resource
	registerVMFilesResource(srv, vmManager, executor)

	// Register VM logs resource
	registerVMLogsResource(srv, vmManager, executor)

	// Register VM environment resources
	registerVMEnvironmentResource(srv, vmManager, executor)

	// Register VM installed tools resource
	registerVMInstalledToolsResource(srv, vmManager, executor)

	log.Info().Msg("All resources registered with MCP server")
}

// registerVMStatusResource registers the VM status resource
func registerVMStatusResource(srv *server.MCPServer, vmManager core.VMManager) {
	statusResource := mcp.NewResource(
		"devvm://status",
		"VM Status",
		mcp.WithResourceDescription("Current development VM status and health"),
	)

	srv.AddResource(statusResource, func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		// Format result
		result := make(map[string]interface{})

		// List VM directories using the accessor
		baseDir := vmManager.GetBaseDir()
		vmDirs, dirErr := filepath.Glob(filepath.Join(baseDir, "*"))
		if dirErr != nil {
			return nil, fmt.Errorf("failed to list VM directories: %w", dirErr)
		}

		for _, vmDir := range vmDirs {
			vmName := filepath.Base(vmDir)
			state, err := vmManager.GetVMState(context.Background(), vmName)
			if err != nil {
				result[vmName] = map[string]interface{}{
					"state": "error",
					"error": err.Error(),
				}
				continue
			}

			result[vmName] = map[string]interface{}{
				"state": state,
			}
		}

		// Marshal to JSON
		jsonData, err := json.Marshal(result)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal status: %w", err)
		}

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      request.Params.URI,
				MIMEType: "application/json",
				Text:     string(jsonData),
			},
		}, nil
	})
}

// registerVMConfigResource registers the VM config resource
func registerVMConfigResource(srv *server.MCPServer, vmManager core.VMManager) {
	configResource := mcp.NewResource(
		"devvm://config/{vmName}",
		"VM Configuration",
		mcp.WithResourceDescription("Current VM configuration and sync settings"),
	)

	srv.AddResource(configResource, func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		// Extract VM name from URI
		uri := request.Params.URI
		vmName := ""

		// Parse VM name from URI (format: devvm://config/{vmName})
		parts := strings.Split(strings.TrimPrefix(uri, "devvm://config/"), "/")
		if len(parts) > 0 {
			vmName = parts[0]
		}

		if vmName == "" {
			return nil, fmt.Errorf("VM name not specified")
		}

		// Get VM configuration
		config, err := vmManager.GetVMConfig(context.Background(), vmName)
		if err != nil {
			return nil, fmt.Errorf("failed to get VM config: %w", err)
		}

		// Marshal to JSON
		jsonData, err := json.Marshal(config)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal config: %w", err)
		}

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      request.Params.URI,
				MIMEType: "application/json",
				Text:     string(jsonData),
			},
		}, nil
	})
}

// registerVMFilesResource registers the VM files resource
func registerVMFilesResource(srv *server.MCPServer, vmManager core.VMManager, executor *exec.Executor) {
	filesResource := mcp.NewResource(
		"devvm://files/{path*}",
		"VM Files",
		mcp.WithResourceDescription("Access to VM file system (read-only)"),
	)

	srv.AddResource(filesResource, func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		// Extract VM name and path from URI
		uri := request.Params.URI
		pathParam := strings.TrimPrefix(uri, "devvm://files/")
		if pathParam == "" {
			return nil, fmt.Errorf("missing path parameter")
		}

		// Split the path to get VM name and file path
		parts := strings.SplitN(pathParam, "/", 2)
		if len(parts) < 2 {
			return nil, fmt.Errorf("invalid path format: expected 'vmName/path'")
		}

		vmName := parts[0]
		path := parts[1]

		// Check VM state
		state, err := vmManager.GetVMState(context.Background(), vmName)
		if err != nil {
			return nil, fmt.Errorf("failed to get VM state: %w", err)
		}

		if state != core.Running {
			return nil, fmt.Errorf("VM is not running (current state: %s)", state)
		}

		// Setup execution context
		execCtx := exec.ExecutionContext{
			VMName:     vmName,
			WorkingDir: "/vagrant",
			SyncBefore: false,
			SyncAfter:  false,
		}

		// Read file content from VM
		command := fmt.Sprintf("cat %s", path)
		result, err := executor.ExecuteCommand(ctx, command, execCtx, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to read file: %w", err)
		}

		// Determine MIME type from file extension
		mimeType := "text/plain"
		ext := filepath.Ext(path)
		switch strings.ToLower(ext) {
		case ".json":
			mimeType = "application/json"
		case ".html":
			mimeType = "text/html"
		case ".js":
			mimeType = "application/javascript"
		case ".css":
			mimeType = "text/css"
		case ".png":
			mimeType = "image/png"
		case ".jpg", ".jpeg":
			mimeType = "image/jpeg"
		case ".gif":
			mimeType = "image/gif"
		case ".md":
			mimeType = "text/markdown"
		}

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      request.Params.URI,
				MIMEType: mimeType,
				Text:     result.Stdout,
			},
		}, nil
	})
}

// registerVMLogsResource registers the VM logs resource
func registerVMLogsResource(srv *server.MCPServer, vmManager core.VMManager, executor *exec.Executor) {
	logsResource := mcp.NewResource(
		"devvm://logs/{logType}",
		"VM Logs",
		mcp.WithResourceDescription("VM logs for sync and provisioning"),
	)

	srv.AddResource(logsResource, func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		// Extract log type from URI
		uri := request.Params.URI
		logType := strings.TrimPrefix(uri, "devvm://logs/")
		if logType == "" {
			return nil, fmt.Errorf("missing logType parameter")
		}

		// Get VM name from URI segment
		// In the real implementation, this would parse VM name from query params
		// For now, let's extract it from the URI or use a default
		vmName := "default"
		if vmName == "" {
			return nil, fmt.Errorf("missing required query parameter: vm")
		}

		// Check VM state
		state, err := vmManager.GetVMState(context.Background(), vmName)
		if err != nil {
			return nil, fmt.Errorf("failed to get VM state: %w", err)
		}

		if state != core.Running {
			return nil, fmt.Errorf("VM is not running (current state: %s)", state)
		}

		// Setup execution context
		execCtx := exec.ExecutionContext{
			VMName:     vmName,
			WorkingDir: "/",
			SyncBefore: false,
			SyncAfter:  false,
		}

		// Get log contents with tail to avoid massive output
		tailCmd := fmt.Sprintf("tail -n 200 '/var/log/%s' 2>/dev/null || echo 'ERROR: log not found'", logType)
		result, err := executor.ExecuteCommand(ctx, tailCmd, execCtx, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to read log: %w", err)
		}

		if result.Stdout == "ERROR: log not found" {
			return nil, fmt.Errorf("log not found: %s", logType)
		}

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      request.Params.URI,
				MIMEType: "text/plain",
				Text:     result.Stdout,
			},
		}, nil
	})
}

// registerVMEnvironmentResource registers the VM environment resource
func registerVMEnvironmentResource(srv *server.MCPServer, vmManager core.VMManager, executor *exec.Executor) {
	envResource := mcp.NewResource(
		"devvm://env/{vmName}",
		"VM Environment",
		mcp.WithResourceDescription("Environment configuration for development VMs"),
	)

	srv.AddResource(envResource, func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		// Extract VM name from URI
		uri := request.Params.URI
		vmName := ""

		// Parse VM name from URI (format: devvm://env/{vmName})
		path := strings.TrimPrefix(uri, "devvm://env/")
		if path != "" {
			vmName = path
		}

		if vmName == "" {
			return nil, fmt.Errorf("VM name not specified")
		}

		// Check VM state
		state, err := vmManager.GetVMState(context.Background(), vmName)
		if err != nil {
			return nil, fmt.Errorf("failed to get VM state: %w", err)
		}

		if state != core.Running {
			return nil, fmt.Errorf("VM is not running (current state: %s)", state)
		}

		// Setup execution context
		execCtx := exec.ExecutionContext{
			VMName:     vmName,
			WorkingDir: "/",
			SyncBefore: false,
			SyncAfter:  false,
		}

		// Get environment information
		envCmd := "echo -n '{\"environment\": {'; " +
			"echo -n '\"os\": \"'; cat /etc/os-release | grep PRETTY_NAME | cut -d '=' -f 2 | tr -d '\"'; echo -n '\", '; " +
			"echo -n '\"kernel\": \"'; uname -r; echo -n '\", '; " +
			"echo -n '\"shell\": \"'; echo $SHELL; echo -n '\"'; " +
			"echo '} }'"

		result, err := executor.ExecuteCommand(ctx, envCmd, execCtx, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to get environment information: %w", err)
		}

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      request.Params.URI,
				MIMEType: "application/json",
				Text:     result.Stdout,
			},
		}, nil
	})
}

// registerVMInstalledToolsResource registers the VM installed tools resource
func registerVMInstalledToolsResource(srv *server.MCPServer, vmManager core.VMManager, executor *exec.Executor) {
	toolsResource := mcp.NewResource(
		"devvm://tools/{vmName}",
		"VM Installed Tools",
		mcp.WithResourceDescription("Information about tools installed in the VM"),
	)

	srv.AddResource(toolsResource, func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		// Extract VM name from URI
		uri := request.Params.URI
		vmName := ""

		// Parse VM name from URI (format: devvm://tools/{vmName})
		path := strings.TrimPrefix(uri, "devvm://tools/")
		if path != "" {
			vmName = path
		}

		if vmName == "" {
			return nil, fmt.Errorf("VM name not specified")
		}

		// Check VM state
		state, err := vmManager.GetVMState(context.Background(), vmName)
		if err != nil {
			return nil, fmt.Errorf("failed to get VM state: %w", err)
		}

		if state != core.Running {
			return nil, fmt.Errorf("VM is not running (current state: %s)", state)
		}

		// Setup execution context
		execCtx := exec.ExecutionContext{
			VMName:     vmName,
			WorkingDir: "/",
			SyncBefore: false,
			SyncAfter:  false,
		}

		// Get installed tools information
		toolsCmd := "echo '{\"tools\": {'; " +
			"echo -n '\"node\": \"'; command -v node > /dev/null && node -v 2>/dev/null || echo 'not installed'; echo '\", '; " +
			"echo -n '\"npm\": \"'; command -v npm > /dev/null && npm -v 2>/dev/null || echo 'not installed'; echo '\", '; " +
			"echo -n '\"python\": \"'; command -v python3 > /dev/null && python3 --version 2>/dev/null || echo 'not installed'; echo '\", '; " +
			"echo -n '\"pip\": \"'; command -v pip3 > /dev/null && pip3 --version 2>/dev/null || echo 'not installed'; echo '\", '; " +
			"echo -n '\"go\": \"'; command -v go > /dev/null && go version 2>/dev/null || echo 'not installed'; echo '\", '; " +
			"echo -n '\"ruby\": \"'; command -v ruby > /dev/null && ruby --version 2>/dev/null || echo 'not installed'; echo '\", '; " +
			"echo -n '\"docker\": \"'; command -v docker > /dev/null && docker --version 2>/dev/null || echo 'not installed'; echo '\"'; " +
			"echo '} }'"

		result, err := executor.ExecuteCommand(ctx, toolsCmd, execCtx, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to get tools information: %w", err)
		}

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      request.Params.URI,
				MIMEType: "application/json",
				Text:     result.Stdout,
			},
		}, nil
	})
}
