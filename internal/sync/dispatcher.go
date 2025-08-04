// Package sync provides synchronization utilities including method dispatching
package sync

import (
	"fmt"
)

// SyncMethodDispatcher handles method dispatch for different sync types
type SyncMethodDispatcher struct {
	engine *Engine
}

// NewSyncMethodDispatcher creates a new sync method dispatcher
func NewSyncMethodDispatcher(engine *Engine) *SyncMethodDispatcher {
	return &SyncMethodDispatcher{
		engine: engine,
	}
}

// DispatchSyncMethod dispatches sync operation based on method and direction
func (d *SyncMethodDispatcher) DispatchSyncMethod(method SyncMethod, vmName, sourcePath string, toVM bool) ([]string, error) {
	switch method {
	case SyncMethodRsync:
		return d.engine.syncWithRsync(vmName, sourcePath, toVM)
	case SyncMethodNFS:
		return d.engine.syncWithNFS(vmName, sourcePath, toVM)
	case SyncMethodSMB:
		return d.engine.syncWithSMB(vmName, sourcePath, toVM)
	default:
		return nil, fmt.Errorf("unsupported sync method: %s", method)
	}
}

// GetSupportedMethods returns a list of supported sync methods
func (d *SyncMethodDispatcher) GetSupportedMethods() []SyncMethod {
	return []SyncMethod{
		SyncMethodRsync,
		SyncMethodNFS,
		SyncMethodSMB,
		SyncMethodVirtualBox,
	}
}

// IsMethodSupported checks if a sync method is supported
func (d *SyncMethodDispatcher) IsMethodSupported(method SyncMethod) bool {
	for _, supported := range d.GetSupportedMethods() {
		if method == supported {
			return true
		}
	}
	return false
}
