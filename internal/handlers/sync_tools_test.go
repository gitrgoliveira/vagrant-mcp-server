package handlers

import (
	"context"
	"os"
	"strings"
	"testing"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
	testfixture "github.com/vagrant-mcp/server/internal/testing"
	"github.com/vagrant-mcp/server/pkg/mcp"
)

// Use the testFixture from test_helper.go for all VM operations

// TestUploadToVMHandler tests the upload_to_vm handler function
func TestUploadToVMHandler(t *testing.T) {
	// Skip by default - can be enabled with TEST_LEVEL=integration
	testLevel := os.Getenv("TEST_LEVEL")
	if testLevel != "integration" && testLevel != "vm-start" {
		t.Skip("Skipping integration test. Set TEST_LEVEL=integration to run")
		return
	}

	// Create test fixture with real VM manager
	fixture, err := testfixture.NewUnifiedFixture(t, testfixture.FixtureOptions{
		PackageName:   "sync-tools",
		SetupVM:       false,
		CreateProject: true,
	})
	if err != nil {
		t.Fatalf("Failed to set up test fixture: %v", err)
	}
	defer fixture.Cleanup()

	testCases := []struct {
		name          string
		params        map[string]interface{}
		expectError   bool
		expectedError string
	}{
		{
			name: "missing vm_name",
			params: map[string]interface{}{
				"source":      "/tmp/source",
				"destination": "/tmp/destination",
			},
			expectError:   true,
			expectedError: "Missing or invalid 'vm_name' parameter",
		},
		{
			name: "missing source",
			params: map[string]interface{}{
				"vm_name":     fixture.VMName,
				"destination": "/tmp/destination",
			},
			expectError:   true,
			expectedError: "Missing or invalid 'source' parameter",
		},
		{
			name: "missing destination",
			params: map[string]interface{}{
				"vm_name": fixture.VMName,
				"source":  "/tmp/source",
			},
			expectError:   true,
			expectedError: "Missing or invalid 'destination' parameter",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			handler := handleUploadToVM(fixture.VMManager)

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
