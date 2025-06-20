package resources

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/vagrant-mcp/server/internal/exec"
	"github.com/vagrant-mcp/server/internal/vm"
)

// LogsResource provides access to VM logs
type LogsResource struct {
	vmManager *vm.Manager
	executor  *exec.Executor
}

// NewLogsResource creates a new logs resource
func NewLogsResource(vmManager *vm.Manager, executor *exec.Executor) *LogsResource {
	return &LogsResource{
		vmManager: vmManager,
		executor:  executor,
	}
}

// Name returns the resource name
func (r *LogsResource) Name() string {
	return "devvm://logs/{type}"
}

// Description returns the resource description
func (r *LogsResource) Description() string {
	return "Access to VM logs (sync, provisioning)"
}

// Get retrieves logs based on the requested type
func (r *LogsResource) Get(path string) (interface{}, error) {
	// Parse VM name and log type from path
	// Expected format: devvm://logs/{type}?vm={vmName}
	parts := strings.Split(path, "?")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid path format, expected devvm://logs/{type}?vm={vmName}")
	}

	// Extract log type
	logType := strings.TrimPrefix(parts[0], "devvm://logs/")

	// Extract VM name from query params
	queryParams := strings.Split(parts[1], "&")
	vmName := ""
	for _, param := range queryParams {
		if strings.HasPrefix(param, "vm=") {
			vmName = strings.TrimPrefix(param, "vm=")
			break
		}
	}

	if vmName == "" {
		return nil, fmt.Errorf("missing vm parameter")
	}

	// Validate VM exists
	state, err := r.vmManager.GetVMState(vmName)
	if err != nil {
		return nil, fmt.Errorf("failed to get VM state: %w", err)
	}

	// Depending on log type, retrieve the appropriate logs
	var logContent string

	switch logType {
	case "sync":
		// For sync logs, we'll log events from our own sync operations
		logContent = "Sync logs are not yet implemented in this version"
	case "provisioning":
		// For provisioning logs, read from the Vagrant log file
		if state == vm.StateRunning || state == vm.StateStopped {
			// Use a standard or documented location for the VM directory, or fallback to a placeholder
			logContent = "Provisioning logs would be read from /vagrant/provision.log or a similar path inside the VM."
		} else {
			logContent = fmt.Sprintf("VM is not in a state where logs can be accessed (current state: %s)", state)
		}
	default:
		return nil, fmt.Errorf("unsupported log type: %s", logType)
	}

	return map[string]interface{}{
		"vm_name":   vmName,
		"log_type":  logType,
		"content":   logContent,
		"timestamp": time.Now().Format(time.RFC3339),
	}, nil
}

// NetworkResource provides network information about the VM
type NetworkResource struct {
	vmManager *vm.Manager
	executor  *exec.Executor
}

// NewNetworkResource creates a new network resource
func NewNetworkResource(vmManager *vm.Manager, executor *exec.Executor) *NetworkResource {
	return &NetworkResource{
		vmManager: vmManager,
		executor:  executor,
	}
}

// Name returns the resource name
func (r *NetworkResource) Name() string {
	return "devvm://network"
}

// Description returns the resource description
func (r *NetworkResource) Description() string {
	return "Network configuration and connectivity"
}

// Get retrieves network information
func (r *NetworkResource) Get(path string) (interface{}, error) {
	// Parse VM name from path
	// Expected format: devvm://network?vm={vmName}
	parts := strings.Split(path, "?")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid path format, expected devvm://network?vm={vmName}")
	}

	// Extract VM name from query params
	queryParams := strings.Split(parts[1], "&")
	vmName := ""
	for _, param := range queryParams {
		if strings.HasPrefix(param, "vm=") {
			vmName = strings.TrimPrefix(param, "vm=")
			break
		}
	}

	if vmName == "" {
		return nil, fmt.Errorf("missing vm parameter")
	}

	// Check VM state
	state, err := r.vmManager.GetVMState(vmName)
	if err != nil {
		return nil, fmt.Errorf("failed to get VM state: %w", err)
	}

	if state != vm.StateRunning {
		return nil, fmt.Errorf("VM is not running (current state: %s)", state)
	}

	// Get SSH config which contains network information
	config, err := r.vmManager.GetSSHConfig(vmName)
	if err != nil {
		return nil, fmt.Errorf("failed to get SSH config: %w", err)
	}

	// Return network information
	return map[string]interface{}{
		"vm_name":  vmName,
		"hostname": config["HostName"],
		"port":     config["Port"],
		// "forwarded_ports": []interface{}{}, // Omitted or empty for now
	}, nil
}

// MonitoringResource provides monitoring information about the VM
type MonitoringResource struct {
	vmManager *vm.Manager
	executor  *exec.Executor
}

// NewMonitoringResource creates a new monitoring resource
func NewMonitoringResource(vmManager *vm.Manager, executor *exec.Executor) *MonitoringResource {
	return &MonitoringResource{
		vmManager: vmManager,
		executor:  executor,
	}
}

// Name returns the resource name
func (r *MonitoringResource) Name() string {
	return "devvm://monitoring/{metric}"
}

// Description returns the resource description
func (r *MonitoringResource) Description() string {
	return "VM monitoring metrics (cpu, memory, disk, processes)"
}

// Get retrieves monitoring metrics
func (r *MonitoringResource) Get(path string) (interface{}, error) {
	// Parse VM name and metric type from path
	// Expected format: devvm://monitoring/{metric}?vm={vmName}
	parts := strings.Split(path, "?")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid path format, expected devvm://monitoring/{metric}?vm={vmName}")
	}

	// Extract metric type
	metricType := strings.TrimPrefix(parts[0], "devvm://monitoring/")

	// Extract VM name from query params
	queryParams := strings.Split(parts[1], "&")
	vmName := ""
	for _, param := range queryParams {
		if strings.HasPrefix(param, "vm=") {
			vmName = strings.TrimPrefix(param, "vm=")
			break
		}
	}

	if vmName == "" {
		return nil, fmt.Errorf("missing vm parameter")
	}

	// Check VM state
	state, err := r.vmManager.GetVMState(vmName)
	if err != nil {
		return nil, fmt.Errorf("failed to get VM state: %w", err)
	}

	if state != vm.StateRunning {
		return nil, fmt.Errorf("VM is not running (current state: %s)", state)
	}

	// Command to execute based on metric type
	var cmd string

	switch metricType {
	case "cpu":
		cmd = "top -bn1 | grep 'Cpu(s)' | awk '{print $2 + $4}'"
	case "memory":
		cmd = "free -m | grep Mem | awk '{print $3/$2 * 100.0}'"
	case "disk":
		cmd = "df -h / | grep / | awk '{print $5}'"
	case "processes":
		cmd = "ps aux | head -10"
	default:
		return nil, fmt.Errorf("unsupported metric type: %s", metricType)
	}

	// Execute command in VM
	execCtx := exec.ExecutionContext{
		VMName:     vmName,
		WorkingDir: "/home/vagrant",
		SyncBefore: false,
		SyncAfter:  false,
	}

	// Execute the command to get metrics
	result, err := r.executor.ExecuteCommand(context.Background(), cmd, execCtx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to execute monitoring command: %w", err)
	}

	// Return monitoring information
	return map[string]interface{}{
		"vm_name":     vmName,
		"metric_type": metricType,
		"value":       strings.TrimSpace(result.Stdout),
		"timestamp":   time.Now().Format(time.RFC3339),
	}, nil
}
