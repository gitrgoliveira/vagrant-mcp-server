# CoPilot Agent Prompt: Create Development VM MCP Server with Vagrant

Create a complete Model Context Protocol (MCP) Server implementation for HashiCorp Vagrant that provides AI agents with the ability to create and manage development VMs with synchronized filesystems and seamless command execution capabilities.

## Requirements

### Core MCP Server Structure
- Implement using Go with proper MCP protocol implementation
- Follow MCP specification for server initialization, capabilities, and communication
- Include proper error handling, logging, and validation
- Support both stdio and SSE transport protocols
- Use Go's native JSON-RPC 2.0 capabilities for MCP communication

### Reference Documentation
Before implementation, review these key documentation sections:

**MCP Protocol Specification:**
- #fetch https://spec.modelcontextprotocol.io/specification/ - Core MCP specification
- #fetch https://spec.modelcontextprotocol.io/specification/server/ - MCP server implementation guide
- #fetch https://spec.modelcontextprotocol.io/specification/basic/tools/ - MCP tools specification
- #fetch https://spec.modelcontextprotocol.io/specification/basic/resources/ - MCP resources specification
- #fetch https://spec.modelcontextprotocol.io/specification/basic/prompts/ - MCP prompts specification
- #fetch https://spec.modelcontextprotocol.io/specification/transports/ - MCP transport protocols

**Vagrant Documentation:**
- #fetch https://developer.hashicorp.com/vagrant/docs/vagrantfile - Vagrantfile configuration
- #fetch https://developer.hashicorp.com/vagrant/docs/synced-folders - Synced folders configuration
- #fetch https://developer.hashicorp.com/vagrant/docs/provisioning - VM provisioning methods
- #fetch https://developer.hashicorp.com/vagrant/docs/cli - Vagrant CLI commands reference
- #fetch https://developer.hashicorp.com/vagrant/docs/networking - Network and port forwarding
- #fetch https://developer.hashicorp.com/vagrant/docs/providers - Provider-specific configurations

### Core Development VM Tools
Create MCP tools specifically designed for development workflow:

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
dev-vm-mcp-server/
├── cmd/
│   └── server/
│       └── main.go           # Main server entry point
├── internal/
│   ├── mcp/                  # MCP protocol implementation
│   │   ├── server.go         # Core MCP server
│   │   ├── tools.go          # Tool implementations
│   │   ├── resources.go      # Resource implementations
│   │   └── transport.go      # Transport layer (stdio/SSE)
│   ├── vm/                   # VM management and lifecycle
│   │   ├── manager.go        # Development VM manager
│   │   ├── provisioner.go    # VM provisioning logic
│   │   └── config.go         # Vagrantfile generation
│   ├── sync/                 # File synchronization
│   │   ├── engine.go         # Core sync logic
│   │   ├── watcher.go        # File system watchers
│   │   └── resolver.go       # Sync conflict handling
│   ├── exec/                 # Command execution
│   │   ├── executor.go       # VM command execution
│   │   ├── stream.go         # Output streaming
│   │   └── context.go        # Execution context management
│   └── utils/                # Utility functions
├── pkg/
│   └── types/                # Public type definitions
├── templates/                # Vagrantfile and provisioning templates
├── go.mod
├── go.sum
└── README.md
```

## Deliverables
1. Complete Go-based MCP server implementation focused on development VM management
2. Native Go MCP protocol implementation with JSON-RPC 2.0 support
3. Comprehensive synchronized command execution system
4. Robust filesystem synchronization with conflict resolution
5. Development environment templates and provisioning scripts
6. Real-time sync monitoring and status reporting
7. Unit and integration tests for VM lifecycle and sync operations
8. Performance benchmarks for sync operations and command execution
9. Documentation including development workflow examples and Go API reference
10. Configuration templates for popular development stacks
11. Docker containerization support for the MCP server
12. Cross-platform binary compilation (Linux, macOS, Windows)

## Go-Specific Implementation Requirements
- Use Go's `context` package for proper cancellation and timeout handling
- Implement proper goroutine management for concurrent operations
- Use channels for communication between sync, exec, and VM management components
- Leverage Go's `os/exec` package for Vagrant CLI interactions
- Use `fsnotify` package for filesystem watching
- Implement structured logging with `slog` package
- Use `embed` package for Vagrantfile templates
- Proper error handling with custom error types
- Configuration management with environment variables and config files

## Testing Requirements
- Unit tests using Go's built-in `testing` package
- Mock Vagrant operations using interfaces and dependency injection
- Integration tests with real development workflows using `testify` suite
- Filesystem sync stress testing with large codebases
- Command execution testing with long-running processes using context cancellation
- Sync conflict simulation and resolution testing
- Performance testing and benchmarking with `testing.B`
- Multi-platform testing (Windows, macOS, Linux hosts)
- Race condition testing with `go test -race`
- Memory leak detection with runtime profiling

## Key Implementation Notes
- **MCP Protocol**: Implement JSON-RPC 2.0 over stdio transport as primary method, with SSE as secondary
- **Command Execution Flow**: Always sync files before command execution, execute in VM, then sync results back
- **Sync Optimization**: Use rsync with efficient delta transfers, exclude build artifacts and dependencies
- **Error Recovery**: Implement robust error handling for sync failures and VM connectivity issues
- **State Persistence**: Maintain VM state and sync status using JSON files or embedded database
- **Security**: Validate all file paths and command parameters to prevent security issues
- **Go Concurrency**: Use goroutines and channels for non-blocking operations, proper context cancellation
- **Cross-platform**: Handle path separators, line endings, and platform-specific Vagrant behaviors
- **Memory Management**: Efficient handling of large file operations and command output streaming

The Go-based MCP server should enable AI agents to seamlessly work with code in a development VM as if it were local, with transparent file synchronization and reliable command execution, leveraging Go's strengths in concurrency, performance, and cross-platform deployment.