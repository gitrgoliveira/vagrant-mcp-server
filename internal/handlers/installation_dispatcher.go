// Package handlers provides unified installation logic
package handlers

import (
	"fmt"
)

// InstallationDispatcher handles different runtime and tool installations
type InstallationDispatcher struct {
	runtimeHandlers map[string]func(vmName string, options map[string]interface{}) ([]string, error)
	toolHandlers    map[string]func(vmName string, options map[string]interface{}) ([]string, error)
}

// NewInstallationDispatcher creates a new installation dispatcher
func NewInstallationDispatcher() *InstallationDispatcher {
	dispatcher := &InstallationDispatcher{
		runtimeHandlers: make(map[string]func(vmName string, options map[string]interface{}) ([]string, error)),
		toolHandlers:    make(map[string]func(vmName string, options map[string]interface{}) ([]string, error)),
	}

	// Register default runtime handlers
	dispatcher.registerDefaultRuntimeHandlers()
	dispatcher.registerDefaultToolHandlers()

	return dispatcher
}

// registerDefaultRuntimeHandlers registers the default runtime installation handlers
func (d *InstallationDispatcher) registerDefaultRuntimeHandlers() {
	d.runtimeHandlers["node"] = d.installNodeRuntime
	d.runtimeHandlers["python"] = d.installPythonRuntime
	d.runtimeHandlers["ruby"] = d.installRubyRuntime
	d.runtimeHandlers["go"] = d.installGoRuntime
	d.runtimeHandlers["rust"] = d.installRustRuntime
	d.runtimeHandlers["java"] = d.installJavaRuntime
}

// registerDefaultToolHandlers registers the default tool installation handlers
func (d *InstallationDispatcher) registerDefaultToolHandlers() {
	d.toolHandlers["docker"] = d.installDockerTool
	d.toolHandlers["git"] = d.installGitTool
	d.toolHandlers["vim"] = d.installVimTool
	d.toolHandlers["emacs"] = d.installEmacsTool
	d.toolHandlers["curl"] = d.installCurlTool
	d.toolHandlers["wget"] = d.installWgetTool
	d.toolHandlers["htop"] = d.installHtopTool
	d.toolHandlers["tree"] = d.installTreeTool
}

// InstallRuntime installs a runtime using the appropriate handler
func (d *InstallationDispatcher) InstallRuntime(runtime, vmName string, options map[string]interface{}) ([]string, error) {
	handler, exists := d.runtimeHandlers[runtime]
	if !exists {
		return nil, fmt.Errorf("unsupported runtime: %s", runtime)
	}
	return handler(vmName, options)
}

// InstallTool installs a tool using the appropriate handler
func (d *InstallationDispatcher) InstallTool(tool, vmName string, options map[string]interface{}) ([]string, error) {
	handler, exists := d.toolHandlers[tool]
	if !exists {
		return nil, fmt.Errorf("unsupported tool: %s", tool)
	}
	return handler(vmName, options)
}

// GetSupportedRuntimes returns a list of supported runtimes
func (d *InstallationDispatcher) GetSupportedRuntimes() []string {
	runtimes := make([]string, 0, len(d.runtimeHandlers))
	for runtime := range d.runtimeHandlers {
		runtimes = append(runtimes, runtime)
	}
	return runtimes
}

// GetSupportedTools returns a list of supported tools
func (d *InstallationDispatcher) GetSupportedTools() []string {
	tools := make([]string, 0, len(d.toolHandlers))
	for tool := range d.toolHandlers {
		tools = append(tools, tool)
	}
	return tools
}

// Runtime installation handlers

func (d *InstallationDispatcher) installNodeRuntime(vmName string, options map[string]interface{}) ([]string, error) {
	version := "lts"
	if v, ok := options["version"].(string); ok {
		version = v
	}

	commands := []string{
		"curl -fsSL https://deb.nodesource.com/setup_lts.x | sudo -E bash -",
		"sudo apt-get install -y nodejs",
		"sudo npm install -g npm@latest",
	}

	if version != "lts" {
		commands = []string{
			"curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.0/install.sh | bash",
			"source ~/.bashrc",
			fmt.Sprintf("nvm install %s", version),
			fmt.Sprintf("nvm use %s", version),
		}
	}

	return commands, nil
}

func (d *InstallationDispatcher) installPythonRuntime(vmName string, options map[string]interface{}) ([]string, error) {
	version := "3.11"
	if v, ok := options["version"].(string); ok {
		version = v
	}

	commands := []string{
		"sudo apt-get update",
		"sudo apt-get install -y software-properties-common",
		"sudo add-apt-repository -y ppa:deadsnakes/ppa",
		"sudo apt-get update",
		fmt.Sprintf("sudo apt-get install -y python%s python%s-venv python%s-pip", version, version, version),
		fmt.Sprintf("sudo ln -sf /usr/bin/python%s /usr/bin/python3", version),
		"python3 -m pip install --upgrade pip",
	}

	return commands, nil
}

func (d *InstallationDispatcher) installRubyRuntime(vmName string, options map[string]interface{}) ([]string, error) {
	version := "3.2"
	if v, ok := options["version"].(string); ok {
		version = v
	}

	commands := []string{
		"sudo apt-get update",
		"sudo apt-get install -y curl gpg",
		"curl -sSL https://rvm.io/mpapis.asc | gpg --import -",
		"curl -sSL https://rvm.io/pkuczynski.asc | gpg --import -",
		"curl -sSL https://get.rvm.io | bash -s stable",
		"source ~/.rvm/scripts/rvm",
		fmt.Sprintf("rvm install ruby-%s", version),
		fmt.Sprintf("rvm use ruby-%s --default", version),
		"gem update --system",
	}

	return commands, nil
}

func (d *InstallationDispatcher) installGoRuntime(vmName string, options map[string]interface{}) ([]string, error) {
	version := "1.21"
	if v, ok := options["version"].(string); ok {
		version = v
	}

	commands := []string{
		fmt.Sprintf("wget https://go.dev/dl/go%s.linux-amd64.tar.gz", version),
		"sudo rm -rf /usr/local/go",
		fmt.Sprintf("sudo tar -C /usr/local -xzf go%s.linux-amd64.tar.gz", version),
		"echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc",
		"source ~/.bashrc",
		fmt.Sprintf("rm go%s.linux-amd64.tar.gz", version),
	}

	return commands, nil
}

func (d *InstallationDispatcher) installRustRuntime(vmName string, options map[string]interface{}) ([]string, error) {
	commands := []string{
		"curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y",
		"source ~/.cargo/env",
		"rustup update",
	}

	return commands, nil
}

func (d *InstallationDispatcher) installJavaRuntime(vmName string, options map[string]interface{}) ([]string, error) {
	version := "17"
	if v, ok := options["version"].(string); ok {
		version = v
	}

	commands := []string{
		"sudo apt-get update",
		fmt.Sprintf("sudo apt-get install -y openjdk-%s-jdk", version),
		fmt.Sprintf("sudo update-alternatives --set java /usr/lib/jvm/java-%s-openjdk-amd64/bin/java", version),
	}

	return commands, nil
}

// Tool installation handlers

func (d *InstallationDispatcher) installDockerTool(vmName string, options map[string]interface{}) ([]string, error) {
	commands := []string{
		"sudo apt-get update",
		"sudo apt-get install -y ca-certificates curl gnupg",
		"sudo install -m 0755 -d /etc/apt/keyrings",
		"curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg",
		"sudo chmod a+r /etc/apt/keyrings/docker.gpg",
		"echo \"deb [arch=\"$(dpkg --print-architecture)\" signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \"$(. /etc/os-release && echo \"$VERSION_CODENAME\")\" stable\" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null",
		"sudo apt-get update",
		"sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin",
		"sudo usermod -aG docker vagrant",
	}

	return commands, nil
}

func (d *InstallationDispatcher) installGitTool(vmName string, options map[string]interface{}) ([]string, error) {
	commands := []string{
		"sudo apt-get update",
		"sudo apt-get install -y git",
	}

	return commands, nil
}

func (d *InstallationDispatcher) installVimTool(vmName string, options map[string]interface{}) ([]string, error) {
	commands := []string{
		"sudo apt-get update",
		"sudo apt-get install -y vim",
	}

	return commands, nil
}

func (d *InstallationDispatcher) installEmacsTool(vmName string, options map[string]interface{}) ([]string, error) {
	commands := []string{
		"sudo apt-get update",
		"sudo apt-get install -y emacs",
	}

	return commands, nil
}

func (d *InstallationDispatcher) installCurlTool(vmName string, options map[string]interface{}) ([]string, error) {
	commands := []string{
		"sudo apt-get update",
		"sudo apt-get install -y curl",
	}

	return commands, nil
}

func (d *InstallationDispatcher) installWgetTool(vmName string, options map[string]interface{}) ([]string, error) {
	commands := []string{
		"sudo apt-get update",
		"sudo apt-get install -y wget",
	}

	return commands, nil
}

func (d *InstallationDispatcher) installHtopTool(vmName string, options map[string]interface{}) ([]string, error) {
	commands := []string{
		"sudo apt-get update",
		"sudo apt-get install -y htop",
	}

	return commands, nil
}

func (d *InstallationDispatcher) installTreeTool(vmName string, options map[string]interface{}) ([]string, error) {
	commands := []string{
		"sudo apt-get update",
		"sudo apt-get install -y tree",
	}

	return commands, nil
}

// Global installation dispatcher instance
var GlobalInstallationDispatcher = NewInstallationDispatcher()
