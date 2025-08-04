// Copyright Ricardo Oliveira 2025.
// SPDX-License-Identifier: MPL-2.0

// Package errors provides standardized error types and handling for the application
package errors

import (
	"errors"
	"fmt"
)

// Standard error types that can be used across the application
var (
	ErrNotFound          = errors.New("resource not found")
	ErrAlreadyExists     = errors.New("resource already exists")
	ErrInvalidInput      = errors.New("invalid input")
	ErrOperationFailed   = errors.New("operation failed")
	ErrNotImplemented    = errors.New("not implemented")
	ErrPermissionDenied  = errors.New("permission denied")
	ErrTimeout           = errors.New("operation timed out")
	ErrCancelled         = errors.New("operation was cancelled")
	ErrDependencyMissing = errors.New("dependency is missing")
	ErrInvalidState      = errors.New("invalid state for operation")
	ErrValidationFailed  = errors.New("validation failed")
)

// ErrorCode represents specific error codes for better error handling
type ErrorCode string

// Standard error codes
const (
	CodeNotFound          ErrorCode = "not_found"
	CodeAlreadyExists     ErrorCode = "already_exists"
	CodeInvalidInput      ErrorCode = "invalid_input"
	CodeOperationFailed   ErrorCode = "operation_failed"
	CodeNotImplemented    ErrorCode = "not_implemented"
	CodePermissionDenied  ErrorCode = "permission_denied"
	CodeTimeout           ErrorCode = "timeout"
	CodeCancelled         ErrorCode = "cancelled"
	CodeDependencyMissing ErrorCode = "dependency_missing"
	CodeInvalidState      ErrorCode = "invalid_state"
	CodeValidationFailed  ErrorCode = "validation_failed"
	CodeVagrantError      ErrorCode = "vagrant_error"
	CodeVMError           ErrorCode = "vm_error"
	CodeSyncError         ErrorCode = "sync_error"
	CodeExecError         ErrorCode = "exec_error"
)

// AppError represents an application-specific error with context
type AppError struct {
	Code    ErrorCode
	Message string
	Err     error
	Context map[string]interface{}
}

// Error implements the error interface for AppError
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Unwrap implements the unwrap interface to support errors.Is and errors.As
func (e *AppError) Unwrap() error {
	return e.Err
}

// WithContext adds context information to the error
func (e *AppError) WithContext(key string, value interface{}) *AppError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// New creates a new AppError with the given code and message
func New(code ErrorCode, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
	}
}

// Wrap wraps an existing error in an AppError
func Wrap(err error, code ErrorCode, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// NotFound creates a new not found error
func NotFound(resourceType, identifier string) *AppError {
	return &AppError{
		Code:    CodeNotFound,
		Message: fmt.Sprintf("%s '%s' not found", resourceType, identifier),
		Err:     ErrNotFound,
		Context: map[string]interface{}{
			"resourceType": resourceType,
			"identifier":   identifier,
		},
	}
}

// AlreadyExists creates a new already exists error
func AlreadyExists(resourceType, identifier string) *AppError {
	return &AppError{
		Code:    CodeAlreadyExists,
		Message: fmt.Sprintf("%s '%s' already exists", resourceType, identifier),
		Err:     ErrAlreadyExists,
		Context: map[string]interface{}{
			"resourceType": resourceType,
			"identifier":   identifier,
		},
	}
}

// InvalidInput creates a new invalid input error
func InvalidInput(details string) *AppError {
	return &AppError{
		Code:    CodeInvalidInput,
		Message: fmt.Sprintf("Invalid input: %s", details),
		Err:     ErrInvalidInput,
	}
}

// OperationFailed creates a new operation failed error
func OperationFailed(operation string, err error) *AppError {
	return &AppError{
		Code:    CodeOperationFailed,
		Message: fmt.Sprintf("Operation '%s' failed", operation),
		Err:     err,
		Context: map[string]interface{}{
			"operation": operation,
		},
	}
}

// IsNotFound checks if the error is a not found error
func IsNotFound(err error) bool {
	return Is(err, CodeNotFound) || errors.Is(err, ErrNotFound)
}

// IsAlreadyExists checks if the error is an already exists error
func IsAlreadyExists(err error) bool {
	return Is(err, CodeAlreadyExists) || errors.Is(err, ErrAlreadyExists)
}

// Is checks if the error is of the specified code
func Is(err error, code ErrorCode) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code == code
	}
	return false
}
