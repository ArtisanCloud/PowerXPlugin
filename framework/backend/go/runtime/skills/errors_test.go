package skills

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMapErrorPreservesSkillErrorCode(t *testing.T) {
	status, result := MapError(NewError(ErrCodeNotFound, "missing"), validInvocation())
	require.Equal(t, http.StatusNotFound, status)
	require.False(t, result.Success)
	require.Equal(t, ErrCodeNotFound, result.Error.Code)
}

func TestMapErrorWrapsUnknownErrors(t *testing.T) {
	status, result := MapError(errors.New("boom"), validInvocation())
	require.Equal(t, http.StatusInternalServerError, status)
	require.Equal(t, ErrCodeExecutionFailed, result.Error.Code)
}
