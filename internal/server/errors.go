// Package server provides HTTP server error handling for Alpine workflow git clone operations.
//
// This module implements Task 7: Add Server-Specific Error Handling from plan.md.
//
// The error handling strategy is:
//  1. Git clone errors (timeout, repo not found, disabled) return appropriate HTTP status codes
//     but allow graceful fallback by continuing with workflow creation
//  2. Authentication and generic errors fail the request entirely
//  3. All responses include user-friendly error messages
package server

import (
	"errors"
	"net/http"
	"strings"
)

// Error messages for different failure scenarios
const (
	// Git clone error messages (allow graceful fallback)
	MsgCloneTimeout  = "Git clone operation timed out. Please try again or check repository availability."
	MsgRepoNotFound  = "Repository not found. Please verify the repository exists and you have access."
	MsgCloneDisabled = "Git clone is disabled. Workflow will proceed with empty directory."

	// Critical error messages (fail the request)
	MsgAuthFailed     = "Authentication failed. Please check your access token for private repositories."
	MsgWorkflowFailed = "Workflow creation failed. Please try again."

	// Response metadata
	MsgFallbackWarning = "Workflow created successfully but with warnings"
)

// Server error types for git clone failures (currently unused but kept for potential future use)
var (
	// ErrServerCloneTimeout indicates a server clone timeout occurred
	ErrServerCloneTimeout = errors.New("server clone timeout")

	// ErrServerRepoNotFound indicates a server repository not found error
	ErrServerRepoNotFound = errors.New("server repository not found")

	// ErrServerCloneDisabled indicates server clone is disabled
	ErrServerCloneDisabled = errors.New("server clone disabled")

	// ErrServerAuthFailed indicates server authentication failure
	ErrServerAuthFailed = errors.New("server authentication failed")

	// ErrServerCloneGeneric indicates a generic server clone error
	ErrServerCloneGeneric = errors.New("server clone failed")
)

// ErrorResponse represents the HTTP response strategy for workflow errors.
// It determines both the HTTP status code to return and whether the workflow
// should continue with graceful fallback despite the error.
type ErrorResponse struct {
	StatusCode     int    // HTTP status code to return to the client
	Message        string // User-friendly error message
	ShouldFallback bool   // If true, continue workflow creation despite error
}

// mapWorkflowErrorToServerError maps workflow errors to appropriate HTTP responses.
//
// This function implements the server error handling strategy defined in Task 7:
//   - Git clone errors (timeout, repo not found, disabled) return proper HTTP status codes
//     but allow graceful fallback by continuing with workflow creation
//   - Authentication and generic errors fail the request entirely
//
// Parameters:
//   - err: The error returned from workflow.StartWorkflow()
//
// Returns:
//   - ErrorResponse: Contains HTTP status code, message, and fallback strategy
func mapWorkflowErrorToServerError(err error) ErrorResponse {
	if err == nil {
		return ErrorResponse{StatusCode: http.StatusCreated, Message: "", ShouldFallback: true}
	}

	errStr := err.Error()

	// Git clone specific errors - return proper error codes but allow graceful fallback
	if errors.Is(err, ErrCloneTimeout) {
		return ErrorResponse{
			StatusCode:     http.StatusGatewayTimeout,
			Message:        MsgCloneTimeout,
			ShouldFallback: true, // Continue workflow creation despite timeout
		}
	}

	if errors.Is(err, ErrRepoNotFound) {
		return ErrorResponse{
			StatusCode:     http.StatusNotFound,
			Message:        MsgRepoNotFound,
			ShouldFallback: true, // Continue workflow creation despite missing repo
		}
	}

	if errors.Is(err, ErrCloneDisabled) {
		return ErrorResponse{
			StatusCode:     http.StatusBadRequest,
			Message:        MsgCloneDisabled,
			ShouldFallback: true, // Continue workflow creation when clone disabled
		}
	}

	// Authentication errors - these should fail the request
	if isAuthenticationError(errStr) {
		return ErrorResponse{
			StatusCode:     http.StatusUnauthorized,
			Message:        MsgAuthFailed,
			ShouldFallback: false,
		}
	}

	// Generic workflow errors - fail the request
	return ErrorResponse{
		StatusCode:     http.StatusInternalServerError,
		Message:        MsgWorkflowFailed,
		ShouldFallback: false,
	}
}

// isAuthenticationError checks if an error message indicates authentication failure.
//
// This function performs case-insensitive pattern matching against common
// authentication error messages to distinguish authentication failures from
// other types of git clone errors.
//
// Authentication errors are treated as critical failures that should not
// allow graceful fallback, unlike other git clone errors.
//
// Parameters:
//   - errStr: The error message to check
//
// Returns:
//   - bool: true if the error appears to be authentication-related
func isAuthenticationError(errStr string) bool {
	authPatterns := []string{
		"authentication failed",
		"permission denied",
		"401 unauthorized",
		"invalid credentials",
		"access denied",
	}

	lowerErr := strings.ToLower(errStr)
	for _, pattern := range authPatterns {
		if strings.Contains(lowerErr, pattern) {
			return true
		}
	}
	return false
}
