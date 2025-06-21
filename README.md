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
