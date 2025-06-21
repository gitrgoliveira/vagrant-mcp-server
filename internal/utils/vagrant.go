package utils

import (
	"fmt"
	"os/exec"
)

// CheckVagrantInstalled checks if the Vagrant CLI is installed and available in the PATH
// It returns an error if Vagrant is not found or if there's an error running the command
func CheckVagrantInstalled() error {
	cmd := exec.Command("vagrant", "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("vagrant CLI is not available: %w", err)
	}

	// If we got output with no error, Vagrant is installed
	if len(output) > 0 {
		return nil
	}

	return fmt.Errorf("vagrant CLI check returned empty output")
}
