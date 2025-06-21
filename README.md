# Vagrant MCP Server (Go Implementation)

A Model Context Protocol (MCP) Server implementation for HashiCorp Vagrant that provides AI agents with the ability to create and manage development VMs with synchronized filesystems and seamless command execution capabilities.

> **Note:** This server must be run directly on the host where Vagrant and your virtualization provider (e.g., VirtualBox, libvirt) are installed. Running inside Docker is not supported, as the server needs direct access to the Vagrant CLI, virtualization drivers, and your project files.

## Features

- **Development VM Management:** Create, ensure, and destroy development VMs
- **Synchronized Command Execution:** Execute commands inside the VM with guaranteed file synchronization
- **File System Synchronization:** Configure sync methods, monitor sync status, and resolve conflicts
- **Development Environment Setup:** Install language runtimes, tools, and dependencies

## System Requirements

- **Vagrant CLI:** The Vagrant command line interface must be installed and available in your PATH
- **Virtualization Provider:** A supported virtualization provider (e.g., VirtualBox, VMware, Hyper-V, or libvirt)
- **Go 1.18+:** Required for building from source

You can verify that Vagrant is installed correctly by running:

```bash
vagrant --version
```

## Installation

```bash
# Clone the repository
git clone https://github.com/yourusername/vagrant-mcp-server.git
cd vagrant-mcp-server

# Build the server
make build

# Start the server (stdio mode by default)
./bin/vagrant-mcp-server
```

## Configuration

The server can be configured using environment variables:

- `MCP_TRANSPORT` - Transport type to use (stdio or sse, default: stdio)
- `MCP_PORT` - Port to use for SSE transport (default: 3000)

## VS Code Integration

The Vagrant MCP Server can be used with Visual Studio Code to allow AI assistants to create and manage development VMs directly from your editor.

### Prerequisites

1. Make sure you've built the server as described in the Installation section
2. Ensure Vagrant CLI and a virtualization provider are installed on your system
3. Install Visual Studio Code

### Steps to Install in VS Code

1. **Configure VS Code Settings**

   Add the following to your VS Code `settings.json` file (Command Palette > Preferences: Open User Settings (JSON)):

   ```json
   "mcp.connections": {
     "vagrant": {
       "command": "/path/to/vagrant-mcp-server/bin/vagrant-mcp-server",
       "title": "Vagrant VMs",
       "description": "Manage Vagrant development VMs",
       "transport": "stdio"
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
     "mcp.connections": {
       "vagrant": {
         "command": "/path/to/vagrant-mcp-server/bin/vagrant-mcp-server",
         "title": "Vagrant VMs",
         "description": "Manage Vagrant development VMs",
         "transport": "stdio",
         "env": {
           "LOG_LEVEL": "debug"
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

- `ensure_dev_vm`: Ensure development VM is running
  - Parameters:
    - `name` (string): Name of the VM to ensure

- `destroy_dev_vm`: Destroy a development VM
  - Parameters:
    - `name` (string): Name of the VM to destroy

#### Command Execution

- `exec_in_vm`: Execute commands inside a VM with pre/post file sync
  - Parameters:
    - `vm_name` (string): Name of the VM
    - `command` (string): Command to execute
    - `working_dir` (string, optional): Working directory
    - `env` (object, optional): Environment variables

- `exec_with_sync`: Execute commands with explicit before/after sync
  - Parameters:
    - `vm_name` (string): Name of the VM
    - `command` (string): Command to execute
    - `sync_before` (boolean): Sync files before execution
    - `sync_after` (boolean): Sync files after execution
    - `working_dir` (string, optional): Working directory
    - `env` (object, optional): Environment variables

- `sync_to_vm`: Manually sync from host to VM
  - Parameters:
    - `vm_name` (string): Name of the VM
    - `path` (string, optional): Path to sync (default: all paths)

- `sync_from_vm`: Manually sync from VM to host
  - Parameters:
    - `vm_name` (string): Name of the VM
    - `path` (string, optional): Path to sync (default: all paths)
    
- `upload_to_vm`: Upload files from host to VM
  - Parameters:
    - `vm_name` (string): Name of the VM
    - `source` (string): Source file or directory path on host
    - `destination` (string): Destination path on VM
    - `compress` (boolean, optional): Whether to compress the file before upload
    - `compression_type` (string, optional): Compression type to use (tgz or zip)

#### Environment Setup

- `setup_dev_environment`: Install language runtimes and tools
  - Parameters:
    - `vm_name` (string): Name of the VM
    - `environment` (string): Environment type (node, python, ruby, go, etc.)
    - `version` (string, optional): Version to install

- `install_dev_tools`: Install specific development tools
  - Parameters:
    - `vm_name` (string): Name of the VM
    - `tools` (array): List of tools to install

- `configure_shell`: Configure shell environment
  - Parameters:
    - `vm_name` (string): Name of the VM
    - `shell` (string): Shell to configure (bash, zsh, etc.)
    - `config` (object): Configuration options

#### Synchronization

- `configure_sync`: Configure sync method and options
  - Parameters:
    - `vm_name` (string): Name of the VM
    - `sync_type` (string): Sync type (rsync, nfs, smb, virtualbox)
    - `exclude` (array, optional): Patterns to exclude
    - `bidirectional` (boolean, optional): Enable bidirectional sync

- `sync_status`: Check sync status
  - Parameters:
    - `vm_name` (string): Name of the VM

- `resolve_sync_conflicts`: Resolve sync conflicts
  - Parameters:
    - `vm_name` (string): Name of the VM
    - `strategy` (string): Conflict resolution strategy (host, guest, manual)

### MCP Resources

- `devvm://status` - Current development VM status and health
- `devvm://config` - Current VM configuration and sync settings
- `devvm://sync-status` - Real-time filesystem sync status
- `devvm://processes` - Running processes inside the VM
- `devvm://ports` - Port forwarding configuration and status
- `devvm://files/{path}` - Access to VM file system (read-only)
- `devvm://logs/sync` - Filesystem synchronization logs
- `devvm://logs/provisioning` - VM provisioning logs
- `devvm://env` - Environment variables and PATH inside VM
- `devvm://installed-tools` - List of installed development tools
- `devvm://services` - Status of development services (databases, etc.)
- `devvm://network` - Network configuration and connectivity
- `devvm://system` - System resource usage (CPU, memory)

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
make integration

# Security checks
make sec

# Build all release binaries
git tag v1.0.0  # or your version
git push --tags
make release
```

## License

MIT
