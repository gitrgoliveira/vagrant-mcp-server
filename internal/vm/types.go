package vm

import "errors"

// State represents the possible states of a VM
type State string

const (
	// NotCreated represents a VM that has not been created yet
	NotCreated State = "not_created"
	// Running represents a VM that is currently running
	Running State = "running"
	// Stopped represents a VM that is currently stopped
	Stopped State = "stopped"
	// Suspended represents a VM that is suspended
	Suspended State = "suspended"
	// Error represents a VM in an error state
	Error State = "error"
)

// Common errors
var (
	ErrVMExists    = errors.New("vm already exists")
	ErrVMNotFound  = errors.New("vm not found")
	ErrInvalidName = errors.New("invalid vm name")
)

// VMConfig represents the configuration for a VM
type VMConfig struct {
	Name                string   `json:"name"`
	Box                 string   `json:"box"`
	CPU                 int      `json:"cpu"`
	Memory              int      `json:"memory"`
	SyncType            string   `json:"sync_type"`
	ProjectPath         string   `json:"project_path"`
	Ports               []Port   `json:"ports,omitempty"`
	Environment         []string `json:"environment,omitempty"`
	Provisioners        []string `json:"provisioners,omitempty"`
	HostPath            string   `json:"host_path,omitempty"`
	GuestPath           string   `json:"guest_path,omitempty"`
	SyncExcludePatterns []string `json:"sync_exclude_patterns,omitempty"`
}

// Port represents a port mapping configuration
type Port struct {
	Guest int `json:"guest"`
	Host  int `json:"host"`
}
