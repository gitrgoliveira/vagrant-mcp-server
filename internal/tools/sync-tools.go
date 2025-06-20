package tools

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/vagrant-mcp/server/internal/sync"
	"github.com/vagrant-mcp/server/internal/vm"
)

// ConfigureSyncTool implements the configure_sync tool
type ConfigureSyncTool struct {
	vmManager  *vm.Manager
	syncEngine *sync.Engine
}

// Name returns the tool name
func (t *ConfigureSyncTool) Name() string {
	return "configure_sync"
}

// Description returns the tool description
func (t *ConfigureSyncTool) Description() string {
	return "Configure sync method and options"
}

// Execute performs the tool action
func (t *ConfigureSyncTool) Execute(params map[string]interface{}) (interface{}, error) {
	// Validate required parameters
	vmName, ok := params["vm_name"].(string)
	if !ok || vmName == "" {
		return nil, fmt.Errorf("missing or invalid 'vm_name' parameter")
	}

	syncType, ok := params["sync_type"].(string)
	if !ok || syncType == "" {
		return nil, fmt.Errorf("missing or invalid 'sync_type' parameter")
	}

	// Get optional parameters
	hostPath, _ := params["host_path"].(string)
	guestPath, _ := params["guest_path"].(string)

	// Get exclude patterns
	excludePatterns := []string{}
	if excludePatternsList, ok := params["exclude_patterns"].([]interface{}); ok {
		for _, pattern := range excludePatternsList {
			if patternStr, ok := pattern.(string); ok {
				excludePatterns = append(excludePatterns, patternStr)
			}
		}
	} else {
		// Default exclude patterns
		excludePatterns = []string{
			"node_modules",
			".git",
			"dist",
			"build",
			"*.log",
			"*.tmp",
		}
	}

	// Get sync direction
	bidirectional := true
	if bidirValue, ok := params["bidirectional"].(bool); ok {
		bidirectional = bidirValue
	}

	// Set sync direction
	var direction sync.SyncDirection
	if bidirectional {
		direction = sync.SyncBidirectional
	} else {
		direction = sync.SyncToVM
	}

	// Convert sync type
	var method sync.SyncMethod
	switch syncType {
	case "rsync":
		method = sync.SyncMethodRsync
	case "nfs":
		method = sync.SyncMethodNFS
	case "smb":
		method = sync.SyncMethodSMB
	case "virtualbox":
		method = sync.SyncMethodVirtualBox
	default:
		return nil, fmt.Errorf("invalid sync type: %s", syncType)
	}

	// Check if VM exists by trying to get its state
	_, err := t.vmManager.GetVMState(vmName)
	if err != nil {
		return nil, fmt.Errorf("VM '%s' does not exist or cannot be accessed: %w", vmName, err)
	}

	// Get project path for the VM
	config, err := t.vmManager.GetVMConfig(vmName)
	if err != nil {
		return nil, fmt.Errorf("failed to get VM config: %w", err)
	}

	projectPath := config.ProjectPath
	if hostPath == "" {
		hostPath = projectPath
	}

	if guestPath == "" {
		guestPath = "/vagrant"
	}

	// Configure sync
	syncConfig := sync.SyncConfig{
		VMName:          vmName,
		ProjectPath:     projectPath,
		Method:          method,
		Direction:       direction,
		ExcludePatterns: excludePatterns,
		WatchEnabled:    true,
	}

	if err := t.syncEngine.ConfigureSync(syncConfig); err != nil {
		return nil, fmt.Errorf("failed to configure sync: %w", err)
	}

	return map[string]interface{}{
		"status":           "configured",
		"vm_name":          vmName,
		"sync_type":        syncType,
		"host_path":        hostPath,
		"guest_path":       guestPath,
		"bidirectional":    bidirectional,
		"exclude_patterns": excludePatterns,
	}, nil
}

// SyncToVMTool implements the sync_to_vm tool
type SyncToVMTool struct {
	vmManager  *vm.Manager
	syncEngine *sync.Engine
}

// Name returns the tool name
func (t *SyncToVMTool) Name() string {
	return "sync_to_vm"
}

// Description returns the tool description
func (t *SyncToVMTool) Description() string {
	return "Synchronize files from host to VM"
}

// Execute performs the tool action
func (t *SyncToVMTool) Execute(params map[string]interface{}) (interface{}, error) {
	// Validate required parameters
	vmName, ok := params["vm_name"].(string)
	if !ok || vmName == "" {
		return nil, fmt.Errorf("missing or invalid 'vm_name' parameter")
	}

	// Check if VM exists and is running
	state, err := t.vmManager.GetVMState(vmName)
	if err != nil {
		return nil, fmt.Errorf("VM '%s' does not exist or cannot be accessed: %w", vmName, err)
	}

	if state != vm.StateRunning {
		return nil, fmt.Errorf("VM '%s' is not running (current state: %s)", vmName, state)
	}

	// Get VM config
	config, err := t.vmManager.GetVMConfig(vmName)
	if err != nil {
		return nil, fmt.Errorf("failed to get VM config: %w", err)
	}

	// Get optional path parameter
	path := ""
	if pathParam, ok := params["path"].(string); ok {
		path = pathParam
	}

	// Get exclude patterns
	excludePatterns := []string{}
	if excludePatternsList, ok := params["exclude_patterns"].([]interface{}); ok {
		for _, pattern := range excludePatternsList {
			if patternStr, ok := pattern.(string); ok {
				excludePatterns = append(excludePatterns, patternStr)
			}
		}
	} else {
		// Default exclude patterns
		excludePatterns = []string{
			".git",
			"node_modules",
			"dist",
			".vagrant",
			"__pycache__",
			"*.pyc",
			"*.pyo",
			".DS_Store",
		}
	}

	// Configure sync
	syncConfig := sync.SyncConfig{
		VMName:          vmName,
		ProjectPath:     config.ProjectPath,
		Method:          sync.SyncMethod(config.SyncType),
		Direction:       sync.SyncToVM,
		ExcludePatterns: excludePatterns,
	}

	// Perform sync
	log.Info().
		Str("vm", vmName).
		Str("project", config.ProjectPath).
		Str("path", path).
		Str("direction", "host->vm").
		Msg("Synchronizing files")

	status, err := t.syncEngine.Sync(syncConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to sync files: %w", err)
	}

	// Return status
	return map[string]interface{}{
		"vm_name":            vmName,
		"direction":          "host->vm",
		"last_sync_time":     status.LastSyncTime,
		"synchronized_files": status.SynchronizedFiles,
		"conflicts":          status.Conflicts,
	}, nil
}

// SyncFromVMTool implements the sync_from_vm tool
type SyncFromVMTool struct {
	vmManager  *vm.Manager
	syncEngine *sync.Engine
}

// Name returns the tool name
func (t *SyncFromVMTool) Name() string {
	return "sync_from_vm"
}

// Description returns the tool description
func (t *SyncFromVMTool) Description() string {
	return "Synchronize files from VM to host"
}

// Execute performs the tool action
func (t *SyncFromVMTool) Execute(params map[string]interface{}) (interface{}, error) {
	// Validate required parameters
	vmName, ok := params["vm_name"].(string)
	if !ok || vmName == "" {
		return nil, fmt.Errorf("missing or invalid 'vm_name' parameter")
	}

	// Check if VM exists and is running
	state, err := t.vmManager.GetVMState(vmName)
	if err != nil {
		return nil, fmt.Errorf("VM '%s' does not exist or cannot be accessed: %w", vmName, err)
	}

	if state != vm.StateRunning {
		return nil, fmt.Errorf("VM '%s' is not running (current state: %s)", vmName, state)
	}

	// Get VM config
	config, err := t.vmManager.GetVMConfig(vmName)
	if err != nil {
		return nil, fmt.Errorf("failed to get VM config: %w", err)
	}

	// Get optional path parameter
	path := ""
	if pathParam, ok := params["path"].(string); ok {
		path = pathParam
	}

	// Get exclude patterns
	excludePatterns := []string{}
	if excludePatternsList, ok := params["exclude_patterns"].([]interface{}); ok {
		for _, pattern := range excludePatternsList {
			if patternStr, ok := pattern.(string); ok {
				excludePatterns = append(excludePatterns, patternStr)
			}
		}
	} else {
		// Default exclude patterns
		excludePatterns = []string{
			".git",
			"node_modules",
			"dist",
			".vagrant",
			"__pycache__",
			"*.pyc",
			"*.pyo",
			".DS_Store",
		}
	}

	// Configure sync
	syncConfig := sync.SyncConfig{
		VMName:          vmName,
		ProjectPath:     config.ProjectPath,
		Method:          sync.SyncMethod(config.SyncType),
		Direction:       sync.SyncFromVM,
		ExcludePatterns: excludePatterns,
	}

	// Perform sync
	log.Info().
		Str("vm", vmName).
		Str("project", config.ProjectPath).
		Str("path", path).
		Str("direction", "vm->host").
		Msg("Synchronizing files")

	status, err := t.syncEngine.Sync(syncConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to sync files: %w", err)
	}

	// Return status
	return map[string]interface{}{
		"vm_name":            vmName,
		"direction":          "vm->host",
		"last_sync_time":     status.LastSyncTime,
		"synchronized_files": status.SynchronizedFiles,
		"conflicts":          status.Conflicts,
	}, nil
}

// SyncStatusTool implements the sync_status tool
type SyncStatusTool struct {
	vmManager  *vm.Manager
	syncEngine *sync.Engine
}

// Name returns the tool name
func (t *SyncStatusTool) Name() string {
	return "sync_status"
}

// Description returns the tool description
func (t *SyncStatusTool) Description() string {
	return "Check synchronization status and detect conflicts"
}

// Execute performs the tool action
func (t *SyncStatusTool) Execute(params map[string]interface{}) (interface{}, error) {
	// Validate required parameters
	vmName, ok := params["vm_name"].(string)
	if !ok || vmName == "" {
		return nil, fmt.Errorf("missing or invalid 'vm_name' parameter")
	}

	// Check if VM exists
	_, err := t.vmManager.GetVMState(vmName)
	if err != nil {
		return nil, fmt.Errorf("VM '%s' does not exist or cannot be accessed: %w", vmName, err)
	}

	// Get sync status
	status, err := t.syncEngine.GetSyncStatus(vmName)
	if err != nil {
		return nil, fmt.Errorf("failed to get sync status: %w", err)
	}

	// Return status
	return map[string]interface{}{
		"vm_name":         vmName,
		"in_progress":     status.InProgress,
		"last_sync_time":  status.LastSyncTime,
		"conflicts":       status.Conflicts,
		"has_conflicts":   len(status.Conflicts) > 0,
		"conflicts_count": len(status.Conflicts),
	}, nil
}

// ResolveSyncConflictsTool implements the resolve_sync_conflicts tool
type ResolveSyncConflictsTool struct {
	vmManager  *vm.Manager
	syncEngine *sync.Engine
}

// Name returns the tool name
func (t *ResolveSyncConflictsTool) Name() string {
	return "resolve_sync_conflicts"
}

// Description returns the tool description
func (t *ResolveSyncConflictsTool) Description() string {
	return "Resolve synchronization conflicts between host and VM"
}

// Execute performs the tool action
func (t *ResolveSyncConflictsTool) Execute(params map[string]interface{}) (interface{}, error) {
	// Validate required parameters
	vmName, ok := params["vm_name"].(string)
	if !ok || vmName == "" {
		return nil, fmt.Errorf("missing or invalid 'vm_name' parameter")
	}

	strategy, ok := params["strategy"].(string)
	if !ok || strategy == "" {
		return nil, fmt.Errorf("missing or invalid 'strategy' parameter")
	}

	// Check if VM exists
	_, err := t.vmManager.GetVMState(vmName)
	if err != nil {
		return nil, fmt.Errorf("VM '%s' does not exist or cannot be accessed: %w", vmName, err)
	}

	// Get current conflicts before resolution
	statusBefore, err := t.syncEngine.GetSyncStatus(vmName)
	if err != nil {
		return nil, fmt.Errorf("failed to get sync status: %w", err)
	}

	conflictsBefore := len(statusBefore.Conflicts)

	// Resolve conflicts
	if err := t.syncEngine.ResolveConflicts(vmName, strategy); err != nil {
		return nil, fmt.Errorf("failed to resolve conflicts: %w", err)
	}

	// Get updated status after resolution
	statusAfter, err := t.syncEngine.GetSyncStatus(vmName)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated sync status: %w", err)
	}

	// Return resolution results
	return map[string]interface{}{
		"vm_name":             vmName,
		"strategy":            strategy,
		"conflicts_before":    conflictsBefore,
		"conflicts_resolved":  conflictsBefore - len(statusAfter.Conflicts),
		"conflicts_remaining": len(statusAfter.Conflicts),
		"success":             len(statusAfter.Conflicts) == 0,
	}, nil
}
