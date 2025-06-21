package sync

import (
	"testing"
)

func TestSyncEngine_RegisterVM(t *testing.T) {
	testCases := []struct {
		name          string
		vmName        string
		expectError   bool
		expectedError string
	}{
		{
			name:        "successful registration",
			vmName:      "test-vm",
			expectError: false,
		},
		{
			name:          "duplicate registration",
			vmName:        "test-vm",
			expectError:   true,
			expectedError: "vm already registered",
		},
		{
			name:          "empty vm name",
			vmName:        "",
			expectError:   true,
			expectedError: "invalid vm name",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			engine, _ := NewEngine()

			// For duplicate registration test case, register first
			if tc.expectedError == "vm already registered" {
				_ = engine.RegisterVM(tc.vmName, SyncConfig{VMName: tc.vmName})
			}

			err := engine.RegisterVM(tc.vmName, SyncConfig{VMName: tc.vmName})

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

func TestSyncEngine_UnregisterVM(t *testing.T) {
	testCases := []struct {
		name          string
		vmName        string
		register      bool // whether to register the VM first
		expectError   bool
		expectedError string
	}{
		{
			name:        "successful unregistration",
			vmName:      "test-vm",
			register:    true,
			expectError: false,
		},
		{
			name:          "vm not registered",
			vmName:        "test-vm",
			register:      false,
			expectError:   true,
			expectedError: "vm not registered",
		},
		{
			name:          "empty vm name",
			vmName:        "",
			register:      false,
			expectError:   true,
			expectedError: "invalid vm name",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			engine, _ := NewEngine()

			if tc.register {
				_ = engine.RegisterVM(tc.vmName, SyncConfig{VMName: tc.vmName})
			}

			err := engine.UnregisterVM(tc.vmName)

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

func TestSyncEngine_StartStop(t *testing.T) {
	testCases := []struct {
		name           string
		operation      string // "start" or "stop"
		alreadyStarted bool
		expectError    bool
		expectedError  string
	}{
		{
			name:           "successful start",
			operation:      "start",
			alreadyStarted: false,
			expectError:    false,
		},
		{
			name:           "start already running",
			operation:      "start",
			alreadyStarted: true,
			expectError:    true,
			expectedError:  "sync engine already running",
		},
		{
			name:           "successful stop",
			operation:      "stop",
			alreadyStarted: true,
			expectError:    false,
		},
		{
			name:           "stop not running",
			operation:      "stop",
			alreadyStarted: false,
			expectError:    true,
			expectedError:  "sync engine not running",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			engine, _ := NewEngine()

			if tc.alreadyStarted {
				_ = engine.Start()
			}

			var err error
			if tc.operation == "start" {
				err = engine.Start()
			} else {
				err = engine.Stop()
			}

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

func TestEngine_IsRunning(t *testing.T) {
	engine, _ := NewEngine()

	// Should start as not running
	if engine.IsRunning() {
		t.Error("Expected engine to not be running initially")
	}

	// Start the engine
	if err := engine.Start(); err != nil {
		t.Fatalf("Failed to start engine: %v", err)
	}

	if !engine.IsRunning() {
		t.Error("Expected engine to be running after Start")
	}

	// Stop the engine
	if err := engine.Stop(); err != nil {
		t.Fatalf("Failed to stop engine: %v", err)
	}

	if engine.IsRunning() {
		t.Error("Expected engine to not be running after Stop")
	}
}
