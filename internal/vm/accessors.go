// Copyright Ricardo Oliveira 2025.
// SPDX-License-Identifier: MPL-2.0

package vm

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/vagrant-mcp/server/internal/errors"
)

// GetBaseDir returns the base directory for VM storage
func (m *Manager) GetBaseDir() string {
	return m.baseDir
}

// ListVMs returns a list of all configured VMs
func (m *Manager) ListVMs(ctx context.Context) ([]string, error) {
	// Check if base directory exists
	if _, err := os.Stat(m.baseDir); os.IsNotExist(err) {
		return nil, nil // No VMs exist yet
	}

	// Get all subdirectories in the base directory
	files, err := os.ReadDir(m.baseDir)
	if err != nil {
		return nil, errors.OperationFailed("list VMs", err)
	}

	var vmNames []string
	for _, file := range files {
		if file.IsDir() {
			// Check if this directory has a .vagrant-name file which indicates it's a VM directory
			vmNameFile := filepath.Join(m.baseDir, file.Name(), ".vagrant-name")
			if _, err := os.Stat(vmNameFile); err == nil {
				// Read the VM name from the file
				vmNameBytes, err := os.ReadFile(vmNameFile)
				if err == nil {
					vmName := strings.TrimSpace(string(vmNameBytes))
					vmNames = append(vmNames, vmName)
				}
			}
		}
	}

	return vmNames, nil
}
