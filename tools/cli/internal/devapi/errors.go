package devapi

import "fmt"

// ErrorType represents the type of error
type ErrorType string

const (
	ErrNetwork  ErrorType = "NETWORK"
	ErrBuild    ErrorType = "BUILD"
	ErrAPI      ErrorType = "API"
	ErrAuth     ErrorType = "AUTH"
	ErrConfig   ErrorType = "CONFIG"
)

// DevAPIError is a custom error type for Dev API errors
type DevAPIError struct {
	Type      ErrorType
	Code      string
	Message   string
	Original  error
	Retryable bool
}

func (e *DevAPIError) Error() string {
	if e.Original != nil {
		return fmt.Sprintf("%s: %s (%s) - %v", e.Type, e.Code, e.Message, e.Original)
	}
	return fmt.Sprintf("%s: %s (%s)", e.Type, e.Code, e.Message)
}

// NewNetworkError creates a network error
func NewNetworkError(message string, original error) *DevAPIError {
	return &DevAPIError{
		Type:      ErrNetwork,
		Code:      "NETWORK_ERROR",
		Message:   message,
		Original:  original,
		Retryable: true,
	}
}

// NewBuildError creates a build error
func NewBuildError(message string, original error) *DevAPIError {
	return &DevAPIError{
		Type:      ErrBuild,
		Code:      "BUILD_ERROR",
		Message:   message,
		Original:  original,
		Retryable: false,
	}
}

// NewAPIError creates an API error
func NewAPIError(code, message string, original error, retryable bool) *DevAPIError {
	return &DevAPIError{
		Type:      ErrAPI,
		Code:      code,
		Message:   message,
		Original:  original,
		Retryable: retryable,
	}
}

// NewAuthError creates an authentication error
func NewAuthError(message string, original error) *DevAPIError {
	return &DevAPIError{
		Type:      ErrAuth,
		Code:      "AUTH_ERROR",
		Message:   message,
		Original:  original,
		Retryable: false,
	}
}

// NewConfigError creates a configuration error
func NewConfigError(message string) *DevAPIError {
	return &DevAPIError{
		Type:      ErrConfig,
		Code:      "CONFIG_ERROR",
		Message:   message,
		Original:  nil,
		Retryable: false,
	}
}
