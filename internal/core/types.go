// Package core provides the core types used throughout the Vagrant MCP Server
package core

import "time"

// VMState represents the state of a VM
type VMState string

const (
	// NotCreated indicates the VM does not exist
	NotCreated VMState = "not_created"
	// Running indicates the VM is running
	Running VMState = "running"
	// Stopped indicates the VM exists but is powered off
	Stopped VMState = "poweroff"
	// Suspended indicates the VM is suspended/saved
	Suspended VMState = "saved"
	// Error indicates an error state
	Error VMState = "error"
	// Unknown indicates the VM state could not be determined
	Unknown VMState = "unknown"
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

// ExecutionContext contains context information for command execution
type ExecutionContext struct {
	VMName      string
	WorkingDir  string
	Environment []string
	SyncBefore  bool
	SyncAfter   bool
}

// CommandResult contains the result of a command execution
type CommandResult struct {
	ExitCode  int
	Output    string
	Error     string
	Duration  time.Duration
	Command   string
	VMName    string
	StartTime time.Time
	EndTime   time.Time
}

// OutputCallback is a function that receives output from command execution
type OutputCallback func(data []byte, isStderr bool)
