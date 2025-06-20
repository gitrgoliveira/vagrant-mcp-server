package tools

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/vagrant-mcp/server/internal/sync"
	"github.com/vagrant-mcp/server/internal/vm"
	"github.com/vagrant-mcp/server/pkg/mcp"
)

// CreateDevVMTool implements the create_dev_vm tool
type CreateDevVMTool struct {
	manager *vm.Manager
}

// Name returns the tool name
func (t *CreateDevVMTool) Name() string {
	return "create_dev_vm"
}

// Description returns the tool description
func (t *CreateDevVMTool) Description() string {
	return "Create and configure a development VM with Vagrant"
}

// Execute performs the tool action
func (t *CreateDevVMTool) Execute(params map[string]interface{}) (interface{}, error) {
	// Validate required parameters
	name, ok := params["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("missing or invalid 'name' parameter")
	}

	projectPath, ok := params["project_path"].(string)
	if !ok || projectPath == "" {
		return nil, fmt.Errorf("missing or invalid 'project_path' parameter")
	}

	// Get optional parameters with defaults
	cpu := 2
	if cpuParam, ok := params["cpu"].(float64); ok {
		cpu = int(cpuParam)
	}

	memory := 2048
	if memoryParam, ok := params["memory"].(float64); ok {
		memory = int(memoryParam)
	}

	box := "ubuntu/focal64"
	if boxParam, ok := params["box"].(string); ok && boxParam != "" {
		box = boxParam
	}

	syncType := "rsync"
	if syncTypeParam, ok := params["sync_type"].(string); ok && syncTypeParam != "" {
		syncType = syncTypeParam
	}

	// Configure port forwarding
	ports := []vm.Port{
		{Guest: 3000, Host: 3000},   // Default for Node.js
		{Guest: 8000, Host: 8000},   // Default for Python/Django
		{Guest: 5432, Host: 5432},   // PostgreSQL
		{Guest: 3306, Host: 3306},   // MySQL
		{Guest: 6379, Host: 6379},   // Redis
		{Guest: 27017, Host: 27017}, // MongoDB
	}

	// Process custom port forwarding if provided
	if portsParam, ok := params["ports"].([]interface{}); ok {
		ports = []vm.Port{} // Reset default ports
		for _, portObj := range portsParam {
			if portMap, ok := portObj.(map[string]interface{}); ok {
				guestPort, guestOk := portMap["guest"].(float64)
				hostPort, hostOk := portMap["host"].(float64)

				if guestOk && hostOk {
					ports = append(ports, vm.Port{
						Guest: int(guestPort),
						Host:  int(hostPort),
					})
				}
			}
		}
	}

	// Process environment variables
	environment := []string{}
	if envObj, ok := params["environment"].([]interface{}); ok {
		for _, envEntry := range envObj {
			if envStr, ok := envEntry.(string); ok {
				environment = append(environment, envStr)
			}
		}
	}

	// Process provisioners
	provisioners := []string{}
	if provObj, ok := params["provisioners"].([]interface{}); ok {
		for _, provEntry := range provObj {
			if provStr, ok := provEntry.(string); ok {
				provisioners = append(provisioners, provStr)
			}
		}
	} else {
		// Default provisioners for development
		provisioners = []string{
			"base",      // Basic tools
			"dev_tools", // Development tools
		}
	}

	// Create config
	config := vm.VMConfig{
		Name:         name,
		Box:          box,
		CPU:          cpu,
		Memory:       memory,
		SyncType:     syncType,
		ProjectPath:  projectPath,
		Ports:        ports,
		Environment:  environment,
		Provisioners: provisioners,
	}

	// Create VM
	log.Info().
		Str("vm", name).
		Str("box", box).
		Int("cpu", cpu).
		Int("memory", memory).
		Str("sync", syncType).
		Str("project", projectPath).
		Msg("Creating development VM")

	if err := t.manager.CreateVM(name, projectPath, config); err != nil {
		return nil, fmt.Errorf("failed to create VM: %w", err)
	}

	// Start VM
	log.Info().Str("vm", name).Msg("Starting development VM")
	if err := t.manager.StartVM(name); err != nil {
		return nil, fmt.Errorf("failed to start VM: %w", err)
	}

	// Get VM state to return
	state, err := t.manager.GetVMState(name)
	if err != nil {
		log.Error().Err(err).Str("vm", name).Msg("Failed to get VM state after creation")
	}

	// Return status
	return map[string]interface{}{
		"name":         name,
		"state":        state,
		"project_path": projectPath,
		"box":          box,
		"cpu":          cpu,
		"memory":       memory,
		"sync_type":    syncType,
	}, nil
}

// EnsureDevVMTool implements the ensure_dev_vm tool
type EnsureDevVMTool struct {
	manager    *vm.Manager
	syncEngine *sync.Engine
}

// Name returns the tool name
func (t *EnsureDevVMTool) Name() string {
	return "ensure_dev_vm"
}

// Description returns the tool description
func (t *EnsureDevVMTool) Description() string {
	return "Ensure development VM is running, create if it doesn't exist"
}

// Execute performs the tool action
func (t *EnsureDevVMTool) Execute(params map[string]interface{}) (interface{}, error) {
	// Validate required parameters
	name, ok := params["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("missing or invalid 'name' parameter")
	}

	// Check if VM exists
	state, err := t.manager.GetVMState(name)
	if err != nil {
		// VM doesn't exist, create it if project_path is provided
		projectPath, ok := params["project_path"].(string)
		if !ok || projectPath == "" {
			return nil, fmt.Errorf("VM '%s' does not exist and 'project_path' not provided for creation", name)
		}

		// Forward to create_dev_vm tool
		createTool := &CreateDevVMTool{
			manager: t.manager,
		}
		return createTool.Execute(params)
	}

	// VM exists, ensure it's running
	if state != vm.StateRunning {
		log.Info().Str("vm", name).Msg("Starting existing development VM")
		if err := t.manager.StartVM(name); err != nil {
			return nil, fmt.Errorf("failed to start VM: %w", err)
		}

		// Get updated state
		state, err = t.manager.GetVMState(name)
		if err != nil {
			log.Error().Err(err).Str("vm", name).Msg("Failed to get VM state after starting")
		}
	}

	// Get VM config
	config, err := t.manager.GetVMConfig(name)
	if err != nil {
		return nil, fmt.Errorf("failed to get VM config: %w", err)
	}

	// Sync files if VM is running
	if state == vm.StateRunning && config.ProjectPath != "" {
		log.Info().Str("vm", name).Msg("Syncing files to ensure VM is up to date")

		syncConfig := sync.SyncConfig{
			VMName:          name,
			ProjectPath:     config.ProjectPath,
			Method:          sync.SyncMethod(config.SyncType),
			Direction:       sync.SyncToVM,
			ExcludePatterns: []string{".git", "node_modules", "dist", ".vagrant", "__pycache__", "*.pyc"},
		}

		_, err := t.syncEngine.Sync(syncConfig)
		if err != nil {
			log.Error().Err(err).Str("vm", name).Msg("Failed to sync files, continuing anyway")
		}
	}

	// Return status
	return map[string]interface{}{
		"name":         name,
		"state":        state,
		"project_path": config.ProjectPath,
		"box":          config.Box,
		"cpu":          config.CPU,
		"memory":       config.Memory,
		"sync_type":    config.SyncType,
	}, nil
}

// DestroyDevVMTool implements the destroy_dev_vm tool
type DestroyDevVMTool struct {
	manager *vm.Manager
}

// Name returns the tool name
func (t *DestroyDevVMTool) Name() string {
	return "destroy_dev_vm"
}

// Description returns the tool description
func (t *DestroyDevVMTool) Description() string {
	return "Destroy a development VM and clean up resources"
}

// Execute performs the tool action
func (t *DestroyDevVMTool) Execute(params map[string]interface{}) (interface{}, error) {
	// Validate required parameters
	name, ok := params["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("missing or invalid 'name' parameter")
	}

	// Check if VM exists
	_, err := t.manager.GetVMState(name)
	if err != nil {
		return nil, fmt.Errorf("VM '%s' does not exist or cannot be accessed: %w", name, err)
	}

	// Destroy VM
	log.Info().Str("vm", name).Msg("Destroying development VM")
	if err := t.manager.DestroyVM(name); err != nil {
		return nil, fmt.Errorf("failed to destroy VM: %w", err)
	}

	// Return success
	return map[string]interface{}{
		"name":      name,
		"destroyed": true,
	}, nil
}

// RegisterVMTools registers all VM-related tools with the MCP server
func RegisterVMTools(server *mcp.Server, manager *vm.Manager, syncEngine *sync.Engine) error {
	// Create VM tools
	createVMTool := &CreateDevVMTool{manager: manager}
	ensureVMTool := &EnsureDevVMTool{manager: manager, syncEngine: syncEngine}
	destroyVMTool := &DestroyDevVMTool{manager: manager}

	// Register tools
	if err := server.RegisterTool(createVMTool); err != nil {
		return fmt.Errorf("failed to register create_dev_vm tool: %w", err)
	}

	if err := server.RegisterTool(ensureVMTool); err != nil {
		return fmt.Errorf("failed to register ensure_dev_vm tool: %w", err)
	}

	if err := server.RegisterTool(destroyVMTool); err != nil {
		return fmt.Errorf("failed to register destroy_dev_vm tool: %w", err)
	}

	log.Info().Msg("VM management tools registered")
	return nil
}
