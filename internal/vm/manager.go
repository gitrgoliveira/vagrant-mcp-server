package vm

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
)

// State represents the state of a Vagrant VM
type State string

const (
	StateNotCreated State = "not_created"
	StateRunning    State = "running"
	StateStopped    State = "stopped"
	StateSuspended  State = "suspended"
	StateError      State = "error"
)

// VMConfig represents the configuration for a Vagrant VM
type VMConfig struct {
	Name         string   `json:"name"`
	Box          string   `json:"box"`
	CPU          int      `json:"cpu"`
	Memory       int      `json:"memory"`
	SyncType     string   `json:"sync_type"`
	ProjectPath  string   `json:"project_path"`
	Ports        []Port   `json:"ports"`
	Environment  []string `json:"environment"`
	Provisioners []string `json:"provisioners"`
}

// Port represents a port forwarding rule
type Port struct {
	Guest int `json:"guest"`
	Host  int `json:"host"`
}

// Manager handles VM lifecycle operations
type Manager struct {
	baseDir string
}

// NewManager creates a new VM manager
func NewManager() (*Manager, error) {
	// Get base directory from environment or use default
	baseDir := os.Getenv("VM_BASE_DIR")
	if baseDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get user home directory: %w", err)
		}
		baseDir = filepath.Join(homeDir, ".vagrant-mcp", "vms")
	}

	// Ensure the base directory exists
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create VM base directory: %w", err)
	}

	return &Manager{
		baseDir: baseDir,
	}, nil
}

// CreateVM creates a new Vagrant VM with the given configuration
func (m *Manager) CreateVM(name string, projectPath string, config VMConfig) error {
	// Create VM directory
	vmDir := m.getVMDir(name)
	if err := os.MkdirAll(vmDir, 0755); err != nil {
		return fmt.Errorf("failed to create VM directory: %w", err)
	}

	// Save VM configuration
	config.Name = name
	config.ProjectPath = projectPath
	if err := m.saveVMConfig(name, config); err != nil {
		return fmt.Errorf("failed to save VM configuration: %w", err)
	}

	// Generate Vagrantfile
	if err := m.generateVagrantfile(name, config); err != nil {
		return fmt.Errorf("failed to generate Vagrantfile: %w", err)
	}

	log.Info().Str("name", name).Msg("VM created successfully")
	return nil
}

// StartVM starts the specified VM
func (m *Manager) StartVM(name string) error {
	vmDir := m.getVMDir(name)

	// Run vagrant up
	cmd := exec.Command("vagrant", "up")
	cmd.Dir = vmDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to start VM: %w, output: %s", err, output)
	}

	log.Info().Str("name", name).Msg("VM started successfully")
	return nil
}

// StopVM stops the specified VM
func (m *Manager) StopVM(name string) error {
	vmDir := m.getVMDir(name)

	// Run vagrant halt
	cmd := exec.Command("vagrant", "halt")
	cmd.Dir = vmDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to stop VM: %w, output: %s", err, output)
	}

	log.Info().Str("name", name).Msg("VM stopped successfully")
	return nil
}

// DestroyVM destroys the specified VM and cleans up resources
func (m *Manager) DestroyVM(name string) error {
	vmDir := m.getVMDir(name)

	// Run vagrant destroy
	cmd := exec.Command("vagrant", "destroy", "-f")
	cmd.Dir = vmDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Error().Str("name", name).Err(err).Str("output", string(output)).Msg("Failed to destroy VM")
		// Continue with cleanup even if destroy fails
	}

	// Remove VM directory
	if err := os.RemoveAll(vmDir); err != nil {
		return fmt.Errorf("failed to clean up VM directory: %w", err)
	}

	// Remove VM config file
	configFile := filepath.Join(filepath.Dir(m.baseDir), fmt.Sprintf("%s.json", name))
	if err := os.Remove(configFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to clean up VM config: %w", err)
	}

	log.Info().Str("name", name).Msg("VM destroyed successfully")
	return nil
}

// GetVMState retrieves the current state of the VM
func (m *Manager) GetVMState(name string) (State, error) {
	vmDir := m.getVMDir(name)

	// Check if VM directory exists
	if _, err := os.Stat(vmDir); os.IsNotExist(err) {
		return StateNotCreated, nil
	}

	// Run vagrant status
	cmd := exec.Command("vagrant", "status", "--machine-readable")
	cmd.Dir = vmDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return StateError, fmt.Errorf("failed to get VM status: %w", err)
	}

	// Parse status output
	return m.parseVagrantStatus(string(output))
}

// GetVMConfig retrieves the VM configuration
func (m *Manager) GetVMConfig(name string) (VMConfig, error) {
	configFile := filepath.Join(filepath.Dir(m.baseDir), fmt.Sprintf("%s.json", name))

	data, err := os.ReadFile(configFile)
	if err != nil {
		return VMConfig{}, fmt.Errorf("failed to read VM config: %w", err)
	}

	var config VMConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return VMConfig{}, fmt.Errorf("failed to parse VM config: %w", err)
	}

	return config, nil
}

// GetSSHConfig retrieves the SSH configuration for the VM
func (m *Manager) GetSSHConfig(name string) (map[string]string, error) {
	vmDir := m.getVMDir(name)

	// Run vagrant ssh-config
	cmd := exec.Command("vagrant", "ssh-config")
	cmd.Dir = vmDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to get SSH config: %w", err)
	}

	// Parse SSH config
	return m.parseSSHConfig(string(output))
}

// Close cleans up resources used by the VM manager
func (m *Manager) Close() {
	// Nothing to clean up currently
}

// getVMDir returns the directory path for a VM
func (m *Manager) getVMDir(name string) string {
	return filepath.Join(m.baseDir, name)
}

// saveVMConfig saves the VM configuration to a file
func (m *Manager) saveVMConfig(name string, config VMConfig) error {
	configDir := filepath.Dir(m.baseDir)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	configFile := filepath.Join(configDir, fmt.Sprintf("%s.json", name))
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal VM config: %w", err)
	}

	return os.WriteFile(configFile, data, 0644)
}

// generateVagrantfile creates a Vagrantfile for the VM
func (m *Manager) generateVagrantfile(name string, config VMConfig) error {
	vagrantfile := `# -*- mode: ruby -*-
# vi: set ft=ruby :
# Generated by Vagrant MCP Server

Vagrant.configure("2") do |config|
  # Box settings
  config.vm.box = "%s"
  
  # Provider-specific configuration
  config.vm.provider "virtualbox" do |vb|
    vb.gui = false
    vb.name = "%s"
    vb.memory = %d
    vb.cpus = %d
    
    # Performance optimizations
    vb.customize ["modifyvm", :id, "--natdnshostresolver1", "on"]
    vb.customize ["modifyvm", :id, "--natdnsproxy1", "on"]
    vb.customize ["modifyvm", :id, "--ioapic", "on"]
  end

  # Network settings
%s
  
  # Sync settings
%s
  
  # Provisioning
  config.vm.provision "shell", inline: <<-SHELL
    # Update package list
    apt-get update
    
    # Install basic development tools
    apt-get install -y build-essential curl git unzip
%s
    echo "Development VM setup completed!"
  SHELL
end`

	// Generate port forwarding configuration
	portsConfig := ""
	for _, port := range config.Ports {
		portsConfig += fmt.Sprintf("  config.vm.network \"forwarded_port\", guest: %d, host: %d, host_ip: \"127.0.0.1\"\n",
			port.Guest, port.Host)
	}

	// Generate sync configuration
	syncConfig := ""
	switch config.SyncType {
	case "rsync":
		syncConfig = fmt.Sprintf(`  config.vm.synced_folder "%s", "/vagrant", 
    type: "rsync",
    rsync__exclude: [".git/", "node_modules/", "dist/", ".vagrant/"],
    rsync__args: ["--verbose", "--archive", "--delete", "-z"]`, config.ProjectPath)
	case "nfs":
		syncConfig = fmt.Sprintf(`  config.vm.synced_folder "%s", "/vagrant", 
    type: "nfs",
    nfs_udp: false,
    nfs_version: 4`, config.ProjectPath)
	case "smb":
		syncConfig = fmt.Sprintf(`  config.vm.synced_folder "%s", "/vagrant", 
    type: "smb"`, config.ProjectPath)
	default:
		syncConfig = fmt.Sprintf(`  config.vm.synced_folder "%s", "/vagrant"`, config.ProjectPath)
	}

	// Generate environment setup
	envSetup := ""
	for _, line := range config.Environment {
		envSetup += "    " + line + "\n"
	}

	// Format the complete Vagrantfile
	content := fmt.Sprintf(vagrantfile,
		config.Box,    // Box name
		name,          // VM name
		config.Memory, // Memory
		config.CPU,    // CPU
		portsConfig,   // Port forwarding
		syncConfig,    // Sync configuration
		envSetup)      // Environment setup

	// Write the Vagrantfile
	vagrantfilePath := filepath.Join(m.getVMDir(name), "Vagrantfile")
	return os.WriteFile(vagrantfilePath, []byte(content), 0644)
}

// parseVagrantStatus parses the output of 'vagrant status --machine-readable'
func (m *Manager) parseVagrantStatus(output string) (State, error) {
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		parts := strings.Split(line, ",")
		if len(parts) >= 4 && parts[2] == "state" {
			switch parts[3] {
			case "running":
				return StateRunning, nil
			case "poweroff", "aborted":
				return StateStopped, nil
			case "saved":
				return StateSuspended, nil
			case "not_created":
				return StateNotCreated, nil
			}
		}
	}

	return StateError, fmt.Errorf("could not determine VM state")
}

// parseSSHConfig parses the output of 'vagrant ssh-config'
func (m *Manager) parseSSHConfig(output string) (map[string]string, error) {
	config := make(map[string]string)
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 2 {
			key := parts[0]
			value := strings.TrimSpace(parts[1])
			config[key] = value
		}
	}

	return config, nil
}
