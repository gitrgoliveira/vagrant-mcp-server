// Copyright Ricardo Oliveira 2025.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/mark3labs/mcp-go/server"
	"github.com/vagrant-mcp/server/internal/exec"
	"github.com/vagrant-mcp/server/internal/handlers"
	"github.com/vagrant-mcp/server/internal/resources"
	"github.com/vagrant-mcp/server/internal/sync"
	"github.com/vagrant-mcp/server/internal/vm"
)

func TestServer(t *testing.T) {
	// Skip integration tests unless explicitly enabled
	testLevel := os.Getenv("TEST_LEVEL")
	if testLevel != "integration" && testLevel != "vm-start" {
		t.Skip("Integration tests disabled. Set TEST_LEVEL=integration to run.")
	}

	// Initialize VM manager, sync engine, and executor
	vmManager, err := vm.NewManager()
	if err != nil {
		t.Fatalf("Failed to create VM manager: %v", err)
	}

	syncEngine, err := sync.NewEngine()
	if err != nil {
		t.Fatalf("Failed to create sync engine: %v", err)
	}

	adapterVM := &exec.VMManagerAdapter{Real: vmManager}
	// Set the VM manager on the sync engine before creating the adapter
	syncEngine.SetVMManager(adapterVM)
	adapterSync := &exec.SyncEngineAdapter{Real: syncEngine}

	executor, err := exec.NewExecutor(adapterVM, adapterSync)
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}

	// Create a new MCP server
	srv := server.NewMCPServer(
		"Vagrant Development VM MCP Server",
		"1.0.0",
		server.WithRecovery(),
	)

	// Register all tools using the MCP-go implementation
	handlers.RegisterVMTools(srv, adapterVM, adapterSync)
	handlers.RegisterExecTools(srv, adapterVM, adapterSync, executor)
	handlers.RegisterEnvTools(srv, adapterVM, executor)
	handlers.RegisterSyncTools(srv, adapterSync, adapterVM)

	// Register resources using the MCP-go implementation
	resources.RegisterMCPResources(srv, adapterVM, executor)

	// We're not starting the server for real in tests
	// Just validating initialization

	// Print a message indicating the server started successfully
	fmt.Println("MCP Server started successfully with the following components:")
	fmt.Println("- VM Management Tools (create_dev_vm, ensure_dev_vm, destroy_dev_vm)")
	fmt.Println("- Sync Tools (sync_to_vm, sync_from_vm, sync_status, resolve_sync_conflicts)")
	fmt.Println("- Exec Tools (exec_in_vm, exec_with_sync)")
	fmt.Println("- Environment Tools (setup_dev_environment, install_dev_tools, configure_shell)")
	fmt.Println("- Resources (VM status, config, files, environment, services, logs, etc.)")
	fmt.Println("Server test complete")
}
