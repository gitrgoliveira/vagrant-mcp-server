package exec

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	synctool "github.com/vagrant-mcp/server/internal/sync"
	"github.com/vagrant-mcp/server/internal/vm"
)

// CommandResult contains the result of a command execution
type CommandResult struct {
	ExitCode int     `json:"exit_code"`
	Stdout   string  `json:"stdout"`
	Stderr   string  `json:"stderr"`
	Duration float64 `json:"duration_seconds"`
}

// ExecutionContext contains the context for command execution
type ExecutionContext struct {
	VMName      string            `json:"vm_name"`
	WorkingDir  string            `json:"working_dir"`
	Environment map[string]string `json:"environment"`
	SyncBefore  bool              `json:"sync_before"`
	SyncAfter   bool              `json:"sync_after"`
}

// OutputCallback is a function called with command output
type OutputCallback func(data []byte, isStderr bool)

// Executor manages command execution in VMs
type Executor struct {
	vmManager  *vm.Manager
	syncEngine *synctool.Engine
	mu         sync.Mutex
}

// NewExecutor creates a new command executor
func NewExecutor(vmManager *vm.Manager, syncEngine *synctool.Engine) (*Executor, error) {
	return &Executor{
		vmManager:  vmManager,
		syncEngine: syncEngine,
	}, nil
}

// ExecuteCommand executes a command in a VM with the given context
func (e *Executor) ExecuteCommand(ctx context.Context, command string, execCtx ExecutionContext, callback OutputCallback) (*CommandResult, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// SAFEGUARD: Prevent execution on host or without VM context
	if execCtx.VMName == "" || strings.ToLower(execCtx.VMName) == "host" {
		errMsg := "SECURITY VIOLATION: Attempted to execute a shell command outside of a VM context. All commands must target a Vagrant VM."
		log.Error().Msg(errMsg)
		return nil, fmt.Errorf("%s", errMsg)
	}

	// Check if VM exists and is running
	state, err := e.vmManager.GetVMState(execCtx.VMName)
	if err != nil {
		return nil, fmt.Errorf("failed to get VM state: %w", err)
	}

	if state != vm.StateRunning {
		return nil, fmt.Errorf("VM is not running (current state: %s)", state)
	}

	// Perform pre-execution sync if requested
	if execCtx.SyncBefore {
		log.Info().Str("vm", execCtx.VMName).Msg("Syncing files to VM before command execution")
		if err := e.syncEngine.SyncToVM(execCtx.VMName, ""); err != nil {
			return nil, fmt.Errorf("failed to sync files to VM: %w", err)
		}
	}

	// Execute command
	startTime := time.Now()
	result, err := e.executeSSHCommand(ctx, command, execCtx, callback)
	duration := time.Since(startTime).Seconds()

	// Set duration in result
	if result != nil {
		result.Duration = duration
	}

	// Handle execution error
	if err != nil {
		return result, fmt.Errorf("command execution failed: %w", err)
	}

	// Perform post-execution sync if requested
	if execCtx.SyncAfter {
		log.Info().Str("vm", execCtx.VMName).Msg("Syncing files from VM after command execution")
		if err := e.syncEngine.SyncFromVM(execCtx.VMName, ""); err != nil {
			return result, fmt.Errorf("failed to sync files from VM: %w", err)
		}
	}

	return result, nil
}

// executeSSHCommand executes a command via SSH in a VM
func (e *Executor) executeSSHCommand(ctx context.Context, command string, execCtx ExecutionContext, callback OutputCallback) (*CommandResult, error) {
	// Get SSH config for the VM
	sshConfig, err := e.vmManager.GetSSHConfig(execCtx.VMName)
	if err != nil {
		return nil, fmt.Errorf("failed to get SSH config: %w", err)
	}

	// Build the SSH command
	sshArgs := []string{
		"-p", sshConfig["Port"],
		"-i", sshConfig["IdentityFile"],
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		fmt.Sprintf("%s@%s", sshConfig["User"], sshConfig["HostName"]),
	}

	// Add working directory if specified
	fullCommand := command
	if execCtx.WorkingDir != "" {
		if strings.HasPrefix(execCtx.WorkingDir, "/vagrant") {
			fullCommand = fmt.Sprintf("cd %s && %s", execCtx.WorkingDir, command)
		} else {
			// If not absolute or under /vagrant, prepend /vagrant
			fullCommand = fmt.Sprintf("cd /vagrant/%s && %s", execCtx.WorkingDir, command)
		}
	}

	// Add environment variables if specified
	if len(execCtx.Environment) > 0 {
		envParts := []string{}
		for key, value := range execCtx.Environment {
			envParts = append(envParts, fmt.Sprintf("export %s=%s", key, value))
		}
		fullCommand = fmt.Sprintf("%s && %s", strings.Join(envParts, "; "), fullCommand)
	}

	// Add command to SSH args
	sshArgs = append(sshArgs, fullCommand)

	// Create SSH command
	cmd := exec.CommandContext(ctx, "ssh", sshArgs...)

	// Capture stdout and stderr
	var stdout, stderr bytes.Buffer

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start command
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start command: %w", err)
	}

	// Process command output in separate goroutines
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		e.streamOutput(stdoutPipe, &stdout, false, callback)
	}()

	go func() {
		defer wg.Done()
		e.streamOutput(stderrPipe, &stderr, true, callback)
	}()

	// Wait for output processing to complete
	wg.Wait()

	// Wait for command to complete
	err = cmd.Wait()

	// Create result
	result := &CommandResult{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}

	// Handle exit code
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = -1
			return result, fmt.Errorf("command failed: %w", err)
		}
	} else {
		result.ExitCode = 0
	}

	return result, nil
}

// streamOutput processes and captures command output
func (e *Executor) streamOutput(r io.Reader, buffer *bytes.Buffer, isStderr bool, callback OutputCallback) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Bytes()

		// Write to buffer
		buffer.Write(line)
		buffer.WriteByte('\n')

		// Call callback if provided
		if callback != nil {
			lineCopy := make([]byte, len(line))
			copy(lineCopy, line)
			callback(lineCopy, isStderr)
		}
	}
}

// GetVMDir returns the directory for a VM
func (e *Executor) GetVMDir(vmName string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	vmDir := filepath.Join(homeDir, ".vagrant-mcp", "vms", vmName)
	if _, err := os.Stat(vmDir); os.IsNotExist(err) {
		return "", fmt.Errorf("VM directory does not exist: %s", vmDir)
	}

	return vmDir, nil
}
