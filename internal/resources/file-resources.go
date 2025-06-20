package resources

import (
	"context"
	"encoding/base64"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/vagrant-mcp/server/internal/exec"
	"github.com/vagrant-mcp/server/internal/vm"
	"github.com/vagrant-mcp/server/pkg/mcp"
)

// VMFilesResource provides access to files in the VM
type VMFilesResource struct {
	vmManager *vm.Manager
	executor  *exec.Executor
}

// NewVMFilesResource creates a new VM files resource
func NewVMFilesResource(vmManager *vm.Manager, executor *exec.Executor) *VMFilesResource {
	return &VMFilesResource{
		vmManager: vmManager,
		executor:  executor,
	}
}

// Name returns the resource name
func (r *VMFilesResource) Name() string {
	return "devvm://files/{path}"
}

// Description returns the resource description
func (r *VMFilesResource) Description() string {
	return "Access to VM file system (read-only)"
}

// Get retrieves a file or directory listing from the VM
func (r *VMFilesResource) Get(path string) (interface{}, error) {
	// Parse VM name and file path from request
	// Expected format: devvm://files/{vmName}/{path}
	if path == "" || !strings.Contains(path, "/") {
		return nil, fmt.Errorf("invalid path format, expected devvm://files/{vmName}/{path}")
	}

	parts := strings.SplitN(path, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid path format")
	}

	vmName := parts[0]
	filePath := "/" + parts[1] // Ensure path starts with /

	// Check VM state
	state, err := r.vmManager.GetVMState(vmName)
	if err != nil {
		return nil, fmt.Errorf("failed to get VM state: %w", err)
	}

	if state != vm.StateRunning {
		return nil, fmt.Errorf("VM is not running (current state: %s)", state)
	}

	// Setup execution context
	execCtx := exec.ExecutionContext{
		VMName:     vmName,
		WorkingDir: "/",
		SyncBefore: false,
		SyncAfter:  false,
	}

	// First check if path exists and what type it is
	statCmd := fmt.Sprintf("stat -c '%%F:%%s:%%Y' '%s' 2>/dev/null || echo 'not found'", filePath)
	result, err := r.executor.ExecuteCommand(context.Background(), statCmd, execCtx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to check file: %w", err)
	}

	statOutput := strings.TrimSpace(result.Stdout)
	if statOutput == "not found" {
		return nil, fmt.Errorf("file or directory not found: %s", filePath)
	}

	// Parse stat output (format: type:size:modified_time)
	statParts := strings.Split(statOutput, ":")
	if len(statParts) != 3 {
		return nil, fmt.Errorf("failed to parse file info")
	}

	fileType := statParts[0]
	size := 0
	if s, err := fmt.Sscanf(statParts[1], "%d", &size); err != nil || s != 1 {
		size = 0
	}

	modTime := int64(0)
	if t, err := fmt.Sscanf(statParts[2], "%d", &modTime); err != nil || t != 1 {
		modTime = 0
	}

	modifiedTime := time.Unix(modTime, 0).Format(time.RFC3339)

	// If it's a directory, list its contents
	if fileType == "directory" {
		lsCmd := fmt.Sprintf("ls -la '%s' | tail -n +4 | awk '{print $1,$5,$6,$7,$8,$9}'", filePath)
		result, err := r.executor.ExecuteCommand(context.Background(), lsCmd, execCtx, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to list directory: %w", err)
		}

		entries := []map[string]interface{}{}
		for _, line := range strings.Split(result.Stdout, "\n") {
			if line == "" {
				continue
			}

			fields := strings.Fields(line)
			if len(fields) < 6 {
				continue
			}

			perms := fields[0]
			var itemSize int
			if _, err := fmt.Sscanf(fields[1], "%d", &itemSize); err != nil {
				itemSize = 0 // fallback if parse fails
			}

			// Get the name from the remaining fields
			name := strings.Join(fields[5:], " ")

			isDir := strings.HasPrefix(perms, "d")

			entries = append(entries, map[string]interface{}{
				"name":      name,
				"is_dir":    isDir,
				"size":      itemSize,
				"perms":     perms,
				"full_path": filepath.Join(filePath, name),
			})
		}

		return map[string]interface{}{
			"vm_name":     vmName,
			"path":        filePath,
			"type":        "directory",
			"entries":     entries,
			"modified":    modifiedTime,
			"entry_count": len(entries),
		}, nil
	}

	// For regular files, return the file content
	if size > 1024*1024 {
		// For files > 1MB, return the file info without content
		return map[string]interface{}{
			"vm_name":   vmName,
			"path":      filePath,
			"type":      "file",
			"size":      size,
			"modified":  modifiedTime,
			"too_large": true,
			"message":   "File too large to return content",
		}, nil
	}

	// Read file content
	catCmd := fmt.Sprintf("base64 '%s'", filePath)
	result, err = r.executor.ExecuteCommand(context.Background(), catCmd, execCtx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Basic file type detection
	fileExtension := filepath.Ext(filePath)
	contentType := "text/plain"

	switch strings.ToLower(fileExtension) {
	case ".json":
		contentType = "application/json"
	case ".xml":
		contentType = "application/xml"
	case ".html", ".htm":
		contentType = "text/html"
	case ".css":
		contentType = "text/css"
	case ".js":
		contentType = "application/javascript"
	case ".png":
		contentType = "image/png"
	case ".jpg", ".jpeg":
		contentType = "image/jpeg"
	case ".gif":
		contentType = "image/gif"
	case ".pdf":
		contentType = "application/pdf"
	}

	// Try to decode base64 content
	content, err := base64.StdEncoding.DecodeString(strings.TrimSpace(result.Stdout))
	if err != nil {
		return nil, fmt.Errorf("failed to decode file content: %w", err)
	}

	return map[string]interface{}{
		"vm_name":      vmName,
		"path":         filePath,
		"type":         "file",
		"size":         size,
		"modified":     modifiedTime,
		"content_type": contentType,
		"content":      string(content),
	}, nil
}

// RegisterFileResources registers all file-related resources with the MCP server
func RegisterFileResources(server interface{}, vmManager *vm.Manager, executor interface{}) error {
	mcpServer, ok := server.(*mcp.Server)
	if !ok {
		return fmt.Errorf("server is not *mcp.Server")
	}

	execExecutor, ok := executor.(*exec.Executor)
	if !ok {
		return fmt.Errorf("executor is not of type *exec.Executor")
	}

	// Register files resource
	filesResource := NewVMFilesResource(vmManager, execExecutor)
	if err := mcpServer.RegisterResource(filesResource); err != nil {
		return fmt.Errorf("failed to register files resource: %w", err)
	}

	return nil
}
