package handlers

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/vagrant-mcp/server/internal/core"
	testfixture "github.com/vagrant-mcp/server/internal/testing"
	"github.com/vagrant-mcp/server/pkg/mcp"
)

// Use the testFixture from test_helper.go for all VM operations

// extractTextContent extracts the first text content from a slice of Content (any type)
func extractTextContent(contents interface{}) string {
	// Always marshal to JSON and parse as []map[string]interface{}
	b, err := json.Marshal(contents)
	if err != nil {
		return ""
	}
	var arr []map[string]interface{}
	if err := json.Unmarshal(b, &arr); err != nil {
		return ""
	}
	for _, m := range arr {
		if t, ok := m["type"].(string); ok && t == "text" {
			if txt, ok := m["text"].(string); ok {
				return txt
			}
		}
	}
	return ""
}

func TestVMTools_HandleRequest(t *testing.T) {
	// Skip by default - can be enabled with TEST_LEVEL=integration
	testLevel := os.Getenv("TEST_LEVEL")
	if testLevel != "integration" && testLevel != "vm-start" {
		t.Skip("Skipping integration test. Set TEST_LEVEL=integration to run")
		return
	}

	// Create test fixture with real VM manager
	fixture, err := testfixture.NewUnifiedFixture(t, testfixture.FixtureOptions{
		PackageName:   "handlers",
		SetupVM:       false,
		CreateProject: true,
	})
	if err != nil {
		t.Fatalf("Failed to set up test fixture: %v", err)
	}
	defer fixture.Cleanup()

	testCases := []struct {
		name          string
		toolName      string
		params        map[string]interface{}
		expectError   bool
		expectedError string
	}{
		{
			name:     "create vm successful",
			toolName: "create_dev_vm",
			params: map[string]interface{}{
				"name":         fixture.VMName,
				"project_path": fixture.ProjectPath,
				"box":          "generic/alpine314",
				"memory":       float64(512),
				"cpu":          float64(1),
			},
			expectError: false,
		},
		{
			name:     "get vm status",
			toolName: "get_vm_status",
			params: map[string]interface{}{
				"name": fixture.VMName,
			},
			expectError: false,
		},
		{
			name:     "destroy vm successful",
			toolName: "destroy_dev_vm",
			params: map[string]interface{}{
				"name": fixture.VMName,
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		tc := tc // capture range variable
		t.Run(tc.name, func(t *testing.T) {
			// Don't run tests in parallel when using real VM operations
			var handlerFunc func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)
			switch tc.toolName {
			case "create_dev_vm":
				handlerFunc = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
						"timestamp":    "test",
					}
					jsonResponse, err := json.Marshal(response)
					if err != nil {
						return mcp.NewToolResultError("Failed to marshal response"), nil
					}
					return mcp.NewToolResultText(string(jsonResponse)), nil
				}
			case "destroy_dev_vm":
				handlerFunc = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
					type DestroyVMArgs struct {
						Name string `json:"name"`
					}
					var args DestroyVMArgs
					if err := request.BindArguments(&args); err != nil {
						return mcp.NewToolResultError("Invalid arguments: " + err.Error()), nil
					}
					if args.Name == "" {
						return mcp.NewToolResultError("Missing required parameter: name"), nil
					}
					if err := fixture.VMManager.DestroyVM(ctx, args.Name); err != nil {
						return mcp.NewToolResultErrorf("Failed to destroy VM: %v", err), nil
					}
					return mcp.NewToolResultText("VM destroyed"), nil
				}
			case "get_vm_status":
				handlerFunc = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
					type GetVMStatusArgs struct {
						Name string `json:"name"`
					}
					var args GetVMStatusArgs
					if err := request.BindArguments(&args); err != nil {
						return mcp.NewToolResultError("Invalid arguments: " + err.Error()), nil
					}
					if args.Name != "" {
						state, err := fixture.VMManager.GetVMState(ctx, args.Name)
						if err != nil {
							return mcp.NewToolResultErrorf("Failed to get VM status: %v", err), nil
						}
						response := map[string]interface{}{
							"name":  args.Name,
							"state": state,
						}
						jsonResponse, err := json.Marshal(response)
						if err != nil {
							return mcp.NewToolResultError("Failed to marshal response"), nil
						}
						return mcp.NewToolResultText(string(jsonResponse)), nil
					}
					// Simulate listing VMs
					vmNames := []string{fixture.VMName}
					vmStates := make([]map[string]interface{}, 0, len(vmNames))
					for _, vmName := range vmNames {
						state, err := fixture.VMManager.GetVMState(ctx, vmName)
						var stateStr string
						if err != nil {
							stateStr = "unknown"
						} else {
							stateStr = string(state)
						}
						vmStates = append(vmStates, map[string]interface{}{
							"name":  vmName,
							"state": stateStr,
						})
					}
					response := map[string]interface{}{
						"vms": vmStates,
					}
					jsonResponse, err := json.Marshal(response)
					if err != nil {
						return mcp.NewToolResultError("Failed to marshal response"), nil
					}
					return mcp.NewToolResultText(string(jsonResponse)), nil
				}
			default:
				t.Fatalf("Unknown toolName: %s", tc.toolName)
			}

			request := mcp.CallToolRequest{
				Params: mcpgo.CallToolParams{
					Name:      tc.toolName,
					Arguments: tc.params,
				},
			}

			resp, err := handlerFunc(context.Background(), request)

			if tc.expectError {
				if err == nil && (resp == nil || !resp.IsError) {
					t.Error("Expected error but got none")
				}
				if tc.expectedError != "" && resp != nil && resp.IsError {
					msg := extractTextContent(resp.Content)
					if !strings.Contains(msg, tc.expectedError) {
						t.Errorf("Expected error '%s' but got '%s'", tc.expectedError, msg)
					}
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if resp == nil || resp.IsError {
					msg := ""
					if resp != nil {
						msg = extractTextContent(resp.Content)
					}
					t.Errorf("Expected non-error result, got error: %s", msg)
				}
				if resp == nil || extractTextContent(resp.Content) == "" {
					t.Error("Expected non-nil result content")
				}
			}
		})
	}
}
