// Package vm provides functionality for managing Vagrant virtual machines
package vm

import (
	"github.com/vagrant-mcp/server/internal/core"
)

// Ensure Manager implements the core.VMManager interface
var _ core.VMManager = (*Manager)(nil)
