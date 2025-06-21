package resources

import (
	"testing"
)

func TestResourceRegistration(t *testing.T) {
	testCases := []struct {
		name          string
		resourceType  string
		expectError   bool
		expectedError string
	}{
		{
			name:         "register valid resource",
			resourceType: "vm",
			expectError:  false,
		},
		{
			name:          "register empty resource type",
			resourceType:  "",
			expectError:   true,
			expectedError: "invalid resource type",
		},
		{
			name:          "register duplicate resource",
			resourceType:  "vm",
			expectError:   true,
			expectedError: "resource type already registered",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := RegisterResourceType(tc.resourceType)
			if tc.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				if tc.expectedError != "" && err.Error() != tc.expectedError {
					t.Errorf("Expected error '%s' but got '%s'", tc.expectedError, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %s", err)
				}

				// Verify resource is registered
				if !IsResourceTypeRegistered(tc.resourceType) {
					t.Errorf("Resource type %s was not registered", tc.resourceType)
				}
			}
		})
	}
}

func TestResourceValidation(t *testing.T) {
	testCases := []struct {
		name          string
		resourceType  string
		resourceData  map[string]interface{}
		expectError   bool
		expectedError string
	}{
		{
			name:         "valid vm resource",
			resourceType: "vm",
			resourceData: map[string]interface{}{
				"name": "test-vm",
				"config": map[string]interface{}{
					"memory": float64(2048),
					"cpu":    float64(2),
				},
			},
			expectError: false,
		},
		{
			name:         "invalid vm resource - missing name",
			resourceType: "vm",
			resourceData: map[string]interface{}{
				"config": map[string]interface{}{
					"memory": float64(2048),
					"cpu":    float64(2),
				},
			},
			expectError:   true,
			expectedError: "missing required field 'name'",
		},
		{
			name:          "unregistered resource type",
			resourceType:  "unknown",
			resourceData:  map[string]interface{}{},
			expectError:   true,
			expectedError: "unknown resource type",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Register VM resource type first
			if tc.resourceType == "vm" {
				_ = RegisterResourceType("vm")
			}

			err := ValidateResource(tc.resourceType, tc.resourceData)
			if tc.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				if tc.expectedError != "" && err.Error() != tc.expectedError {
					t.Errorf("Expected error '%s' but got '%s'", tc.expectedError, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %s", err)
				}
			}
		})
	}
}

func TestResourceConversion(t *testing.T) {
	testCases := []struct {
		name         string
		resourceType string
		resourceData map[string]interface{}
		expectError  bool
	}{
		{
			name:         "convert vm resource",
			resourceType: "vm",
			resourceData: map[string]interface{}{
				"name": "test-vm",
				"config": map[string]interface{}{
					"memory": float64(2048),
					"cpu":    float64(2),
					"ports": []interface{}{
						map[string]interface{}{
							"guest": float64(80),
							"host":  float64(8080),
						},
					},
				},
			},
			expectError: false,
		},
		{
			name:         "invalid port mapping",
			resourceType: "vm",
			resourceData: map[string]interface{}{
				"name": "test-vm",
				"config": map[string]interface{}{
					"memory": float64(2048),
					"cpu":    float64(2),
					"ports": []interface{}{
						map[string]interface{}{
							"guest": "invalid",
						},
					},
				},
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Register VM resource type first
			if tc.resourceType == "vm" {
				_ = RegisterResourceType("vm")
			}

			converted, err := ConvertResource(tc.resourceType, tc.resourceData)
			if tc.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %s", err)
				}
				if converted == nil {
					t.Error("Expected non-nil converted resource")
				}
			}
		})
	}
}
