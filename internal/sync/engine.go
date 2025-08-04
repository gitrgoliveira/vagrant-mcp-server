// Copyright Ricardo Oliveira 2025.
// SPDX-License-Identifier: MPL-2.0

package sync

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog/log"
	"github.com/vagrant-mcp/server/internal/errors"
)

// SyncDirection represents the direction of synchronization
type SyncDirection int

const (
	// SyncToVM represents synchronization from host to VM
	SyncToVM SyncDirection = iota
	// SyncFromVM represents synchronization from VM to host
	SyncFromVM
	// SyncBidirectional represents bidirectional synchronization
	SyncBidirectional
)

// SyncMethod represents the method used for synchronization
type SyncMethod string

const (
	// SyncMethodRsync uses rsync for synchronization
	SyncMethodRsync SyncMethod = "rsync"
	// SyncMethodNFS uses NFS for synchronization
	SyncMethodNFS SyncMethod = "nfs"
	// SyncMethodSMB uses SMB for synchronization
	SyncMethodSMB SyncMethod = "smb"
	// SyncMethodVirtualBox uses VirtualBox shared folders
	SyncMethodVirtualBox SyncMethod = "virtualbox"
)

// SyncConfig represents the configuration for file synchronization
type SyncConfig struct {
	VMName          string        `json:"vm_name"`
	ProjectPath     string        `json:"project_path"`
	Method          SyncMethod    `json:"method"`
	Direction       SyncDirection `json:"direction"`
	ExcludePatterns []string      `json:"exclude_patterns"`
	WatchEnabled    bool          `json:"watch_enabled"`
	WatchInterval   time.Duration `json:"watch_interval"`
}

// SyncResult represents the result of a synchronization operation
type SyncResult struct {
	SyncedFiles []string `json:"synced_files"`
	SyncTimeMs  int      `json:"sync_time_ms"`
}

// SyncStatus represents the status of a synchronization operation
type SyncStatus struct {
	LastSyncTime         time.Time      `json:"last_sync_time"`
	InProgress           bool           `json:"in_progress"`
	Conflicts            []SyncConflict `json:"conflicts"`
	SynchronizedFiles    int            `json:"synchronized_files"`
	Error                string         `json:"error,omitempty"`
	LastSyncToVM         time.Time      `json:"last_sync_to_vm"`
	LastSyncFromVM       time.Time      `json:"last_sync_from_vm"`
	FilesPendingUpload   []string       `json:"files_pending_upload"`
	FilesPendingDownload []string       `json:"files_pending_download"`
	TotalSyncs           int            `json:"total_syncs"`
	TotalFilesSynced     int            `json:"total_files_synced"`
	TotalSyncTimeMs      int            `json:"total_sync_time_ms"`
}

// SyncConflict represents a file conflict during synchronization
type SyncConflict struct {
	Path         string    `json:"path"`
	HostModTime  time.Time `json:"host_mod_time"`
	VMModTime    time.Time `json:"vm_mod_time"`
	HostContent  string    `json:"host_content,omitempty"` // Content of the file on host
	VMContent    string    `json:"vm_content,omitempty"`   // Content of the file on VM
	ConflictType string    `json:"conflict_type"`          // "modification", "deletion", "creation"
}

// SearchResult represents a search result from the VM
type SearchResult struct {
	Path      string `json:"path"`
	Line      int    `json:"line"`
	Content   string `json:"content"`
	MatchType string `json:"match_type"` // "exact", "fuzzy", "semantic"
}

// Engine handles file synchronization between host and VM
type Engine struct {
	configs       map[string]SyncConfig
	statuses      map[string]SyncStatus
	watchers      map[string]*fsnotify.Watcher
	watcherStopCh map[string]chan struct{}
	mu            sync.RWMutex
	running       bool
	vmManager     VMManager             // Reference to the VM Manager for Vagrant commands
	dispatcher    *SyncMethodDispatcher // Method dispatcher
}

// VMManager interface defines the methods required from a VM Manager
type VMManager interface {
	GetBaseDir() string
	SyncToVM(name, source, target string) error
	SyncFromVM(name, source, target string) error
}

// NewEngine creates a new synchronization engine
func NewEngine() (*Engine, error) {
	engine := &Engine{
		configs:       make(map[string]SyncConfig),
		statuses:      make(map[string]SyncStatus),
		watchers:      make(map[string]*fsnotify.Watcher),
		watcherStopCh: make(map[string]chan struct{}),
	}

	// Initialize the dispatcher
	engine.dispatcher = NewSyncMethodDispatcher(engine)

	return engine, nil
}

// SetVMManager sets the VM manager for the sync engine
// This must be called before any sync operations
func (e *Engine) SetVMManager(vmManager VMManager) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.vmManager = vmManager
}

// RegisterVM registers a VM with the sync engine
func (e *Engine) RegisterVM(vmName string, config SyncConfig) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Validate VM name
	if vmName == "" {
		return ErrInvalidVMName
	}

	// Check if already registered
	if _, exists := e.configs[vmName]; exists {
		return ErrVMAlreadyRegistered
	}

	// Set default values if not provided
	if config.Method == "" {
		config.Method = SyncMethodRsync
	}
	if config.Direction == 0 {
		config.Direction = SyncBidirectional
	}
	if config.WatchInterval == 0 {
		config.WatchInterval = 5 * time.Second
	}

	// Store config
	config.VMName = vmName
	e.configs[vmName] = config

	// Initialize status
	e.statuses[vmName] = SyncStatus{
		LastSyncTime: time.Now(),
		InProgress:   false,
		Conflicts:    []SyncConflict{},
	}

	// Start file watcher if enabled
	if config.WatchEnabled {
		if err := e.startWatcher(vmName); err != nil {
			log.Error().Err(err).Str("vm", vmName).Msg("Failed to start file watcher")
		}
	}

	log.Info().Str("vm", vmName).Msg("VM registered with sync engine")
	return nil
}

// UnregisterVM unregisters a VM from the sync engine
func (e *Engine) UnregisterVM(vmName string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Validate VM name
	if vmName == "" {
		return ErrInvalidVMName
	}

	// Check if registered
	if _, exists := e.configs[vmName]; !exists {
		return ErrVMNotRegistered
	}

	// Stop watcher if running
	if watcher, exists := e.watchers[vmName]; exists {
		stopCh := e.watcherStopCh[vmName]
		close(stopCh)
		if err := watcher.Close(); err != nil {
			log.Warn().Err(err).Msg("Failed to close watcher")
		}
		delete(e.watchers, vmName)
		delete(e.watcherStopCh, vmName)
	}

	// Remove config and status
	delete(e.configs, vmName)
	delete(e.statuses, vmName)

	log.Info().Str("vm", vmName).Msg("VM unregistered from sync engine")
	return nil
}

// SyncToVM synchronizes files from host to VM
func (e *Engine) SyncToVM(vmName string, sourcePath string) (*SyncResult, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Validate VM name
	if vmName == "" {
		return nil, ErrInvalidVMName
	}

	// Check if registered
	config, exists := e.configs[vmName]
	if !exists {
		return nil, ErrVMNotRegistered
	}

	// Update status
	status := e.statuses[vmName]
	status.InProgress = true
	e.statuses[vmName] = status

	// Determine source path
	if sourcePath == "" {
		sourcePath = config.ProjectPath
	}

	// Ensure source path exists
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		status.InProgress = false
		errMsg := fmt.Sprintf("Source path does not exist: %s", sourcePath)
		status.Error = errMsg
		e.statuses[vmName] = status
		return nil, errors.OperationFailed("sync operation", fmt.Errorf("%s", errMsg))
	}

	// Start timer
	startTime := time.Now()

	// Perform sync based on method
	syncedFiles, err := e.dispatcher.DispatchSyncMethod(config.Method, vmName, sourcePath, true)
	if err != nil {
		return nil, errors.OperationFailed("sync to VM", err)
	}

	// Calculate sync time
	syncTime := time.Since(startTime)
	syncTimeMs := int(syncTime.Milliseconds())

	// Update status
	status = e.statuses[vmName]
	status.InProgress = false
	status.LastSyncTime = time.Now()
	status.LastSyncToVM = time.Now()
	status.TotalSyncs++
	status.TotalSyncTimeMs += syncTimeMs

	status.SynchronizedFiles = len(syncedFiles)
	status.TotalFilesSynced += len(syncedFiles)
	status.Error = ""
	e.statuses[vmName] = status

	// Return result
	return &SyncResult{
		SyncedFiles: syncedFiles,
		SyncTimeMs:  syncTimeMs,
	}, nil
}

// SyncFromVM synchronizes files from VM to host
func (e *Engine) SyncFromVM(vmName string, sourcePath string) (*SyncResult, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Validate VM name
	if vmName == "" {
		return nil, ErrInvalidVMName
	}

	// Check if registered
	config, exists := e.configs[vmName]
	if !exists {
		return nil, ErrVMNotRegistered
	}

	// Update status
	status := e.statuses[vmName]
	status.InProgress = true
	e.statuses[vmName] = status

	// Determine source path
	if sourcePath == "" {
		sourcePath = "/vagrant"
	}

	// Start timer
	startTime := time.Now()

	// Perform sync based on method
	var err error
	var syncedFiles []string

	// Perform sync based on method using dispatcher
	syncedFiles, err = e.dispatcher.DispatchSyncMethod(config.Method, vmName, sourcePath, false)
	if err != nil {
		return nil, errors.OperationFailed("sync from VM", err)
	}

	// Calculate sync time
	syncTime := time.Since(startTime)
	syncTimeMs := int(syncTime.Milliseconds())

	// Update status
	status = e.statuses[vmName]
	status.InProgress = false
	status.LastSyncTime = time.Now()
	status.LastSyncFromVM = time.Now()
	status.TotalSyncs++
	status.TotalSyncTimeMs += syncTimeMs

	status.SynchronizedFiles = len(syncedFiles)
	status.TotalFilesSynced += len(syncedFiles)
	status.Error = ""
	e.statuses[vmName] = status

	// Return result
	return &SyncResult{
		SyncedFiles: syncedFiles,
		SyncTimeMs:  syncTimeMs,
	}, nil
}

// GetSyncStatus returns the sync status for a VM
func (e *Engine) GetSyncStatus(vmName string) (SyncStatus, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Validate VM name
	if vmName == "" {
		return SyncStatus{}, ErrInvalidVMName
	}

	// Check if registered
	status, exists := e.statuses[vmName]
	if !exists {
		return SyncStatus{}, ErrVMNotRegistered
	}

	return status, nil
}

// GetSyncConfig returns the sync configuration for a VM
func (e *Engine) GetSyncConfig(vmName string) (SyncConfig, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Validate VM name
	if vmName == "" {
		return SyncConfig{}, ErrInvalidVMName
	}

	// Check if registered
	config, exists := e.configs[vmName]
	if !exists {
		return SyncConfig{}, ErrVMNotRegistered
	}

	return config, nil
}

// UpdateSyncConfig updates the sync configuration for a VM
func (e *Engine) UpdateSyncConfig(vmName string, config SyncConfig) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Validate VM name
	if vmName == "" {
		return ErrInvalidVMName
	}

	// Check if registered
	oldConfig, exists := e.configs[vmName]
	if !exists {
		return ErrVMNotRegistered
	}

	// Update config
	config.VMName = vmName
	if config.ProjectPath == "" {
		config.ProjectPath = oldConfig.ProjectPath
	}
	if config.Method == "" {
		config.Method = oldConfig.Method
	}
	if config.Direction == 0 {
		config.Direction = oldConfig.Direction
	}
	if config.WatchInterval == 0 {
		config.WatchInterval = oldConfig.WatchInterval
	}
	if len(config.ExcludePatterns) == 0 {
		config.ExcludePatterns = oldConfig.ExcludePatterns
	}

	e.configs[vmName] = config

	// Restart watcher if watching enabled/disabled
	if oldConfig.WatchEnabled != config.WatchEnabled {
		if config.WatchEnabled {
			if err := e.startWatcher(vmName); err != nil {
				log.Error().Err(err).Str("vm", vmName).Msg("Failed to start file watcher")
			}
		} else if watcher, exists := e.watchers[vmName]; exists {
			stopCh := e.watcherStopCh[vmName]
			close(stopCh)
			if err := watcher.Close(); err != nil {
				log.Warn().Err(err).Msg("Failed to close watcher")
			}
			delete(e.watchers, vmName)
			delete(e.watcherStopCh, vmName)
		}
	}

	log.Info().Str("vm", vmName).Msg("Sync configuration updated")
	return nil
}

// ResolveSyncConflict resolves a sync conflict
func (e *Engine) ResolveSyncConflict(vmName string, path string, resolution string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Validate VM name
	if vmName == "" {
		return ErrInvalidVMName
	}

	// Check if registered
	status, exists := e.statuses[vmName]
	if !exists {
		return ErrVMNotRegistered
	}

	// Find conflict
	var foundIndex = -1
	for i, conflict := range status.Conflicts {
		if conflict.Path == path {
			foundIndex = i
			break
		}
	}

	if foundIndex == -1 {
		return errors.NotFound("conflict", path)
	}

	conflict := status.Conflicts[foundIndex]

	// Resolve conflict based on resolution
	switch resolution {
	case "use_host":
		// Sync file from host to VM
		if _, err := e.syncFilesToVM(vmName, []string{path}); err != nil {
			return errors.OperationFailed("sync file to VM", err)
		}
	case "use_vm":
		// Sync file from VM to host
		if _, err := e.syncFilesFromVM(vmName, []string{path}); err != nil {
			return errors.OperationFailed("sync file from VM", err)
		}
	case "merge":
		// Attempt to merge changes
		if err := e.mergeConflict(vmName, conflict); err != nil {
			return errors.OperationFailed("merge conflict", err)
		}
	case "keep_both":
		// Keep both versions with different names
		if err := e.keepBothVersions(vmName, conflict); err != nil {
			return errors.OperationFailed("keep both versions", err)
		}
	default:
		return errors.InvalidInput(fmt.Sprintf("invalid resolution: %s (must be 'use_host', 'use_vm', 'merge', or 'keep_both')", resolution))
	}

	// Remove conflict from list
	status.Conflicts = append(status.Conflicts[:foundIndex], status.Conflicts[foundIndex+1:]...)
	e.statuses[vmName] = status

	log.Info().Str("vm", vmName).Str("path", path).Str("resolution", resolution).Msg("Sync conflict resolved")
	return nil
}

// SemanticSearch performs a semantic search across synchronized files
func (e *Engine) SemanticSearch(vmName string, query string, maxResults int) ([]SearchResult, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Validate VM name
	if vmName == "" {
		return nil, ErrInvalidVMName
	}

	// Check if registered
	config, exists := e.configs[vmName]
	if !exists {
		return nil, ErrVMNotRegistered
	}

	// Define search paths
	searchPath := config.ProjectPath
	if searchPath == "" {
		return nil, errors.NotFound("project path for VM", vmName)
	}

	log.Info().Str("vm", vmName).Str("query", query).Msg("Executing semantic search")

	// Execute search - in a real implementation, this would use a more sophisticated
	// semantic search algorithm. For now, we're using simple grep as a placeholder.
	cmd := exec.Command("grep", "-r", "-l", "-i", query, searchPath)
	output, err := cmd.CombinedOutput()
	if err != nil && !strings.Contains(err.Error(), "exit status 1") {
		return nil, errors.OperationFailed("search", err)
	}

	// Process results
	results := []SearchResult{}
	for _, line := range strings.Split(string(output), "\n") {
		if line == "" {
			continue
		}

		// For each file that matches, get exact line matches
		contentCmd := exec.Command("grep", "-n", "-i", query, line)
		contentOutput, err := contentCmd.CombinedOutput()
		if err != nil && !strings.Contains(err.Error(), "exit status 1") {
			continue
		}

		for _, contentLine := range strings.Split(string(contentOutput), "\n") {
			if contentLine == "" {
				continue
			}

			parts := strings.SplitN(contentLine, ":", 3)
			if len(parts) < 3 {
				continue
			}

			lineNum := 0
			if _, err := fmt.Sscanf(parts[1], "%d", &lineNum); err != nil {
				log.Warn().Err(err).Msg("Failed to parse line number")
			}

			result := SearchResult{
				Path:      line,
				Line:      lineNum,
				Content:   parts[2],
				MatchType: "exact",
			}
			results = append(results, result)

			if len(results) >= maxResults {
				break
			}
		}

		if len(results) >= maxResults {
			break
		}
	}

	return results, nil
}

// ExactSearch performs an exact string search across synchronized files
func (e *Engine) ExactSearch(vmName string, query string, caseSensitive bool, maxResults int) ([]SearchResult, error) {
	// Implementation similar to SemanticSearch but using exact matching
	// Using case-sensitive or case-insensitive search based on the parameter

	// This is a simplified implementation that could be enhanced
	// with better search algorithms in a real-world scenario
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Validate VM name
	if vmName == "" {
		return nil, ErrInvalidVMName
	}

	// Check if registered
	config, exists := e.configs[vmName]
	if !exists {
		return nil, ErrVMNotRegistered
	}

	// Define search paths
	searchPath := config.ProjectPath
	if searchPath == "" {
		return nil, errors.NotFound("project path for VM", vmName)
	}

	log.Info().Str("vm", vmName).Str("query", query).Msg("Executing exact search")

	// Set up grep arguments
	grepArgs := []string{"-r", "-n"}
	if !caseSensitive {
		grepArgs = append(grepArgs, "-i")
	}
	grepArgs = append(grepArgs, query, searchPath)

	// Execute search
	cmd := exec.Command("grep", grepArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil && !strings.Contains(err.Error(), "exit status 1") {
		return nil, errors.OperationFailed("search", err)
	}

	// Process results
	results := []SearchResult{}
	for _, line := range strings.Split(string(output), "\n") {
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 3)
		if len(parts) < 3 {
			continue
		}

		lineNum := 0
		if _, err := fmt.Sscanf(parts[1], "%d", &lineNum); err != nil {
			log.Warn().Err(err).Msg("Failed to parse line number")
		}

		result := SearchResult{
			Path:      parts[0],
			Line:      lineNum,
			Content:   parts[2],
			MatchType: "exact",
		}
		results = append(results, result)

		if len(results) >= maxResults {
			break
		}
	}

	return results, nil
}

// FuzzySearch performs a fuzzy search across synchronized files
func (e *Engine) FuzzySearch(vmName string, query string, maxResults int) ([]SearchResult, error) {
	// This would implement a fuzzy search algorithm
	// For now, we'll use a basic approximation with grep

	e.mu.RLock()
	defer e.mu.RUnlock()

	// Validate VM name
	if vmName == "" {
		return nil, ErrInvalidVMName
	}

	// Check if registered
	config, exists := e.configs[vmName]
	if !exists {
		return nil, ErrVMNotRegistered
	}

	// Define search paths
	searchPath := config.ProjectPath
	if searchPath == "" {
		return nil, errors.NotFound("project path for VM", vmName)
	}

	log.Info().Str("vm", vmName).Str("query", query).Msg("Executing fuzzy search")

	// Split query into words for fuzzy searching
	words := strings.Fields(query)
	results := []SearchResult{}

	for _, word := range words {
		if len(word) < 3 {
			continue // Skip very short words
		}

		// Execute search with word
		cmd := exec.Command("grep", "-r", "-n", "-i", word, searchPath)
		output, err := cmd.CombinedOutput()
		if err != nil && !strings.Contains(err.Error(), "exit status 1") {
			continue
		}

		// Process results
		for _, line := range strings.Split(string(output), "\n") {
			if line == "" {
				continue
			}

			parts := strings.SplitN(line, ":", 3)
			if len(parts) < 3 {
				continue
			}

			lineNum := 0
			if _, err := fmt.Sscanf(parts[1], "%d", &lineNum); err != nil {
				log.Warn().Err(err).Msg("Failed to parse line number")
			}

			// Only add if it's not already in the results
			isDuplicate := false
			for _, existing := range results {
				if existing.Path == parts[0] && existing.Line == lineNum {
					isDuplicate = true
					break
				}
			}

			if !isDuplicate {
				result := SearchResult{
					Path:      parts[0],
					Line:      lineNum,
					Content:   parts[2],
					MatchType: "fuzzy",
				}
				results = append(results, result)
			}

			if len(results) >= maxResults {
				break
			}
		}

		if len(results) >= maxResults {
			break
		}
	}

	return results, nil
}

// Helper methods

// syncWithRsync synchronizes files using rsync
func (e *Engine) syncWithRsync(vmName string, sourcePath string, toVM bool) ([]string, error) {
	// Get VM config
	config, exists := e.configs[vmName]
	if !exists {
		return nil, ErrVMNotRegistered
	}

	// Get exclude patterns - these are used by the VM manager internally
	// Just logging them for reference
	if len(config.ExcludePatterns) > 0 {
		log.Debug().Str("vm", vmName).Strs("exclude_patterns", config.ExcludePatterns).Msg("Using exclude patterns for sync")
	}

	// We'll use the VM manager directly instead of creating temporary files and running commands

	// Check if VM manager is set
	if e.vmManager == nil {
		return nil, errors.OperationFailed("VM manager not set before sync operations", nil)
	}

	// Use the VM manager to perform the sync
	var syncErr error
	if toVM {
		// Sync from host to VM using the VM manager
		syncErr = e.vmManager.SyncToVM(vmName, sourcePath, "/vagrant")
	} else {
		// Sync from VM to host using the VM manager
		syncErr = e.vmManager.SyncFromVM(vmName, "/vagrant", sourcePath)
	}

	if syncErr != nil {
		return nil, errors.OperationFailed("sync operation", syncErr)
	}

	// Since we're using the VM manager, we don't have a direct way to determine which files were synced
	// In a real-world scenario, we could compare file timestamps before and after sync

	// For now, we'll return a placeholder to indicate successful sync
	syncedFiles := []string{
		fmt.Sprintf("sync_completed_%s_%s", vmName, time.Now().Format(time.RFC3339)),
	}

	return syncedFiles, nil
}

// syncWithNFS synchronizes files using NFS
func (e *Engine) syncWithNFS(vmName string, sourcePath string, toVM bool) ([]string, error) {
	// NFS is typically set up as a mount, so individual sync operations are not needed
	// Check if VM manager is set
	if e.vmManager == nil {
		return nil, errors.OperationFailed("VM manager not set before sync operations", nil)
	}

	// For NFS, we need to ensure the VM is running for the mount to be accessible
	// Use the VM manager to perform the sync
	var syncErr error
	if toVM {
		// Sync from host to VM using the VM manager
		syncErr = e.vmManager.SyncToVM(vmName, sourcePath, "/vagrant")
	} else {
		// Sync from VM to host using the VM manager
		syncErr = e.vmManager.SyncFromVM(vmName, "/vagrant", sourcePath)
	}

	if syncErr != nil {
		return nil, errors.OperationFailed("sync operation", syncErr)
	}

	// Return a placeholder to indicate successful sync
	return []string{
		fmt.Sprintf("sync_completed_%s_%s", vmName, time.Now().Format(time.RFC3339)),
	}, nil
}

// syncWithSMB synchronizes files using SMB
func (e *Engine) syncWithSMB(vmName string, sourcePath string, toVM bool) ([]string, error) {
	// SMB is typically set up as a mount, so individual sync operations are not needed
	// Check if VM manager is set
	if e.vmManager == nil {
		return nil, errors.OperationFailed("VM manager not set before sync operations", nil)
	}

	// For SMB, we need to ensure the VM is running for the mount to be accessible
	// Use the VM manager to perform the sync
	var syncErr error
	if toVM {
		// Sync from host to VM using the VM manager
		syncErr = e.vmManager.SyncToVM(vmName, sourcePath, "/vagrant")
	} else {
		// Sync from VM to host using the VM manager
		syncErr = e.vmManager.SyncFromVM(vmName, "/vagrant", sourcePath)
	}

	if syncErr != nil {
		return nil, errors.OperationFailed("sync operation", syncErr)
	}

	// Return a placeholder to indicate successful sync
	return []string{
		fmt.Sprintf("sync_completed_%s_%s", vmName, time.Now().Format(time.RFC3339)),
	}, nil
}

// syncFilesToVM synchronizes specific files to the VM
func (e *Engine) syncFilesToVM(vmName string, files []string) ([]string, error) {
	// Check if VM manager is set
	if e.vmManager == nil {
		return nil, errors.OperationFailed("VM manager not set before sync operations", nil)
	}

	// Get VM config
	config, exists := e.configs[vmName]
	if !exists {
		return nil, ErrVMNotRegistered
	}

	// For selective file sync, we need to iterate through each file and sync individually
	syncedFiles := []string{}
	for _, file := range files {
		// Get the relative path within the project
		relPath, err := filepath.Rel(config.ProjectPath, file)
		if err != nil {
			continue // Skip files outside the project
		}

		// Use the VM manager to sync this specific file
		guestPath := filepath.Join("/vagrant", relPath)
		if err := e.vmManager.SyncToVM(vmName, file, guestPath); err != nil {
			return syncedFiles, errors.OperationFailed("failed to sync file to VM", err)
		}

		syncedFiles = append(syncedFiles, file)
	}

	return syncedFiles, nil
}

// syncFilesFromVM synchronizes specific files from the VM
func (e *Engine) syncFilesFromVM(vmName string, files []string) ([]string, error) {
	// Check if VM manager is set
	if e.vmManager == nil {
		return nil, errors.OperationFailed("VM manager not set before sync operations", nil)
	}

	// Get VM config
	config, exists := e.configs[vmName]
	if !exists {
		return nil, ErrVMNotRegistered
	}

	// For selective file sync, we need to iterate through each file and sync individually
	syncedFiles := []string{}
	for _, file := range files {
		// Determine the paths for source and destination
		// Convert the path to be relative to /vagrant on the VM
		vmPath := filepath.Join("/vagrant", filepath.Base(file))
		hostPath := filepath.Join(config.ProjectPath, filepath.Base(file))

		// Use the VM manager to sync this specific file
		if err := e.vmManager.SyncFromVM(vmName, vmPath, hostPath); err != nil {
			return syncedFiles, errors.OperationFailed("failed to sync file from VM", err)
		}

		syncedFiles = append(syncedFiles, hostPath)
	}

	return syncedFiles, nil
}

// startWatcher starts a file watcher for a VM
func (e *Engine) startWatcher(vmName string) error {
	// Get VM config
	config, exists := e.configs[vmName]
	if !exists {
		return ErrVMNotRegistered
	}

	// Create watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %w", err)
	}

	// Add project directory to watcher
	err = filepath.Walk(config.ProjectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			// Skip excluded directories
			for _, pattern := range config.ExcludePatterns {
				if matched, _ := filepath.Match(pattern, filepath.Base(path)); matched {
					return filepath.SkipDir
				}
			}
			return watcher.Add(path)
		}
		return nil
	})
	if err != nil {
		if cerr := watcher.Close(); cerr != nil {
			log.Warn().Err(cerr).Msg("Failed to close watcher after error")
		}
		return fmt.Errorf("failed to add directories to watcher: %w", err)
	}

	// Create stop channel
	stopCh := make(chan struct{})
	e.watchers[vmName] = watcher
	e.watcherStopCh[vmName] = stopCh

	// Start watcher goroutine
	go func() {
		defer func() {
			if err := watcher.Close(); err != nil {
				log.Warn().Err(err).Msg("Failed to close watcher in goroutine")
			}
		}()

		// Create a timer for batching changes
		var timer *time.Timer
		var pendingChanges = make(map[string]bool)

		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&(fsnotify.Write|fsnotify.Create) != 0 {
					// Check exclude patterns
					isExcluded := false
					for _, pattern := range config.ExcludePatterns {
						if matched, _ := filepath.Match(pattern, filepath.Base(event.Name)); matched {
							isExcluded = true
							break
						}
					}
					if !isExcluded {
						pendingChanges[event.Name] = true
						if timer == nil {
							timer = time.AfterFunc(config.WatchInterval, func() {
								e.mu.Lock()
								defer e.mu.Unlock()

								// Sync changed files
								files := make([]string, 0, len(pendingChanges))
								for file := range pendingChanges {
									files = append(files, file)
								}

								if len(files) > 0 {
									log.Info().Str("vm", vmName).Int("count", len(files)).Msg("File changes detected, syncing to VM")
									if _, err := e.syncFilesToVM(vmName, files); err != nil {
										log.Error().Err(err).Str("vm", vmName).Msg("Failed to sync changes to VM")
									}
								}

								// Reset pending changes
								pendingChanges = make(map[string]bool)
								timer = nil
							})
						}
					}
				}

				// Add new directories to watch
				if event.Op&fsnotify.Create != 0 {
					info, err := os.Stat(event.Name)
					if err == nil && info.IsDir() {
						if err := watcher.Add(event.Name); err != nil {
							log.Warn().Err(err).Msg("Failed to add new directory to watcher")
						}
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Error().Err(err).Str("vm", vmName).Msg("File watcher error")
			case <-stopCh:
				if timer != nil {
					timer.Stop()
				}
				return
			}
		}
	}()

	log.Info().Str("vm", vmName).Str("path", config.ProjectPath).Msg("File watcher started")
	return nil
}

// mergeConflict attempts to merge changes from both versions of a file
func (e *Engine) mergeConflict(vmName string, conflict SyncConflict) error {
	config, exists := e.configs[vmName]
	if !exists {
		return ErrVMNotRegistered
	}

	// Create temporary files for diff3 merge
	hostFile := fmt.Sprintf("%s.host", conflict.Path)
	vmFile := fmt.Sprintf("%s.vm", conflict.Path)
	baseFile := fmt.Sprintf("%s.base", conflict.Path)

	// Get file content from VM if not already in the conflict
	vmContent := conflict.VMContent
	if vmContent == "" {
		// Command to get content from VM
		cmd := exec.Command("vagrant", "ssh", vmName, "-c", fmt.Sprintf("cat %s", conflict.Path))
		cmd.Dir = config.ProjectPath
		output, err := cmd.Output()
		if err != nil {
			return fmt.Errorf("failed to get VM file content: %w", err)
		}
		vmContent = string(output)
	}

	// Get host content if not already in the conflict
	hostContent := conflict.HostContent
	if hostContent == "" {
		content, err := os.ReadFile(conflict.Path)
		if err != nil {
			return fmt.Errorf("failed to read host file: %w", err)
		}
		hostContent = string(content)
	}

	// Try to find a common base version (could be enhanced with git or other VCS)
	// For now, we'll create a simplified base file
	baseContent := e.createBaseContent(hostContent, vmContent)

	// Write files for merge tool
	if err := os.WriteFile(hostFile, []byte(hostContent), 0644); err != nil {
		return err
	}
	if err := os.WriteFile(vmFile, []byte(vmContent), 0644); err != nil {
		return err
	}
	if err := os.WriteFile(baseFile, []byte(baseContent), 0644); err != nil {
		return err
	}

	// Perform merge using diff3
	cmd := exec.Command("diff3", "-m", hostFile, baseFile, vmFile)
	output, err := cmd.CombinedOutput()

	// Clean up temp files
	if err := os.Remove(hostFile); err != nil {
		log.Warn().Err(err).Msg("Failed to remove hostFile")
	}
	if err := os.Remove(vmFile); err != nil {
		log.Warn().Err(err).Msg("Failed to remove vmFile")
	}
	if err := os.Remove(baseFile); err != nil {
		log.Warn().Err(err).Msg("Failed to remove baseFile")
	}

	if err != nil {
		// If automatic merge failed, return conflict markers
		if err := os.WriteFile(conflict.Path, output, 0644); err != nil {
			return err
		}

		// Also sync the conflict-marked file to the VM
		if _, err := e.syncFilesToVM(vmName, []string{conflict.Path}); err != nil {
			return err
		}

		return fmt.Errorf("automatic merge had conflicts, file saved with conflict markers")
	}

	// Write merged content and sync to VM
	if err := os.WriteFile(conflict.Path, output, 0644); err != nil {
		return err
	}

	if _, err := e.syncFilesToVM(vmName, []string{conflict.Path}); err != nil {
		return err
	}

	return nil
}

// keepBothVersions keeps both versions of a conflicted file with different names
func (e *Engine) keepBothVersions(vmName string, conflict SyncConflict) error {
	config, exists := e.configs[vmName]
	if !exists {
		return ErrVMNotRegistered
	}

	// Generate filenames
	// Using the conflict path directly in the code below
	vmFile := fmt.Sprintf("%s.vm", conflict.Path)

	// Get file content from VM if not already in the conflict
	vmContent := conflict.VMContent
	if vmContent == "" {
		// Command to get content from VM
		cmd := exec.Command("vagrant", "ssh", vmName, "-c", fmt.Sprintf("cat %s", conflict.Path))
		cmd.Dir = config.ProjectPath
		output, err := cmd.Output()
		if err != nil {
			return fmt.Errorf("failed to get VM file content: %w", err)
		}
		vmContent = string(output)
	}

	// Write VM version to host
	if err := os.WriteFile(vmFile, []byte(vmContent), 0644); err != nil {
		return err
	}

	// Sync the VM version back to VM with the .vm extension
	if _, err := e.syncFilesToVM(vmName, []string{vmFile}); err != nil {
		return err
	}

	return nil
}

// createBaseContent creates a simplified base version for merge operations
func (e *Engine) createBaseContent(hostContent, vmContent string) string {
	// This is a very simplified approach - in a real implementation,
	// you might use a more sophisticated algorithm or store previous versions

	hostLines := strings.Split(hostContent, "\n")
	vmLines := strings.Split(vmContent, "\n")

	commonLines := []string{}

	// Find common beginning
	minLen := len(hostLines)
	if len(vmLines) < minLen {
		minLen = len(vmLines)
	}

	for i := 0; i < minLen; i++ {
		if hostLines[i] == vmLines[i] {
			commonLines = append(commonLines, hostLines[i])
		} else {
			break
		}
	}

	// Find common ending
	hostEndIndex := len(hostLines) - 1
	vmEndIndex := len(vmLines) - 1

	for hostEndIndex >= 0 && vmEndIndex >= 0 && hostLines[hostEndIndex] == vmLines[vmEndIndex] {
		hostEndIndex--
		vmEndIndex--
	}

	// Add common ending in reverse order
	endingLines := []string{}
	for i := hostEndIndex + 1; i < len(hostLines); i++ {
		endingLines = append(endingLines, hostLines[i])
	}

	// Combine common beginning and ending
	return strings.Join(commonLines, "\n") + "\n" + strings.Join(endingLines, "\n")
}

// IsRunning checks if the sync engine is currently running
func (e *Engine) IsRunning() bool {
	return e.running
}

// Start starts the sync engine
func (e *Engine) Start() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.running {
		return fmt.Errorf("sync engine already running")
	}

	e.running = true
	return nil
}

// Stop stops the sync engine
func (e *Engine) Stop() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.running {
		return fmt.Errorf("sync engine not running")
	}

	e.running = false
	return nil
}
