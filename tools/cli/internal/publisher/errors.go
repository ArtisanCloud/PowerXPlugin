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
	var remote *marketError
	if errors.As(err, &remote) {
		e := failure(MarketRejected, nil)
		e.Params = map[string]string{"reason": remote.Code}
		if remote.Status == 401 {
			e.Code = MarketCredential
			e.MessageKey = MarketCredential
		}
		return errorRequestID(e, err)
	}
	var typed *Error
	if errors.As(err, &typed) {
		return errorRequestID(typed, err)
	}
	return errorRequestID(failure(StorageFailure, err), err)
}

func errorRequestID(value *Error, cause error) *Error {
	var trace *marketRequestError
	if !errors.As(cause, &trace) || !canonicalUUID(trace.requestID) {
		return value
	}
	copy := *value
	copy.Params = make(map[string]string, len(value.Params)+1)
	for k, v := range value.Params {
		copy.Params[k] = v
	}
	copy.Params["request_id"] = trace.requestID
	return &copy
}
