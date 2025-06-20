package vm

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// TestCreateVM tests the creation of a new VM
func TestCreateVM(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "vm-manager-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Errorf("Failed to remove temp dir: %v", err)
		}
	}()

	// Create a test manager with the temp directory as base
	manager := &Manager{
		baseDir: tempDir,
	}

	// Test creating a VM
	vmName := "test-vm"
	projectPath := filepath.Join(tempDir, "project")

	// Create project directory
	if err := os.MkdirAll(projectPath, 0755); err != nil {
		t.Fatalf("Failed to create project dir: %v", err)
	}

	// Create test VM config
	config := VMConfig{
		Box:    "ubuntu/focal64",
		CPU:    2,
		Memory: 2048,
		Ports: []Port{
			{Guest: 3000, Host: 3000},
		},
	}

	// Test VM creation
	if err := manager.CreateVM(vmName, projectPath, config); err != nil {
		t.Fatalf("CreateVM failed: %v", err)
	}

	// Check if VM directory was created
	vmDir := filepath.Join(tempDir, vmName)
	if _, err := os.Stat(vmDir); os.IsNotExist(err) {
		t.Errorf("VM directory was not created at %s", vmDir)
	}

	// Check if Vagrantfile was created
	vagrantfilePath := filepath.Join(vmDir, "Vagrantfile")
	if _, err := os.Stat(vagrantfilePath); os.IsNotExist(err) {
		t.Errorf("Vagrantfile was not created at %s", vagrantfilePath)
	}

	// Check if VM config was saved
	configPath := filepath.Join(filepath.Dir(tempDir), fmt.Sprintf("%s.json", vmName))
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Errorf("VM config file was not created at %s", configPath)
	}
}

// TestGetVMState tests retrieving VM state
func TestGetVMState(t *testing.T) {
	// This would require mocking the shell commands
	// In a real test, we would mock exec.Command
	t.Skip("Skipping VM state test as it requires mocking shell commands")
}

// TestStartVM tests starting a VM
func TestStartVM(t *testing.T) {
	// This would require mocking the shell commands
	t.Skip("Skipping VM start test as it requires mocking shell commands")
}

// TestStopVM tests stopping a VM
func TestStopVM(t *testing.T) {
	// This would require mocking the shell commands
	t.Skip("Skipping VM stop test as it requires mocking shell commands")
}

// TestDestroyVM tests destroying a VM
func TestDestroyVM(t *testing.T) {
	// This would require mocking the shell commands
	t.Skip("Skipping VM destroy test as it requires mocking shell commands")
}
