package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSyncToVMTool_Name(t *testing.T) {
	tool := &SyncToVMTool{}
	assert.Equal(t, "sync_to_vm", tool.Name())
}

func TestSyncFromVMTool_Name(t *testing.T) {
	tool := &SyncFromVMTool{}
	assert.Equal(t, "sync_from_vm", tool.Name())
}

func TestSyncStatusTool_Name(t *testing.T) {
	tool := &SyncStatusTool{}
	assert.Equal(t, "sync_status", tool.Name())
}

func TestResolveSyncConflictsTool_Name(t *testing.T) {
	tool := &ResolveSyncConflictsTool{}
	assert.Equal(t, "resolve_sync_conflicts", tool.Name())
}

// Add more tests for Execute methods with mocks for sync.Engine and vm.Manager as needed.
