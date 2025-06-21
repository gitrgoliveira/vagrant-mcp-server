package exec

import (
	"fmt"
	"os/exec"

	syncmod "github.com/vagrant-mcp/server/internal/sync"
	"github.com/vagrant-mcp/server/internal/vm"
)

// VMManagerAdapter adapts *vm.Manager to the VMManager interface
// Only implements the methods needed by Executor

type VMManagerAdapter struct {
	Real *vm.Manager
}

func (a *VMManagerAdapter) CreateVM(name, projectPath string, config vm.VMConfig) error {
	return a.Real.CreateVM(name, projectPath, config)
}
func (a *VMManagerAdapter) StartVM(name string) error                    { return a.Real.StartVM(name) }
func (a *VMManagerAdapter) StopVM(name string) error                     { return a.Real.StopVM(name) }
func (a *VMManagerAdapter) DestroyVM(name string) error                  { return a.Real.DestroyVM(name) }
func (a *VMManagerAdapter) GetVMState(name string) (vm.State, error)     { return a.Real.GetVMState(name) }
func (a *VMManagerAdapter) SyncToVM(name, source, target string) error   { return nil }
func (a *VMManagerAdapter) SyncFromVM(name, source, target string) error { return nil }
func (a *VMManagerAdapter) GetSSHConfig(name string) (map[string]string, error) {
	return a.Real.GetSSHConfig(name)
}
func (a *VMManagerAdapter) GetVMConfig(name string) (vm.VMConfig, error) {
	return a.Real.GetVMConfig(name)
}
func (a *VMManagerAdapter) UpdateVMConfig(name string, config vm.VMConfig) error {
	return a.Real.UpdateVMConfig(name, config)
}
func (a *VMManagerAdapter) GetBaseDir() string {
	return a.Real.GetBaseDir()
}

// ExecuteCommand runs a command in the VM using SSH
func (a *VMManagerAdapter) ExecuteCommand(name string, cmd string, args []string, workingDir string) (string, string, int, error) {
	sshConfig, err := a.Real.GetSSHConfig(name)
	if err != nil {
		return "", "", 1, err
	}
	sshArgs := []string{
		"-p", sshConfig["Port"],
		"-i", sshConfig["IdentityFile"],
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		fmt.Sprintf("%s@%s", sshConfig["User"], sshConfig["HostName"]),
	}
	fullCmd := cmd
	if workingDir != "" {
		fullCmd = fmt.Sprintf("cd %s && %s", workingDir, cmd)
	}
	sshArgs = append(sshArgs, fullCmd)
	c := exec.Command("ssh", sshArgs...)
	out, err := c.CombinedOutput()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}
	return string(out), "", exitCode, err
}

// SyncEngineAdapter adapts *sync.Engine to the SyncEngine interface
type SyncEngineAdapter struct {
	Real *syncmod.Engine
}

func (a *SyncEngineAdapter) RegisterVM(vmName string) error {
	return a.Real.RegisterVM(vmName, syncmod.SyncConfig{})
}
func (a *SyncEngineAdapter) UnregisterVM(vmName string) error {
	return a.Real.UnregisterVM(vmName)
}
func (a *SyncEngineAdapter) SyncToVM(vmName string, sourcePath string) (*syncmod.SyncResult, error) {
	return a.Real.SyncToVM(vmName, sourcePath)
}
func (a *SyncEngineAdapter) SyncFromVM(vmName string, sourcePath string) (*syncmod.SyncResult, error) {
	return a.Real.SyncFromVM(vmName, sourcePath)
}
func (a *SyncEngineAdapter) GetSyncStatus(vmName string) (syncmod.SyncStatus, error) {
	return a.Real.GetSyncStatus(vmName)
}
func (a *SyncEngineAdapter) ResolveSyncConflict(vmName, path, resolution string) error {
	return a.Real.ResolveSyncConflict(vmName, path, resolution)
}
func (a *SyncEngineAdapter) SemanticSearch(vmName, query string, maxResults int) ([]syncmod.SearchResult, error) {
	return a.Real.SemanticSearch(vmName, query, maxResults)
}
func (a *SyncEngineAdapter) ExactSearch(vmName, query string, caseSensitive bool, maxResults int) ([]syncmod.SearchResult, error) {
	return a.Real.ExactSearch(vmName, query, caseSensitive, maxResults)
}
func (a *SyncEngineAdapter) FuzzySearch(vmName, query string, maxResults int) ([]syncmod.SearchResult, error) {
	return a.Real.FuzzySearch(vmName, query, maxResults)
}
