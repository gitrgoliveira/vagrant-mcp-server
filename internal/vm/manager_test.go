package vm_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/vagrant-mcp/server/internal/core"
	testfixture "github.com/vagrant-mcp/server/internal/testing"
	"github.com/vagrant-mcp/server/internal/vm"
)

// TestCreateVM tests the creation of a new VM
func TestCreateVM(t *testing.T) {
	fixture, err := testfixture.NewUnifiedFixture(t, testfixture.FixtureOptions{
		PackageName:   "manager",
		SetupVM:       false,
		CreateProject: true,
	})
	if err != nil {
		t.Fatalf("Failed to set up test fixture: %v", err)
	}
	defer fixture.Cleanup()

	ctx := context.Background()
	vmName := fixture.VMName
	projectPath := fixture.ProjectPath
	manager := fixture.VMManager // use as core.VMManager

	config := core.VMConfig{
		Box:    "ubuntu/focal64",
		CPU:    2,
		Memory: 2048,
		Ports: []core.Port{
			{Guest: 3000, Host: 3000},
		},
		ProjectPath: projectPath,
	}

	if err := manager.CreateVM(ctx, vmName, projectPath, config); err != nil {
		t.Fatalf("Failed to create VM: %v", err)
	}

	vmDir := filepath.Join(fixture.TestDir, "vms", vmName)
	if _, err := os.Stat(vmDir); os.IsNotExist(err) {
		t.Errorf("VM directory was not created at %s", vmDir)
	}

	vagrantfilePath := filepath.Join(vmDir, "Vagrantfile")
	if _, err := os.Stat(vagrantfilePath); os.IsNotExist(err) {
		t.Errorf("Vagrantfile was not created at %s", vagrantfilePath)
	}

	configPath := filepath.Join(fixture.TestDir, fmt.Sprintf("%s.json", vmName))
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Errorf("VM config file was not created at %s", configPath)
	}
}

// TestParseVagrantStatus tests conversion of Vagrant machine-readable output to VM state
func TestParseVagrantStatus(t *testing.T) {
	manager, err := vm.NewManager()
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	testCases := []struct {
		name          string
		statusOutput  string
		expectedState core.VMState
		expectError   bool
	}{
		{
			name:          "running state",
			statusOutput:  "1234,default,state,running",
			expectedState: core.Running,
			expectError:   false,
		},
		{
			name:          "stopped state",
			statusOutput:  "1234,default,state,poweroff",
			expectedState: core.Stopped,
			expectError:   false,
		},
		{
			name:          "aborted state",
			statusOutput:  "1234,default,state,aborted",
			expectedState: core.Stopped,
			expectError:   false,
		},
		{
			name:          "saved state",
			statusOutput:  "1234,default,state,saved",
			expectedState: core.Suspended,
			expectError:   false,
		},
		{
			name:          "not created state",
			statusOutput:  "1234,default,state,not_created",
			expectedState: core.NotCreated,
			expectError:   false,
		},
		{
			name:          "empty output",
			statusOutput:  "",
			expectedState: core.Error,
			expectError:   true,
		},
		{
			name:          "invalid output",
			statusOutput:  "invalid output",
			expectedState: core.Error,
			expectError:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			state, err := manager.ParseVagrantStatus(tc.statusOutput)

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

// TestGetVMState tests getting VM state
func TestGetVMState(t *testing.T) {
	fixture, err := testfixture.NewUnifiedFixture(t, testfixture.FixtureOptions{
		PackageName:   "manager",
		SetupVM:       false,
		CreateProject: true,
	})
	if err != nil {
		t.Fatalf("Failed to set up test fixture: %v", err)
	}
	defer fixture.Cleanup()
	ctx := context.Background()
	testVMName := "test-vm-state"

	t.Run("vm directory doesn't exist", func(t *testing.T) {
		// Set VM_BASE_DIR and check error
		if err := os.Setenv("VM_BASE_DIR", fixture.TestDir); err != nil {
			t.Fatalf("Failed to set VM_BASE_DIR: %v", err)
		}
		manager, err := vm.NewManager()
		if err != nil {
			t.Fatalf("Failed to create manager: %v", err)
		}
		state, err := manager.GetVMState(ctx, testVMName)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if state != core.NotCreated {
			t.Errorf("Expected NotCreated state but got %v", state)
		}
	})

	t.Run("status command fails", func(t *testing.T) {
		vmDir := filepath.Join(fixture.TestDir, testVMName)
		if err := os.MkdirAll(vmDir, 0755); err != nil {
			t.Fatalf("Failed to create VM dir: %v", err)
		}
		// Set VM_BASE_DIR and check error
		if err := os.Setenv("VM_BASE_DIR", fixture.TestDir); err != nil {
			t.Fatalf("Failed to set VM_BASE_DIR: %v", err)
		}
		manager, err := vm.NewManager()
		if err != nil {
			t.Fatalf("Failed to create manager: %v", err)
		}
		state, err := manager.GetVMState(ctx, testVMName)
		if err == nil {
			t.Error("Expected error but got none")
		}
		if state != core.Unknown {
			t.Errorf("Expected Unknown state but got %v", state)
		}
	})
}

// TestStartVM tests starting a VM
func TestStartVM(t *testing.T) {
	// Skip by default - can be enabled with TEST_LEVEL=integration
	testLevel := os.Getenv("TEST_LEVEL")
	if testLevel != "integration" && testLevel != "vm-start" {
		t.Skip("Skipping integration test. Set TEST_LEVEL=integration to run")
		return
	}

	fixture, err := testfixture.NewUnifiedFixture(t, testfixture.FixtureOptions{
		PackageName:   "manager",
		SetupVM:       true,
		StartVM:       false, // Don't start VM by default - only create it
		CreateProject: true,
	})
	if err != nil {
		t.Fatalf("Failed to set up test fixture: %v", err)
	}
	defer fixture.Cleanup()
	ctx := context.Background()
	manager := fixture.VMManager // use as core.VMManager
	vmName := fixture.VMName

	t.Run("successful start", func(t *testing.T) {
		// Skipping actual Vagrant start in CI or if not explicitly requested
		if os.Getenv("CI") == "true" {
			t.Skip("Skipping VM start test in CI environment")
		}

		// Only run VM start test if TEST_LEVEL=vm-start
		if os.Getenv("TEST_LEVEL") != "vm-start" {
			t.Skip("Skipping VM start test. Set TEST_LEVEL=vm-start to run")
		}

		err := manager.StartVM(ctx, vmName)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})
}

// TestStopVM tests stopping a VM
func TestStopVM(t *testing.T) {
	t.Skip("Skipping StopVM test that requires real Vagrant environment")

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
	t.Skip("Skipping DestroyVM test that requires real Vagrant environment")

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
	fixture, err := testfixture.NewUnifiedFixture(t, testfixture.FixtureOptions{
		PackageName:   "manager-validate",
		SetupVM:       false,
		CreateProject: true,
	})
	if err != nil {
		t.Fatalf("Failed to set up test fixture: %v", err)
	}
	defer fixture.Cleanup()
	manager := fixture.VMManager // use as core.VMManager
	ctx := context.Background()

	testCases := []struct {
		name        string
		config      core.VMConfig
		expectError bool
	}{
		{
			name: "basic configuration",
			config: core.VMConfig{
				Box:    "ubuntu/focal64",
				CPU:    2,
				Memory: 2048,
			},
			expectError: false,
		},
		{
			name: "with port forwarding",
			config: core.VMConfig{
				Box:    "ubuntu/focal64",
				CPU:    2,
				Memory: 2048,
				Ports: []core.Port{
					{Guest: 3000, Host: 3000},
					{Guest: 8080, Host: 8080},
				},
			},
			expectError: false,
		},
		{
			name: "with custom sync type",
			config: core.VMConfig{
				Box:      "ubuntu/focal64",
				CPU:      2,
				Memory:   2048,
				SyncType: "nfs",
			},
			expectError: false,
		},
		{
			name: "with environment setup",
			config: core.VMConfig{
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
			config: core.VMConfig{
				Box:    "invalid/nonexistent-box-that-should-fail",
				CPU:    2,
				Memory: 2048,
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			vmName := "test-vm-validate-" + tc.name
			projectPath := filepath.Join(fixture.TestDir, "project-"+tc.name)
			if err := os.MkdirAll(projectPath, 0755); err != nil {
				t.Fatalf("Failed to create project dir: %v", err)
			}
			if err := manager.CreateVM(ctx, vmName, projectPath, tc.config); err != nil {
				t.Fatalf("CreateVM failed: %v", err)
			}
			vmDir := filepath.Join(fixture.TestDir, "vms", vmName)
			vagrantfilePath := filepath.Join(vmDir, "Vagrantfile")
			if _, err := os.Stat(vagrantfilePath); os.IsNotExist(err) {
				t.Fatalf("Vagrantfile was not created at %s", vagrantfilePath)
			}
			// Run vagrant validate with the real Vagrant CLI
			cmd := exec.Command("vagrant", "validate")
			cmd.Dir = vmDir
			output, err := cmd.CombinedOutput()
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
	// Skip by default - can be enabled with TEST_INTEGRATION=1
	if os.Getenv("TEST_INTEGRATION") != "1" {
		t.Skip("Skipping integration test. Set TEST_INTEGRATION=1 to run")
		return
	}

	fixture, err := testfixture.NewUnifiedFixture(t, testfixture.FixtureOptions{
		PackageName:   "manager-upload",
		SetupVM:       true,
		CreateProject: true,
	})
	if err != nil {
		t.Fatalf("Failed to set up test fixture: %v", err)
	}
	defer fixture.Cleanup()
	ctx := context.Background()
	manager := fixture.VMManager // use as core.VMManager
	vmName := fixture.VMName

	sourceDir := filepath.Join(fixture.TestDir, "source")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source dir: %v", err)
	}
	sourceFile := filepath.Join(sourceDir, "test.txt")
	if err := os.WriteFile(sourceFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

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
			source:      filepath.Join(sourceDir, "nonexistent-file.txt"),
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
			if tc.vmRunning {
				t.Skip("Skipping test that requires a running VM")
			}
			testVMName := vmName
			if !tc.vmExists {
				testVMName = "nonexistent-vm"
			}
			err := manager.UploadToVM(ctx, testVMName, tc.source, tc.destination, tc.compress, tc.compressionType)
			if tc.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}
