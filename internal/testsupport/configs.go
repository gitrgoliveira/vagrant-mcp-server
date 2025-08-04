// Package testsupport provides common test configurations
package testsupport

import (
	"os"
	"strconv"
	"time"

	"github.com/vagrant-mcp/server/internal/config"
	"github.com/vagrant-mcp/server/internal/core"
)

// VMConfigs provides standard VM configurations for tests
var VMConfigs = struct {
	// Minimal is a minimal VM configuration suitable for basic tests
	Minimal core.VMConfig
	// Standard is a standard VM configuration with reasonable defaults (renamed from Default)
	Standard core.VMConfig
	// Dev is a VM configuration for development environment tests (renamed from DevEnvironment)
	Dev core.VMConfig
	// CI is a VM configuration optimized for CI environments
	CI core.VMConfig
}{
	Minimal: core.VMConfig{
		Box:         "generic/alpine314",
		CPU:         1,
		Memory:      512,
		SyncType:    "rsync",
		Ports:       []core.Port{{Guest: 8080, Host: 8080}},
		Environment: []string{},
	},
	Standard: core.VMConfig{
		Box:         "ubuntu/focal64",
		CPU:         2,
		Memory:      1024,
		SyncType:    "rsync",
		Ports:       []core.Port{{Guest: 3000, Host: 3000}, {Guest: 8080, Host: 8080}},
		Environment: []string{"TERM=xterm"},
		SyncExcludePatterns: []string{
			"node_modules", ".git", "*.log", "dist", "build",
			"__pycache__", "*.pyc", "venv", ".venv",
		},
	},
	Dev: core.VMConfig{
		Box:      "ubuntu/focal64",
		CPU:      4,
		Memory:   4096,
		SyncType: "rsync",
		Ports: []core.Port{
			{Guest: 3000, Host: 3000}, // Node.js/React
			{Guest: 8000, Host: 8000}, // Python/Django
			{Guest: 5432, Host: 5432}, // PostgreSQL
			{Guest: 3306, Host: 3306}, // MySQL
			{Guest: 6379, Host: 6379}, // Redis
		},
		Environment: []string{"TERM=xterm", "LANG=C.UTF-8"},
		Provisioners: []string{
			"apt-get install -y build-essential git curl unzip",
			"apt-get install -y python3 python3-pip",
		},
		SyncExcludePatterns: []string{
			"node_modules", ".git", "*.log", "dist", "build",
			"__pycache__", "*.pyc", "venv", ".venv",
			"*.o", "*.out", "*.class", "target/", ".cache/",
		},
	},
	CI: core.VMConfig{
		Box:         "generic/alpine314", // Use small box for CI
		CPU:         1,
		Memory:      512,
		SyncType:    "rsync",
		Ports:       []core.Port{{Guest: 8080, Host: 8080}},
		Environment: []string{"CI=true"},
		SyncExcludePatterns: []string{
			"node_modules", ".git", "*.log", "dist", "build",
		},
	},
}

// GetTestProjectPath returns a project path for tests
func GetTestProjectPath() string {
	path, _ := os.MkdirTemp("", "vagrant-mcp-test-project-*")
	return path
}

// SanitizeVMConfig creates a copy of the VM config with appropriate values set for testing
func SanitizeVMConfig(config core.VMConfig, projectPath string) core.VMConfig {
	// Create a copy to avoid modifying the original
	result := config

	// Ensure project path is set
	if projectPath != "" {
		result.ProjectPath = projectPath
	}

	// Ensure unique VM name if not provided
	if result.Name == "" {
		result.Name = "test-vm-" + strconv.FormatInt(time.Now().UnixNano(), 36)
	}

	return result
}

// GetVMConfig returns a copy of the requested VM config with the project path set
func GetVMConfig(configType string, projectPath string) core.VMConfig {
	vmConfig, err := config.GlobalVMRegistry.GetConfigWithDefaults(configType, projectPath, nil)
	if err != nil {
		// Fall back to minimal configuration
		vmConfig, _ = config.GlobalVMRegistry.GetConfig("minimal")
		vmConfig = SanitizeVMConfig(vmConfig, projectPath)
	}

	return vmConfig
}
