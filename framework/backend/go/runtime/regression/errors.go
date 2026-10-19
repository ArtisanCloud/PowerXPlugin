package regression

import "errors"

const (
	InvalidRequest     = "REGRESSION_INVALID_REQUEST"
	NotFound           = "REGRESSION_NOT_FOUND"
	Busy               = "REGRESSION_BUSY"
	RequestConflict    = "REGRESSION_REQUEST_CONFLICT"
	DependencyBlocked  = "REGRESSION_DEPENDENCY_BLOCKED"
	Timeout            = "REGRESSION_TIMEOUT"
	CleanupFailed      = "REGRESSION_CLEANUP_FAILED"
	Interrupted        = "REGRESSION_INTERRUPTED"
	StoreUnavailable   = "REGRESSION_STORE_UNAVAILABLE"
	CleanupUnavailable = "REGRESSION_CLEANUP_UNAVAILABLE"
	EventMetaInvalid   = "REGRESSION_EVENT_META_INVALID"
	AssertionMissing   = "REGRESSION_ASSERTION_MISSING"
	AssertionDuplicate = "REGRESSION_ASSERTION_DUPLICATE"
	AssertionFailed    = "REGRESSION_ASSERTION_FAILED"
	EvidenceMissing    = "REGRESSION_EVIDENCE_MISSING"
)

type Error struct {
	Code    string
	Details any
}

func (e *Error) Error() string   { return e.Code }
func NewError(code string) error { return &Error{Code: code} }
func CodeOf(err error) string {
	if err == nil {
		return ""
	}
	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}
	return InvalidRequest
}
func MessageKey(code string) string { return "regression.errors." + code }
