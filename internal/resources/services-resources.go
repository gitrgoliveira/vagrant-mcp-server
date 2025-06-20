package resources

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/vagrant-mcp/server/internal/exec"
	"github.com/vagrant-mcp/server/internal/vm"
	"github.com/vagrant-mcp/server/pkg/mcp"
)

// ServicesResource provides information about services running in the VM
type ServicesResource struct {
	vmManager *vm.Manager
	executor  *exec.Executor
}

// NewServicesResource creates a new services resource
func NewServicesResource(vmManager *vm.Manager, executor *exec.Executor) *ServicesResource {
	return &ServicesResource{
		vmManager: vmManager,
		executor:  executor,
	}
}

// Name returns the resource name
func (r *ServicesResource) Name() string {
	return "devvm://services"
}

// Description returns the resource description
func (r *ServicesResource) Description() string {
	return "Status of development services"
}

// Get retrieves information about running services
func (r *ServicesResource) Get(path string) (interface{}, error) {
	// Parse VM name from path
	// Expected format: devvm://services?vm={vmName}
	parts := strings.Split(path, "?")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid path format, expected devvm://services?vm={vmName}")
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

	// Common services to check
	commonServices := []string{
		"ssh", "docker", "nginx", "apache2", "httpd",
		"postgresql", "mysql", "mongodb", "redis-server",
		"rabbitmq-server", "elasticsearch",
	}

	// Service detection commands
	// First check if we have systemctl
	hasSystemd := false
	result, err := r.executor.ExecuteCommand(context.Background(), "which systemctl", execCtx, nil)
	if err == nil && result.ExitCode == 0 {
		hasSystemd = true
	}

	services := make([]map[string]interface{}, 0)

	// Check each service
	for _, serviceName := range commonServices {
		var statusCmd string
		var portCmd string

		if hasSystemd {
			statusCmd = fmt.Sprintf("systemctl is-active %s 2>/dev/null || echo inactive", serviceName)
		} else {
			statusCmd = fmt.Sprintf("service %s status 2>/dev/null | grep -q 'running\\|start' && echo active || echo inactive", serviceName)
		}

		// Execute status command
		result, err := r.executor.ExecuteCommand(context.Background(), statusCmd, execCtx, nil)
		if err != nil {
			continue // Skip this service if command fails
		}

		status := strings.TrimSpace(result.Stdout)

		// If service is running, try to get its port
		var port *int
		if status == "active" {
			// Different commands for different services
			switch serviceName {
			case "ssh":
				portCmd = "grep -oP '^Port\\s+\\K\\d+' /etc/ssh/sshd_config 2>/dev/null || echo 22"
			case "nginx", "apache2", "httpd":
				portCmd = fmt.Sprintf("grep -r 'listen' /etc/%s/sites-enabled/ 2>/dev/null | grep -oP 'listen\\s+\\K\\d+' | head -1 || echo 80", serviceName)
			case "postgresql":
				portCmd = "grep -oP '^port\\s*=\\s*\\K\\d+' /etc/postgresql/*/main/postgresql.conf 2>/dev/null || echo 5432"
			case "mysql":
				portCmd = "grep -oP '^port\\s*=\\s*\\K\\d+' /etc/mysql/my.cnf 2>/dev/null || echo 3306"
			case "mongodb":
				portCmd = "grep -oP '^\\s*port:\\s*\\K\\d+' /etc/mongod.conf 2>/dev/null || echo 27017"
			case "redis-server":
				portCmd = "grep -oP '^port\\s+\\K\\d+' /etc/redis/redis.conf 2>/dev/null || echo 6379"
			case "rabbitmq-server":
				portCmd = "grep -oP '^\\s*tcp_listeners\\s*,\\s*\\[\\{\".*\"\\s*,\\s*\\K\\d+' /etc/rabbitmq/rabbitmq.config 2>/dev/null || echo 5672"
			case "elasticsearch":
				portCmd = "grep -oP '^\\s*http\\.port:\\s*\\K\\d+' /etc/elasticsearch/elasticsearch.yml 2>/dev/null || echo 9200"
			default:
				portCmd = ""
			}

			if portCmd != "" {
				result, err := r.executor.ExecuteCommand(context.Background(), portCmd, execCtx, nil)
				if err == nil && result.ExitCode == 0 {
					portStr := strings.TrimSpace(result.Stdout)
					if portStr != "" {
						portNum, err := strconv.Atoi(portStr)
						if err == nil {
							port = &portNum
						}
					}
				}
			}
		}

		// Only include services that are active or ones we explicitly checked
		if status == "active" {
			serviceInfo := map[string]interface{}{
				"name":   serviceName,
				"status": status,
			}

			if port != nil {
				serviceInfo["port"] = *port
			}

			services = append(services, serviceInfo)
		}
	}

	return map[string]interface{}{
		"vm_name":       vmName,
		"services":      services,
		"service_count": len(services),
	}, nil
}

// RegisterServicesResources registers all services-related resources with the MCP server
func RegisterServicesResources(server interface{}, vmManager *vm.Manager, executor interface{}) error {
	mcpServer, ok := server.(*mcp.Server)
	if !ok {
		return fmt.Errorf("server is not *mcp.Server")
	}

	execExecutor, ok := executor.(*exec.Executor)
	if !ok {
		return fmt.Errorf("executor is not of type *exec.Executor")
	}

	// Register services resource
	servicesResource := NewServicesResource(vmManager, execExecutor)
	if err := mcpServer.RegisterResource(servicesResource); err != nil {
		return fmt.Errorf("failed to register services resource: %w", err)
	}

	return nil
}
