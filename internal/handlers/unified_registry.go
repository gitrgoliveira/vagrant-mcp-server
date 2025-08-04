// Package handlers provides unified handler registration utilities
package handlers

import (
	"github.com/mark3labs/mcp-go/server"
	"github.com/vagrant-mcp/server/internal/core"
	"github.com/vagrant-mcp/server/internal/exec"
)

// HandlerRegistry provides unified handler registration functionality
type HandlerRegistry struct {
	vmManager  core.VMManager
	syncEngine core.SyncEngine
	executor   *exec.Executor
}

// NewHandlerRegistry creates a new handler registry
func NewHandlerRegistry(vmManager core.VMManager, syncEngine core.SyncEngine, executor *exec.Executor) *HandlerRegistry {
	return &HandlerRegistry{
		vmManager:  vmManager,
		syncEngine: syncEngine,
		executor:   executor,
	}
}

// RegisterAllTools registers all handler groups using existing functions
func (r *HandlerRegistry) RegisterAllTools(srv *server.MCPServer) {
	// Use existing registration functions but centralize the call
	RegisterVMTools(srv, r.vmManager, r.syncEngine)
	RegisterSyncTools(srv, r.syncEngine, r.vmManager)
	RegisterExecTools(srv, r.vmManager, r.syncEngine, r.executor)
	RegisterEnvTools(srv, r.vmManager, r.executor)
}
