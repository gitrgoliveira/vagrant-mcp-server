package tools

import (
	"bytes"
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/vagrant-mcp/server/internal/exec"
	"github.com/vagrant-mcp/server/internal/sync"
	"github.com/vagrant-mcp/server/internal/vm"
	"github.com/vagrant-mcp/server/pkg/mcp"
)

// ExecInVMTool implements the exec_in_vm tool
type ExecInVMTool struct {
	manager    *vm.Manager
	syncEngine *sync.Engine
	executor   *exec.Executor
}

// Name returns the tool name
func (t *ExecInVMTool) Name() string {
	return "exec_in_vm"
}

// Description returns the tool description
func (t *ExecInVMTool) Description() string {
	return "Execute a command in the VM with guaranteed file synchronization"
}

// Execute performs the tool action
func (t *ExecInVMTool) Execute(params map[string]interface{}) (interface{}, error) {
	// Validate required parameters
	vmName, ok := params["vm_name"].(string)
	if !ok || vmName == "" {
		return nil, fmt.Errorf("missing or invalid 'vm_name' parameter")
	}

	command, ok := params["command"].(string)
	if !ok || command == "" {
		return nil, fmt.Errorf("missing or invalid 'command' parameter")
	}

	// Check if VM exists and is running
	state, err := t.manager.GetVMState(vmName)
	if err != nil {
		return nil, fmt.Errorf("VM '%s' does not exist or cannot be accessed: %w", vmName, err)
	}

	if state != vm.StateRunning {
		return nil, fmt.Errorf("VM '%s' is not running (current state: %s)", vmName, state)
	}

	// Get optional parameters
	workingDir := "/vagrant"
	if wdParam, ok := params["working_dir"].(string); ok && wdParam != "" {
		workingDir = wdParam
	}

	// Parse environment variables
	envVars := make(map[string]string)
	if envObj, ok := params["env"].(map[string]interface{}); ok {
		for key, val := range envObj {
			if valStr, ok := val.(string); ok {
				envVars[key] = valStr
			}
		}
	}

	// Setup execution context
	execCtx := exec.ExecutionContext{
		VMName:      vmName,
		WorkingDir:  workingDir,
		Environment: envVars,
		SyncBefore:  true, // Always sync before execution
		SyncAfter:   true, // Always sync after execution
	}

	// Determine if we need real-time streaming
	streaming := false
	if streamParam, ok := params["stream"].(bool); ok {
		streaming = streamParam
	}

	// Execute with or without streaming
	var result *exec.CommandResult
	ctx := context.Background()

	if streaming {
		// For streaming, we'll capture output in a buffer and return it later
		var stdout, stderr bytes.Buffer
		callback := func(data []byte, isStderr bool) {
			if isStderr {
				stderr.Write(data)
			} else {
				stdout.Write(data)
			}
		}

		result, err = t.executor.ExecuteCommand(ctx, command, execCtx, callback)
	} else {
		// For non-streaming, just execute and return the result
		result, err = t.executor.ExecuteCommand(ctx, command, execCtx, nil)
	}

	if err != nil {
		return nil, fmt.Errorf("command execution failed: %w", err)
	}

	// Return execution result
	return map[string]interface{}{
		"vm_name":       vmName,
		"command":       command,
		"working_dir":   workingDir,
		"exit_code":     result.ExitCode,
		"stdout":        result.Stdout,
		"stderr":        result.Stderr,
		"duration":      result.Duration,
		"success":       result.ExitCode == 0,
		"synced_before": true,
		"synced_after":  true,
	}, nil
}

// ExecWithSyncTool implements the exec_with_sync tool
type ExecWithSyncTool struct {
	manager    *vm.Manager
	syncEngine *sync.Engine
	executor   *exec.Executor
}

// Name returns the tool name
func (t *ExecWithSyncTool) Name() string {
	return "exec_with_sync"
}

// Description returns the tool description
func (t *ExecWithSyncTool) Description() string {
	return "Execute a command in the VM with explicit sync control"
}

// Execute performs the tool action
func (t *ExecWithSyncTool) Execute(params map[string]interface{}) (interface{}, error) {
	// Validate required parameters
	vmName, ok := params["vm_name"].(string)
	if !ok || vmName == "" {
		return nil, fmt.Errorf("missing or invalid 'vm_name' parameter")
	}

	command, ok := params["command"].(string)
	if !ok || command == "" {
		return nil, fmt.Errorf("missing or invalid 'command' parameter")
	}

	// Check if VM exists and is running
	state, err := t.manager.GetVMState(vmName)
	if err != nil {
		return nil, fmt.Errorf("VM '%s' does not exist or cannot be accessed: %w", vmName, err)
	}

	if state != vm.StateRunning {
		return nil, fmt.Errorf("VM '%s' is not running (current state: %s)", vmName, state)
	}

	// Get optional parameters
	workingDir := "/vagrant"
	if wdParam, ok := params["working_dir"].(string); ok && wdParam != "" {
		workingDir = wdParam
	}

	// Parse environment variables
	envVars := make(map[string]string)
	if envObj, ok := params["env"].(map[string]interface{}); ok {
		for key, val := range envObj {
			if valStr, ok := val.(string); ok {
				envVars[key] = valStr
			}
		}
	}

	// Get sync control options with defaults
	syncBefore := true
	if syncBeforeParam, ok := params["sync_before"].(bool); ok {
		syncBefore = syncBeforeParam
	}

	syncAfter := true
	if syncAfterParam, ok := params["sync_after"].(bool); ok {
		syncAfter = syncAfterParam
	}

	// Setup execution context
	execCtx := exec.ExecutionContext{
		VMName:      vmName,
		WorkingDir:  workingDir,
		Environment: envVars,
		SyncBefore:  syncBefore,
		SyncAfter:   syncAfter,
	}

	// Execute command
	ctx := context.Background()
	result, err := t.executor.ExecuteCommand(ctx, command, execCtx, nil)
	if err != nil {
		return nil, fmt.Errorf("command execution failed: %w", err)
	}

	// Return execution result
	return map[string]interface{}{
		"vm_name":       vmName,
		"command":       command,
		"working_dir":   workingDir,
		"exit_code":     result.ExitCode,
		"stdout":        result.Stdout,
		"stderr":        result.Stderr,
		"duration":      result.Duration,
		"success":       result.ExitCode == 0,
		"synced_before": syncBefore,
		"synced_after":  syncAfter,
	}, nil
}

// RegisterExecTools registers all execution-related tools with the MCP server
func RegisterExecTools(server *mcp.Server, vmManager *vm.Manager, syncEngine *sync.Engine, executor *exec.Executor) error {
	// Create tools
	execInVMTool := &ExecInVMTool{
		manager:    vmManager,
		syncEngine: syncEngine,
		executor:   executor,
	}

	execWithSyncTool := &ExecWithSyncTool{
		manager:    vmManager,
		syncEngine: syncEngine,
		executor:   executor,
	}

	// Register tools
	if err := server.RegisterTool(execInVMTool); err != nil {
		return fmt.Errorf("failed to register exec_in_vm tool: %w", err)
	}

	if err := server.RegisterTool(execWithSyncTool); err != nil {
		return fmt.Errorf("failed to register exec_with_sync tool: %w", err)
	}

	// TODO: Implement and register other execution tools

	log.Info().Msg("Execution tools registered")
	return nil
}

// RegisterEnvTools registers all environment-related tools with the MCP server
func RegisterEnvTools(server *mcp.Server, vmManager *vm.Manager, executor *exec.Executor) error {
	// Import environment tools from env-tools.go
	// Register setup environment tool
	setupTool := &SetupDevEnvironmentTool{
		manager:  vmManager,
		executor: executor,
	}
	if err := server.RegisterTool(setupTool); err != nil {
		return fmt.Errorf("failed to register setup_dev_environment tool: %w", err)
	}

	// Register install tools tool
	installTool := &InstallDevToolsTool{
		manager:  vmManager,
		executor: executor,
	}
	if err := server.RegisterTool(installTool); err != nil {
		return fmt.Errorf("failed to register install_dev_tools tool: %w", err)
	}

	// Register configure shell tool
	shellTool := &ConfigureShellTool{
		manager:  vmManager,
		executor: executor,
	}
	if err := server.RegisterTool(shellTool); err != nil {
		return fmt.Errorf("failed to register configure_shell tool: %w", err)
	}

	log.Info().Msg("Environment tools registered")
	return nil
}

// RegisterSyncTools registers all synchronization-related tools with the MCP server
func RegisterSyncTools(server *mcp.Server, vmManager *vm.Manager, syncEngine *sync.Engine) error {
	// Register configure sync tool
	configTool := &ConfigureSyncTool{
		vmManager:  vmManager,
		syncEngine: syncEngine,
	}
	if err := server.RegisterTool(configTool); err != nil {
		return fmt.Errorf("failed to register configure_sync tool: %w", err)
	}

	// Register sync to VM tool
	toVMTool := &SyncToVMTool{
		vmManager:  vmManager,
		syncEngine: syncEngine,
	}
	if err := server.RegisterTool(toVMTool); err != nil {
		return fmt.Errorf("failed to register sync_to_vm tool: %w", err)
	}

	// Register sync from VM tool
	fromVMTool := &SyncFromVMTool{
		vmManager:  vmManager,
		syncEngine: syncEngine,
	}
	if err := server.RegisterTool(fromVMTool); err != nil {
		return fmt.Errorf("failed to register sync_from_vm tool: %w", err)
	}

	// Register sync status tool
	statusTool := &SyncStatusTool{
		vmManager:  vmManager,
		syncEngine: syncEngine,
	}
	if err := server.RegisterTool(statusTool); err != nil {
		return fmt.Errorf("failed to register sync_status tool: %w", err)
	}

	log.Info().Msg("Synchronization tools registered")
	return nil
}
