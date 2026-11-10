package publisher

import "errors"

const (
	InvalidInput     = "PUBLISHER_INVALID_INPUT"
	InvalidWorkspace = "PUBLISHER_INVALID_WORKSPACE"
	WorkspaceChanged = "PUBLISHER_WORKSPACE_CHANGED"
	InvalidRecord    = "PUBLISHER_INVALID_RECORD"
	StorageFailure   = "PUBLISHER_STORAGE_FAILURE"
	NotFound         = "PUBLISHER_NOT_FOUND"
	Conflict         = "PUBLISHER_CONFLICT"
	Unauthorized     = "PUBLISHER_UNAUTHORIZED"
	Forbidden        = "PUBLISHER_FORBIDDEN"
	Unsupported      = "PUBLISHER_UNSUPPORTED_PLATFORM"
	TooLarge         = "PUBLISHER_REQUEST_TOO_LARGE"
	MediaType        = "PUBLISHER_MEDIA_TYPE_REQUIRED"
	Method           = "PUBLISHER_METHOD_NOT_ALLOWED"
)

// Error contains machine semantics only. Presentation uses the locale catalog.
type Error struct {
	Code       string            `json:"code"`
	MessageKey string            `json:"message_key"`
	Params     map[string]string `json:"params,omitempty"`
	cause      error
}

func failure(code string, cause error) *Error {
	return &Error{Code: code, MessageKey: code, cause: cause}
}
func invalid(field string) *Error {
	err := failure(InvalidInput, nil)
	err.Params = map[string]string{"field": field}
	return err
}
func (e *Error) Error() string { return e.Code }
func (e *Error) Unwrap() error { return e.cause }
func (e *Error) ExitCode() int {
	switch e.Code {
	case InvalidInput, InvalidWorkspace, BuildPlatform, SourceUnsafe:
		return 2
	case Conflict, WorkspaceChanged, BuildBusy, SourceChanged, ArtifactChanged:
		return 3
	default:
		return 1
	}
}
func AsError(err error) *Error {
	var typed *Error
	if errors.As(err, &typed) {
		return typed
	}
	return failure(StorageFailure, err)
}
