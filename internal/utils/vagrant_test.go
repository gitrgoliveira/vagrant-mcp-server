package utils

import (
	"testing"
)

func TestCheckVagrantInstalled(t *testing.T) {
	err := CheckVagrantInstalled()
	if err != nil {
		t.Errorf("Vagrant check failed: %v", err)
		t.Log("This test requires Vagrant CLI to be installed and in PATH")
		t.Log("Please ensure Vagrant is installed correctly")
	}
}
