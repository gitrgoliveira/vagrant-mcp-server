package exec

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/vagrant-mcp/server/internal/core"
	syncmod "github.com/vagrant-mcp/server/internal/sync"
	"github.com/vagrant-mcp/server/internal/testsupport"
	"github.com/vagrant-mcp/server/internal/utils"
	"github.com/vagrant-mcp/server/internal/vm"
)

// testFixture represents a test environment with real VMs
type testFixture struct {
	VMManager   core.VMManager
	SyncEngine  core.SyncEngine
	Executor    *Executor
	baseFixture *testsupport.BaseFixture
	ctx         context.Context
}

// setupTestFixture creates a new test fixture with real VM manager and sync engine
func setupTestFixture(t *testing.T, setupVM bool) (*testFixture, error) {
	// Create context for operations
	ctx := context.Background()

	// Skip if Vagrant is not installed
	if err := utils.CheckVagrantInstalled(); err != nil {
		t.Skipf("Skipping test because Vagrant is not installed: %v", err)
		return nil, err
	}

	// Create base fixture
	baseFixture, err := testsupport.SetupBaseFixture(t, "exec", nil)
	if err != nil {
		return nil, err
	}

	// Create VM manager and sync engine
	vmManager, err := vm.NewManager()
	if err != nil {
		baseFixture.Cleanup()
		return nil, fmt.Errorf("failed to create VM manager: %w", err)
	}

	syncEngine, err := syncmod.NewEngine()
	if err != nil {
		baseFixture.Cleanup()
		return nil, fmt.Errorf("failed to create sync engine: %w", err)
	}

	// Create adapters
	vmAdapter := &VMManagerAdapter{Real: vmManager}
	syncAdapter := &SyncEngineAdapter{Real: syncEngine}

	// Set VM Manager on Sync Engine
	syncEngine.SetVMManager(vmAdapter)

	// Create executor
	executor, err := NewExecutor(vmAdapter, syncAdapter)
	if err != nil {
		baseFixture.Cleanup()
		return nil, fmt.Errorf("failed to create executor: %w", err)
	}

	fixture := &testFixture{
		VMManager:   vmAdapter,
		SyncEngine:  syncAdapter,
		Executor:    executor,
		baseFixture: baseFixture,
		ctx:         ctx,
	}

	// Setup VM if requested
	if setupVM {
		err := fixture.setupVM()
		if err != nil {
			baseFixture.Cleanup()
			return nil, err
		}
	}

	return fixture, nil
}

// setupVM creates and starts a VM for testing
func (f *testFixture) setupVM() error {
	// Use standard test configuration from testsupport
	config := testsupport.GetVMConfig("minimal", f.baseFixture.ProjectPath)

	log.Info().Str("vm", f.baseFixture.VMName).Str("path", f.baseFixture.ProjectPath).Msg("Creating VM for test")
	f.baseFixture.T.Logf("Creating VM %s with project path %s", f.baseFixture.VMName, f.baseFixture.ProjectPath)
	if err := f.VMManager.CreateVM(f.ctx, f.baseFixture.VMName, f.baseFixture.ProjectPath, config); err != nil {
		return fmt.Errorf("failed to create VM: %w", err)
	}

	// Start the VM
	f.baseFixture.T.Logf("Starting VM %s", f.baseFixture.VMName)
	if err := f.VMManager.StartVM(f.ctx, f.baseFixture.VMName); err != nil {
		return fmt.Errorf("failed to start VM: %w", err)
	}

	// Wait for VM to be ready (this can take a while)
	f.baseFixture.T.Logf("Waiting for VM %s to be ready", f.baseFixture.VMName)
	deadline := time.Now().Add(5 * time.Minute)
	for time.Now().Before(deadline) {
		state, err := f.VMManager.GetVMState(f.ctx, f.baseFixture.VMName)
		if err != nil {
			f.baseFixture.T.Logf("Error checking VM state: %v", err)
		} else if state == core.Running {
			f.baseFixture.T.Logf("VM %s is now running", f.baseFixture.VMName)
			return nil
		}
		time.Sleep(5 * time.Second)
	}

	return fmt.Errorf("timed out waiting for VM to start")
}

// cleanup removes the test VM and directories
func (f *testFixture) cleanup() {
	if f == nil {
		return
	}

	f.baseFixture.T.Logf("Cleaning up test fixture")

	// Try to destroy the VM if it exists
	if f.baseFixture.VMName != "" {
		f.baseFixture.T.Logf("Destroying VM %s", f.baseFixture.VMName)
		if err := f.VMManager.DestroyVM(f.ctx, f.baseFixture.VMName); err != nil {
			f.baseFixture.T.Logf("Failed to destroy VM: %v", err)
		}
	}

	// Clean up the base fixture
	f.baseFixture.Cleanup()
}

// TestExecutor_ExecuteCommand tests the ExecuteCommand method with a real VM
func TestExecutor_ExecuteCommand(t *testing.T) {
	// Skip by default - can be enabled with TEST_LEVEL=integration
	testLevel := os.Getenv("TEST_LEVEL")
	if testLevel != "integration" && testLevel != "vm-start" {
		t.Skip("Skipping integration test. Set TEST_LEVEL=integration to run")
		return
	}

	// Setup fixture with a real VM
	fixture, err := setupTestFixture(t, true)
	if err != nil {
		t.Fatalf("Failed to set up test fixture: %v", err)
	}
	defer fixture.cleanup()

	// Run a simple echo command in the VM
	execContext := ExecutionContext{
		VMName:     fixture.baseFixture.VMName,
		WorkingDir: "/vagrant",
		SyncBefore: true,
		SyncAfter:  true,
	}

	var outputCalled bool
	var outputMu sync.Mutex
	outputCallback := func(data []byte, isStderr bool) {
		outputMu.Lock()
		defer outputMu.Unlock()
		outputCalled = true
		t.Logf("Output: %s (stderr: %v)", string(data), isStderr)
	}

	// Execute a simple command
	result, err := fixture.Executor.ExecuteCommand(context.Background(), "echo", execContext, outputCallback)
	if err != nil {
		t.Fatalf("Failed to execute command: %v", err)
	}

	if !outputCalled {
		t.Error("Output callback was not called")
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}

	// Allow for some flexibility in duration comparison
	if result.Duration <= 0 {
		t.Error("Expected non-zero duration")
	}

	t.Logf("Command executed successfully with exit code %d", result.ExitCode)
}

// TestExecuteCommand_NotRunning tests the behavior when VM is not running
func TestExecuteCommand_NotRunning(t *testing.T) {
	// Skip if Vagrant is not installed
	if err := utils.CheckVagrantInstalled(); err != nil {
		t.Skip("Skipping test because Vagrant is not installed")
		return
	}

	// Skip by default - can be enabled with TEST_LEVEL=integration
	testLevel := os.Getenv("TEST_LEVEL")
	if testLevel != "integration" && testLevel != "vm-start" {
		t.Skip("Skipping integration test. Set TEST_LEVEL=integration to run")
		return
	}

	// Setup fixture without VM
	fixture, err := setupTestFixture(t, false)
	if err != nil {
		t.Fatalf("Failed to set up test fixture: %v", err)
	}
	defer fixture.cleanup()

	// Create VM but don't start it
	config := testsupport.GetVMConfig("minimal", fixture.baseFixture.ProjectPath)

	if err := fixture.VMManager.CreateVM(fixture.ctx, fixture.baseFixture.VMName, fixture.baseFixture.ProjectPath, config); err != nil {
		t.Fatalf("Failed to create VM: %v", err)
	}

	// Try to execute a command on the non-running VM
	execContext := ExecutionContext{
		VMName:     fixture.baseFixture.VMName,
		WorkingDir: "/vagrant",
	}

	outputCallback := func(data []byte, isStderr bool) {}

	// This should fail because VM is not running
	_, err = fixture.Executor.ExecuteCommand(context.Background(), "echo", execContext, outputCallback)
	if err == nil {
		t.Fatal("Expected error when VM is not running, but got none")
	}

	t.Logf("Got expected error when VM is not running: %v", err)
}
