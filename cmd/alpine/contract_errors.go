package main

import "fmt"

const (
	ErrCommandReplaced         = "ERR_COMMAND_REPLACED"
	ErrCallerIdentityRequired  = "ERR_CALLER_IDENTITY_REQUIRED"
	ErrRepoAuthMissing         = "ERR_REPO_AUTH_MISSING"
	ErrSessionForbidden        = "ERR_SESSION_FORBIDDEN"
	ErrSessionNotFound         = "ERR_SESSION_NOT_FOUND"
	ErrOperationConflict       = "ERR_OPERATION_CONFLICT"
	ErrPersistPayloadTooLarge  = "ERR_PERSIST_PAYLOAD_TOO_LARGE"
	ErrInvalidRepoURL          = "ERR_INVALID_REPO_URL"
	ErrInvalidBranch           = "ERR_INVALID_BRANCH"
	ErrBranchRequired          = "ERR_BRANCH_REQUIRED"
	ErrLegacyUnsupported       = "ERR_LEGACY_UNSUPPORTED"
	ErrSandboxProvisionFailed  = "ERR_SANDBOX_PROVISION_FAILED"
	ErrPersistenceVerification = "ERR_PERSISTENCE_VERIFICATION_FAILED"
)

type commandError struct {
	msg         string
	exitCode    int
	errorCode   string
	cause       string
	nextStep    string
	retryable   bool
	operationID string
}

func (e *commandError) Error() string { return e.msg }

func newCommandError(exitCode int, errorCode, cause, nextStep string, retryable bool, operationID string) error {
	msg := cause
	if nextStep != "" {
		msg = fmt.Sprintf("%s. %s", cause, nextStep)
	}
	return &commandError{
		msg:         msg,
		exitCode:    exitCode,
		errorCode:   errorCode,
		cause:       cause,
		nextStep:    nextStep,
		retryable:   retryable,
		operationID: operationID,
	}
}
