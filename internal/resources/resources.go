package resources

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/vagrant-mcp/server/internal/exec"
	"github.com/vagrant-mcp/server/internal/vm"
	"github.com/vagrant-mcp/server/pkg/mcp"
)

// VMStatusResource provides VM status information
type VMStatusResource struct {
	vmManager *vm.Manager
}

// NewVMStatusResource creates a new VM status resource
func NewVMStatusResource(vmManager *vm.Manager) *VMStatusResource {
	return &VMStatusResource{
		vmManager: vmManager,
	}
}

// Name returns the resource name
func (r *VMStatusResource) Name() string {
	return "devvm://status"
}

// Description returns the resource description
func (r *VMStatusResource) Description() string {
	return "Current development VM status and health"
}

// Get retrieves the VM status
func (r *VMStatusResource) Get(path string) (interface{}, error) {
	// Extract VM name from path
	vmName, err := parseVMNameFromPath(path)
	if err != nil {
		return nil, err
	}

	// Get VM state
	state, err := r.vmManager.GetVMState(vmName)
	if err != nil {
		return nil, fmt.Errorf("failed to get VM state: %w", err)
	}

	// Build status response
	status := map[string]interface{}{
		"vm_name": vmName,
		"state":   state,
	}

	// Add additional details if VM is running
	if state == vm.StateRunning {
		config, err := r.vmManager.GetVMConfig(vmName)
		if err == nil {
			status["config"] = map[string]interface{}{
				"box":       config.Box,
				"cpu":       config.CPU,
				"memory":    config.Memory,
				"sync_type": config.SyncType,
			}
		}

		sshConfig, err := r.vmManager.GetSSHConfig(vmName)
		if err == nil {
			status["ssh"] = map[string]interface{}{
				"host":     sshConfig["HostName"],
				"port":     sshConfig["Port"],
				"username": sshConfig["User"],
			}
		}
	}

	return status, nil
}

// VMConfigResource provides VM configuration information
type VMConfigResource struct {
	vmManager *vm.Manager
}

// NewVMConfigResource creates a new VM config resource
func NewVMConfigResource(vmManager *vm.Manager) *VMConfigResource {
	return &VMConfigResource{
		vmManager: vmManager,
	}
}

// Name returns the resource name
func (r *VMConfigResource) Name() string {
	return "devvm://config"
}

// Description returns the resource description
func (r *VMConfigResource) Description() string {
	return "Current VM configuration and sync settings"
}

// Get retrieves the VM configuration
func (r *VMConfigResource) Get(path string) (interface{}, error) {
	// Extract VM name from path
	vmName, err := parseVMNameFromPath(path)
	if err != nil {
		return nil, err
	}

	// Get VM configuration
	config, err := r.vmManager.GetVMConfig(vmName)
	if err != nil {
		return nil, fmt.Errorf("failed to get VM config: %w", err)
	}

	// Return config
	return config, nil
}

// RegisterVMResources registers all VM-related resources with the MCP server
func RegisterVMResources(server *mcp.Server, vmManager *vm.Manager) error {
	// Create resources
	statusResource := NewVMStatusResource(vmManager)
	configResource := NewVMConfigResource(vmManager)

	// Register resources
	if err := server.RegisterResource(statusResource); err != nil {
		return fmt.Errorf("failed to register VM status resource: %w", err)
	}

	if err := server.RegisterResource(configResource); err != nil {
		return fmt.Errorf("failed to register VM config resource: %w", err)
	}

	log.Info().Msg("VM resources registered")
	return nil
}

// RegisterLogResources registers all log-related resources with the MCP server
func RegisterLogResources(server *mcp.Server) error {
	// Log resources are registered in RegisterMonitoringResources to avoid circular dependencies
	log.Info().Msg("Log resources registration deferred to monitoring resources")
	return nil
}

// RegisterNetworkResources registers all network-related resources with the MCP server
func RegisterNetworkResources(server *mcp.Server, vmManager *vm.Manager) error {
	// Register network resource
	networkResource := &NetworkResource{
		vmManager: vmManager,
	}
	if err := server.RegisterResource(networkResource); err != nil {
		return fmt.Errorf("failed to register network resource: %w", err)
	}

	log.Info().Msg("Network resources registered")
	return nil
}

// RegisterMonitoringResources registers all monitoring-related resources with the MCP server
func RegisterMonitoringResources(server *mcp.Server, vmManager *vm.Manager, executor interface{}) error {
	// Convert executor to proper type
	execExecutor, ok := executor.(*exec.Executor)
	if !ok {
		log.Warn().Msg("Executor is not of type *exec.Executor, monitoring resources will be limited")
		return nil
	}

	// Register monitoring resource
	monitoringResource := &MonitoringResource{
		vmManager: vmManager,
		executor:  execExecutor,
	}
	if err := server.RegisterResource(monitoringResource); err != nil {
		return fmt.Errorf("failed to register monitoring resource: %w", err)
	}

	// Register log resource
	logsResource := &LogsResource{
		vmManager: vmManager,
		executor:  execExecutor,
	}
	if err := server.RegisterResource(logsResource); err != nil {
		return fmt.Errorf("failed to register logs resource: %w", err)
	}

	log.Info().Msg("Monitoring and log resources registered")
	return nil
}

// parseVMNameFromPath extracts VM name from resource path
func parseVMNameFromPath(path string) (string, error) {
	// If path is empty, return error
	if path == "" {
		return "", fmt.Errorf("VM name not specified in path")
	}

	// If path starts with /, remove it
	if path[0] == '/' {
		path = path[1:]
	}

	return path, nil
}
