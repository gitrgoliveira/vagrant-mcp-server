# Vagrant MCP Server (Go Implementation)

> [!CAUTION]
> This is work in progress and not yet ready for use.

A Model Context Protocol (MCP) Server implementation for HashiCorp Vagrant that provides AI agents with the ability to create and manage development VMs with synchronized filesystems and seamless command execution capabilities.

> **Note:** This server must be run directly on the host where Vagrant and your virtualization provider (e.g., VirtualBox, libvirt) are installed. Running inside Docker is not supported, as the server needs direct access to the Vagrant CLI, virtualization drivers, and your project files.

## Features

- **Development VM Management:** Create, ensure, and destroy development VMs
- **Synchronized Command Execution:** Execute commands inside the VM with guaranteed file synchronization
- **File System Synchronization:** Configure sync methods, monitor sync status, and resolve conflicts
- **Development Environment Setup:** Install language runtimes, tools, and dependencies

## Working Example Prompts

Here are example prompts you can use with AI assistants to demonstrate the server's capabilities:

### VM Management Examples
1. **"Create a development VM for this Node.js project with 4GB RAM and sync the current directory"**
2. **"Set up a new development environment called 'myapp-dev' for this Python project with automatic file synchronization"**
3. **"Spin up a development VM with the default Ubuntu box and ensure port 3000 is forwarded to the host"**

### Command Execution Examples
1. **"Run 'npm install' in the development VM and make sure all files are synced before and after"**
2. **"Execute the test suite in the VM environment and show me the results"**
3. **"Run the development server in the background inside the VM and forward port 3000"**

### Environment Setup Examples
1. **"Install Node.js version 18 and npm in the development VM"**
2. **"Set up a Python development environment with pip and virtualenv"**
3. **"Install Docker and docker-compose in the VM for containerized development"**

### File Synchronization Examples
1. **"Sync all my local changes to the development VM"**
2. **"Upload the dist/ folder to the VM and extract it"**
3. **"Check the sync status and resolve any conflicts by keeping the host version"**

## System Requirements

- **Vagrant CLI:** The Vagrant command line interface must be installed and available in your PATH
- **Virtualization Provider:** A supported virtualization provider (e.g., VirtualBox, VMware, Hyper-V, or libvirt)
- **Go 1.18+:** Required for building from source

You can verify that Vagrant is installed correctly by running:

```bash
vagrant --version
```

## Installation

### Download Pre-built Binary (Recommended)

Download the latest release for your platform from the [Releases page](https://github.com/vagrant-mcp/server/releases):

| Platform | Architecture | Download |
|----------|--------------|----------|
| Linux | x86_64 | [vagrant-mcp-server-linux-amd64](https://github.com/vagrant-mcp/server/releases/latest/download/vagrant-mcp-server-linux-amd64) |
| Linux | ARM64 | [vagrant-mcp-server-linux-arm64](https://github.com/vagrant-mcp/server/releases/latest/download/vagrant-mcp-server-linux-arm64) |
| macOS | Intel | [vagrant-mcp-server-darwin-amd64](https://github.com/vagrant-mcp/server/releases/latest/download/vagrant-mcp-server-darwin-amd64) |
| macOS | Apple Silicon | [vagrant-mcp-server-darwin-arm64](https://github.com/vagrant-mcp/server/releases/latest/download/vagrant-mcp-server-darwin-arm64) |
| Windows | x86_64 | [vagrant-mcp-server-windows-amd64.exe](https://github.com/vagrant-mcp/server/releases/latest/download/vagrant-mcp-server-windows-amd64.exe) |

**Installation steps:**
```bash
# Download the appropriate binary for your platform
curl -L -o vagrant-mcp-server https://github.com/vagrant-mcp/server/releases/latest/download/vagrant-mcp-server-linux-amd64

# Make it executable (Linux/macOS)
chmod +x vagrant-mcp-server

# Move to your PATH (optional)
sudo mv vagrant-mcp-server /usr/local/bin/

# Verify installation
vagrant-mcp-server -version
```

**Verify integrity:**
```bash
# Download checksums file
curl -L -O https://github.com/vagrant-mcp/server/releases/latest/download/checksums.txt

# Verify your binary
sha256sum -c checksums.txt --ignore-missing
```

### Build from Source

```bash
# Clone the repository
git clone https://github.com/vagrant-mcp/server.git vagrant-mcp-server
cd vagrant-mcp-server

# Build the server
make build

# Start the server (stdio mode by default)
./bin/vagrant-mcp-server
```

## Configuration

The server can be configured using environment variables:

- `MCP_TRANSPORT` - Transport type to use (stdio or sse, default: stdio)
- `MCP_PORT` - Port to use for SSE transport (default: 8080)
- `LOG_LEVEL` - Logging level (debug, info, warn, error, default: info)
- `VSCODE_MCP` - Set to "true" when running from VS Code 
- `VM_BASE_DIR` - Base directory for VM files (default: ~/.vagrant-mcp-server/vms)

## VS Code Integration

The Vagrant MCP Server can be used with Visual Studio Code to allow AI assistants to create and manage development VMs directly from your editor.

### Prerequisites

1. Make sure you've built the server as described in the Installation section
2. Ensure Vagrant CLI and a virtualization provider are installed on your system
3. Install Visual Studio Code

### Steps to Install in VS Code

1. **Configure VS Code Settings**

   Add the following to your VS Code `settings.json` or `mcp.json` file (Command Palette > Preferences: Open User Settings (JSON)):

   ```json
   "mcp": {
     "inputs": [],
     "servers": {
       "vagrant-mcp-server": {
         "type": "stdio",
         "command": "/path/to/vagrant-mcp-server/bin/vagrant-mcp-server",
         "description": "Manage Vagrant development VMs",
         "env": {
           "VSCODE_MCP": "true"
         }
       }
     }
   }
   ```

   Replace `/path/to/vagrant-mcp-server` with the absolute path to your built server binary.

2. **Restart VS Code**

   Restart VS Code to apply the new MCP connection settings.

3. **Using the MCP Server with VS Code AI Features**

   Now AI assistants in VS Code will be able to:
   - Create development VMs for your projects
   - Execute commands inside VMs
   - Synchronize files between your local system and VMs
   - Install development tools and languages
   - Manage VM lifecycle (start/stop/destroy)

4. **Example Usage with AI Assistant**

   You can ask the AI assistant questions like:
   - "Create a development VM for this project"
   - "Run the tests in a VM"
   - "Install Node.js in the development VM"
   - "Execute the application in the VM and forward port 3000"

### Security Considerations

When using the Vagrant MCP Server with VS Code:

1. **Permissions**: The server runs with your user privileges and can:
   - Create, modify, and delete files in your project directories
   - Execute commands on your system through Vagrant
   - Manage virtual machines on your behalf

2. **Access Control**: 
   - Only install this MCP Server on systems where you trust the AI assistant with the above permissions
   - The server provides no authentication mechanisms itself - it relies on VS Code's security model
   - Do not expose the SSE transport on public networks (if using SSE mode)

3. **Data Handling**:
   - Be cautious when syncing sensitive data between your host and VM
   - Consider using sync exclusion patterns for confidential files

4. **Resource Management**:
   - Monitor resource usage of created VMs to prevent excessive consumption
   - Always destroy VMs when they are no longer needed

### Troubleshooting VS Code Integration

If you encounter issues with the VS Code integration:

1. **Check Server Logs**:
   - Set the `LOG_LEVEL` environment variable to `debug` in your settings.json:
     ```json
    "mcp": {
        "inputs": [],
        "servers": {
          "vagrant-mcp-server": {
            "type": "stdio",
            "command": "/path/to/vagrant-mcp-server/bin/vagrant-mcp-server",
            "description": "Manage Vagrant development VMs",
            "env": {
                "LOG_LEVEL": "debug",
                "VSCODE_MCP": "true"
            }
          }
        }
      }
     ```

2. **Verify Vagrant Installation**:
   - Run `vagrant --version` in your terminal to confirm Vagrant is properly installed
   - Ensure your virtualization provider (VirtualBox, etc.) is working correctly

3. **Check Paths**:
   - Make sure the path to the server binary is correct in your settings.json
   - Verify that project paths used with the server are valid and accessible

4. **Common Issues**:
   - "Vagrant is not installed" error: Add Vagrant to your PATH or specify the full path in your environment
   - "Failed to create VM": Check your virtualization provider is running and properly configured
   - Connection issues: Restart VS Code and check that the MCP server is correctly configured

## Usage

### MCP Tools

#### Development VM Management

- `create_dev_vm`: Create and configure a development VM
  - Parameters:
    - `name` (string): Name for the development VM
    - `project_path` (string): Path to the project directory to sync
    - `cpu` (number, optional): Number of CPU cores (default: 2)
    - `memory` (number, optional): Amount of memory in MB (default: 2048)
    - `box` (string, optional): Vagrant box to use (default: "ubuntu/focal64")
    - `sync_type` (string, optional): Sync type to use (default: "rsync")
  - **Example Prompts:**
    - "Create a development VM named 'webapp-dev' for the current project directory"
    - "Set up a VM called 'api-server' with 4GB RAM for the project in /home/user/myapi"
    - "Create a high-performance VM with 8 cores and 8GB RAM for the machine learning project"

- `ensure_dev_vm`: Ensure development VM is running
  - Parameters:
    - `name` (string): Name of the VM to ensure
  - **Example Prompts:**
    - "Make sure the 'webapp-dev' VM is running and ready"
    - "Start the development VM if it's not already running"
    - "Ensure my project VM is up and available for development"

- `destroy_dev_vm`: Destroy a development VM
  - Parameters:
    - `name` (string): Name of the VM to destroy
  - **Example Prompts:**
    - "Clean up and destroy the 'old-project' development VM"
    - "Remove the VM to free up disk space"
    - "Permanently delete the VM and all its resources"

#### Command Execution

- `exec_in_vm`: Execute commands inside a VM with pre/post file sync
  - Parameters:
    - `vm_name` (string): Name of the VM
    - `command` (string): Command to execute
    - `working_dir` (string, optional): Working directory
    - `env` (object, optional): Environment variables
  - **Example Prompts:**
    - "Run 'npm test' in the development VM and sync files before and after"
    - "Execute the build script in the VM with the latest code changes"
    - "Run the database migration command in the VM environment"

- `exec_with_sync`: Execute commands with explicit before/after sync
  - Parameters:
    - `vm_name` (string): Name of the VM
    - `command` (string): Command to execute
    - `sync_before` (boolean): Sync files before execution
    - `sync_after` (boolean): Sync files after execution
    - `working_dir` (string, optional): Working directory
    - `env` (object, optional): Environment variables
  - **Example Prompts:**
    - "Run the tests without syncing files first, but sync the results back"
    - "Execute the linter and sync only the fixed files back to the host"
    - "Run the development server without any file synchronization"

- `run_background_task`: Run a command in the VM as a background task
  - Parameters:
    - `vm_name` (string): Name of the VM
    - `command` (string): Command to execute
    - `sync_before` (boolean): Sync files before execution
    - `working_dir` (string, optional): Working directory
  - **Example Prompts:**
    - "Start the development server in the background in the VM"
    - "Run the file watcher process in the VM background"
    - "Start the database server in the VM and keep it running"

- `sync_to_vm`: Manually sync from host to VM
  - Parameters:
    - `vm_name` (string): Name of the VM
    - `path` (string, optional): Path to sync (default: all paths)
  - **Example Prompts:**
    - "Sync my latest code changes to the development VM"
    - "Upload the new configuration files to the VM"
    - "Push all my uncommitted changes to the VM environment"

- `sync_from_vm`: Manually sync from VM to host
  - Parameters:
    - `vm_name` (string): Name of the VM
    - `path` (string, optional): Path to sync (default: all paths)
  - **Example Prompts:**
    - "Download the generated build artifacts from the VM"
    - "Sync the log files from the VM to my local machine"
    - "Pull any changes made in the VM back to my host"
    
- `upload_to_vm`: Upload files from host to VM
  - Parameters:
    - `vm_name` (string): Name of the VM
    - `source` (string): Source file or directory path on host
    - `destination` (string): Destination path on VM
    - `compress` (boolean, optional): Whether to compress the file before upload
    - `compression_type` (string, optional): Compression type to use (tgz or zip)
  - **Example Prompts:**
    - "Upload the data files to /tmp/data in the VM"
    - "Copy the backup.tar.gz file to the VM's home directory"
    - "Upload and extract the dependencies folder to the VM"

#### Environment Setup

- `setup_dev_environment`: Install language runtimes and tools
  - Parameters:
    - `vm_name` (string): Name of the VM
    - `runtimes` (array): Language runtimes to install (e.g., 'node', 'python', 'go')
    - `tools` (array, optional): Additional tools to install
  - **Example Prompts:**
    - "Install Node.js and Python in the development VM"
    - "Set up a Go development environment with all necessary tools"
    - "Install Ruby and Rails for web development"

- `install_dev_tools`: Install specific development tools
  - Parameters:
    - `vm_name` (string): Name of the VM
    - `tools` (array): List of tools to install
  - **Example Prompts:**
    - "Install Docker and docker-compose in the VM"
    - "Add git, vim, and curl to the development environment"
    - "Install the latest version of PostgreSQL and Redis"

- `configure_shell`: Configure shell environment
  - Parameters:
    - `vm_name` (string): Name of the VM
    - `shell_type` (string, optional): Shell to configure (bash, zsh, etc.)
    - `env_vars` (array, optional): Environment variables to set
    - `aliases` (array, optional): Shell aliases to configure
  - **Example Prompts:**
    - "Set up zsh with development aliases in the VM"
    - "Configure bash with custom environment variables"
    - "Add useful aliases for common development commands"

#### Synchronization

- `configure_sync`: Configure sync method and options
  - Parameters:
    - `vm_name` (string): Name of the VM
    - `sync_type` (string): Sync type (rsync, nfs, smb, virtualbox)
    - `exclude_patterns` (array, optional): Patterns to exclude
    - `guest_path` (string, optional): Guest path to sync
    - `host_path` (string, optional): Host path to sync
  - **Example Prompts:**
    - "Configure NFS sync for faster file operations"
    - "Set up rsync with exclusions for node_modules and .git folders"
    - "Switch to SMB sync for better Windows host compatibility"

- `sync_status`: Check sync status
  - Parameters:
    - `vm_name` (string): Name of the VM
  - **Example Prompts:**
    - "Check if all files are synchronized between host and VM"
    - "Show me the current sync status and any pending changes"
    - "Verify that the file synchronization is working properly"

- `resolve_sync_conflicts`: Resolve sync conflicts
  - Parameters:
    - `vm_name` (string): Name of the VM
    - `path` (string): Path of the conflicted file
    - `resolution` (string): Resolution method ('use_host', 'use_vm', 'merge', 'keep_both')
  - **Example Prompts:**
    - "Resolve sync conflicts by keeping the host version"
    - "Fix sync conflicts in the config file by using the VM version"
    - "Merge the conflicting files and keep both versions"

- `search_code`: Search code semantically in the VM
  - Parameters:
    - `vm_name` (string): Name of the VM
    - `query` (string): Search query
    - `search_type` (string, optional): Type of search ('semantic', 'exact', 'fuzzy')
    - `max_results` (number, optional): Maximum results to return
    - `case_sensitive` (boolean, optional): Case sensitive search
  - **Example Prompts:**
    - "Find all functions that handle user authentication"
    - "Search for database connection code in the VM"
    - "Look for TODO comments across all project files"

- `get_vm_status`: Get status of development VMs
  - Parameters:
    - `name` (string, optional): Name of specific VM to check
  - **Example Prompts:**
    - "Show me the status of all development VMs"
    - "Check if the 'webapp-dev' VM is running and healthy"
    - "Get resource usage statistics for the development VM"

## Privacy Policy

**Data Collection:** The Vagrant MCP Server does not collect, store, or transmit any personal data or project information to external servers. All operations are performed locally on your development machine.

**Local Data Handling:**
- Project files are synchronized only between your host machine and local VMs
- Commands are executed locally within your development environment
- No telemetry, analytics, or usage data is collected
- No network connections are made to external services (except for downloading Vagrant boxes as configured by you)

**VM Data:** Virtual machines created by this server contain only the data you explicitly provide. VMs are stored locally on your machine and are not shared or transmitted anywhere.

**Logging:** The server generates local logs for debugging purposes. These logs remain on your machine and are not transmitted externally.

## Security

### Security Considerations

This MCP server provides powerful capabilities that require careful consideration:

**Permissions and Access:**
- The server runs with your user privileges and can create, modify, and delete files in your project directories  
- Commands executed through the server run in VMs but can affect your host system through file synchronization
- The server can manage virtual machines on your behalf, which includes resource allocation and network configuration

**Network Security:**
- VMs created by this server may forward ports to your host machine
- Ensure firewall rules are appropriate for your development needs
- Do not expose forwarded ports on public networks without proper security measures

**File System Security:**
- Be cautious when syncing sensitive data between host and VM
- Use sync exclusion patterns for confidential files (`.env`, private keys, etc.)
- Regularly review sync configurations to prevent unintended data exposure

**Resource Security:**
- Monitor resource usage of created VMs to prevent resource exhaustion
- Destroy VMs when no longer needed to free resources
- Set appropriate memory and CPU limits for VMs


## Development

### Prerequisites

1. **Go 1.18 or higher**
2. **Vagrant CLI** - Required for both running the server and tests
   - All tests use the real Vagrant CLI for validation
   - Tests requiring Vagrant will be skipped if Vagrant is not installed
   - Some tests that require a full VM environment may be skipped in CI
3. **A supported virtualization provider** - VirtualBox is recommended for development

### Common Tasks

```bash
# Format code
go fmt ./...

# Lint code
make lint

# Run unit tests (requires Vagrant CLI installed)
make test

# Run integration tests (requires Vagrant CLI and a virtualization provider)
make test-integration

# Run VM start tests (actually starts VMs - very slow)
make test-vm-start

# See all test options
make help-test

# Security checks
make sec

# Build all release binaries
git tag v1.0.0  # or your version
git push --tags
make release
```

## Developer Scripts

The `dev-scripts/` directory contains optional utilities for development and manual testing:

- `test_script.sh`: Example shell script for testing the `exec_in_vm` tool. Can be uploaded and executed in a VM to verify command execution and environment setup.
- `test_mcp.py`: Python utility for sending JSON-RPC requests to the MCP server for manual or ad-hoc testing. Useful for developers who want to interact with the server outside of the normal client workflow.

These scripts are not required for normal operation or production use, but may be helpful for contributors and advanced users.

## Testing and Validation

Before using in production, we recommend testing the server with the MCP Inspector:

```bash
# Install the MCP Inspector (if not already installed)
npm install -g @modelcontextprotocol/inspector

# Test the server
mcp-inspector /path/to/vagrant-mcp-server/bin/vagrant-mcp-server
```

## Developer Scripts

The `dev-scripts/` directory contains optional utilities for development and manual testing:

- `test_script.sh`: Example shell script for testing the `exec_in_vm` tool. Can be uploaded and executed in a VM to verify command execution and environment setup.
- `test_mcp.py`: Python utility for sending JSON-RPC requests to the MCP server for manual or ad-hoc testing. Useful for developers who want to interact with the server outside of the normal client workflow.

These scripts are not required for normal operation or production use, but may be helpful for contributors and advanced users.

### Compatibility Testing

This server has been tested and validated with:
- **Claude.ai** - Full compatibility with web interface
- **Claude for Desktop** - Complete VS Code integration support  
- **MCP Connector** - Standard MCP protocol compliance
- **Vagrant 2.3+** - All supported Vagrant versions
- **VirtualBox, VMware, Hyper-V, libvirt** - Major virtualization providers

### VM Cleanup

During testing and development, VMs may occasionally not be cleaned up properly. We provide several mechanisms to handle this:

#### Automatic Cleanup
Tests automatically clean up VMs using a robust cleanup process that:
- Attempts normal VM stop and destroy operations
- Falls back to force destroy using Vagrant global commands
- Logs all cleanup activities for debugging

#### Manual Cleanup
If you notice orphaned test VMs, you can clean them up manually:

```bash
# Check for any running VMs
vagrant global-status

# Destroy a specific VM by ID
vagrant destroy VM_ID --force
```

There is currently no bundled cleanup script. Use the above Vagrant commands for manual cleanup.

#### Integration Test Configuration
Long-running integration tests (that actually create VMs) are gated behind an environment variable:

```bash
# Run unit tests only (default, no VMs created)
make test

# Run integration tests (creates real VMs)
make test-integration
```

This prevents accidental VM creation during normal development while allowing full integration testing when needed.

### Release Process

Our automated release process ensures quality and reliability:

1. **Automated Testing** - Comprehensive test suite runs on every commit
2. **Security Scanning** - Code is scanned for security vulnerabilities  
3. **Integration Testing** - Real Vagrant environments are tested
4. **Documentation Review** - All documentation is verified for accuracy
5. **MCP Protocol Compliance** - Validated against official MCP specifications

**Creating a Release:**
Releases are automatically created when a git tag is pushed:

```bash
# Create and push a new version tag
git tag v1.0.0
git push origin v1.0.0
```

This triggers a GitHub Actions workflow that:
- Validates the version tag format
- Runs full test suite including integration tests
- Builds binaries for all supported platforms (Linux, macOS, Windows)
- Generates SHA256 checksums for all binaries
- Creates a GitHub Release with all assets
- Includes detailed release notes with changelog

**Version Management:**
- Version numbers are automatically extracted from git tags
- No hardcoded versions in source code
- Build-time injection of version, commit, and build metadata
- Supports semantic versioning (e.g., v1.0.0, v1.0.0-beta.1)

## Contact and Support

**Project Maintainer:** Vagrant MCP Server Team  
**Email:** support@vagrant-mcp-server.dev  
**Repository:** [https://github.com/vagrant-mcp/server](https://github.com/vagrant-mcp/server)  
**Issues:** [https://github.com/vagrant-mcp/server/issues](https://github.com/vagrant-mcp/server/issues)  

**Response Times:**
- Security vulnerabilities: Within 24 hours
- Bug reports: Within 72 hours  
- Feature requests: Within 1 week

**Maintenance Commitment:** This project is actively maintained with regular updates, security patches, and feature enhancements. We commit to supporting the latest stable version of Vagrant and major virtualization providers.

## License

This project is licensed under the Mozilla Public License 2.0 (MPL-2.0).

Copyright (c) 2025 Ricardo Oliveira

This Source Code Form is subject to the terms of the Mozilla Public License, v. 2.0. If a copy of the MPL was not distributed with this file, You can obtain one at http://mozilla.org/MPL/2.0/.
