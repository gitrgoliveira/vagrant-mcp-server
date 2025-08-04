// Copyright Ricardo Oliveira 2025.
// SPDX-License-Identifier: MPL-2.0

// Package cmdexec provides command execution utilities
package cmdexec

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog/log"
)

// VagrantExecutor provides Vagrant-specific command execution utilities
type VagrantExecutor struct {
	// BaseDir is the base directory for Vagrant VMs
	BaseDir string
	// DefaultTimeout is the default timeout for Vagrant commands
	DefaultTimeout time.Duration
}

// NewVagrantExecutor creates a new Vagrant executor
func NewVagrantExecutor(baseDir string) *VagrantExecutor {
	if baseDir == "" {
		// Use default Vagrant base directory if not specified
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Error().Err(err).Msg("Failed to get user home directory")
			homeDir = "."
		}
		baseDir = filepath.Join(homeDir, ".vagrant-mcp-server", "vms")
	}

	return &VagrantExecutor{
		BaseDir:        baseDir,
		DefaultTimeout: 10 * time.Minute, // Vagrant operations can be slow
	}
}

// GetVMDir returns the directory path for a VM
func (e *VagrantExecutor) GetVMDir(vmName string) string {
	return filepath.Join(e.BaseDir, vmName)
}

// ExecuteVagrant executes a Vagrant command for a specific VM
func (e *VagrantExecutor) ExecuteVagrant(ctx context.Context, vmName string, args []string, options *CmdOptions) (*Result, error) {
	// Determine the VM directory
	vmDir := e.GetVMDir(vmName)

	// Create the VM directory if it doesn't exist (for commands like vagrant init)
	if _, err := os.Stat(vmDir); os.IsNotExist(err) {
		if err := os.MkdirAll(vmDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create VM directory: %w", err)
		}
	}

	// Create a copy of the options to avoid modifying the original
	var execOptions CmdOptions
	if options != nil {
		execOptions = *options
	}

	// Set the working directory to the VM directory
	execOptions.Directory = vmDir

	// Execute the command
	return Execute(ctx, "vagrant", args, execOptions)
}

// Up starts a VM
func (e *VagrantExecutor) Up(ctx context.Context, vmName string, options *CmdOptions) (*Result, error) {
	return e.ExecuteVagrant(ctx, vmName, []string{"up"}, options)
}

// Halt stops a VM
func (e *VagrantExecutor) Halt(ctx context.Context, vmName string, options *CmdOptions) (*Result, error) {
	return e.ExecuteVagrant(ctx, vmName, []string{"halt"}, options)
}

// Destroy destroys a VM
func (e *VagrantExecutor) Destroy(ctx context.Context, vmName string, force bool, options *CmdOptions) (*Result, error) {
	args := []string{"destroy"}
	if force {
		args = append(args, "-f")
	}
	return e.ExecuteVagrant(ctx, vmName, args, options)
}

// Status gets the status of a VM
func (e *VagrantExecutor) Status(ctx context.Context, vmName string, options *CmdOptions) (*Result, error) {
	return e.ExecuteVagrant(ctx, vmName, []string{"status", "--machine-readable"}, options)
}

// SSHConfig gets the SSH configuration for a VM
func (e *VagrantExecutor) SSHConfig(ctx context.Context, vmName string, options *CmdOptions) (*Result, error) {
	return e.ExecuteVagrant(ctx, vmName, []string{"ssh-config"}, options)
}

// SSH executes a command via SSH in the VM
func (e *VagrantExecutor) SSH(ctx context.Context, vmName string, command string, options *CmdOptions) (*Result, error) {
	return e.ExecuteVagrant(ctx, vmName, []string{"ssh", "-c", command}, options)
}

// Rsync synchronizes files from host to VM
func (e *VagrantExecutor) Rsync(ctx context.Context, vmName string, options *CmdOptions) (*Result, error) {
	return e.ExecuteVagrant(ctx, vmName, []string{"rsync"}, options)
}

// RsyncBack synchronizes files from VM to host (requires rsync-back plugin)
func (e *VagrantExecutor) RsyncBack(ctx context.Context, vmName string, options *CmdOptions) (*Result, error) {
	return e.ExecuteVagrant(ctx, vmName, []string{"rsync-back"}, options)
}

// Upload uploads a file to the VM
func (e *VagrantExecutor) Upload(ctx context.Context, vmName string, source string, destination string, options *CmdOptions) (*Result, error) {
	return e.ExecuteVagrant(ctx, vmName, []string{"upload", source, destination}, options)
}
