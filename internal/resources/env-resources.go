package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/vagrant-mcp/server/internal/exec"
	"github.com/vagrant-mcp/server/internal/vm"
	"github.com/vagrant-mcp/server/pkg/mcp"
)

// VMEnvironmentResource provides access to VM environment variables
type VMEnvironmentResource struct {
	vmManager *vm.Manager
	executor  *exec.Executor
}

// NewVMEnvironmentResource creates a new VM environment resource
func NewVMEnvironmentResource(vmManager *vm.Manager, executor *exec.Executor) *VMEnvironmentResource {
	return &VMEnvironmentResource{
		vmManager: vmManager,
		executor:  executor,
	}
}

// Name returns the resource name
func (r *VMEnvironmentResource) Name() string {
	return "devvm://env"
}

// Description returns the resource description
func (r *VMEnvironmentResource) Description() string {
	return "Environment variables and PATH inside VM"
}

// Get retrieves environment variables from the VM
func (r *VMEnvironmentResource) Get(path string) (interface{}, error) {
	// Parse VM name from path
	// Expected format: devvm://env?vm={vmName}
	parts := strings.Split(path, "?")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid path format, expected devvm://env?vm={vmName}")
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

	// Execute command to get environment variables
	execCtx := exec.ExecutionContext{
		VMName:     vmName,
		WorkingDir: "/home/vagrant",
		SyncBefore: false,
		SyncAfter:  false,
	}

	result, err := r.executor.ExecuteCommand(context.Background(), "printenv | sort", execCtx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to execute environment command: %w", err)
	}

	// Parse environment variables
	environment := make(map[string]string)
	for _, line := range strings.Split(result.Stdout, "\n") {
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			environment[parts[0]] = parts[1]
		}
	}

	// Get PATH separately for better structure
	result, err = r.executor.ExecuteCommand(context.Background(), "echo $PATH", execCtx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get PATH: %w", err)
	}

	pathEntries := strings.Split(strings.TrimSpace(result.Stdout), ":")

	return map[string]interface{}{
		"vm_name":     vmName,
		"environment": environment,
		"path":        pathEntries,
	}, nil
}

// InstalledToolsResource provides information about installed development tools
type InstalledToolsResource struct {
	vmManager *vm.Manager
	executor  *exec.Executor
}

// NewInstalledToolsResource creates a new installed tools resource
func NewInstalledToolsResource(vmManager *vm.Manager, executor *exec.Executor) *InstalledToolsResource {
	return &InstalledToolsResource{
		vmManager: vmManager,
		executor:  executor,
	}
}

// Name returns the resource name
func (r *InstalledToolsResource) Name() string {
	return "devvm://installed-tools"
}

// Description returns the resource description
func (r *InstalledToolsResource) Description() string {
	return "List of installed development tools and their versions"
}

// Get retrieves information about installed tools
func (r *InstalledToolsResource) Get(path string) (interface{}, error) {
	// Parse VM name from path
	// Expected format: devvm://installed-tools?vm={vmName}
	parts := strings.Split(path, "?")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid path format, expected devvm://installed-tools?vm={vmName}")
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

	// Setup execution context
	execCtx := exec.ExecutionContext{
		VMName:     vmName,
		WorkingDir: "/home/vagrant",
		SyncBefore: false,
		SyncAfter:  false,
	}

	// Commands to get tool versions
	commands := map[string]string{
		"nodejs":   "node --version 2>/dev/null || echo 'not installed'",
		"npm":      "npm --version 2>/dev/null || echo 'not installed'",
		"python2":  "python2 --version 2>&1 || echo 'not installed'",
		"python3":  "python3 --version 2>&1 || echo 'not installed'",
		"pip":      "pip --version 2>/dev/null || echo 'not installed'",
		"ruby":     "ruby --version 2>/dev/null || echo 'not installed'",
		"gem":      "gem --version 2>/dev/null || echo 'not installed'",
		"go":       "go version 2>/dev/null || echo 'not installed'",
		"java":     "java -version 2>&1 || echo 'not installed'",
		"docker":   "docker --version 2>/dev/null || echo 'not installed'",
		"postgres": "psql --version 2>/dev/null || echo 'not installed'",
		"mysql":    "mysql --version 2>/dev/null || echo 'not installed'",
		"git":      "git --version 2>/dev/null || echo 'not installed'",
		"gcc":      "gcc --version 2>/dev/null | head -1 || echo 'not installed'",
		"make":     "make --version 2>/dev/null | head -1 || echo 'not installed'",
	}

	// Collect tool information
	tools := make(map[string]interface{})
	for tool, cmd := range commands {
		result, err := r.executor.ExecuteCommand(context.Background(), cmd, execCtx, nil)
		if err != nil {
			tools[tool] = map[string]interface{}{
				"installed": false,
				"error":     err.Error(),
			}
			continue
		}

		output := strings.TrimSpace(result.Stdout)
		if output == "" {
			output = strings.TrimSpace(result.Stderr)
		}

		if strings.Contains(output, "not installed") || result.ExitCode != 0 {
			tools[tool] = map[string]interface{}{
				"installed": false,
			}
		} else {
			tools[tool] = map[string]interface{}{
				"installed": true,
				"version":   output,
			}
		}
	}

	return map[string]interface{}{
		"vm_name": vmName,
		"tools":   tools,
	}, nil
}

// RegisterEnvironmentResources registers all environment-related resources with the MCP server
func RegisterEnvironmentResources(server interface{}, vmManager *vm.Manager, executor interface{}) error {
	mcpServer, ok := server.(*mcp.Server)
	if !ok {
		return fmt.Errorf("server is not *mcp.Server")
	}

	execExecutor, ok := executor.(*exec.Executor)
	if !ok {
		return fmt.Errorf("executor is not of type *exec.Executor")
	}

	// Register environment resource
	environmentResource := NewVMEnvironmentResource(vmManager, execExecutor)
	if err := mcpServer.RegisterResource(environmentResource); err != nil {
		return fmt.Errorf("failed to register environment resource: %w", err)
	}

	// Register installed tools resource
	installedToolsResource := NewInstalledToolsResource(vmManager, execExecutor)
	if err := mcpServer.RegisterResource(installedToolsResource); err != nil {
		return fmt.Errorf("failed to register installed tools resource: %w", err)
	}

	return nil
}
