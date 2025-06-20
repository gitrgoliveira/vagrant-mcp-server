package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetupDevEnvironmentTool_Name(t *testing.T) {
	tool := &SetupDevEnvironmentTool{}
	assert.Equal(t, "setup_dev_environment", tool.Name())
}

func TestSetupDevEnvironmentTool_Description(t *testing.T) {
	tool := &SetupDevEnvironmentTool{}
	assert.Contains(t, tool.Description(), "Install language runtimes")
}

func TestInstallDevToolsTool_Name(t *testing.T) {
	tool := &InstallDevToolsTool{}
	assert.Equal(t, "install_dev_tools", tool.Name())
}

func TestConfigureShellTool_Name(t *testing.T) {
	tool := &ConfigureShellTool{}
	assert.Equal(t, "configure_shell", tool.Name())
}

// Add more tests for Execute methods with mocks for vm.Manager and exec.Executor as needed.
