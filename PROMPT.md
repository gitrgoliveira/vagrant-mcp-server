# CoPilot Agent Prompt: Create Development VM MCP Server with Vagrant

Create a complete Model Context Protocol (MCP) Server implementation for HashiCorp Vagrant that provides AI agents with the ability to create and manage development VMs with synchronized filesystems and seamless command execution capabilities.

## Requirements

### Core MCP Server Structure
- Implement using Go with the MCP-go library (github.com/mark3labs/mcp-go)
- Leverage MCP-go's server lifecycle, capabilities, and communication patterns
- Include comprehensive error handling, structured logging, and validation
- Support both stdio and SSE transport protocols as provided by MCP-go
- Ensure all code compiles, tests pass, and linting succeeds (using `make all`)
- Remove any legacy code and completely implement all functions

### Reference Documentation
Before implementation, review these key documentation sections:

**MCP Protocol Specification:**
- #fetch https://github.com/mark3labs/mcp-go - Main implementation library to use
- #fetch https://github.com/mark3labs/mcp-go/blob/main/www/docs/pages/servers/index.mdx - MCP-go documentation for servers
- #fetch https://github.com/mark3labs/mcp-go/blob/main/www/docs/pages/servers/tools.mdx - MCP-go tools implementation
- #fetch https://github.com/mark3labs/mcp-go/blob/main/www/docs/pages/servers/resources.mdx - MCP-go resources implementation
- #fetch https://github.com/mark3labs/mcp-go/blob/main/www/docs/pages/servers/advanced.mdx - MCP-go hooks and middleware

**Vagrant Documentation:**
- #fetch https://developer.hashicorp.com/vagrant/docs/vagrantfile - Vagrantfile configuration
- #fetch https://developer.hashicorp.com/vagrant/docs/synced-folders - Synced folders configuration
- #fetch https://developer.hashicorp.com/vagrant/docs/provisioning - VM provisioning methods
- #fetch https://developer.hashicorp.com/vagrant/docs/cli - Vagrant CLI commands reference
- #fetch https://developer.hashicorp.com/vagrant/docs/networking - Network and port forwarding
- #fetch https://developer.hashicorp.com/vagrant/docs/providers - Provider-specific configurations

### Core Development VM Tools
Create MCP tools using MCP-go's tool definition patterns specifically designed for development workflow:

**Development VM Management:**
- `create_dev_vm` - Create and configure a development VM with:
  - Automatic Vagrantfile generation with development-optimized settings
  - Bi-directional filesystem synchronization (rsync, NFS, or SMB)
  - Port forwarding for common development ports (3000, 8000, 5432, etc.)
  - Provisioning with essential development tools
  - Reference: #fetch https://developer.hashicorp.com/vagrant/docs/vagrantfile/machine_settings
- `ensure_dev_vm` - Ensure dev VM is running, create if doesn't exist
- `destroy_dev_vm` - Clean up development VM and associated resources

**Synchronized Command Execution:**
- `exec_in_vm` - Execute commands inside the VM with guaranteed file sync
  - Pre-execution sync: Ensure VM has latest host files
  - Command execution with real-time output streaming
  - Post-execution sync: Sync any changes back to host
  - Working directory context preservation
  - Reference: #fetch https://developer.hashicorp.com/vagrant/docs/cli/ssh
- `exec_with_sync` - Execute commands with explicit before/after sync
- `sync_to_vm` - Manual sync from host to VM
- `sync_from_vm` - Manual sync from VM to host

**Development Environment Setup:**
- `setup_dev_environment` - Install language runtimes, tools, and dependencies
  - Support for Node.js, Python, Ruby, Go, Java, etc.
  - Package manager setup (npm, pip, gem, etc.)
  - Database setup (PostgreSQL, MySQL, Redis, etc.)
  - Docker installation and configuration
  - Reference: #fetch https://developer.hashicorp.com/vagrant/docs/provisioning/shell
- `install_dev_tools` - Install specific development tools
- `configure_shell` - Setup shell environment (zsh, oh-my-zsh, etc.)

**File System Synchronization:**
- `configure_sync` - Configure sync method and options
  - Support for rsync, NFS, SMB, and VirtualBox shared folders
  - Exclude patterns for node_modules, .git, build artifacts
  - Bidirectional sync with conflict resolution
  - Reference: #fetch https://developer.hashicorp.com/vagrant/docs/synced-folders/rsync
  - Reference: #fetch https://developer.hashicorp.com/vagrant/docs/synced-folders/nfs
- `sync_status` - Check sync status and any conflicts
- `resolve_sync_conflicts` - Handle sync conflicts interactively

### MCP Resources to Implement
Provide comprehensive access to development VM state and configuration:

**Development VM State:**
- `devvm://status` - Current development VM status and health
- `devvm://config` - Current VM configuration and sync settings
- `devvm://sync-status` - Real-time filesystem sync status
- `devvm://processes` - Running processes inside the VM
- `devvm://ports` - Port forwarding configuration and status

**File System Resources:**
- `devvm://files/{path}` - Access to VM file system (read-only)
- `devvm://logs/sync` - Filesystem synchronization logs
- `devvm://logs/provisioning` - VM provisioning logs
- `devvm://env` - Environment variables and PATH inside VM

**Development Tools:**
- `devvm://installed-tools` - List of installed development tools
- `devvm://services` - Status of development services (databases, etc.)
- `devvm://network` - Network configuration and connectivity

### Technical Specifications

**Development VM Configuration:**
- Generate optimized Vagrantfiles with development-focused settings
- Configure appropriate resource allocation (CPU, memory)
- Setup efficient filesystem synchronization (prefer rsync over shared folders)
- Configure port forwarding for common development services
- Install essential development packages during provisioning
- Reference: #fetch https://developer.hashicorp.com/vagrant/docs/vagrantfile/tips
- Reference: #fetch https://developer.hashicorp.com/vagrant/docs/networking/forwarded_ports

**Synchronized Command Execution:**
- Implement pre-command sync to ensure VM has latest files
- Stream command output in real-time to the MCP client
- Capture exit codes and handle command failures gracefully
- Post-command sync to bring changes back to host
- Maintain working directory context across commands
- Support interactive commands with proper TTY handling
- Reference: #fetch https://developer.hashicorp.com/vagrant/docs/cli/ssh-config

**File System Synchronization:**
- Implement intelligent sync with exclude patterns
- Handle large file operations efficiently
- Detect and resolve sync conflicts
- Support one-way and bidirectional sync modes
- Monitor filesystem changes for automatic sync triggers
- Optimize sync performance with incremental updates
- Reference: #fetch https://developer.hashicorp.com/vagrant/docs/synced-folders/basic_usage
- Reference: #fetch https://developer.hashicorp.com/vagrant/docs/synced-folders/smb

**State Management:**
- Persist VM state and configuration
- Track sync status and last sync timestamps
- Maintain command execution history
- Store environment configuration and installed tools

### Additional Features

**Intelligent Development Environment:**
- Auto-detect project type (Node.js, Python, Ruby, etc.) and configure accordingly
- Template-based VM provisioning for different tech stacks
- Automatic dependency installation based on project files (package.json, requirements.txt, etc.)
- Development service orchestration (databases, message queues, etc.)
- Reference: #fetch https://developer.hashicorp.com/vagrant/docs/provisioning/ansible
- Reference: #fetch https://developer.hashicorp.com/vagrant/docs/provisioning/docker

**Advanced Synchronization:**
- File watching for automatic sync on changes
- Selective sync based on file patterns and project needs
- Sync conflict resolution with merge strategies
- Performance optimization for large codebases
- Background sync processes with minimal performance impact
- Reference: #fetch https://developer.hashicorp.com/vagrant/docs/synced-folders/virtualbox

**Development Workflow Integration:**
- Git integration within the VM environment
- Environment variable management and secrets handling
- Development server management and process monitoring
- Hot reload and live development support
- Testing environment isolation and cleanup
- Reference: #fetch https://developer.hashicorp.com/vagrant/docs/provisioning/file
- Reference: #fetch https://developer.hashicorp.com/vagrant/docs/multi-machine

## Project Structure
```
vagrant-mcp-server/
├── cmd/
│   └── server/
│       └── main.go           # Main server entry point with MCP-go integration
├── internal/
│   ├── handlers/             # MCP tool and resource handlers
│   │   ├── vm_tools.go       # VM management tool handlers
│   │   ├── exec_tools.go     # Command execution tool handlers
│   │   ├── sync_tools.go     # Synchronization tool handlers
│   │   └── env_tools.go      # Environment setup tool handlers 
│   ├── vm/                   # VM management and lifecycle
│   │   ├── manager.go        # Development VM manager
│   │   ├── types.go          # VM type definitions
│   │   └── accessors.go      # VM state access methods
│   ├── sync/                 # File synchronization
│   │   ├── engine.go         # Core sync logic
│   │   └── errors.go         # Sync-specific error definitions
│   ├── exec/                 # Command execution
│   │   └── executor.go       # VM command execution with sync
│   └── resources/            # MCP resource implementations
│       ├── resources.go      # Resource handler implementation
│       └── resources-mcp-go.go # MCP-go specific resource adapters
├── pkg/
│   └── mcp/                  # Public MCP interface
│       ├── types.go          # Public type definitions
│       └── server.go         # Server configuration
├── templates/                # Vagrantfile and provisioning templates
├── go.mod                    # With github.com/mark3labs/mcp-go dependency
├── go.sum
└── README.md
```

## Deliverables
1. Complete MCP server implementation using MCP-go library, focused on development VM management
2. Full implementation of all required functionality with no legacy code or TODOs remaining
3. Comprehensive synchronized command execution system integrated with MCP-go tools
4. Robust filesystem synchronization with conflict resolution leveraging MCP-go resources
5. Development environment templates and provisioning scripts for Vagrant
6. Real-time sync monitoring and status reporting via MCP-go notification capabilities
7. Unit and integration tests with high coverage for VM lifecycle and sync operations
8. All tests and linting passing with `make all` command
9. Documentation including development workflow examples for MCP-go server integration
10. Configuration templates for popular development stacks
11. Docker containerization support for the MCP server
12. Cross-platform binary compilation (Linux, macOS, Windows)

## Go-Specific Implementation Requirements
- Use MCP-go's context handling and hooks system for proper lifecycle management
- Leverage MCP-go's server structure and tool handler patterns
- Implement proper goroutine management for concurrent operations
- Use channels for communication between sync, exec, and VM management components
- Leverage Go's `os/exec` package for Vagrant CLI interactions
- Use `embed` package for Vagrantfile templates
- Implement proper error handling with consistent error types and status codes
- Ensure all code passes Go linting standards (run with `make lint`)
- Complete all TODOs and implement all required functions
- Configuration management with environment variables and config files

## Testing Requirements
- Unit tests using Go's built-in `testing` package and MCP-go test utilities
- Mock Vagrant operations using interfaces and dependency injection
- Integration tests with real development workflows using `testify` suite
- Comprehensive test coverage of all major components
- Command execution testing with proper context cancellation handling
- Sync conflict simulation and resolution testing
- All tests must pass when running `make test`
- Multi-platform testing (Windows, macOS, Linux hosts)
- Race condition testing with `go test -race`
- Memory leak detection with runtime profiling

## Key Implementation Notes
- **MCP-go Integration**: Use MCP-go library for all MCP protocol handling and server lifecycle management
- **MCP Tool Definition**: Define tools using MCP-go's tool definition patterns with proper argument validation
- **Command Execution Flow**: Always sync files before command execution, execute in VM, then sync results back
- **Sync Optimization**: Use rsync with efficient delta transfers, exclude build artifacts and dependencies
- **Error Recovery**: Implement robust error handling for sync failures and VM connectivity issues using MCP-go error patterns
- **State Persistence**: Use MCP-go's session management capabilities along with local state management
- **Security**: Validate all file paths and command parameters to prevent security issues
- **Go Concurrency**: Use goroutines and channels with MCP-go's context handling for robust operations
- **Cross-platform**: Handle path separators, line endings, and platform-specific Vagrant behaviors
- **Code Quality**: Ensure all code compiles, passes tests, and meets linting standards with `make all`
- **Complete Implementation**: Remove all TODOs and fully implement all required functionality

The MCP-go based server should enable AI agents to seamlessly work with code in a development VM as if it were local, with transparent file synchronization and reliable command execution. By leveraging the MCP-go library, the implementation should be robust, maintainable, and fully compliant with the MCP specification. The final code must compile successfully, pass all tests and linting checks with `make all`, have no TODOs or legacy code, and fully implement all required functionality.