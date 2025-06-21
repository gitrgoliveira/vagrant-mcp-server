package sync

import "errors"

// Common errors returned by the sync engine
var (
	ErrVMAlreadyRegistered  = errors.New("vm already registered")
	ErrVMNotRegistered      = errors.New("vm not registered")
	ErrEngineAlreadyRunning = errors.New("sync engine already running")
	ErrEngineNotRunning     = errors.New("sync engine not running")
	ErrInvalidVMName        = errors.New("invalid vm name")
)
