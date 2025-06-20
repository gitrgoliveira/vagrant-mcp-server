package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExecInVMTool_Name(t *testing.T) {
	tool := &ExecInVMTool{}
	assert.Equal(t, "exec_in_vm", tool.Name())
}

func TestExecWithSyncTool_Name(t *testing.T) {
	tool := &ExecWithSyncTool{}
	assert.Equal(t, "exec_with_sync", tool.Name())
}

// Add more tests for Execute methods with mocks for exec.Executor, sync.Engine, and vm.Manager as needed.
