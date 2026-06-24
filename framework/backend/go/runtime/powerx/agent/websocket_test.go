package agent

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDecodeWSMessage(t *testing.T) {
	ev, err := DecodeWSMessage([]byte(`{"type":"final","trace_id":"trace_001"}`))
	require.NoError(t, err)
	require.Equal(t, EventFinal, ev.Type)
}

func TestMapWSCloseError(t *testing.T) {
	err := MapWSCloseError(errors.New("closed"))
	require.Error(t, err)
	require.Contains(t, err.Error(), ErrCodeDisconnected)
}
