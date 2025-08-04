package vm

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"github.com/vagrant-mcp/server/internal/cmdexec"
	"github.com/vagrant-mcp/server/internal/core"
	"github.com/vagrant-mcp/server/internal/errors"
	"github.com/vagrant-mcp/server/internal/utils"
)

// Manager represents the configuration for a Vagrant VM Manager

// Manager handles VM lifecycle operations
type Manager struct {
	baseDir string
}

// NewManager creates a new VM manager
func NewManager() (*Manager, error) {
	// Check if Vagrant CLI is installed
	if err := utils.CheckVagrantInstalled(); err != nil {
		return nil, fmt.Errorf("failed to initialize VM manager: %w", err)
	}

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
func (m *Manager) CreateVM(ctx context.Context, name string, projectPath string, config core.VMConfig) error {
	vmDir := m.getVMDir(name)
	if err := os.MkdirAll(vmDir, 0755); err != nil {
		return errors.OperationFailed("create VM directory", err)
	}
	config.Name = name
	config.ProjectPath = projectPath
	if err := m.saveVMConfig(name, config); err != nil {
		return errors.OperationFailed("save VM configuration", err)
	}
	if err := m.generateVagrantfile(name, config); err != nil {
		return errors.OperationFailed("generate Vagrantfile", err)
	}
	log.Info().Str("name", name).Msg("VM created successfully")
	return nil
}

// StartVM starts the specified VM
func (m *Manager) StartVM(ctx context.Context, name string) error {
	vmDir := m.getVMDir(name)
	cmd := exec.CommandContext(ctx, "vagrant", "up")
	cmd.Dir = vmDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrap(err, errors.CodeOperationFailed, fmt.Sprintf("failed to start VM: %s", output))
	}
	log.Info().Str("name", name).Msg("VM started successfully")
	return nil
}

// StopVM stops the specified VM
func (m *Manager) StopVM(ctx context.Context, name string) error {
	vmDir := m.getVMDir(name)
	cmd := exec.CommandContext(ctx, "vagrant", "halt")
	cmd.Dir = vmDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrap(err, errors.CodeOperationFailed, fmt.Sprintf("failed to stop VM: %s", output))
	}
	log.Info().Str("name", name).Msg("VM stopped successfully")
	return nil
}

// DestroyVM destroys the specified VM and cleans up resources
func (m *Manager) DestroyVM(ctx context.Context, name string) error {
	vmDir := m.getVMDir(name)
	cmd := exec.CommandContext(ctx, "vagrant", "destroy", "-f")
	cmd.Dir = vmDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Error().Str("name", name).Err(err).Str("output", string(output)).Msg("Failed to destroy VM")
		// Continue with cleanup even if destroy fails
	}
	if err := os.RemoveAll(vmDir); err != nil {
		return errors.OperationFailed("clean up VM directory", err)
	}
	configFile := filepath.Join(filepath.Dir(m.baseDir), fmt.Sprintf("%s.json", name))
	if err := os.Remove(configFile); err != nil && !os.IsNotExist(err) {
		return errors.OperationFailed("clean up VM config", err)
	}
	log.Info().Str("name", name).Msg("VM destroyed successfully")
	return nil
}

// GetVMState returns the current state of the VM as core.VMState
func (m *Manager) GetVMState(ctx context.Context, name string) (core.VMState, error) {
	vmDir := m.getVMDir(name)
	if _, err := os.Stat(vmDir); os.IsNotExist(err) {
		return core.NotCreated, nil
	}
	cmd := exec.CommandContext(ctx, "vagrant", "status", "--machine-readable")
	cmd.Dir = vmDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return core.Unknown, errors.OperationFailed("get VM status", err)
	}
	state, err := m.parseVagrantStatus(string(output))
	if err != nil {
		return core.Unknown, errors.OperationFailed("parse vagrant status", err)
	}
	return state, nil
}

// GetVMConfig returns the VM configuration as core.VMConfig
func (m *Manager) GetVMConfig(ctx context.Context, name string) (core.VMConfig, error) {
	configFile := filepath.Join(filepath.Dir(m.baseDir), fmt.Sprintf("%s.json", name))
	data, err := os.ReadFile(configFile)
	if err != nil {
		return core.VMConfig{}, errors.OperationFailed("read VM config", err)
	}
	var config core.VMConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return core.VMConfig{}, errors.OperationFailed("parse VM config", err)
	}
	return config, nil
}

// UpdateVMConfig updates the VM configuration using core.VMConfig
func (m *Manager) UpdateVMConfig(ctx context.Context, name string, config core.VMConfig) error {
	log.Debug().Str("vm", name).Msg("Updating VM configuration")
	vmDir := filepath.Join(m.baseDir, name)
	if _, err := os.Stat(vmDir); os.IsNotExist(err) {
		return errors.NotFound("VM directory", vmDir)
	}
	configPath := filepath.Join(vmDir, "config.json")
	configData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return errors.OperationFailed("marshal VM config", err)
	}
	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		return errors.OperationFailed("write VM config", err)
	}
	log.Info().Str("vm", name).Msg("VM configuration updated")
	return nil
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
func (m *Manager) saveVMConfig(name string, config core.VMConfig) error {
	configDir := filepath.Dir(m.baseDir)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return errors.OperationFailed("create config directory", err)
	}

	configFile := filepath.Join(configDir, fmt.Sprintf("%s.json", name))
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return errors.OperationFailed("marshal VM config", err)
	}

	return os.WriteFile(configFile, data, 0644)
}

// generateVagrantfile creates a Vagrantfile for the VM and validates it
func (m *Manager) generateVagrantfile(name string, config core.VMConfig) error {
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
	vmDir := m.getVMDir(name)
	vagrantfilePath := filepath.Join(vmDir, "Vagrantfile")
	if err := os.WriteFile(vagrantfilePath, []byte(content), 0644); err != nil {
		return errors.OperationFailed("write Vagrantfile", err)
	}

	// Always validate the Vagrantfile to ensure it's correct
	cmd := exec.Command("vagrant", "validate")
	cmd.Dir = vmDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrap(err, errors.CodeOperationFailed, fmt.Sprintf("vagrantfile validation failed: %s", output))
	}
	log.Info().Str("name", name).Msg("Vagrantfile validated successfully")

	return nil
}

// parseVagrantStatus parses the output of 'vagrant status --machine-readable'
func (m *Manager) parseVagrantStatus(output string) (core.VMState, error) {
	return GlobalStateMapper.ParseVagrantState(output)
}

// ParseVagrantStatus parses the output of 'vagrant status --machine-readable'
func (m *Manager) ParseVagrantStatus(output string) (core.VMState, error) {
	return m.parseVagrantStatus(output)
}

// parseSSHConfig parses the output of 'vagrant ssh-config'
func (m *Manager) parseSSHConfig(output string) (map[string]string, error) {
	return utils.GlobalOutputParser.ParseSSHConfig(output)
}

// ExecuteCommand executes a command in a VM
func (m *Manager) ExecuteCommand(ctx context.Context, name string, cmd string, args []string, workingDir string) (string, string, int, error) {
	vmDir := m.getVMDir(name)
	options := cmdexec.CmdOptions{
		Directory:  vmDir,
		OutputMode: cmdexec.OutputModeCapture,
	}
	// If a workingDir is provided, use it as a subdirectory inside the VM directory
	if workingDir != "" {
		options.Directory = filepath.Join(vmDir, workingDir)
	}
	result, err := cmdexec.Execute(ctx, cmd, args, options)
	if err != nil {
		return string(result.StdOut), string(result.StdErr), result.ExitCode, errors.OperationFailed("execute command in VM", err)
	}
	return string(result.StdOut), string(result.StdErr), result.ExitCode, nil
}

// UploadToVM uploads a file or directory to the VM using vagrant upload
func (m *Manager) UploadToVM(ctx context.Context, name string, source string, destination string, compress bool, compressionType string) error {
	vmDir := m.getVMDir(name)
	if _, err := os.Stat(vmDir); os.IsNotExist(err) {
		return errors.NotFound("VM", name)
	}
	state, err := m.GetVMState(ctx, name)
	if err != nil {
		return errors.OperationFailed("get VM state", err)
	}
	if state != core.Running {
		return errors.Wrap(fmt.Errorf("VM is not running (current state: %s)", state), errors.CodeInvalidState, "VM is not running")
	}
	if _, err := os.Stat(source); os.IsNotExist(err) {
		return errors.NotFound("source path", source)
	}
	args := []string{"upload"}
	if compress {
		args = append(args, "--compress")
		if compressionType != "" {
			args = append(args, "--compression-type", compressionType)
		}
	}
	args = append(args, source, destination)
	cmd := exec.CommandContext(ctx, "vagrant", args...)
	cmd.Dir = vmDir
	log.Debug().Str("vm", name).Str("source", source).Str("destination", destination).
		Bool("compress", compress).Str("compression", compressionType).
		Msg("Uploading file to VM")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return errors.OperationFailed("upload file to VM", fmt.Errorf("%w: %s", err, output))
	}
	log.Info().Str("vm", name).Str("source", source).Str("destination", destination).
		Msg("File uploaded to VM successfully")
	return nil
}

// SyncToVM synchronizes files from host to VM using rsync
func (m *Manager) SyncToVM(name, source, target string) error {
	// Use rsync to copy files from host to VM
	// This is a simplified implementation; in production, handle SSH config, errors, etc.
	vmDir := m.getVMDir(name)
	if vmDir == "" {
		return fmt.Errorf("could not determine VM directory for %s", name)
	}
	// Assume target is relative to /vagrant in the VM
	cmd := exec.Command("rsync", "-az", "--delete", source+"/", vmDir+"/vagrant/"+target+"/")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("rsync to VM failed: %v, output: %s", err, string(output))
	}
	return nil
}

// SyncFromVM synchronizes files from VM to host using rsync
func (m *Manager) SyncFromVM(name, source, target string) error {
	// Use rsync to copy files from VM to host
	vmDir := m.getVMDir(name)
	if vmDir == "" {
		return fmt.Errorf("could not determine VM directory for %s", name)
	}
	cmd := exec.Command("rsync", "-az", "--delete", vmDir+"/vagrant/"+source+"/", target+"/")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("rsync from VM failed: %v, output: %s", err, string(output))
	}
	return nil
}

// GetSSHConfig retrieves the SSH configuration for the VM using 'vagrant ssh-config'
func (m *Manager) GetSSHConfig(ctx context.Context, name string) (map[string]string, error) {
	vmDir := m.getVMDir(name)
	cmd := exec.CommandContext(ctx, "vagrant", "ssh-config")
	cmd.Dir = vmDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to get SSH config: %w, output: %s", err, string(output))
	}
	return m.parseSSHConfig(string(output))
}
