package handlers

import (
	"context"
	"strings"
	"testing"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/vagrant-mcp/server/internal/vm"
	"github.com/vagrant-mcp/server/pkg/mcp"
)

// Custom mock VM manager for sync tools tests
type syncToolsMockVMManager struct {
	getState   func(name string) (vm.State, error)
	uploadToVM func(name, source, destination string, compress bool, compressionType string) error
}

func (m *syncToolsMockVMManager) CreateVM(name, projectPath string, config vm.VMConfig) error {
	return nil
}
func (m *syncToolsMockVMManager) DestroyVM(name string) error { return nil }
func (m *syncToolsMockVMManager) GetVMState(name string) (vm.State, error) {
	if m.getState != nil {
		return m.getState(name)
	}
	return vm.NotCreated, nil
}
func (m *syncToolsMockVMManager) StartVM(name string) error { return nil }
func (m *syncToolsMockVMManager) StopVM(name string) error  { return nil }
func (m *syncToolsMockVMManager) ExecuteCommand(name string, cmd string, args []string, workingDir string) (string, string, int, error) {
	return "", "", 0, nil
}
func (m *syncToolsMockVMManager) SyncToVM(name, source, target string) error   { return nil }
func (m *syncToolsMockVMManager) SyncFromVM(name, source, target string) error { return nil }
func (m *syncToolsMockVMManager) UploadToVM(name, source, destination string, compress bool, compressionType string) error {
	if m.uploadToVM != nil {
		return m.uploadToVM(name, source, destination, compress, compressionType)
	}
	return nil
}
func (m *syncToolsMockVMManager) GetSSHConfig(name string) (map[string]string, error) {
	return map[string]string{}, nil
}
func (m *syncToolsMockVMManager) GetVMConfig(name string) (vm.VMConfig, error) {
	return vm.VMConfig{}, nil
}
func (m *syncToolsMockVMManager) UpdateVMConfig(name string, config vm.VMConfig) error { return nil }
func (m *syncToolsMockVMManager) GetBaseDir() string                                   { return "/mock/base/dir" }

// TestUploadToVMHandler tests the upload_to_vm handler function
func TestUploadToVMHandler(t *testing.T) {
	testCases := []struct {
		name          string
		params        map[string]interface{}
		vmState       vm.State
		vmError       error
		uploadError   error
		expectError   bool
		expectedError string
	}{
		{
			name: "successful upload",
			params: map[string]interface{}{
				"vm_name":     "test-vm",
				"source":      "/path/to/source",
				"destination": "/path/to/destination",
			},
			vmState:     vm.Running,
			expectError: false,
		},
		{
			name: "missing vm_name",
			params: map[string]interface{}{
				"source":      "/path/to/source",
				"destination": "/path/to/destination",
			},
			expectError:   true,
			expectedError: "Missing or invalid 'vm_name' parameter",
		},
		{
			name: "missing source",
			params: map[string]interface{}{
				"vm_name":     "test-vm",
				"destination": "/path/to/destination",
			},
			expectError:   true,
			expectedError: "Missing or invalid 'source' parameter",
		},
		{
			name: "missing destination",
			params: map[string]interface{}{
				"vm_name": "test-vm",
				"source":  "/path/to/source",
			},
			expectError:   true,
			expectedError: "Missing or invalid 'destination' parameter",
		},
		{
			name: "vm not running",
			params: map[string]interface{}{
				"vm_name":     "test-vm",
				"source":      "/path/to/source",
				"destination": "/path/to/destination",
			},
			vmState:       vm.Stopped,
			expectError:   true,
			expectedError: "not running",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockManager := &syncToolsMockVMManager{
				getState: func(name string) (vm.State, error) {
					return tc.vmState, tc.vmError
				},
				uploadToVM: func(name, source, destination string, compress bool, compressionType string) error {
					return tc.uploadError
				},
			}

			handler := handleUploadToVM(mockManager)

			// Create request
			request := mcp.CallToolRequest{
				Params: mcpgo.CallToolParams{
					Name:      "upload_to_vm",
					Arguments: tc.params,
				},
			}

			// Call handler
			result, err := handler(context.Background(), request)

			// Check for unexpected errors
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Check expected errors
			if tc.expectError {
				if !result.IsError {
					t.Errorf("Expected error, got success result")
				}
				if tc.expectedError != "" {
					resultText := extractTextContent(result.Content)
					if !strings.Contains(resultText, tc.expectedError) {
						t.Errorf("Expected error to contain '%s', got '%s'", tc.expectedError, resultText)
					}
				}
			} else {
				if result.IsError {
					resultText := extractTextContent(result.Content)
					t.Errorf("Expected success result, got error: %s", resultText)
				}
			}
		})
	}
}
