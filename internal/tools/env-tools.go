package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/vagrant-mcp/server/internal/exec"
	"github.com/vagrant-mcp/server/internal/vm"
)

// SetupDevEnvironmentTool implements the setup_dev_environment tool
type SetupDevEnvironmentTool struct {
	manager  *vm.Manager
	executor *exec.Executor
}

// Name returns the tool name
func (t *SetupDevEnvironmentTool) Name() string {
	return "setup_dev_environment"
}

// Description returns the tool description
func (t *SetupDevEnvironmentTool) Description() string {
	return "Install language runtimes, tools, and dependencies in the VM"
}

// Execute performs the tool action
func (t *SetupDevEnvironmentTool) Execute(params map[string]interface{}) (interface{}, error) {
	// Validate required parameters
	vmName, ok := params["vm_name"].(string)
	if !ok || vmName == "" {
		return nil, fmt.Errorf("missing or invalid 'vm_name' parameter")
	}

	runtimesParam, ok := params["runtimes"].([]interface{})
	if !ok || len(runtimesParam) == 0 {
		return nil, fmt.Errorf("missing or invalid 'runtimes' parameter")
	}

	// Extract runtimes
	runtimes := make([]string, len(runtimesParam))
	for i, rt := range runtimesParam {
		rtStr, ok := rt.(string)
		if !ok {
			return nil, fmt.Errorf("runtime at index %d is not a string", i)
		}
		runtimes[i] = rtStr
	}

	// Get optional versions parameter
	versions := make(map[string]string)
	if versionsObj, ok := params["versions"].(map[string]interface{}); ok {
		for key, val := range versionsObj {
			if valStr, ok := val.(string); ok {
				versions[key] = valStr
			}
		}
	}

	// Check if VM exists and is running
	state, err := t.manager.GetVMState(vmName)
	if err != nil {
		return nil, fmt.Errorf("VM '%s' does not exist or cannot be accessed: %w", vmName, err)
	}

	if state != vm.StateRunning {
		return nil, fmt.Errorf("VM '%s' is not running (current state: %s)", vmName, state)
	}

	results := make(map[string]interface{})
	for _, runtime := range runtimes {
		// Get version if specified
		version := versions[runtime]

		// Generate setup script based on runtime
		setupScript := t.generateSetupScript(runtime, version)
		if setupScript == "" {
			log.Warn().Str("runtime", runtime).Msg("Unsupported runtime, skipping")
			results[runtime] = map[string]interface{}{
				"status":  "skipped",
				"message": "Unsupported runtime",
			}
			continue
		}

		// Execute setup script
		log.Info().Str("vm", vmName).Str("runtime", runtime).Msg("Setting up runtime")

		// Setup execution context
		execCtx := exec.ExecutionContext{
			VMName:      vmName,
			WorkingDir:  "/tmp", // Use /tmp for setup scripts
			Environment: make(map[string]string),
			SyncBefore:  false, // No need to sync for setup
			SyncAfter:   false,
		}

		// Execute command
		ctx := context.Background()
		result, err := t.executor.ExecuteCommand(ctx, setupScript, execCtx, nil)

		if err != nil {
			log.Error().Err(err).Str("runtime", runtime).Msg("Failed to execute setup script")
			results[runtime] = map[string]interface{}{
				"status":  "error",
				"message": err.Error(),
			}
			continue
		}

		if result.ExitCode != 0 {
			log.Error().
				Str("runtime", runtime).
				Int("exit_code", result.ExitCode).
				Str("stderr", result.Stderr).
				Msg("Runtime setup failed")

			results[runtime] = map[string]interface{}{
				"status":    "failed",
				"exit_code": result.ExitCode,
				"stderr":    result.Stderr,
			}
			continue
		}

		// Success
		log.Info().Str("runtime", runtime).Msg("Runtime setup completed successfully")
		results[runtime] = map[string]interface{}{
			"status":    "success",
			"version":   version,
			"exit_code": result.ExitCode,
		}
	}

	return map[string]interface{}{
		"vm_name":  vmName,
		"runtimes": results,
		"success":  true,
	}, nil
}

// generateSetupScript creates a shell script to install the specified runtime
func (t *SetupDevEnvironmentTool) generateSetupScript(runtime, version string) string {
	switch runtime {
	case "nodejs":
		if version == "" {
			version = "18"
		}
		return fmt.Sprintf(`
			#!/bin/bash
			set -e
			echo "Setting up Node.js %s..."
			curl -sL https://deb.nodesource.com/setup_%s.x | sudo -E bash -
			sudo apt-get install -y nodejs
			sudo npm install -g npm@latest
			node --version
			npm --version
		`, version, version)

	case "python":
		if version == "" {
			version = "3"
		}
		return fmt.Sprintf(`
			#!/bin/bash
			set -e
			echo "Setting up Python %s..."
			sudo apt-get update
			sudo apt-get install -y python%s python%s-venv python%s-dev python%s-pip
			sudo pip%s install --upgrade pip setuptools wheel
			python%s --version
			pip%s --version
		`, version, version, version, version, version, version, version, version)

	case "ruby":
		if version == "" {
			version = "3.0"
		}
		return fmt.Sprintf(`
			#!/bin/bash
			set -e
			echo "Setting up Ruby %s..."
			sudo apt-get update
			sudo apt-get install -y software-properties-common
			sudo apt-add-repository -y ppa:rael-gc/rvm
			sudo apt-get update
			sudo apt-get install -y rvm
			source /etc/profile.d/rvm.sh
			rvm install %s
			rvm use %s --default
			ruby --version
			gem --version
		`, version, version, version)

	case "go":
		if version == "" {
			version = "1.20"
		}
		return fmt.Sprintf(`
			#!/bin/bash
			set -e
			echo "Setting up Go %s..."
			wget https://golang.org/dl/go%s.linux-amd64.tar.gz
			sudo rm -rf /usr/local/go
			sudo tar -C /usr/local -xzf go%s.linux-amd64.tar.gz
			rm go%s.linux-amd64.tar.gz
			echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
			echo 'export PATH=$PATH:~/go/bin' >> ~/.bashrc
			export PATH=$PATH:/usr/local/go/bin
			go version
		`, version, version, version, version)

	case "java":
		if version == "" {
			version = "17"
		}
		return fmt.Sprintf(`
			#!/bin/bash
			set -e
			echo "Setting up Java %s..."
			sudo apt-get update
			sudo apt-get install -y openjdk-%s-jdk
			java -version
		`, version, version)

	case "docker":
		return `
			#!/bin/bash
			set -e
			echo "Setting up Docker..."
			sudo apt-get update
			sudo apt-get install -y apt-transport-https ca-certificates curl gnupg lsb-release
			curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg
			echo "deb [arch=amd64 signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
			sudo apt-get update
			sudo apt-get install -y docker-ce docker-ce-cli containerd.io
			sudo systemctl enable docker
			sudo systemctl start docker
			sudo usermod -aG docker vagrant
			docker --version
		`

	case "postgres":
		if version == "" {
			version = "14"
		}
		return fmt.Sprintf(`
			#!/bin/bash
			set -e
			echo "Setting up PostgreSQL %s..."
			sudo apt-get update
			sudo apt-get install -y postgresql-%s postgresql-%s-contrib
			sudo systemctl enable postgresql
			sudo systemctl start postgresql
			sudo -u postgres createuser --superuser vagrant || echo "User already exists"
			sudo -u postgres createdb -O vagrant vagrant || echo "Database already exists"
			psql --version
		`, version, version, version)

	case "mysql":
		return `
			#!/bin/bash
			set -e
			echo "Setting up MySQL..."
			sudo apt-get update
			sudo debconf-set-selections <<< 'mysql-server mysql-server/root_password password vagrant'
			sudo debconf-set-selections <<< 'mysql-server mysql-server/root_password_again password vagrant'
			sudo apt-get install -y mysql-server
			sudo systemctl enable mysql
			sudo systemctl start mysql
			mysql --version
		`

	default:
		return ""
	}
}

// InstallDevToolsTool implements the install_dev_tools tool
type InstallDevToolsTool struct {
	manager  *vm.Manager
	executor *exec.Executor
}

// Name returns the tool name
func (t *InstallDevToolsTool) Name() string {
	return "install_dev_tools"
}

// Description returns the tool description
func (t *InstallDevToolsTool) Description() string {
	return "Install development tools in the VM"
}

// Execute performs the tool action
func (t *InstallDevToolsTool) Execute(params map[string]interface{}) (interface{}, error) {
	// Validate required parameters
	vmName, ok := params["vm_name"].(string)
	if !ok || vmName == "" {
		return nil, fmt.Errorf("missing or invalid 'vm_name' parameter")
	}

	toolsParam, ok := params["tools"].([]interface{})
	if !ok || len(toolsParam) == 0 {
		return nil, fmt.Errorf("missing or invalid 'tools' parameter")
	}

	// Extract tools list
	tools := make([]string, len(toolsParam))
	for i, t := range toolsParam {
		tStr, ok := t.(string)
		if !ok {
			return nil, fmt.Errorf("tool at index %d is not a string", i)
		}
		tools[i] = tStr
	}

	// Check VM state
	state, err := t.manager.GetVMState(vmName)
	if err != nil {
		return nil, fmt.Errorf("VM '%s' not found: %w", vmName, err)
	}

	if state != vm.StateRunning {
		return nil, fmt.Errorf("VM '%s' is not running (current state: %s)", vmName, state)
	}

	// Setup execution context
	execCtx := exec.ExecutionContext{
		VMName:      vmName,
		WorkingDir:  "/tmp",
		Environment: make(map[string]string),
		SyncBefore:  false,
		SyncAfter:   false,
	}

	// Update apt cache first
	updateCmd := "sudo apt-get update"
	ctx := context.Background()
	_, err = t.executor.ExecuteCommand(ctx, updateCmd, execCtx, nil)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to update apt cache")
	}

	// Install each tool
	results := make(map[string]interface{})
	for _, tool := range tools {
		var installCmd string

		switch tool {
		case "build-essential":
			installCmd = "sudo apt-get install -y build-essential"
		case "git":
			installCmd = "sudo apt-get install -y git"
		case "vim":
			installCmd = "sudo apt-get install -y vim"
		case "nano":
			installCmd = "sudo apt-get install -y nano"
		case "curl":
			installCmd = "sudo apt-get install -y curl"
		case "wget":
			installCmd = "sudo apt-get install -y wget"
		case "htop":
			installCmd = "sudo apt-get install -y htop"
		case "jq":
			installCmd = "sudo apt-get install -y jq"
		case "zip":
			installCmd = "sudo apt-get install -y zip unzip"
		case "make":
			installCmd = "sudo apt-get install -y make"
		case "tmux":
			installCmd = "sudo apt-get install -y tmux"
		case "git-lfs":
			installCmd = "sudo apt-get install -y git-lfs && git lfs install"
		case "docker-cli":
			installCmd = "if ! command -v docker >/dev/null; then curl -fsSL https://get.docker.com | sudo sh; sudo usermod -aG docker vagrant; fi"
		case "docker-compose":
			installCmd = "sudo apt-get install -y docker-compose"
		case "kubectl":
			installCmd = "if ! command -v kubectl >/dev/null; then curl -LO \"https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl\" && sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl && rm kubectl; fi"
		case "ansible":
			installCmd = "sudo apt-get install -y ansible"
		case "terraform":
			installCmd = "if ! command -v terraform >/dev/null; then sudo apt-get install -y software-properties-common && curl -fsSL https://apt.releases.hashicorp.com/gpg | sudo apt-key add - && sudo apt-add-repository \"deb [arch=amd64] https://apt.releases.hashicorp.com $(lsb_release -cs) main\" && sudo apt-get update && sudo apt-get install -y terraform; fi"
		case "aws-cli":
			installCmd = "if ! command -v aws >/dev/null; then curl \"https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip\" -o \"awscliv2.zip\" && unzip -q awscliv2.zip && sudo ./aws/install && rm -rf aws awscliv2.zip; fi"
		case "gh-cli":
			installCmd = "if ! command -v gh >/dev/null; then curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg | sudo dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg && echo \"deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main\" | sudo tee /etc/apt/sources.list.d/github-cli.list > /dev/null && sudo apt update && sudo apt install -y gh; fi"
		case "redis-cli":
			installCmd = "sudo apt-get install -y redis-tools"
		default:
			log.Warn().Str("tool", tool).Msg("Unknown tool, attempting generic apt-get install")
			installCmd = fmt.Sprintf("sudo apt-get install -y %s || echo 'Failed to install %s'", tool, tool)
		}

		// Execute install command
		log.Info().Str("vm", vmName).Str("tool", tool).Msg("Installing tool")
		result, err := t.executor.ExecuteCommand(ctx, installCmd, execCtx, nil)

		if err != nil {
			log.Error().Err(err).Str("tool", tool).Msg("Failed to execute install command")
			results[tool] = map[string]interface{}{
				"status":  "error",
				"message": err.Error(),
			}
			continue
		}

		if result.ExitCode != 0 {
			log.Error().
				Str("tool", tool).
				Int("exit_code", result.ExitCode).
				Str("stderr", result.Stderr).
				Msg("Tool installation failed")

			results[tool] = map[string]interface{}{
				"status":    "failed",
				"exit_code": result.ExitCode,
				"stderr":    result.Stderr,
			}
			continue
		}

		// Success
		log.Info().Str("tool", tool).Msg("Tool installation completed successfully")
		results[tool] = map[string]interface{}{
			"status":    "success",
			"exit_code": result.ExitCode,
		}
	}

	return map[string]interface{}{
		"vm_name": vmName,
		"tools":   results,
		"success": true,
	}, nil
}

// ConfigureShellTool implements the configure_shell tool
type ConfigureShellTool struct {
	manager  *vm.Manager
	executor *exec.Executor
}

// Name returns the tool name
func (t *ConfigureShellTool) Name() string {
	return "configure_shell"
}

// Description returns the tool description
func (t *ConfigureShellTool) Description() string {
	return "Configure shell environment in the VM"
}

// Execute performs the tool action
func (t *ConfigureShellTool) Execute(params map[string]interface{}) (interface{}, error) {
	// Validate required parameters
	vmName, ok := params["vm_name"].(string)
	if !ok || vmName == "" {
		return nil, fmt.Errorf("missing or invalid 'vm_name' parameter")
	}

	// Get shell type (defaults to bash)
	shellType := "bash"
	if shell, ok := params["shell"].(string); ok && shell != "" {
		shellType = shell
	}

	// Extract aliases
	aliases := make(map[string]string)
	if aliasesObj, ok := params["aliases"].(map[string]interface{}); ok {
		for key, val := range aliasesObj {
			if valStr, ok := val.(string); ok {
				aliases[key] = valStr
			}
		}
	}

	// Extract environment variables
	envVars := make(map[string]string)
	if envObj, ok := params["env"].(map[string]interface{}); ok {
		for key, val := range envObj {
			if valStr, ok := val.(string); ok {
				envVars[key] = valStr
			}
		}
	}

	// Extract PATH additions
	var pathAdditions []string
	if pathObj, ok := params["path"].([]interface{}); ok {
		for _, p := range pathObj {
			if pStr, ok := p.(string); ok {
				pathAdditions = append(pathAdditions, pStr)
			}
		}
	}

	// Check VM state
	state, err := t.manager.GetVMState(vmName)
	if err != nil {
		return nil, fmt.Errorf("VM '%s' not found: %w", vmName, err)
	}

	if state != vm.StateRunning {
		return nil, fmt.Errorf("VM '%s' is not running (current state: %s)", vmName, state)
	}

	// Setup execution context
	execCtx := exec.ExecutionContext{
		VMName:      vmName,
		WorkingDir:  "/home/vagrant",
		Environment: make(map[string]string),
		SyncBefore:  false,
		SyncAfter:   false,
	}

	// Determine config file based on shell
	configFile := "/home/vagrant/.bashrc"
	if shellType == "zsh" {
		configFile = "/home/vagrant/.zshrc"

		// Check if zsh is installed, install if not
		checkZshCmd := "command -v zsh || echo 'not-installed'"
		result, err := t.executor.ExecuteCommand(context.Background(), checkZshCmd, execCtx, nil)
		if err != nil || strings.Contains(result.Stdout, "not-installed") {
			installZshCmd := "sudo apt-get update && sudo apt-get install -y zsh"
			if _, err := t.executor.ExecuteCommand(context.Background(), installZshCmd, execCtx, nil); err != nil {
				return nil, fmt.Errorf("failed to install zsh: %w", err)
			}

			// Set zsh as default shell
			chshCmd := "sudo chsh -s $(which zsh) vagrant"
			if _, err := t.executor.ExecuteCommand(context.Background(), chshCmd, execCtx, nil); err != nil {
				return nil, fmt.Errorf("failed to set zsh as default shell: %w", err)
			}
		}
	}

	// Create a script to update the shell configuration
	script := fmt.Sprintf(`
# Create/update shell configuration
touch %s

# Create marker lines
ALIAS_START="# START VAGRANT MCP ALIASES"
ALIAS_END="# END VAGRANT MCP ALIASES"
ENV_START="# START VAGRANT MCP ENV VARS"
ENV_END="# END VAGRANT MCP ENV VARS"
PATH_START="# START VAGRANT MCP PATH"
PATH_END="# END VAGRANT MCP PATH"

# Remove any existing config sections
sed -i "/$ALIAS_START/,/$ALIAS_END/d" %s
sed -i "/$ENV_START/,/$ENV_END/d" %s
sed -i "/$PATH_START/,/$PATH_END/d" %s

# Add aliases
echo "$ALIAS_START" >> %s
`, configFile, configFile, configFile, configFile, configFile)

	// Add aliases
	for alias, command := range aliases {
		script += fmt.Sprintf("echo 'alias %s=\"%s\"' >> %s\n", alias, command, configFile)
	}
	script += fmt.Sprintf("echo \"$ALIAS_END\" >> %s\n\n", configFile)

	// Add environment variables
	script += fmt.Sprintf("echo \"$ENV_START\" >> %s\n", configFile)
	for name, value := range envVars {
		script += fmt.Sprintf("echo 'export %s=\"%s\"' >> %s\n", name, value, configFile)
	}
	script += fmt.Sprintf("echo \"$ENV_END\" >> %s\n\n", configFile)

	// Add PATH entries
	if len(pathAdditions) > 0 {
		script += fmt.Sprintf("echo \"$PATH_START\" >> %s\n", configFile)
		for _, path := range pathAdditions {
			script += fmt.Sprintf("echo 'export PATH=\"%s:$PATH\"' >> %s\n", path, configFile)
		}
		script += fmt.Sprintf("echo \"$PATH_END\" >> %s\n", configFile)
	}

	// Execute script to update shell configuration
	result, err := t.executor.ExecuteCommand(context.Background(), script, execCtx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to execute shell configuration script: %w", err)
	}

	if result.ExitCode != 0 {
		return nil, fmt.Errorf("shell configuration failed: %s", result.Stderr)
	}

	return map[string]interface{}{
		"vm_name":     vmName,
		"shell_type":  shellType,
		"config_file": configFile,
		"configured": map[string]interface{}{
			"aliases":    len(aliases),
			"env_vars":   len(envVars),
			"path_added": len(pathAdditions),
		},
		"success": true,
	}, nil
}
