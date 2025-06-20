package main

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/vagrant-mcp/server/internal/server"
)

func TestServer(t *testing.T) {
	// Skip integration tests unless explicitly enabled
	if os.Getenv("INTEGRATION_TESTS") != "1" {
		t.Skip("Integration tests disabled. Set INTEGRATION_TESTS=1 to run.")
	}

	// Create a new server instance
	srv, err := server.NewServer()
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Create a context with timeout for server shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the server
	if err := srv.Start(ctx); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer func() {
		if err := srv.Stop(); err != nil {
			t.Errorf("Failed to stop server: %v", err)
		}
	}()

	// Wait for server to fully initialize
	time.Sleep(1 * time.Second)

	// Print a message indicating the server started successfully
	fmt.Println("MCP Server started successfully with the following components:")
	fmt.Println("- VM Management Tools (create_dev_vm, ensure_dev_vm, destroy_dev_vm)")
	fmt.Println("- Sync Tools (sync_to_vm, sync_from_vm, sync_status, resolve_sync_conflicts)")
	fmt.Println("- Exec Tools (exec_in_vm, exec_with_sync)")
	fmt.Println("- Environment Tools (setup_dev_environment, install_dev_tools, configure_shell)")
	fmt.Println("- Resources (VM status, config, files, environment, services, logs, etc.)")
	fmt.Println("Server test complete")
}
