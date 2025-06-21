package handlers

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/vagrant-mcp/server/internal/vm"
	"github.com/vagrant-mcp/server/pkg/mcp"
)

// mockVMManager implements the exec.VMManager interface for testing
// Only the methods used in these tests are implemented with logic; others are stubs.
type mockVMManager struct {
	createVM  func(name, projectPath string, config vm.VMConfig) error
	destroyVM func(name string) error
	getState  func(name string) (vm.State, error)
}

func (m *mockVMManager) CreateVM(name, projectPath string, config vm.VMConfig) error {
	if m.createVM != nil {
		return m.createVM(name, projectPath, config)
	}
	return nil
}
func (m *mockVMManager) DestroyVM(name string) error {
	if m.destroyVM != nil {
		return m.destroyVM(name)
	}
	return nil
}
func (m *mockVMManager) GetVMState(name string) (vm.State, error) {
	if m.getState != nil {
		return m.getState(name)
	}
	return vm.State("not_created"), nil
}
func (m *mockVMManager) StartVM(name string) error { return nil }
func (m *mockVMManager) StopVM(name string) error  { return nil }
func (m *mockVMManager) ExecuteCommand(name string, cmd string, args []string, workingDir string) (string, string, int, error) {
	return "", "", 0, nil
}
func (m *mockVMManager) SyncToVM(name, source, target string) error   { return nil }
func (m *mockVMManager) SyncFromVM(name, source, target string) error { return nil }
func (m *mockVMManager) GetSSHConfig(name string) (map[string]string, error) {
	return map[string]string{}, nil
}
func (m *mockVMManager) GetVMConfig(name string) (vm.VMConfig, error)         { return vm.VMConfig{}, nil }
func (m *mockVMManager) UpdateVMConfig(name string, config vm.VMConfig) error { return nil }
func (m *mockVMManager) GetBaseDir() string                                   { return "/mock/base/dir" }

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
	t.Parallel()
	testCases := []struct {
		name          string
		toolName      string
		params        map[string]interface{}
		mockManager   *mockVMManager
		expectError   bool
		expectedError string
	}{
		{
			name:     "create vm successful",
			toolName: "create_dev_vm",
			params: map[string]interface{}{
				"name":         "test-vm",
				"project_path": "/test/project",
				"box":          "test/box",
				"memory":       float64(2048),
				"cpu":          float64(2),
			},
			mockManager: &mockVMManager{
				createVM: func(name, projectPath string, config vm.VMConfig) error {
					return nil
				},
			},
			expectError: false,
		},
		{
			name:     "destroy vm successful",
			toolName: "destroy_dev_vm",
			params: map[string]interface{}{
				"name": "test-vm",
			},
			mockManager: &mockVMManager{
				destroyVM: func(name string) error {
					return nil
				},
			},
			expectError: false,
		},
		{
			name:     "get vm state successful",
			toolName: "get_vm_status",
			params: map[string]interface{}{
				"name": "test-vm",
			},
			mockManager: &mockVMManager{
				getState: func(name string) (vm.State, error) {
					return vm.State("running"), nil
				},
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		tc := tc // capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var handlerFunc func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)
			switch tc.toolName {
			case "create_dev_vm":
				handlerFunc = handleCreateDevVM(tc.mockManager)
			case "destroy_dev_vm":
				handlerFunc = handleDestroyDevVM(tc.mockManager)
			case "get_vm_status":
				handlerFunc = handleGetVMStatus(tc.mockManager)
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
