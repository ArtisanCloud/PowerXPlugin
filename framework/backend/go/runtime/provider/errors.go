package provider

import "errors"

var (
	ErrModeInvalid  = errors.New("provider mode must be local or delegated")
	ErrModeConflict = errors.New("provider mode conflict: config and env differ")
)

func IsInvalid(err error) bool {
	return errors.Is(err, ErrModeInvalid) || errors.Is(err, ErrProxyInvalid)
}

func IsConflict(err error) bool {
	return errors.Is(err, ErrModeConflict)
}
