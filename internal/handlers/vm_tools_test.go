package handlers

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/vagrant-mcp/server/internal/core"
	testfixture "github.com/vagrant-mcp/server/internal/testing"
	"github.com/vagrant-mcp/server/pkg/mcp"
)

// TestHandleCreateDevVM_Integration is an integration test that creates a real VM
func TestHandleCreateDevVM_Integration(t *testing.T) {
	// Skip by default - can be enabled with TEST_LEVEL=integration
	testLevel := os.Getenv("TEST_LEVEL")
	if testLevel != "integration" && testLevel != "vm-start" {
		t.Skip("Skipping integration test. Set TEST_LEVEL=integration to run")
		return
	}

	// Setup fixture without VM
	fixture, err := testfixture.NewUnifiedFixture(t, testfixture.FixtureOptions{
		PackageName:   "vm-tools",
		SetupVM:       false,
		CreateProject: true,
	})
	if err != nil {
		t.Fatalf("Failed to set up test fixture: %v", err)
	}
	defer fixture.Cleanup()

	// Create handler function using the same logic as RegisterVMTools
	handlerFunc := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		type CreateVMArgs struct {
			Name            string                   `json:"name"`
			ProjectPath     string                   `json:"project_path"`
			CPU             float64                  `json:"cpu"`
			Memory          float64                  `json:"memory"`
			Box             string                   `json:"box"`
			SyncType        string                   `json:"sync_type"`
			Ports           []map[string]interface{} `json:"ports"`
			ExcludePatterns []string                 `json:"exclude_patterns"`
		}
		var args CreateVMArgs
		if err := request.BindArguments(&args); err != nil {
			return mcp.NewToolResultError("Invalid arguments: " + err.Error()), nil
		}
		if args.Name == "" || args.ProjectPath == "" {
			return mcp.NewToolResultError("Missing required parameter: name or project_path"), nil
		}

		var ports []core.Port
		for _, portMap := range args.Ports {
			var port core.Port
			if guest, ok := portMap["guest"].(float64); ok {
				port.Guest = int(guest)
			}
			if host, ok := portMap["host"].(float64); ok {
				port.Host = int(host)
			}
			ports = append(ports, port)
		}
		if len(ports) == 0 {
			ports = []core.Port{
				{Guest: 3000, Host: 3000},
				{Guest: 8000, Host: 8000},
				{Guest: 5432, Host: 5432},
				{Guest: 3306, Host: 3306},
				{Guest: 6379, Host: 6379},
			}
		}
		excludePatterns := args.ExcludePatterns
		if len(excludePatterns) == 0 {
			excludePatterns = []string{"node_modules", ".git", "*.log", "dist", "build", "__pycache__", "*.pyc", "venv", ".venv", "*.o", "*.out"}
		}
		config := core.VMConfig{
			Box:                 args.Box,
			CPU:                 int(args.CPU),
			Memory:              int(args.Memory),
			SyncType:            args.SyncType,
			Ports:               ports,
			SyncExcludePatterns: excludePatterns,
		}
		if err := fixture.VMManager.CreateVM(ctx, args.Name, args.ProjectPath, config); err != nil {
			return mcp.NewToolResultErrorf("Failed to create VM: %v", err), nil
		}
		response := map[string]interface{}{
			"name":         args.Name,
			"project_path": args.ProjectPath,
			"config":       config,
			"status":       "created",
			"timestamp":    time.Now().Format(time.RFC3339),
		}
		jsonResponse, err := json.Marshal(response)
		if err != nil {
			return mcp.NewToolResultError("Failed to marshal response"), nil
		}
		return mcp.NewToolResultText(string(jsonResponse)), nil
	}

	// Create a unique VM name for this test
	vmName := "test-vm-" + time.Now().Format("20060102150405")

	// Create MCP request with VM parameters
	request := mcp.CallToolRequest{
		Params: mcpgo.CallToolParams{
			Name: "create_dev_vm",
			Arguments: map[string]interface{}{
				"name":         vmName,
				"project_path": fixture.ProjectPath,
				"box":          "generic/alpine314", // Small box for faster tests
				"cpu":          float64(1),
				"memory":       float64(512),
			},
		},
	}

	// Call the handler function
	t.Log("Creating VM:", vmName)
	resp, err := handlerFunc(context.Background(), request)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Validate response
	if resp == nil || resp.IsError {
		msg := ""
		if resp != nil {
			msg = extractTextContent(resp.Content)
		}
		t.Fatalf("Expected non-error result, got error: %s", msg)
	}

	// Check VM was actually created by verifying the VM directory and Vagrantfile exist
	// First, let's get the actual base directory that the VM manager is using
	homeDir, _ := os.UserHomeDir()
	baseDir := os.Getenv("VM_BASE_DIR")
	if baseDir == "" {
		baseDir = filepath.Join(homeDir, ".vagrant-mcp", "vms")
	}
	vmDir := filepath.Join(baseDir, vmName)
	t.Logf("Checking for VM directory at: %s", vmDir)

	if _, err := os.Stat(vmDir); os.IsNotExist(err) {
		t.Fatalf("VM directory was not created: %s", vmDir)
	}

	vagrantfilePath := filepath.Join(vmDir, "Vagrantfile")
	if _, err := os.Stat(vagrantfilePath); os.IsNotExist(err) {
		t.Fatalf("Vagrantfile was not created: %s", vagrantfilePath)
	}

	// Check VM state - should be not_created since we only created the Vagrantfile but didn't start the VM
	state, err := fixture.VMManager.GetVMState(fixture.Context(), vmName)
	if err != nil {
		t.Fatalf("Failed to get VM state: %v", err)
	}

	t.Logf("VM created with state: %s", state)

	// VM should be in not_created state since we only created the Vagrantfile but didn't start it
	if state == core.NotCreated {
		t.Logf("Successfully created VM configuration: %s", vmName)
	} else {
		t.Errorf("Unexpected VM state after creation: %s (expected: not_created)", state)
	}
}

// TestHandleCreateDevVM_ValidationError tests parameter validation
func TestHandleCreateDevVM_ValidationError(t *testing.T) {
	// Setup fixture without VM
	fixture, err := testfixture.NewUnifiedFixture(t, testfixture.FixtureOptions{
		PackageName:   "vm-tools-validation",
		SetupVM:       false,
		CreateProject: true,
	})
	if err != nil {
		t.Fatalf("Failed to set up test fixture: %v", err)
	}
	defer fixture.Cleanup()

	// Create handler function using the same logic as RegisterVMTools
	handlerFunc := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		type CreateVMArgs struct {
			Name            string                   `json:"name"`
			ProjectPath     string                   `json:"project_path"`
			CPU             float64                  `json:"cpu"`
			Memory          float64                  `json:"memory"`
			Box             string                   `json:"box"`
			SyncType        string                   `json:"sync_type"`
			Ports           []map[string]interface{} `json:"ports"`
			ExcludePatterns []string                 `json:"exclude_patterns"`
		}
		var args CreateVMArgs
		if err := request.BindArguments(&args); err != nil {
			return mcp.NewToolResultError("Invalid arguments: " + err.Error()), nil
		}
		if args.Name == "" || args.ProjectPath == "" {
			return mcp.NewToolResultError("Missing required parameter: name or project_path"), nil
		}

		var ports []core.Port
		for _, portMap := range args.Ports {
			var port core.Port
			if guest, ok := portMap["guest"].(float64); ok {
				port.Guest = int(guest)
			}
			if host, ok := portMap["host"].(float64); ok {
				port.Host = int(host)
			}
			ports = append(ports, port)
		}
		if len(ports) == 0 {
			ports = []core.Port{
				{Guest: 3000, Host: 3000},
				{Guest: 8000, Host: 8000},
				{Guest: 5432, Host: 5432},
				{Guest: 3306, Host: 3306},
				{Guest: 6379, Host: 6379},
			}
		}
		excludePatterns := args.ExcludePatterns
		if len(excludePatterns) == 0 {
			excludePatterns = []string{"node_modules", ".git", "*.log", "dist", "build", "__pycache__", "*.pyc", "venv", ".venv", "*.o", "*.out"}
		}
		config := core.VMConfig{
			Box:                 args.Box,
			CPU:                 int(args.CPU),
			Memory:              int(args.Memory),
			SyncType:            args.SyncType,
			Ports:               ports,
			SyncExcludePatterns: excludePatterns,
		}
		if err := fixture.VMManager.CreateVM(ctx, args.Name, args.ProjectPath, config); err != nil {
			return mcp.NewToolResultErrorf("Failed to create VM: %v", err), nil
		}
		response := map[string]interface{}{
			"name":         args.Name,
			"project_path": args.ProjectPath,
			"config":       config,
			"status":       "created",
			"timestamp":    time.Now().Format(time.RFC3339),
		}
		jsonResponse, err := json.Marshal(response)
		if err != nil {
			return mcp.NewToolResultError("Failed to marshal response"), nil
		}
		return mcp.NewToolResultText(string(jsonResponse)), nil
	}

	// Create MCP request with missing required parameter
	request := mcp.CallToolRequest{
		Params: mcpgo.CallToolParams{
			Name: "create_dev_vm",
			Arguments: map[string]interface{}{
				"name": "test-vm", // Missing project_path which is required
			},
		},
	}

	// Call the handler function
	resp, err := handlerFunc(context.Background(), request)

	// Should get a validation error
	if err == nil && (resp == nil || !resp.IsError) {
		t.Error("Expected error but got none")
	}

	if resp != nil && resp.IsError {
		msg := extractTextContent(resp.Content)
		expectedError := "Missing required parameter: name or project_path"
		if !strings.Contains(msg, expectedError) {
			t.Errorf("Expected error '%s' but got '%s'", expectedError, msg)
		} else {
			t.Logf("Got expected validation error: %s", msg)
		}
	}
}
