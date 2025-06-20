package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
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

// Add more tests for Execute methods with mocks for vm.Manager as needed.
