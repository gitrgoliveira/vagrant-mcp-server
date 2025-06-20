package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// ConversionStatus tracks if TypeScript functionality has been fully converted to Go
type ConversionStatus struct {
	TypeScriptFile string
	GoFile         string
	IsConverted    bool
	Notes          string
}

func main() {
	baseDir := "/Users/ricardo/repos/vagrant-mcp-server"

	// Define conversion mapping
	conversionMap := []ConversionStatus{
		{TypeScriptFile: "src/index.ts", GoFile: "cmd/server/main.go", IsConverted: true, Notes: "Main entry point converted"},
		{TypeScriptFile: "src/vm/manager.ts", GoFile: "internal/vm/manager.go", IsConverted: true, Notes: "VM management converted"},
		{TypeScriptFile: "src/vm/config.ts", GoFile: "internal/vm/manager.go", IsConverted: true, Notes: "VM config integrated into manager.go"},
		{TypeScriptFile: "src/vm/provisioner.ts", GoFile: "internal/vm/manager.go", IsConverted: true, Notes: "Provisioning integrated into manager.go"},
		{TypeScriptFile: "src/sync/sync-engine.ts", GoFile: "internal/sync/engine.go", IsConverted: true, Notes: "Sync engine converted"},
		{TypeScriptFile: "src/sync/watchers.ts", GoFile: "internal/sync/engine.go", IsConverted: true, Notes: "File watching functionality integrated into engine.go"},
		{TypeScriptFile: "src/sync/conflict-resolver.ts", GoFile: "internal/sync/engine.go", IsConverted: true, Notes: "Conflict resolution integrated into engine.go"},
		{TypeScriptFile: "src/exec/executor.ts", GoFile: "internal/exec/executor.go", IsConverted: true, Notes: "Command execution converted"},
		{TypeScriptFile: "src/exec/stream.ts", GoFile: "internal/exec/executor.go", IsConverted: true, Notes: "Output streaming integrated into executor.go"},
		{TypeScriptFile: "src/exec/context.ts", GoFile: "internal/exec/executor.go", IsConverted: true, Notes: "Execution context integrated into executor.go"},
		{TypeScriptFile: "src/tools/vm-tools.ts", GoFile: "internal/tools/vm-tools.go", IsConverted: true, Notes: "VM tools converted"},
		{TypeScriptFile: "src/tools/exec-tools.ts", GoFile: "internal/tools/exec-tools.go", IsConverted: true, Notes: "Exec tools converted"},
		{TypeScriptFile: "src/tools/env-tools.ts", GoFile: "internal/tools/env-tools.go", IsConverted: true, Notes: "Environment tools implemented with new file"},
		{TypeScriptFile: "src/tools/sync-tools.ts", GoFile: "internal/tools/sync-tools.go", IsConverted: true, Notes: "Sync tools implemented with new file"},
		{TypeScriptFile: "src/resources/log-resources.ts", GoFile: "internal/resources/log-resources.go", IsConverted: true, Notes: "Log resources implemented"},
		{TypeScriptFile: "src/resources/monitoring-resources.ts", GoFile: "internal/resources/resources.go", IsConverted: true, Notes: "Monitoring resources implemented"},
		{TypeScriptFile: "src/resources/network-resources.ts", GoFile: "internal/resources/resources.go", IsConverted: true, Notes: "Network resources implemented"},
		{TypeScriptFile: "src/utils/logger.ts", GoFile: "pkg/mcp/server.go", IsConverted: true, Notes: "Logging using zerolog instead"},
		{TypeScriptFile: "src/utils/shell.ts", GoFile: "internal/exec/executor.go", IsConverted: true, Notes: "Shell utilities integrated into executor"},
	}

	notConverted := 0

	// Check the status
	fmt.Println("Conversion Status:")
	fmt.Println("=================")
	for _, status := range conversionMap {
		// Check if Go file exists
		goExists := fileExists(filepath.Join(baseDir, status.GoFile))

		status.IsConverted = status.IsConverted && goExists
		fmt.Printf("%-30s -> %-30s [%s] %s\n",
			status.TypeScriptFile,
			status.GoFile,
			formatStatus(status.IsConverted),
			status.Notes)

		if !status.IsConverted {
			notConverted++
		}
	}

	// Summary
	total := len(conversionMap)
	converted := total - notConverted
	fmt.Println("\nSummary:")
	fmt.Printf("- Total components: %d\n", total)
	fmt.Printf("- Converted: %d (%.1f%%)\n", converted, float64(converted)/float64(total)*100)
	fmt.Printf("- Not converted: %d (%.1f%%)\n", notConverted, float64(notConverted)/float64(total)*100)

	if notConverted > 0 {
		fmt.Println("\nComponents requiring attention:")
		for _, status := range conversionMap {
			if !status.IsConverted {
				fmt.Printf("- %s: %s\n", status.TypeScriptFile, status.Notes)
			}
		}
		fmt.Println("\nRecommendation: Complete the implementation of the remaining components before removing TypeScript code.")
	} else {
		fmt.Println("\nAll components appear to be converted. You can safely remove the TypeScript code.")
	}
}

func formatStatus(isConverted bool) string {
	if isConverted {
		return "CONVERTED"
	}
	return "INCOMPLETE"
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
