package handlers

import (
	"context"
	"strings"
	"testing"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/vagrant-mcp/server/internal/vm"
	"github.com/vagrant-mcp/server/pkg/mcp"
)

func TestHandleCreateDevVM(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name          string
		params        map[string]interface{}
		mockManager   *mockVMManager
		expectError   bool
		expectedError string
	}{
		{
			name: "successful creation",
			params: map[string]interface{}{
				"name":         "test-vm",
				"project_path": "/test/project",
				"box":          "ubuntu/focal64",
				"cpu":          float64(2),
				"memory":       float64(2048),
			},
			mockManager: &mockVMManager{
				createVM: func(name, projectPath string, config vm.VMConfig) error {
					return nil
				},
			},
			expectError: false,
		},
		{
			name: "missing required fields",
			params: map[string]interface{}{
				"name": "test-vm",
			},
			mockManager:   &mockVMManager{},
			expectError:   true,
			expectedError: "Missing required parameter: project_path",
		},
		{
			name: "vm creation error",
			params: map[string]interface{}{
				"name":         "test-vm",
				"project_path": "/test/project",
				"box":          "ubuntu/focal64",
				"cpu":          float64(2),
				"memory":       float64(2048),
			},
			mockManager: &mockVMManager{
				createVM: func(name, projectPath string, config vm.VMConfig) error {
					return vm.ErrVMExists
				},
			},
			expectError:   true,
			expectedError: "already exists",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			handlerFunc := handleCreateDevVM(tc.mockManager)
			request := mcp.CallToolRequest{
				Params: mcpgo.CallToolParams{
					Name:      "create_dev_vm",
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
