// Copyright Ricardo Oliveira 2025.
// SPDX-License-Identifier: MPL-2.0

package core

// Port represents a port mapping between guest and host
type Port struct {
	Guest int `json:"guest"`
	Host  int `json:"host"`
}

// VMConfig represents the configuration for a virtual machine
type VMConfig struct {
	Name                string   `json:"name"`
	Box                 string   `json:"box"`
	CPU                 int      `json:"cpu"`
	Memory              int      `json:"memory"`
	ProjectPath         string   `json:"project_path"`
	SyncType            string   `json:"sync_type"`
	HostPath            string   `json:"host_path,omitempty"`
	GuestPath           string   `json:"guest_path,omitempty"`
	SyncExcludePatterns []string `json:"sync_exclude_patterns,omitempty"`
	Ports               []Port   `json:"ports,omitempty"`
	Environment         []string `json:"environment,omitempty"`
	Provisioners        []string `json:"provisioners,omitempty"`
}

// UploadOptions contains options for uploading files to a VM
type UploadOptions struct {
	Compress        bool   `json:"compress"`
	CompressionType string `json:"compression_type,omitempty"`
}
