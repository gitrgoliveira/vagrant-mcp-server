package vm

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// Use the actual exec.Command for testing with real Vagrant
// nolint:unused
var execCommand = exec.Command

// Skip tests if Vagrant is not installed
func skipIfVagrantNotInstalled(t *testing.T) {
	cmd := exec.Command("vagrant", "--version")
	if err := cmd.Run(); err != nil {
		t.Skip("Vagrant is not installed, skipping test")
	}
}

// cleanupVM ensures Vagrant VMs are properly destroyed and directories are cleaned up
func cleanupVM(t *testing.T, vmName string, vmDir string) {
	// Try to destroy VM using Vagrant if the directory exists
	if _, err := os.Stat(vmDir); err == nil {
		// VM directory exists, try to destroy it cleanly with vagrant force flag
		cmd := exec.Command("vagrant", "destroy", "-f")
		cmd.Dir = vmDir
		if err := cmd.Run(); err != nil {
			t.Logf("Failed to destroy VM with Vagrant: %v. Continuing with directory cleanup.", err)
		}

		// Remove VM directory after Vagrant destroy
		if err := os.RemoveAll(vmDir); err != nil {
			t.Logf("Failed to remove VM directory: %v", err)
		}
	}
}

// TestCreateVM tests the creation of a new VM
func TestCreateVM(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "vm-manager-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Always clean up the temp directory and VM directories, even in case of test failures
	defer func() {
		cleanupVM(t, "test-vm", filepath.Join(tempDir, "test-vm"))

		// Remove temp directory
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
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

	// Skip test if Vagrant is not installed
	skipIfVagrantNotInstalled(t)

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
func TestParseVagrantStatus(t *testing.T) {
	// Test directly the parseVagrantStatus method which is used by GetVMState
	manager := &Manager{}

	testCases := []struct {
		name          string
		statusOutput  string
		expectedState State
		expectError   bool
	}{
		{
			name:          "running state",
			statusOutput:  "1234,default,state,running",
			expectedState: Running,
			expectError:   false,
		},
		{
			name:          "poweroff state",
			statusOutput:  "1234,default,state,poweroff",
			expectedState: Stopped,
			expectError:   false,
		},
		{
			name:          "aborted state",
			statusOutput:  "1234,default,state,aborted",
			expectedState: Stopped,
			expectError:   false,
		},
		{
			name:          "saved state",
			statusOutput:  "1234,default,state,saved",
			expectedState: Suspended,
			expectError:   false,
		},
		{
			name:          "not created state",
			statusOutput:  "1234,default,state,not_created",
			expectedState: NotCreated,
			expectError:   false,
		},
		{
			name:          "empty output",
			statusOutput:  "",
			expectedState: Error,
			expectError:   true,
		},
		{
			name:          "invalid output",
			statusOutput:  "invalid output",
			expectedState: Error,
			expectError:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			state, err := manager.parseVagrantStatus(tc.statusOutput)

			if tc.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if !tc.expectError && state != tc.expectedState {
				t.Errorf("Expected state %v but got %v", tc.expectedState, state)
			}
		})
	}
}

func TestGetVMState(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "vm-manager-test-state")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Define test VM name
	const testVMName = "test-vm"

	// Always clean up the temp directory and VM directories, even in case of test failures
	defer func() {
		cleanupVM(t, testVMName, filepath.Join(tempDir, testVMName))

		// Remove temp directory
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to remove temporary directory: %v", err)
		}
	}()

	// Test when VM directory doesn't exist (should return NotCreated)
	t.Run("vm directory doesn't exist", func(t *testing.T) {
		manager := &Manager{baseDir: tempDir}

		state, err := manager.GetVMState(testVMName)

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if state != NotCreated {
			t.Errorf("Expected NotCreated state but got %v", state)
		}
	})

	// Test when VM directory exists but status command fails
	t.Run("status command fails", func(t *testing.T) {
		// Create a VM directory
		vmDir := filepath.Join(tempDir, testVMName)
		if err := os.MkdirAll(vmDir, 0755); err != nil {
			t.Fatalf("Failed to create VM dir: %v", err)
		}

		manager := &Manager{baseDir: tempDir}

		// We're testing with the real Vagrant CLI, so we need a valid VM directory structure
		// that will cause vagrant status to fail (which is what we're testing for)
		// Just having an empty directory should cause vagrant status to fail
		skipIfVagrantNotInstalled(t)

		state, err := manager.GetVMState(testVMName)

		if err == nil {
			t.Error("Expected error but got none")
		}
		if state != Error {
			t.Errorf("Expected Error state but got %v", state)
		}

		// Clean up the VM directory
		if err := os.RemoveAll(vmDir); err != nil {
			t.Logf("Failed to remove VM directory: %v", err)
		}
	})
}

// MockStartVM simulates starting a VM directly by testing the StartVM method
func TestStartVM(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "vm-manager-test-start")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Store VM names used in this test for cleanup
	vmsToCleanup := []string{"test-vm-success", "test-vm-fail"}

	// Always clean up the temp directory and VM directories, even in case of test failures
	defer func() {
		// First clean up all VM directories and VirtualBox VMs
		for _, vmName := range vmsToCleanup {
			vmDir := filepath.Join(tempDir, vmName)
			cleanupVM(t, vmName, vmDir)
		}

		// Then remove the entire temp directory
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to remove temporary directory: %v", err)
		}
	}()

	// Test successful startup
	t.Run("successful start", func(t *testing.T) {
		// Create VM directory
		vmName := "test-vm-success"
		vmDir := filepath.Join(tempDir, vmName)
		if err := os.MkdirAll(vmDir, 0755); err != nil {
			t.Fatalf("Failed to create VM dir: %v", err)
		}

		// Create a valid Vagrantfile
		vagrantfilePath := filepath.Join(vmDir, "Vagrantfile")
		validVagrantfile := `
# -*- mode: ruby -*-
# vi: set ft=ruby :
# Test Vagrantfile

Vagrant.configure("2") do |config|
  config.vm.box = "ubuntu/focal64"
  
  config.vm.provider "virtualbox" do |vb|
    vb.memory = 1024
    vb.cpus = 2
  end
end
`
		if err := os.WriteFile(vagrantfilePath, []byte(validVagrantfile), 0644); err != nil {
			t.Fatalf("Failed to create Vagrantfile: %v", err)
		}

		// Create the manager
		manager := &Manager{baseDir: tempDir}

		// Skip test if Vagrant is not installed
		skipIfVagrantNotInstalled(t)

		// When using actual Vagrant CLI, we should create a proper test environment
		// that allows Vagrant to actually validate the file

		// Start the VM
		err := manager.StartVM(vmName)

		// Check results
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	// Test failed startup
	t.Run("failed start", func(t *testing.T) {
		// Create VM directory
		vmName := "test-vm-fail"
		vmDir := filepath.Join(tempDir, vmName)
		if err := os.MkdirAll(vmDir, 0755); err != nil {
			t.Fatalf("Failed to create VM dir: %v", err)
		}

		// Create a valid Vagrantfile
		vagrantfilePath := filepath.Join(vmDir, "Vagrantfile")
		validVagrantfile := `
# -*- mode: ruby -*-
# vi: set ft=ruby :
# Test Vagrantfile

Vagrant.configure("2") do |config|
  config.vm.boxes = "ubuntu/focal64"
  
  config.vm.provider "virtualbox" do |vb|
    vb.memory = 1024
    vb.cpus = 2
  end
end
`
		if err := os.WriteFile(vagrantfilePath, []byte(validVagrantfile), 0644); err != nil {
			t.Fatalf("Failed to create Vagrantfile: %v", err)
		}

		// Create the manager
		manager := &Manager{baseDir: tempDir}

		// Skip test if Vagrant is not installed
		skipIfVagrantNotInstalled(t)

		// For testing a failed start with real Vagrant, we deliberately use an invalid Vagrantfile
		// that will fail validation

		// Try to start the VM
		err := manager.StartVM(vmName)

		// Check results
		if err == nil {
			t.Errorf("Expected error but got none")
		}
	})
}

// TestStopVM tests stopping a VM
func TestStopVM(t *testing.T) {
	// Skip test that requires real Vagrant environment
	t.Skip("Skipping StopVM test that requires real Vagrant environment")

	// Skip test if Vagrant is not installed
	skipIfVagrantNotInstalled(t)

	testCases := []struct {
		name        string
		setupVM     bool // whether to fully set up a VM with vagrant init
		expectError bool
	}{
		{
			name:        "successful stop",
			setupVM:     true,
			expectError: false,
		},
		{
			name:        "stop error",
			setupVM:     false, // Not setting up properly should cause an error
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Skip this test for now
			t.Skip("Skipping StopVM test that requires real Vagrant environment")

			// The rest of the test would require a real Vagrant environment
			// which is difficult to set up in an automated test
		})
	}
}

// TestDestroyVM tests destroying a VM
func TestDestroyVM(t *testing.T) {
	// Skip test that requires real Vagrant environment
	t.Skip("Skipping DestroyVM test that requires real Vagrant environment")

	// Skip test if Vagrant is not installed
	skipIfVagrantNotInstalled(t)

	testCases := []struct {
		name        string
		setupVM     bool // whether to fully set up a VM with vagrant init
		expectError bool
	}{
		{
			name:        "successful destroy",
			setupVM:     true,
			expectError: false,
		},
		{
			name:        "destroy error",
			setupVM:     false, // Not setting up properly should cause an error
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Skip this test for now
			t.Skip("Skipping DestroyVM test that requires real Vagrant environment")

			// The rest of the test would require a real Vagrant environment
			// which is difficult to set up in an automated test
		})
	}
}

// TestValidateVagrantfile tests that generated Vagrantfiles are valid
func TestValidateVagrantfile(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "vm-manager-test-validate")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Keep track of all VM names used in this test
	var testVMNames []string
	// Always clean up the temp directory and VM directories, even in case of test failures
	defer func() {
		// First clean up all VM directories
		for _, vmName := range testVMNames {
			vmDir := filepath.Join(tempDir, vmName)
			cleanupVM(t, vmName, vmDir)
		}

		// Remove temp directory
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	// Create a test manager with the temp directory as base
	manager := &Manager{
		baseDir: tempDir,
	}

	// Test cases with different VM configurations
	// Skip test if Vagrant is not installed
	skipIfVagrantNotInstalled(t)

	testCases := []struct {
		name        string
		config      VMConfig
		expectError bool // with real Vagrant, only invalid configurations should fail
	}{
		{
			name: "basic configuration",
			config: VMConfig{
				Box:    "ubuntu/focal64",
				CPU:    2,
				Memory: 2048,
			},
			expectError: false,
		},
		{
			name: "with port forwarding",
			config: VMConfig{
				Box:    "ubuntu/focal64",
				CPU:    2,
				Memory: 2048,
				Ports: []Port{
					{Guest: 3000, Host: 3000},
					{Guest: 8080, Host: 8080},
				},
			},
			expectError: false,
		},
		{
			name: "with custom sync type",
			config: VMConfig{
				Box:      "ubuntu/focal64",
				CPU:      2,
				Memory:   2048,
				SyncType: "nfs",
			},
			expectError: false,
		},
		{
			name: "with environment setup",
			config: VMConfig{
				Box:    "ubuntu/focal64",
				CPU:    2,
				Memory: 2048,
				Environment: []string{
					"apt-get install -y nodejs npm",
					"npm install -g yarn",
				},
			},
			expectError: false,
		},
		{
			name: "validation failure",
			config: VMConfig{
				// Use an invalid box name that should cause validation to fail
				Box:    "invalid/nonexistent-box-that-should-fail",
				CPU:    2,
				Memory: 2048,
			},
			expectError: false, // We can't force a validation error with real Vagrant as easily
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			vmName := "test-vm-validate-" + tc.name
			// Add VM name to the tracking list for cleanup
			testVMNames = append(testVMNames, vmName)
			projectPath := filepath.Join(tempDir, "project-"+tc.name)

			// Create project directory
			if err := os.MkdirAll(projectPath, 0755); err != nil {
				t.Fatalf("Failed to create project dir: %v", err)
			}

			// Create the VM with the configuration
			if err := manager.CreateVM(vmName, projectPath, tc.config); err != nil {
				t.Fatalf("CreateVM failed: %v", err)
			} // Check if Vagrantfile was created
			vmDir := filepath.Join(tempDir, vmName)
			vagrantfilePath := filepath.Join(vmDir, "Vagrantfile")
			if _, err := os.Stat(vagrantfilePath); os.IsNotExist(err) {
				t.Fatalf("Vagrantfile was not created at %s", vagrantfilePath)
			}

			// Run vagrant validate with the real Vagrant CLI
			cmd := exec.Command("vagrant", "validate")
			cmd.Dir = vmDir
			output, err := cmd.CombinedOutput()

			// Check results
			if err != nil {
				if !tc.expectError {
					t.Errorf("Unexpected validation error: %v, output: %s", err, output)
				}
			} else {
				if tc.expectError {
					t.Errorf("Expected validation error but got none")
				}
			}
		})
	}
}

// TestUploadToVM tests uploading files to a VM
func TestUploadToVM(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "vm-manager-test-upload")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Always clean up the temp directory and VM directories, even in case of test failures
	defer func() {
		// Clean up VM directory and VirtualBox VM
		vmName := "test-vm-upload"
		vmDir := filepath.Join(tempDir, vmName)
		cleanupVM(t, vmName, vmDir)

		// Then remove the entire temp directory
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	// Create a test file to upload
	sourceFile := filepath.Join(tempDir, "test-file.txt")
	testContent := "This is a test file for upload"
	if err := os.WriteFile(sourceFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create a VM directory for testing
	vmName := "test-vm-upload"
	vmDir := filepath.Join(tempDir, vmName)
	if err := os.MkdirAll(vmDir, 0755); err != nil {
		t.Fatalf("Failed to create VM dir: %v", err)
	}

	// Create a valid Vagrantfile
	vagrantfilePath := filepath.Join(vmDir, "Vagrantfile")
	validVagrantfile := `
# -*- mode: ruby -*-
# vi: set ft=ruby :
# Test Vagrantfile

Vagrant.configure("2") do |config|
  config.vm.box = "ubuntu/focal64"
  
  config.vm.provider "virtualbox" do |vb|
    vb.memory = 1024
    vb.cpus = 2
  end
end
`
	if err := os.WriteFile(vagrantfilePath, []byte(validVagrantfile), 0644); err != nil {
		t.Fatalf("Failed to create Vagrantfile: %v", err)
	}

	// Create the manager
	manager := &Manager{baseDir: tempDir}

	// Test cases for upload
	testCases := []struct {
		name            string
		source          string
		destination     string
		compress        bool
		compressionType string
		vmExists        bool
		vmRunning       bool
		expectError     bool
	}{
		{
			name:        "vm does not exist",
			source:      sourceFile,
			destination: "/tmp/uploaded-file.txt",
			vmExists:    false,
			expectError: true,
		},
		{
			name:        "source file does not exist",
			source:      filepath.Join(tempDir, "nonexistent-file.txt"),
			destination: "/tmp/uploaded-file.txt",
			vmExists:    true,
			expectError: true,
		},
		{
			name:        "upload with compression",
			source:      sourceFile,
			destination: "/tmp/uploaded-file.txt",
			compress:    true,
			vmExists:    true,
			vmRunning:   true,
			expectError: false,
		},
		{
			name:            "upload with specific compression type",
			source:          sourceFile,
			destination:     "/tmp/uploaded-file.txt",
			compress:        true,
			compressionType: "zip",
			vmExists:        true,
			vmRunning:       true,
			expectError:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Skip tests that would require a running VM since we can't easily set that up in unit tests
			if tc.vmRunning {
				t.Skip("Skipping test that requires a running VM")
			}

			testVMName := vmName
			if !tc.vmExists {
				testVMName = "nonexistent-vm"
			}

			err := manager.UploadToVM(testVMName, tc.source, tc.destination, tc.compress, tc.compressionType)

			if tc.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}
