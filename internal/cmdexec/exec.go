// Copyright Ricardo Oliveira 2025.
// SPDX-License-Identifier: MPL-2.0

// Package cmdexec provides a unified command execution interface
package cmdexec

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/vagrant-mcp/server/internal/errors"
)

// OutputMode specifies how command output should be handled
type OutputMode int

const (
	// OutputModeCapture captures the output into the result
	OutputModeCapture OutputMode = iota
	// OutputModeStream streams the output through the provided callback
	OutputModeStream
	// OutputModeBoth both captures and streams the output
	OutputModeBoth
)

// StreamCallback is a function type for streaming command output
type StreamCallback func(data []byte, isStderr bool)

// CmdOptions represents options for command execution
type CmdOptions struct {
	// Directory is the working directory for the command
	Directory string
	// Environment variables to pass to the command (format: "KEY=VALUE")
	Environment []string
	// OutputMode specifies how output should be handled
	OutputMode OutputMode
	// OutputCallback is called with output data when streaming is enabled
	OutputCallback StreamCallback
	// Timeout specifies a timeout for the command execution (0 means no timeout)
	Timeout time.Duration
}

// Result contains the results of a command execution
type Result struct {
	// Command that was executed
	Command string
	// Arguments that were passed to the command
	Args []string
	// ExitCode returned by the command
	ExitCode int
	// StdOut output from the command
	StdOut []byte
	// StdErr output from the command
	StdErr []byte
	// Error if any occurred during execution
	Error error
	// Duration of the command execution
	Duration time.Duration
	// StartTime when the command started
	StartTime time.Time
	// EndTime when the command completed
	EndTime time.Time
}

// FormatCommand returns the full command that was executed as a string
func (r *Result) FormatCommand() string {
	if len(r.Args) == 0 {
		return r.Command
	}
	return fmt.Sprintf("%s %v", r.Command, r.Args)
}

// FormatDuration returns the duration as a human-readable string
func (r *Result) FormatDuration() string {
	return r.Duration.String()
}

// isSuccessful returns true if the command executed successfully
func (r *Result) IsSuccessful() bool {
	return r.ExitCode == 0 && r.Error == nil
}

// Execute runs a command and returns the result
func Execute(ctx context.Context, command string, args []string, options CmdOptions) (*Result, error) {
	result := &Result{
		Command:   command,
		Args:      args,
		StartTime: time.Now(),
	}

	// Create a command with the context
	cmd := exec.CommandContext(ctx, command, args...)

	// Set working directory if specified
	if options.Directory != "" {
		cmd.Dir = options.Directory
	}

	// Set environment variables if specified
	if len(options.Environment) > 0 {
		cmd.Env = options.Environment
	}

	// Create pipes for stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, errors.OperationFailed("create stdout pipe", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, errors.OperationFailed("create stderr pipe", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return nil, errors.OperationFailed("start command", err)
	}

	// Create waitgroups for goroutines
	var wg sync.WaitGroup

	// Process stdout
	wg.Add(1)
	go func() {
		defer wg.Done()
		processOutput(stdout, false, &result.StdOut, options)
	}()

	// Process stderr
	wg.Add(1)
	go func() {
		defer wg.Done()
		processOutput(stderr, true, &result.StdErr, options)
	}()

	// Wait for stdout and stderr to be processed
	wg.Wait()

	// Wait for the command to complete
	err = cmd.Wait()
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	// Get the exit code
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.Error = err
		}
	}

	// Log the result
	logger := log.With().
		Str("command", command).
		Strs("args", args).
		Int("exitCode", result.ExitCode).
		Dur("duration", result.Duration).
		Logger()

	if result.IsSuccessful() {
		logger.Debug().Msg("Command executed successfully")
	} else {
		logger.Warn().
			Err(result.Error).
			Str("stderr", string(result.StdErr)).
			Msg("Command execution failed")
	}

	return result, nil
}

// processOutput reads from a reader and handles it according to the output mode
func processOutput(r io.Reader, isStderr bool, buffer *[]byte, options CmdOptions) {
	// Determine if we need to capture output
	captureOutput := options.OutputMode == OutputModeCapture || options.OutputMode == OutputModeBoth

	// Determine if we need to stream output
	streamOutput := options.OutputMode == OutputModeStream || options.OutputMode == OutputModeBoth
	callback := options.OutputCallback

	// Read the output
	buf := make([]byte, 1024)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			data := buf[:n]

			// Capture output if required
			if captureOutput {
				*buffer = append(*buffer, data...)
			}

			// Stream output if required
			if streamOutput && callback != nil {
				callback(data, isStderr)
			}
		}

		if err != nil {
			break
		}
	}
}
