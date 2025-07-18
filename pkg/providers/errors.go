package providers

import (
	"errors"
	"fmt"
)

// Common provider errors
var (
	// ErrProviderNotFound is returned when a provider is not found
	ErrProviderNotFound = errors.New("provider not found")
	
	// ErrProviderNotAuthenticated is returned when a provider is not authenticated
	ErrProviderNotAuthenticated = errors.New("provider not authenticated")
	
	// ErrResourceNotFound is returned when a resource is not found
	ErrResourceNotFound = errors.New("resource not found")
	
	// ErrInvalidConfiguration is returned when provider configuration is invalid
	ErrInvalidConfiguration = errors.New("invalid provider configuration")
	
	// ErrUnsupportedOperation is returned when an operation is not supported
	ErrUnsupportedOperation = errors.New("operation not supported")
	
	// ErrResourceTypeNotSupported is returned when a resource type is not supported
	ErrResourceTypeNotSupported = errors.New("resource type not supported")
	
	// ErrRegionNotSupported is returned when a region is not supported
	ErrRegionNotSupported = errors.New("region not supported")
	
	// ErrRateLimitExceeded is returned when API rate limits are exceeded
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
	
	// ErrPermissionDenied is returned when access is denied
	ErrPermissionDenied = errors.New("permission denied")
	
	// ErrServiceUnavailable is returned when a service is unavailable
	ErrServiceUnavailable = errors.New("service unavailable")
)

// ProviderError represents a provider-specific error
type ProviderError struct {
	Provider  string
	Operation string
	Message   string
	Cause     error
}

// Error implements the error interface
func (e *ProviderError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("provider %s operation %s failed: %s (caused by: %v)", 
			e.Provider, e.Operation, e.Message, e.Cause)
	}
	return fmt.Sprintf("provider %s operation %s failed: %s", 
		e.Provider, e.Operation, e.Message)
}

// Unwrap returns the underlying error
func (e *ProviderError) Unwrap() error {
	return e.Cause
}

// NewProviderError creates a new provider error
func NewProviderError(provider, operation, message string, cause error) *ProviderError {
	return &ProviderError{
		Provider:  provider,
		Operation: operation,
		Message:   message,
		Cause:     cause,
	}
}

// AuthenticationError represents an authentication-related error
type AuthenticationError struct {
	Provider string
	Message  string
	Cause    error
}

// Error implements the error interface
func (e *AuthenticationError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("authentication failed for provider %s: %s (caused by: %v)", 
			e.Provider, e.Message, e.Cause)
	}
	return fmt.Sprintf("authentication failed for provider %s: %s", 
		e.Provider, e.Message)
}

// Unwrap returns the underlying error
func (e *AuthenticationError) Unwrap() error {
	return e.Cause
}

// NewAuthenticationError creates a new authentication error
func NewAuthenticationError(provider, message string, cause error) *AuthenticationError {
	return &AuthenticationError{
		Provider: provider,
		Message:  message,
		Cause:    cause,
	}
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Value   interface{}
	Message string
}

// Error implements the error interface
func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed for field %s (value: %v): %s", 
		e.Field, e.Value, e.Message)
}

// NewValidationError creates a new validation error
func NewValidationError(field string, value interface{}, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Value:   value,
		Message: message,
	}
}

// ResourceError represents a resource-specific error
type ResourceError struct {
	ResourceID   string
	ResourceType string
	Provider     string
	Operation    string
	Message      string
	Cause        error
}

// Error implements the error interface
func (e *ResourceError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("resource error for %s %s in provider %s during %s: %s (caused by: %v)", 
			e.ResourceType, e.ResourceID, e.Provider, e.Operation, e.Message, e.Cause)
	}
	return fmt.Sprintf("resource error for %s %s in provider %s during %s: %s", 
		e.ResourceType, e.ResourceID, e.Provider, e.Operation, e.Message)
}

// Unwrap returns the underlying error
func (e *ResourceError) Unwrap() error {
	return e.Cause
}

// NewResourceError creates a new resource error
func NewResourceError(resourceID, resourceType, provider, operation, message string, cause error) *ResourceError {
	return &ResourceError{
		ResourceID:   resourceID,
		ResourceType: resourceType,
		Provider:     provider,
		Operation:    operation,
		Message:      message,
		Cause:        cause,
	}
}

// IsRetryable checks if an error is retryable
func IsRetryable(err error) bool {
	switch {
	case errors.Is(err, ErrRateLimitExceeded):
		return true
	case errors.Is(err, ErrServiceUnavailable):
		return true
	default:
		return false
	}
}

// IsNotFound checks if an error indicates a resource was not found
func IsNotFound(err error) bool {
	return errors.Is(err, ErrResourceNotFound)
}

// IsAuthenticationError checks if an error is an authentication error
func IsAuthenticationError(err error) bool {
	var authErr *AuthenticationError
	return errors.As(err, &authErr) || errors.Is(err, ErrProviderNotAuthenticated)
}

// IsPermissionError checks if an error is a permission error
func IsPermissionError(err error) bool {
	return errors.Is(err, ErrPermissionDenied)
}

// IsValidationError checks if an error is a validation error
func IsValidationError(err error) bool {
	var validErr *ValidationError
	return errors.As(err, &validErr) || errors.Is(err, ErrInvalidConfiguration)
}