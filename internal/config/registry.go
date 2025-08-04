// Package config provides configuration registries to eliminate redundant switch statements
package config

import (
	"fmt"
	"sync"

	"github.com/vagrant-mcp/server/internal/core"
)

// VMConfigRegistry manages VM configuration templates
type VMConfigRegistry struct {
	configs map[string]core.VMConfig
	mutex   sync.RWMutex
}

var (
	// GlobalVMRegistry is the global VM configuration registry
	GlobalVMRegistry = NewVMConfigRegistry()
)

// NewVMConfigRegistry creates a new VM configuration registry
func NewVMConfigRegistry() *VMConfigRegistry {
	registry := &VMConfigRegistry{
		configs: make(map[string]core.VMConfig),
	}

	// Register default configurations
	registry.registerDefaultConfigs()

	return registry
}

// registerDefaultConfigs registers the standard VM configurations
func (r *VMConfigRegistry) registerDefaultConfigs() {
	// Minimal configuration
	r.RegisterConfig("minimal", core.VMConfig{
		Box:                 DefaultVM.Boxes.Alpine,
		CPU:                 1,
		Memory:              512,
		SyncType:            "rsync",
		Ports:               []core.Port{DefaultVM.Ports.HTTP},
		Environment:         []string{},
		SyncExcludePatterns: []string{".git", "*.log"},
	})

	// Standard configuration
	r.RegisterConfig("standard", core.VMConfig{
		Box:                 DefaultVM.Boxes.Ubuntu,
		CPU:                 2,
		Memory:              1024,
		SyncType:            "rsync",
		Ports:               []core.Port{DefaultVM.Ports.HTTP, DefaultVM.Ports.HTTPS, DefaultVM.Ports.NodeJS},
		Environment:         []string{"TERM=xterm"},
		SyncExcludePatterns: DefaultVM.ExcludePatterns,
	})

	// Development configuration
	r.RegisterConfig("dev", core.VMConfig{
		Box:      DefaultVM.Boxes.Ubuntu,
		CPU:      4,
		Memory:   4096,
		SyncType: "rsync",
		Ports: []core.Port{
			DefaultVM.Ports.NodeJS,
			DefaultVM.Ports.Python,
			DefaultVM.Ports.PostgreSQL,
			DefaultVM.Ports.MySQL,
			DefaultVM.Ports.Redis,
		},
		Environment: []string{"TERM=xterm", "LANG=C.UTF-8"},
		Provisioners: []string{
			"apt-get install -y build-essential git curl unzip",
			"apt-get install -y python3 python3-pip",
		},
		SyncExcludePatterns: DefaultVM.ExcludePatterns,
	})

	// CI configuration
	r.RegisterConfig("ci", core.VMConfig{
		Box:         DefaultVM.Boxes.Alpine,
		CPU:         1,
		Memory:      512,
		SyncType:    "rsync",
		Ports:       []core.Port{DefaultVM.Ports.HTTP},
		Environment: []string{"CI=true"},
		SyncExcludePatterns: []string{
			"node_modules", ".git", "*.log", "dist", "build",
		},
	})
}

// RegisterConfig registers a new VM configuration
func (r *VMConfigRegistry) RegisterConfig(name string, config core.VMConfig) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.configs[name] = config
}

// GetConfig retrieves a VM configuration by name
func (r *VMConfigRegistry) GetConfig(name string) (core.VMConfig, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	config, exists := r.configs[name]
	if !exists {
		// Return minimal as default
		if defaultConfig, ok := r.configs["minimal"]; ok {
			return defaultConfig, nil
		}
		return core.VMConfig{}, fmt.Errorf("VM configuration '%s' not found", name)
	}

	return config, nil
}

// GetConfigWithDefaults retrieves a VM configuration with project-specific customizations
func (r *VMConfigRegistry) GetConfigWithDefaults(name, projectPath string, customizations map[string]interface{}) (core.VMConfig, error) {
	config, err := r.GetConfig(name)
	if err != nil {
		return core.VMConfig{}, err
	}

	// Set project path if provided
	if projectPath != "" {
		config.ProjectPath = projectPath
	}

	// Apply customizations
	if customizations != nil {
		r.applyCustomizations(&config, customizations)
	}

	return config, nil
}

// applyCustomizations applies custom values to the configuration
func (r *VMConfigRegistry) applyCustomizations(config *core.VMConfig, customizations map[string]interface{}) {
	// Use the global config mapper for consistent handling
	if err := GlobalConfigMapper.ApplyCustomizations(config, customizations); err != nil {
		// Log error but don't fail - configuration errors should not break functionality
		_ = err
	}
}

// ListConfigs returns all available configuration names
func (r *VMConfigRegistry) ListConfigs() []string {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	names := make([]string, 0, len(r.configs))
	for name := range r.configs {
		names = append(names, name)
	}
	return names
}
