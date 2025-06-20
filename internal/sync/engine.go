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

// SyncStatus represents the status of a synchronization operation
type SyncStatus struct {
	LastSyncTime      time.Time      `json:"last_sync_time"`
	InProgress        bool           `json:"in_progress"`
	Conflicts         []SyncConflict `json:"conflicts"`
	SynchronizedFiles int            `json:"synchronized_files"`
	Error             string         `json:"error,omitempty"`
}

// SyncConflict represents a file conflict during synchronization
type SyncConflict struct {
	Path        string    `json:"path"`
	HostModTime time.Time `json:"host_mod_time"`
	VMModTime   time.Time `json:"vm_mod_time"`
}

// Engine handles file synchronization between host and VM
type Engine struct {
	configs       map[string]SyncConfig
	statuses      map[string]SyncStatus
	watchers      map[string]*fsnotify.Watcher
	watcherStopCh map[string]chan struct{}
	mu            sync.RWMutex
}

// NewEngine creates a new synchronization engine
func NewEngine() (*Engine, error) {
	return &Engine{
		configs:       make(map[string]SyncConfig),
		statuses:      make(map[string]SyncStatus),
		watchers:      make(map[string]*fsnotify.Watcher),
		watcherStopCh: make(map[string]chan struct{}),
	}, nil
}

// ConfigureSync sets up synchronization for a VM
func (e *Engine) ConfigureSync(config SyncConfig) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.configs[config.VMName] = config
	e.statuses[config.VMName] = SyncStatus{
		LastSyncTime: time.Now(),
		InProgress:   false,
		Conflicts:    []SyncConflict{},
	}

	// Stop existing watcher if any
	if stopCh, exists := e.watcherStopCh[config.VMName]; exists {
		close(stopCh)
		delete(e.watcherStopCh, config.VMName)
	}

	if watcher, exists := e.watchers[config.VMName]; exists {
		if err := watcher.Close(); err != nil {
			log.Error().Err(err).Msgf("Failed to close watcher for VM %s", config.VMName)
		}
		delete(e.watchers, config.VMName)
	}

	// Setup file watcher if enabled
	if config.WatchEnabled {
		if err := e.setupWatcher(config); err != nil {
			return err
		}
	}

	return nil
}

// SyncToVM synchronizes files from host to VM
func (e *Engine) SyncToVM(vmName string, path string) error {
	e.mu.Lock()
	config, exists := e.configs[vmName]
	if !exists {
		e.mu.Unlock()
		return fmt.Errorf("no sync configuration found for VM: %s", vmName)
	}

	status := e.statuses[vmName]
	status.InProgress = true
	e.statuses[vmName] = status
	e.mu.Unlock()

	defer func() {
		e.mu.Lock()
		status := e.statuses[vmName]
		status.InProgress = false
		status.LastSyncTime = time.Now()
		e.statuses[vmName] = status
		e.mu.Unlock()
	}()

	// Determine source and destination paths
	sourcePath := config.ProjectPath
	if path != "" {
		sourcePath = filepath.Join(config.ProjectPath, path)
	}

	destPath := "/vagrant"
	if path != "" {
		destPath = filepath.Join("/vagrant", path)
	}

	// Use rsync for file synchronization
	if config.Method == SyncMethodRsync {
		return e.syncWithRsync(vmName, sourcePath, destPath, SyncToVM)
	}

	// For other sync methods, rely on Vagrant's built-in sync
	return e.syncWithVagrant(vmName)
}

// SyncFromVM synchronizes files from VM to host
func (e *Engine) SyncFromVM(vmName string, path string) error {
	e.mu.Lock()
	config, exists := e.configs[vmName]
	if !exists {
		e.mu.Unlock()
		return fmt.Errorf("no sync configuration found for VM: %s", vmName)
	}

	status := e.statuses[vmName]
	status.InProgress = true
	e.statuses[vmName] = status
	e.mu.Unlock()

	defer func() {
		e.mu.Lock()
		status := e.statuses[vmName]
		status.InProgress = false
		status.LastSyncTime = time.Now()
		e.statuses[vmName] = status
		e.mu.Unlock()
	}()

	// Determine source and destination paths
	sourcePath := "/vagrant"
	if path != "" {
		sourcePath = filepath.Join("/vagrant", path)
	}

	destPath := config.ProjectPath
	if path != "" {
		destPath = filepath.Join(config.ProjectPath, path)
	}

	// Use rsync for file synchronization
	if config.Method == SyncMethodRsync {
		return e.syncWithRsync(vmName, sourcePath, destPath, SyncFromVM)
	}

	// For other sync methods, rely on Vagrant's built-in sync
	return e.syncWithVagrant(vmName)
}

// GetSyncStatus returns the current synchronization status
func (e *Engine) GetSyncStatus(vmName string) (SyncStatus, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	status, exists := e.statuses[vmName]
	if !exists {
		return SyncStatus{}, fmt.Errorf("no sync status found for VM: %s", vmName)
	}

	return status, nil
}

// ResolveConflicts handles file conflicts during synchronization
func (e *Engine) ResolveConflicts(vmName string, strategy string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	status, exists := e.statuses[vmName]
	if !exists {
		return fmt.Errorf("no sync status found for VM: %s", vmName)
	}

	if len(status.Conflicts) == 0 {
		return nil // No conflicts to resolve
	}

	switch strategy {
	case "host":
		// Resolve conflicts by preferring host files
		for _, conflict := range status.Conflicts {
			if err := e.SyncToVM(vmName, conflict.Path); err != nil {
				return err
			}
		}
	case "guest":
		// Resolve conflicts by preferring VM files
		for _, conflict := range status.Conflicts {
			if err := e.SyncFromVM(vmName, conflict.Path); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("invalid conflict resolution strategy: %s", strategy)
	}

	// Clear resolved conflicts
	status.Conflicts = []SyncConflict{}
	e.statuses[vmName] = status

	return nil
}

// Close releases resources used by the sync engine
func (e *Engine) Close() {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Stop all file watchers
	for vmName, stopCh := range e.watcherStopCh {
		close(stopCh)
		delete(e.watcherStopCh, vmName)
	}

	for vmName, watcher := range e.watchers {
		if err := watcher.Close(); err != nil {
			log.Error().Err(err).Msgf("Failed to close watcher for VM %s", vmName)
		}
		delete(e.watchers, vmName)
	}
}

// setupWatcher configures a file watcher for automatic synchronization
func (e *Engine) setupWatcher(config SyncConfig) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %w", err)
	}

	stopCh := make(chan struct{})
	e.watchers[config.VMName] = watcher
	e.watcherStopCh[config.VMName] = stopCh

	// Add the project directory to the watcher
	if err := filepath.Walk(config.ProjectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip excluded patterns
		for _, pattern := range config.ExcludePatterns {
			if matched, _ := filepath.Match(pattern, filepath.Base(path)); matched {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		if info.IsDir() {
			return watcher.Add(path)
		}
		return nil
	}); err != nil {
		if err := watcher.Close(); err != nil {
			log.Error().Err(err).Msg("Failed to close watcher after walk error")
		}
		return fmt.Errorf("failed to walk directory tree: %w", err)
	}

	// Start watching for changes
	go func() {
		debounceTimer := time.NewTimer(config.WatchInterval)
		defer debounceTimer.Stop()

		changedFiles := make(map[string]bool)

		for {
			select {
			case <-stopCh:
				return

			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				// Skip excluded patterns
				skipFile := false
				for _, pattern := range config.ExcludePatterns {
					match, _ := filepath.Match(pattern, filepath.Base(event.Name))
					if match {
						skipFile = true
						break
					}
				}

				if skipFile {
					continue
				}

				if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove) != 0 {
					changedFiles[event.Name] = true
					debounceTimer.Reset(config.WatchInterval)
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Error().Err(err).Str("vm", config.VMName).Msg("File watcher error")

			case <-debounceTimer.C:
				if len(changedFiles) > 0 {
					log.Info().Str("vm", config.VMName).Int("files", len(changedFiles)).Msg("Detected file changes, syncing")

					// Perform sync operation
					if config.Direction == SyncToVM || config.Direction == SyncBidirectional {
						if err := e.SyncToVM(config.VMName, ""); err != nil {
							log.Error().Err(err).Msg("File watcher sync to VM failed")
						}
					}

					changedFiles = make(map[string]bool)
				}
			}
		}
	}()

	log.Info().Str("vm", config.VMName).Str("path", config.ProjectPath).Msg("File watcher started")
	return nil
}

// syncWithRsync performs file synchronization using rsync
func (e *Engine) syncWithRsync(vmName string, sourcePath, destPath string, direction SyncDirection) error {
	// Build rsync command
	var cmd *exec.Cmd

	if direction == SyncToVM {
		// Sync from host to VM
		sshConfig, err := e.getSSHConfig(vmName)
		if err != nil {
			return err
		}

		rsyncArgs := []string{
			"-avz",
			"--delete",
			"-e", fmt.Sprintf("ssh -p %s -i %s -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null",
				sshConfig["Port"], sshConfig["IdentityFile"]),
			sourcePath + "/",
			fmt.Sprintf("%s@%s:%s", sshConfig["User"], sshConfig["HostName"], destPath),
		}

		// Add exclude patterns
		config := e.configs[vmName]
		for _, pattern := range config.ExcludePatterns {
			rsyncArgs = append(rsyncArgs, "--exclude", pattern)
		}

		cmd = exec.Command("rsync", rsyncArgs...)

	} else {
		// Sync from VM to host
		sshConfig, err := e.getSSHConfig(vmName)
		if err != nil {
			return err
		}

		rsyncArgs := []string{
			"-avz",
			"--delete",
			"-e", fmt.Sprintf("ssh -p %s -i %s -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null",
				sshConfig["Port"], sshConfig["IdentityFile"]),
			fmt.Sprintf("%s@%s:%s/", sshConfig["User"], sshConfig["HostName"], sourcePath),
			destPath,
		}

		// Add exclude patterns
		config := e.configs[vmName]
		for _, pattern := range config.ExcludePatterns {
			rsyncArgs = append(rsyncArgs, "--exclude", pattern)
		}

		cmd = exec.Command("rsync", rsyncArgs...)
	}

	// Run rsync
	output, err := cmd.CombinedOutput()
	if err != nil {
		e.mu.Lock()
		status := e.statuses[vmName]
		status.Error = fmt.Sprintf("sync error: %v - %s", err, string(output))
		e.statuses[vmName] = status
		e.mu.Unlock()

		return fmt.Errorf("sync failed: %w\nOutput: %s", err, output)
	}

	// Update sync status with file count
	fileCount := strings.Count(string(output), "\n")
	e.mu.Lock()
	status := e.statuses[vmName]
	status.SynchronizedFiles = fileCount
	status.Error = ""
	e.statuses[vmName] = status
	e.mu.Unlock()

	log.Info().
		Str("vm", vmName).
		Int("files", fileCount).
		Str("direction", directionToString(direction)).
		Msg("Sync completed")

	return nil
}

// syncWithVagrant uses Vagrant's built-in sync mechanisms
func (e *Engine) syncWithVagrant(vmName string) error {
	vmDir, err := e.getVMDir(vmName)
	if err != nil {
		return err
	}

	cmd := exec.Command("vagrant", "rsync")
	cmd.Dir = vmDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		e.mu.Lock()
		status := e.statuses[vmName]
		status.Error = fmt.Sprintf("vagrant rsync error: %v - %s", err, string(output))
		e.statuses[vmName] = status
		e.mu.Unlock()

		return fmt.Errorf("vagrant sync failed: %w\nOutput: %s", err, output)
	}

	log.Info().Str("vm", vmName).Msg("Vagrant sync completed")
	return nil
}

// getSSHConfig retrieves the SSH configuration for a VM
func (e *Engine) getSSHConfig(vmName string) (map[string]string, error) {
	vmDir, err := e.getVMDir(vmName)
	if err != nil {
		return nil, err
	}

	cmd := exec.Command("vagrant", "ssh-config")
	cmd.Dir = vmDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to get SSH config: %w", err)
	}

	return parseSSHConfig(string(output))
}

// getVMDir returns the directory for a VM
func (e *Engine) getVMDir(vmName string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	vmDir := filepath.Join(homeDir, ".vagrant-mcp", "vms", vmName)
	if _, err := os.Stat(vmDir); os.IsNotExist(err) {
		return "", fmt.Errorf("VM directory does not exist: %s", vmDir)
	}

	return vmDir, nil
}

// directionToString converts a sync direction to a string representation
func directionToString(direction SyncDirection) string {
	switch direction {
	case SyncToVM:
		return "host->vm"
	case SyncFromVM:
		return "vm->host"
	case SyncBidirectional:
		return "bidirectional"
	default:
		return "unknown"
	}
}

// parseSSHConfig parses the output of 'vagrant ssh-config'
func parseSSHConfig(output string) (map[string]string, error) {
	config := make(map[string]string)
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 2 {
			key := parts[0]
			value := strings.TrimSpace(parts[1])
			config[key] = value
		}
	}

	// Validate required fields
	requiredFields := []string{"HostName", "User", "Port", "IdentityFile"}
	for _, field := range requiredFields {
		if _, exists := config[field]; !exists {
			return nil, fmt.Errorf("missing required SSH config field: %s", field)
		}
	}

	return config, nil
}

// Sync performs synchronization according to the provided configuration
func (e *Engine) Sync(config SyncConfig) (*SyncStatus, error) {
	// Configure sync with the provided config
	if err := e.ConfigureSync(config); err != nil {
		return nil, fmt.Errorf("failed to configure sync: %w", err)
	}

	// Perform sync based on direction
	var err error
	switch config.Direction {
	case SyncToVM:
		err = e.SyncToVM(config.VMName, "")
	case SyncFromVM:
		err = e.SyncFromVM(config.VMName, "")
	case SyncBidirectional:
		// For bidirectional sync, first sync to VM then from VM
		if err = e.SyncToVM(config.VMName, ""); err != nil {
			return nil, fmt.Errorf("failed to sync to VM: %w", err)
		}
		err = e.SyncFromVM(config.VMName, "")
	default:
		return nil, fmt.Errorf("invalid sync direction: %v", config.Direction)
	}

	if err != nil {
		return nil, err
	}

	// Return current sync status
	status, err := e.GetSyncStatus(config.VMName)
	if err != nil {
		return nil, fmt.Errorf("failed to get sync status: %w", err)
	}

	return &status, nil
}
