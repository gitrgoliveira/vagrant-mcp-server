// Package testsupport provides shared test utilities that don't depend on other packages
package testsupport

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/vagrant-mcp/server/internal/config"
	"github.com/vagrant-mcp/server/internal/core"
)

// VMState is an alias for the core VMState
type VMState = core.VMState

const (
	// NotCreated indicates the VM does not exist
	NotCreated = core.NotCreated
	// Running indicates the VM is running
	Running = core.Running
	// Stopped indicates the VM exists but is powered off
	Stopped = core.Stopped
	// Unknown indicates the VM state could not be determined
	Unknown = core.Unknown
)

// TestConfigOptions contains options for configuring a test fixture
type TestConfigOptions struct {
	// VMType specifies what kind of VM config to use (minimal, standard, dev)
	VMType string
	// SetupVM specifies whether to automatically set up a VM
	SetupVM bool
	// SkipIfNotIntegration skips the test if TEST_INTEGRATION=1 is not set
	SkipIfNotIntegration bool
}

// DefaultVMOptions provides common VM configurations for tests
var DefaultVMOptions = struct {
	Minimal  TestConfigOptions
	Standard TestConfigOptions
	Dev      TestConfigOptions
}{
	Minimal: TestConfigOptions{
		VMType:               "minimal",
		SetupVM:              false,
		SkipIfNotIntegration: true,
	},
	Standard: TestConfigOptions{
		VMType:               "standard",
		SetupVM:              true,
		SkipIfNotIntegration: true,
	},
	Dev: TestConfigOptions{
		VMType:               "dev",
		SetupVM:              true,
		SkipIfNotIntegration: true,
	},
}

// BaseFixture represents common test environment settings without specific dependencies
type BaseFixture struct {
	// TestDir is the temporary directory used for this test fixture
	TestDir string
	// VMName is the VM name for tests
	VMName string
	// ProjectPath is the project path used for VM creation
	ProjectPath string
	// T is the testing.T instance used for logging and test control
	T *testing.T
	// PackageName is used to create unique test directories for different packages
	PackageName string
	// Options contains the options used to create this fixture
	Options TestConfigOptions
}

// SetupBaseFixture creates a base test fixture without any specific implementations
func SetupBaseFixture(t *testing.T, packageName string, options *TestConfigOptions) (*BaseFixture, error) {
	if options == nil {
		options = &DefaultVMOptions.Minimal
	}

	// Skip integration test if not enabled
	if options.SkipIfNotIntegration && os.Getenv("TEST_INTEGRATION") != "1" {
		t.Skip("Skipping integration test. Set TEST_INTEGRATION=1 to run")
		return nil, fmt.Errorf("integration testing not enabled")
	}

	// Skip if Vagrant is not installed
	cmd := exec.Command("vagrant", "--version")
	if err := cmd.Run(); err != nil {
		t.Skipf("Skipping test because Vagrant is not installed: %v", err)
		return nil, err
	}

	// Create test directory
	testDir, err := os.MkdirTemp("", fmt.Sprintf("vagrant-mcp-%s-test-*", packageName))
	if err != nil {
		return nil, fmt.Errorf("failed to create test directory: %w", err)
	}

	// Set VM_BASE_DIR to use the test directory
	if err := os.Setenv("VM_BASE_DIR", filepath.Join(testDir, "vms")); err != nil {
		return nil, fmt.Errorf("failed to set VM_BASE_DIR: %w", err)
	}

	// Create project directory
	projectPath := filepath.Join(testDir, "project")
	if err := os.MkdirAll(projectPath, 0755); err != nil {
		if removeErr := os.RemoveAll(testDir); removeErr != nil {
			return nil, fmt.Errorf("failed to remove test directory: %w (original error: %w)", removeErr, err)
		}
		return nil, fmt.Errorf("failed to create project directory: %w", err)
	}

	fixture := &BaseFixture{
		TestDir:     testDir,
		VMName:      fmt.Sprintf("test-vm-%s-%s", packageName, time.Now().Format("20060102150405")),
		ProjectPath: projectPath,
		T:           t,
		PackageName: packageName,
		Options:     *options,
	}

	// Create a test file in the project directory
	testFilePath := filepath.Join(projectPath, "test.txt")
	if err := os.WriteFile(testFilePath, []byte("Test content"), 0644); err != nil {
		fixture.Cleanup()
		return nil, fmt.Errorf("failed to create test file: %w", err)
	}

	return fixture, nil
}

// GetVMConfig returns a VM configuration suitable for this test
func (f *BaseFixture) GetVMConfig() map[string]interface{} {
	// Get the VM configuration from the registry
	vmConfig, err := config.GlobalVMRegistry.GetConfig(f.Options.VMType)
	if err != nil {
		// Fall back to minimal configuration
		vmConfig, _ = config.GlobalVMRegistry.GetConfig("minimal")
	}

	// Convert core.Port slice to map format for compatibility
	var ports []map[string]int
	for _, port := range vmConfig.Ports {
		ports = append(ports, map[string]int{"guest": port.Guest, "host": port.Host})
	}

	// Convert to map for use with VM creation functions
	return map[string]interface{}{
		"box":                   vmConfig.Box,
		"cpu":                   vmConfig.CPU,
		"memory":                vmConfig.Memory,
		"sync_type":             vmConfig.SyncType,
		"project_path":          f.ProjectPath,
		"ports":                 ports,
		"environment":           vmConfig.Environment,
		"sync_exclude_patterns": vmConfig.SyncExcludePatterns,
	}
}

// Cleanup removes the test VM and directories
func (f *BaseFixture) Cleanup() {
	if f == nil {
		return
	}

	f.T.Logf("Cleaning up base fixture")

	// Try to destroy VM using Vagrant if it exists
	vmDir := filepath.Join(os.Getenv("VM_BASE_DIR"), f.VMName)
	if _, err := os.Stat(vmDir); err == nil {
		// VM directory exists, try to destroy it cleanly with vagrant force flag
		cmd := exec.Command("vagrant", "destroy", "-f")
		cmd.Dir = vmDir
		if err := cmd.Run(); err != nil {
			f.T.Logf("Failed to destroy VM with Vagrant: %v. Continuing with directory cleanup.", err)
		}
	}

	// Remove the test directory
	if f.TestDir != "" {
		f.T.Logf("Removing test directory %s", f.TestDir)
		if err := os.RemoveAll(f.TestDir); err != nil {
			f.T.Logf("Failed to remove test directory: %v", err)
		}
	}

	// Reset environment variable
	if err := os.Unsetenv("VM_BASE_DIR"); err != nil {
		f.T.Logf("Failed to unset VM_BASE_DIR: %v", err)
	}
}

// SkipIfNotIntegration checks the TEST_INTEGRATION environment variable
// and skips the test if it's not set to "1"
func SkipIfNotIntegration(t *testing.T) {
	if os.Getenv("TEST_INTEGRATION") != "1" {
		t.Skip("Skipping integration test. Set TEST_INTEGRATION=1 to run")
	}
}
