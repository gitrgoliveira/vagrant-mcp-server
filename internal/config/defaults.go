// Package config provides standard configurations and defaults
package config

import "github.com/vagrant-mcp/server/internal/core"

// DefaultVM contains default VM configuration values
var DefaultVM = struct {
	// Standard box choices
	Boxes struct {
		Alpine string
		Ubuntu string
		Debian string
		CentOS string
	}
	// Common resources configurations
	Resources struct {
		Minimal  core.VMConfig
		Standard core.VMConfig
		Dev      core.VMConfig
	}
	// Default port mappings for common services
	Ports struct {
		HTTP       core.Port
		HTTPS      core.Port
		NodeJS     core.Port
		Python     core.Port
		PostgreSQL core.Port
		MySQL      core.Port
		Redis      core.Port
		MongoDB    core.Port
	}
	// Common sync exclude patterns
	ExcludePatterns []string
}{
	Boxes: struct {
		Alpine string
		Ubuntu string
		Debian string
		CentOS string
	}{
		Alpine: "generic/alpine314",
		Ubuntu: "ubuntu/focal64",
		Debian: "debian/bullseye64",
		CentOS: "centos/8",
	},
	Resources: struct {
		Minimal  core.VMConfig
		Standard core.VMConfig
		Dev      core.VMConfig
	}{
		Minimal: core.VMConfig{
			Box:    "generic/alpine314",
			CPU:    1,
			Memory: 512,
		},
		Standard: core.VMConfig{
			Box:    "ubuntu/focal64",
			CPU:    2,
			Memory: 1024,
		},
		Dev: core.VMConfig{
			Box:    "ubuntu/focal64",
			CPU:    4,
			Memory: 4096,
		},
	},
	Ports: struct {
		HTTP       core.Port
		HTTPS      core.Port
		NodeJS     core.Port
		Python     core.Port
		PostgreSQL core.Port
		MySQL      core.Port
		Redis      core.Port
		MongoDB    core.Port
	}{
		HTTP:       core.Port{Guest: 80, Host: 8080},
		HTTPS:      core.Port{Guest: 443, Host: 8443},
		NodeJS:     core.Port{Guest: 3000, Host: 3000},
		Python:     core.Port{Guest: 8000, Host: 8000},
		PostgreSQL: core.Port{Guest: 5432, Host: 5432},
		MySQL:      core.Port{Guest: 3306, Host: 3306},
		Redis:      core.Port{Guest: 6379, Host: 6379},
		MongoDB:    core.Port{Guest: 27017, Host: 27017},
	},
	ExcludePatterns: []string{
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
		"*.class",
		"target/",
		".cache/",
	},
}

// GetTestVMConfig returns a VM configuration suitable for testing
func GetTestVMConfig(vmType string) core.VMConfig {
	config, err := GlobalVMRegistry.GetConfig(vmType)
	if err != nil {
		// Return minimal as fallback
		config, _ = GlobalVMRegistry.GetConfig("minimal")
	}
	return config
}
