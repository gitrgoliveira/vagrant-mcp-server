// Package testing provides unified test infrastructure for the Vagrant MCP Server
package testing

import (
	"context"
	"fmt"
	"os"
	osExec "os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/vagrant-mcp/server/internal/core"
	"github.com/vagrant-mcp/server/internal/exec"
	"github.com/vagrant-mcp/server/internal/sync"
	"github.com/vagrant-mcp/server/internal/utils"
	"github.com/vagrant-mcp/server/internal/vm"
)

// isCI returns true if running in a CI environment
func isCI() bool {
	return os.Getenv("CI") == "true" || os.Getenv("GITHUB_ACTIONS") == "true"
}

// shouldSkipProviderTests returns true if provider-dependent tests should be skipped
func shouldSkipProviderTests() bool {
	return isCI() || os.Getenv("SKIP_VAGRANT_VALIDATION") == "true"
}

// UnifiedFixture provides a unified test environment for all packages
type UnifiedFixture struct {
	VMManager   core.VMManager
	SyncEngine  core.SyncEngine
	Executor    *exec.Executor
	TestDir     string
	VMName      string
	ProjectPath string
	T           *testing.T
	ctx         context.Context
	packageName string
	vmCreated   bool // Track whether a VM was actually created
}

// FixtureOptions configures the test fixture setup
type FixtureOptions struct {
	PackageName   string
	SetupVM       bool
	StartVM       bool // Control whether to actually start the VM after creating it
	CreateProject bool
	EnableSync    bool
}

// NewUnifiedFixture creates a new unified test fixture
func NewUnifiedFixture(t *testing.T, opts FixtureOptions) (*UnifiedFixture, error) {
	// Skip if Vagrant is not installed
	if err := utils.CheckVagrantInstalled(); err != nil {
		t.Skipf("Skipping test because Vagrant is not installed: %v", err)
		return nil, err
	}

	ctx := context.Background()

	// Create test directory
	testDir, err := os.MkdirTemp("", fmt.Sprintf("vagrant-mcp-%s-test-*", opts.PackageName))
	if err != nil {
		return nil, fmt.Errorf("failed to create test directory: %w", err)
	}

	// Set VM_BASE_DIR to use the test directory
	vmBaseDir := filepath.Join(testDir, "vms")
	if err := os.Setenv("VM_BASE_DIR", vmBaseDir); err != nil {
		return nil, fmt.Errorf("failed to set VM_BASE_DIR: %w", err)
	}

	// Create VM manager
	vmManager, err := vm.NewManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create VM manager: %w", err)
	}

	// Create sync engine
	syncEngine, err := sync.NewEngine()
	if err != nil {
		return nil, fmt.Errorf("failed to create sync engine: %w", err)
	}

	// Create adapters for interface compatibility
	adapterVM := &exec.VMManagerAdapter{Real: vmManager}
	syncEngine.SetVMManager(adapterVM)
	adapterSync := &exec.SyncEngineAdapter{Real: syncEngine}

	// Create executor if requested
	var executor *exec.Executor
	if opts.SetupVM {
		executor, err = exec.NewExecutor(adapterVM, adapterSync)
		if err != nil {
			return nil, fmt.Errorf("failed to create executor: %w", err)
		}
	}

	fixture := &UnifiedFixture{
		VMManager:   adapterVM,
		SyncEngine:  adapterSync,
		Executor:    executor,
		TestDir:     testDir,
		VMName:      fmt.Sprintf("test-vm-%s-%d", opts.PackageName, time.Now().Unix()),
		ProjectPath: filepath.Join(testDir, "project"),
		T:           t,
		ctx:         ctx,
		packageName: opts.PackageName,
		vmCreated:   false, // Initialize as false
	}

	// Create project directory if requested
	if opts.CreateProject {
		if err := fixture.createProjectDirectory(); err != nil {
			fixture.Cleanup()
			return nil, err
		}
	}

	// Setup VM if requested
	if opts.SetupVM {
		if err := fixture.setupVM(opts); err != nil {
			fixture.Cleanup()
			return nil, err
		}

		// Setup sync if requested
		if opts.EnableSync {
			if err := fixture.setupSync(); err != nil {
				fixture.Cleanup()
				return nil, err
			}
		}
	}

	return fixture, nil
}

// Context returns the test context
func (f *UnifiedFixture) Context() context.Context {
	return f.ctx
}

// createProjectDirectory creates a test project directory
func (f *UnifiedFixture) createProjectDirectory() error {
	if err := os.MkdirAll(f.ProjectPath, 0755); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	// Create a simple test file
	testFile := filepath.Join(f.ProjectPath, "test.txt")
	content := fmt.Sprintf("Test file for %s package\nCreated at: %s\n", f.packageName, time.Now().Format(time.RFC3339))
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to create test file: %w", err)
	}

	return nil
}

// setupVM creates and starts a test VM
func (f *UnifiedFixture) setupVM(opts FixtureOptions) error {
	log.Info().Str("vm", f.VMName).Msg("Setting up test VM")

	config := &core.VMConfig{
		Name:        f.VMName,
		Box:         "ubuntu/focal64",
		CPU:         1,
		Memory:      1024,
		ProjectPath: f.ProjectPath,
		SyncType:    "rsync",
	}

	if err := f.VMManager.CreateVM(f.ctx, f.VMName, f.ProjectPath, *config); err != nil {
		return fmt.Errorf("failed to create VM: %w", err)
	}

	// Mark VM as created (even if not started)
	f.vmCreated = true

	// Only start the VM if explicitly requested and not in CI
	if opts.StartVM && !shouldSkipProviderTests() {
		if err := f.VMManager.StartVM(f.ctx, f.VMName); err != nil {
			return fmt.Errorf("failed to start VM: %w", err)
		}

		// Wait for VM to be ready only if we started it
		return f.waitForVMReady()
	}

	return nil
}

// setupSync configures file synchronization
func (f *UnifiedFixture) setupSync() error {
	log.Info().Str("vm", f.VMName).Msg("Setting up sync for test VM")

	syncConfig := core.SyncConfig{
		VMName:      f.VMName,
		ProjectPath: f.ProjectPath,
		Method:      core.SyncMethodRsync,
		Direction:   core.SyncBidirectional,
	}

	return f.SyncEngine.RegisterVM(f.ctx, f.VMName, syncConfig)
}

// waitForVMReady waits for the VM to be ready for operations
func (f *UnifiedFixture) waitForVMReady() error {
	maxAttempts := 30
	for i := 0; i < maxAttempts; i++ {
		state, err := f.VMManager.GetVMState(f.ctx, f.VMName)
		if err == nil && state == core.Running {
			// Additional check - try to execute a simple command
			if f.Executor != nil {
				execCtx := exec.ExecutionContext{
					VMName:     f.VMName,
					WorkingDir: "/vagrant",
				}
				_, err := f.Executor.ExecuteCommand(f.ctx, "echo 'VM ready'", execCtx, nil)
				if err == nil {
					log.Info().Str("vm", f.VMName).Msg("Test VM is ready")
					return nil
				}
			} else {
				return nil
			}
		}

		time.Sleep(2 * time.Second)
	}

	return fmt.Errorf("VM did not become ready within timeout")
}

// Cleanup removes the test VM and directories
func (f *UnifiedFixture) Cleanup() {
	log.Info().Str("vm", f.VMName).Msg("Cleaning up test fixture")

	// Only try to clean up VM if one was actually created
	if f.VMManager != nil && f.vmCreated {
		// First try to stop the VM
		if err := f.VMManager.StopVM(f.ctx, f.VMName); err != nil {
			log.Warn().Err(err).Str("vm", f.VMName).Msg("Failed to stop VM during cleanup")
		}

		// Then try to destroy it
		if err := f.VMManager.DestroyVM(f.ctx, f.VMName); err != nil {
			log.Warn().Err(err).Str("vm", f.VMName).Msg("Failed to destroy VM during cleanup")

			// If normal destroy fails, try force destroy by VM name
			f.forceDestroyVM()
		}
	}

	// Remove the test directory
	if f.TestDir != "" {
		if err := os.RemoveAll(f.TestDir); err != nil {
			log.Warn().Err(err).Str("dir", f.TestDir).Msg("Failed to remove test directory")
		}
	}
}

// forceDestroyVM attempts to force destroy a VM using global vagrant commands
func (f *UnifiedFixture) forceDestroyVM() {
	log.Info().Str("vm", f.VMName).Msg("Attempting force destroy of VM")

	// Try to find and destroy the VM using vagrant global-status
	cmd := osExec.Command("vagrant", "global-status", "--machine-readable")
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Warn().Err(err).Msg("Failed to get global vagrant status")
		return
	}

	// Parse the output to find our VM
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		parts := strings.Split(line, ",")
		if len(parts) >= 4 && parts[1] == f.VMName {
			vmID := parts[0]
			if vmID != "" {
				log.Info().Str("vm_id", vmID).Str("vm", f.VMName).Msg("Force destroying VM by ID")
				destroyCmd := osExec.Command("vagrant", "destroy", vmID, "--force")
				if err := destroyCmd.Run(); err != nil {
					log.Warn().Err(err).Str("vm_id", vmID).Msg("Failed to force destroy VM")
				} else {
					log.Info().Str("vm_id", vmID).Msg("Successfully force destroyed VM")
				}
				break
			}
		}
	}
}

// CreateTestFile creates a test file in the project directory
func (f *UnifiedFixture) CreateTestFile(filename, content string) error {
	filePath := filepath.Join(f.ProjectPath, filename)
	return os.WriteFile(filePath, []byte(content), 0644)
}

// ExecuteCommand executes a command in the test VM (requires VM setup)
func (f *UnifiedFixture) ExecuteCommand(command string) (*exec.CommandResult, error) {
	if f.Executor == nil {
		return nil, fmt.Errorf("executor not available - VM not set up")
	}

	execCtx := exec.ExecutionContext{
		VMName:     f.VMName,
		WorkingDir: "/vagrant",
	}

	return f.Executor.ExecuteCommand(f.ctx, command, execCtx, nil)
}

// WaitForVM waits for the VM to reach a specific state
func (f *UnifiedFixture) WaitForVM(expectedState core.VMState, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		state, err := f.VMManager.GetVMState(f.ctx, f.VMName)
		if err == nil && state == expectedState {
			return nil
		}
		time.Sleep(1 * time.Second)
	}

	return fmt.Errorf("VM did not reach state %v within timeout", expectedState)
}
