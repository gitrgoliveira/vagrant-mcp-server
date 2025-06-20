package tools

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vagrant-mcp/server/internal/vm"
)

func TestCreateDevVMTool_Name(t *testing.T) {
	tool := &CreateDevVMTool{}
	assert.Equal(t, "create_dev_vm", tool.Name())
}

func TestEnsureDevVMTool_Name(t *testing.T) {
	tool := &EnsureDevVMTool{}
	assert.Equal(t, "ensure_dev_vm", tool.Name())
}

func TestDestroyDevVMTool_Name(t *testing.T) {
	tool := &DestroyDevVMTool{}
	assert.Equal(t, "destroy_dev_vm", tool.Name())
}

// --- Mocks ---
type mockVMManager struct {
	CreateVMFunc   func(name, projectPath string, config vm.VMConfig) error
	StartVMFunc    func(name string) error
	GetVMStateFunc func(name string) (vm.State, error)
}

func (m *mockVMManager) CreateVM(name, projectPath string, config vm.VMConfig) error {
	if m.CreateVMFunc != nil {
		return m.CreateVMFunc(name, projectPath, config)
	}
	return nil
}
func (m *mockVMManager) StartVM(name string) error {
	if m.StartVMFunc != nil {
		return m.StartVMFunc(name)
	}
	return nil
}
func (m *mockVMManager) GetVMState(name string) (vm.State, error) {
	if m.GetVMStateFunc != nil {
		return m.GetVMStateFunc(name)
	}
	return vm.StateRunning, nil
}

func TestCreateDevVMTool_Execute_Success(t *testing.T) {
	mockMgr := &mockVMManager{
		CreateVMFunc: func(name, projectPath string, config vm.VMConfig) error { return nil },
		StartVMFunc:  func(name string) error { return nil },
		GetVMStateFunc: func(name string) (vm.State, error) {
			return vm.StateRunning, nil
		},
	}
	tool := &CreateDevVMTool{manager: mockMgr}
	params := map[string]interface{}{
		"name":         "testvm",
		"project_path": "/tmp/project",
		"box":          "ubuntu/focal64",
		"cpu":          2,
		"memory":       2048,
		"sync_type":    "rsync",
	}
	result, err := tool.Execute(params)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateDevVMTool_Execute_Error(t *testing.T) {
	errMsg := "failed to create VM"
	mockMgr := &mockVMManager{
		CreateVMFunc: func(name, projectPath string, config vm.VMConfig) error {
			return fmt.Errorf("%s", errMsg)
		},
		StartVMFunc:    func(name string) error { return nil },
		GetVMStateFunc: func(name string) (vm.State, error) { return vm.StateRunning, nil },
	}
	tool := &CreateDevVMTool{manager: mockMgr}
	params := map[string]interface{}{
		"name":         "testvm",
		"project_path": "/tmp/project",
		"box":          "ubuntu/focal64",
		"cpu":          2,
		"memory":       2048,
		"sync_type":    "rsync",
	}
	result, err := tool.Execute(params)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), errMsg)
	assert.Nil(t, result)
}

// Add more tests for Execute methods with mocks for vm.Manager as needed.
